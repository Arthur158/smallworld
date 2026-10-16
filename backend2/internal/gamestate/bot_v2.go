package gamestate

import (
	"fmt"
	"math"
	"slices"
	"sort"
)

const BotSchemaV2 = "smallworld-bot-v2"

var botV2Kinds = []string{"pick", "conquer", "abandon", "redeploy", "redeploy_start", "finish", "decline", "redeploy_out", "move", "opponent", "entry"}

type BotGraph struct {
	Nodes [][]float64 `json:"nodes"`
	Edges [][2]int    `json:"edges"`
}
type BotV2Memory struct {
	Turn     string
	Rejected map[BotAction]bool
	Uses     map[BotAction]int
	Placed   map[string]bool
}

func BotV2Races() []string  { return slices.Clone(botV2Races) }
func BotV2Traits() []string { return slices.Clone(botV2Traits) }
func BotV2Powers() []string { return slices.Clone(botV2Powers) }
func BotV2Spec() map[string]interface{} {
	return map[string]interface{}{"schema": BotSchemaV2, "training_races": BotTrainingRaces(), "training_traits": BotTrainingTraits(), "action_kinds": botV2Kinds, "races": botV2Races, "traits": botV2Traits, "powers": botV2Powers, "stacks": botV2Stacks, "states": botV2States, "state_features": BotV2StateWidth(), "action_features": BotV2ActionWidth(), "node_features": BotV2NodeWidth()}
}
func BotV2GroupFeatures() int { return len(botV2Kinds) + len(botV2Stacks) }
func BotV2StateWidth() int {
	return BotStateFeatures + 6*(1+2*len(botV2Races)+3*len(botV2Traits)+len(botV2Stacks)) + 5*(2+len(botV2Races)+len(botV2Traits)) + len(botV2States) + 3*len(botV2Powers) + 9
}
func BotV2ActionWidth() int {
	return BotActionFeatures + len(botV2Kinds) + len(botV2Stacks) + len(botV2Races) + len(botV2Traits) + 6
}
func BotV2NodeWidth() int {
	return 10 + 3 + 1 + 8 + 1 + len(botV2Races) + 2*len(botV2Traits) + len(botV2Stacks) + len(botV2States) + 8
}
func stackVector(stacks []PieceStack) []float64 {
	f := make([]float64, len(botV2Stacks))
	for i, k := range botV2Stacks {
		f[i] = float64(stackAmount(stacks, k)) / 20
	}
	return f
}
func tribeIdentity(t *Tribe) []float64 {
	r, tr := "", ""
	if t != nil {
		r, tr = string(t.Race), string(t.Trait)
	}
	f := oneHot(r, botV2Races)
	f = append(f, oneHot(tr, botV2Traits)...)
	extra := make([]float64, len(botV2Traits))
	if t != nil {
		for _, key := range t.AdditionalPowers {
			for i, k := range botV2Traits {
				if k == string(key) {
					extra[i] = 1
				}
			}
		}
	}
	return append(f, extra...)
}
func numeric(v interface{}) float64 {
	switch x := v.(type) {
	case int:
		return float64(x) / 20
	case float64:
		if !math.IsNaN(x) && !math.IsInf(x, 0) {
			return x / 20
		}
	case bool:
		return yes(x)
	}
	return 0
}
func tribeState(t *Tribe, p *Player) []float64 {
	f := make([]float64, len(botV2States))
	if t != nil {
		for i, k := range botV2States {
			if k != "loots" { // These counters are derived from public actions/messages.
				f[i] = numeric(t.State[k])
			}
		}
	}
	return f
}
func (gs *GameState) botV2State(p *Player) []float64 {
	f := gs.botStateFeatures(p)
	for relative := 0; relative < 6; relative++ {
		var q *Player
		if relative < len(gs.Players) {
			q = gs.Players[(p.Index+relative)%len(gs.Players)]
		}
		f = append(f, yes(q != nil))
		var active *Tribe
		var stacks []PieceStack
		if q != nil {
			active = q.ActiveTribe
			stacks = q.PieceStacks
		}
		f = append(f, tribeIdentity(active)...)
		f = append(f, stackVector(stacks)...)
		passive := make([]float64, len(botV2Races)+len(botV2Traits))
		if q != nil {
			for _, t := range q.PassiveTribes {
				v := append(oneHot(string(t.Race), botV2Races), oneHot(string(t.Trait), botV2Traits)...)
				for i, x := range v {
					passive[i] += x
				}
			}
		}
		f = append(f, passive...)
	}
	for i := 0; i < 5; i++ {
		r, tr := "", ""
		coins, pieces := 0, 0
		if i < len(gs.TribeList) {
			e := gs.TribeList[i]
			r, tr, coins, pieces = string(e.Race), string(e.Trait), e.CoinPile, e.PiecePile
		}
		f = append(f, float64(coins)/10, float64(pieces)/20)
		f = append(f, oneHot(r, botV2Races)...)
		f = append(f, oneHot(tr, botV2Traits)...)
	}
	f = append(f, tribeState(p.ActiveTribe, p)...)
	attacked := make([]float64, 6)
	if p.ActiveTribe != nil {
		if values, ok := p.ActiveTribe.State["playersAttacked"].([]int); ok {
			for _, i := range values {
				if i >= 0 && i < len(gs.Players) {
					attacked[(i-p.Index+len(gs.Players))%len(gs.Players)] = 1
				}
			}
		}
	}
	f = append(f, attacked...)
	next := 0.0
	if gs.RealTurninfo != nil {
		next = float64((gs.RealTurninfo.PlayerIndex-p.Index+len(gs.Players))%len(gs.Players)) / 6
	}
	f = append(f, yes(gs.RealTurninfo != nil), next, float64(len(gs.ModifierTurnsAfter))/10)
	for _, name := range botV2Powers {
		power := gs.Powers[name]
		if power == nil {
			f = append(f, 0, 0, 0)
		} else {
			f = append(f, 1, yes(power.Owner == p), numeric(power.State["hasMoved"]))
		}
	}
	return f
}
func (gs *GameState) botV2Tile(tile *Tile, p *Player) []float64 {
	f := make([]float64, 10)
	f[int(tile.Biome)] = 1
	for i := 0; i < 3; i++ {
		f = append(f, yes(slices.Contains(tile.Attributes, Attribute(i))))
	}
	f = append(f, yes(tile.IsEdge))
	owner := make([]float64, 8)
	t := tile.OwningTribe
	if t == nil {
		owner[7] = 1
	} else if t.Owner == nil || t.Owner.Index < 0 {
		owner[6] = 1
	} else {
		owner[(t.Owner.Index-p.Index+len(gs.Players))%len(gs.Players)] = 1
	}
	f = append(f, owner...)
	f = append(f, yes(t != nil && t.IsActive))
	f = append(f, tribeIdentity(t)...)
	f = append(f, stackVector(tile.PieceStacks)...)
	f = append(f, tribeState(t, p)...)
	// Private Skag loot is known to its owner once placed, never its next draw.
	loot, known := 0.0, 0.0
	if t != nil && t.Owner == p {
		if value, ok := tile.State["loot"]; ok {
			loot, known = numeric(value), 1
		}
	}
	f = append(f, float64(len(tile.AdjacentTiles))/10, loot, known, float64(tile.countPoints())/5)
	abandoned, mountain, deploy := 0.0, 0.0, 0.0
	tribes := append([]*Tribe{p.ActiveTribe}, p.PassiveTribes...)
	for _, tribe := range tribes {
		if tribe == nil {
			continue
		}
		if ids, ok := tribe.State["abandonedTiles"].([]string); ok && slices.Contains(ids, tile.Id) {
			abandoned = 1
		}
		if ids, ok := tribe.State["mountains"].([]string); ok && slices.Contains(ids, tile.Id) {
			mountain = 1
		}
		if values, ok := tribe.State["deploy"].(map[string]int); ok {
			deploy += float64(values[tile.Id]) / 20
		}
	}
	f = append(f, abandoned, mountain, deploy, yes(tile.IsSkyIsland))
	return f
}

