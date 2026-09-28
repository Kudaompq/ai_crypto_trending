package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type MarketAnalysisCandleSource interface {
	GetKlines(symbol, interval string, limit int, endTime *int64) ([]model.Candle, error)
}

type MarketAnalysisOpenInterestSource interface {
	GetOpenInterestHistory(symbol, period string, limit int, startTime, endTime *int64) ([]model.OpenInterestSample, error)
}

type MarketContextService struct {
	source             MarketAnalysisCandleSource
	openInterestSource MarketAnalysisOpenInterestSource
	analysis           *AnalysisService
}

func NewMarketContextService() *MarketContextService {
	source := repository.NewBinanceRepository()
	return NewMarketContextServiceWithSources(source, source)
}

func NewMarketContextServiceWithSource(source MarketAnalysisCandleSource) *MarketContextService {
	openInterestSource, _ := source.(MarketAnalysisOpenInterestSource)
	return NewMarketContextServiceWithSources(source, openInterestSource)
}

func NewMarketContextServiceWithSources(source MarketAnalysisCandleSource, openInterestSource MarketAnalysisOpenInterestSource) *MarketContextService {
	return &MarketContextService{source: source, openInterestSource: openInterestSource, analysis: NewAnalysisService()}
}

func (s *MarketContextService) GetMarketAnalysisContext(ctx context.Context, symbol, interval string) (MarketAnalysisContext, error) {
	if err := ctx.Err(); err != nil {
		return MarketAnalysisContext{}, fmt.Errorf("%w: %v", ErrMarketContextUnavailable, err)
	}
	if !IsSupportedKlineInterval(interval) {
		return MarketAnalysisContext{}, ErrInvalidChatRequest
	}
	if s == nil || s.source == nil || s.analysis == nil {
		return MarketAnalysisContext{}, ErrMarketContextUnavailable
	}

	candles, err := s.source.GetKlines(symbol, interval, 100, nil)
	if err != nil {
		return MarketAnalysisContext{}, fmt.Errorf("%w: load candles: %v", ErrMarketContextUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return MarketAnalysisContext{}, fmt.Errorf("%w: %v", ErrMarketContextUnavailable, err)
	}
	if len(candles) > 100 {
		candles = candles[len(candles)-100:]
	}
	if len(candles) < 20 {
		return MarketAnalysisContext{}, ErrMarketContextUnavailable
	}

	analysis, err := s.analysis.AnalyzeCandles(symbol, interval, len(candles), candles)
	if err != nil {
		return MarketAnalysisContext{}, fmt.Errorf("%w: calculate analysis: %v", ErrMarketContextUnavailable, err)
	}
	contextTime := candles[len(candles)-1].Timestamp
	if analysis.Timestamp != contextTime {
		return MarketAnalysisContext{}, errors.New("analysis timestamp does not match the candle snapshot")
	}
	return MarketAnalysisContext{
		Symbol: symbol, Interval: interval, ContextTime: contextTime,
		Candles: append([]model.Candle(nil), candles...), Analysis: analysis,
		OpenInterest: s.getOpenInterestSummary(symbol),
	}, nil
}

func (s *MarketContextService) getOpenInterestSummary(symbol string) MarketAnalysisOpenInterestSummary {
	if s.openInterestSource == nil {
		return MarketAnalysisOpenInterestSummary{Status: "unavailable", Interval: "1h", ErrorCode: "source_not_configured"}
	}
	samples, err := s.openInterestSource.GetOpenInterestHistory(symbol, "1h", 2, nil, nil)
	if err != nil {
		return MarketAnalysisOpenInterestSummary{Status: "unavailable", Interval: "1h", ErrorCode: "source_unavailable"}
	}
	if len(samples) == 0 {
		return MarketAnalysisOpenInterestSummary{Status: "unavailable", Interval: "1h", ErrorCode: "no_data"}
	}
	return summarizeOpenInterest(samples, "1h")
}
