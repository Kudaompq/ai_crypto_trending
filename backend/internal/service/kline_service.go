package service

import (
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

// KlineService handles K-line data operations
type KlineService struct {
	binanceRepo *repository.BinanceRepository
}

// NewKlineService creates a new K-line service
func NewKlineService() *KlineService {
	return &KlineService{
		binanceRepo: repository.NewBinanceRepository(),
	}
}

// GetKlineData fetches K-line data
func (s *KlineService) GetKlineData(symbol, interval string, limit int) (*model.KlineData, error) {
	candles, err := s.binanceRepo.GetKlines(symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	return &model.KlineData{
		Symbol:   symbol,
		Interval: interval,
		Data:     candles,
	}, nil
}
