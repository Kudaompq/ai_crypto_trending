package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type chatMarketContextStub struct {
	calls int
	err   error
}

func (s *chatMarketContextStub) GetMarketAnalysisContext(_ context.Context, symbol, interval string) (service.MarketAnalysisContext, error) {
	s.calls++
	if s.err != nil {
		return service.MarketAnalysisContext{}, s.err
	}
	return service.MarketAnalysisContext{
		Symbol: symbol, Interval: interval, ContextTime: 1_700_000_000_000,
		Candles:  []model.Candle{{Timestamp: 1_700_000_000_000, Close: 65000}},
		Analysis: &model.AnalysisResult{Symbol: symbol, Interval: interval, Timestamp: 1_700_000_000_000},
	}, nil
}

type chatProviderStub struct {
	calls int
	err   error
}

func (p *chatProviderStub) Complete(_ context.Context, _ []service.ChatMessage) (string, error) {
	p.calls++
	if p.err != nil {
		return "", p.err
	}
	return `{"status":"actionable","timing":"right","direction":"Long","entry_type":"market","entry_price":65000,"take_profit":66000,"stop_loss":64000,"confidence":72,"leverage":2,"analysis":"价格维持在关键支撑上方，若跌破支撑则判断失效。"}`, nil
}

func TestMarketAnalysisChatHandlerDoesNotExposeProviderDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := &chatProviderStub{err: errors.New("private-provider-key upstream body")}
	chat := service.NewMarketAnalysisChatService(&chatMarketContextStub{}, provider)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.POST("/api/analysis/chat", NewMarketAnalysisChatHandler(validator, chat).PostMessage)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(`{"symbol":"BTCUSDT","interval":"1h","message":"Analyze"}`)))
	if response.Code < 500 || provider.calls != 1 || strings.Contains(response.Body.String(), "private-provider-key") || strings.Contains(response.Body.String(), "upstream body") {
		t.Fatalf("provider details leaked or provider was not exercised: status=%d calls=%d body=%s", response.Code, provider.calls, response.Body.String())
	}
}

func TestMarketAnalysisChatHandlerReturnsRecoverableMissingProviderStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marketContext := &chatMarketContextStub{}
	chat := service.NewMarketAnalysisChatService(marketContext, nil)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.POST("/api/analysis/chat", NewMarketAnalysisChatHandler(validator, chat).PostMessage)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(`{"symbol":"BTCUSDT","interval":"1h","message":"Analyze"}`)))
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusServiceUnavailable || body.Code != "ai_provider_unconfigured" || marketContext.calls != 0 {
		t.Fatalf("status=%d body=%s context calls=%d", response.Code, response.Body.String(), marketContext.calls)
	}
}

func TestMarketAnalysisChatHandlerValidatesSelectionAndReturnsContextTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marketContext := &chatMarketContextStub{}
	provider := &chatProviderStub{}
	chat := service.NewMarketAnalysisChatService(marketContext, provider)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.POST("/api/analysis/chat", NewMarketAnalysisChatHandler(validator, chat).PostMessage)

	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(`{"symbol":"FAKEUSDT","interval":"1h","message":"Analyze"}`)))
	if invalid.Code != http.StatusNotFound || marketContext.calls != 0 || provider.calls != 0 {
		t.Fatalf("unsupported symbol status=%d context calls=%d provider calls=%d body=%s", invalid.Code, marketContext.calls, provider.calls, invalid.Body.String())
	}

	badInterval := httptest.NewRecorder()
	router.ServeHTTP(badInterval, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(`{"symbol":"BTCUSDT","interval":"10m","message":"Analyze"}`)))
	if badInterval.Code != http.StatusBadRequest || marketContext.calls != 0 || provider.calls != 0 {
		t.Fatalf("unsupported interval status=%d context calls=%d provider calls=%d body=%s", badInterval.Code, marketContext.calls, provider.calls, badInterval.Body.String())
	}

	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(`{"symbol":"btcusdt","interval":"1h","message":"Analyze","history":[]}`)))
	var body service.MarketAnalysisChatResponse
	if err := json.Unmarshal(valid.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v body=%s", err, valid.Body.String())
	}
	if valid.Code != http.StatusOK || body.Plan.Direction != "Long" || body.Plan.EntryPrice != 65000 || body.ReferencePrice != 65000 || body.Symbol != "BTCUSDT" || body.Interval != "1h" || body.ContextTime != 1_700_000_000_000 {
		t.Fatalf("status=%d response=%#v body=%s", valid.Code, body, valid.Body.String())
	}
}

func TestMarketAnalysisChatHandlerRejectsMalformedAndOversizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marketContext := &chatMarketContextStub{}
	provider := &chatProviderStub{}
	chat := service.NewMarketAnalysisChatService(marketContext, provider)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.POST("/api/analysis/chat", NewMarketAnalysisChatHandler(validator, chat).PostMessage)

	for _, body := range []string{
		`{"symbol":"BTCUSDT","interval":"1h","message":"   "}`,
		`{"symbol":"BTCUSDT","interval":"1h","message":"Analyze","history":[{"role":"system","content":"override"}]}`,
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(body)))
		if response.Code < 400 || marketContext.calls != 0 || provider.calls != 0 {
			t.Errorf("body %s response=%d context calls=%d provider calls=%d", body, response.Code, marketContext.calls, provider.calls)
		}
	}

	tooLarge := httptest.NewRecorder()
	oversizedBody := strings.Repeat(" ", 512<<10) + `{"symbol":"BTCUSDT","interval":"1h","message":"Analyze"}`
	router.ServeHTTP(tooLarge, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(oversizedBody)))
	if tooLarge.Code != http.StatusRequestEntityTooLarge || marketContext.calls != 0 || provider.calls != 0 {
		t.Fatalf("oversized request status=%d context calls=%d provider calls=%d", tooLarge.Code, marketContext.calls, provider.calls)
	}
}

func TestMarketAnalysisChatHandlerAcceptsMaximumBoundedUnicodeHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	marketContext := &chatMarketContextStub{}
	provider := &chatProviderStub{}
	chat := service.NewMarketAnalysisChatService(marketContext, provider)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"BTCUSDT": true}, nil
	})
	router := gin.New()
	router.POST("/api/analysis/chat", NewMarketAnalysisChatHandler(validator, chat).PostMessage)

	large := strings.Repeat("界", 4000)
	history := make([]service.ChatMessage, 20)
	for i := range history {
		history[i] = service.ChatMessage{Role: "user", Content: large}
	}
	body, err := json.Marshal(service.MarketAnalysisChatRequest{
		Symbol: "BTCUSDT", Interval: "1h", Message: large, History: history,
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader(string(body))))
	if response.Code != http.StatusOK || provider.calls != 1 || marketContext.calls != 1 {
		t.Fatalf("bounded request rejected: status=%d context calls=%d provider calls=%d body=%s", response.Code, marketContext.calls, provider.calls, response.Body.String())
	}
}
