package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	TelegramBotKey string  `json:"telegramBotKey"`
	TelegramChatID string  `json:"telegramChatId"`
	MinLiquidity   float64 `json:"minLiquidity"`
	MinHolders     int     `json:"minHolders"`
	MaxTopHolder   float64 `json:"maxTopHolder"`
	MinLockTime    int64   `json:"minLockTime"`
}

func LoadConfig(filepath string) (*Config, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
