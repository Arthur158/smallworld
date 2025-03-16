package gamestate;

import (
	"fmt"
	"math/rand"
)


func DoesPlayerHaveStack(stackType string, player *Player) bool {
    for _, playerStack := range player.PieceStacks {
		if playerStack.Type == stackType {
			return true;
		}
	}
    return false;
}

func (player *Player) getTribe(stackType string) (*Tribe, error) {
    if player.ActiveTribe != nil && player.ActiveTribe.IsStackValid(stackType) {
	return player.ActiveTribe, nil;
    }
    // Thinking of zombie-like tribes here
    for _, tribe := range player.PassiveTribes{
	if tribe.IsStackValid(stackType) {
	    return tribe, nil;
	}
    }

    return nil, fmt.Errorf("Player did not have valid tribe for piecestack")
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

func SubtractPieceStacks(reserves, expanses []PieceStack) ([]PieceStack, bool) {
	result := []PieceStack{} // Start with an empty list

	for _, stack1 := range reserves {
		subtracted := false

		// Search for a matching type in expanses
		for _, stack2 := range expanses {
			if stack1.Type == stack2.Type {
				if stack1.Amount < stack2.Amount {
					// Not enough quantity to subtract
					return nil, false
				}
				// Subtract the amount
				remainingAmount := stack1.Amount - stack2.Amount
				if remainingAmount > 0 {
					// Only add to result if the remaining amount is greater than 0
					result = append(result, PieceStack{Type: stack1.Type, Amount: remainingAmount, Tribe: stack1.Tribe})
				}
				subtracted = true
				break
			}
		}

		// If no match was found in expanses, add the stack1 element unchanged
		if !subtracted {
			result = append(result, stack1)
		}
	}

	// Verify no types in expanses are missing from reserves
	for _, stack2 := range expanses {
		found := false
		for _, stack1 := range reserves {
			if stack1.Type == stack2.Type {
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}

	return result, true
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
