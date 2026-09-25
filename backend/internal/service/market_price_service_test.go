package service

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestParseMiniTickerArrayKeepsOnlyValidUSDMSymbols(t *testing.T) {
	quotes, err := ParseMiniTickerArray([]byte(`[
		{"e":"!miniTicker@arr","E":1000,"s":"BTCUSDT","o":"60000","c":"66000","st":1},
		{"e":"!miniTicker@arr","E":1001,"s":"ETHUSDT","o":"2900","c":"3000","st":2},
		{"e":"!miniTicker@arr","E":1002,"s":"BADUSDT","o":"10","c":"not-a-price","st":1},
		{"e":"!miniTicker@arr","E":1003,"s":"ZEROUSDT","o":"0","c":"1","st":1}
	]`))
	if err != nil {
		t.Fatalf("ParseMiniTickerArray returned error: %v", err)
	}
	if len(quotes) != 1 || quotes[0] != (PriceQuote{Symbol: "BTCUSDT", Price: 66000, Change24hPercent: 10, EventTime: 1000}) {
		t.Fatalf("unexpected parsed quotes: %#v", quotes)
	}
}

func TestMarketPriceServiceFiltersPerSubscriberAndAnnouncesGlobalFeedHealth(t *testing.T) {
	connected := make(chan struct{})
	service := newMarketPriceServiceWithSources(func(ctx context.Context, _ func([]byte)) error {
		close(connected)
		<-ctx.Done()
		return ctx.Err()
	}, nil)
	btcEvents, unsubscribeBTC := service.Subscribe([]string{"BTCUSDT"})
	defer unsubscribeBTC()
	ethEvents, unsubscribeETH := service.Subscribe([]string{"ETHUSDT"})
	defer unsubscribeETH()
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("upstream was not connected")
	}

	service.publishPayload([]byte(`[{"E":2000,"s":"ETHUSDT","o":"3000","c":"3010.5","st":1}]`))

	waitForLiveStatus(t, ethEvents)
	select {
	case event := <-ethEvents:
		if event.Type != "price" || event.Symbol != "ETHUSDT" || event.Price != 3010.5 || math.Abs(event.Change24hPercent-0.35) > 1e-9 {
			t.Fatalf("unexpected ETH quote: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("ETH subscriber did not receive its quote")
	}
	waitForLiveStatus(t, btcEvents)
	select {
	case event := <-btcEvents:
		if event.Type == "price" {
			t.Fatalf("BTC subscriber received unrelated quote: %#v", event)
		}
	default:
	}
}

func waitForLiveStatus(t *testing.T, events <-chan PriceEvent) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case event := <-events:
			if event.Type == "status" && event.State == "live" {
				return
			}
		case <-timer.C:
			t.Fatal("subscriber did not receive shared feed live status")
		}
	}
}

func TestMarketPriceServiceSnapshotFetchesOnceAndFiltersRequestedSymbols(t *testing.T) {
	fetches := 0
	service := newMarketPriceServiceWithSources(nil, func(context.Context) ([]PriceQuote, error) {
		fetches++
		return []PriceQuote{
			{Symbol: "BTCUSDT", Price: 65000, Change24hPercent: 1.25},
			{Symbol: "ETHUSDT", Price: 3000, Change24hPercent: -0.5},
		}, nil
	})

	quotes, err := service.Snapshot(context.Background(), []string{"ETHUSDT"})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if fetches != 1 || len(quotes) != 1 || quotes[0].Symbol != "ETHUSDT" || quotes[0].Change24hPercent != -0.5 || quotes[0].EventTime <= 0 {
		t.Fatalf("unexpected snapshot: fetches=%d quotes=%#v", fetches, quotes)
	}
}

