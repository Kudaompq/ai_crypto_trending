package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

const maxMarketAnalysisToolLimit = 100

type MarketAnalysisToolKlineSource interface {
	GetKlines(symbol, interval string, limit int, endTime *int64) ([]model.Candle, error)
}

type MarketAnalysisToolOpenInterestSource interface {
	GetOpenInterestData(symbol, interval string, limit int, startTime, endTime *int64) (*model.OpenInterestData, error)
}

type marketAnalysisToolExecutor struct {
	klines       MarketAnalysisToolKlineSource
	openInterest MarketAnalysisToolOpenInterestSource
}

func NewMarketAnalysisToolExecutor() MarketAnalysisToolExecutor {
	repo := repository.NewBinanceRepository()
	return NewMarketAnalysisToolExecutorWithSources(repo, NewOpenInterestServiceWithRepository(repo))
}

func NewMarketAnalysisToolExecutorWithSources(klines MarketAnalysisToolKlineSource, openInterest MarketAnalysisToolOpenInterestSource) MarketAnalysisToolExecutor {
	return &marketAnalysisToolExecutor{klines: klines, openInterest: openInterest}
}

func (e *marketAnalysisToolExecutor) Definitions() []ChatToolDefinition {
	return []ChatToolDefinition{
		{
			Type: "function",
			Function: ChatToolFunctionDefinition{
				Name:        "get_klines",
				Description: "读取当前交易对指定 USDⓈ-M K 线周期的最新 OHLCV 数据。只能读取行情，不可交易。",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"interval":{"type":"string","enum":["1m","3m","5m","15m","30m","1h","2h","4h","6h","8h","12h","1d","3d","1w","1M"]},"limit":{"type":"integer","minimum":1,"maximum":100}},"required":["interval"],"additionalProperties":false}`),
			},
		},
		{
			Type: "function",
			Function: ChatToolFunctionDefinition{
				Name:        "get_open_interest",
				Description: "读取当前交易对指定周期的未平仓量历史数据。只能读取行情，不可交易。",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"interval":{"type":"string","enum":["5m","15m","30m","1h","2h","4h","6h","12h","1d"]},"limit":{"type":"integer","minimum":1,"maximum":100}},"required":["interval"],"additionalProperties":false}`),
			},
		},
	}
}

