package services

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func MonitorMarket(targetToken solana.PublicKey) error {
	// First connect
	client, err := ws.Connect(context.Background(), rpc.MainNetBeta_WS)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	// Then subscribe
	sub, err := client.ProgramSubscribe(targetToken, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("failed to subscribe to program: %w", err)
	}

	// Process subscription messages
	for {
		select {
		case err := <-sub.Err():
			return fmt.Errorf("subscription error: %w", err)
		default:
			result, err := sub.Recv()
			if err != nil {
				return fmt.Errorf("receive error: %w", err)
			}
			if result == nil {
				return fmt.Errorf("received nil result")
			}
			log.Printf("Received program update: %+v\n", result)
		}
	}
}

func FetchPoolAccounts(ammId string) (*PoolAccounts, error) {
	// First try to get from Raydium's API
	accounts, err := FetchFromRaydiumAPI(ammId)
	if err != nil {
		// Fallback to on-chain data if API fails
		return FetchFromBlockchain(ammId)
	}
	return accounts, nil
}

func FetchFromBlockchain(ammId string) (*PoolAccounts, error) {
	// Connect to Solana
	client := rpc.New(rpc.MainNetBeta_RPC)

	// Get the AMM account data
	ammPubKey := solana.MustPublicKeyFromBase58(ammId)
	accountInfo, err := client.GetAccountInfo(
		context.Background(),
		ammPubKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AMM account: %w", err)
	}

	data := accountInfo.Value.Data.GetBinary()
	if len(data) < 256 { // Minimum size needed for the account data
		return nil, fmt.Errorf("invalid account data size")
	}

	const (
		baseVaultOffset  = 104 // Adjust these offsets based on actual layout
		quoteVaultOffset = baseVaultOffset + 32
		feeAccountOffset = quoteVaultOffset + 96 // Skip some fields to get to fee account
	)

	baseVault := solana.PublicKeyFromBytes(data[baseVaultOffset : baseVaultOffset+32])
	quoteVault := solana.PublicKeyFromBytes(data[quoteVaultOffset : quoteVaultOffset+32])
	feeAccount := solana.PublicKeyFromBytes(data[feeAccountOffset : feeAccountOffset+32])

	return &PoolAccounts{
		BaseVault:  baseVault,
		QuoteVault: quoteVault,
		FeeAccount: feeAccount,
	}, nil
}

