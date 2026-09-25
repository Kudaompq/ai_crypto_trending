package service

import (
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

// KlineService handles K-line data operations
type KlineService struct {
	binanceRepo interface {
		GetKlines(symbol, interval string, limit int, endTime *int64) ([]model.Candle, error)
	}
}

// NewKlineServiceWithRepository creates a service using an injected repository.
func NewKlineServiceWithRepository(repo interface {
	GetKlines(symbol, interval string, limit int, endTime *int64) ([]model.Candle, error)
}) *KlineService {
	return &KlineService{binanceRepo: repo}
}

// NewKlineService creates a new K-line service
func NewKlineService() *KlineService {
	return &KlineService{
		binanceRepo: repository.NewBinanceRepository(),
	}
}

// GetKlineData fetches K-line data
func (s *KlineService) GetKlineData(symbol, interval string, limit int, endTime *int64) (*model.KlineData, error) {
	candles, err := s.binanceRepo.GetKlines(symbol, interval, limit, endTime)
	if err != nil {
		return nil, err
	}

	return &model.KlineData{
		Symbol:        symbol,
		Interval:      interval,
		Data:          candles,
		HasMoreBefore: len(candles) == limit && len(candles) > 0,
	}, nil
}
