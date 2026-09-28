package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type OpenAICompatibleConfig struct {
	Endpoint   string
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxHistory int
}

type OpenAICompatibleProvider struct {
	config OpenAICompatibleConfig
	client interface {
		Do(*http.Request) (*http.Response, error)
	}
}

func NewOpenAICompatibleProvider(config OpenAICompatibleConfig) *OpenAICompatibleProvider {
	config = withOpenAICompatibleDefaults(config)
	return &OpenAICompatibleProvider{config: config, client: &http.Client{Timeout: config.Timeout}}
}

func NewOpenAICompatibleProviderWithClient(config OpenAICompatibleConfig, client interface {
	Do(*http.Request) (*http.Response, error)
}) *OpenAICompatibleProvider {
	if client == nil {
		client = &http.Client{Timeout: withOpenAICompatibleDefaults(config).Timeout}
	}
	return &OpenAICompatibleProvider{config: withOpenAICompatibleDefaults(config), client: client}
}

func OpenAICompatibleConfigFromEnv() (OpenAICompatibleConfig, bool) {
	config := OpenAICompatibleConfig{
		Endpoint:   strings.TrimSpace(os.Getenv("AI_CHAT_ENDPOINT")),
		APIKey:     strings.TrimSpace(os.Getenv("AI_CHAT_API_KEY")),
		Model:      strings.TrimSpace(os.Getenv("AI_CHAT_MODEL")),
		Timeout:    20 * time.Second,
		MaxHistory: 20,
	}
	if timeout := strings.TrimSpace(os.Getenv("AI_CHAT_TIMEOUT")); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil && parsed > 0 {
			config.Timeout = parsed
		}
	}
	config = withOpenAICompatibleDefaults(config)
	return config, config.Endpoint != "" && config.APIKey != "" && config.Model != ""
}

func withOpenAICompatibleDefaults(config OpenAICompatibleConfig) OpenAICompatibleConfig {
	if config.Timeout <= 0 {
		config.Timeout = 20 * time.Second
	}
	if config.MaxHistory <= 0 {
		config.MaxHistory = 20
	}
	return config
}

func (p *OpenAICompatibleProvider) Complete(ctx context.Context, messages []ChatMessage) (string, error) {
	result, err := p.CompleteWithTools(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	if result.Content == "" || len(result.ToolCalls) > 0 {
		return "", ErrChatProviderFailed
	}
	return result.Content, nil
}

func (p *OpenAICompatibleProvider) CompleteWithTools(ctx context.Context, messages []ChatMessage, tools []ChatToolDefinition) (ChatCompletionResult, error) {
	if p == nil || p.client == nil || strings.TrimSpace(p.config.Endpoint) == "" || strings.TrimSpace(p.config.APIKey) == "" || strings.TrimSpace(p.config.Model) == "" {
		return ChatCompletionResult{}, ErrChatProviderNotConfigured
	}
	endpoint, err := url.Parse(p.config.Endpoint)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return ChatCompletionResult{}, ErrChatProviderNotConfigured
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	if !strings.HasSuffix(endpoint.Path, "/chat/completions") && endpoint.Path != "/chat/completions" {
		endpoint.Path += "/chat/completions"
	}
	endpoint.RawPath = ""
	if len(messages) == 0 {
		return ChatCompletionResult{}, ErrInvalidChatRequest
	}
	for _, message := range messages {
		if !isValidProviderMessage(message) {
			return ChatCompletionResult{}, ErrInvalidChatRequest
		}
	}
	for _, tool := range tools {
		if tool.Type != "function" || strings.TrimSpace(tool.Function.Name) == "" || len(tool.Function.Parameters) == 0 || !json.Valid(tool.Function.Parameters) {
			return ChatCompletionResult{}, ErrInvalidChatRequest
		}
	}

	messages = limitChatMessages(messages, p.config.MaxHistory)
	body, err := json.Marshal(struct {
		Model    string               `json:"model"`
		Messages []ChatMessage        `json:"messages"`
		Tools    []ChatToolDefinition `json:"tools,omitempty"`
	}{Model: p.config.Model, Messages: messages, Tools: tools})
	if err != nil {
		return ChatCompletionResult{}, ErrChatProviderFailed
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResult{}, ErrChatProviderFailed
	}
	request.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return ChatCompletionResult{}, ErrChatProviderFailed
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ChatCompletionResult{}, fmt.Errorf("%w: upstream status %d", ErrChatProviderFailed, response.StatusCode)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content   json.RawMessage `json:"content"`
				ToolCalls []ChatToolCall  `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil || len(result.Choices) == 0 {
		return ChatCompletionResult{}, ErrChatProviderFailed
	}
	choice := result.Choices[0].Message
	content := ""
	if len(choice.Content) > 0 && string(choice.Content) != "null" {
		if err := json.Unmarshal(choice.Content, &content); err != nil {
			return ChatCompletionResult{}, ErrChatProviderFailed
		}
	}
	content = strings.TrimSpace(content)
	for _, call := range choice.ToolCalls {
		if strings.TrimSpace(call.ID) == "" || call.Type != "function" || strings.TrimSpace(call.Function.Name) == "" || !json.Valid([]byte(call.Function.Arguments)) {
			return ChatCompletionResult{}, ErrChatProviderFailed
		}
	}
	if content == "" && len(choice.ToolCalls) == 0 {
		return ChatCompletionResult{}, ErrChatProviderFailed
	}
	return ChatCompletionResult{Content: content, ToolCalls: choice.ToolCalls}, nil
}

func isValidProviderMessage(message ChatMessage) bool {
	if message.Role != "system" && message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
		return false
	}
	if message.Role == "tool" {
		return strings.TrimSpace(message.ToolCallID) != "" && strings.TrimSpace(message.Content) != ""
	}
	if message.Role == "assistant" && len(message.ToolCalls) > 0 {
		return true
	}
	return strings.TrimSpace(message.Content) != ""
}

func limitChatMessages(messages []ChatMessage, maxHistory int) []ChatMessage {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	if len(messages) <= maxHistory+1 {
		return messages
	}
	if messages[0].Role == "system" {
		limited := make([]ChatMessage, 0, maxHistory+1)
		limited = append(limited, messages[0])
		limited = append(limited, messages[len(messages)-maxHistory:]...)
		return limited
	}
	return messages[len(messages)-maxHistory:]
}
