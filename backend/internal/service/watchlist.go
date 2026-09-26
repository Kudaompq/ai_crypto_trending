package service

import (
	"context"
	"strings"

	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type WatchlistSymbolValidator interface {
	Validate(context.Context, string) (string, *SymbolError)
}

type WatchlistService struct {
	repository repository.WatchlistRepository
	validator  WatchlistSymbolValidator
}

func NewWatchlistService(repository repository.WatchlistRepository, validator WatchlistSymbolValidator) *WatchlistService {
	return &WatchlistService{repository: repository, validator: validator}
}

func (s *WatchlistService) Get(ctx context.Context) (repository.WatchlistSnapshot, error) {
	return s.repository.Get(ctx)
}

func (s *WatchlistService) ImportLegacy(ctx context.Context, rawSymbols []string) (repository.WatchlistSnapshot, error) {
	if len(rawSymbols) > repository.MaxWatchlistSymbols {
		return repository.WatchlistSnapshot{}, repository.ErrWatchlistLimit
	}
	symbols := make([]string, 0, len(rawSymbols))
	seen := make(map[string]struct{}, len(rawSymbols))
	for _, raw := range rawSymbols {
		symbol, validationErr := s.validator.Validate(ctx, raw)
		if validationErr != nil {
			if validationErr.Status >= 500 {
				return repository.WatchlistSnapshot{}, validationErr
			}
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}
	return s.repository.ImportLegacy(ctx, symbols)
}

func (s *WatchlistService) Add(ctx context.Context, raw string) (repository.WatchlistSnapshot, error) {
	symbol, validationErr := s.validator.Validate(ctx, raw)
	if validationErr != nil {
		return repository.WatchlistSnapshot{}, validationErr
	}
	return s.repository.Add(ctx, symbol)
}

func (s *WatchlistService) Remove(ctx context.Context, raw string) (repository.WatchlistSnapshot, error) {
	symbol := strings.ToUpper(strings.TrimSpace(raw))
	if !symbolPattern.MatchString(symbol) {
		return repository.WatchlistSnapshot{}, &SymbolError{Code: "invalid_symbol", Message: "交易对格式错误，请输入例如 BTCUSDT", Status: 400}
	}
	return s.repository.Remove(ctx, symbol)
}

func (s *WatchlistService) Reorder(ctx context.Context, revision int64, rawSymbols []string) (repository.WatchlistSnapshot, error) {
	if len(rawSymbols) > repository.MaxWatchlistSymbols {
		return repository.WatchlistSnapshot{}, repository.ErrWatchlistLimit
	}
	symbols := make([]string, len(rawSymbols))
	for i, raw := range rawSymbols {
		symbols[i] = strings.ToUpper(strings.TrimSpace(raw))
		if !symbolPattern.MatchString(symbols[i]) {
			return repository.WatchlistSnapshot{}, &SymbolError{Code: "invalid_symbol", Message: "交易对格式错误，请输入例如 BTCUSDT", Status: 400}
		}
	}
	return s.repository.Reorder(ctx, revision, symbols)
}
