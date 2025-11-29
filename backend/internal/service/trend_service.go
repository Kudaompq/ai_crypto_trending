package service

import (
	"math"

	"github.com/kudaompq/ai_trending/backend/internal/indicator"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// TrendService handles trend analysis
type TrendService struct{}

// NewTrendService creates a new trend service
func NewTrendService() *TrendService {
	return &TrendService{}
}

// AnalyzeTrend analyzes the trend based on multiple indicators
func (s *TrendService) AnalyzeTrend(candles []model.Candle) model.TrendAnalysis {
	if len(candles) < 26 {
		return model.TrendAnalysis{
			Direction:         "盘整",
			Strength:          0.5,
			ChangeProbability: 0.5,
		}
	}

	// Calculate indicators
	macd := indicator.CalculateMACD(candles)
	kdj := indicator.CalculateKDJWithHistory(candles)
	rsi := indicator.CalculateRSI(candles)

	// Score each indicator
	macdScore := s.scoreMACDTrend(macd)
	kdjScore := s.scoreKDJTrend(kdj)
	rsiScore := s.scoreRSITrend(rsi)

	// Weighted average (MACD: 40%, KDJ: 30%, RSI: 30%)
	trendScore := macdScore*0.4 + kdjScore*0.3 + rsiScore*0.3

	// Determine direction and strength
	direction := "盘整"
	strength := 0.5
	changeProbability := 0.5

	if trendScore > 0.6 {
		direction = "上升"
		strength = (trendScore - 0.5) * 2 // Scale to 0-1
		changeProbability = 1 - strength
	} else if trendScore < 0.4 {
		direction = "下降"
		strength = (0.5 - trendScore) * 2 // Scale to 0-1
		changeProbability = 1 - strength
	} else {
		direction = "盘整"
		strength = 1 - math.Abs(trendScore-0.5)*2
		changeProbability = 0.6 // Higher probability of change in consolidation
	}

	return model.TrendAnalysis{
		Direction:         direction,
		Strength:          math.Min(1.0, strength),
		ChangeProbability: math.Min(1.0, changeProbability),
	}
}

// scoreMACDTrend scores MACD for trend (0 = bearish, 0.5 = neutral, 1 = bullish)
func (s *TrendService) scoreMACDTrend(macd model.MACDIndicator) float64 {
	score := 0.5

	// DIF vs DEA
	if macd.DIF > macd.DEA {
		score += 0.2
	} else {
		score -= 0.2
	}

	// Histogram
	if macd.Histogram > 0 {
		score += math.Min(0.3, macd.Histogram/10) // Normalize
	} else {
		score -= math.Min(0.3, -macd.Histogram/10)
	}

	return math.Max(0, math.Min(1, score))
}

// scoreKDJTrend scores KDJ for trend (0 = bearish, 0.5 = neutral, 1 = bullish)
func (s *TrendService) scoreKDJTrend(kdj model.KDJIndicator) float64 {
	score := 0.5

	// K vs D
	if kdj.K > kdj.D {
		score += 0.2
	} else {
		score -= 0.2
	}

	// J value
	if kdj.J > 80 {
		score += 0.3
	} else if kdj.J < 20 {
		score -= 0.3
	} else {
		// Neutral zone
		score += (kdj.J - 50) / 100
	}

	return math.Max(0, math.Min(1, score))
}

// scoreRSITrend scores RSI for trend (0 = bearish, 0.5 = neutral, 1 = bullish)
func (s *TrendService) scoreRSITrend(rsi model.RSIIndicator) float64 {
	// Use RSI14 as primary, RSI6 as confirmation
	score := rsi.RSI14 / 100

	// Adjust based on RSI6
	if rsi.RSI6 > 70 && rsi.RSI14 > 60 {
		score += 0.1
	} else if rsi.RSI6 < 30 && rsi.RSI14 < 40 {
		score -= 0.1
	}

	return math.Max(0, math.Min(1, score))
}

// DetermineTrendDirection determines trend direction from candles
func (s *TrendService) DetermineTrendDirection(candles []model.Candle) string {
	if len(candles) < 20 {
		return "盘整"
	}

	// Simple trend detection using moving averages
	recentAvg := 0.0
	olderAvg := 0.0

	for i := len(candles) - 10; i < len(candles); i++ {
		recentAvg += candles[i].Close
	}
	recentAvg /= 10

	for i := len(candles) - 20; i < len(candles)-10; i++ {
		olderAvg += candles[i].Close
	}
	olderAvg /= 10

	diff := (recentAvg - olderAvg) / olderAvg

	if diff > 0.02 {
		return "上升"
	} else if diff < -0.02 {
		return "下降"
	}
	return "盘整"
}
