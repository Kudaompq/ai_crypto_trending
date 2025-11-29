package indicator

import (
	"github.com/markcheno/go-talib"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// CalculateRSI calculates RSI indicator for both 6 and 14 periods
// RSI = 100 - 100/(1 + RS)
// RS = Average Gain / Average Loss
func CalculateRSI(candles []model.Candle) model.RSIIndicator {
	if len(candles) < 14 {
		return model.RSIIndicator{}
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate RSI using TA-Lib
	rsi6 := talib.Rsi(closes, 6)
	rsi14 := talib.Rsi(closes, 14)

	// Get the latest values
	return model.RSIIndicator{
		RSI6:  rsi6[len(rsi6)-1],
		RSI14: rsi14[len(rsi14)-1],
	}
}
