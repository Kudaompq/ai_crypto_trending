package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

type marketToolKlineSourceStub struct {
	candles []model.Candle
	err     error
	symbol  string
	period  string
	limit   int
	calls   int
}

func (s *marketToolKlineSourceStub) GetKlines(symbol, period string, limit int, _ *int64) ([]model.Candle, error) {
	s.calls++
	s.symbol, s.period, s.limit = symbol, period, limit
	return s.candles, s.err
}

type marketToolOpenInterestSourceStub struct {
	data   *model.OpenInterestData
	err    error
	symbol string
	period string
	limit  int
	calls  int
}

func (s *marketToolOpenInterestSourceStub) GetOpenInterestData(symbol, period string, limit int, _, _ *int64) (*model.OpenInterestData, error) {
	s.calls++
	s.symbol, s.period, s.limit = symbol, period, limit
	return s.data, s.err
}

func TestMarketAnalysisToolExecutorQueriesRequestSymbolAndBoundsKlines(t *testing.T) {
	source := &marketToolKlineSourceStub{candles: []model.Candle{
		{Timestamp: 1, Close: 100, Volume: 10},
		{Timestamp: 2, Close: 101, Volume: 30},
	}}
	executor := NewMarketAnalysisToolExecutorWithSources(source, nil)
	result, err := executor.Execute(context.Background(), "btcusdt", ChatToolCall{
		ID: "call-kline", Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: `{"interval":"15m","limit":100}`},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if source.symbol != "BTCUSDT" || source.period != "15m" || source.limit != maxMarketAnalysisToolLimit || source.calls != 1 {
		t.Fatalf("query symbol=%q period=%q limit=%d calls=%d", source.symbol, source.period, source.limit, source.calls)
	}
	var payload struct {
		Symbol        string                      `json:"symbol"`
		Interval      string                      `json:"interval"`
		DataTime      int64                       `json:"data_time"`
		VolumeSummary MarketAnalysisVolumeSummary `json:"volume_summary"`
		Candles       []model.Candle              `json:"candles"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if payload.Symbol != "BTCUSDT" || payload.Interval != "15m" || payload.DataTime != 2 || len(payload.Candles) != 2 || payload.VolumeSummary.Latest != 30 {
		t.Fatalf("unexpected tool payload: %+v", payload)
	}
}

func TestMarketAnalysisToolExecutorRejectsModelSymbolAndUnsupportedArguments(t *testing.T) {
	source := &marketToolKlineSourceStub{candles: []model.Candle{{Timestamp: 1}}}
	executor := NewMarketAnalysisToolExecutorWithSources(source, nil)
	for _, args := range []string{
		`{"interval":"1h","symbol":"ETHUSDT"}`,
		`{"interval":"2w"}`,
		`{"interval":"1h","limit":0}`,
		`{"interval":"1h","limit":101}`,
		`{"interval":"1h","unexpected":true}`,
	} {
		_, err := executor.Execute(context.Background(), "BTCUSDT", ChatToolCall{Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: args}})
		if err == nil {
			t.Errorf("arguments %s unexpectedly succeeded", args)
		}
	}
	if source.calls != 0 {
		t.Fatalf("invalid tool arguments reached upstream %d times", source.calls)
	}
	for _, definition := range executor.Definitions() {
		if strings.Contains(string(definition.Function.Parameters), `"symbol"`) {
			t.Errorf("tool %q allows the model to pick a symbol", definition.Function.Name)
		}
	}
}

func TestMarketAnalysisToolExecutorSummarizesOIForAllowedPeriod(t *testing.T) {
	source := &marketToolOpenInterestSourceStub{data: &model.OpenInterestData{Data: []model.OpenInterestSample{
		{Timestamp: 2, Quantity: 12, Value: 1500},
		{Timestamp: 1, Quantity: 10, Value: 1000},
	}}}
	executor := NewMarketAnalysisToolExecutorWithSources(nil, source)
	result, err := executor.Execute(context.Background(), "BTCUSDT", ChatToolCall{
		Type: "function", Function: ChatToolCallFunction{Name: "get_open_interest", Arguments: `{"interval":"4h","limit":2}`},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	var payload struct {
		Symbol  string                            `json:"symbol"`
		Summary MarketAnalysisOpenInterestSummary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if source.symbol != "BTCUSDT" || source.period != "4h" || source.limit != 2 || payload.Summary.Status != "available" || payload.Summary.Timestamp != 2 || payload.Summary.QuantityChangePercent == nil || *payload.Summary.QuantityChangePercent != 20 || payload.Summary.ValueChangePercent == nil || *payload.Summary.ValueChangePercent != 50 {
		t.Fatalf("query=%+v payload=%+v", source, payload)
	}
}

func TestMarketAnalysisToolExecutorHonorsCancellationAndReportsSourceFailure(t *testing.T) {
	source := &marketToolKlineSourceStub{err: errors.New("private upstream details")}
	executor := NewMarketAnalysisToolExecutorWithSources(source, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := executor.Execute(ctx, "BTCUSDT", ChatToolCall{Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: `{"interval":"1h"}`}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled execution error=%v", err)
	}
	if source.calls != 0 {
		t.Fatalf("canceled call reached source %d times", source.calls)
	}
	if _, err := executor.Execute(context.Background(), "BTCUSDT", ChatToolCall{Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: `{"interval":"1h"}`}}); err == nil || strings.Contains(err.Error(), "private upstream details") {
		t.Fatalf("source error=%v; want sanitized non-nil error", err)
	}
}
