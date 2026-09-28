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
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	Name       string         `json:"name,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
}

type ChatToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function ChatToolCallFunction `json:"function"`
}

type ChatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatToolDefinition struct {
	Type     string                     `json:"type"`
	Function ChatToolFunctionDefinition `json:"function"`
}

type ChatToolFunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ChatCompletionResult struct {
	Content   string         `json:"content"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"`
}

type MarketAnalysisChatRequest struct {
	Symbol   string        `json:"symbol"`
	Interval string        `json:"interval"`
	Message  string        `json:"message"`
	History  []ChatMessage `json:"history"`
}

type MarketAnalysisChatResponse struct {
	Reply          string                            `json:"reply"`
	Plan           MarketAnalysisPlan                `json:"plan"`
	KeyLevels      MarketAnalysisKeyLevels           `json:"key_levels"`
	ReferencePrice float64                           `json:"reference_price"`
	ATR            float64                           `json:"atr"`
	Symbol         string                            `json:"symbol"`
	Interval       string                            `json:"interval"`
	ContextTime    int64                             `json:"context_time"`
	VolumeSummary  MarketAnalysisVolumeSummary       `json:"volume_summary"`
	OpenInterest   MarketAnalysisOpenInterestSummary `json:"open_interest"`
	ToolEvidence   []MarketAnalysisToolEvidence      `json:"tool_evidence,omitempty"`
}

type MarketAnalysisPlan struct {
	Status     string  `json:"status"`
	Timing     string  `json:"timing"`
	Direction  string  `json:"direction,omitempty"`
	EntryType  string  `json:"entry_type,omitempty"`
	EntryPrice float64 `json:"entry_price,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`
	StopLoss   float64 `json:"stop_loss,omitempty"`
	Confidence int     `json:"confidence"`
	Leverage   float64 `json:"leverage,omitempty"`
	Analysis   string  `json:"analysis"`
}

type MarketAnalysisKeyLevels struct {
	Resistance *float64 `json:"resistance"`
	Support    *float64 `json:"support"`
}

type MarketAnalysisContext struct {
	Symbol         string                            `json:"symbol"`
	Interval       string                            `json:"interval"`
	ContextTime    int64                             `json:"context_time"`
	ReferencePrice float64                           `json:"reference_price"`
	KeyLevels      MarketAnalysisKeyLevels           `json:"key_levels"`
	VolumeSummary  MarketAnalysisVolumeSummary       `json:"volume_summary"`
	OpenInterest   MarketAnalysisOpenInterestSummary `json:"open_interest"`
	Candles        []model.Candle                    `json:"candles"`
	Analysis       *model.AnalysisResult             `json:"analysis"`
}

type MarketAnalysisVolumeSummary struct {
	Latest            float64  `json:"latest"`
	AveragePrior20    float64  `json:"average_prior_20"`
	RelativeToAverage *float64 `json:"relative_to_average,omitempty"`
}

type MarketAnalysisOpenInterestSummary struct {
	Status         string   `json:"status"`
	Interval       string   `json:"interval,omitempty"`
	SampleCount    int      `json:"sample_count"`
	LatestValue    float64  `json:"latest_value,omitempty"`
	LatestQuantity float64  `json:"latest_quantity,omitempty"`
	QuantityChangePercent *float64 `json:"quantity_change_percent,omitempty"`
	ValueChangePercent    *float64 `json:"value_change_percent,omitempty"`
	Timestamp      int64    `json:"timestamp,omitempty"`
	ErrorCode      string   `json:"error_code,omitempty"`
}

