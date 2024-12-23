package gamestate;

import (
	"fmt"
)


func DoesPlayerHaveStack(stackType string, player *Player) bool {
    for _, playerStack := range player.PieceStacks {
		if playerStack.Type == stackType {
			return true;
		}
	}
    return false;
}

func GetPlayerTribe(stackType string, player *Player) (*Tribe, error) {
    if player.ActiveTribe.IsStackValid(stackType) {
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

func (player *Player) addReserves(stacks []PieceStack) {
    for _, stack := range stacks { // Iterate over the incoming stacks
        found := false

        // Search for a matching stack in player's PieceStacks
        for i := range player.PieceStacks {
            if player.PieceStacks[i].Type == stack.Type {
                player.PieceStacks[i].Amount += stack.Amount // Add to the existing stack
                found = true
                break
            }
        }

        // If no matching stack was found, add a new one
        if !found {
            player.PieceStacks = append(player.PieceStacks, stack)
        }
    }
}
func AddPieceStacks(first, second []PieceStack) []PieceStack {
    result := make([]PieceStack, 0)
    pieceMap := make(map[string]int)

    // Combine amounts from the first slice
    for _, stack := range first {
        pieceMap[stack.Type] += stack.Amount
    }

    // Combine amounts from the second slice
    for _, stack := range second {
        pieceMap[stack.Type] += stack.Amount
    }

    // Construct the result slice
    for pieceType, totalAmount := range pieceMap {
        result = append(result, PieceStack{
            Type:   pieceType,
            Amount: totalAmount,
        })
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
					result = append(result, PieceStack{Type: stack1.Type, Amount: remainingAmount})
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