func TestMarketPriceServiceSnapshotSurfacesSourceErrorsAndAllowsEmptySnapshot(t *testing.T) {
	service := newMarketPriceServiceWithSources(nil, func(context.Context) ([]PriceQuote, error) {
		return nil, errors.New("upstream unavailable")
	})
	if _, err := service.Snapshot(context.Background(), []string{"BTCUSDT"}); err == nil {
		t.Fatal("expected snapshot source error")
	}

	service = newMarketPriceServiceWithSources(nil, func(context.Context) ([]PriceQuote, error) {
		return []PriceQuote{}, nil
	})
	quotes, err := service.Snapshot(context.Background(), []string{"BTCUSDT"})
	if err != nil || len(quotes) != 0 {
		t.Fatalf("empty upstream response should produce an empty snapshot, got %#v, %v", quotes, err)
	}
}

func TestMarketPriceServiceSnapshotDropsInvalidNumericPrices(t *testing.T) {
	service := newMarketPriceServiceWithSources(nil, func(context.Context) ([]PriceQuote, error) {
		return []PriceQuote{
			{Symbol: "BTCUSDT", Price: math.NaN()},
			{Symbol: "ETHUSDT", Price: 0},
			{Symbol: "SOLUSDT", Price: 119.5},
		}, nil
	})
	quotes, err := service.Snapshot(context.Background(), []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"})
	if err != nil || len(quotes) != 1 || quotes[0].Symbol != "SOLUSDT" {
		t.Fatalf("snapshot retained invalid prices: %#v, %v", quotes, err)
	}
}

func TestMarketPriceServiceUnsubscribeStopsLastUpstreamConnection(t *testing.T) {
	connected := make(chan struct{})
	cancelled := make(chan struct{})
	service := newMarketPriceServiceWithSources(func(ctx context.Context, _ func([]byte)) error {
		close(connected)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}, nil)
	_, unsubscribe := service.Subscribe([]string{"BTCUSDT"})
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("upstream was not connected")
	}
	unsubscribe()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("last unsubscribe did not cancel the upstream")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if len(service.subscribers) != 0 || service.state != "connecting" {
		t.Fatalf("subscription resources/state were not reset: subscribers=%d state=%q", len(service.subscribers), service.state)
	}
}

func TestMarketPriceServiceReconnectsAndPublishesLiveOnlyAfterValidPayload(t *testing.T) {
	var attempts atomic.Int32
	service := newMarketPriceServiceWithSources(func(ctx context.Context, onPayload func([]byte)) error {
		if attempts.Add(1) == 1 {
			return errors.New("test disconnect")
		}
		onPayload([]byte(`[{"E":3000,"s":"BTCUSDT","o":"64000","c":"65001","st":1}]`))
		<-ctx.Done()
		return ctx.Err()
	}, nil)
	service.retryDelay = 5 * time.Millisecond
	events, unsubscribe := service.Subscribe([]string{"BTCUSDT"})
	defer unsubscribe()

	waitForStatus(t, events, "reconnecting")
	waitForLiveStatus(t, events)
	select {
	case event := <-events:
		if event.Type != "price" || event.Symbol != "BTCUSDT" || event.Price != 65001 || event.Change24hPercent != 1.5640625 {
			t.Fatalf("unexpected quote after reconnect: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("no quote arrived after reconnect")
	}
	if attempts.Load() < 2 {
		t.Fatalf("expected reconnect attempt, got %d connections", attempts.Load())
	}
}

func waitForStatus(t *testing.T, events <-chan PriceEvent, state string) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case event := <-events:
			if event.Type == "status" && event.State == state {
				return
			}
		case <-timer.C:
			t.Fatalf("subscriber did not receive status %q", state)
		}
	}
}

func TestConnectBinanceMiniTickerReadsLocalWebSocketAndClosesOnCancel(t *testing.T) {
	message := []byte(`[{"E":100,"s":"BTCUSDT","o":"64000","c":"65000","st":1}]`)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		_ = connection.WriteMessage(websocket.TextMessage, message)
		_, _, _ = connection.ReadMessage()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan []byte, 1)
	done := make(chan error, 1)
	go func() {
		done <- connectBinanceMiniTickerAt(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), func(payload []byte) {
			received <- payload
		})
	}()
	select {
	case got := <-received:
		if string(got) != string(message) {
			t.Fatalf("received unexpected websocket payload: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive local websocket payload")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("websocket reader did not stop after cancellation")
	}
}
