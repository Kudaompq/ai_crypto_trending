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

	// Calculate existing indicators
	macd := indicator.CalculateMACD(candles)
	kdj := indicator.CalculateKDJWithHistory(candles)
	rsi := indicator.CalculateRSI(candles)

	// Calculate new indicators
	// ATR (14-period)
	atrResult := indicator.CalculateATR(candles, 14)
	atrIndicator := model.ATRIndicator{
		Value:  atrResult.GetCurrentATR(),
		Period: 14,
	}

	// EMA (multiple periods: 9, 21, 50, 200)
	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}

	emaResults := indicator.CalculateMultipleEMA(closePrices, []int{9, 21, 50, 200})
	emaIndicator := model.EMAIndicator{
		EMA9:   emaResults[9].GetCurrentEMA(),
		EMA21:  emaResults[21].GetCurrentEMA(),
		EMA50:  emaResults[50].GetCurrentEMA(),
		EMA200: emaResults[200].GetCurrentEMA(),
	}

	// Fibonacci levels (using last 100 candles for swing high/low)
	var fibLevels *model.FibonacciLevels
	if len(candles) >= 50 {
		lookback := 100
		if len(candles) < lookback {
			lookback = len(candles)
		}
		fibResult := indicator.CalculateFibonacciFromCandles(candles, lookback)
		if fibResult != nil {
			fibLevels = &model.FibonacciLevels{
				High:        fibResult.High,
				Low:         fibResult.Low,
				Retracement: fibResult.Retracement,
				Extension:   fibResult.Extension,
				Direction:   fibResult.Direction,
			}
		}
	}

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
		Symbol:    symbol,
		Interval:  interval,
		Timestamp: candles[len(candles)-1].Timestamp,
		Trend:     trend,
		Indicators: model.Indicators{
			MACD:      macd,
			KDJ:       kdj,
			RSI:       rsi,
			ATR:       atrIndicator,
			EMA:       emaIndicator,
			Fibonacci: fibLevels,
		},
		SRLevels:            srLevels,
		CandlestickPatterns: patterns,
		MarketStructure:     marketStructure,
	}, nil
}
