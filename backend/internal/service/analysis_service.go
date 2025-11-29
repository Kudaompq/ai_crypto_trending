package service

import (
	"fmt"
	
	"github.com/kudaompq/ai_trending/backend/internal/indicator"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

// AnalysisService orchestrates the complete analysis
type AnalysisService struct {
	binanceRepo            *repository.BinanceRepository
	trendService           *TrendService
	marketStructureService *MarketStructureService
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService() *AnalysisService {
	return &AnalysisService{
		binanceRepo:            repository.NewBinanceRepository(),
		trendService:           NewTrendService(),
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

	if len(candles) < 20 {
		return nil, fmt.Errorf("insufficient data: need at least 20 candles")
	}

	// Calculate indicators
	macd := indicator.CalculateMACD(candles)
	kdj := indicator.CalculateKDJWithHistory(candles)
	rsi := indicator.CalculateRSI(candles)

	// Analyze trend
	trend := s.trendService.AnalyzeTrend(candles)

	// Calculate SR levels with interval awareness
	srLevels := indicator.CalculateSRLevelsWithInterval(candles, limit, interval)

	// Identify candlestick patterns
	trendDirection := s.trendService.DetermineTrendDirection(candles)
	patterns := indicator.IdentifyPatterns(candles, trendDirection)

	// Analyze market structure
	marketStructure := s.marketStructureService.AnalyzeStructure(candles, trendDirection)

	return &model.AnalysisResult{
		Symbol:              symbol,
		Interval:            interval,
		Timestamp:           candles[len(candles)-1].Timestamp,
		Trend:               trend,
		Indicators:          model.Indicators{MACD: macd, KDJ: kdj, RSI: rsi},
		SRLevels:            srLevels,
		CandlestickPatterns: patterns,
		MarketStructure:     marketStructure,
	}, nil
}
