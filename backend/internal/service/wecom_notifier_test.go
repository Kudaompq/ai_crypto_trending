package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

func TestWeComNotifierSendsTextAndRequiresSuccessfulResponse(t *testing.T) {
	var gotMethod, gotContentType, gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotContentType = r.Method, r.Header.Get("Content-Type")
		var body struct {
			MessageType string `json:"msgtype"`
			Text        struct {
				Content string `json:"content"`
			} `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode WeCom request: %v", err)
		}
		gotContent = body.Text.Content
		if body.MessageType != "text" {
			t.Errorf("msgtype=%q, want text", body.MessageType)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer server.Close()
	notifier := NewWeComNotifierWithClient(server.URL+"?key=private-webhook-token", server.Client(), 0)
	status, err := notifier.Send(context.Background(), "BTCUSDT · 右侧 Long")
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if status != http.StatusOK || gotMethod != http.MethodPost || gotContentType != "application/json" || gotContent != "BTCUSDT · 右侧 Long" {
		t.Fatalf("status=%d method=%q type=%q content=%q", status, gotMethod, gotContentType, gotContent)
	}
}

func TestWeComNotifierRejectsHTTPJSONAndAPIErrorsWithoutExposingWebhook(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		wantCode string
		apiCode  int
	}{
		{name: "HTTP failure", status: http.StatusServiceUnavailable, body: `{"errcode":0}`, wantCode: "http_status"},
		{name: "invalid JSON", status: http.StatusOK, body: "not json", wantCode: "invalid_response"},
		{name: "missing response code", status: http.StatusOK, body: `{}`, wantCode: "invalid_response"},
		{name: "WeCom rejected", status: http.StatusOK, body: `{"errcode":93000,"errmsg":"private detail"}`, wantCode: "wecom_rejected", apiCode: 93000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			const secret = "private-webhook-token"
			notifier := NewWeComNotifierWithClient(server.URL+"?key="+secret, server.Client(), 0)
			_, err := notifier.Send(context.Background(), "notification")
			var deliveryErr *WeComDeliveryError
			if !errors.As(err, &deliveryErr) || deliveryErr.Code != tc.wantCode || deliveryErr.APICode != tc.apiCode {
				t.Fatalf("error=%v, want safe code %q", err, tc.wantCode)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "private detail") {
				t.Fatalf("delivery error exposed sensitive details: %v", err)
			}
		})
	}
}

func TestWeComWebhookConfigurationIsServerSideAndOptional(t *testing.T) {
	t.Setenv("WECOM_WEBHOOK_URL", "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-key")
	t.Setenv("WECOM_TIMEOUT", "3s")
	config, configured := WeComWebhookConfigFromEnv()
	if !configured || config.Timeout != 3*time.Second || config.URL == "" {
		t.Fatalf("configuration=%+v configured=%v", config, configured)
	}
	t.Setenv("WECOM_WEBHOOK_URL", "")
	if _, configured := WeComWebhookConfigFromEnv(); configured {
		t.Fatal("empty webhook was treated as configured")
	}
	t.Setenv("WECOM_WEBHOOK_URL", "file:///tmp/secret")
	if _, configured := WeComWebhookConfigFromEnv(); configured {
		t.Fatal("non-HTTP webhook was treated as configured")
	}
	t.Setenv("WECOM_WEBHOOK_URL", "https://attacker.example/cgi-bin/webhook/send?key=test")
	if _, configured := WeComWebhookConfigFromEnv(); configured {
		t.Fatal("non-WeCom host was treated as configured")
	}
}

func TestWeComNotifierHonorsMessageSizeAndRateLimit(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"errcode":0}`)
	}))
	defer server.Close()
	notifier := NewWeComNotifierWithClient(server.URL, server.Client(), 20*time.Millisecond)
	started := time.Now()
	for i := 0; i < 2; i++ {
		if _, err := notifier.Send(context.Background(), "ok"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 || time.Since(started) < 15*time.Millisecond {
		t.Fatalf("calls=%d elapsed=%s; want two sends separated by the configured interval", calls, time.Since(started))
	}
	if _, err := notifier.Send(context.Background(), strings.Repeat("x", wecomMaxMessageBytes+1)); err == nil {
		t.Fatal("oversized message was sent")
	}
}

func TestPackAlertItemsKeepsBlocksWithinUTF8ByteLimit(t *testing.T) {
	items := []repository.ScheduledAnalysisOutboxItem{
		{ID: "a", Payload: "BTCUSDT\n" + strings.Repeat("币", 500)},
		{ID: "b", Payload: "ETHUSDT\n" + strings.Repeat("以", 500)},
		{ID: "c", Payload: "SOLUSDT\n" + strings.Repeat("索", 500)},
	}
	groups := packAlertItems(items, 2048)
	if len(groups) < 2 {
		t.Fatalf("packed into %d message(s), expected byte-boundary split", len(groups))
	}
	var seen int
	for _, group := range groups {
		if len([]byte(group.body)) > 2048 || !utf8.ValidString(group.body) {
			t.Fatalf("message exceeds UTF-8 byte limit: bytes=%d valid=%v", len([]byte(group.body)), utf8.ValidString(group.body))
		}
		seen += len(group.items)
	}
	if seen != len(items) {
		t.Fatalf("packed %d of %d complete alert blocks", seen, len(items))
	}
}

func TestPackAlertItemsPrioritizesHigherConfidenceFirst(t *testing.T) {
	items := []repository.ScheduledAnalysisOutboxItem{
		{ID: "low", Confidence: 72, Payload: "LOW"},
		{ID: "high", Confidence: 91, Payload: "HIGH"},
	}
	groups := packAlertItems(items, 2048)
	if len(groups) != 1 || len(groups[0].items) != 2 || groups[0].items[0].ID != "high" || !strings.HasPrefix(groups[0].body, "HIGH") {
		t.Fatalf("alert order=%+v", groups)
	}
}