type MarketAnalysisToolEvidence struct {
	Symbol    string `json:"symbol,omitempty"`
	Tool      string `json:"tool"`
	Interval  string `json:"interval,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Summary   string `json:"summary"`
}

type MarketAnalysisToolResult struct {
	Content  string
	Evidence MarketAnalysisToolEvidence
}

type MarketAnalysisContextSource interface {
	GetMarketAnalysisContext(context.Context, string, string) (MarketAnalysisContext, error)
}

type ChatCompletionProvider interface {
	Complete(context.Context, []ChatMessage) (string, error)
}

type ToolCallingChatCompletionProvider interface {
	CompleteWithTools(context.Context, []ChatMessage, []ChatToolDefinition) (ChatCompletionResult, error)
}

type MarketAnalysisToolExecutor interface {
	Definitions() []ChatToolDefinition
	Execute(context.Context, string, ChatToolCall) (MarketAnalysisToolResult, error)
}

type MarketAnalysisChatService struct {
	contextSource MarketAnalysisContextSource
	provider      ChatCompletionProvider
	toolExecutor  MarketAnalysisToolExecutor
}

func NewMarketAnalysisChatService(contextSource MarketAnalysisContextSource, provider ChatCompletionProvider, toolExecutors ...MarketAnalysisToolExecutor) *MarketAnalysisChatService {
	service := &MarketAnalysisChatService{contextSource: contextSource, provider: provider}
	if len(toolExecutors) > 0 {
		service.toolExecutor = toolExecutors[0]
	}
	return service
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
	marketContext.VolumeSummary = deriveMarketAnalysisVolumeSummary(marketContext.Candles)
	if marketContext.OpenInterest.Status == "" {
		marketContext.OpenInterest = MarketAnalysisOpenInterestSummary{Status: "unavailable", ErrorCode: "not_provided"}
	}

	encodedContext, err := json.Marshal(marketContext)
	if err != nil {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
	messages := make([]ChatMessage, 0, len(request.History)+2)
	systemPrompt := `你是只读的 USDⓈ-M 合约行情分析助手。用简体中文，只能依据本次行情快照、技术指标、成交量、OI 和工具返回的数据作答。数据时间以各数据源时间戳为准。不得假称获得了未提供的数据，不得执行或声称已执行交易、修改应用数据。

初始快照包含至多 100 根所选周期 OHLCV K 线、成交量摘要、精简 OI（未平仓量）摘要、现有指标和关键价位。OI 不可用时必须说明不可用，不得把缺失数据说成看多或看空证据。需要更多确认时，可以调用只读工具查询当前交易对的受支持周期；工具结果包含实际交易对、周期和数据时间。只能分析请求中的当前币种，不得要求查询其他交易对。不得编造账户持仓、风险承受能力、清算价、资金费率、订单簿、爆仓分布、多空账户盈亏或地址排名。

在同一轮分析中比较左侧和右侧机会：左侧是确认前在关键结构附近预判反转，右侧是突破、回踩或结构确认后跟随。若有多个明确候选，只返回置信度最高的一套；若双方证据不足或相互矛盾，返回等待确认，不要为凑方案强行给方向。

只返回一个严格 JSON 对象，不要代码围栏、Markdown、前后说明或其它键。可执行结果格式：
{"status":"actionable","timing":"left 或 right","direction":"Long 或 Short","entry_type":"market 或 limit","entry_price":数字,"take_profit":数字,"stop_loss":数字,"confidence":0到100的整数,"leverage":1到5的数字,"analysis":"简体中文行情分析"}
等待结果格式：
{"status":"wait","timing":"undetermined","confidence":0到100的整数,"analysis":"简体中文说明等待确认的条件"}

confidence 表示当前证据下的主观信号强弱，不是胜率或概率。可执行价位必须为正数：Long 时 take_profit > entry_price > stop_loss；Short 时 take_profit < entry_price < stop_loss。market 入场价必须等于 reference_price；limit 入场价只能基于已有行情。止盈/止损应参考同一上下文的支撑阻力、ATR 与市场结构。支撑和阻力只来自 key_levels 中非空的服务端分析值，不得修改或补造关键价位。杠杆只给 1–5 倍的通用参考，不代表针对用户账户的个性化建议。analysis 用 2–3 句简短说明指标结构和失效/确认条件，控制在 160 个汉字以内；不得复述具体入场/止盈/止损价格、百分比或杠杆，不要输出原始 JSON 字段名。不得输出止损保证金亏损百分比（应用会自行计算）。


行情上下文 JSON：` + string(encodedContext)
	messages = append(messages, ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	for _, message := range request.History {
		messages = append(messages, ChatMessage{Role: message.Role, Content: strings.TrimSpace(message.Content)})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: request.Message})

	reply, toolEvidence, err := s.completeWithTools(ctx, request.Symbol, messages)
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
		ReferencePrice: marketContext.ReferencePrice, ATR: marketContext.Analysis.Indicators.ATR.Value,
		Symbol: request.Symbol, Interval: request.Interval, ContextTime: marketContext.ContextTime,
		VolumeSummary: marketContext.VolumeSummary, OpenInterest: marketContext.OpenInterest, ToolEvidence: toolEvidence,
	}, nil
}

const maxMarketAnalysisToolRounds = 4
const maxMarketAnalysisToolCallsPerRound = 3

func (s *MarketAnalysisChatService) completeWithTools(ctx context.Context, symbol string, messages []ChatMessage) (string, []MarketAnalysisToolEvidence, error) {
	toolProvider, supportsTools := s.provider.(ToolCallingChatCompletionProvider)
	if !supportsTools || s.toolExecutor == nil {
		markToolCallingUnavailable(messages)
		reply, err := s.provider.Complete(ctx, messages)
		return reply, []MarketAnalysisToolEvidence{{Symbol: symbol, Tool: "tool_calling", Summary: "当前 provider 不支持工具调用；未查询额外周期"}}, err
	}
	definitions := s.toolExecutor.Definitions()
	if len(definitions) == 0 {
		markToolCallingUnavailable(messages)
		reply, err := s.provider.Complete(ctx, messages)
		return reply, []MarketAnalysisToolEvidence{{Symbol: symbol, Tool: "tool_calling", Summary: "当前 provider 不支持工具调用；未查询额外周期"}}, err
	}

	evidence := make([]MarketAnalysisToolEvidence, 0)
	for round := 0; round < maxMarketAnalysisToolRounds; round++ {
		completion, err := toolProvider.CompleteWithTools(ctx, messages, definitions)
		if err != nil {
			return "", evidence, err
		}
		if len(completion.ToolCalls) == 0 {
			return completion.Content, evidence, nil
		}
		if len(completion.ToolCalls) > maxMarketAnalysisToolCallsPerRound {
			return "", evidence, ErrChatProviderFailed
		}
		if round == maxMarketAnalysisToolRounds-1 {
			return "", evidence, ErrChatProviderFailed
		}
		messages = append(messages, ChatMessage{Role: "assistant", Content: completion.Content, ToolCalls: completion.ToolCalls})
		for _, call := range completion.ToolCalls {
			result, executeErr := s.toolExecutor.Execute(ctx, symbol, call)
			if executeErr != nil || strings.TrimSpace(result.Content) == "" {
				result = MarketAnalysisToolResult{Content: `{"status":"unavailable"}`, Evidence: MarketAnalysisToolEvidence{Symbol: symbol, Tool: call.Function.Name, Interval: requestedMarketToolInterval(call.Function.Arguments), Summary: "查询不可用"}}
			}
			if result.Evidence.Symbol == "" {
				result.Evidence.Symbol = symbol
			}
			if result.Evidence.Tool == "" {
				result.Evidence.Tool = call.Function.Name
			}
			if utf8.RuneCountInString(result.Evidence.Summary) > 500 {
				runes := []rune(result.Evidence.Summary)
				result.Evidence.Summary = string(runes[:500])
			}
			evidence = append(evidence, result.Evidence)
			messages = append(messages, ChatMessage{Role: "tool", ToolCallID: call.ID, Name: call.Function.Name, Content: result.Content})
		}
	}
	return "", evidence, ErrChatProviderFailed
}

func markToolCallingUnavailable(messages []ChatMessage) {
	if len(messages) == 0 || messages[0].Role != "system" {
		return
	}
	messages[0].Content += "\n当前模型 provider 不支持工具调用，本轮只能依据初始上下文分析；不得声称已查询额外周期。"
}

func requestedMarketToolInterval(arguments string) string {
	var request struct {
		Interval string `json:"interval"`
	}
	if err := json.Unmarshal([]byte(arguments), &request); err != nil {
		return ""
	}
	return request.Interval
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
	if (plan.Status != "actionable" && plan.Status != "wait") || plan.Confidence < 0 || plan.Confidence > 100 ||
		strings.TrimSpace(plan.Analysis) == "" || utf8.RuneCountInString(plan.Analysis) > 500 {
		return MarketAnalysisPlan{}, errors.New("model response fields are outside allowed values")
	}
	if plan.Status == "wait" {
		if plan.Timing != "undetermined" || plan.Direction != "" || plan.EntryType != "" || plan.EntryPrice != 0 || plan.TakeProfit != 0 || plan.StopLoss != 0 || plan.Leverage != 0 {
			return MarketAnalysisPlan{}, errors.New("wait result must not contain executable plan fields")
		}
		plan.Analysis = strings.TrimSpace(plan.Analysis)
		return plan, nil
	}
	if (plan.Timing != "left" && plan.Timing != "right") || (plan.Direction != "Long" && plan.Direction != "Short") ||
		(plan.EntryType != "market" && plan.EntryType != "limit") ||
		!isPositiveFinite(plan.EntryPrice) || !isPositiveFinite(plan.TakeProfit) || !isPositiveFinite(plan.StopLoss) ||
		!isFinite(plan.Leverage) || plan.Leverage < 1 || plan.Leverage > 5 {
		return MarketAnalysisPlan{}, errors.New("actionable plan fields are outside allowed values")
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

func deriveMarketAnalysisVolumeSummary(candles []model.Candle) MarketAnalysisVolumeSummary {
	if len(candles) == 0 {
		return MarketAnalysisVolumeSummary{}
	}
	latest := candles[len(candles)-1].Volume
	start := len(candles) - 21
	if start < 0 {
		start = 0
	}
	end := len(candles) - 1
	if end <= start {
		return MarketAnalysisVolumeSummary{Latest: latest}
	}
	total := 0.0
	for _, candle := range candles[start:end] {
		total += candle.Volume
	}
	average := total / float64(end-start)
	summary := MarketAnalysisVolumeSummary{Latest: latest, AveragePrior20: average}
	if average > 0 {
		relative := latest / average
		summary.RelativeToAverage = &relative
	}
	return summary
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
