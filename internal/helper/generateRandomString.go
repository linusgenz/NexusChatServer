package helper

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateRandomString(length int) (string, error) {
	randomBytes := make([]byte, length)

	// Use crypto/rand to generate random bytes
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode random bytes to base64, stripping non-alphanumeric characters
	randomString := base64.URLEncoding.EncodeToString(randomBytes)[:length]

	return randomString, nil
}
