package service

import (
	"context"
	"errors"
	"testing"
)

func TestSymbolValidator(t *testing.T) {
	calls := 0
	v := NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		calls++
		return map[string]bool{"ETHUSDT": true, "OLDUSDT": false, "AVAXUSDT": true}, nil
	})
	for _, tc := range []struct {
		input string
		want  string
		code  string
	}{
		{" ethusdt ", "ETHUSDT", ""},
		{"avaxusdt", "AVAXUSDT", ""},
		{"ETH/USDT", "", "invalid_symbol"},
		{"UNKNOWNUSDT", "", "unsupported_symbol"},
		{"OLDUSDT", "", "unsupported_symbol"},
	} {
		got, err := v.Validate(context.Background(), tc.input)
		if got != tc.want || (err != nil && err.Code != tc.code) || (err == nil && tc.code != "") {
			t.Fatalf("Validate(%q) = %q, %v; want %q, %q", tc.input, got, err, tc.want, tc.code)
		}
	}
	if calls != 1 {
		t.Fatalf("exchange metadata fetched %d times; want 1", calls)
	}
}

func TestSymbolValidatorSourceUnavailable(t *testing.T) {
	v := NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return nil, errors.New("network unavailable")
	})
	_, err := v.Validate(context.Background(), "AVAXUSDT")
	if err == nil || err.Code != "symbol_source_unavailable" || err.Status != 503 {
		t.Fatalf("unexpected error: %v", err)
	}
}
