package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

// Helper function to generate a random state string
func GenerateState() string {
	bytes := make([]byte, 8) // 8 bytes will result in 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}
