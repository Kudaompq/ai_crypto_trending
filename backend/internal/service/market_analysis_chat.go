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
	messages = append(messages, ChatMessage{
		Role:    "system",
		Content: "你是只读的 USDⓈ-M 合约行情分析助手。只能依据本次提供的行情快照、指标、标的和周期回答；说明数据时间，不得假称获取了未提供的数据。不得执行交易或修改应用数据。\n当前行情上下文：" + string(encodedContext),
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
