package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type watchlistServiceStub struct {
	snapshot        repository.WatchlistSnapshot
	getErr          error
	importErr       error
	addErr          error
	removeErr       error
	reorderErr      error
	imported        []string
	added           string
	removed         string
	reorderRevision int64
	reorderSymbols  []string
}

func (s *watchlistServiceStub) Get(context.Context) (repository.WatchlistSnapshot, error) {
	return s.snapshot, s.getErr
}

func (s *watchlistServiceStub) ImportLegacy(_ context.Context, symbols []string) (repository.WatchlistSnapshot, error) {
	s.imported = append([]string(nil), symbols...)
	return s.snapshot, s.importErr
}

func (s *watchlistServiceStub) Add(_ context.Context, symbol string) (repository.WatchlistSnapshot, error) {
	s.added = symbol
	return s.snapshot, s.addErr
}

func (s *watchlistServiceStub) Remove(_ context.Context, symbol string) (repository.WatchlistSnapshot, error) {
	s.removed = symbol
	return s.snapshot, s.removeErr
}

func (s *watchlistServiceStub) Reorder(_ context.Context, revision int64, symbols []string) (repository.WatchlistSnapshot, error) {
	s.reorderRevision = revision
	s.reorderSymbols = append([]string(nil), symbols...)
	return s.snapshot, s.reorderErr
}

func newWatchlistHandlerTestRouter(svc *watchlistServiceStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterWatchlistRoutes(router.Group("/api"), svc)
	return router
}

func TestWatchlistHandlerGetReturnsSharedListAndRevision(t *testing.T) {
	svc := &watchlistServiceStub{snapshot: repository.WatchlistSnapshot{
		Symbols: []string{"BTCUSDT", "ETHUSDT"}, Revision: 7, LegacyImportPending: true,
	}}
	response := httptest.NewRecorder()
	newWatchlistHandlerTestRouter(svc).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/watchlist", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", response.Code, response.Body.String())
	}
	var payload repository.WatchlistSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(payload.Symbols, svc.snapshot.Symbols) || payload.Revision != 7 || !payload.LegacyImportPending {
		t.Fatalf("GET payload=%+v", payload)
	}
}

func TestWatchlistHandlerSupportsImportAddRemoveAndReorderContracts(t *testing.T) {
	svc := &watchlistServiceStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"ETHUSDT", "BTCUSDT"}, Revision: 3}}
	router := newWatchlistHandlerTestRouter(svc)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
		}
		return response
	}

	request(http.MethodPost, "/api/watchlist/import-legacy", `{"symbols":["ETHUSDT","BTCUSDT"]}`)
	if !reflect.DeepEqual(svc.imported, []string{"ETHUSDT", "BTCUSDT"}) {
		t.Fatalf("imported=%v", svc.imported)
	}
	request(http.MethodPost, "/api/watchlist/symbols", `{"symbol":"AVAXUSDT"}`)
	if svc.added != "AVAXUSDT" {
		t.Fatalf("added=%q", svc.added)
	}
	request(http.MethodDelete, "/api/watchlist/symbols/ADAUSDT", "")
	if svc.removed != "ADAUSDT" {
		t.Fatalf("removed=%q", svc.removed)
	}
	request(http.MethodPut, "/api/watchlist/order", `{"revision":2,"symbols":["ETHUSDT","BTCUSDT"]}`)
	if svc.reorderRevision != 2 || !reflect.DeepEqual(svc.reorderSymbols, []string{"ETHUSDT", "BTCUSDT"}) {
		t.Fatalf("reorder args revision=%d symbols=%v", svc.reorderRevision, svc.reorderSymbols)
	}
}

func TestWatchlistHandlerReturnsCurrentListOnStaleReorder(t *testing.T) {
	svc := &watchlistServiceStub{
		snapshot:   repository.WatchlistSnapshot{Symbols: []string{"ETHUSDT", "BTCUSDT", "AVAXUSDT"}, Revision: 9},
		reorderErr: repository.ErrWatchlistConflict,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/watchlist/order", bytes.NewBufferString(`{"revision":8,"symbols":["BTCUSDT","ETHUSDT"]}`))
	newWatchlistHandlerTestRouter(svc).ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale reorder status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Symbols  []string `json:"symbols"`
		Revision int64    `json:"revision"`
		Code     string   `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if !reflect.DeepEqual(payload.Symbols, svc.snapshot.Symbols) || payload.Revision != 9 || payload.Code != "watchlist_conflict" {
		t.Fatalf("conflict payload=%+v", payload)
	}
}

func TestWatchlistHandlerMapsMutationErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "duplicate", err: repository.ErrWatchlistDuplicate, want: http.StatusConflict},
		{name: "last symbol", err: repository.ErrWatchlistLastSymbol, want: http.StatusConflict},
		{name: "database unavailable", err: errors.New("database unavailable"), want: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := &watchlistServiceStub{addErr: test.err}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/watchlist/symbols", bytes.NewBufferString(`{"symbol":"BTCUSDT"}`))
			newWatchlistHandlerTestRouter(svc).ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), test.want)
			}
		})
	}
}
