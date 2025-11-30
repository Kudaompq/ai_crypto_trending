package indicator

import (
	"math"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// ATRResult represents the ATR calculation result
type ATRResult struct {
	Values []float64 // ATR values for each candle
	Period int       // Period used for calculation
}

// CalculateATR calculates the Average True Range (ATR) indicator
// ATR measures market volatility by decomposing the entire range of an asset price for that period
func CalculateATR(candles []model.Candle, period int) *ATRResult {
	if len(candles) < period+1 {
		return &ATRResult{
			Values: []float64{},
			Period: period,
		}
	}

	// Calculate True Range for each candle
	trueRanges := make([]float64, len(candles))
	for i := 0; i < len(candles); i++ {
		trueRanges[i] = calculateTrueRange(candles, i)
	}

	// Calculate ATR using exponential moving average
	atrValues := make([]float64, len(candles))

	// First ATR is simple average of first 'period' true ranges
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += trueRanges[i]
	}
	atrValues[period-1] = sum / float64(period)

	// Subsequent ATR values use exponential smoothing
	// ATR = ((Prior ATR * (period - 1)) + Current TR) / period
	for i := period; i < len(candles); i++ {
		atrValues[i] = ((atrValues[i-1] * float64(period-1)) + trueRanges[i]) / float64(period)
	}

	return &ATRResult{
		Values: atrValues,
		Period: period,
	}
}

// calculateTrueRange calculates the True Range for a single candle
// TR = max(high - low, abs(high - previous_close), abs(low - previous_close))
func calculateTrueRange(candles []model.Candle, index int) float64 {
	if index == 0 {
		// For first candle, TR is simply high - low
		return candles[index].High - candles[index].Low
	}

	high := candles[index].High
	low := candles[index].Low
	prevClose := candles[index-1].Close

	// Calculate three possible ranges
	highLow := high - low
	highPrevClose := math.Abs(high - prevClose)
	lowPrevClose := math.Abs(low - prevClose)

	// Return the maximum
	return math.Max(highLow, math.Max(highPrevClose, lowPrevClose))
}

// GetCurrentATR returns the most recent ATR value
func (r *ATRResult) GetCurrentATR() float64 {
	if len(r.Values) == 0 {
		return 0
	}
	return r.Values[len(r.Values)-1]
}

// GetATRAtIndex returns the ATR value at a specific index
func (r *ATRResult) GetATRAtIndex(index int) float64 {
	if index < 0 || index >= len(r.Values) {
		return 0
	}
	return r.Values[index]
}

// IsHighVolatility checks if current volatility is high compared to average
// Returns true if current ATR > average ATR * threshold
func (r *ATRResult) IsHighVolatility(threshold float64) bool {
	if len(r.Values) < r.Period*2 {
		return false
	}

	// Calculate average ATR over the last 'period' values
	sum := 0.0
	start := len(r.Values) - r.Period
	for i := start; i < len(r.Values); i++ {
		sum += r.Values[i]
	}
	avgATR := sum / float64(r.Period)

	currentATR := r.GetCurrentATR()
	return currentATR > avgATR*threshold
}
