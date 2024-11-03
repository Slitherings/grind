package services

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
)

func IsValidPair(pair RaydiumPair) bool {
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

type RaydiumResponse []RaydiumPair

func FetchRaydiumPairs() ([]RaydiumPair, error) {
	log.Println("Fetching Raydium pairs...")

	// Increase timeouts even further and optimize transport settings
	client := &http.Client{
		Timeout: 60 * time.Second, // Increased to 60s
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   30 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Add retry logic with better error handling
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d after error: %v", attempt+1, maxRetries, lastErr)
			time.Sleep(time.Second * time.Duration(attempt+1) * 2) // Increased backoff
		}

		// Create request with context
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.raydium.io/v2/main/pairs", nil)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Add headers to potentially reduce response size
		req.Header.Add("Accept-Encoding", "gzip")
		req.Header.Add("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to fetch pairs: %w", err)
			continue
		}

		log.Printf("Response status: %d", resp.StatusCode)

		// Create a buffer to efficiently read the response
		var body []byte
		if resp.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				resp.Body.Close()
				cancel()
				lastErr = fmt.Errorf("failed to create gzip reader: %w", err)
				continue
			}
			body, err = io.ReadAll(reader)
			if err != nil {
				reader.Close()
				resp.Body.Close()
				cancel()
				lastErr = fmt.Errorf("failed to read gzipped response: %w", err)
				continue
			}
			reader.Close()
		} else {
			body, err = io.ReadAll(resp.Body)
		}
		resp.Body.Close()
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		log.Printf("Successfully read %d bytes from response", len(body))

		// Add sample logging before parsing
		log.Printf("Sample of raw response body: %s", string(body[:min(1000, len(body))]))

		var pairs RaydiumResponse
		if err := json.Unmarshal(body, &pairs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response (error: %w), raw data sample: %s",
				err, string(body[:min(1000, len(body))]))
		}

		// Log the first few pairs before validation
		log.Printf("First pair before validation: %+v", pairs[0])

		// Log a sample of the first valid and invalid pairs
		log.Printf("First 3 pairs from response:")
		for i := 0; i < min(3, len(pairs)); i++ {
			log.Printf("Pair %d: %+v", i+1, pairs[i])
		}

		// Modified validation logic with better counting
		validPairs := make(RaydiumResponse, 0)
		invalidCount := 0
		validCount := 0

		for _, pair := range pairs {
			if IsValidPair(pair) {
				validPairs = append(validPairs, pair)
				validCount++
				if validCount <= 3 {
					log.Printf("Sample valid pair %d: %+v", validCount, pair)
				}
			} else {
				invalidCount++
				if invalidCount <= 3 {
					log.Printf("Sample invalid pair %d: %+v", invalidCount, pair)
				}
			}
		}

		log.Printf("Validation Results:")
		log.Printf("- Total pairs processed: %d", len(pairs))
		log.Printf("- Valid pairs: %d", validCount)
		log.Printf("- Invalid pairs: %d", invalidCount)

		// Return valid pairs if we have any
		if validCount > 0 {
			return validPairs, nil
		}

		// If no valid pairs, return error with more context
		return nil, fmt.Errorf("no valid pairs found after filtering %d pairs. First pair in response: %+v",
			len(pairs), pairs[0])
	}
	return nil, fmt.Errorf("max retries exceeded, last error: %v", lastErr)
}

func ProcessNewTokens(ctx context.Context, tokenChan chan<- RaydiumPair, db Database, notifier Notifier) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pairs, err := FetchRaydiumPairs()
			if err != nil {
				log.Printf("Error fetching Raydium pairs: %v", err)
				continue
			}

			for _, pair := range pairs {
				if IsValidPair(pair) {
					select {
					case <-ctx.Done():
						return
					case tokenChan <- pair:
						// Optional: Add notification or database logging
						if err := db.StorePair(pair); err != nil {
							log.Printf("Error storing pair: %v", err)
						}
						notifier.NotifyNewPair(pair)
					}
				}
			}
		}
	}
}

func FetchFromRaydiumAPI(ammId string) (*PoolAccounts, error) {
	// Raydium's API endpoint for pool info
	url := fmt.Sprintf("https://api.raydium.io/v2/main/pool/%s", ammId)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pool info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var poolInfo struct {
		BaseVault  string `json:"baseVault"`
		QuoteVault string `json:"quoteVault"`
		FeeAccount string `json:"feeAccount"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&poolInfo); err != nil {
		return nil, fmt.Errorf("failed to decode pool info: %w", err)
	}

	return &PoolAccounts{
		BaseVault:  solana.MustPublicKeyFromBase58(poolInfo.BaseVault),
		QuoteVault: solana.MustPublicKeyFromBase58(poolInfo.QuoteVault),
		FeeAccount: solana.MustPublicKeyFromBase58(poolInfo.FeeAccount),
	}, nil
}

func FetchPoolInfo(tokenMint string) (*RaydiumPool, error) {
	pairs, err := FetchRaydiumPairs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pairs: %w", err)
	}

	// Search for the token as either base or quote mint
	for _, pair := range pairs {
		if pair.Pool.BaseMint == tokenMint || pair.Pool.QuoteMint == tokenMint {
			// Found the pool where our token is traded
			return &pair.Pool, nil
		}
	}

	// If we get here, we need to check if the token itself is an AMM ID
	for _, pair := range pairs {
		if pair.Pool.AmmId == tokenMint {
			return &pair.Pool, nil
		}
	}

	return nil, fmt.Errorf("pool not found for token: %s", tokenMint)
}