func TrackNewTokens(tokenChan chan<- RaydiumPair) {
	log.Println("Starting trackNewTokens goroutine...")
	seenTokens := make(map[string]time.Time)
	tracker := NewTokenTracker("tracked_tokens.json")
	// Start with a longer lookback period to catch more tokens initially
	lastFetchTime := time.Now().Add(-24 * time.Hour)

	for {
		log.Printf("Starting new token fetch cycle... (lastFetchTime: %s)", lastFetchTime)
		pairs, err := FetchRaydiumPairs()
		if err != nil {
			log.Printf("Error fetching pairs: %v\n", err)
			time.Sleep(time.Second * FETCH_INTERVAL_SECONDS)
			continue
		}

		log.Printf("Successfully fetched %d pairs from Raydium", len(pairs))
		currentTime := time.Now()
		skippedCount := 0

		// Process each pair
		for _, pair := range pairs {
			// Debug logging for each pair
			log.Printf("Examining pair: %s (Address: %s, Timestamp: %s)",
				pair.Symbol, pair.Address, pair.Timestamp)

			// Skip invalid tokens with logging
			if pair.Address == "" || pair.Address == "11111111111111111111111111111111" {
				skippedCount++
				log.Printf("Skipping invalid token address: %s", pair.Symbol)
				continue
			}

			// Parse timestamp with better error handling
			var pairTime time.Time
			if pair.Timestamp == "" || pair.Timestamp == "-" {
				if _, exists := seenTokens[pair.Address]; exists {
					skippedCount++
					log.Printf("Skipping previously seen token without timestamp: %s", pair.Symbol)
					continue
				}
				pairTime = currentTime
				log.Printf("New token found without timestamp: %s (%s)", pair.Symbol, pair.Address)
			} else {
				var err error
				pairTime, err = time.Parse(time.RFC3339, pair.Timestamp)
				if err != nil {
					log.Printf("Failed to parse timestamp for token %s: %v", pair.Symbol, err)
					continue
				}
			}

			// Check if this is a new token
			if !pairTime.After(lastFetchTime) {
				continue
			}

			log.Printf("Processing potential new token: %s (%s)", pair.Symbol, pair.Address)

			// Fetch metrics and safety data
			metrics, err := FetchTokenMetrics(pair)
			if err != nil {
				log.Printf("Failed to fetch metrics for %s: %v", pair.Symbol, err)
				continue
			}

			safety, err := CheckTokenSafety(pair.Address)
			if err != nil {
				log.Printf("Failed to check safety for %s: %v", pair.Symbol, err)
				continue
			}

			// Basic filtering with logging
			if metrics.Liquidity < float64(MIN_LIQUIDITY_USD) {
				log.Printf("Token %s skipped: insufficient liquidity (%.2f < %.2f)",
					pair.Symbol, metrics.Liquidity, float64(MIN_LIQUIDITY_USD))
				continue
			}
			if metrics.MarketCap > MAX_MARKET_CAP_USD {
				log.Printf("Token %s skipped: market cap too high (%.2f > %.2f)",
					pair.Symbol, metrics.MarketCap, MAX_MARKET_CAP_USD)
				continue
			}
			if safety.HolderCount < MIN_HOLDER_COUNT {
				log.Printf("Token %s skipped: too few holders (%d < %d)",
					pair.Symbol, safety.HolderCount, MIN_HOLDER_COUNT)
				continue
			}

			log.Printf("Token %s passed initial filters", pair.Symbol)
			seenTokens[pair.Address] = currentTime
			tracker.Add(pair)

			log.Printf("🔥 High potential token found: %s", pair.Symbol)
			log.Printf("Metrics: Volume: $%.2f, Liquidity: $%.2f, Market Cap: $%.2f",
				metrics.Volume24h, metrics.Liquidity, metrics.MarketCap)
			log.Printf("Safety: Holders: %d, Top Holder Share: %.2f%%",
				safety.HolderCount, safety.TopHolderShare*100)

			select {
			case tokenChan <- pair:
				log.Printf("✅ Tracking new token: %s (%s)", pair.Name, pair.Address)
			default:
				log.Printf("⚠️ Channel full, skipping token: %s", pair.Name)
			}
		}

		lastFetchTime = currentTime
		// Add logging before sleep
		log.Printf("Completed processing cycle, sleeping for %d seconds...", FETCH_INTERVAL_SECONDS)
		runtime.GC()
		time.Sleep(time.Second * FETCH_INTERVAL_SECONDS)
	}
}

func HandleMarketActivity(activity *ws.ProgramResult) error {
	// Log basic activity information
	log.Printf("Market Activity Detected:")
	log.Printf("- Slot: %d", activity.Context.Slot)

	// Process account update
	account := activity.Value
	if account.Account != nil && account.Account.Owner != solana.SystemProgramID {
		// Fetch account data
		client := rpc.New(rpc.MainNetBeta_RPC)
		accountInfo, err := client.GetAccountInfo(
			context.Background(),
			account.Pubkey,
		)
		if err != nil {
			log.Printf("Warning: Failed to fetch account info for %s: %v", account.Pubkey, err)
			return err
		}

		// Log account changes
		log.Printf("- Account Updated: %s", account.Pubkey)
		log.Printf("  - Owner: %s", accountInfo.Value.Owner)
		log.Printf("  - Data Size: %d bytes", len(accountInfo.Value.Data.GetBinary()))
	}

	return nil
}
