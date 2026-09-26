package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type openInterestServiceStub struct {
	calls            int
	symbol, interval string
	limit            int
	start, end       *int64
	data             *model.OpenInterestData
	err              error
}

func (s *openInterestServiceStub) GetOpenInterestData(symbol, interval string, limit int, startTime, endTime *int64) (*model.OpenInterestData, error) {
	s.calls++
	s.symbol, s.interval, s.limit, s.start, s.end = symbol, interval, limit, startTime, endTime
	return s.data, s.err
}

func TestOpenInterestHandlerValidatesAndForwardsQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openInterest := &openInterestServiceStub{data: &model.OpenInterestData{Symbol: "ETHUSDT", Interval: "5m", Data: []model.OpenInterestSample{}}}
	router := gin.New()
	router.GET("/api/open-interest", NewOpenInterestHandlerWithService(openInterest, testSymbolValidator()).GetOpenInterest)
	start, end := int64(1_700_000_000_000), int64(1_700_001_000_000)
	response := httptest.NewRecorder()
	path := "/api/open-interest?symbol=ethusdt&interval=5m&limit=20&startTime=" + strconv.FormatInt(start, 10) + "&endTime=" + strconv.FormatInt(end, 10)
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK || openInterest.symbol != "ETHUSDT" || openInterest.interval != "5m" || openInterest.limit != 20 {
		t.Fatalf("response=%d %s; forwarded %s/%s limit=%d", response.Code, response.Body.String(), openInterest.symbol, openInterest.interval, openInterest.limit)
	}
	if openInterest.start == nil || *openInterest.start != start || openInterest.end == nil || *openInterest.end != end {
		t.Fatalf("forwarded range start=%v end=%v", openInterest.start, openInterest.end)
	}
}

func TestOpenInterestHandlerRejectsInvalidRangesAndSymbolsWithoutServiceCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openInterest := &openInterestServiceStub{data: &model.OpenInterestData{Data: []model.OpenInterestSample{}}}
	router := gin.New()
	router.GET("/api/open-interest", NewOpenInterestHandlerWithService(openInterest, testSymbolValidator()).GetOpenInterest)
	for _, query := range []string{
		"symbol=ETHUSDT&startTime=nope",
		"symbol=ETHUSDT&startTime=10&endTime=9",
		"symbol=ETHUSDT&limit=501",
		"symbol=FAKEUSDT",
	} {
		before := openInterest.calls
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/open-interest?"+query, nil))
		if response.Code < 400 || openInterest.calls != before {
			t.Errorf("query %q response=%d %s service calls=%d", query, response.Code, response.Body.String(), openInterest.calls)
		}
	}
}

func TestOpenInterestHandlerReturnsRecoverableUpstreamFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openInterest := &openInterestServiceStub{err: errors.New("upstream unavailable")}
	router := gin.New()
	router.GET("/api/open-interest", NewOpenInterestHandlerWithService(openInterest, testSymbolValidator()).GetOpenInterest)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/open-interest?symbol=ETHUSDT&interval=5m", nil))
	var body struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	if response.Code != http.StatusServiceUnavailable || body.Code != "open_interest_source_unavailable" {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

func TestOpenInterestHandlerUnsupportedPeriodCanReturnEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openInterest := &openInterestServiceStub{data: &model.OpenInterestData{Symbol: "ETHUSDT", Interval: "1m", Data: []model.OpenInterestSample{}}}
	router := gin.New()
	router.GET("/api/open-interest", NewOpenInterestHandlerWithService(openInterest, testSymbolValidator()).GetOpenInterest)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/open-interest?symbol=ETHUSDT&interval=1m", nil))
	var body model.OpenInterestData
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusOK || len(body.Data) != 0 {
		t.Fatalf("unsupported interval should be empty: status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
}

func TestOpenInterestHandlerSymbolSourceFailureIsNotReportedAsUnsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openInterest := &openInterestServiceStub{data: &model.OpenInterestData{Data: []model.OpenInterestSample{}}}
	validator := service.NewSymbolValidatorWithFetcher(func(context.Context) (map[string]bool, error) {
		return nil, errors.New("exchange info unavailable")
	})
	router := gin.New()
	router.GET("/api/open-interest", NewOpenInterestHandlerWithService(openInterest, validator).GetOpenInterest)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/open-interest?symbol=ETHUSDT", nil))
	if response.Code != http.StatusServiceUnavailable || openInterest.calls != 0 {
		t.Fatalf("symbol source failure response=%d calls=%d body=%s", response.Code, openInterest.calls, response.Body.String())
	}
}
