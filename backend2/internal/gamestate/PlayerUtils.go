package gamestate

import (
    "fmt"
    "math/rand"
)

func DoesPlayerHaveStack(stackType string, player *Player) bool {
    for _, playerStack := range player.PieceStacks {
        if playerStack.Type == stackType {
            return true
        }
    }
    return false
}

func (player *Player) getTribe(stackType string) (*Tribe, error) {
    if player.ActiveTribe != nil && player.ActiveTribe.IsStackValid(stackType) {
        return player.ActiveTribe, nil
    }
    for _, tribe := range player.PassiveTribes {
        if tribe.IsStackValid(stackType) {
            return tribe, nil
        }
    }

    return nil, fmt.Errorf("Player did not have valid tribe for piecestack")
}

func (player *Player) StartRedeployment(gs *GameState) {

    if player.ActiveTribe != nil { player.PieceStacks = AddPieceStacks(player.PieceStacks, player.ActiveTribe.startRedeployment(gs)) }

    for _, tribe := range player.PassiveTribes {
        player.PieceStacks = AddPieceStacks(player.PieceStacks, tribe.startRedeployment(gs))
    }

    for _, power := range gs.Powers {
        if power.Owner == player && power.StartRedeployment != nil {
            player.PieceStacks = AddPieceStacks(player.PieceStacks, power.StartRedeployment(gs))
        }
    }
}

func AddPieceStacks(first, second []PieceStack) []PieceStack {
    result := append([]PieceStack{}, first...) // Copy first slice

    // Merge second slice into the result
    for _, stack := range second {
        merged := false
        for i := range result {
            if result[i].Type == stack.Type {
                result[i].Amount += stack.Amount
                merged = true
                break
            }
        }
        if !merged && stack.Amount > 0 {
            result = append(result, stack)
        }
    }

    return result
}

// Subtraction is atomic: failed expenses leave every reserve intact. Zero
// expenses do not require a matching stack (e.g. depleted temporary pieces).
func SubtractPieceStacks(reserves, expenses []PieceStack) ([]PieceStack, bool) {
    cost:=make(map[string]int)
    for _,s:=range expenses {if s.Amount<0{return reserves,false};cost[s.Type]+=s.Amount}
    available:=make(map[string]int)
    for _,s:=range reserves {available[s.Type]+=s.Amount}
    for kind,amount:=range cost {if available[kind]<amount{return reserves,false}}
    result:=make([]PieceStack,0,len(reserves))
    for _,s:=range reserves {take:=min(s.Amount,cost[s.Type]);s.Amount-=take;cost[s.Type]-=take;if s.Amount>0{result=append(result,s)}}
    return result,true
}

// rollCustomDiceWithLocalRNG returns a random number from a dice with faces 0, 0, 0, 1, 2, 3 using a local RNG
func RollDice() int {
    // Define the dice faces
    faces := []int{0, 0, 0, 1, 2, 3}

    // Return a random face
    result := faces[rand.Intn(len(faces))]
    println(result)
    return result
}
