package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

var (
	ErrChatProviderNotConfigured = errors.New("market analysis provider is not configured")
	ErrInvalidChatRequest        = errors.New("invalid market analysis chat request")
	ErrMarketContextUnavailable  = errors.New("market context is unavailable")
	ErrChatProviderFailed        = errors.New("market analysis provider failed")
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MarketAnalysisChatRequest struct {
	Symbol   string        `json:"symbol"`
	Interval string        `json:"interval"`
	Message  string        `json:"message"`
	History  []ChatMessage `json:"history"`
}

type MarketAnalysisChatResponse struct {
	Reply          string                  `json:"reply"`
	Plan           MarketAnalysisPlan      `json:"plan"`
	KeyLevels      MarketAnalysisKeyLevels `json:"key_levels"`
	ReferencePrice float64                 `json:"reference_price"`
	Symbol         string                  `json:"symbol"`
	Interval       string                  `json:"interval"`
	ContextTime    int64                   `json:"context_time"`
}

type MarketAnalysisPlan struct {
	Direction  string  `json:"direction"`
	EntryType  string  `json:"entry_type"`
	EntryPrice float64 `json:"entry_price"`
	TakeProfit float64 `json:"take_profit"`
	StopLoss   float64 `json:"stop_loss"`
	Confidence int     `json:"confidence"`
	Leverage   float64 `json:"leverage"`
	Analysis   string  `json:"analysis"`
}

type MarketAnalysisKeyLevels struct {
	POC          *float64 `json:"poc"`
	POCEstimated bool     `json:"poc_estimated"`
	Resistance   *float64 `json:"resistance"`
	Support      *float64 `json:"support"`
}

type MarketAnalysisContext struct {
	Symbol         string                  `json:"symbol"`
	Interval       string                  `json:"interval"`
	ContextTime    int64                   `json:"context_time"`
	ReferencePrice float64                 `json:"reference_price"`
	KeyLevels      MarketAnalysisKeyLevels `json:"key_levels"`
	Candles        []model.Candle          `json:"candles"`
	Analysis       *model.AnalysisResult   `json:"analysis"`
}

type MarketAnalysisContextSource interface {
	GetMarketAnalysisContext(context.Context, string, string) (MarketAnalysisContext, error)
}

type ChatCompletionProvider interface {
	Complete(context.Context, []ChatMessage) (string, error)
}

type MarketAnalysisChatService struct {
	contextSource MarketAnalysisContextSource
	provider      ChatCompletionProvider
}

func NewMarketAnalysisChatService(contextSource MarketAnalysisContextSource, provider ChatCompletionProvider) *MarketAnalysisChatService {
	return &MarketAnalysisChatService{contextSource: contextSource, provider: provider}
}

func (s *MarketAnalysisChatService) Chat(ctx context.Context, request MarketAnalysisChatRequest) (MarketAnalysisChatResponse, error) {
	if s == nil || s.provider == nil {
		return MarketAnalysisChatResponse{}, ErrChatProviderNotConfigured
	}
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Interval = strings.TrimSpace(request.Interval)
	request.Message = strings.TrimSpace(request.Message)
	if !symbolPattern.MatchString(request.Symbol) || !IsSupportedKlineInterval(request.Interval) || request.Message == "" || utf8.RuneCountInString(request.Message) > 4000 || len(request.History) > 20 {
		return MarketAnalysisChatResponse{}, ErrInvalidChatRequest
	}
	for _, message := range request.History {
		content := strings.TrimSpace(message.Content)
		if (message.Role != "user" && message.Role != "assistant") || content == "" || utf8.RuneCountInString(content) > 4000 {
			return MarketAnalysisChatResponse{}, ErrInvalidChatRequest
		}
	}
	if s.contextSource == nil {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}

	marketContext, err := s.contextSource.GetMarketAnalysisContext(ctx, request.Symbol, request.Interval)
	if err != nil || marketContext.Symbol != request.Symbol || marketContext.Interval != request.Interval || len(marketContext.Candles) == 0 || marketContext.Analysis == nil {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
	latestCandleTime := marketContext.Candles[len(marketContext.Candles)-1].Timestamp
	if marketContext.ContextTime <= 0 || marketContext.ContextTime != latestCandleTime || marketContext.Analysis.Symbol != request.Symbol || marketContext.Analysis.Interval != request.Interval || marketContext.Analysis.Timestamp != marketContext.ContextTime {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
	marketContext.ReferencePrice = marketContext.Candles[len(marketContext.Candles)-1].Close
	marketContext.KeyLevels = deriveMarketAnalysisKeyLevels(marketContext.Analysis.SRLevels, marketContext.Candles)

	encodedContext, err := json.Marshal(marketContext)
	if err != nil {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
	messages := make([]ChatMessage, 0, len(request.History)+2)
	systemPrompt := `你是只读的 USDⓈ-M 合约行情分析助手。用简体中文，只能依据本次附带的行情快照、技术指标和关键价位作答。数据时间是快照中的最新 K 线时间。不得假称获得了未提供的数据，不得执行或声称已执行交易、修改应用数据。

数据边界：当前快照包含至多 100 根所选周期 K 线及其 analysis。它不包含账户持仓、账户风险承受能力、入场历史、清算价、资金费率、未平仓量、订单簿、爆仓分布、多空账户盈亏、地址排名或其它周期数据，不得编造这些数据。` +
		`key_levels.poc 是从最高成交量 K 线的典型价估算的 POC，不是逐笔成交量分布；只能将其称作估算值。支撑和阻力只来自 key_levels 中非空的服务端分析值，不得修改或补造关键价位。

只返回一个严格 JSON 对象，不要代码围栏、Markdown、前后说明或其它键。字段必须完整且类型准确：
{"direction":"Long 或 Short","entry_type":"market 或 limit","entry_price":数字,"take_profit":数字,"stop_loss":数字,"confidence":0到100的整数,"leverage":1到5的数字,"analysis":"简体中文行情分析"}

方向只能是 Long 或 Short。根据证据较强的一侧选择方向；行情混杂时仍选较强的一侧并降低 confidence。confidence 表示当前快照下的主观信号强弱，不是胜率或概率。价格必须为正数：Long 时 take_profit > entry_price > stop_loss；Short 时 take_profit < entry_price < stop_loss。market 入场价必须等于 reference_price；limit 入场价只能基于已有 K 线或 analysis。止盈/止损应参考已有支撑阻力、ATR 与当前结构。杠杆只给 1–5 倍的通用参考，不代表针对用户账户的个性化建议。analysis 用 2–3 句简短说明指标结构和失效条件，控制在 160 个汉字以内；不得复述具体入场/止盈/止损价格、百分比或杠杆，不要输出原始 JSON 字段名。不得输出止损保证金亏损百分比（应用会自行计算）。

行情上下文 JSON：` + string(encodedContext)
	messages = append(messages, ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	for _, message := range request.History {
		messages = append(messages, ChatMessage{Role: message.Role, Content: strings.TrimSpace(message.Content)})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: request.Message})

	reply, err := s.provider.Complete(ctx, messages)
	if err != nil || strings.TrimSpace(reply) == "" {
		return MarketAnalysisChatResponse{}, fmt.Errorf("%w", ErrChatProviderFailed)
	}
	plan, err := parseMarketAnalysisPlan(reply, marketContext.ReferencePrice)
	if err != nil {
		return MarketAnalysisChatResponse{}, fmt.Errorf("%w", ErrChatProviderFailed)
	}
	canonicalReply, err := json.Marshal(plan)
	if err != nil {
		return MarketAnalysisChatResponse{}, fmt.Errorf("%w", ErrChatProviderFailed)
	}
	return MarketAnalysisChatResponse{
		Reply: string(canonicalReply), Plan: plan, KeyLevels: marketContext.KeyLevels,
		ReferencePrice: marketContext.ReferencePrice, Symbol: request.Symbol, Interval: request.Interval,
		ContextTime: marketContext.ContextTime,
	}, nil
}

func parseMarketAnalysisPlan(reply string, referencePrice float64) (MarketAnalysisPlan, error) {
	var plan MarketAnalysisPlan
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(reply)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return MarketAnalysisPlan{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return MarketAnalysisPlan{}, errors.New("model response must contain one JSON object")
	}
	if (plan.Direction != "Long" && plan.Direction != "Short") ||
		(plan.EntryType != "market" && plan.EntryType != "limit") ||
		!isPositiveFinite(plan.EntryPrice) || !isPositiveFinite(plan.TakeProfit) || !isPositiveFinite(plan.StopLoss) ||
		plan.Confidence < 0 || plan.Confidence > 100 || !isFinite(plan.Leverage) || plan.Leverage < 1 || plan.Leverage > 5 ||
		strings.TrimSpace(plan.Analysis) == "" || utf8.RuneCountInString(plan.Analysis) > 500 {
		return MarketAnalysisPlan{}, errors.New("model response fields are outside allowed values")
	}
	if plan.Direction == "Long" && !(plan.TakeProfit > plan.EntryPrice && plan.EntryPrice > plan.StopLoss) {
		return MarketAnalysisPlan{}, errors.New("long plan price order is invalid")
	}
	if plan.Direction == "Short" && !(plan.TakeProfit < plan.EntryPrice && plan.EntryPrice < plan.StopLoss) {
		return MarketAnalysisPlan{}, errors.New("short plan price order is invalid")
	}
	if plan.EntryType == "market" && (!isPositiveFinite(referencePrice) || math.Abs(plan.EntryPrice-referencePrice) > referencePrice*1e-9) {
		return MarketAnalysisPlan{}, errors.New("market entry must match the snapshot price")
	}
	plan.Analysis = strings.TrimSpace(plan.Analysis)
	return plan, nil
}

func deriveMarketAnalysisKeyLevels(levels model.SRLevels, candles []model.Candle) MarketAnalysisKeyLevels {
	keyLevels := MarketAnalysisKeyLevels{}
	if len(candles) == 0 {
		return keyLevels
	}
	referencePrice := candles[len(candles)-1].Close
	if !isPositiveFinite(referencePrice) {
		return keyLevels
	}
	var highestVolumeCandle *model.Candle
	for i := range candles {
		candle := &candles[i]
		if !isPositiveFinite(candle.Volume) || !isPositiveFinite(candle.High) || !isPositiveFinite(candle.Low) || !isPositiveFinite(candle.Close) {
			continue
		}
		if highestVolumeCandle == nil || candle.Volume >= highestVolumeCandle.Volume {
			highestVolumeCandle = candle
		}
	}
	if highestVolumeCandle != nil {
		poc := (highestVolumeCandle.High + highestVolumeCandle.Low + highestVolumeCandle.Close) / 3
		if isPositiveFinite(poc) {
			keyLevels.POC = &poc
			keyLevels.POCEstimated = true
		}
	}
	keyLevels.Support = nearestSRLevel(levels.Support, referencePrice, true)
	keyLevels.Resistance = nearestSRLevel(levels.Resistance, referencePrice, false)
	return keyLevels
}

func nearestSRLevel(levels []model.SRLevel, referencePrice float64, support bool) *float64 {
	var nearest *float64
	for _, level := range levels {
		if !isPositiveFinite(level.Price) {
			continue
		}
		if support && level.Price >= referencePrice || !support && level.Price <= referencePrice {
			continue
		}
		if nearest == nil || support && level.Price > *nearest || !support && level.Price < *nearest {
			price := level.Price
			nearest = &price
		}
	}
	return nearest
}

func isPositiveFinite(value float64) bool {
	return value > 0 && isFinite(value)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
