package analytics

import (
	"fmt"
	"time"
)

type TokenMetrics struct {
	Price          float64
	Volume24h      float64
	MarketCap      float64
	Liquidity      float64
	HolderCount    int
	TopHolderShare float64
	CreationTime   time.Time
	LastTradeTime  time.Time
}

type TokenAnalyzerConfig struct {
	MinLiquidity   float64
	MinHolderCount int
	MaxTopHolder   float64
	MinAge         time.Duration
}

type TokenAnalyzer struct {
	minLiquidity   float64
	minHolderCount int
	maxTopHolder   float64
	minAge         time.Duration
}

func NewTokenAnalyzer(config TokenAnalyzerConfig) *TokenAnalyzer {
	return &TokenAnalyzer{
		minLiquidity:   config.MinLiquidity,
		minHolderCount: config.MinHolderCount,
		maxTopHolder:   config.MaxTopHolder,
		minAge:         config.MinAge,
	}
}

func (a *TokenAnalyzer) AnalyzeToken(metrics TokenMetrics) (bool, []string) {
	var issues []string

	if metrics.Liquidity < a.minLiquidity {
		issues = append(issues, fmt.Sprintf("Low liquidity: %.2f < %.2f",
			metrics.Liquidity, a.minLiquidity))
	}

	if metrics.HolderCount < a.minHolderCount {
		issues = append(issues, fmt.Sprintf("Few holders: %d < %d",
			metrics.HolderCount, a.minHolderCount))
	}

	if metrics.TopHolderShare > a.maxTopHolder {
		issues = append(issues, fmt.Sprintf("High concentration: %.2f%% > %.2f%%",
			metrics.TopHolderShare*100, a.maxTopHolder*100))
	}

	tokenAge := time.Since(metrics.CreationTime)
	if tokenAge < a.minAge {
		issues = append(issues, fmt.Sprintf("Too new: %s < %s",
			tokenAge.String(), a.minAge.String()))
	}

	return len(issues) == 0, issues
}
