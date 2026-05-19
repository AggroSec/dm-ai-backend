package game

import "math/rand"

func DiceRoll(sides, count int) ([]int, int) {
	var results []int
	total := 0

	for range count {
		roll := rand.Intn(sides) + 1
		results = append(results, roll)
		total += roll
	}

	return results, total
}
