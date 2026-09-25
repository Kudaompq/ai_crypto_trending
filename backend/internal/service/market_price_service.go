package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/gorilla/websocket"
)

const (
	binanceFuturesMiniTickerStreamURL = "wss://fstream.binance.com/market/ws/!miniTicker@arr"
	priceStreamSilenceTimeout         = 15 * time.Second
	priceStreamRetryDelay             = 2 * time.Second
)

// PriceQuote is a current USDⓈ-M contract price.
type PriceQuote struct {
	Symbol           string  `json:"symbol"`
	Price            float64 `json:"price"`
	Change24hPercent float64 `json:"change_24h_percent"`
	EventTime        int64   `json:"event_time"`
}

// PriceEvent is either a quote update or the state of the shared upstream feed.
type PriceEvent struct {
	Type      string  `json:"type"`
	Symbol           string  `json:"symbol,omitempty"`
	Price            float64 `json:"price,omitempty"`
	Change24hPercent float64 `json:"change_24h_percent"`
	EventTime        int64   `json:"event_time,omitempty"`
	State            string  `json:"state,omitempty"`
	Message          string  `json:"message,omitempty"`
}

type tickerConnection func(context.Context, func([]byte)) error
type priceSnapshotFetcher func(context.Context) ([]PriceQuote, error)

type priceSubscriber struct {
	symbols map[string]struct{}
	events  chan PriceEvent
}

// MarketPriceService owns one all-market Binance mini-ticker connection and
// filters its updates into per-watchlist subscriber channels.
type MarketPriceService struct {
	mu          sync.Mutex
	subscribers map[chan PriceEvent]*priceSubscriber
	connect     tickerConnection
	fetch       priceSnapshotFetcher
	cancel      context.CancelFunc
	state       string
	message     string
	streamID    uint64
	stopSilence time.Duration
	retryDelay  time.Duration
}

func NewMarketPriceService() *MarketPriceService {
	client := futures.NewClient("", "")
	return newMarketPriceServiceWithSources(connectBinanceMiniTicker, func(ctx context.Context) ([]PriceQuote, error) {
		prices, err := client.NewListPriceChangeStatsService().Do(ctx)
		if err != nil {
			return nil, err
		}
		quotes := make([]PriceQuote, 0, len(prices))
		for _, item := range prices {
			price, priceErr := strconv.ParseFloat(item.LastPrice, 64)
			change, changeErr := strconv.ParseFloat(item.PriceChangePercent, 64)
			if priceErr != nil || changeErr != nil || price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) || math.IsNaN(change) || math.IsInf(change, 0) {
				continue
			}
			quotes = append(quotes, PriceQuote{Symbol: item.Symbol, Price: price, Change24hPercent: change})
		}
		return quotes, nil
	})
}

// newMarketPriceServiceWithSources makes the external market sources replaceable in tests.
func newMarketPriceServiceWithSources(connect tickerConnection, fetch priceSnapshotFetcher) *MarketPriceService {
	return &MarketPriceService{
		subscribers: make(map[chan PriceEvent]*priceSubscriber),
		connect:     connect,
		fetch:       fetch,
		state:       "connecting",
		message:     "正在连接全市场实时行情",
		stopSilence: priceStreamSilenceTimeout,
		retryDelay:  priceStreamRetryDelay,
	}
}

// Snapshot fetches one all-contract REST snapshot and returns only requested symbols.
func (s *MarketPriceService) Snapshot(ctx context.Context, symbols []string) ([]PriceQuote, error) {
	if s.fetch == nil {
		return nil, errors.New("market price snapshot source is unavailable")
	}
	requested := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		requested[symbol] = struct{}{}
	}
	eventTime := time.Now().UnixMilli()
	allQuotes, err := s.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch Binance USDⓈ-M prices: %w", err)
	}
	bySymbol := make(map[string]PriceQuote, len(allQuotes))
	for _, quote := range allQuotes {
		if _, ok := requested[quote.Symbol]; !ok || quote.Price <= 0 || math.IsNaN(quote.Price) || math.IsInf(quote.Price, 0) || math.IsNaN(quote.Change24hPercent) || math.IsInf(quote.Change24hPercent, 0) {
			continue
		}
		quote.EventTime = eventTime
		bySymbol[quote.Symbol] = quote
	}
	filtered := make([]PriceQuote, 0, len(requested))
	for _, symbol := range symbols {
		if quote, ok := bySymbol[symbol]; ok {
			filtered = append(filtered, quote)
		}
	}
	return filtered, nil
}

// Subscribe creates a filtered subscriber and starts the shared upstream on demand.
func (s *MarketPriceService) Subscribe(symbols []string) (<-chan PriceEvent, func()) {
	symbolSet := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbolSet[symbol] = struct{}{}
	}
	events := make(chan PriceEvent, 64)
	subscriber := &priceSubscriber{symbols: symbolSet, events: events}
	s.mu.Lock()
	s.subscribers[events] = subscriber
	events <- PriceEvent{Type: "status", State: s.state, Message: s.message}
	if s.cancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.streamID++
		id := s.streamID
		go s.run(ctx, id)
	}
	s.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers, events)
			close(events)
			if len(s.subscribers) == 0 && s.cancel != nil {
				s.cancel()
				s.cancel = nil
				s.state = "connecting"
				s.message = "等待实时行情连接"
			}
			s.mu.Unlock()
		})
	}
	return events, unsubscribe
}

