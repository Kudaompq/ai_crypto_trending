package indicator

import (
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// FibonacciLevels represents Fibonacci retracement and extension levels
type FibonacciLevels struct {
	High        float64            `json:"high"`        // Swing high
	Low         float64            `json:"low"`         // Swing low
	Retracement map[string]float64 `json:"retracement"` // Retracement levels
	Extension   map[string]float64 `json:"extension"`   // Extension levels
	Direction   string             `json:"direction"`   // "UPTREND" or "DOWNTREND"
}

// CalculateFibonacciLevels calculates Fibonacci retracement and extension levels
func CalculateFibonacciLevels(high, low float64, isUptrend bool) *FibonacciLevels {
	diff := high - low

	// Fibonacci ratios
	retracementRatios := map[string]float64{
		"0%":    0.0,
		"23.6%": 0.236,
		"38.2%": 0.382,
		"50%":   0.5,
		"61.8%": 0.618,
		"78.6%": 0.786,
		"100%":  1.0,
	}

	extensionRatios := map[string]float64{
		"1.272": 1.272,
		"1.618": 1.618,
		"2.0":   2.0,
		"2.618": 2.618,
	}

	retracement := make(map[string]float64)
	extension := make(map[string]float64)

	direction := "UPTREND"
	if !isUptrend {
		direction = "DOWNTREND"
	}

	// Calculate retracement levels
	if isUptrend {
		// For uptrend: retracement from high
		for label, ratio := range retracementRatios {
			retracement[label] = high - (diff * ratio)
		}
		// Extension levels above high
		for label, ratio := range extensionRatios {
			extension[label] = low + (diff * ratio)
		}
	} else {
		// For downtrend: retracement from low
		for label, ratio := range retracementRatios {
			retracement[label] = low + (diff * ratio)
		}
		// Extension levels below low
		for label, ratio := range extensionRatios {
			extension[label] = high - (diff * ratio)
		}
	}

	return &FibonacciLevels{
		High:        high,
		Low:         low,
		Retracement: retracement,
		Extension:   extension,
		Direction:   direction,
	}
}

// FindSwingHighLow finds the most recent swing high and low in the candles
func FindSwingHighLow(candles []model.Candle, lookback int) (high, low float64, highIndex, lowIndex int) {
	if len(candles) < lookback {
		lookback = len(candles)
	}

	startIndex := len(candles) - lookback
	high = candles[startIndex].High
	low = candles[startIndex].Low
	highIndex = startIndex
	lowIndex = startIndex

	for i := startIndex; i < len(candles); i++ {
		if candles[i].High > high {
			high = candles[i].High
			highIndex = i
		}
		if candles[i].Low < low {
			low = candles[i].Low
			lowIndex = i
		}
	}

	return high, low, highIndex, lowIndex
}

// CalculateFibonacciFromCandles calculates Fibonacci levels from candle data
func CalculateFibonacciFromCandles(candles []model.Candle, lookback int) *FibonacciLevels {
	if len(candles) < 2 {
		return nil
	}

	high, low, highIndex, lowIndex := FindSwingHighLow(candles, lookback)

	// Determine trend direction based on which came first
	isUptrend := lowIndex < highIndex

	return CalculateFibonacciLevels(high, low, isUptrend)
}

// GetKeyRetracementLevels returns the most important retracement levels
func (f *FibonacciLevels) GetKeyRetracementLevels() []float64 {
	return []float64{
		f.Retracement["38.2%"],
		f.Retracement["50%"],
		f.Retracement["61.8%"],
	}
}

// GetKeyExtensionLevels returns the most important extension levels
func (f *FibonacciLevels) GetKeyExtensionLevels() []float64 {
	return []float64{
		f.Extension["1.272"],
		f.Extension["1.618"],
		f.Extension["2.0"],
	}
}

// IsNearLevel checks if a price is near a Fibonacci level (within threshold %)
func (f *FibonacciLevels) IsNearLevel(price float64, threshold float64) (bool, string) {
	// Check retracement levels
	for label, level := range f.Retracement {
		diff := abs(price - level)
		if diff/level < threshold {
			return true, "Retracement " + label
		}
	}

	// Check extension levels
	for label, level := range f.Extension {
		diff := abs(price - level)
		if diff/level < threshold {
			return true, "Extension " + label
		}
	}

	return false, ""
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
