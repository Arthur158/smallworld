"""Format 7: shared perception, general strategy, and dedicated specialists.

Routing uses public tribe identities supplied by the engine, not tactical rules.
Only participating modules execute and receive optimizer updates. The unchanged
format-6 actor supplies the initial policy; zero residual heads preserve it.
"""
import numpy as np
import torch
from torch import nn
from torch.nn import functional as F
from torch.distributions import Categorical

from . import ACTION_FEATURES
from .entity import EntityPolicy, WIDTH
from .graph import SPEC

RACES = tuple(SPEC['training_races'])
TRAITS = tuple(SPEC['training_traits'])
EXPERT_WIDTH = 64


def route_batch(decisions, device, count):
    races = np.full((len(decisions), count), -1, np.int64)
    traits = np.zeros((len(decisions), count, len(TRAITS)), np.float32)
    for i, decision in enumerate(decisions):
        for j, candidate in enumerate(decision['candidates']):
            if 'expert_race' not in candidate or 'expert_traits' not in candidate:
                raise ValueError('specialists require botenv with public expert routing metadata')
            race = candidate['expert_race']
            if race in RACES:
                races[i, j] = RACES.index(race)
            for trait in candidate['expert_traits']:
                if trait in TRAITS:
                    traits[i, j, TRAITS.index(trait)] = 1
    return tuple(torch.from_numpy(value).to(device) for value in (races, traits))


def expert():
    return nn.Sequential(nn.Linear(2*WIDTH, 96), nn.GELU(), nn.Linear(96, EXPERT_WIDTH), nn.GELU())


