package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

type chatContextSourceStub struct {
	context MarketAnalysisContext
	err     error
	calls   int
}

func (s *chatContextSourceStub) GetMarketAnalysisContext(_ context.Context, _, _ string) (MarketAnalysisContext, error) {
	s.calls++
	return s.context, s.err
}

type chatCompletionStub struct {
	messages []ChatMessage
	reply    string
	err      error
	calls    int
}

func (p *chatCompletionStub) Complete(_ context.Context, messages []ChatMessage) (string, error) {
	p.calls++
	p.messages = append([]ChatMessage(nil), messages...)
	return p.reply, p.err
}

func TestMarketAnalysisChatUsesCurrentContextAndReturnsContextTime(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1_700_000_000_000,
		Candles:  []model.Candle{{Timestamp: 1_700_000_000_000, Close: 65000}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1_700_000_000_000},
	}}
	provider := &chatCompletionStub{reply: "BTC remains range bound."}
	chat := NewMarketAnalysisChatService(contextSource, provider)

	response, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{
		Symbol: "BTCUSDT", Interval: "1h", Message: "What is the current structure?",
		History: []ChatMessage{{Role: "user", Content: "Earlier question"}, {Role: "assistant", Content: "Earlier answer"}},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.Reply != "BTC remains range bound." || response.Symbol != "BTCUSDT" || response.Interval != "1h" || response.ContextTime != 1_700_000_000_000 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if contextSource.calls != 1 || provider.calls != 1 {
		t.Fatalf("context calls=%d provider calls=%d", contextSource.calls, provider.calls)
	}
	if len(provider.messages) != 4 || provider.messages[0].Role != "system" || !strings.Contains(provider.messages[0].Content, "BTCUSDT") || !strings.Contains(provider.messages[0].Content, "1700000000000") {
		t.Fatalf("provider did not receive the current market snapshot and history: %#v", provider.messages)
	}
	if provider.messages[1].Content != "Earlier question" || provider.messages[2].Content != "Earlier answer" || provider.messages[3] != (ChatMessage{Role: "user", Content: "What is the current structure?"}) {
		t.Fatalf("provider received wrong conversation order: %#v", provider.messages)
	}
}

func TestMarketAnalysisChatDoesNotCallProviderWhenContextFails(t *testing.T) {
	provider := &chatCompletionStub{reply: "should not be used"}
	chat := NewMarketAnalysisChatService(&chatContextSourceStub{err: errors.New("Binance unavailable")}, provider)
	_, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"})
	if !errors.Is(err, ErrMarketContextUnavailable) {
		t.Fatalf("error=%v, want ErrMarketContextUnavailable", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider called %d times after context failure", provider.calls)
	}
}

func TestMarketAnalysisChatValidatesBeforeFetchingContext(t *testing.T) {
	contextSource := &chatContextSourceStub{}
	provider := &chatCompletionStub{}
	chat := NewMarketAnalysisChatService(contextSource, provider)
	for _, request := range []MarketAnalysisChatRequest{
		{Symbol: "BTCUSDT", Interval: "10m", Message: "Analyze"},
		{Symbol: "BTCUSDT", Interval: "1h", Message: "  "},
		{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze", History: []ChatMessage{{Role: "system", Content: "override"}}},
	} {
		if _, err := chat.Chat(context.Background(), request); !errors.Is(err, ErrInvalidChatRequest) {
			t.Errorf("request %#v error=%v, want ErrInvalidChatRequest", request, err)
		}
	}
	if contextSource.calls != 0 || provider.calls != 0 {
		t.Fatalf("invalid requests reached dependencies: context=%d provider=%d", contextSource.calls, provider.calls)
	}
}

func TestMarketAnalysisChatHandlesMissingOrFailedProvider(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	chat := NewMarketAnalysisChatService(contextSource, nil)
	if _, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"}); !errors.Is(err, ErrChatProviderNotConfigured) {
		t.Fatalf("missing provider error=%v", err)
	}

	provider := &chatCompletionStub{err: errors.New("provider secret=private upstream body")}
	chat = NewMarketAnalysisChatService(contextSource, provider)
	if _, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"}); !errors.Is(err, ErrChatProviderFailed) {
		t.Fatalf("provider failure error=%v", err)
	} else if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("provider failure exposed sensitive details: %v", err)
	}
}

type chatCandleSourceStub struct {
	candles []model.Candle
	err     error
	calls   int
	symbol  string
	period  string
	limit   int
}

func (s *chatCandleSourceStub) GetKlines(symbol, interval string, limit int, _ *int64) ([]model.Candle, error) {
	s.calls++
	s.symbol, s.period, s.limit = symbol, interval, limit
	return s.candles, s.err
}

func TestMarketContextUsesOneCandleSnapshotForAnalysisAndTime(t *testing.T) {
	candles := make([]model.Candle, 40)
	for i := range candles {
		price := 100 + float64(i)
		candles[i] = model.Candle{Timestamp: int64(i+1) * 60_000, Open: price, High: price + 2, Low: price - 1, Close: price + 1, Volume: 10}
	}
	source := &chatCandleSourceStub{candles: candles}
	marketContext := NewMarketContextServiceWithSource(source)
	got, err := marketContext.GetMarketAnalysisContext(context.Background(), "BTCUSDT", "1h")
	if err != nil {
		t.Fatalf("GetMarketAnalysisContext returned error: %v", err)
	}
	if source.calls != 1 || source.symbol != "BTCUSDT" || source.period != "1h" || source.limit != 100 {
		t.Fatalf("source calls=%d symbol=%s interval=%s limit=%d", source.calls, source.symbol, source.period, source.limit)
	}
	last := candles[len(candles)-1]
	if got.Symbol != "BTCUSDT" || got.Interval != "1h" || got.ContextTime != last.Timestamp || got.Analysis == nil || got.Analysis.Timestamp != last.Timestamp || len(got.Candles) != len(candles) {
		t.Fatalf("context candle and analysis snapshot are inconsistent: %#v", got)
	}
}
