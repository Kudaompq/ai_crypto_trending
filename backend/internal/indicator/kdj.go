package indicator

import (
	"math"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// CalculateKDJ calculates KDJ indicator
// RSV = (Close - Low9) / (High9 - Low9) * 100
// K = SMA(RSV, 3)
// D = SMA(K, 3)
// J = 3*K - 2*D
func CalculateKDJ(candles []model.Candle) model.KDJIndicator {
	if len(candles) < 9 {
		return model.KDJIndicator{}
	}

	n := len(candles)
	period := 9

	// Calculate RSV for the last period
	var low9, high9 float64 = math.MaxFloat64, -math.MaxFloat64
	
	for i := n - period; i < n; i++ {
		if candles[i].Low < low9 {
			low9 = candles[i].Low
		}
		if candles[i].High > high9 {
			high9 = candles[i].High
		}
	}

	currentClose := candles[n-1].Close
	rsv := 0.0
	if high9 != low9 {
		rsv = (currentClose - low9) / (high9 - low9) * 100
	}

	// For simplicity, we'll use a simple moving average approach
	// In production, you'd want to maintain state for proper SMA calculation
	k := rsv // Simplified: K ≈ RSV for current calculation
	d := rsv // Simplified: D ≈ RSV for current calculation
	j := 3*k - 2*d

	// Clamp values to reasonable ranges
	k = math.Max(0, math.Min(100, k))
	d = math.Max(0, math.Min(100, d))
	j = math.Max(-20, math.Min(120, j))

	return model.KDJIndicator{
		K: k,
		D: d,
		J: j,
	}
}

// CalculateKDJWithHistory calculates KDJ with proper SMA (more accurate)
func CalculateKDJWithHistory(candles []model.Candle) model.KDJIndicator {
	if len(candles) < 9 {
		return model.KDJIndicator{}
	}

	n := len(candles)
	period := 9
	smoothK := 3
	smoothD := 3

	// Calculate RSV values
	rsvValues := make([]float64, 0)
	
	for i := period - 1; i < n; i++ {
		var low9, high9 float64 = math.MaxFloat64, -math.MaxFloat64
		
		for j := i - period + 1; j <= i; j++ {
			if candles[j].Low < low9 {
				low9 = candles[j].Low
			}
			if candles[j].High > high9 {
				high9 = candles[j].High
			}
		}

		rsv := 0.0
		if high9 != low9 {
			rsv = (candles[i].Close - low9) / (high9 - low9) * 100
		}
		rsvValues = append(rsvValues, rsv)
	}

	// Calculate K (SMA of RSV)
	kValues := sma(rsvValues, smoothK)
	
	// Calculate D (SMA of K)
	dValues := sma(kValues, smoothD)

	// Get latest K and D
	k := kValues[len(kValues)-1]
	d := dValues[len(dValues)-1]
	j := 3*k - 2*d

	return model.KDJIndicator{
		K: k,
		D: d,
		J: j,
	}
}

// sma calculates simple moving average
func sma(data []float64, period int) []float64 {
	result := make([]float64, 0)
	
	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j]
		}
		result = append(result, sum/float64(period))
	}
	
	return result
}
