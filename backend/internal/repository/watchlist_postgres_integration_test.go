//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kudaompq/ai_trending/backend/internal/database"
)

func newTestWatchlistRepository(t *testing.T) (*sql.DB, *PostgresWatchlistRepository) {
	t.Helper()
	dsn := os.Getenv("WATCHLIST_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("WATCHLIST_TEST_DATABASE_URL must point to an isolated test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	if !strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "watchlist_test") {
		t.Fatalf("refusing to reset non-test database %q", parsed.Path)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset isolated test schema: %v", err)
	}
	if err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("apply database migrations: %v", err)
	}
	return db, NewPostgresWatchlistRepository(db)
}

func TestPostgresWatchlistRepositorySeedsAndReadsOrderedDefaults(t *testing.T) {
	db, repo := newTestWatchlistRepository(t)
	if err := database.ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("reapply database migrations: %v", err)
	}
	got, err := repo.Get(context.Background())
	if err != nil {
		t.Fatalf("get seeded Watchlist: %v", err)
	}
	want := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "POLUSDT"}
	if !equalStrings(got.Symbols, want) || got.Revision != 1 || !got.LegacyImportPending {
		t.Fatalf("seeded Watchlist = %+v, want symbols %v, revision 1, pending import", got, want)
	}
}

func TestPostgresWatchlistRepositoryPersistsMutationsAcrossReopen(t *testing.T) {
	db, repo := newTestWatchlistRepository(t)
	ctx := context.Background()
	if _, err := repo.Add(ctx, "AVAXUSDT"); err != nil {
		t.Fatalf("add symbol: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	db, err := sql.Open("pgx", os.Getenv("WATCHLIST_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo = NewPostgresWatchlistRepository(db)
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("read after reopen: %v", err)
	}
	if got.Symbols[len(got.Symbols)-1] != "AVAXUSDT" || got.Revision != 2 {
		t.Fatalf("Watchlist after reopen = %+v, want AVAXUSDT appended at revision 2", got)
	}
}

func TestPostgresWatchlistRepositoryPersistsReorderAndRemoval(t *testing.T) {
	_, repo := newTestWatchlistRepository(t)
	ctx := context.Background()
	initial, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	order := append([]string(nil), initial.Symbols...)
	order[0], order[1] = order[1], order[0]
	reordered, err := repo.Reorder(ctx, initial.Revision, order)
	if err != nil {
		t.Fatalf("reorder symbols: %v", err)
	}
	removed, err := repo.Remove(ctx, "ADAUSDT")
	if err != nil {
		t.Fatalf("remove symbol: %v", err)
	}
	want := []string{"ETHUSDT", "BTCUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "POLUSDT"}
	if !equalStrings(reordered.Symbols[:2], []string{"ETHUSDT", "BTCUSDT"}) || !equalStrings(removed.Symbols, want) || removed.Revision != 3 {
		t.Fatalf("reorder result = %+v, removal result = %+v", reordered, removed)
	}
}

func TestPostgresWatchlistRepositoryRejectsStaleReorder(t *testing.T) {
	_, repo := newTestWatchlistRepository(t)
	ctx := context.Background()
	initial, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, "AVAXUSDT"); err != nil {
		t.Fatal(err)
	}
	staleOrder := append([]string(nil), initial.Symbols...)
	staleOrder[0], staleOrder[1] = staleOrder[1], staleOrder[0]
	got, err := repo.Reorder(ctx, initial.Revision, staleOrder)
	if err != ErrWatchlistConflict {
		t.Fatalf("stale reorder error = %v, want ErrWatchlistConflict", err)
	}
	if len(got.Symbols) == 0 || got.Symbols[len(got.Symbols)-1] != "AVAXUSDT" || got.Revision != 2 {
		t.Fatalf("conflict response = %+v, want latest committed list and revision", got)
	}
}

func TestPostgresWatchlistRepositoryImportsLegacyOnlyOnce(t *testing.T) {
	_, repo := newTestWatchlistRepository(t)
	ctx := context.Background()
	first, err := repo.ImportLegacy(ctx, []string{"ETHUSDT", "BTCUSDT"})
	if err != nil {
		t.Fatalf("first legacy import: %v", err)
	}
	second, err := repo.ImportLegacy(ctx, []string{"SOLUSDT"})
	if err != nil {
		t.Fatalf("second legacy import: %v", err)
	}
	if !equalStrings(first.Symbols, []string{"ETHUSDT", "BTCUSDT"}) ||
		!equalStrings(second.Symbols, first.Symbols) || second.LegacyImportPending {
		t.Fatalf("imports = first %+v, second %+v; want first ordered list retained and import complete", first, second)
	}
}

func TestPostgresWatchlistRepositorySerializesConcurrentLegacyImports(t *testing.T) {
	_, repo := newTestWatchlistRepository(t)
	inputs := [][]string{{"ETHUSDT", "BTCUSDT"}, {"SOLUSDT", "AVAXUSDT"}}
	results := make([]WatchlistSnapshot, len(inputs))
	errors := make([]error, len(inputs))
	start := make(chan struct{})
	var group sync.WaitGroup
	for i := range inputs {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			results[i], errors[i] = repo.ImportLegacy(ctx, inputs[i])
		}(i)
	}
	close(start)
	group.Wait()
	for _, err := range errors {
		if err != nil {
			t.Fatalf("concurrent import: %v", err)
		}
	}
	if !equalStrings(results[0].Symbols, results[1].Symbols) {
		t.Fatalf("concurrent import results differ: %v vs %v", results[0].Symbols, results[1].Symbols)
	}
	if !equalStrings(results[0].Symbols, inputs[0]) && !equalStrings(results[0].Symbols, inputs[1]) {
		t.Fatalf("committed import %v is not either submitted list", results[0].Symbols)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
