package gamestate

// A small serializable PRNG keeps training, dice and restored games independent
// of process-global math/rand. It is not used for security-sensitive IDs.
func (gs *GameState) randomIntn(n int) int {
	if gs.RNGState == 0 {
		gs.RNGState = 1
	}
	x := gs.RNGState
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	gs.RNGState = x
	return int((x * 2685821657736338717) % uint64(n))
}

func (gs *GameState) shuffle(n int, swap func(int, int)) {
	for i := n - 1; i > 0; i-- {
		swap(i, gs.randomIntn(i+1))
	}
}

func (gs *GameState) rollDice() int {
	return [...]int{0, 0, 0, 1, 2, 3}[gs.randomIntn(6)]
}
