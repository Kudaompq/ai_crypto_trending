package service

import (
	"errors"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func TestMarketStreamReportsFailureThenRecovers(t *testing.T) {
	s := NewMarketStreamService()
	connectCalls := 0
	s.connect = func(symbol, interval string, onKline futures.WsKlineHandler, onError futures.ErrHandler) (chan struct{}, chan struct{}, error) {
		connectCalls++
		if connectCalls == 1 {
			return nil, nil, errors.New("dial failed")
		}
		if symbol != "AVAXUSDT" || interval != "1m" {
			t.Errorf("unexpected subscription: %s %s", symbol, interval)
		}
		go onKline(&futures.WsKlineEvent{
			Symbol: symbol, Time: 123,
			Kline: futures.WsKline{Interval: interval, StartTime: 100, Open: "10", High: "11", Low: "9", Close: "10.5", Volume: "12"},
		})
		return make(chan struct{}), make(chan struct{}), nil
	}
	events, unsubscribe := s.Subscribe("AVAXUSDT", "1m")
	defer unsubscribe()
	seenFailure := false
	seenKline := false
	deadline := time.After(5 * time.Second)
	for !seenFailure || !seenKline {
		select {
		case event := <-events:
			if event.Type == "status" && event.State == "reconnecting" {
				seenFailure = true
			}
			if event.Type == "kline" && event.Symbol == "AVAXUSDT" && event.Candle.Close == 10.5 {
				seenKline = true
			}
		case <-deadline:
			t.Fatalf("did not observe failure and recovery: failure=%v kline=%v", seenFailure, seenKline)
		}
	}
}
