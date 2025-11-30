package indicator

// EMAResult represents the EMA calculation result
type EMAResult struct {
	Values []float64 // EMA values
	Period int       // Period used for calculation
}

// CalculateEMA calculates the Exponential Moving Average
// EMA gives more weight to recent prices
func CalculateEMA(prices []float64, period int) *EMAResult {
	if len(prices) < period {
		return &EMAResult{
			Values: []float64{},
			Period: period,
		}
	}

	emaValues := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// First EMA is simple moving average
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	emaValues[period-1] = sum / float64(period)

	// Calculate subsequent EMA values
	// EMA = (Price - Previous EMA) * Multiplier + Previous EMA
	for i := period; i < len(prices); i++ {
		emaValues[i] = (prices[i]-emaValues[i-1])*multiplier + emaValues[i-1]
	}

	return &EMAResult{
		Values: emaValues,
		Period: period,
	}
}

// CalculateMultipleEMA calculates EMA for multiple periods
func CalculateMultipleEMA(prices []float64, periods []int) map[int]*EMAResult {
	results := make(map[int]*EMAResult)
	for _, period := range periods {
		results[period] = CalculateEMA(prices, period)
	}
	return results
}

// GetCurrentEMA returns the most recent EMA value
func (r *EMAResult) GetCurrentEMA() float64 {
	if len(r.Values) == 0 {
		return 0
	}
	return r.Values[len(r.Values)-1]
}

// GetEMAAtIndex returns the EMA value at a specific index
func (r *EMAResult) GetEMAAtIndex(index int) float64 {
	if index < 0 || index >= len(r.Values) {
		return 0
	}
	return r.Values[index]
}

// IsBullishAlignment checks if EMAs are in bullish order (shorter > longer)
func IsBullishAlignment(ema9, ema21, ema50, ema200 float64) bool {
	return ema9 > ema21 && ema21 > ema50 && ema50 > ema200
}

// IsBearishAlignment checks if EMAs are in bearish order (shorter < longer)
func IsBearishAlignment(ema9, ema21, ema50, ema200 float64) bool {
	return ema9 < ema21 && ema21 < ema50 && ema50 < ema200
}

// DetectCrossover detects if a crossover occurred between two EMAs
// Returns: 1 for golden cross (fast crosses above slow), -1 for death cross, 0 for no cross
func DetectCrossover(fastEMA, slowEMA *EMAResult) int {
	if len(fastEMA.Values) < 2 || len(slowEMA.Values) < 2 {
		return 0
	}

	currentFast := fastEMA.GetCurrentEMA()
	currentSlow := slowEMA.GetCurrentEMA()
	prevFast := fastEMA.GetEMAAtIndex(len(fastEMA.Values) - 2)
	prevSlow := slowEMA.GetEMAAtIndex(len(slowEMA.Values) - 2)

	// Golden cross: fast crosses above slow
	if prevFast <= prevSlow && currentFast > currentSlow {
		return 1
	}

	// Death cross: fast crosses below slow
	if prevFast >= prevSlow && currentFast < currentSlow {
		return -1
	}

	return 0
}
