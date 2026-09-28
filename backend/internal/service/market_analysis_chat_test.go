package service

import (
	"context"
	"encoding/json"
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

type toolCallingChatCompletionStub struct {
	results  []ChatCompletionResult
	calls    int
	messages [][]ChatMessage
	tools    []ChatToolDefinition
}

func (p *toolCallingChatCompletionStub) Complete(context.Context, []ChatMessage) (string, error) {
	return "", errors.New("plain completion should not be used")
}

func (p *toolCallingChatCompletionStub) CompleteWithTools(_ context.Context, messages []ChatMessage, tools []ChatToolDefinition) (ChatCompletionResult, error) {
	p.calls++
	p.messages = append(p.messages, append([]ChatMessage(nil), messages...))
	p.tools = append([]ChatToolDefinition(nil), tools...)
	if p.calls > len(p.results) {
		return ChatCompletionResult{}, errors.New("unexpected extra completion")
	}
	return p.results[p.calls-1], nil
}

type chatMarketToolExecutorStub struct {
	definitions []ChatToolDefinition
	result      string
	err         error
	calls       int
	symbol      string
	toolCall    ChatToolCall
}

func (s *chatMarketToolExecutorStub) Definitions() []ChatToolDefinition { return s.definitions }

func (s *chatMarketToolExecutorStub) Execute(_ context.Context, symbol string, call ChatToolCall) (MarketAnalysisToolResult, error) {
	s.calls++
	s.symbol, s.toolCall = symbol, call
	return MarketAnalysisToolResult{Content: s.result, Evidence: MarketAnalysisToolEvidence{Interval: "4h", Timestamp: 2, Summary: "OI 上升 3.2%"}}, s.err
}

func (p *chatCompletionStub) Complete(_ context.Context, messages []ChatMessage) (string, error) {
	p.calls++
	p.messages = append([]ChatMessage(nil), messages...)
	return p.reply, p.err
}

func TestMarketAnalysisChatUsesToolEvidenceAndKeepsSymbolBoundToRequest(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Open: 100, High: 102, Low: 99, Close: 101, Volume: 120}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	toolCall := ChatToolCall{ID: "call-oi-1", Type: "function", Function: ChatToolCallFunction{Name: "get_open_interest", Arguments: `{"interval":"4h","limit":2,"symbol":"ETHUSDT"}`}}
	provider := &toolCallingChatCompletionStub{results: []ChatCompletionResult{
		{ToolCalls: []ChatToolCall{toolCall}},
		{Content: `{"status":"actionable","timing":"right","direction":"Long","entry_type":"market","entry_price":101,"take_profit":110,"stop_loss":95,"confidence":78,"leverage":2,"analysis":"成交量放大且 OI 同步增加，结构突破得到确认。"}`},
	}}
	executor := &chatMarketToolExecutorStub{
		definitions: []ChatToolDefinition{{Type: "function", Function: ChatToolFunctionDefinition{Name: "get_open_interest", Parameters: json.RawMessage(`{"type":"object"}`)}}},
		result:      `{"symbol":"BTCUSDT","interval":"4h","timestamp":2,"change_percent":3.2}`,
	}
	chat := NewMarketAnalysisChatService(contextSource, provider, executor)
	response, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze both entry styles"})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.Plan.Direction != "Long" || response.Plan.Timing != "right" || response.Plan.Confidence != 78 {
		t.Fatalf("unexpected selected plan: %#v", response.Plan)
	}
	if len(response.ToolEvidence) != 1 || response.ToolEvidence[0].Symbol != "BTCUSDT" || response.ToolEvidence[0].Interval != "4h" || response.ToolEvidence[0].Timestamp != 2 {
		t.Fatalf("final result did not retain tool evidence source: %+v", response.ToolEvidence)
	}
	if provider.calls != 2 || executor.calls != 1 || executor.symbol != "BTCUSDT" || executor.toolCall.Function.Name != "get_open_interest" {
		t.Fatalf("provider calls=%d executor calls=%d symbol=%q tool=%#v", provider.calls, executor.calls, executor.symbol, executor.toolCall)
	}
	if len(provider.messages[1]) < 3 || provider.messages[1][len(provider.messages[1])-2].Role != "assistant" || provider.messages[1][len(provider.messages[1])-1].Role != "tool" || provider.messages[1][len(provider.messages[1])-1].ToolCallID != "call-oi-1" || !strings.Contains(provider.messages[1][len(provider.messages[1])-1].Content, `"symbol":"BTCUSDT"`) {
		t.Fatalf("tool evidence was not returned to the model: %#v", provider.messages[1])
	}
	if strings.Contains(provider.messages[1][len(provider.messages[1])-1].Content, `"symbol":"ETHUSDT"`) {
		t.Fatal("model-selected symbol escaped the request-bound symbol")
	}
}

