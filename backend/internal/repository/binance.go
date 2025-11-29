package repository

import (
	"context"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// BinanceRepository handles data fetching from Binance API
type BinanceRepository struct {
	client *binance.Client
}

// NewBinanceRepository creates a new Binance repository
func NewBinanceRepository() *BinanceRepository {
	// Initialize Binance client (no API key needed for public data)
	client := binance.NewClient("", "")
	return &BinanceRepository{
		client: client,
	}
}

// GetKlines fetches K-line data from Binance
func (r *BinanceRepository) GetKlines(symbol, interval string, limit int) ([]model.Candle, error) {
	klines, err := r.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(limit).
		Do(context.Background())
	
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

	return candles, nil
}