class SpecialistPolicy(EntityPolicy):
    def __init__(self, hidden=320):
        super().__init__(hidden)
        self.format_version, self.architecture = 7, 'specialists-v1'
        self.specialist_races = nn.ModuleList([expert() for _ in RACES])
        self.specialist_traits = nn.ModuleList([expert() for _ in TRAITS])
        # General strategy participates explicitly in both learned coordinators.
        self.specialist_general = nn.Sequential(nn.Linear(2*WIDTH, 128), nn.GELU())
        width = 128+3*EXPERT_WIDTH+4
        self.specialist_coordinator = nn.Sequential(nn.Linear(width, 128), nn.GELU(), nn.Linear(128, 1))
        self.specialist_defense = nn.Sequential(nn.Linear(width, 128), nn.GELU(), nn.Linear(128, 1))
        self.specialist_selector = nn.Sequential(nn.Linear(width, 128), nn.GELU(), nn.Linear(128, 1))
        self.specialist_value = nn.Sequential(nn.Linear(WIDTH, 128), nn.GELU(), nn.Linear(128, 1))
        self.specialist_survival = nn.Sequential(nn.Linear(2*WIDTH, 128), nn.GELU(), nn.Linear(128, 1))
        self.specialist_response = nn.Linear(WIDTH, 128)
        self.specialist_acceptance = nn.Linear(width, hidden)
        for layer in (self.specialist_coordinator[-1], self.specialist_selector[-1],
                      self.specialist_defense[-1], self.specialist_value[-1], self.specialist_survival[-1],
                      self.specialist_response, self.specialist_acceptance):
            nn.init.zeros_(layer.weight)
            nn.init.zeros_(layer.bias)
        self.freeze_backbone(True)

    def freeze_backbone(self, freeze):
        self.backbone_frozen = freeze
        for name, parameter in self.named_parameters():
            parameter.requires_grad_(not freeze or name.startswith('specialist_'))

    def warm_start(self, old):
        if getattr(old, 'format_version', 0) != 6 or old.hidden != self.hidden:
            raise ValueError('specialists require a format-6 entity checkpoint of the same width')
        missing, unexpected = self.load_state_dict(old.state_dict(), strict=False)
        if unexpected or any(not name.startswith('specialist_') for name in missing):
            raise ValueError('incompatible specialist expansion')

    def specialist_forward(self, batch, value=True, actions=None, ablate=None, expert_credit=False):
        if ablate not in (None, "race", "trait", "general", "all"):
            raise ValueError("unknown specialist ablation")
        if len(batch) != 9:
            raise ValueError('specialist policy requires routed observations')
        base = batch[:7]
        original = self.relational_forward(*base, value=value, actions=actions)
        packed = self.training and torch.is_grad_enabled() and getattr(self, 'pack_candidates', True)
        residual, entity_value, board, selected, context, queries = self.entity(
            base, value=value, actions=actions, packed=packed, return_features=True)
        mask, candidates = batch[2], batch[1]
        if packed:
            rows, columns = mask.nonzero(as_tuple=True)
            choices = candidates[rows, columns]
            race_ids, trait_ids = batch[7][rows, columns], batch[8][rows, columns]
        else:
            rows = torch.arange(mask.shape[0], device=mask.device).repeat_interleave(mask.shape[1])
            choices = candidates.flatten(0, 1)
            race_ids, trait_ids = batch[7].flatten(), batch[8].flatten(0, 1)
            queries = queries.flatten(0, 1)
        features = torch.cat((queries, context[rows]), -1)
        picking = choices[:, ACTION_FEATURES+SPEC['action_kinds'].index('pick')] > 0
        race_features = features.new_zeros((features.shape[0], EXPERT_WIDTH))
        trait_features = torch.zeros_like(race_features)
        # Dispatch only used modules. Separate Parameters keep inactive Adam
        # moments from updating unrelated experts. Ordinary CUDA inference is
        # used: dynamic dispatch intentionally does not use CUDA graph capture.
        for index in race_ids.unique().tolist():
            if index >= 0:
                for market in (False, True):
                    selected_rows = (race_ids == index) & (picking == market)
                    if selected_rows.any():
                        with torch.set_grad_enabled(torch.is_grad_enabled() and not market):
                            race_features[selected_rows] = self.specialist_races[index](features[selected_rows])
        for index in (trait_ids.sum(0) > 0).nonzero().flatten().tolist():
            for market in (False, True):
                selected_rows = (trait_ids[:, index] > 0) & (picking == market)
                if selected_rows.any():
                    with torch.set_grad_enabled(torch.is_grad_enabled() and not market):
                        trait_features[selected_rows] += self.specialist_traits[index](features[selected_rows])
        trait_features = trait_features / trait_ids.sum(-1, keepdim=True).clamp_min(1)
        picking = choices[:, ACTION_FEATURES+SPEC['action_kinds'].index('pick')] > 0
        # Market alternatives inform the selector without updating experts
        # merely because an unplayed tribe was offered in the market.
        race_features = torch.where(picking[:, None], race_features.detach(), race_features)
        trait_features = torch.where(picking[:, None], trait_features.detach(), trait_features)
        if ablate == "race":
            race_features = torch.zeros_like(race_features)
        if ablate == "trait":
            trait_features = torch.zeros_like(trait_features)
        general = self.specialist_general(features)
        if ablate == "general":
            general = torch.zeros_like(general)
        # Threat is a shared prediction trained only from observed retention.
        # It is meaningful for currently owned tiles; unknown targets carry an
        # explicit mask. Detach so actor rewards cannot distort its calibration.
        # Entity attention is global: adjacency only supplies a learned bias,
        # never a visibility mask. Include the global state explicitly as well,
        # so the visible market, every player's reserves and relative turn order
        # can influence each tile, including threats with no nearby attacker.
        survival = self.specialist_survival(torch.cat((board, context[:, None].expand_as(board)), -1)).squeeze(-1)
        known = batch[3][:, :, 14] * batch[5]
        risk = (1-survival.detach().sigmoid())*known
        tile_feedback = F.pad(torch.stack((risk, known), -1), (0, 0, 1, 0))
        spatial = batch[6][rows, columns] if packed else batch[6].flatten(0, 1)
        danger = torch.cat((tile_feedback[rows, spatial[:, 0]+1], tile_feedback[rows, spatial[:, 1]+1]), -1)
        combined = torch.cat((general, race_features, trait_features, race_features*trait_features, danger), -1)
        defense = sum(choices[:, ACTION_FEATURES+SPEC['action_kinds'].index(kind)]
                      for kind in ('redeploy', 'redeploy_out', 'finish')) > 0
        move_logits = torch.where(defense, self.specialist_defense(combined).squeeze(-1),
                                  self.specialist_coordinator(combined).squeeze(-1))
        logits = torch.where(picking, self.specialist_selector(combined).squeeze(-1), move_logits)
        if packed:
            corrections = residual.new_zeros(mask.shape)
            corrections[rows, columns] = logits
        else:
            corrections = logits.reshape(mask.shape)
        if ablate == "all":
            corrections = torch.zeros_like(corrections)
        distribution = Categorical(logits=(original[0].logits+residual+corrections).masked_fill(~mask, -1e9), validate_args=False)
        values = original[1]+entity_value+self.specialist_value(context).squeeze(-1) if value else None
        if actions is None:
            return distribution, values
        row_ids = torch.arange(mask.shape[0], device=mask.device)
        positions = ((mask.sum(-1).cumsum(0)-mask.sum(-1))+mask.long().cumsum(-1)[row_ids, actions]-1
                     if packed else row_ids*mask.shape[1]+actions)
        embedding = original[4]+self.entity_board_feedback(board)+self.specialist_response(board)
        hidden = original[2]+self.entity_action_feedback(selected)+self.specialist_acceptance(combined[positions])
        result = (distribution, values, hidden, survival, embedding)
        return (*result, features[positions]) if expert_credit else result

    def forward(self, *batch):
        return self.specialist_forward(batch)

    def action_distribution(self, *batch):
        return self.specialist_forward(batch, value=False)[0]

    def training_forward(self, batch, actions, expert_credit=False):
        return self.specialist_forward(batch, actions=actions, expert_credit=expert_credit)

    @torch.inference_mode()
    def choose(self, decision):
        from .model import batch_decisions
        return int(self.action_distribution(*batch_decisions([decision], next(self.parameters()).device, self)).logits.argmax(-1).item())
