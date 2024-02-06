package helper

import (
	"github.com/google/uuid"
	"hash/fnv"
)

// Internal function for unique ids

func GenerateUniqueId() int64 {
	uuid := uuid.New()
	hashedValue := hashUUID(uuid)
	return int64(hashedValue)
}

func hashUUID(u uuid.UUID) uint64 {
	h := fnv.New64a()
	h.Write(u[:])
	return h.Sum64()
}
