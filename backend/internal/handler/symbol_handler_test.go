package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

func TestSymbolValidationAcrossEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"ETHUSDT": true, "AVAXUSDT": true}, nil
	})
	router := gin.New()
	router.GET("/api/symbols/validate", ValidateSymbol(validator))
	router.GET("/api/kline", NewKlineHandler(validator).GetKline)
	router.GET("/api/analysis", NewAnalysisHandler(validator).GetAnalysis)
	router.GET("/api/stream", NewStreamHandler(service.NewMarketStreamService(), validator).GetMarketStream)

	for _, path := range []string{"symbols/validate", "kline", "analysis", "stream"} {
		for _, tc := range []struct {
			symbol string
			status int
			code   string
		}{
			{"ETH/USDT", 400, "invalid_symbol"},
			{"BADUSDT", 404, "unsupported_symbol"},
		} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/"+path+"?symbol="+tc.symbol, nil))
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != tc.status || body.Code != tc.code {
				t.Errorf("%s %s: status=%d code=%s", path, tc.symbol, recorder.Code, body.Code)
			}
		}
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/symbols/validate?symbol=+avaxusdt+", nil))
	if recorder.Code != 200 || !json.Valid(recorder.Body.Bytes()) {
		t.Fatalf("valid custom symbol: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestSymbolValidationSourceFailure(t *testing.T) {
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return nil, errors.New("upstream unavailable")
	})
	router := gin.New()
	router.GET("/api/symbols/validate", ValidateSymbol(validator))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/symbols/validate?symbol=AVAXUSDT", nil))
	if recorder.Code != 503 {
		t.Fatalf("source failure: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
