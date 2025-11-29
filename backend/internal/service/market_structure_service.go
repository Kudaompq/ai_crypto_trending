package service

import (
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// MarketStructureService handles market structure analysis
type MarketStructureService struct{}

// NewMarketStructureService creates a new market structure service
func NewMarketStructureService() *MarketStructureService {
	return &MarketStructureService{}
}

// AnalyzeStructure analyzes market structure (Higher High, Higher Low, etc.)
func (s *MarketStructureService) AnalyzeStructure(candles []model.Candle, trend string) model.MarketStructure {
	if len(candles) < 20 {
		return model.MarketStructure{
			HigherHigh:     false,
			HigherLow:      false,
			StructureBreak: false,
			RiskLevel:      "中",
		}
	}

	// Find recent swing highs and lows
	swingHighs := s.findSwingHighs(candles, 5)
	swingLows := s.findSwingLows(candles, 5)

	higherHigh := false
	higherLow := false
	lowerHigh := false
	lowerLow := false

	// Compare recent swings
	if len(swingHighs) >= 2 {
		if swingHighs[len(swingHighs)-1] > swingHighs[len(swingHighs)-2] {
			higherHigh = true
		} else {
			lowerHigh = true
		}
	}

	if len(swingLows) >= 2 {
		if swingLows[len(swingLows)-1] > swingLows[len(swingLows)-2] {
			higherLow = true
		} else {
			lowerLow = true
		}
	}

	// Determine structure break
	structureBreak := false
	if trend == "上升" && lowerLow {
		structureBreak = true
	} else if trend == "下降" && higherHigh {
		structureBreak = true
	}

	// Determine risk level
	riskLevel := s.calculateRiskLevel(trend, higherHigh, higherLow, lowerHigh, lowerLow, structureBreak)

	return model.MarketStructure{
		HigherHigh:     higherHigh,
		HigherLow:      higherLow,
		StructureBreak: structureBreak,
		RiskLevel:      riskLevel,
	}
}

// findSwingHighs finds swing high points
func (s *MarketStructureService) findSwingHighs(candles []model.Candle, lookback int) []float64 {
	swingHighs := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingHigh := true

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= candles[i].High {
				isSwingHigh = false
				break
			}
		}

		if isSwingHigh {
			swingHighs = append(swingHighs, candles[i].High)
		}
	}

	return swingHighs
}

// findSwingLows finds swing low points
func (s *MarketStructureService) findSwingLows(candles []model.Candle, lookback int) []float64 {
	swingLows := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingLow := true

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= candles[i].Low {
				isSwingLow = false
				break
			}
		}

		if isSwingLow {
			swingLows = append(swingLows, candles[i].Low)
		}
	}

	return swingLows
}

// calculateRiskLevel determines risk level based on market structure
func (s *MarketStructureService) calculateRiskLevel(trend string, hh, hl, lh, ll, structureBreak bool) string {
	if structureBreak {
		return "高"
	}

	if trend == "上升" {
		if hh && hl {
			return "低" // Healthy uptrend
		}
		return "中"
	} else if trend == "下降" {
		if lh && ll {
			return "低" // Healthy downtrend (low risk for shorts)
		}
		return "中"
	}

	return "中" // Consolidation
}
