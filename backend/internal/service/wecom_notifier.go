package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const wecomMaxMessageBytes = 2048

var ErrWeComNotConfigured = errors.New("WeCom webhook is not configured")

type WeComWebhookConfig struct {
	URL     string
	Timeout time.Duration
}

type WeComDeliveryError struct {
	Code       string
	HTTPStatus int
	APICode    int
}

func (e *WeComDeliveryError) Error() string {
	if e == nil {
		return "WeCom delivery failed"
	}
	return fmt.Sprintf("WeCom delivery failed (%s)", e.Code)
}

type wecomHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type WeComNotifier struct {
	webhookURL  string
	client      wecomHTTPClient
	minInterval time.Duration
	limitMu     sync.Mutex
	lastSentAt  time.Time
}

func WeComWebhookConfigFromEnv() (WeComWebhookConfig, bool) {
	config := WeComWebhookConfig{URL: strings.TrimSpace(os.Getenv("WECOM_WEBHOOK_URL")), Timeout: 10 * time.Second}
	if timeout := strings.TrimSpace(os.Getenv("WECOM_TIMEOUT")); timeout != "" {
		if parsed, err := time.ParseDuration(timeout); err == nil && parsed > 0 {
			config.Timeout = parsed
		}
	}
	parsed, err := url.Parse(config.URL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "qyapi.weixin.qq.com") || parsed.Path != "/cgi-bin/webhook/send" || strings.TrimSpace(parsed.Query().Get("key")) == "" {
		return WeComWebhookConfig{}, false
	}
	return config, true
}

func NewWeComNotifierFromEnv() (*WeComNotifier, bool) {
	config, configured := WeComWebhookConfigFromEnv()
	if !configured {
		return nil, false
	}
	return NewWeComNotifier(config), true
}

func NewWeComNotifier(config WeComWebhookConfig) *WeComNotifier {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	return NewWeComNotifierWithClient(config.URL, &http.Client{Timeout: config.Timeout}, 3*time.Second)
}

func NewWeComNotifierWithClient(webhookURL string, client wecomHTTPClient, minInterval time.Duration) *WeComNotifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &WeComNotifier{webhookURL: strings.TrimSpace(webhookURL), client: client, minInterval: minInterval}
}

func (n *WeComNotifier) Send(ctx context.Context, content string) (int, error) {
	if n == nil || n.client == nil {
		return 0, ErrWeComNotConfigured
	}
	parsed, err := url.Parse(n.webhookURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return 0, ErrWeComNotConfigured
	}
	content = strings.TrimSpace(content)
	if content == "" || !utf8.ValidString(content) || len([]byte(content)) > wecomMaxMessageBytes {
		return 0, &WeComDeliveryError{Code: "invalid_message"}
	}
	if err := n.waitForRateLimit(ctx); err != nil {
		return 0, err
	}
	body, err := json.Marshal(struct {
		MessageType string `json:"msgtype"`
		Text        struct {
			Content string `json:"content"`
		} `json:"text"`
	}{MessageType: "text", Text: struct {
		Content string `json:"content"`
	}{Content: content}})
	if err != nil {
		return 0, &WeComDeliveryError{Code: "request_encode"}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, ErrWeComNotConfigured
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := n.client.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, &WeComDeliveryError{Code: "request_failed"}
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return response.StatusCode, &WeComDeliveryError{Code: "http_status", HTTPStatus: response.StatusCode}
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return response.StatusCode, &WeComDeliveryError{Code: "invalid_response", HTTPStatus: response.StatusCode}
	}
	var result struct {
		ErrCode *int `json:"errcode"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || result.ErrCode == nil {
		return response.StatusCode, &WeComDeliveryError{Code: "invalid_response", HTTPStatus: response.StatusCode}
	}
	if *result.ErrCode != 0 {
		return response.StatusCode, &WeComDeliveryError{Code: "wecom_rejected", HTTPStatus: response.StatusCode, APICode: *result.ErrCode}
	}
	return response.StatusCode, nil
}

func (n *WeComNotifier) waitForRateLimit(ctx context.Context) error {
	n.limitMu.Lock()
	defer n.limitMu.Unlock()
	if n.minInterval <= 0 || n.lastSentAt.IsZero() {
		n.lastSentAt = time.Now()
		return nil
	}
	wait := time.Until(n.lastSentAt.Add(n.minInterval))
	if wait <= 0 {
		n.lastSentAt = time.Now()
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		n.lastSentAt = time.Now()
		return nil
	}
}

func wecomErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var deliveryErr *WeComDeliveryError
	if errors.As(err, &deliveryErr) {
		return deliveryErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "delivery_failed"
}

func wecomAPICode(err error) int {
	var deliveryErr *WeComDeliveryError
	if errors.As(err, &deliveryErr) {
		return deliveryErr.APICode
	}
	return 0
}
