package gamestate

import (
	"fmt"
	"slices"
	"sort"
)

// Changing feature order, scaling, actions or supported rules requires a new
// schema version. Training and production inference both use this adapter.
const BotSchema = "smallworld-bot-v1"

var botRaces = []string{"Dwarves", "Elves", "Giants", "Goblins", "Humans", "Orcs", "Ratmen", "Shrubmen", "Tritons", "Trolls", "White Ladies", "Wizards"}
var botTraits = []string{"Alchemist", "Commando", "Flying", "Forest", "Hill", "Hordes of", "Merchant", "Pillaging", "Seafaring", "Swamp", "Underworld", "Wealthy"}
var botKinds = []string{"pick", "conquer", "abandon", "redeploy", "redeploy_start", "finish", "decline"}

// Return copies: room setup must never mutate the feature vocabulary.
func BotRaces() []string  { return slices.Clone(botRaces) }
func BotTraits() []string { return slices.Clone(botTraits) }

type BotAction struct {
	Kind     string `json:"kind"`
	Stack    string `json:"stack,omitempty"`
	From     string `json:"from,omitempty"`
	Opponent int    `json:"opponent,omitempty"`
	Tile     string `json:"tile,omitempty"`
	Entry    int    `json:"entry,omitempty"`
}

type BotCandidate struct {
	ExpertRace   string    `json:"expert_race"`
	ExpertTraits []string  `json:"expert_traits"`
	Action       BotAction `json:"action"`
	Target       int       `json:"target"`
	Source       int       `json:"source"`
	Features     []float64 `json:"features"`
}

type BotDecision struct {
	Schema     string         `json:"schema"`
	Graph      *BotGraph      `json:"graph,omitempty"`
	Player     int            `json:"player"`
	State      []float64      `json:"state"`
	Candidates []BotCandidate `json:"candidates"`
}

// These hook sets have been audited to be pure when computing legality. In
// particular, Halflings' adjacency hook and Berserk's dice checks are excluded.
// Do not expand this list without adding action and full-match coverage.
func ValidateBotChoices(races, traits, powers []string) error {
	if len(races) == 0 || len(traits) == 0 {
		return fmt.Errorf("bots need at least one race and trait")
	}
	for _, r := range races {
		if !slices.Contains(botRaces, r) {
			return fmt.Errorf("race %q is not in the bot rules preset", r)
		}
	}
	for _, t := range traits {
		if !slices.Contains(botTraits, t) {
			return fmt.Errorf("trait %q is not in the bot rules preset", t)
		}
	}
	if len(powers) != 0 {
		return fmt.Errorf("powers are not supported by the bot rules preset")
	}
	return nil
}

func (gs *GameState) ValidateBotState() error {
	if len(gs.Powers) != 0 {
		return fmt.Errorf("bots cannot play saves with powers")
	}
	for _, p := range gs.Players {
		tribes := append([]*Tribe{p.ActiveTribe}, p.PassiveTribes...)
		for _, t := range tribes {
			if t == nil {
				continue
			}
			if err := ValidateBotChoices([]string{string(t.Race)}, []string{string(t.Trait)}, nil); err != nil {
				return err
			}
			if len(t.AdditionalPowers) != 0 {
				return fmt.Errorf("additional powers are not supported by bots")
			}
		}
	}
	for _, t := range gs.TribeList {
		if err := ValidateBotChoices([]string{string(t.Race)}, []string{string(t.Trait)}, nil); err != nil {
			return err
		}
	}
	return nil
}