func (s *MarketPriceService) run(ctx context.Context, id uint64) {
	for {
		if ctx.Err() != nil {
			return
		}
		s.broadcastStatus(id, "connecting", "正在连接全市场实时行情")
		connectionCtx, cancelConnection := context.WithCancel(ctx)
		activity := make(chan struct{}, 1)
		finished := make(chan error, 1)
		go func() {
			if s.connect == nil {
				finished <- errors.New("market ticker websocket source is unavailable")
				return
			}
			finished <- s.connect(connectionCtx, func(payload []byte) {
				s.publishPayload(payload)
				select {
				case activity <- struct{}{}:
				default:
				}
			})
		}()

		timer := time.NewTimer(s.stopSilence)
		connected := true
		for connected {
			select {
			case <-ctx.Done():
				connected = false
			case err := <-finished:
				if err != nil && ctx.Err() == nil {
					log.Printf("Binance mini-ticker stream disconnected: %v", err)
				}
				connected = false
			case <-activity:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(s.stopSilence)
			case <-timer.C:
				s.broadcastStatus(id, "reconnecting", "全市场行情长时间无更新，正在重连")
				connected = false
			}
		}
		cancelConnection()
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		if ctx.Err() != nil {
			return
		}
		s.broadcastStatus(id, "reconnecting", "实时行情中断，正在重试")
		timer = time.NewTimer(s.retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (s *MarketPriceService) broadcastStatus(id uint64, state, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != s.streamID || len(s.subscribers) == 0 {
		return
	}
	s.state, s.message = state, message
	s.broadcastLocked(PriceEvent{Type: "status", State: state, Message: message}, nil)
}

func (s *MarketPriceService) publishPayload(payload []byte) {
	quotes, err := ParseMiniTickerArray(payload)
	if err != nil {
		log.Printf("Ignoring malformed Binance mini-ticker payload: %v", err)
		return
	}
	if len(quotes) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state, s.message = "live", "全市场实时行情已连接"
	s.broadcastLocked(PriceEvent{Type: "status", State: s.state, Message: s.message}, nil)
	for _, quote := range quotes {
		event := PriceEvent{Type: "price", Symbol: quote.Symbol, Price: quote.Price, Change24hPercent: quote.Change24hPercent, EventTime: quote.EventTime}
		s.broadcastLocked(event, map[string]struct{}{quote.Symbol: {}})
	}
}

func (s *MarketPriceService) broadcastLocked(event PriceEvent, only map[string]struct{}) {
	for _, subscriber := range s.subscribers {
		if only != nil {
			if _, interested := subscriber.symbols[event.Symbol]; !interested {
				continue
			}
		}
		select {
		case subscriber.events <- event:
		default:
			// A slow browser skips an obsolete quote; a later update carries the latest price.
		}
	}
}

// ParseMiniTickerArray decodes Binance's !miniTicker@arr payload and keeps
// valid USDⓈ-M contracts only (st == 1 after the market-stream migration).
func ParseMiniTickerArray(payload []byte) ([]PriceQuote, error) {
	var tickers []map[string]json.RawMessage
	if err := json.Unmarshal(payload, &tickers); err != nil {
		return nil, err
	}
	quotes := make([]PriceQuote, 0, len(tickers))
	for _, ticker := range tickers {
		var eventTime int64
		var symbol, rawPrice, rawOpenPrice string
		var status int
		if json.Unmarshal(ticker["E"], &eventTime) != nil ||
			json.Unmarshal(ticker["s"], &symbol) != nil ||
			json.Unmarshal(ticker["c"], &rawPrice) != nil ||
			json.Unmarshal(ticker["o"], &rawOpenPrice) != nil ||
			json.Unmarshal(ticker["st"], &status) != nil ||
			status != 1 || !symbolPattern.MatchString(symbol) {
			continue
		}
		price, err := strconv.ParseFloat(rawPrice, 64)
		openPrice, openErr := strconv.ParseFloat(rawOpenPrice, 64)
		if err != nil || openErr != nil || price <= 0 || openPrice <= 0 || math.IsNaN(price) || math.IsInf(price, 0) || math.IsNaN(openPrice) || math.IsInf(openPrice, 0) || eventTime <= 0 {
			continue
		}
		change := (price - openPrice) / openPrice * 100
		quotes = append(quotes, PriceQuote{Symbol: symbol, Price: price, Change24hPercent: change, EventTime: eventTime})
	}
	return quotes, nil
}

func connectBinanceMiniTicker(ctx context.Context, onPayload func([]byte)) error {
	return connectBinanceMiniTickerAt(ctx, binanceFuturesMiniTickerStreamURL, onPayload)
}

func connectBinanceMiniTickerAt(ctx context.Context, streamURL string, onPayload func([]byte)) error {
	connection, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL, nil)
	if err != nil {
		return err
	}
	defer connection.Close()
	closed := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-closed:
		}
	}()
	defer close(closed)
	for {
		_, payload, err := connection.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		onPayload(payload)
	}
}
