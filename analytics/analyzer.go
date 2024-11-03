package analytics

type TokenAnalyzerConfig struct {
	MinLiquidity   float64
	MinHolderCount int
	MaxTopHolder   float64
	MinAge         int64
}

type TokenAnalyzer struct {
	config TokenAnalyzerConfig
}

func NewTokenAnalyzer(config TokenAnalyzerConfig) *TokenAnalyzer {
	return &TokenAnalyzer{
		config: config,
	}
}