func (e *marketAnalysisToolExecutor) Execute(ctx context.Context, symbol string, call ChatToolCall) (MarketAnalysisToolResult, error) {
	if err := ctx.Err(); err != nil {
		return MarketAnalysisToolResult{}, err
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !symbolPattern.MatchString(symbol) || call.Type != "function" {
		return MarketAnalysisToolResult{}, errors.New("invalid market tool request")
	}
	switch call.Function.Name {
	case "get_klines":
		if e.klines == nil {
			return MarketAnalysisToolResult{}, errors.New("kline source unavailable")
		}
		var args struct {
			Interval string `json:"interval"`
			Limit    *int   `json:"limit,omitempty"`
		}
		if err := decodeMarketToolArguments(call.Function.Arguments, &args); err != nil || !IsSupportedKlineInterval(args.Interval) {
			return MarketAnalysisToolResult{}, errors.New("invalid kline query arguments")
		}
		limit, err := normalizeMarketToolLimit(args.Limit)
		if err != nil {
			return MarketAnalysisToolResult{}, err
		}
		candles, err := e.klines.GetKlines(symbol, args.Interval, limit, nil)
		if err != nil {
			return MarketAnalysisToolResult{}, errors.New("kline source unavailable")
		}
		if len(candles) > limit {
			candles = candles[len(candles)-limit:]
		}
		volume := deriveMarketAnalysisVolumeSummary(candles)
		timestamp := int64(0)
		if len(candles) > 0 {
			timestamp = candles[len(candles)-1].Timestamp
		}
		content, err := json.Marshal(struct {
			Symbol        string                      `json:"symbol"`
			Interval      string                      `json:"interval"`
			DataTime      int64                       `json:"data_time"`
			VolumeSummary MarketAnalysisVolumeSummary `json:"volume_summary"`
			Candles       []model.Candle              `json:"candles"`
		}{symbol, args.Interval, timestamp, volume, candles})
		if err != nil {
			return MarketAnalysisToolResult{}, err
		}
		return MarketAnalysisToolResult{
			Content:  string(content),
			Evidence: MarketAnalysisToolEvidence{Symbol: symbol, Tool: call.Function.Name, Interval: args.Interval, Timestamp: timestamp, Summary: "K 线数量 " + itoa(len(candles)) + "，包含成交量"},
		}, nil
	case "get_open_interest":
		if e.openInterest == nil {
			return MarketAnalysisToolResult{}, errors.New("open interest source unavailable")
		}
		var args struct {
			Interval string `json:"interval"`
			Limit    *int   `json:"limit,omitempty"`
		}
		if err := decodeMarketToolArguments(call.Function.Arguments, &args); err != nil {
			return MarketAnalysisToolResult{}, errors.New("invalid open interest query arguments")
		}
		if _, supported := openInterestPeriod(args.Interval); !supported {
			return MarketAnalysisToolResult{}, errors.New("unsupported open interest interval")
		}
		limit, err := normalizeMarketToolLimit(args.Limit)
		if err != nil {
			return MarketAnalysisToolResult{}, err
		}
		data, err := e.openInterest.GetOpenInterestData(symbol, args.Interval, limit, nil, nil)
		if err != nil {
			return MarketAnalysisToolResult{}, errors.New("open interest source unavailable")
		}
		if data == nil {
			return MarketAnalysisToolResult{}, errors.New("open interest source unavailable")
		}
		summary := summarizeOpenInterest(data.Data, args.Interval)
		content, err := json.Marshal(struct {
			Symbol  string                            `json:"symbol"`
			Summary MarketAnalysisOpenInterestSummary `json:"summary"`
			Data    []model.OpenInterestSample        `json:"data"`
		}{symbol, summary, data.Data})
		if err != nil {
			return MarketAnalysisToolResult{}, err
		}
		return MarketAnalysisToolResult{
			Content:  string(content),
			Evidence: MarketAnalysisToolEvidence{Symbol: symbol, Tool: call.Function.Name, Interval: args.Interval, Timestamp: summary.Timestamp, Summary: summarizeOpenInterestText(summary)},
		}, nil
	default:
		return MarketAnalysisToolResult{}, errors.New("unknown market analysis tool")
	}
}

func decodeMarketToolArguments(raw string, target any) error {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("tool arguments contain trailing data")
	}
	return nil
}

func normalizeMarketToolLimit(requested *int) (int, error) {
	if requested == nil {
		return 20, nil
	}
	limit := *requested
	if limit < 1 || limit > maxMarketAnalysisToolLimit {
		return 0, errors.New("market tool limit is outside allowed range")
	}
	return limit, nil
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func summarizeOpenInterest(samples []model.OpenInterestSample, interval string) MarketAnalysisOpenInterestSummary {
	if len(samples) == 0 {
		return MarketAnalysisOpenInterestSummary{Status: "unavailable", Interval: interval, ErrorCode: "no_data"}
	}
	ordered := append([]model.OpenInterestSample(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Timestamp < ordered[j].Timestamp })
	latest := ordered[len(ordered)-1]
	summary := MarketAnalysisOpenInterestSummary{
		Status: "insufficient", Interval: interval, SampleCount: len(ordered),
		LatestValue: latest.Value, LatestQuantity: latest.Quantity, Timestamp: latest.Timestamp,
	}
	if len(ordered) < 2 || ordered[len(ordered)-2].Quantity <= 0 {
		return summary
	}
	previous := ordered[len(ordered)-2]
	quantityChange := (latest.Quantity - previous.Quantity) / previous.Quantity * 100
	summary.Status = "available"
	summary.QuantityChangePercent = &quantityChange
	if previous.Value > 0 {
		valueChange := (latest.Value - previous.Value) / previous.Value * 100
		summary.ValueChangePercent = &valueChange
	}
	return summary
}

func summarizeOpenInterestText(summary MarketAnalysisOpenInterestSummary) string {
	if summary.Status != "available" || summary.QuantityChangePercent == nil {
		return "OI 数据不足或不可用"
	}
	text := "OI 持仓量变化 " + strconv.FormatFloat(*summary.QuantityChangePercent, 'f', 2, 64) + "%"
	if summary.ValueChangePercent != nil {
		text += "，名义价值变化 " + strconv.FormatFloat(*summary.ValueChangePercent, 'f', 2, 64) + "%"
	}
	return text
}
