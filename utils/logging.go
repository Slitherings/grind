package utils

import (
	"grind/services"
	"log"
)

func LogValidPair(pair services.RaydiumPair) {
	log.Printf("🚀 New token found: %s", pair.Name)
	log.Printf("   💰 Liquidity: $%.2f", pair.Liquidity)
	// ... other logging
}
