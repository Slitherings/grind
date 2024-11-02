package main

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"

	// Replace these relative imports with your module path
	"github.com/yourusername/yourproject/analytics"
	"github.com/yourusername/yourproject/config"
	"github.com/yourusername/yourproject/db"
	"github.com/yourusername/yourproject/notifications"
)

// Add these type definitions before the main function
type RaydiumPool struct {
	AmmId         string `json:"ammId"`
	LpMint        string `json:"lpMint"`
	BaseMint      string `json:"baseMint"`
	QuoteMint     string `json:"quoteMint"`
	BaseDecimals  int    `json:"baseDecimals"`
	QuoteDecimals int    `json:"quoteDecimals"`
	LpDecimals    int    `json:"lpDecimals"`
}

type RaydiumPair struct {
	Name      string      `json:"name"`
	Symbol    string      `json:"symbol"`
	Address   string      `json:"address"`
	Timestamp string      `json:"timestamp"`
	Pool      RaydiumPool `json:"pool"`
}

type RaydiumResponse []RaydiumPair

// Add these type definitions before the RaydiumPair struct
type TokenTracker struct {
	filepath string
}

// Keep only one instance of this function
func NewTokenTracker(filepath string) *TokenTracker {
	return &TokenTracker{
		filepath: filepath,
	}
}

// Add this method
func (t *TokenTracker) Add(pair RaydiumPair) {
	// Skip invalid tokens
	if pair.Address == "" || pair.Address == "11111111111111111111111111111111" {
		log.Printf("Skipping invalid token: %s", pair.Name)
		return
	}
	log.Printf("Added token: %s (%s)", pair.Name, pair.Address)
	// TODO: Implement persistence to file if needed
}

// Replace the wallet generation with your Phantom address
func getWallet() solana.PublicKey {
	// Replace this with your Phantom wallet's private key if you want to use the full wallet
	// Or just use the public address if you only need to receive funds
	phantomAddress := "79hjkpSwnJ4g7PJ7YYQfJRGEwHwWWUB7ziyve15fC4YC" // Replace this with your address
	pubKey, err := solana.PublicKeyFromBase58(phantomAddress)
	if err != nil {
		log.Fatalf("Failed to parse wallet address: %v", err)
	}
	return pubKey
}

// Add these types at the top with other type definitions
type PoolAccounts struct {
	BaseVault  solana.PublicKey
	QuoteVault solana.PublicKey
	FeeAccount solana.PublicKey
}

// Add this function
func fetchPoolAccounts(ammId string) (*PoolAccounts, error) {
	// For demonstration, returning mock accounts
	// In production, you would fetch these from Raydium's API or blockchain
	return &PoolAccounts{
		BaseVault:  solana.MustPublicKeyFromBase58(ammId), // Replace with actual vault address
		QuoteVault: solana.MustPublicKeyFromBase58(ammId), // Replace with actual vault address
		FeeAccount: solana.MustPublicKeyFromBase58(ammId), // Replace with actual fee account address
	}, nil
}

// Add these new types at the top with other type definitions
type TokenMetrics struct {
	PriceChange24h float64
	Volume24h      float64
	MarketCap      float64
	Liquidity      float64
}

// Add these new types with other type definitions
type TokenSafetyMetrics struct {
	LiquidityLocked   bool
	LiquidityLockTime time.Duration
	IsHoneypot        bool
	TopHolderShare    float64
	HolderCount       int
	SocialMetrics     SocialMetrics
	ContractVerified  bool
	CreatorAddress    string
	TokenAge          time.Duration
}

type SocialMetrics struct {
	TwitterFollowers int
	TelegramMembers  int
	WebsiteExists    bool
	GitHubExists     bool
	HasWhitepaper    bool
}

