package repository

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// BinanceRepository handles data fetching from Binance API
type BinanceRepository struct {
	client *futures.Client
}

// NewBinanceRepository creates a new Binance repository
func NewBinanceRepository() *BinanceRepository {
	// Initialize Binance Futures client (no API key needed for public data)
	client := futures.NewClient("", "")
	return &BinanceRepository{
		client: client,
	}
}

// NewBinanceRepositoryWithClient creates a repository using the supplied client.
// It allows callers to provide a controlled HTTP transport in tests.
func NewBinanceRepositoryWithClient(client *futures.Client) *BinanceRepository {
	return &BinanceRepository{client: client}
}

// GetKlines fetches K-line data from Binance Futures
func (r *BinanceRepository) GetKlines(symbol, interval string, limit int, endTime *int64) ([]model.Candle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	klinesService := r.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(limit)
	if endTime != nil {
		klinesService.EndTime(*endTime)
	}
	klines, err := klinesService.Do(ctx)

	if err != nil {
		return nil, err
	}

	candles := make([]model.Candle, 0, len(klines))
	for _, k := range klines {
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		close, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		candles = append(candles, model.Candle{
			Timestamp: k.OpenTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].Timestamp < candles[j].Timestamp })
	if len(candles) > limit {
		candles = candles[len(candles)-limit:]
	}

	return candles, nil
}
