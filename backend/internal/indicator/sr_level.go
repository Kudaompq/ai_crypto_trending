package indicator

import (
	"math"
	"sort"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// TimeframeConfig holds configuration for different timeframes
type TimeframeConfig struct {
	ClusterThreshold float64 // Percentage threshold for price clustering
	MinClusterSize   int     // Minimum points to form a cluster
	LookbackPeriod   int     // How many candles to analyze
	MinLevels        int     // Minimum SR levels to return
}

// getTimeframeConfig returns optimized config based on interval
func getTimeframeConfig(interval string, totalCandles int) TimeframeConfig {
	switch interval {
	case "15m":
		return TimeframeConfig{
			ClusterThreshold: 0.003, // 0.3% for 15min (tighter)
			MinClusterSize:   2,
			LookbackPeriod:   min(96, totalCandles),  // ~24 hours
			MinLevels:        2,
		}
	case "1h":
		return TimeframeConfig{
			ClusterThreshold: 0.005, // 0.5% for 1h
			MinClusterSize:   2,
			LookbackPeriod:   min(168, totalCandles), // ~1 week
			MinLevels:        2,
		}
	case "4h":
		return TimeframeConfig{
			ClusterThreshold: 0.01, // 1% for 4h
			MinClusterSize:   2,
			LookbackPeriod:   min(84, totalCandles), // ~2 weeks
			MinLevels:        2,
		}
	case "1d":
		return TimeframeConfig{
			ClusterThreshold: 0.02, // 2% for daily
			MinClusterSize:   2,
			LookbackPeriod:   min(60, totalCandles), // ~2 months
			MinLevels:        1,
		}
	default:
		return TimeframeConfig{
			ClusterThreshold: 0.015,
			MinClusterSize:   2,
			LookbackPeriod:   min(100, totalCandles),
			MinLevels:        2,
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PricePoint represents a single price point with volume
type PricePoint struct {
	Price  float64
	Volume float64
	IsHigh bool
}

// PriceCluster represents a cluster of prices
type PriceCluster struct {
	Price    float64
	Strength float64
	Count    int
	Volume   float64
}

// CalculateSRLevels calculates support and resistance levels with timeframe awareness
func CalculateSRLevels(candles []model.Candle, lookback int) model.SRLevels {
	return CalculateSRLevelsWithInterval(candles, lookback, "1d")
}

// CalculateSRLevelsWithInterval calculates SR levels with interval-specific parameters
func CalculateSRLevelsWithInterval(candles []model.Candle, lookback int, interval string) model.SRLevels {
	if len(candles) < 10 {
		return model.SRLevels{
			Resistance: []model.SRLevel{},
			Support:    []model.SRLevel{},
		}
	}

	config := getTimeframeConfig(interval, len(candles))
	if lookback > 0 && lookback < config.LookbackPeriod {
		config.LookbackPeriod = lookback
	}

	// Calculate ATR for dynamic threshold adjustment
	atr := calculateATR(candles, 14)
	currentPrice := candles[len(candles)-1].Close

	// Collect price points with volume weighting
	pricePoints := make([]PricePoint, 0)
	startIdx := len(candles) - config.LookbackPeriod
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(candles); i++ {
		pricePoints = append(pricePoints, PricePoint{
			Price:  candles[i].High,
			Volume: candles[i].Volume,
			IsHigh: true,
		})
		pricePoints = append(pricePoints, PricePoint{
			Price:  candles[i].Low,
			Volume: candles[i].Volume,
			IsHigh: false,
		})
	}

	// Sort by price
	sort.Slice(pricePoints, func(i, j int) bool {
		return pricePoints[i].Price < pricePoints[j].Price
	})

	// Dynamic threshold based on ATR and config
	dynamicThreshold := math.Max(
		currentPrice*config.ClusterThreshold,
		atr*0.5, // At least half ATR
	)

	// Cluster prices
	clusters := clusterPricesAdvanced(pricePoints, dynamicThreshold, config.MinClusterSize)

	// Separate into support and resistance
	resistance := make([]model.SRLevel, 0)
	support := make([]model.SRLevel, 0)

	for _, cluster := range clusters {
		level := model.SRLevel{
			Price:    cluster.Price,
			Strength: cluster.Strength,
		}

		// Add buffer zone around current price
		bufferZone := currentPrice * 0.002 // 0.2% buffer

		if cluster.Price > currentPrice+bufferZone {
			resistance = append(resistance, level)
		} else if cluster.Price < currentPrice-bufferZone {
			support = append(support, level)
		}
	}

	// Sort by price instead of strength
	// Resistance: low to high (nearest resistance first)
	sort.Slice(resistance, func(i, j int) bool {
		return resistance[i].Price < resistance[j].Price
	})
	// Support: high to low (nearest support first)
	sort.Slice(support, func(i, j int) bool {
		return support[i].Price > support[j].Price
	})

	// Ensure minimum levels - add recent highs/lows if needed
	resistance = ensureMinimumLevels(resistance, candles, currentPrice, true, config.MinLevels, config.LookbackPeriod)
	support = ensureMinimumLevels(support, candles, currentPrice, false, config.MinLevels, config.LookbackPeriod)

	// Limit to top 5
	if len(resistance) > 5 {
		resistance = resistance[:5]
	}
	if len(support) > 5 {
		support = support[:5]
	}

	return model.SRLevels{
		Resistance: resistance,
		Support:    support,
	}
}

// clusterPricesAdvanced uses volume-weighted clustering
func clusterPricesAdvanced(points []PricePoint, threshold float64, minSize int) []PriceCluster {
	if len(points) == 0 {
		return []PriceCluster{}
	}

	clusters := make([]PriceCluster, 0)
	i := 0

	for i < len(points) {
		clusterPrices := []float64{points[i].Price}
		totalVolume := points[i].Volume
		j := i + 1

		// Group nearby prices
		for j < len(points) && points[j].Price-points[i].Price <= threshold {
			clusterPrices = append(clusterPrices, points[j].Price)
			totalVolume += points[j].Volume
			j++
		}

		// Only create cluster if meets minimum size
		if len(clusterPrices) >= minSize {
			avgPrice := average(clusterPrices)
			
			// Calculate strength based on count and volume
			countStrength := float64(len(clusterPrices)) / float64(len(points))
			volumeStrength := math.Min(1.0, totalVolume/100000) // Normalize volume
			
			clusters = append(clusters, PriceCluster{
				Price:    avgPrice,
				Count:    len(clusterPrices),
				Volume:   totalVolume,
				Strength: (countStrength*0.6 + volumeStrength*0.4), // Weighted combination
			})
		}

		i = j
	}

	return clusters
}

// ensureMinimumLevels adds recent swing highs/lows if not enough levels found
func ensureMinimumLevels(levels []model.SRLevel, candles []model.Candle, currentPrice float64, isResistance bool, minLevels int, lookback int) []model.SRLevel {
	if len(levels) >= minLevels {
		return levels
	}

	// Find swing points
	swingPoints := findSwingPoints(candles, lookback, isResistance)
	
	// Filter by direction
	for _, swing := range swingPoints {
		if isResistance && swing.Price > currentPrice {
			// Check if not already in levels
			exists := false
			for _, level := range levels {
				if math.Abs(level.Price-swing.Price)/swing.Price < 0.005 {
					exists = true
					break
				}
			}
			if !exists {
				levels = append(levels, swing)
				if len(levels) >= minLevels {
					break
				}
			}
		} else if !isResistance && swing.Price < currentPrice {
			exists := false
			for _, level := range levels {
				if math.Abs(level.Price-swing.Price)/swing.Price < 0.005 {
					exists = true
					break
				}
			}
			if !exists {
				levels = append(levels, swing)
				if len(levels) >= minLevels {
					break
				}
			}
		}
	}

	return levels
}

// findSwingPoints identifies local highs and lows
func findSwingPoints(candles []model.Candle, lookback int, findHighs bool) []model.SRLevel {
	swings := make([]model.SRLevel, 0)
	
	startIdx := len(candles) - lookback
	if startIdx < 2 {
		startIdx = 2
	}

	for i := startIdx; i < len(candles)-2; i++ {
		if findHighs {
			// Check if local high
			if candles[i].High > candles[i-1].High && 
			   candles[i].High > candles[i-2].High &&
			   candles[i].High > candles[i+1].High &&
			   candles[i].High > candles[i+2].High {
				swings = append(swings, model.SRLevel{
					Price:    candles[i].High,
					Strength: 0.3, // Lower strength for swing points
				})
			}
		} else {
			// Check if local low
			if candles[i].Low < candles[i-1].Low && 
			   candles[i].Low < candles[i-2].Low &&
			   candles[i].Low < candles[i+1].Low &&
			   candles[i].Low < candles[i+2].Low {
				swings = append(swings, model.SRLevel{
					Price:    candles[i].Low,
					Strength: 0.3,
				})
			}
		}
	}

	// Sort by strength
	sort.Slice(swings, func(i, j int) bool {
		return swings[i].Strength > swings[j].Strength
	})

	return swings
}

// calculateATR calculates Average True Range
func calculateATR(candles []model.Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	trs := make([]float64, 0)
	for i := len(candles) - period; i < len(candles); i++ {
		if i == 0 {
			continue
		}
		
		tr := math.Max(
			candles[i].High-candles[i].Low,
			math.Max(
				math.Abs(candles[i].High-candles[i-1].Close),
				math.Abs(candles[i].Low-candles[i-1].Close),
			),
		)
		trs = append(trs, tr)
	}

	return average(trs)
}

// average calculates the average of a slice of floats
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateSRLevelsAdvanced is deprecated, use CalculateSRLevelsWithInterval instead
func CalculateSRLevelsAdvanced(candles []model.Candle, lookback int) model.SRLevels {
	return CalculateSRLevelsWithInterval(candles, lookback, "1d")
}
