package service

import (
	"errors"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type OpenInterestService struct {
	repo interface {
		GetOpenInterestHistory(symbol, period string, limit int, startTime, endTime *int64) ([]model.OpenInterestSample, error)
	}
	now func() time.Time
}

func NewOpenInterestServiceWithRepository(repo interface {
	GetOpenInterestHistory(symbol, period string, limit int, startTime, endTime *int64) ([]model.OpenInterestSample, error)
}) *OpenInterestService {
	return &OpenInterestService{repo: repo, now: time.Now}
}

func NewOpenInterestService() *OpenInterestService {
	return NewOpenInterestServiceWithRepository(repository.NewBinanceRepository())
}

func (s *OpenInterestService) GetOpenInterestData(symbol, interval string, limit int, startTime, endTime *int64) (*model.OpenInterestData, error) {
	data := &model.OpenInterestData{Symbol: symbol, Interval: interval, Data: []model.OpenInterestSample{}}
	period, supported := openInterestPeriod(interval)
	if !supported {
		return data, nil
	}
	if startTime != nil && endTime != nil && *startTime > *endTime {
		return nil, errors.New("startTime must not be after endTime")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	// Binance's OI endpoint accepts a rolling 30-day window. A calendar month
	// can be 31 days, which causes its startTime validation to fail.
	cutoff := s.now().Add(-30 * 24 * time.Hour).UnixMilli()
	if endTime != nil && *endTime < cutoff {
		return data, nil
	}
	requestedStart := startTime
	if requestedStart != nil && *requestedStart < cutoff {
		clamped := cutoff
		requestedStart = &clamped
	}
	if requestedStart != nil && endTime != nil && *requestedStart > *endTime {
		return data, nil
	}

	samples, err := s.repo.GetOpenInterestHistory(symbol, period, limit, requestedStart, endTime)
	if err != nil {
		return nil, err
	}
	data.Data = samples
	return data, nil
}

func openInterestPeriod(interval string) (string, bool) {
	switch interval {
	case "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d":
		return interval, true
	default:
		return "", false
	}
}
