package indicator

import (
	"math"
	"sort"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// CalculateSRLevels calculates support and resistance levels using clustering
func CalculateSRLevels(candles []model.Candle, lookback int) model.SRLevels {
	if len(candles) < lookback {
		lookback = len(candles)
	}

	// Collect high and low prices
	prices := make([]float64, 0)
	for i := len(candles) - lookback; i < len(candles); i++ {
		prices = append(prices, candles[i].High)
		prices = append(prices, candles[i].Low)
	}

	// Use simple clustering to find price levels
	clusters := clusterPrices(prices, 5) // Find top 5 clusters

	currentPrice := candles[len(candles)-1].Close

	resistance := make([]model.SRLevel, 0)
	support := make([]model.SRLevel, 0)

	for _, cluster := range clusters {
		level := model.SRLevel{
			Price:    cluster.Price,
			Strength: cluster.Strength,
		}

		if cluster.Price > currentPrice {
			resistance = append(resistance, level)
		} else {
			support = append(support, level)
		}
	}

	// Sort by price
	sort.Slice(resistance, func(i, j int) bool {
		return resistance[i].Price < resistance[j].Price
	})
	sort.Slice(support, func(i, j int) bool {
		return support[i].Price > support[j].Price
	})

	// Keep top 3-5 levels
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

// PriceCluster represents a cluster of prices
type PriceCluster struct {
	Price    float64
	Strength float64
	Count    int
}

// clusterPrices uses a simple density-based clustering algorithm
func clusterPrices(prices []float64, maxClusters int) []PriceCluster {
	if len(prices) == 0 {
		return []PriceCluster{}
	}

	sort.Float64s(prices)

	// Calculate price range and threshold
	minPrice := prices[0]
	maxPrice := prices[len(prices)-1]
	threshold := (maxPrice - minPrice) * 0.02 // 2% threshold

	clusters := make([]PriceCluster, 0)
	currentCluster := []float64{prices[0]}

	for i := 1; i < len(prices); i++ {
		if prices[i]-prices[i-1] <= threshold {
			currentCluster = append(currentCluster, prices[i])
		} else {
			// Finalize current cluster
			if len(currentCluster) >= 2 {
				avgPrice := average(currentCluster)
				clusters = append(clusters, PriceCluster{
					Price:    avgPrice,
					Count:    len(currentCluster),
					Strength: float64(len(currentCluster)) / float64(len(prices)),
				})
			}
			currentCluster = []float64{prices[i]}
		}
	}

	// Add last cluster
	if len(currentCluster) >= 2 {
		avgPrice := average(currentCluster)
		clusters = append(clusters, PriceCluster{
			Price:    avgPrice,
			Count:    len(currentCluster),
			Strength: float64(len(currentCluster)) / float64(len(prices)),
		})
	}

	// Sort by strength and return top clusters
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Strength > clusters[j].Strength
	})

	if len(clusters) > maxClusters {
		clusters = clusters[:maxClusters]
	}

	return clusters
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

// CalculateSRLevelsAdvanced uses volume-weighted clustering
func CalculateSRLevelsAdvanced(candles []model.Candle, lookback int) model.SRLevels {
	if len(candles) < lookback {
		lookback = len(candles)
	}

	type WeightedPrice struct {
		Price  float64
		Volume float64
	}

	weightedPrices := make([]WeightedPrice, 0)
	
	for i := len(candles) - lookback; i < len(candles); i++ {
		weightedPrices = append(weightedPrices, WeightedPrice{
			Price:  candles[i].High,
			Volume: candles[i].Volume,
		})
		weightedPrices = append(weightedPrices, WeightedPrice{
			Price:  candles[i].Low,
			Volume: candles[i].Volume,
		})
	}

	// Sort by price
	sort.Slice(weightedPrices, func(i, j int) bool {
		return weightedPrices[i].Price < weightedPrices[j].Price
	})

	// Find clusters with volume weighting
	currentPrice := candles[len(candles)-1].Close
	priceRange := candles[len(candles)-1].High - candles[len(candles)-1].Low
	threshold := priceRange * 0.5

	clusters := make([]PriceCluster, 0)
	i := 0

	for i < len(weightedPrices) {
		clusterPrices := []float64{weightedPrices[i].Price}
		totalVolume := weightedPrices[i].Volume
		j := i + 1

		for j < len(weightedPrices) && math.Abs(weightedPrices[j].Price-weightedPrices[i].Price) <= threshold {
			clusterPrices = append(clusterPrices, weightedPrices[j].Price)
			totalVolume += weightedPrices[j].Volume
			j++
		}

		if len(clusterPrices) >= 2 {
			avgPrice := average(clusterPrices)
			clusters = append(clusters, PriceCluster{
				Price:    avgPrice,
				Count:    len(clusterPrices),
				Strength: math.Min(1.0, totalVolume/10000), // Normalize
			})
		}

		i = j
	}

	// Separate into support and resistance
	resistance := make([]model.SRLevel, 0)
	support := make([]model.SRLevel, 0)

	for _, cluster := range clusters {
		level := model.SRLevel{
			Price:    cluster.Price,
			Strength: cluster.Strength,
		}

		if cluster.Price > currentPrice*1.001 { // 0.1% above current
			resistance = append(resistance, level)
		} else if cluster.Price < currentPrice*0.999 { // 0.1% below current
			support = append(support, level)
		}
	}

	// Sort and limit
	sort.Slice(resistance, func(i, j int) bool {
		return resistance[i].Strength > resistance[j].Strength
	})
	sort.Slice(support, func(i, j int) bool {
		return support[i].Strength > support[j].Strength
	})

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