func TestMarketAnalysisChatSanitizesToolFailureAndContinuesWithWaitResult(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Close: 100}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	provider := &toolCallingChatCompletionStub{results: []ChatCompletionResult{
		{ToolCalls: []ChatToolCall{{ID: "call-1", Type: "function", Function: ChatToolCallFunction{Name: "get_open_interest", Arguments: `{"interval":"1h"}`}}}},
		{Content: `{"status":"wait","timing":"undetermined","confidence":30,"analysis":"OI 暂不可用，等待价格结构确认。"}`},
	}}
	executor := &chatMarketToolExecutorStub{
		definitions: []ChatToolDefinition{{Type: "function", Function: ChatToolFunctionDefinition{Name: "get_open_interest", Parameters: json.RawMessage(`{"type":"object"}`)}}},
		err:         errors.New("private upstream credential failure"),
	}
	response, err := NewMarketAnalysisChatService(contextSource, provider, executor).Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"})
	if err != nil {
		t.Fatalf("Chat should continue after a tool failure: %v", err)
	}
	if response.Plan.Status != "wait" || len(response.ToolEvidence) != 1 || response.ToolEvidence[0].Summary != "查询不可用" {
		t.Fatalf("unexpected wait response or evidence: response=%+v evidence=%+v", response, response.ToolEvidence)
	}
	toolMessage := provider.messages[1][len(provider.messages[1])-1]
	if strings.Contains(toolMessage.Content, "private upstream") || !strings.Contains(toolMessage.Content, `"status":"unavailable"`) {
		t.Fatalf("tool failure leaked details or was not represented as unavailable: %s", toolMessage.Content)
	}
}

func TestMarketAnalysisChatStopsAtToolRoundLimit(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Close: 100}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	results := make([]ChatCompletionResult, maxMarketAnalysisToolRounds)
	for i := range results {
		results[i].ToolCalls = []ChatToolCall{{ID: "call", Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: `{"interval":"1h"}`}}}
	}
	provider := &toolCallingChatCompletionStub{results: results}
	executor := &chatMarketToolExecutorStub{
		definitions: []ChatToolDefinition{{Type: "function", Function: ChatToolFunctionDefinition{Name: "get_klines", Parameters: json.RawMessage(`{"type":"object"}`)}}},
		result:      `{"ok":true}`,
	}
	_, err := NewMarketAnalysisChatService(contextSource, provider, executor).Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"})
	if !errors.Is(err, ErrChatProviderFailed) || provider.calls != maxMarketAnalysisToolRounds || executor.calls != maxMarketAnalysisToolRounds-1 {
		t.Fatalf("error=%v provider rounds=%d executed tools=%d", err, provider.calls, executor.calls)
	}
}

func TestMarketAnalysisChatCapsParallelToolCallsPerRound(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Close: 100}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	calls := make([]ChatToolCall, maxMarketAnalysisToolCallsPerRound+1)
	for i := range calls {
		calls[i] = ChatToolCall{ID: "call", Type: "function", Function: ChatToolCallFunction{Name: "get_klines", Arguments: `{"interval":"1h"}`}}
	}
	provider := &toolCallingChatCompletionStub{results: []ChatCompletionResult{{ToolCalls: calls}}}
	executor := &chatMarketToolExecutorStub{definitions: []ChatToolDefinition{{Type: "function", Function: ChatToolFunctionDefinition{Name: "get_klines", Parameters: json.RawMessage(`{"type":"object"}`)}}}}
	_, err := NewMarketAnalysisChatService(contextSource, provider, executor).Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"})
	if !errors.Is(err, ErrChatProviderFailed) || executor.calls != 0 {
		t.Fatalf("error=%v executed tool calls=%d", err, executor.calls)
	}
}

