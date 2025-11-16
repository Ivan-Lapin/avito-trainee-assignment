package randutil

import (
	"math/rand/v2"
)

// PickUpToN перемешивает ids и возвращает не более n уникальных значений.
func PickUpToN(ids []string, n int) []string {
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	if len(ids) <= n {
		return ids
	}
	return ids[:n]
}