// BotDecision enumerates available decisions without playing trial moves,
// consuming random numbers, or revealing other players' private coin totals.
// Redeploy-out and special expansion actions are deliberately outside v1.
func (gs *GameState) BotDecision() (BotDecision, error) {
	d := BotDecision{Schema: BotSchema, Candidates: []BotCandidate{}}
	if gs.TurnInfo == nil || gs.TurnInfo.PlayerIndex < 0 || gs.TurnInfo.PlayerIndex >= len(gs.Players) {
		return d, fmt.Errorf("game has no current player")
	}
	if err := gs.ValidateBotState(); err != nil {
		return d, err
	}
	d.Player = gs.TurnInfo.PlayerIndex
	p := gs.Players[d.Player]
	d.State = gs.botStateFeatures(p)
	if gs.TurnInfo.Phase == GameFinished {
		return d, nil
	}
	add := func(a BotAction, cost int) {
		d.Candidates = append(d.Candidates, BotCandidate{Action: a, Features: gs.botActionFeatures(p, a, cost)})
	}
	if gs.TurnInfo.Phase == TribeChoice {
		// Match the five entries visible to human players in megaUpdate.
		for i := 0; i < min(5, len(gs.TribeList)); i++ {
			if i == 0 || p.CoinPile >= i {
				add(BotAction{Kind: "pick", Entry: i}, i)
			}
		}
		if len(d.Candidates) == 0 {
			return d, fmt.Errorf("tribe market exhausted")
		}
		return d, nil
	}
	if p.ActiveTribe == nil {
		return d, fmt.Errorf("current player has no active tribe")
	}
	t := p.ActiveTribe
	reserve := stackAmount(p.PieceStacks, string(t.Race))
	ids := make([]string, 0, len(gs.TileList))
	for id := range gs.TileList {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	phase := gs.TurnInfo.Phase
	for _, id := range ids {
		tile := gs.TileList[id]
		owned := tile.OwningTribe == t
		if phase == Redeployment {
			if owned && reserve > 0 && t.canBeRedeployedIn(tile, string(t.Race), gs) {
				add(BotAction{Kind: "redeploy", Tile: id}, 1)
			}
			continue
		}
		if owned {
			if (phase == DeclineChoice || phase == TileAbandonment) && t.canTileBeAbandoned(tile) {
				add(BotAction{Kind: "abandon", Tile: id}, 0)
			}
			continue
		}
		if reserve < 1 || t.checkZoneAccess(tile) != nil || t.checkAdjacency(tile, gs) != nil {
			continue
		}
		cost, _, _, err := tile.countDefense(gs)
		if tile.OwningTribe != nil {
			cost, _, _, err = tile.OwningTribe.countDefense(tile, p, gs)
		}
		if err != nil {
			continue
		}
		expenses, _, _, _, err := t.countAttack(tile, cost, string(t.Race))
		if err != nil || len(expenses) != 1 || expenses[0].Type != string(t.Race) {
			continue
		}
		cost = expenses[0].Amount
		// A final reinforcement-die attempt is legal, including a failed roll.
		if cost <= reserve+3 {
			add(BotAction{Kind: "conquer", Tile: id}, cost)
		}
	}
	if phase == Redeployment {
		add(BotAction{Kind: "finish"}, 0)
	} else {
		add(BotAction{Kind: "redeploy_start"}, 0)
		waiting := 0
		for _, player := range gs.Players {
			if player.ActiveTribe == nil {
				waiting++
			}
		}
		if t.canGoIntoDecline(gs) && len(gs.TribeList) > waiting {
			add(BotAction{Kind: "decline"}, 0)
		}
	}
	return d, nil
}

// The integer is scoped to the current decision. Recompute candidates here so
// callers cannot submit arbitrary tile IDs or an unvalidated action.
func (gs *GameState) ApplyBotAction(index int) error {
	d, err := gs.BotDecision()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(d.Candidates) {
		return fmt.Errorf("invalid bot action index %d", index)
	}
	a := d.Candidates[index].Action
	p := gs.Players[d.Player]
	switch a.Kind {
	case "pick":
		return gs.HandleTribeChoice(d.Player, a.Entry)
	case "conquer":
		return gs.HandleConquest(a.Tile, d.Player, string(p.ActiveTribe.Race))
	case "abandon":
		return gs.HandleAbandonment(d.Player, a.Tile, string(p.ActiveTribe.Race))
	case "redeploy":
		return gs.HandleRedeploymentIn(d.Player, a.Tile, string(p.ActiveTribe.Race), 1)
	case "redeploy_start":
		return gs.HandleStartRedeployment(d.Player)
	case "finish":
		return gs.HandleFinishTurn(d.Player)
	case "decline":
		return gs.HandleDecline(d.Player)
	default:
		return fmt.Errorf("unsupported bot action %q", a.Kind)
	}
}

func stackAmount(stacks []PieceStack, kind string) int {
	for _, s := range stacks {
		if s.Type == kind {
			return s.Amount
		}
	}
	return 0
}

func oneHot(v string, vocabulary []string) []float64 {
	f := make([]float64, len(vocabulary))
	for i, key := range vocabulary {
		if key == v {
			f[i] = 1
		}
	}
	return f
}

func yes(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// State: round, phase (6), private own coins, player count, relative seat,
// reserve, own active/passive regions, enemy/neutral/empty regions, market size,
// active race (12), active trait (12), visible market: each [coins,pieces,race,trait].
// Counts are normalized but not clipped; no opponent coins or hidden market.
func (gs *GameState) botStateFeatures(p *Player) []float64 {
	f := []float64{float64(gs.TurnInfo.TurnIndex) / 9}
	for i := 0; i < 6; i++ {
		f = append(f, yes(int(gs.TurnInfo.Phase) == i))
	}
	f = append(f, float64(p.CoinPile)/100, float64(len(gs.Players))/6, float64(p.Index)/6)
	race, trait, reserve := "", "", 0
	if p.ActiveTribe != nil {
		race, trait = string(p.ActiveTribe.Race), string(p.ActiveTribe.Trait)
		reserve = stackAmount(p.PieceStacks, race)
	}
	regions := make([]float64, 5)
	for _, tile := range gs.TileList {
		i := 4
		if t := tile.OwningTribe; t != nil {
			i = 3
			if t.Owner != nil && t.Owner.Index >= 0 {
				i = 2
				if t.Owner == p {
					i = 1
					if t.IsActive {
						i = 0
					}
				}
			}
		}
		regions[i] += 1.0 / 50
	}
	f = append(f, float64(reserve)/20)
	f = append(f, regions...)
	// Only the number of cards is public, never their identities below the market.
	f = append(f, float64(len(gs.TribeList))/12)
	f = append(f, oneHot(race, botRaces)...)
	f = append(f, oneHot(trait, botTraits)...)
	for i := 0; i < 5; i++ {
		r, t, coins, pieces := "", "", 0, 0
		if i < len(gs.TribeList) {
			e := gs.TribeList[i]
			r, t, coins, pieces = string(e.Race), string(e.Trait), e.CoinPile, e.PiecePile
		}
		f = append(f, float64(coins)/10, float64(pieces)/20)
		f = append(f, oneHot(r, botRaces)...)
		f = append(f, oneHot(t, botTraits)...)
	}
	return f
}

// Action: kind (7), cost, entry price/pile/pieces, entry race/trait, tile
// terrain (10), attributes (3), edge/owner/presence, army/points, neighboring
// own/enemy/empty regions. No global tile ID means generated maps also work.
func (gs *GameState) botActionFeatures(p *Player, a BotAction, cost int) []float64 {
	f := oneHot(a.Kind, botKinds)
	f = append(f, float64(cost)/20)
	race, trait, coins, pieces, entry := "", "", 0, 0, 0
	if a.Kind == "pick" {
		e := gs.TribeList[a.Entry]
		race, trait, coins, pieces, entry = string(e.Race), string(e.Trait), e.CoinPile, e.PiecePile, a.Entry
	}
	f = append(f, float64(entry)/5, float64(coins)/10, float64(pieces)/20)
	f = append(f, oneHot(race, botRaces)...)
	f = append(f, oneHot(trait, botTraits)...)
	tile := gs.TileList[a.Tile]
	for i := 0; i < 10; i++ {
		f = append(f, yes(tile != nil && int(tile.Biome) == i))
	}
	for i := 0; i < 3; i++ {
		f = append(f, yes(tile != nil && slices.Contains(tile.Attributes, Attribute(i))))
	}
	tail := make([]float64, 10)
	if tile != nil {
		tail[0] = yes(tile.IsEdge)
		t := tile.OwningTribe
		tail[1] = yes(t != nil && t.Owner == p)
		tail[2] = yes(t != nil && t.Owner != nil && t.Owner.Index >= 0 && t.Owner != p)
		tail[3] = yes(t != nil && t.IsActive)
		for _, s := range tile.PieceStacks {
			tail[4] += float64(s.Amount) / 20
		}
		tail[5] = float64(tile.countPoints()) / 5
		if p.ActiveTribe != nil {
			tail[5] = float64(p.ActiveTribe.countPoints(tile)) / 5
		}
		for _, n := range tile.AdjacentTiles {
			i := 8
			if n.OwningTribe != nil {
				i = 7
				if n.OwningTribe.Owner == p {
					i = 6
				}
			}
			tail[i] += 1.0 / 10
		}
		tail[9] = float64(len(tile.AdjacentTiles)) / 10
	}
	return append(f, tail...)
}

const BotStateFeatures = 171
const BotActionFeatures = 58
