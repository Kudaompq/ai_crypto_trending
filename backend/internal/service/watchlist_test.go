package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type watchlistValidatorStub struct {
	validate func(context.Context, string) (string, *SymbolError)
}

func (s watchlistValidatorStub) Validate(ctx context.Context, symbol string) (string, *SymbolError) {
	return s.validate(ctx, symbol)
}

type watchlistRepositoryStub struct {
	snapshot       repository.WatchlistSnapshot
	added          []string
	removed        []string
	imported       [][]string
	reordered      [][]string
	reorderVersion []int64
	addErr         error
	removeErr      error
	importErr      error
	reorderErr     error
}

func (s *watchlistRepositoryStub) Get(context.Context) (repository.WatchlistSnapshot, error) {
	return s.snapshot, nil
}

func (s *watchlistRepositoryStub) ImportLegacy(_ context.Context, symbols []string) (repository.WatchlistSnapshot, error) {
	s.imported = append(s.imported, append([]string(nil), symbols...))
	return s.snapshot, s.importErr
}

func (s *watchlistRepositoryStub) Add(_ context.Context, symbol string) (repository.WatchlistSnapshot, error) {
	s.added = append(s.added, symbol)
	return s.snapshot, s.addErr
}

func (s *watchlistRepositoryStub) Remove(_ context.Context, symbol string) (repository.WatchlistSnapshot, error) {
	s.removed = append(s.removed, symbol)
	return s.snapshot, s.removeErr
}

func (s *watchlistRepositoryStub) Reorder(_ context.Context, revision int64, symbols []string) (repository.WatchlistSnapshot, error) {
	s.reorderVersion = append(s.reorderVersion, revision)
	s.reordered = append(s.reordered, append([]string(nil), symbols...))
	return s.snapshot, s.reorderErr
}

func TestWatchlistServiceValidatesAndPersistsCanonicalSymbols(t *testing.T) {
	repo := &watchlistRepositoryStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"BTCUSDT", "AVAXUSDT"}, Revision: 2}}
	validator := watchlistValidatorStub{validate: func(_ context.Context, raw string) (string, *SymbolError) {
		if raw != " avaxusdt " {
			t.Fatalf("validator received %q, want trimmed original input", raw)
		}
		return "AVAXUSDT", nil
	}}
	svc := NewWatchlistService(repo, validator)
	got, err := svc.Add(context.Background(), " avaxusdt ")
	if err != nil {
		t.Fatalf("add symbol: %v", err)
	}
	if !reflect.DeepEqual(repo.added, []string{"AVAXUSDT"}) || got.Revision != 2 {
		t.Fatalf("repository add=%v, response=%+v", repo.added, got)
	}
}

func TestWatchlistServiceRejectsInvalidAddBeforePersistence(t *testing.T) {
	repo := &watchlistRepositoryStub{}
	validatorErr := &SymbolError{Code: "unsupported_symbol", Message: "not supported", Status: 404}
	validator := watchlistValidatorStub{validate: func(context.Context, string) (string, *SymbolError) {
		return "", validatorErr
	}}
	svc := NewWatchlistService(repo, validator)
	_, err := svc.Add(context.Background(), "FAKEUSDT")
	if err != validatorErr || len(repo.added) != 0 {
		t.Fatalf("add error=%v, repository writes=%v, want validation error and no write", err, repo.added)
	}
}

func TestWatchlistServiceSurfacesDuplicateAndLastSymbolErrors(t *testing.T) {
	duplicate := errors.New("duplicate")
	lastSymbol := errors.New("last symbol")
	repo := &watchlistRepositoryStub{addErr: duplicate, removeErr: lastSymbol}
	validator := watchlistValidatorStub{validate: func(_ context.Context, raw string) (string, *SymbolError) {
		return raw, nil
	}}
	svc := NewWatchlistService(repo, validator)
	if _, err := svc.Add(context.Background(), "SOLUSDT"); err != duplicate {
		t.Fatalf("add error=%v, want duplicate error", err)
	}
	if _, err := svc.Remove(context.Background(), "BTCUSDT"); err != lastSymbol {
		t.Fatalf("remove error=%v, want last-symbol error", err)
	}
}

func TestWatchlistServicePassesRevisionAndOrderToRepository(t *testing.T) {
	repo := &watchlistRepositoryStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"ETHUSDT", "BTCUSDT"}, Revision: 9}}
	validator := watchlistValidatorStub{validate: func(_ context.Context, raw string) (string, *SymbolError) {
		return raw, nil
	}}
	svc := NewWatchlistService(repo, validator)
	got, err := svc.Reorder(context.Background(), 8, []string{"ETHUSDT", "BTCUSDT"})
	if err != nil {
		t.Fatalf("reorder symbols: %v", err)
	}
	if !reflect.DeepEqual(repo.reorderVersion, []int64{8}) || !reflect.DeepEqual(repo.reordered, [][]string{{"ETHUSDT", "BTCUSDT"}}) || got.Revision != 9 {
		t.Fatalf("reorder args=%v/%v response=%+v", repo.reorderVersion, repo.reordered, got)
	}
}
