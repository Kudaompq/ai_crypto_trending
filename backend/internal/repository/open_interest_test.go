package repository

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

func TestGetOpenInterestHistoryQueriesAndParsesSamples(t *testing.T) {
	var query url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/futures/data/openInterestHist" {
			t.Errorf("path=%q, want /futures/data/openInterestHist", r.URL.Path)
		}
		query = r.URL.Query()
		_, _ = w.Write([]byte(`[
			{"symbol":"ETHUSDT","sumOpenInterest":"12.5","sumOpenInterestValue":"25000.75","timestamp":1700000300000},
			{"symbol":"ETHUSDT","sumOpenInterest":"10","sumOpenInterestValue":"20000","timestamp":1700000000000}
		]`))
	}))
	defer upstream.Close()
	client := futures.NewClient("", "")
	client.BaseURL = upstream.URL
	client.HTTPClient = upstream.Client()
	repo := NewBinanceRepositoryWithClient(client)
	start, end := int64(1_700_000_000_000), int64(1_700_000_600_000)

	got, err := repo.GetOpenInterestHistory("ETHUSDT", "5m", 2, &start, &end)
	if err != nil {
		t.Fatalf("GetOpenInterestHistory() error = %v", err)
	}
	if query.Get("symbol") != "ETHUSDT" || query.Get("period") != "5m" || query.Get("limit") != "2" {
		t.Errorf("query = %v, want symbol=ETHUSDT period=5m limit=2", query)
	}
	if query.Get("startTime") != strconv.FormatInt(start, 10) || query.Get("endTime") != strconv.FormatInt(end, 10) {
		t.Errorf("query range = %v, want startTime=%d endTime=%d", query, start, end)
	}
	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2: %+v", len(got), got)
	}
	if got[0].Timestamp != 1_700_000_000_000 || got[0].Quantity != 10 || got[0].Value != 20_000 {
		t.Errorf("first sample = %+v, want chronological first sample", got[0])
	}
}

func TestGetOpenInterestHistoryKeepsEmptyAndReportsInvalidDataOrUpstreamFailure(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		body       string
		wantLength int
		wantError  bool
	}{
		{name: "empty history", status: http.StatusOK, body: `[]`, wantLength: 0},
		{name: "invalid number", status: http.StatusOK, body: `[{"sumOpenInterest":"bad","sumOpenInterestValue":"1","timestamp":1}]`, wantError: true},
		{name: "upstream rate limit", status: http.StatusTooManyRequests, body: `{"code":-1003,"msg":"rate limit"}`, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer upstream.Close()
			client := futures.NewClient("", "")
			client.BaseURL = upstream.URL
			client.HTTPClient = upstream.Client()

			got, err := NewBinanceRepositoryWithClient(client).GetOpenInterestHistory("ETHUSDT", "15m", 10, nil, nil)
			if (err != nil) != test.wantError {
				t.Fatalf("GetOpenInterestHistory() error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && len(got) != test.wantLength {
				t.Errorf("got %d samples, want %d", len(got), test.wantLength)
			}
		})
	}
}
