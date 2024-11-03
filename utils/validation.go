package utils

import (
	"grind/services"
	"log"
	"strings"
	"time"
)

func IsValidPair(pair services.RaydiumPair) bool {
	// Log the full pair data for debugging
	log.Printf("Checking pair: %+v", pair)

	// Check for Market field as an alternative to AmmId
	if pair.Market != "" {
		// If we have a valid market, check for either base or quote mint
		if pair.Pool.BaseMint != "" || pair.Pool.QuoteMint != "" {
			log.Printf("Valid pair found (using Market): %s (Market: %s, BaseMint: %s, QuoteMint: %s)",
				pair.Name, pair.Market, pair.Pool.BaseMint, pair.Pool.QuoteMint)
			return true
		}
	}

	// Check for non-zero liquidity as another validity indicator
	if pair.Liquidity > 0 {
		log.Printf("Valid pair found (using Liquidity): %s (Liquidity: %.2f)",
			pair.Name, pair.Liquidity)
		return true
	}

	// Check for valid price
	if pair.Price > 0 {
		log.Printf("Valid pair found (using Price): %s (Price: %.8f)",
			pair.Name, pair.Price)
		return true
	}

	// If we have both base and quote mints, consider it valid
	if pair.Pool.BaseMint != "" && pair.Pool.QuoteMint != "" {
		log.Printf("Valid pair found (using Mints): %s (BaseMint: %s, QuoteMint: %s)",
			pair.Name, pair.Pool.BaseMint, pair.Pool.QuoteMint)
		return true
	}

	// Log why the pair was invalid
	log.Printf("Invalid pair: %s - No valid identifiers found (Market: %s, Liquidity: %.2f, Price: %.8f, BaseMint: %s, QuoteMint: %s)",
		pair.Name, pair.Market, pair.Liquidity, pair.Price, pair.Pool.BaseMint, pair.Pool.QuoteMint)
	return false
}

func IsValidBase58Address(address string) bool {
	// Basic length check for Solana addresses
	if len(address) < 32 || len(address) > 44 {
		return false
	}

	// Check if it contains only valid base58 characters
	validChars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	for _, char := range address {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

func validateLockParameters(lockDuration time.Duration, percentage float64) bool {
	const (
		MIN_LOCK_DURATION   = 30 * 24 * time.Hour
		MIN_LOCK_PERCENTAGE = 80.0
	)
	return lockDuration >= MIN_LOCK_DURATION && percentage >= MIN_LOCK_PERCENTAGE
}