// V2 exposes action intents, including special expansion controls. Legality is
// decided only by the real handler when an action is submitted. Rejections are
// observable feedback, not look-ahead simulations (which would leak dice/loot).
func (gs *GameState) BotDecisionV2() (BotDecision, error) {
	if err := gs.ValidateBotStateV2(); err != nil {
		return BotDecision{}, err
	}
	d := BotDecision{Schema: BotSchemaV2, Candidates: []BotCandidate{}, Graph: &BotGraph{Nodes: [][]float64{}, Edges: [][2]int{}}}
	if gs.TurnInfo == nil || gs.TurnInfo.PlayerIndex < 0 || gs.TurnInfo.PlayerIndex >= len(gs.Players) {
		return d, fmt.Errorf("no current player")
	}
	p := gs.Players[gs.TurnInfo.PlayerIndex]
	d.Player = p.Index
	d.State = gs.botV2State(p)
	ids := make([]string, 0, len(gs.TileList))
	for id := range gs.TileList {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	index := map[string]int{}
	for i, id := range ids {
		index[id] = i
		d.Graph.Nodes = append(d.Graph.Nodes, gs.botV2Tile(gs.TileList[id], p))
	}
	for i, id := range ids {
		neighbors := []int{}
		for _, n := range gs.TileList[id].AdjacentTiles {
			neighbors = append(neighbors, index[n.Id])
		}
		sort.Ints(neighbors)
		for _, j := range neighbors {
			d.Graph.Edges = append(d.Graph.Edges, [2]int{i, j})
		}
	}
	if gs.TurnInfo.Phase == GameFinished {
		return d, nil
	}
	// Enumeration itself stays pure; the execution path initializes this memory.
	memory := gs.BotV2Memory
	turn := fmt.Sprintf("%d:%d", gs.TurnInfo.TurnIndex, p.Index)
	if memory == nil || memory.Turn != turn {
		memory = &BotV2Memory{}
	}
	add := func(a BotAction, cost int) {
		if memory.Rejected[a] || memory.Uses[a] >= 32 {
			return
		}
		f := gs.botActionFeatures(p, a, cost)
		f = append(f, oneHot(a.Kind, botV2Kinds)...)
		f = append(f, oneHot(a.Stack, botV2Stacks)...)
		r, tr := "", ""
		if a.Kind == "pick" || a.Kind == "entry" {
			r, tr = string(gs.TribeList[a.Entry].Race), string(gs.TribeList[a.Entry].Trait)
		}
		f = append(f, oneHot(r, botV2Races)...)
		f = append(f, oneHot(tr, botV2Traits)...)
		for i := 0; i < 6; i++ {
			f = append(f, yes(a.Kind == "opponent" && i == (a.Opponent-p.Index+len(gs.Players))%len(gs.Players)))
		}
		target, from := -1, -1
		if a.Tile != "" {
			target = index[a.Tile]
		}
		if a.From != "" {
			from = index[a.From]
		}
		race, traits := gs.botExpertIdentity(p, a)
		d.Candidates = append(d.Candidates, BotCandidate{Action: a, Features: f, Target: target, Source: from, ExpertRace: race, ExpertTraits: traits})
	}
	phase := gs.TurnInfo.Phase
	if phase == TribeChoice {
		for i := 0; i < min(5, len(gs.TribeList)); i++ {
			if i == 0 || p.CoinPile >= i {
				add(BotAction{Kind: "pick", Entry: i}, i)
			}
		}
		return d, nil
	}
	kinds := map[string]bool{}
	for _, s := range p.PieceStacks {
		if s.Amount > 0 {
			kinds[s.Type] = true
		}
	}
	stacks := []string{}
	for k := range kinds {
		stacks = append(stacks, k)
	}
	sort.Strings(stacks)
	for _, id := range ids {
		tile := gs.TileList[id]
		owned := tile.OwningTribe != nil && tile.OwningTribe.Owner == p
		if owned && (phase == DeclineChoice || phase == TileAbandonment) && tile.OwningTribe.canTileBeAbandoned(tile) {
			add(BotAction{Kind: "abandon", Tile: id, Stack: string(tile.OwningTribe.Race)}, 0)
		}
		if phase == Redeployment && owned && (!memory.Placed[id] || (tile.OwningTribe.Race == "Amazons" && stackAmount(p.PieceStacks, "Amazons") < 4)) {
			for _, s := range tile.PieceStacks {
				t, err := p.getTribe(s.Type)
				if err == nil {
					if _, err := t.getStacksOutRedeployment(tile, s.Type); err == nil {
						add(BotAction{Kind: "redeploy_out", Tile: id, Stack: s.Type}, 0)
					}
				}
			}
		}
		for _, stack := range stacks {
			t, _ := p.getTribe(stack)
			if phase == Redeployment {
				if t != nil && (t.canBeRedeployedIn(tile, stack, gs) || len(t.handleDeploymentInMap) > 0) {
					if stack == string(t.Race) && t.Race == "Amazons" && stackAmount(p.PieceStacks, stack) <= 4 {
						continue
					}
					add(BotAction{Kind: "redeploy", Tile: id, Stack: stack}, 1)
				} else if power := gs.botPowerForStack(p, stack); power != nil && power.HandleRedeploymentIn != nil && owned {
					add(BotAction{Kind: "redeploy", Tile: id, Stack: stack}, 1)
				}
			} else {
				if t != nil && (stack == string(t.Race) || len(t.specialConquestMap) > 0 || len(t.countAttackMap) > 0) {
					// Pure reachability removes obvious ordinary-attack rejections; special
					// handlers may bypass it and are always allowed to receive their intent.
					if stack == string(t.Race) && len(t.specialConquestMap) == 0 && (owned && tile.OwningTribe == t || t.checkZoneAccess(tile) != nil || t.checkAdjacency(tile, gs) != nil) {
						continue
					}
					cost := 2
					for _, s := range tile.PieceStacks {
						cost += s.Amount
					}
					if tile.Biome == Mountain {
						cost++
					}
					add(BotAction{Kind: "conquer", Tile: id, Stack: stack}, cost)
				} else if power := gs.botPowerForStack(p, stack); power != nil && power.HandleConquest != nil {
					add(BotAction{Kind: "conquer", Tile: id, Stack: stack}, 2)
				}
			}
		}
		// Movement uses pieces on the board, not necessarily pieces in hand.
		if phase != Redeployment {
			for _, s := range tile.PieceStacks {
				if s.Amount <= 0 {
					continue
				}
				t, _ := p.getTribe(s.Type)
				movable := owned && t != nil && len(t.handleMovementMap) > 0 && slices.Contains([]string{"Great Ancient", "Left Cannon", "Right Cannon"}, s.Type)
				if power := gs.botPowerForStack(p, s.Type); power != nil && power.HandleMovement != nil && s.Type == "Balrog" {
					movable = true
				}
				if movable {
					for _, to := range ids {
						if id != to {
							add(BotAction{Kind: "move", From: id, Tile: to, Stack: s.Type}, 0)
						}
					}
				}
			}
		}
	}
	for _, stack := range stacks {
		t, _ := p.getTribe(stack)
		if t == nil {
			continue
		}
		if len(t.handleOpponentActionMap) > 0 && phase == Redeployment {
			for _, q := range gs.Players {
				if q != p {
					add(BotAction{Kind: "opponent", Opponent: q.Index, Stack: stack}, 0)
				}
			}
		}
		if len(t.handleEntryActionMap) > 0 {
			for i := 0; i < min(5, len(gs.TribeList)); i++ {
				add(BotAction{Kind: "entry", Entry: i, Stack: stack}, 0)
			}
		}
	}
	if phase == Redeployment {
		if p.ActiveTribe == nil || p.ActiveTribe.canEndTurn(gs) == nil {
			add(BotAction{Kind: "finish"}, 0)
		}
	} else {
		add(BotAction{Kind: "redeploy_start"}, 0)
		waiting := 0
		for _, q := range gs.Players {
			if q.ActiveTribe == nil {
				waiting++
			}
		}
		if t := p.ActiveTribe; t != nil && len(gs.TribeList) > waiting && t.canGoIntoDecline(gs) && t.canEndTurn(gs) == nil {
			add(BotAction{Kind: "decline"}, 0)
		}
	}
	if len(d.Candidates) == 0 {
		return d, fmt.Errorf("v2 has no remaining actions: round %d player %d phase %v", gs.TurnInfo.TurnIndex, p.Index, phase)
	}
	return d, nil
}
func (gs *GameState) botPowerForStack(p *Player, stack string) *Power {
	name := stack
	if stack == "Tomb-raider" {
		name = "Crypt of the Tomb-raider"
	}
	if stack == "Balrog" {
		name = "Wickedest Pentacle"
	}
	power := gs.Powers[name]
	if power != nil && power.Owner == p {
		return power
	}
	return nil
}
func (gs *GameState) ApplyBotActionV2(index int) (bool, error) {
	d, err := gs.BotDecisionV2()
	if err != nil {
		return false, err
	}
	if index < 0 || index >= len(d.Candidates) {
		return false, fmt.Errorf("invalid v2 action index")
	}
	a := d.Candidates[index].Action
	turn := fmt.Sprintf("%d:%d", gs.TurnInfo.TurnIndex, d.Player)
	if gs.BotV2Memory == nil || gs.BotV2Memory.Turn != turn {
		gs.BotV2Memory = &BotV2Memory{Turn: turn, Rejected: map[BotAction]bool{}, Uses: map[BotAction]int{}, Placed: map[string]bool{}}
	}
	m := gs.BotV2Memory
	switch a.Kind {
	case "pick":
		err = gs.HandleTribeChoice(d.Player, a.Entry)
	case "conquer":
		err = gs.HandleConquest(a.Tile, d.Player, a.Stack)
	case "abandon":
		err = gs.HandleAbandonment(d.Player, a.Tile, a.Stack)
	case "redeploy":
		err = gs.HandleRedeploymentIn(d.Player, a.Tile, a.Stack, 1)
	case "redeploy_out":
		err = gs.HandleRedeploymentOut(d.Player, a.Tile, a.Stack)
	case "move":
		err = gs.HandleMovement(d.Player, a.From, a.Tile, a.Stack)
	case "opponent":
		err = gs.HandleOpponentAction(d.Player, a.Opponent, a.Stack)
	case "entry":
		err = gs.HandleEntryAction(d.Player, a.Entry, a.Stack)
	case "redeploy_start":
		err = gs.HandleStartRedeployment(d.Player)
	case "finish":
		err = gs.HandleFinishTurn(d.Player)
	case "decline":
		err = gs.HandleDecline(d.Player)
	}
	if err != nil {
		m.Rejected[a] = true
		return true, nil
	}
	m.Rejected = map[BotAction]bool{}
	m.Uses[a]++
	if a.Kind == "redeploy" {
		m.Placed[a.Tile] = true
	}
	return false, nil
}

func ValidateBotChoicesV2(races, traits, powers []string) error {
	if len(races) == 0 || len(traits) == 0 {
		return fmt.Errorf("surface bots need races and traits")
	}
	if len(powers) > 0 {
		return fmt.Errorf("v2 surface training excludes powers")
	}
	for _, r := range races {
		if !slices.Contains(botV2Races, r) {
			return fmt.Errorf("race %q is not in the surface rules", r)
		}
	}
	for _, tr := range traits {
		if !slices.Contains(botV2Traits, tr) {
			return fmt.Errorf("trait %q is not in the surface rules", tr)
		}
	}
	return nil
}
func (gs *GameState) ValidateBotStateV2() error {
	if len(gs.Powers) > 0 {
		return fmt.Errorf("surface bot cannot play powers")
	}
	for _, tile := range gs.TileList {
		if tile.Biome == MineArea || tile.Biome == CrystalArea || tile.Biome == AbyssalChasm || tile.Biome == River {
			return fmt.Errorf("surface bot cannot play underground terrain")
		}
	}
	for _, entry := range gs.TribeList {
		if err := ValidateBotChoicesV2([]string{string(entry.Race)}, []string{string(entry.Trait)}, nil); err != nil {
			return err
		}
	}
	for _, p := range gs.Players {
		for _, t := range append([]*Tribe{p.ActiveTribe}, p.PassiveTribes...) {
			if t == nil {
				continue
			}
			if err := ValidateBotChoicesV2([]string{string(t.Race)}, []string{string(t.Trait)}, nil); err != nil {
				return err
			}
			for _, extra := range t.AdditionalPowers {
				if !slices.Contains(botV2Traits, string(extra)) {
					return fmt.Errorf("unsupported additional trait")
				}
			}
		}
	}
	return nil
}

// Training varies neutral placement as well as topology. Empty generated maps
// remain part of the distribution, matching the live generator's default.
func (gs *GameState) PopulateBotSurfaceMap() {
	density := gs.randomIntn(5) * 10
	tribe := CreateBaseTribe()
	tribe.Race = "Lost Tribe"
	tribe.Trait = "Lost"
	tribe.IsActive = false
	tribe.Owner = &Player{Index: -1, Name: "Lost Tribe", ActiveTribe: tribe}
	ids := []string{}
	for id := range gs.TileList {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		tile := gs.TileList[id]
		if tile.Biome != Water && gs.randomIntn(100) < density {
			tile.OwningTribe = tribe
			tile.PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
		}
	}
}

// Public controller identity, using the same tribe lookup as action execution.
// No move scores or tactical rules are included in this metadata.
func (gs *GameState) botExpertIdentity(p *Player, a BotAction) (string, []string) {
	if a.Kind == "pick" {
		entry := gs.TribeList[a.Entry]
		return string(entry.Race), []string{string(entry.Trait)}
	}
	t := p.ActiveTribe
	if a.Stack != "" {
		t, _ = p.getTribe(a.Stack)
	}
	if a.Kind == "abandon" {
		t = gs.TileList[a.Tile].OwningTribe
	}
	traits := []string{}
	if t == nil {
		return "", traits
	}
	traits = append(traits, string(t.Trait))
	for _, trait := range t.AdditionalPowers {
		if !slices.Contains(traits, string(trait)) {
			traits = append(traits, string(trait))
		}
	}
	sort.Strings(traits)
	return string(t.Race), traits
}