func TestMarketAnalysisPromptIncludesVolumeOIAndLeftRightDecisionRules(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Open: 100, High: 102, Low: 99, Close: 101, Volume: 120}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	provider := &chatCompletionStub{reply: `{"status":"wait","timing":"undetermined","confidence":40,"leverage":1,"analysis":"等待价格结构和 OI 进一步确认。"}`}
	chat := NewMarketAnalysisChatService(contextSource, provider)
	_, _ = chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"})
	if provider.calls == 0 {
		t.Fatal("provider was not called")
	}
	for _, required := range []string{"成交量摘要", "OI（未平仓量）摘要", "左侧", "右侧", "置信度最高", "等待确认"} {
		if !strings.Contains(provider.messages[0].Content, required) {
			t.Errorf("system prompt missing %q", required)
		}
	}
}

func TestMarketAnalysisChatUsesCurrentContextAndReturnsContextTime(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1_700_000_000_000,
		Candles: []model.Candle{
			{Timestamp: 1_699_999_940_000, Open: 10, High: 12, Low: 9, Close: 11, Volume: 2},
			{Timestamp: 1_700_000_000_000, Open: 11, High: 14, Low: 10, Close: 12, Volume: 10},
		},
		Analysis: &model.AnalysisResult{
			Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1_700_000_000_000,
			SRLevels: model.SRLevels{
				Support:    []model.SRLevel{{Price: 10}, {Price: 11}},
				Resistance: []model.SRLevel{{Price: 13}, {Price: 14}},
			},
		},
	}}
	provider := &chatCompletionStub{reply: `{"status":"actionable","timing":"right","direction":"Long","entry_type":"market","entry_price":12,"take_profit":13,"stop_loss":11,"confidence":72,"leverage":3,"analysis":"价格保持在结构支撑上方。"}`}
	chat := NewMarketAnalysisChatService(contextSource, provider)

	response, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{
		Symbol: "BTCUSDT", Interval: "1h", Message: "What is the current structure?",
		History: []ChatMessage{{Role: "user", Content: "Earlier question"}, {Role: "assistant", Content: "Earlier answer"}},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.Plan.Status != "actionable" || response.Symbol != "BTCUSDT" || response.Interval != "1h" || response.ContextTime != 1_700_000_000_000 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.Plan.Direction != "Long" || response.Plan.EntryPrice != 12 || response.Plan.TakeProfit != 13 || response.Plan.StopLoss != 11 || response.ReferencePrice != 12 {
		t.Fatalf("structured plan or reference price was not returned: %#v", response)
	}
	if len(response.ToolEvidence) != 1 || response.ToolEvidence[0].Tool != "tool_calling" || response.ToolEvidence[0].Summary != "当前 provider 不支持工具调用；未查询额外周期" {
		t.Fatalf("provider tool capability was not recorded: %+v", response.ToolEvidence)
	}
	if !strings.Contains(provider.messages[0].Content, "当前模型 provider 不支持工具调用") {
		t.Fatal("model was not told that additional periods were not queried")
	}
	if response.KeyLevels.Resistance == nil || *response.KeyLevels.Resistance != 13 || response.KeyLevels.Support == nil || *response.KeyLevels.Support != 11 {
		t.Fatalf("key levels were not derived from the current snapshot: %#v", response.KeyLevels)
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if strings.Contains(strings.ToLower(string(responseJSON)), "poc") {
		t.Errorf("response must not expose POC: %s", responseJSON)
	}
	if contextSource.calls != 1 || provider.calls != 1 {
		t.Fatalf("context calls=%d provider calls=%d", contextSource.calls, provider.calls)
	}
	if len(provider.messages) != 4 || provider.messages[0].Role != "system" || !strings.Contains(provider.messages[0].Content, "BTCUSDT") || !strings.Contains(provider.messages[0].Content, "1700000000000") {
		t.Fatalf("provider did not receive the current market snapshot and history: %#v", provider.messages)
	}
	for _, required := range []string{`"status"`, `"timing"`, `"direction"`, `"entry_type"`, `"entry_price"`, `"take_profit"`, `"stop_loss"`, `"confidence"`, `"leverage"`, `"analysis"`, `"key_levels"`, `"resistance":13`, `"support":11`, `Long`, `Short`, "成交量摘要", "OI（未平仓量）摘要", "资金费率", "未平仓量", "爆仓分布", "JSON"} {
		if !strings.Contains(provider.messages[0].Content, required) {
			t.Errorf("system prompt missing %q", required)
		}
	}
	if !strings.Contains(provider.messages[0].Content, `"volume_summary":{"latest":10,"average_prior_20":2,"relative_to_average":5}`) {
		t.Errorf("prompt must include latest volume and prior average: %s", provider.messages[0].Content)
	}
	if strings.Contains(strings.ToLower(provider.messages[0].Content), "poc") {
		t.Error("system prompt must not request or mention POC prices")
	}
	if provider.messages[1].Content != "Earlier question" || provider.messages[2].Content != "Earlier answer" || provider.messages[3].Role != "user" || provider.messages[3].Content != "What is the current structure?" {
		t.Fatalf("provider received wrong conversation order: %#v", provider.messages)
	}
}

func TestMarketAnalysisPlanAcceptsShortPriceOrdering(t *testing.T) {
	plan, err := parseMarketAnalysisPlan(`{"status":"actionable","timing":"right","direction":"Short","entry_type":"market","entry_price":100,"take_profit":90,"stop_loss":110,"confidence":64,"leverage":2,"analysis":"价格跌破支撑后，反弹失守则判断失效。"}`, 100)
	if err != nil {
		t.Fatalf("valid short plan rejected: %v", err)
	}
	if plan.Direction != "Short" || plan.Timing != "right" || plan.TakeProfit >= plan.EntryPrice || plan.StopLoss <= plan.EntryPrice {
		t.Fatalf("invalid short price ordering returned: %#v", plan)
	}
}

func TestMarketAnalysisChatRejectsInvalidStructuredProviderReplies(t *testing.T) {
	contextSource := &chatContextSourceStub{context: MarketAnalysisContext{
		Symbol: "BTCUSDT", Interval: "1h", ContextTime: 1,
		Candles:  []model.Candle{{Timestamp: 1, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1}},
		Analysis: &model.AnalysisResult{Symbol: "BTCUSDT", Interval: "1h", Timestamp: 1},
	}}
	for _, reply := range []string{
		"not JSON",
		`{"status":"actionable","timing":"left","direction":"Long","entry_type":"market","entry_price":100,"take_profit":99,"stop_loss":98,"confidence":70,"leverage":3,"analysis":"invalid TP"}`,
		`{"status":"actionable","timing":"left","direction":"Short","entry_type":"market","entry_price":100,"take_profit":101,"stop_loss":102,"confidence":70,"leverage":3,"analysis":"invalid TP"}`,
		`{"status":"actionable","timing":"left","direction":"Long","entry_type":"market","entry_price":100,"take_profit":110,"stop_loss":90,"confidence":101,"leverage":3,"analysis":"bad confidence"}`,
		`{"status":"actionable","timing":"left","direction":"Long","entry_type":"market","entry_price":100,"take_profit":110,"stop_loss":90,"confidence":70,"leverage":6,"analysis":"bad leverage"}`,
	} {
		chat := NewMarketAnalysisChatService(contextSource, &chatCompletionStub{reply: reply})
		if _, err := chat.Chat(context.Background(), MarketAnalysisChatRequest{Symbol: "BTCUSDT", Interval: "1h", Message: "Analyze"}); !errors.Is(err, ErrChatProviderFailed) {
			t.Errorf("reply %q error=%v, want ErrChatProviderFailed", reply, err)
		}
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

type chatOpenInterestSourceStub struct {
	samples []model.OpenInterestSample
	err     error
	symbol  string
	period  string
	limit   int
	calls   int
}

func (s *chatOpenInterestSourceStub) GetOpenInterestHistory(symbol, period string, limit int, _, _ *int64) ([]model.OpenInterestSample, error) {
	s.calls++
	s.symbol, s.period, s.limit = symbol, period, limit
	return s.samples, s.err
}

func TestMarketContextIncludesOIAndTreatsUnavailableAsEvidenceState(t *testing.T) {
	candles := make([]model.Candle, 30)
	for i := range candles {
		price := 100 + float64(i)
		candles[i] = model.Candle{Timestamp: int64(i+1) * 60_000, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 2}
	}
	oiSource := &chatOpenInterestSourceStub{samples: []model.OpenInterestSample{
		{Timestamp: 2, Quantity: 12, Value: 1500},
		{Timestamp: 1, Quantity: 10, Value: 1000},
	}}
	ctxService := NewMarketContextServiceWithSources(&chatCandleSourceStub{candles: candles}, oiSource)
	marketContext, err := ctxService.GetMarketAnalysisContext(context.Background(), "BTCUSDT", "1h")
	if err != nil {
		t.Fatalf("GetMarketAnalysisContext returned error: %v", err)
	}
	if oiSource.calls != 1 || oiSource.symbol != "BTCUSDT" || oiSource.period != "1h" || oiSource.limit != 2 {
		t.Fatalf("OI query calls=%d symbol=%s period=%s limit=%d", oiSource.calls, oiSource.symbol, oiSource.period, oiSource.limit)
	}
	if marketContext.OpenInterest.Status != "available" || marketContext.OpenInterest.QuantityChangePercent == nil || *marketContext.OpenInterest.QuantityChangePercent != 20 || marketContext.OpenInterest.ValueChangePercent == nil || *marketContext.OpenInterest.ValueChangePercent != 50 || marketContext.OpenInterest.LatestQuantity != 12 || marketContext.OpenInterest.Timestamp != 2 {
		t.Fatalf("unexpected OI summary: %+v", marketContext.OpenInterest)
	}

	oiSource.err = errors.New("upstream unavailable")
	marketContext, err = ctxService.GetMarketAnalysisContext(context.Background(), "BTCUSDT", "1h")
	if err != nil {
		t.Fatalf("unavailable OI must not fail candle analysis: %v", err)
	}
	if marketContext.OpenInterest.Status != "unavailable" || marketContext.OpenInterest.ErrorCode != "source_unavailable" {
		t.Fatalf("unexpected unavailable OI state: %+v", marketContext.OpenInterest)
	}
}

func TestMarketAnalysisPlanAcceptsWaitWithoutExecutablePrices(t *testing.T) {
	plan, err := parseMarketAnalysisPlan(`{"status":"wait","timing":"undetermined","confidence":45,"analysis":"价格结构与 OI 证据不一致，等待突破后重新确认。"}`, 100)
	if err != nil {
		t.Fatalf("valid wait result rejected: %v", err)
	}
	if plan.Status != "wait" || plan.Timing != "undetermined" || plan.Direction != "" || plan.EntryPrice != 0 || plan.TakeProfit != 0 || plan.StopLoss != 0 {
		t.Fatalf("wait result contains an executable plan: %+v", plan)
	}
	for _, reply := range []string{
		`{"status":"wait","timing":"undetermined","direction":"Long","entry_price":100,"confidence":45,"analysis":"bad wait fields"}`,
		`{"status":"actionable","timing":"undetermined","direction":"Long","entry_type":"market","entry_price":100,"take_profit":110,"stop_loss":90,"confidence":70,"leverage":2,"analysis":"bad timing"}`,
	} {
		if _, err := parseMarketAnalysisPlan(reply, 100); err == nil {
			t.Errorf("invalid result accepted: %s", reply)
		}
	}
}
