package services

import (
	"encoding/json"
	"log"
)

type TokenTracker struct {
	filepath string
}

func NewTokenTracker(filename string) *TokenTracker {
	return &TokenTracker{
		filepath: filename,
	}
}

func (t *TokenTracker) Add(pair RaydiumPair) {
	// Skip invalid tokens
	if pair.Address == "" || pair.Address == "11111111111111111111111111111111" {
		log.Printf("Skipping invalid token: %s", pair.Name)
		return
	}
	log.Printf("Added token: %s (%s)", pair.Name, pair.Address)
	// TODO: Implement persistence to file if needed
}
func LogRawPairSample(pairs []interface{}, sampleSize int) {
	log.Printf("Sampling first %d raw pairs:", sampleSize)
	for i := 0; i < min(sampleSize, len(pairs)); i++ {
		rawJSON, _ := json.MarshalIndent(pairs[i], "", "  ")
		log.Printf("Raw pair %d: %s", i, string(rawJSON))
	}
}
