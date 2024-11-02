package config

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	// Network settings
	RPCEndpoint string `json:"rpc_endpoint"`
	WSEndpoint  string `json:"ws_endpoint"`
	NetworkType string `json:"network_type"` // "mainnet", "devnet", "testnet"

	// Trading parameters
	MinLiquidity  float64       `json:"min_liquidity"`
	MaxSlippage   float64       `json:"max_slippage"`
	TradeAmount   float64       `json:"trade_amount"`
	FetchInterval time.Duration `json:"fetch_interval"`

	// Safety thresholds
	MinHolders   int           `json:"min_holders"`
	MaxTopHolder float64       `json:"max_top_holder"`
	MinLockTime  time.Duration `json:"min_lock_time"`

	// API Keys
	EtherscanKey   string `json:"etherscan_key"`
	TelegramBotKey string `json:"telegram_bot_key"`

	// Notification settings
	TelegramChatID string `json:"telegram_chat_id"`
	EnableDiscord  bool   `json:"enable_discord"`
	DiscordWebhook string `json:"discord_webhook"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	return config, err
}
