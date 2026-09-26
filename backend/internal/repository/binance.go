package repository

import (
	"context"
	"fmt"
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

// GetOpenInterestHistory fetches historical open-interest samples for a symbol.
func (r *BinanceRepository) GetOpenInterestHistory(symbol, period string, limit int, startTime, endTime *int64) ([]model.OpenInterestSample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := r.client.NewOpenInterestStatisticsService().Symbol(symbol).Period(period).Limit(limit)
	if startTime != nil {
		request.StartTime(*startTime)
	}
	if endTime != nil {
		request.EndTime(*endTime)
	}
	statistics, err := request.Do(ctx)
	if err != nil {
		return nil, err
	}

	samples := make([]model.OpenInterestSample, 0, len(statistics))
	for _, statistic := range statistics {
		if statistic == nil {
			return nil, fmt.Errorf("Binance returned an empty open-interest sample")
		}
		quantity, err := strconv.ParseFloat(statistic.SumOpenInterest, 64)
		if err != nil {
			return nil, fmt.Errorf("parse Binance open-interest quantity at %d: %w", statistic.Timestamp, err)
		}
		value, err := strconv.ParseFloat(statistic.SumOpenInterestValue, 64)
		if err != nil {
			return nil, fmt.Errorf("parse Binance open-interest value at %d: %w", statistic.Timestamp, err)
		}
		samples = append(samples, model.OpenInterestSample{
			Timestamp: statistic.Timestamp,
			Quantity:  quantity,
			Value:     value,
		})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Timestamp < samples[j].Timestamp })
	return samples, nil
}
