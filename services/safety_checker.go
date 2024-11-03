package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TokenMetrics struct {
	Liquidity float64
	Volume24h float64
	MarketCap float64
	// Add other needed fields
}

type TokenSafetyMetrics struct {
	LiquidityLocked   bool
	LiquidityLockTime time.Duration
	IsHoneypot        bool
	TopHolderShare    float64
	HolderCount       int
	SocialMetrics     SocialMetrics
}

func FetchTokenMetrics(pair RaydiumPair) (*TokenMetrics, error) {
	// Solscan API endpoint for token metrics
	url := fmt.Sprintf("https://public-api.solscan.io/token/meta?tokenAddress=%s", pair.TokenAddress)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token metrics: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			MarketCap      string  `json:"marketCap"`
			Volume24h      string  `json:"volume24h"`
			PriceChange24h float64 `json:"priceChange24h"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	marketCap, _ := strconv.ParseFloat(result.Data.MarketCap, 64)
	volume24h, _ := strconv.ParseFloat(result.Data.Volume24h, 64)

	return &TokenMetrics{
		Liquidity: pair.Liquidity, // Keep from Raydium as it's more accurate
		Volume24h: volume24h,
		MarketCap: marketCap,
	}, nil
}

func RunSafetyChecks(tokenAddress string) (bool, string) {
	log.Printf("Running safety checks for token: %s", tokenAddress)

	// Check liquidity lock
	locked, lockDuration, err := CheckLiquidityLock(tokenAddress)
	if err != nil || !locked {
		return false, "Liquidity not locked"
	}

	// Validate lock parameters
	if !ValidateLockParameters(lockDuration, 80.0) { // Assuming 80% minimum lock
		return false, "Lock parameters invalid"
	}

	// Check for honeypot
	isHoneypot, err := DetectHoneypot(tokenAddress)
	if err != nil || isHoneypot {
		return false, "Detected honeypot characteristics"
	}

	// Analyze holders
	topHolderShare, holderCount, err := AnalyzeHolders(tokenAddress)
	if err != nil {
		return false, "Failed to analyze holders"
	}
	if topHolderShare > 0.15 { // 15% max for top holder
		return false, fmt.Sprintf("Top holder owns too much: %.1f%%", topHolderShare*100)
	}
	if holderCount < 100 { // Minimum 100 holders
		return false, fmt.Sprintf("Too few holders: %d", holderCount)
	}

	// Check social presence
	social := CheckSocialPresence(tokenAddress)
	socialScore := 0
	if social.TwitterFollowers > 100 {
		socialScore++
	}
	if social.TelegramMembers > 100 {
		socialScore++
	}
	if social.WebsiteExists {
		socialScore++
	}
	if social.GitHubExists {
		socialScore++
	}
	if social.HasWhitepaper {
		socialScore++
	}

	if socialScore < 2 { // Require at least 2 social criteria
		return false, "Insufficient social presence"
	}

	return true, ""
}

func CalculateTokenScore(metrics TokenMetrics, safety TokenSafetyMetrics) float64 {
	// Base score from metrics
	score := 0.0

	// Liquidity weight (higher is better, up to a point)
	liquidityScore := math.Min(metrics.Liquidity/MIN_LIQUIDITY_USD, 5.0)
	score += liquidityScore * 20

	// Volume/Liquidity ratio (higher is better, indicates trading activity)
	if metrics.Liquidity > 0 {
		volumeRatio := metrics.Volume24h / metrics.Liquidity
		score += math.Min(volumeRatio*50, 100)
	}

	// Market cap (lower is better, more room to grow)
	marketCapScore := 1.0 - (metrics.MarketCap / MAX_MARKET_CAP_USD)
	score += marketCapScore * 30

	// Safety multipliers
	safetyMultiplier := 1.0

	// Holder count bonus
	holderScore := float64(safety.HolderCount) / float64(MIN_HOLDER_COUNT)
	safetyMultiplier *= math.Min(holderScore, 2.0)

	// Top holder penalty (lower is better)
	if safety.TopHolderShare > 0.5 { // More than 50% held by top holder
		safetyMultiplier *= 0.5
	}

	// Liquidity lock bonus
	if safety.LiquidityLocked {
		safetyMultiplier *= 1.2
	}

	// Social presence bonus
	socialScore := 0.0
	if safety.SocialMetrics.WebsiteExists {
		socialScore += 0.1
	}
	if safety.SocialMetrics.GitHubExists {
		socialScore += 0.1
	}
	if safety.SocialMetrics.HasWhitepaper {
		socialScore += 0.1
	}
	safetyMultiplier *= (1.0 + socialScore)

	return score * safetyMultiplier
}

func CheckSocialPresence(tokenAddress string) SocialMetrics {
	log.Printf("Checking social presence for token: %s", tokenAddress)

	// Initialize metrics
	metrics := SocialMetrics{}

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Check Twitter using Twitter API v2
	twitterEndpoint := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", tokenAddress)
	req, _ := http.NewRequest("GET", twitterEndpoint, nil)
	req.Header.Add("Authorization", "Bearer YOUR_TWITTER_API_KEY")
	if resp, err := client.Do(req); err == nil {
		var result struct {
			Data struct {
				PublicMetrics struct {
					FollowersCount int `json:"followers_count"`
				} `json:"public_metrics"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		metrics.TwitterFollowers = result.Data.PublicMetrics.FollowersCount
		resp.Body.Close()
	}

	// Check Telegram
	telegramEndpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMembersCount?chat_id=@%s",
		"YOUR_TELEGRAM_BOT_TOKEN", tokenAddress)
	if resp, err := client.Get(telegramEndpoint); err == nil {
		var result struct {
			Ok     bool `json:"ok"`
			Result int  `json:"result"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		if result.Ok {
			metrics.TelegramMembers = result.Result
		}
		resp.Body.Close()
	}

	// Check Website existence
	websiteUrl := fmt.Sprintf("https://%s.io", tokenAddress)
	if resp, err := client.Head(websiteUrl); err == nil {
		metrics.WebsiteExists = resp.StatusCode == http.StatusOK
		resp.Body.Close()
	}

	// Check GitHub
	githubEndpoint := fmt.Sprintf("https://api.github.com/repos/%s", tokenAddress)
	if resp, err := client.Get(githubEndpoint); err == nil {
		metrics.GitHubExists = resp.StatusCode == http.StatusOK
		resp.Body.Close()
	}

	// Check for whitepaper
	whitepaperUrls := []string{
		fmt.Sprintf("https://%s.io/whitepaper.pdf", tokenAddress),
		fmt.Sprintf("https://%s.io/docs/whitepaper.pdf", tokenAddress),
	}
	for _, url := range whitepaperUrls {
		if resp, err := client.Head(url); err == nil && resp.StatusCode == http.StatusOK {
			metrics.HasWhitepaper = true
			resp.Body.Close()
			break
		}
	}

	log.Printf("Social metrics for %s: %+v", tokenAddress, metrics)
	return metrics
}

func AnalyzeHolders(tokenAddress string) (float64, int, error) {
	// Solscan API endpoint for token holders
	url := fmt.Sprintf("https://public-api.solscan.io/token/holders?tokenAddress=%s&limit=100", tokenAddress)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add required headers
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch holders: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			TotalHolders int `json:"total"`
			Items        []struct {
				Amount string `json:"amount"`
				Owner  string `json:"owner"`
				Rank   int    `json:"rank"`
				Share  string `json:"share"`
			} `json:"items"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Get top holder share
	var topHolderShare float64
	if len(result.Data.Items) > 0 {
		share, err := strconv.ParseFloat(strings.TrimSuffix(result.Data.Items[0].Share, "%"), 64)
		if err == nil {
			topHolderShare = share / 100 // Convert percentage to decimal
		}
	}

	return topHolderShare, result.Data.TotalHolders, nil
}

func CheckTokenSafety(address string) (TokenSafetyMetrics, error) {
	safety := TokenSafetyMetrics{}
	// Check liquidity lock status
	locked, lockDuration, err := CheckLiquidityLock(address)
	if err != nil {
		return safety, fmt.Errorf("failed to check liquidity lock: %w", err)
	}
	safety.LiquidityLocked = locked
	safety.LiquidityLockTime = lockDuration

	// Check for honeypot characteristics
	isHoneypot, err := DetectHoneypot(address)
	if err != nil {
		return safety, fmt.Errorf("failed to check honeypot: %w", err)
	}
	safety.IsHoneypot = isHoneypot

	// Analyze token distribution
	topHolder, holderCount, err := AnalyzeHolders(address)
	if err != nil {
		return safety, fmt.Errorf("failed to analyze holders: %w", err)
	}
	safety.TopHolderShare = topHolder
	safety.HolderCount = holderCount

	// Check social presence
	safety.SocialMetrics = CheckSocialPresence(address)

	return safety, nil
}

func AnalyzeTokenPotential(metrics TokenMetrics, safety TokenSafetyMetrics) (bool, string) {
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

func ValidateLockParameters(lockDuration time.Duration, percentage float64) bool {
	const (
		MIN_LOCK_DURATION   = 30 * 24 * time.Hour // 30 days
		MIN_LOCK_PERCENTAGE = 80.0                // 80%
	)

	if lockDuration < MIN_LOCK_DURATION {
		log.Printf("Warning: Lock duration too short: %v < %v",
			lockDuration.Round(time.Hour),
			MIN_LOCK_DURATION)
		return false
	}

	if percentage < MIN_LOCK_PERCENTAGE {
		log.Printf("Warning: Lock percentage too low: %.2f%% < %.2f%%",
			percentage,
			MIN_LOCK_PERCENTAGE)
		return false
	}

	return true
}

func CheckLiquidityLock(tokenAddress string) (bool, time.Duration, error) {
	// GoPlus API endpoint for Solana token security
	url := fmt.Sprintf("https://api.gopluslabs.io/api/v1/token_security/solana?contract_addresses=%s", tokenAddress)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers if needed (check GoPlus documentation for any required API keys)
	// req.Header.Add("X-API-KEY", "your-api-key")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode response
	var result GoPlusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we got a valid response
	if result.Code != 1 {
		return false, 0, fmt.Errorf("API error: %s", result.Message)
	}

	// Get lock info
	lockInfo := result.Data.Solana.LockInfo

	// If not locked, return immediately
	if !lockInfo.IsLocked {
		return false, 0, nil
	}

	// Parse end time
	endTime, err := time.Parse("2006-01-02 15:04:05", lockInfo.EndTime)
	if err != nil {
		return true, 0, fmt.Errorf("failed to parse lock end time: %w", err)
	}

	// Calculate remaining lock duration
	remainingDuration := time.Until(endTime)
	if remainingDuration < 0 {
		return false, 0, nil // Lock has expired
	}

	// Log detailed information
	log.Printf("Token %s liquidity lock info:", tokenAddress)
	log.Printf("Locked Amount: %s", lockInfo.LockedAmount)
	log.Printf("Lock Percentage: %.2f%%", lockInfo.Percentage)
	log.Printf("Lock End Time: %s", endTime.Format(time.RFC3339))
	log.Printf("Remaining Duration: %s", remainingDuration.Round(time.Hour))

	return true, remainingDuration, nil
}

func DetectHoneypot(tokenAddress string) (bool, error) {
	// GoPlus API endpoint for Solana token security
	url := fmt.Sprintf("https://api.gopluslabs.io/api/v1/token_security/solana?contract_addresses=%s", tokenAddress)

	// Make request with retry handling
	resp, err := MakeGoPlusRequest(url)
	if err != nil {
		return false, fmt.Errorf("failed to fetch security info: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    map[string]struct {
			IsSellable     string `json:"is_sellable"`
			SellTax        string `json:"sell_tax"`
			BuyTax         string `json:"buy_tax"`
			TransferPaused string `json:"transfer_pausable"`
			IsBlacklisted  string `json:"is_blacklisted"`
			IsProxy        string `json:"is_proxy"`
			IsHoneypot     string `json:"is_honeypot"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we got a valid response
	if result.Code != 1 {
		return false, fmt.Errorf("API error: %s", result.Message)
	}

	// Get token data
	tokenData, exists := result.Data[strings.ToLower(tokenAddress)]
	if !exists {
		return false, fmt.Errorf("token data not found in response")
	}

	// Log detailed information
	log.Printf("Token %s security check:", tokenAddress)
	log.Printf("Sellable: %s", tokenData.IsSellable)
	log.Printf("Sell Tax: %s", tokenData.SellTax)
	log.Printf("Buy Tax: %s", tokenData.BuyTax)
	log.Printf("Transfer Paused: %s", tokenData.TransferPaused)
	log.Printf("Blacklisted: %s", tokenData.IsBlacklisted)
	log.Printf("Is Proxy: %s", tokenData.IsProxy)
	log.Printf("Is Honeypot: %s", tokenData.IsHoneypot)

	// Check for honeypot characteristics
	isHoneypot := false

	// Direct honeypot flag
	if tokenData.IsHoneypot == "1" {
		isHoneypot = true
	}

	// Not sellable
	if tokenData.IsSellable == "0" {
		isHoneypot = true
	}

	// High taxes (over 20%)
	if sellTax, err := strconv.ParseFloat(tokenData.SellTax, 64); err == nil && sellTax > 20.0 {
		isHoneypot = true
	}
	if buyTax, err := strconv.ParseFloat(tokenData.BuyTax, 64); err == nil && buyTax > 20.0 {
		isHoneypot = true
	}

	// Transfers paused
	if tokenData.TransferPaused == "1" {
		isHoneypot = true
	}

	// Blacklisted
	if tokenData.IsBlacklisted == "1" {
		isHoneypot = true
	}

	return isHoneypot, nil
}
