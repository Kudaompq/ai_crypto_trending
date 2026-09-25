package service

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// MarketEvent is a real-time kline update sent to browser clients.
type MarketEvent struct {
	Type      string       `json:"type"`
	Symbol    string       `json:"symbol"`
	Interval  string       `json:"interval"`
	Candle    model.Candle `json:"candle"`
	IsFinal   bool         `json:"is_final"`
	EventTime int64        `json:"event_time"`
	State     string       `json:"state,omitempty"`
	Message   string       `json:"message,omitempty"`
}

type marketStream struct {
	symbol      string
	interval    string
	mu          sync.RWMutex
	subscribers map[chan MarketEvent]struct{}
	stop        chan struct{}
	stopOnce    sync.Once
}

// MarketStreamService shares one Binance websocket connection between all
// browser subscribers interested in the same symbol and interval.
type MarketStreamService struct {
	mu      sync.Mutex
	streams map[string]*marketStream
	connect func(string, string, futures.WsKlineHandler, futures.ErrHandler) (chan struct{}, chan struct{}, error)
}

func NewMarketStreamService() *MarketStreamService {
	return &MarketStreamService{streams: make(map[string]*marketStream), connect: futures.WsKlineServe}
}

func (s *MarketStreamService) Subscribe(symbol, interval string) (<-chan MarketEvent, func()) {
	key := symbol + ":" + interval
	subscriber := make(chan MarketEvent, 16)

	s.mu.Lock()
	stream, exists := s.streams[key]
	if !exists {
		stream = &marketStream{
			symbol:      symbol,
			interval:    interval,
			subscribers: make(map[chan MarketEvent]struct{}),
			stop:        make(chan struct{}),
		}
		s.streams[key] = stream
		go s.run(stream)
	}
	stream.mu.Lock()
	stream.subscribers[subscriber] = struct{}{}
	stream.mu.Unlock()
	s.mu.Unlock()

	var unsubscribeOnce sync.Once
	unsubscribe := func() {
		unsubscribeOnce.Do(func() {
			s.mu.Lock()
			stream.mu.Lock()
			delete(stream.subscribers, subscriber)
			close(subscriber)
			empty := len(stream.subscribers) == 0
			stream.mu.Unlock()
			if empty {
				if current, ok := s.streams[key]; ok && current == stream {
					delete(s.streams, key)
					stream.stopOnce.Do(func() { close(stream.stop) })
				}
			}
			s.mu.Unlock()
		})
	}

	return subscriber, unsubscribe
}

func (s *MarketStreamService) run(stream *marketStream) {
	for {
		stream.broadcast(MarketEvent{Type: "status", Symbol: stream.symbol, Interval: stream.interval, State: "connecting", Message: "正在连接行情数据源"})
		doneC, stopC, err := s.connect(
			stream.symbol,
			stream.interval,
			func(event *futures.WsKlineEvent) {
				marketEvent, parseErr := convertKlineEvent(event)
				if parseErr == nil {
					stream.broadcast(marketEvent)
				}
			},
			func(err error) {
				log.Printf("Binance stream %s:%s disconnected: %v", stream.symbol, stream.interval, err)
				stream.broadcast(MarketEvent{Type: "status", Symbol: stream.symbol, Interval: stream.interval, State: "reconnecting", Message: "实时行情中断，正在重试"})
			},
		)

		if err != nil {
			log.Printf("Binance stream %s:%s connection failed: %v", stream.symbol, stream.interval, err)
			stream.broadcast(MarketEvent{Type: "status", Symbol: stream.symbol, Interval: stream.interval, State: "reconnecting", Message: "实时行情连接失败，正在重试"})
			select {
			case <-stream.stop:
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		select {
		case <-stream.stop:
			close(stopC)
			return
		case <-doneC:
			select {
			case <-stream.stop:
				return
			case <-time.After(time.Second):
			}
		}
	}
}

func (s *marketStream) broadcast(event MarketEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
			// A slow browser receives the next complete candle update.
		}
	}
}

func convertKlineEvent(event *futures.WsKlineEvent) (MarketEvent, error) {
	open, err := strconv.ParseFloat(event.Kline.Open, 64)
	if err != nil {
		return MarketEvent{}, fmt.Errorf("parse open: %w", err)
	}
	high, err := strconv.ParseFloat(event.Kline.High, 64)
	if err != nil {
		return MarketEvent{}, fmt.Errorf("parse high: %w", err)
	}
	low, err := strconv.ParseFloat(event.Kline.Low, 64)
	if err != nil {
		return MarketEvent{}, fmt.Errorf("parse low: %w", err)
	}
	closePrice, err := strconv.ParseFloat(event.Kline.Close, 64)
	if err != nil {
		return MarketEvent{}, fmt.Errorf("parse close: %w", err)
	}
	volume, err := strconv.ParseFloat(event.Kline.Volume, 64)
	if err != nil {
		return MarketEvent{}, fmt.Errorf("parse volume: %w", err)
	}

	return MarketEvent{
		Type:     "kline",
		Symbol:   event.Symbol,
		Interval: event.Kline.Interval,
		Candle: model.Candle{
			Timestamp: event.Kline.StartTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
		},
		IsFinal:   event.Kline.IsFinal,
		EventTime: event.Time,
	}, nil
}
