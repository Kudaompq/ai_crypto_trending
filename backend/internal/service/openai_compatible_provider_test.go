package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAICompatibleProviderSendsConfiguredRequestAndParsesReply(t *testing.T) {
	var gotAuthorization, gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel = body.Model
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || len(body.Messages) != 2 || body.Messages[1].Content != "Analyze BTC" {
			t.Errorf("unexpected provider request: method=%s path=%s body=%#v", r.Method, r.URL.Path, body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"BTC response"}}]}`)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Endpoint: server.URL + "/v1/chat/completions", APIKey: "test-provider-key", Model: "test-model", Timeout: time.Second,
	})
	reply, err := provider.Complete(context.Background(), []ChatMessage{{Role: "system", Content: "Market context"}, {Role: "user", Content: "Analyze BTC"}})
	if err != nil || reply != "BTC response" {
		t.Fatalf("Complete reply=%q err=%v", reply, err)
	}
	if gotAuthorization != "Bearer test-provider-key" || gotModel != "test-model" {
		t.Fatalf("authorization=%q model=%q", gotAuthorization, gotModel)
	}
}

func TestOpenAICompatibleProviderAppendsChatCompletionsPathToBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		wantPath string
	}{
		{name: "root base URL", wantPath: "/chat/completions"},
		{name: "versioned base URL", basePath: "/v1", wantPath: "/v1/chat/completions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Errorf("request path = %q, want %q", r.URL.Path, tt.wantPath)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
			}))
			defer server.Close()

			provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
				Endpoint: server.URL + tt.basePath, APIKey: "test-key", Model: "test-model",
			})
			if reply, err := provider.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil || reply != "ok" {
				t.Fatalf("Complete() = (%q, %v), want (\"ok\", nil)", reply, err)
			}
		})
	}
}

func TestOpenAICompatibleProviderSanitizesUpstreamFailure(t *testing.T) {
	const secret = "private-provider-secret"
	const upstreamDetail = "account token leaked upstream"
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w, `{"error":%q}`, upstreamDetail)
	}))
	defer server.Close()
	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Endpoint: server.URL, APIKey: secret, Model: "test-model", Timeout: time.Second,
	})
	_, err := provider.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "Hi"}})
	if err == nil || hits.Load() != 1 || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), upstreamDetail) {
		t.Fatalf("provider exposed sensitive upstream details: %v", err)
	}
}

func TestOpenAICompatibleProviderHonorsTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"late response"}}]}`)
	}))
	defer server.Close()
	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Endpoint: server.URL, APIKey: "key", Model: "test-model", Timeout: 20 * time.Millisecond,
	})
	completed := make(chan error, 1)
	go func() {
		_, err := provider.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "Hi"}})
		completed <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("request did not reach the local provider")
	}
	var err error
	select {
	case err = <-completed:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("provider call did not return after the configured timeout")
	}
	close(release)
	if err == nil {
		t.Fatal("provider timeout returned success")
	}
}

func TestOpenAICompatibleConfigFromEnvRequiresEndpointKeyAndModel(t *testing.T) {
	t.Setenv("AI_CHAT_ENDPOINT", "https://llm.example/v1/chat/completions")
	t.Setenv("AI_CHAT_API_KEY", "configured-secret")
	t.Setenv("AI_CHAT_MODEL", "market-model")

	config, configured := OpenAICompatibleConfigFromEnv()
	if !configured || config.Endpoint != "https://llm.example/v1/chat/completions" || config.APIKey != "configured-secret" || config.Model != "market-model" {
		t.Fatalf("configuration=%#v configured=%v", config, configured)
	}

	t.Setenv("AI_CHAT_API_KEY", "")
	if _, configured = OpenAICompatibleConfigFromEnv(); configured {
		t.Fatal("partial provider configuration was treated as enabled")
	}
}
