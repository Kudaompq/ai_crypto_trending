package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

var symbolPattern = regexp.MustCompile(`^[A-Z0-9]{5,30}$`)

type SymbolError struct {
	Code    string
	Message string
	Status  int
}

func (e *SymbolError) Error() string { return e.Message }

type SymbolValidator struct {
	mu      sync.Mutex
	fetch   func(context.Context) (map[string]bool, error)
	symbols map[string]bool
	expires time.Time
}

func NewSymbolValidator() *SymbolValidator {
	client := futures.NewClient("", "")
	return &SymbolValidator{fetch: func(ctx context.Context) (map[string]bool, error) {
		info, err := client.NewExchangeInfoService().Do(ctx)
		if err != nil {
			return nil, err
		}
		symbols := make(map[string]bool, len(info.Symbols))
		for _, item := range info.Symbols {
			symbols[item.Symbol] = item.Status == string(futures.SymbolStatusTypeTrading)
		}
		return symbols, nil
	}}
}

// NewSymbolValidatorWithFetcher allows the exchange metadata source to be replaced.
func NewSymbolValidatorWithFetcher(fetch func(context.Context) (map[string]bool, error)) *SymbolValidator {
	return &SymbolValidator{fetch: fetch}
}

func (v *SymbolValidator) Validate(ctx context.Context, raw string) (string, *SymbolError) {
	symbol := strings.ToUpper(strings.TrimSpace(raw))
	if !symbolPattern.MatchString(symbol) {
		return "", &SymbolError{Code: "invalid_symbol", Message: "交易对格式错误，请输入例如 BTCUSDT", Status: 400}
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Now().After(v.expires) {
		fetchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		symbols, err := v.fetch(fetchCtx)
		cancel()
		if err != nil || symbols == nil {
			if err == nil {
				err = errors.New("empty exchange info")
			}
			return "", &SymbolError{Code: "symbol_source_unavailable", Message: "交易对数据源暂不可用，请稍后重试", Status: 503}
		}
		v.symbols = symbols
		v.expires = time.Now().Add(5 * time.Minute)
	}
	if !v.symbols[symbol] {
		return "", &SymbolError{Code: "unsupported_symbol", Message: "该交易对暂不受 Binance 合约支持", Status: 404}
	}
	return symbol, nil
}
