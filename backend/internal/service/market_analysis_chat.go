package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Reply       string `json:"reply"`
	Symbol      string `json:"symbol"`
	Interval    string `json:"interval"`
	ContextTime int64  `json:"context_time"`
}

type MarketAnalysisContext struct {
	Symbol      string                `json:"symbol"`
	Interval    string                `json:"interval"`
	ContextTime int64                 `json:"context_time"`
	Candles     []model.Candle        `json:"candles"`
	Analysis    *model.AnalysisResult `json:"analysis"`
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

	encodedContext, err := json.Marshal(marketContext)
	if err != nil {
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
	messages := make([]ChatMessage, 0, len(request.History)+2)
	systemPrompt := `你是只读的 USDⓈ-M 合约行情分析助手。请用简体中文回答，只能依据本次附带的行情快照、技术指标、标的和周期分析，并以快照中的最新 K 线时间作为数据时间。不得假称获得了未提供的数据，不得执行交易或修改应用数据。

数据边界：当前快照包含至多 100 根所选周期的 K 线和由这些 K 线计算出的 analysis。它不包含账户持仓、入场价、杠杆、清算价、资金费率、未平仓量、订单簿、爆仓分布、多空账户盈亏、地址排名或其它周期数据。用户询问这些内容时，明确说明当前数据无法统计，不得编造数值或把技术指标说成链上/全市场数据。

回答格式：严格按以下五行标签组织，每个标签一行；可在标签后写简洁内容，不要省略标签：
观点：偏多 / 偏空 / 震荡 / 观望（选最符合的一项，不代表概率或确定性）
结论：一句话概括当前结构
结构依据：列出支持判断的已提供指标或 K 线特征
关键价位：只引用 analysis 中已有的支撑/阻力价位；没有时写“当前分析未识别”
风险提示：指出判断失效条件或数据限制，不提供未经请求的个性化入场、杠杆、仓位或止盈止损建议

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
	return MarketAnalysisChatResponse{
		Reply: strings.TrimSpace(reply), Symbol: request.Symbol, Interval: request.Interval, ContextTime: marketContext.ContextTime,
	}, nil
}
