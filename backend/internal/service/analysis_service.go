package service

import (
	"github.com/kudaompq/ai_trending/backend/internal/indicator"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

// AnalysisService orchestrates the complete analysis
type AnalysisService struct {
	binanceRepo           *repository.BinanceRepository
	trendService          *TrendService
	marketStructureService *MarketStructureService
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService() *AnalysisService {
	return &AnalysisService{
		binanceRepo:           repository.NewBinanceRepository(),
		trendService:          NewTrendService(),
		marketStructureService: NewMarketStructureService(),
	}
}

// PerformAnalysis performs complete analysis for a symbol
func (s *AnalysisService) PerformAnalysis(symbol, interval string, limit int) (*model.AnalysisResult, error) {
	// Fetch K-line data
	candles, err := s.binanceRepo.GetKlines(symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	if len(candles) == 0 {
		return nil, err
	}

	// Calculate indicators
	macd := indicator.CalculateMACD(candles)
	kdj := indicator.CalculateKDJWithHistory(candles)
	rsi := indicator.CalculateRSI(candles)

	indicators := model.Indicators{
		MACD: macd,
		KDJ:  kdj,
		RSI:  rsi,
	}

	// Analyze trend
	trend := s.trendService.AnalyzeTrend(candles)
	trendDirection := s.trendService.DetermineTrendDirection(candles)

	// Calculate SR levels
	srLevels := indicator.CalculateSRLevels(candles, 100)

	// Identify candlestick patterns
	patterns := indicator.IdentifyPatterns(candles, trendDirection)

	// Analyze market structure
	marketStructure := s.marketStructureService.AnalyzeStructure(candles, trendDirection)

	// Build result
	result := &model.AnalysisResult{
		Symbol:              symbol,
		Interval:            interval,
		Timestamp:           candles[len(candles)-1].Timestamp,
		Trend:               trend,
		Indicators:          indicators,
		SRLevels:            srLevels,
		CandlestickPatterns: patterns,
		MarketStructure:     marketStructure,
	}

	return result, nil
}