// Add this function near other analysis functions
func checkTokenSafety(tokenAddress string) (TokenSafetyMetrics, error) {
	// In production, implement API calls to services like:
	// - GoPlus for contract security analysis
	// - DexScreener/DexTools for liquidity info
	// - Blockchain explorer APIs for holder analysis
	// - Social media APIs for community metrics

	safety := TokenSafetyMetrics{}
	// Check liquidity lock status
	locked, lockDuration, err := checkLiquidityLock(tokenAddress)
	if err != nil {
		return safety, fmt.Errorf("failed to check liquidity lock: %w", err)
	}
	safety.LiquidityLocked = locked
	safety.LiquidityLockTime = lockDuration

	// Check for honeypot characteristics
	isHoneypot, err := detectHoneypot(tokenAddress)
	if err != nil {
		return safety, fmt.Errorf("failed to check honeypot: %w", err)
	}
	safety.IsHoneypot = isHoneypot

	// Analyze token distribution
	topHolder, holderCount, err := analyzeHolders(tokenAddress)
	if err != nil {
		return safety, fmt.Errorf("failed to analyze holders: %w", err)
	}
	safety.TopHolderShare = topHolder
	safety.HolderCount = holderCount

	// Check social presence
	safety.SocialMetrics = checkSocialPresence(tokenAddress)

	return safety, nil
}

// Modify analyzeTokenPotential to include safety checks
func analyzeTokenPotential(metrics TokenMetrics, safety TokenSafetyMetrics) (bool, string) {
	const (
		MIN_LIQUIDITY     = 10000.0
		MIN_VOLUME        = 5000.0
		MIN_MARKET_CAP    = 50000.0
		MAX_MARKET_CAP    = 10000000.0
		MIN_PRICE_CHANGE  = 5.0
		MAX_TOP_HOLDER    = 0.15                // 15% maximum for largest holder
		MIN_HOLDER_COUNT  = 100                 // Minimum number of holders
		MIN_LOCK_DURATION = 30 * 24 * time.Hour // 30 days minimum lock
		MIN_SOCIAL_SCORE  = 2                   // Minimum number of social criteria met
	)

	reasons := []string{}

	// Original metrics checks...
	if metrics.Liquidity < MIN_LIQUIDITY {
		reasons = append(reasons, fmt.Sprintf("Low liquidity: $%.2f < $%.2f", metrics.Liquidity, MIN_LIQUIDITY))
	}

	// Add safety checks
	if !safety.LiquidityLocked {
		reasons = append(reasons, "Liquidity not locked")
	} else if safety.LiquidityLockTime < MIN_LOCK_DURATION {
		reasons = append(reasons, fmt.Sprintf("Lock duration too short: %v < %v", safety.LiquidityLockTime, MIN_LOCK_DURATION))
	}

	if safety.IsHoneypot {
		reasons = append(reasons, "Detected honeypot characteristics")
	}

	if safety.TopHolderShare > MAX_TOP_HOLDER {
		reasons = append(reasons, fmt.Sprintf("Top holder owns too much: %.1f%% > %.1f%%",
			safety.TopHolderShare*100, MAX_TOP_HOLDER*100))
	}

	if safety.HolderCount < MIN_HOLDER_COUNT {
		reasons = append(reasons, fmt.Sprintf("Too few holders: %d < %d",
			safety.HolderCount, MIN_HOLDER_COUNT))
	}

	// Check social presence
	socialScore := 0
	if safety.SocialMetrics.TwitterFollowers > 100 {
		socialScore++
	}
	if safety.SocialMetrics.TelegramMembers > 100 {
		socialScore++
	}
	if safety.SocialMetrics.WebsiteExists {
		socialScore++
	}
	if safety.SocialMetrics.GitHubExists {
		socialScore++
	}
	if safety.SocialMetrics.HasWhitepaper {
		socialScore++
	}

	if socialScore < MIN_SOCIAL_SCORE {
		reasons = append(reasons, fmt.Sprintf("Weak social presence: %d/%d criteria met",
			socialScore, MIN_SOCIAL_SCORE))
	}

	isGoodToken := len(reasons) == 0
	reasonStr := strings.Join(reasons, ", ")

	return isGoodToken, reasonStr
}

// Add these helper functions (implement API calls in production)
func checkLiquidityLock(tokenAddress string) (bool, time.Duration, error) {
	// TODO: Implement API calls to check liquidity lock status
	// Example services: GoPlus, DexTools API
	return true, 180 * 24 * time.Hour, nil
}

func detectHoneypot(tokenAddress string) (bool, error) {
	// TODO: Implement honeypot detection
	// Check buy/sell tax, transfer restrictions, blacklists
	// Example services: GoPlus Security API, Honeypot.is API
	return false, nil
}

func analyzeHolders(tokenAddress string) (float64, int, error) {
	// TODO: Implement holder analysis using blockchain explorer APIs
	// Example: Solscan API, Solana RPC calls
	return 0.05, 1000, nil
}

