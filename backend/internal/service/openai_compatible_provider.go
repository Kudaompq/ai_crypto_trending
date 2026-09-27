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
	if p == nil || p.client == nil || strings.TrimSpace(p.config.Endpoint) == "" || strings.TrimSpace(p.config.APIKey) == "" || strings.TrimSpace(p.config.Model) == "" {
		return "", ErrChatProviderNotConfigured
	}
	endpoint, err := url.Parse(p.config.Endpoint)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return "", ErrChatProviderNotConfigured
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	if !strings.HasSuffix(endpoint.Path, "/chat/completions") && endpoint.Path != "/chat/completions" {
		endpoint.Path += "/chat/completions"
	}
	endpoint.RawPath = ""
	if len(messages) == 0 {
		return "", ErrInvalidChatRequest
	}
	for _, message := range messages {
		if (message.Role != "system" && message.Role != "user" && message.Role != "assistant") || strings.TrimSpace(message.Content) == "" {
			return "", ErrInvalidChatRequest
		}
	}

	messages = limitChatMessages(messages, p.config.MaxHistory)
	body, err := json.Marshal(struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
	}{Model: p.config.Model, Messages: messages})
	if err != nil {
		return "", ErrChatProviderFailed
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return "", ErrChatProviderFailed
	}
	request.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return "", ErrChatProviderFailed
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%w: upstream status %d", ErrChatProviderFailed, response.StatusCode)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil || len(result.Choices) == 0 {
		return "", ErrChatProviderFailed
	}
	reply := strings.TrimSpace(result.Choices[0].Message.Content)
	if reply == "" {
		return "", ErrChatProviderFailed
	}
	return reply, nil
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
