package helper

import (
	"math/rand"
	"time"
)

// Internal function for unique ids
func GenerateUniqueId() int64 {
	currentTime := time.Now()
	timestamp := currentTime.UnixMilli()
	randomFactor := rand.Intn(100) + 1
	uniqueID := timestamp * int64(randomFactor)
	return uniqueID
}