func checkSocialPresence(tokenAddress string) SocialMetrics {
	// TODO: Implement social media presence verification
	// Use Twitter API, Telegram API, etc.
	return SocialMetrics{
		TwitterFollowers: 500,
		TelegramMembers:  1000,
		WebsiteExists:    true,
		GitHubExists:     true,
		HasWhitepaper:    true,
	}
}

func fetchTokenMetrics(_ RaydiumPair) (TokenMetrics, error) {
	// TODO: In production, fetch real metrics from your data source
	// This is a mock implementation
	return TokenMetrics{
		PriceChange24h: 10.0,   // 10% price increase
		Volume24h:      10000,  // $10k volume
		MarketCap:      100000, // $100k market cap
		Liquidity:      20000,  // $20k liquidity
	}, nil
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.NewDatabase("tokens.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize notifiers
	telegramNotifier := notifications.NewTelegramNotifier(
		cfg.TelegramBotKey,
		cfg.TelegramChatID,
	)

	// Initialize analyzer
	analyzer := analytics.NewTokenAnalyzer(analytics.TokenAnalyzerConfig{
		MinLiquidity:   cfg.MinLiquidity,
		MinHolderCount: cfg.MinHolders,
		MaxTopHolder:   cfg.MaxTopHolder,
		MinAge:         cfg.MinLockTime,
	})

	// Add signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Change to devnet connection
	ws, err := ws.Connect(ctx, rpc.DevNet_WS)
	if err != nil {
		log.Fatal(err)
	}

	// Create RPC client for devnet
	client := rpc.New(rpc.DevNet_RPC)

	// Generate wallet and request airdrop
	wallet := getWallet()

	// Add balance check after airdrop
	currentBalance := checkBalance(client, wallet)
	fmt.Printf("💰 Current wallet balance: %.2f SOL\n", currentBalance)

	// Amount of SOL you want to spend
	amountToSpend := 0.1 // in SOL

	// Create a channel to receive new token pairss
	tokenChan := make(chan RaydiumPair, 100) // Added buffer size

	// Initialize targetToken with a meaningful default or wait for first token
	var targetToken solana.PublicKey

	// Start the token tracking goroutine
	go trackNewTokens(tokenChan)

	// Monitor both new tokens and DEX activity
	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down gracefully...")
			cancel()
			return
		case newToken := <-tokenChan:
			fmt.Printf("\n🆕 New token detected!\n")
			fmt.Printf("Name: %s\n", newToken.Name)
			fmt.Printf("Symbol: %s\n", newToken.Symbol)
			fmt.Printf("Address: %s\n", newToken.Address)
			fmt.Printf("Timestamp: %s\n", newToken.Timestamp)

			// Add validation for empty address
			if newToken.Address == "" || newToken.Address == "11111111111111111111111111111111" {
				log.Printf("Skipping token with invalid address")
				continue
			}

			// Fetch token metrics and safety metrics
			metrics, err := fetchTokenMetrics(newToken)
			if err != nil {
				log.Printf("Failed to fetch token metrics: %v", err)
				continue
			}

			safety, err := checkTokenSafety(newToken.Address)
			if err != nil {
				log.Printf("Failed to check token safety: %v", err)
				continue
			}

			// Analyze token potential with safety checks
			isGoodToken, reasons := analyzeTokenPotential(metrics, safety)
			if !isGoodToken {
				log.Printf("Skipping token %s. Reasons: %s", newToken.Symbol, reasons)
				continue
			}

			log.Printf("✨ Found promising token: %s", newToken.Symbol)
			log.Printf("Metrics: Volume: $%.2f, Liquidity: $%.2f, Market Cap: $%.2f, 24h Change: %.2f%%",
				metrics.Volume24h, metrics.Liquidity, metrics.MarketCap, metrics.PriceChange24h)

			targetToken = solana.MustPublicKeyFromBase58(newToken.Address)

			// Attempt to buy the new token immediately
			fmt.Printf("🎯 Attempting to snipe token: %s\n", newToken.Symbol)
			if err := attemptBuy(wallet, targetToken, amountToSpend); err != nil {
				log.Printf("❌ Failed to snipe token: %v\n", err)
			}

			// Create new subscription
			sub, err := ws.ProgramSubscribe(
				targetToken,
				rpc.CommitmentConfirmed,
			)
			if err != nil {
				log.Printf("Error subscribing to new token: %v\n", err)
				continue
			}

			// Handle the subscription in a separate goroutine
			go func() {
				for {
					got, err := sub.Recv()
					if err != nil {
						log.Printf("Error receiving subscription data: %v\n", err)
						return
					}

					if got != nil {
						_ = got.Value.Account.Data.GetBinary()
						fmt.Printf("🔔 Raydium DEX activity detected for %s!\n", targetToken.String())
						attemptBuy(wallet, targetToken, amountToSpend)
					}
				}
			}()

		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// Modified tracker function to send tokens through channel
func trackNewTokens(tokenChan chan<- RaydiumPair) {
	seenTokens := make(map[string]time.Time)
	tracker := NewTokenTracker("tracked_tokens.json")

	// Start 1 minute ago
	lastFetchTime := time.Now().Add(-time.Minute)

	for {
		log.Printf("Starting new token fetch cycle... (lastFetchTime: %s)", lastFetchTime)
		pairs, err := fetchRaydiumPairs()
		if err != nil {
			log.Printf("Error fetching pairs: %v\n", err)
			time.Sleep(time.Second * 10)
			continue
		}

		currentTime := time.Now()
		newPairsCount := 0

		// Add sample logging
		log.Printf("Sample of first 5 pairs timestamps:")
		for i, pair := range pairs {
			if i >= 5 {
				break
			}
			log.Printf("Pair %d: %s - %s", i, pair.Symbol, pair.Timestamp)
		}

		for _, pair := range pairs {
			// Skip invalid tokens immediately
			if pair.Address == "" || pair.Address == "11111111111111111111111111111111" {
				continue
			}

			// Parse the timestamp from the pair
			pairTime, err := time.Parse(time.RFC3339, pair.Timestamp)
			if err != nil {
				log.Printf("Error parsing timestamp for %s: %v", pair.Name, err)
				continue
			}

			// More detailed timestamp comparison logging
			if pairTime.After(lastFetchTime) {
				log.Printf("Potential new token found: %s (created: %s, lastFetch: %s)",
					pair.Symbol, pairTime.Format(time.RFC3339), lastFetchTime.Format(time.RFC3339))
			}

			// Check if this is a new token or if it was created after our last fetch
			lastSeen, exists := seenTokens[pair.Address]
			if !exists || pairTime.After(lastSeen) {
				seenTokens[pair.Address] = currentTime
				tracker.Add(pair)
				newPairsCount++

				// Only send truly new tokens (created after our last fetch)
				if pairTime.After(lastFetchTime) {
					log.Printf("Found new token: %s (created: %s)", pair.Symbol, pairTime)
					select {
					case tokenChan <- pair:
						log.Printf("🆕 Successfully sent new token: %s (%s) created at %s",
							pair.Name, pair.Address, pair.Timestamp)
					case <-time.After(time.Millisecond * 100):
						log.Printf("Timeout sending token to channel: %s", pair.Name)
					}
				} else {
					log.Printf("Token %s is new to us but created before lastFetchTime", pair.Symbol)
				}
			}
		}

		lastFetchTime = currentTime
		log.Printf("Found %d new pairs out of %d total pairs", newPairsCount, len(pairs))

		// Clear some memory before next fetch
		pairs = nil
		runtime.GC()

		log.Printf("Completed processing cycle, waiting before next fetch...")
		time.Sleep(time.Second * 30)
	}
}

func attemptBuy(wallet, tokenMint solana.PublicKey, amount float64) error {
	// Change to devnet RPC
	rpcEndpoint := rpc.DevNet_RPC
	client := rpc.New(rpcEndpoint)

	// Devnet Raydium Program ID (you'll need to verify this)
	raydiumProgramID := solana.MustPublicKeyFromBase58("DnXyn8k7mwDkw8QhNuhg7qhQ8Bv3Art5JqQk1qNQhk9J") // Example - verify this

	// WSOL (Wrapped SOL) token mint
	wsolMint := solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")

	// Fetch pool information from Raydium API
	poolInfo, err := fetchPoolInfo(tokenMint.String())
	if err != nil {
		return fmt.Errorf("failed to fetch pool info: %w", err)
	}

	// Get Associated Token Accounts (ATAs)
	userWSOLAccount, _, err := solana.FindAssociatedTokenAddress(wallet, wsolMint)
	if err != nil {
		return fmt.Errorf("error finding WSOL ATA: %w", err)
	}

	userTokenAccount, _, err := solana.FindAssociatedTokenAddress(wallet, tokenMint)
	if err != nil {
		return fmt.Errorf("error finding token ATA: %w", err)
	}

	// Convert pool addresses from string to PublicKey
	ammId := solana.MustPublicKeyFromBase58(poolInfo.AmmId)
	lpMint := solana.MustPublicKeyFromBase58(poolInfo.LpMint)

	// Fetch additional pool accounts
	poolAccounts, err := fetchPoolAccounts(poolInfo.AmmId)
	if err != nil {
		return fmt.Errorf("failed to fetch pool accounts: %w", err)
	}

	// Prepare swap instruction
	amountIn := uint64(amount * 1e9)                 // Convert SOL to lamports
	minAmountOut := uint64(float64(amountIn) * 0.99) // 1% slippage tolerance

	swapInstruction := createSwapInstruction(
		raydiumProgramID,
		ammId,
		userWSOLAccount,
		poolAccounts.BaseVault,
		poolAccounts.QuoteVault,
		userTokenAccount,
		lpMint,
		poolAccounts.FeeAccount,
		wallet,
		amountIn,
		minAmountOut,
	)

	// Build transaction
	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{swapInstruction},
		recent.Value.Blockhash,
		solana.TransactionPayer(wallet),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign and send transaction
	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	fmt.Printf("✅ Swap transaction sent: %s\n", sig)
	return nil
}

func checkBalance(client *rpc.Client, wallet solana.PublicKey) float64 {
	balance, err := client.GetBalance(
		context.Background(),
		wallet,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		return 0
	}
	return float64(balance.Value) / 1e9 // Convert lamports to SOL
}

func fetchRaydiumPairs() (RaydiumResponse, error) {
	log.Println("Fetching Raydium pairs...")

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			MaxConnsPerHost:    100,
			DisableKeepAlives:  false,
		},
	}

	req, err := http.NewRequest("GET", "https://api.raydium.io/v2/main/pairs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add more specific headers to ensure proper encoding
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")

	log.Printf("Sending request to Raydium API...")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create a reader based on the content encoding
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Read and decode with explicit UTF-8 handling
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the first few bytes of the response for debugging
	if len(body) > 0 {
		log.Printf("First 100 bytes of response: %s", string(body[:min(100, len(body))]))
	}

	var pairs RaydiumResponse
	if err := json.Unmarshal(body, &pairs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Successfully parsed %d pairs", len(pairs))
	return pairs, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fetchPoolInfo(tokenMint string) (*RaydiumPool, error) {
	pairs, err := fetchRaydiumPairs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pairs: %w", err)
	}

	for _, pair := range pairs {
		if pair.Pool.BaseMint == tokenMint || pair.Pool.QuoteMint == tokenMint {
			return &pair.Pool, nil
		}
	}

	return nil, fmt.Errorf("pool not found for token: %s", tokenMint)
}

func createSwapInstruction(
	programID solana.PublicKey,
	ammId solana.PublicKey,
	userSourceTokenAccount solana.PublicKey,
	poolSourceTokenAccount solana.PublicKey,
	poolDestinationTokenAccount solana.PublicKey,
	userDestinationTokenAccount solana.PublicKey,
	lpMint solana.PublicKey,
	feeAccount solana.PublicKey,
	userAuthority solana.PublicKey,
	amountIn uint64,
	minAmountOut uint64,
) solana.Instruction {
	data := make([]byte, 10)
	data[0] = 9 // Swap instruction code
	binary.LittleEndian.PutUint64(data[1:], amountIn)
	data[9] = uint8(minAmountOut)

	accounts := solana.AccountMetaSlice{
		{PublicKey: ammId, IsSigner: false, IsWritable: true},
		{PublicKey: userAuthority, IsSigner: true, IsWritable: false},
		{PublicKey: userSourceTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: poolSourceTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: poolDestinationTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: userDestinationTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: lpMint, IsSigner: false, IsWritable: false},
		{PublicKey: feeAccount, IsSigner: false, IsWritable: true},
	}

	return solana.NewInstruction(programID, accounts, data)
}
