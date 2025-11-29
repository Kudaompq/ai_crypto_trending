package indicator

import (
	"github.com/markcheno/go-talib"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// CalculateMACD calculates MACD indicator
// DIF = EMA(12) - EMA(26)
// DEA = EMA(DIF, 9)
// Histogram = (DIF - DEA) * 2
func CalculateMACD(candles []model.Candle) model.MACDIndicator {
	if len(candles) < 26 {
		return model.MACDIndicator{}
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate MACD using TA-Lib
	macd, signal, hist := talib.Macd(closes, 12, 26, 9)

	// Get the latest values
	lastIdx := len(macd) - 1
	
	return model.MACDIndicator{
		DIF:       macd[lastIdx],
		DEA:       signal[lastIdx],
		Histogram: hist[lastIdx],
	}
}
