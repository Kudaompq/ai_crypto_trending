package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type fakeWatchlistPrices struct {
	quotes          []service.PriceQuote
	snapshotSymbols []string
	events          chan service.PriceEvent
	streamSymbols   []string
	unsubscribed    bool
}

func (f *fakeWatchlistPrices) Snapshot(_ context.Context, symbols []string) ([]service.PriceQuote, error) {
	f.snapshotSymbols = symbols
	return f.quotes, nil
}

func (f *fakeWatchlistPrices) Subscribe(symbols []string) (<-chan service.PriceEvent, func()) {
	f.streamSymbols = symbols
	return f.events, func() { f.unsubscribed = true }
}

func TestWatchlistPriceSnapshotValidatesAndReturnsOnlyRequestedSymbols(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prices := &fakeWatchlistPrices{quotes: []service.PriceQuote{
		{Symbol: "BTCUSDT", Price: 65000, EventTime: 100},
		{Symbol: "ETHUSDT", Price: 3000, EventTime: 100},
	}}
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true, "ETHUSDT": true, "DOGEUSDT": false}, nil
	})
	handler := NewWatchlistPriceHandler(prices, validator)
	router := gin.New()
	router.GET("/watchlist/prices", handler.GetPrices)

	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/watchlist/prices?symbols=btcusdt,ethusdt", nil)
	router.ServeHTTP(response, request)

	if response.Code != 200 || !strings.Contains(response.Body.String(), `"symbol":"BTCUSDT"`) || !strings.Contains(response.Body.String(), `"symbol":"ETHUSDT"`) {
		t.Fatalf("unexpected snapshot response %d: %s", response.Code, response.Body.String())
	}
	if strings.Join(prices.snapshotSymbols, ",") != "BTCUSDT,ETHUSDT" {
		t.Fatalf("snapshot did not receive normalized watchlist: %#v", prices.snapshotSymbols)
	}
}

func TestWatchlistPriceSnapshotSkipsUnsupportedSymbolWithoutBlockingOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prices := &fakeWatchlistPrices{quotes: []service.PriceQuote{{Symbol: "BTCUSDT", Price: 65000, EventTime: 100}}}
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.GET("/watchlist/prices", NewWatchlistPriceHandler(prices, validator).GetPrices)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/watchlist/prices?symbols=BTCUSDT,FAKEUSDT", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"symbol":"BTCUSDT"`) ||
		!strings.Contains(response.Body.String(), `"unavailable_symbols":["FAKEUSDT"]`) {
		t.Fatalf("unsupported symbol blocked valid snapshot: %d: %s", response.Code, response.Body.String())
	}
	if strings.Join(prices.snapshotSymbols, ",") != "BTCUSDT" {
		t.Fatalf("snapshot source received unsupported symbol: %#v", prices.snapshotSymbols)
	}
}

func TestWatchlistPriceStreamEmitsStatusAndScopedPrices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prices := &fakeWatchlistPrices{events: make(chan service.PriceEvent, 2)}
	prices.events <- service.PriceEvent{Type: "status", State: "live", Message: "实时"}
	prices.events <- service.PriceEvent{Type: "price", Symbol: "BTCUSDT", Price: 65000, EventTime: 200}
	close(prices.events)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.GET("/watchlist/stream", NewWatchlistPriceHandler(prices, validator).GetStream)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/watchlist/stream?symbols=BTCUSDT,FAKEUSDT", nil))

	if response.Code != 200 || !strings.Contains(response.Body.String(), "event:status") || !strings.Contains(response.Body.String(), "event:price") {
		t.Fatalf("expected status and price SSE events, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Join(prices.streamSymbols, ",") != "BTCUSDT" {
		t.Fatalf("stream did not receive requested symbols: %#v", prices.streamSymbols)
	}
	if !strings.Contains(response.Body.String(), `"unavailable_symbols":["FAKEUSDT"]`) {
		t.Fatalf("ready event did not report unsupported symbols: %s", response.Body.String())
	}
	if !prices.unsubscribed {
		t.Fatal("SSE response completion did not release the stream subscription")
	}
}
