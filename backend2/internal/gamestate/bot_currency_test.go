package gamestate

import "testing"

func TestFreeTribeRemainsAvailableAfterCoinLoss(t *testing.T) {
	gs := v2TestState(t)
	player := gs.Players[0]
	// Haggling treaty payouts can leave a player below zero. The next free
	// tribe must still be pickable; otherwise observation has no valid moves.
	tribe, err := CreateTribe("Humans", "Haggling")
	if err != nil {
		t.Fatal(err)
	}
	tribe.Owner = player
	player.ActiveTribe = tribe
	player.CoinPile = 0
	gs.Players[1].PieceStacks = []PieceStack{{Type: "Treaty", Amount: 3}}
	tribe.getStacksForConquestTurnMap["Haggling"](player, gs)
	if player.CoinPile != -3 {
		t.Fatalf("expected coin-loss fixture, got %d", player.CoinPile)
	}
	player.ActiveTribe = nil
	gs.TurnInfo.Phase = TribeChoice
	decision, err := gs.BotDecisionV2()
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Candidates) != 1 || decision.Candidates[0].Action.Kind != "pick" || decision.Candidates[0].Action.Entry != 0 {
		t.Fatalf("free choice unavailable: %+v", decision.Candidates)
	}
	if err := gs.HandleTribeChoice(0, 1); err == nil {
		t.Fatal("paid entry allowed with insufficient coins")
	}
	rejected, err := gs.ApplyBotActionV2(0)
	if err != nil || rejected {
		t.Fatalf("free entry rejected: %v %v", rejected, err)
	}
	if gs.TurnInfo.Phase != TileAbandonment || player.ActiveTribe == nil {
		t.Fatal("free choice did not advance game")
	}
}
