package utils

import (
	"math/rand"
	"time"
)

// Random generates random number between min and max
func Random(min, max int) int {
	rand.Seed(int64(time.Now().Nanosecond()))
	return rand.Intn(max-min) + min
}
