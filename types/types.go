package types

import (
	"time"

	"github.com/gagliardetto/solana-go"
)

type PoolAccounts struct {
	BaseVault  solana.PublicKey
	QuoteVault solana.PublicKey
	FeeAccount solana.PublicKey
}

type SocialMetrics struct {
	TwitterFollowers int
	TelegramMembers  int
	WebsiteExists    bool
	GitHubExists     bool
	HasWhitepaper    bool
}

type RaydiumPair struct {
	Name         string      `json:"name"`
	Symbol       string      `json:"symbol"`
	Address      string      `json:"address"`
	Timestamp    string      `json:"timestamp"`
	Pool         RaydiumPool `json:"pool"`
	Market       string      `json:"market"`
	Liquidity    float64     `json:"liquidity"`
	Price        float64     `json:"price"`
	Volume24h    float64     `json:"volume24h"`
	MarketCap    float64     `json:"marketCap"`
	TokenAmount  float64     `json:"tokenAmount"`
	TokenAddress string      `json:"tokenAddress"`
}

type RaydiumPool struct {
	AmmId           string  `json:"ammId"`
	LpMint          string  `json:"lpMint"`
	BaseMint        string  `json:"baseMint"`
	QuoteMint       string  `json:"quoteMint"`
	BaseDecimals    int     `json:"baseDecimals"`
	QuoteDecimals   int     `json:"quoteDecimals"`
	LpDecimals      int     `json:"lpDecimals"`
	Version         int     `json:"version"`
	Status          int     `json:"status"`
	PriceKey        string  `json:"priceKey"`
	TokenAmountCoin float64 `json:"tokenAmountCoin"`
	TokenAmountPc   float64 `json:"tokenAmountPc"`
}

const (
	PHANTOM_WALLET_ADDRESS = "79hjkpSwnJ4g7PJ7YYQfJRGEwHwWWUB7ziyve15fC4YC"
	MIN_LIQUIDITY_USD      = 500.0
	MAX_MARKET_CAP_USD     = 1000000.0
	MIN_HOLDER_COUNT       = 100
	MIN_PRICE              = 0.0
	MAX_PRICE              = 1.0
	MIN_MARKET_AGE         = 1 * time.Hour
	MAX_MARKET_AGE         = 24 * time.Hour
	FETCH_INTERVAL_SECONDS = 5
	MAX_TOKENS_TO_TRACK    = 10 // Maximum number of new tokens to track at once
)
