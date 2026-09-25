package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

func TestKlineHistoryCursorAndBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var requests []url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		count := 2
		if r.URL.Query().Get("limit") == "500" {
			count = 501 // model an upstream response that exceeds the project API limit
		}
		rows := make([][]any, count)
		for i := range rows {
			// Deliberately return descending timestamps to verify API ordering.
			timestamp := int64(count-i) * 60_000
			rows[i] = []any{timestamp, "1", "2", "0.5", "1.5", "10", timestamp + 59_999, "20", 1, "5", "10", "0"}
		}
		_ = json.NewEncoder(w).Encode(rows)
	}))
	defer upstream.Close()

	client := futures.NewClient("", "")
	client.BaseURL = upstream.URL
	client.HTTPClient = upstream.Client()
	validator := testSymbolValidator()
	service := service.NewKlineServiceWithRepository(repository.NewBinanceRepositoryWithClient(client))
	router := gin.New()
	router.GET("/api/kline", NewKlineHandlerWithService(service, validator).GetKline)

	cursor := int64(1_700_000_000_000)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/api/kline?symbol=ETHUSDT&interval=1h&limit=2&endTime="+strconv.FormatInt(cursor, 10), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("cursor request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := requests[0].Get("endTime"); got != strconv.FormatInt(cursor, 10) {
		t.Errorf("upstream endTime=%q, want %d", got, cursor)
	}
	var page struct {
		Data          []model.Candle `json:"data"`
		HasMoreBefore *bool          `json:"has_more_before"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 2 || page.Data[0].Timestamp >= page.Data[1].Timestamp {
		t.Errorf("page timestamps must be ascending, got %+v", page.Data)
	}
	if page.HasMoreBefore == nil || !*page.HasMoreBefore {
		t.Errorf("has_more_before=%v, want true for a full page", page.HasMoreBefore)
	}

	// Requests without a cursor retain the old latest-data query behavior.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/api/kline?symbol=ETHUSDT&interval=1h&limit=2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("default request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if requests[1].Has("endTime") {
		t.Errorf("default request unexpectedly sent endTime=%q", requests[1].Get("endTime"))
	}

	for _, badCursor := range []string{"nope", "-1", "0", "9223372036854775808"} {
		before := len(requests)
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
			"/api/kline?symbol=ETHUSDT&interval=1h&endTime="+url.QueryEscape(badCursor), nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("endTime=%q status=%d, want 400 body=%s", badCursor, recorder.Code, recorder.Body.String())
		}
		if len(requests) != before {
			t.Errorf("invalid endTime=%q reached upstream", badCursor)
		}
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/api/kline?symbol=ETHUSDT&interval=1h&limit=500", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("maximum page status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var maximum struct {
		Data []model.Candle `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &maximum); err != nil {
		t.Fatal(err)
	}
	if len(maximum.Data) > 500 {
		t.Errorf("returned %d candles, want at most 500", len(maximum.Data))
	}
	if got := requests[len(requests)-1].Get("limit"); got != "500" {
		t.Errorf("upstream limit=%q, want 500", got)
	}
}

func TestKlineRESTAndSSEAcceptNativeIntervalsAndRejectUnknown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	intervals := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}
	repo := &intervalTestRepository{}
	klineService := service.NewKlineServiceWithRepository(repo)
	stream := &intervalTestStream{}
	validator := testSymbolValidator()
	router := gin.New()
	router.GET("/api/kline", NewKlineHandlerWithService(klineService, validator).GetKline)
	router.GET("/api/stream", NewStreamHandler(stream, validator).GetMarketStream)

	for _, interval := range intervals {
		t.Run(interval, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
				"/api/kline?symbol=ETHUSDT&interval="+url.QueryEscape(interval), nil))
			if recorder.Code != http.StatusOK {
				t.Errorf("REST interval %s status=%d body=%s", interval, recorder.Code, recorder.Body.String())
			}
			if interval == "1m" {
				var page struct {
					HasMoreBefore bool `json:"has_more_before"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				if page.HasMoreBefore {
					t.Error("short REST page should report no earlier page")
				}
			}

			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
				"/api/stream?symbol=ETHUSDT&interval="+url.QueryEscape(interval), nil))
			if recorder.Code != http.StatusOK {
				t.Errorf("SSE interval %s status=%d body=%s", interval, recorder.Code, recorder.Body.String())
			}
		})
	}

	beforeRepoCalls, beforeSubscriptions := repo.calls, stream.subscriptions
	for _, endpoint := range []string{"/api/kline", "/api/stream"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, endpoint+"?symbol=ETHUSDT&interval=2w", nil))
		var body struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(recorder.Body.Bytes(), &body)
		if recorder.Code != http.StatusBadRequest || body.Code != "unsupported_interval" {
			t.Errorf("%s unsupported interval response status=%d body=%s", endpoint, recorder.Code, recorder.Body.String())
		}
	}
	if repo.calls != beforeRepoCalls {
		t.Errorf("unsupported REST period reached repository: calls %d -> %d", beforeRepoCalls, repo.calls)
	}
	if stream.subscriptions != beforeSubscriptions {
		t.Errorf("unsupported SSE period subscribed upstream: calls %d -> %d", beforeSubscriptions, stream.subscriptions)
	}
}

func testSymbolValidator() *service.SymbolValidator {
	return service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"ETHUSDT": true}, nil
	})
}

type intervalTestRepository struct{ calls int }

func (r *intervalTestRepository) GetKlines(_, _ string, _ int, _ *int64) ([]model.Candle, error) {
	r.calls++
	return []model.Candle{{Timestamp: 60_000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}}, nil
}

type intervalTestStream struct{ subscriptions int }

func (s *intervalTestStream) Subscribe(_, _ string) (<-chan service.MarketEvent, func()) {
	s.subscriptions++
	events := make(chan service.MarketEvent)
	close(events)
	return events, func() {}
}
