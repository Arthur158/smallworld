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

func SubtractPieceStacks(reserves, expanses []PieceStack) ([]PieceStack, bool) {
	result := []PieceStack{} // Start with an empty list

	for _, stack1 := range reserves {
		subtracted := false

		// Search for a matching type in list2
		for _, stack2 := range expanses {
			if stack1.Type == stack2.Type {
				if stack1.Amount < stack2.Amount {
					// Not enough quantity to subtract
					return nil, false
				}
				// Subtract the amount and mark as processed
				result = append(result, PieceStack{Type: stack1.Type, Amount: stack1.Amount - stack2.Amount})
				subtracted = true
				break
			}
		}

		// If no match was found in list2, add the stack1 element unchanged
		if !subtracted {
			result = append(result, stack1)
		}
	}

	// Verify no types in list2 are missing from list1
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


