package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type WatchlistSnapshot struct {
	Symbols             []string `json:"symbols"`
	Revision            int64    `json:"revision"`
	LegacyImportPending bool     `json:"legacy_import_pending"`
}

var (
	ErrWatchlistConflict       = errors.New("watchlist revision conflict")
	ErrWatchlistDuplicate      = errors.New("watchlist symbol already exists")
	ErrWatchlistLastSymbol     = errors.New("watchlist must contain at least one symbol")
	ErrWatchlistSymbolNotFound = errors.New("watchlist symbol does not exist")
	ErrWatchlistInvalidOrder   = errors.New("watchlist order must contain the current symbols exactly once")
	ErrWatchlistLimit          = errors.New("watchlist symbol limit reached")
)

const MaxWatchlistSymbols = 100

type WatchlistRepository interface {
	Get(context.Context) (WatchlistSnapshot, error)
	ImportLegacy(context.Context, []string) (WatchlistSnapshot, error)
	Add(context.Context, string) (WatchlistSnapshot, error)
	Remove(context.Context, string) (WatchlistSnapshot, error)
	Reorder(context.Context, int64, []string) (WatchlistSnapshot, error)
}

type PostgresWatchlistRepository struct {
	db *sql.DB
}

func NewPostgresWatchlistRepository(db *sql.DB) *PostgresWatchlistRepository {
	return &PostgresWatchlistRepository{db: db}
}

func (r *PostgresWatchlistRepository) Get(ctx context.Context) (WatchlistSnapshot, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("begin watchlist read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	snapshot, err := readWatchlistSnapshot(ctx, tx)
	if err != nil {
		return WatchlistSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("commit watchlist read: %w", err)
	}
	return snapshot, nil
}

func (r *PostgresWatchlistRepository) Add(ctx context.Context, rawSymbol string) (WatchlistSnapshot, error) {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("begin add watchlist symbol: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := lockWatchlistState(ctx, tx); err != nil {
		return WatchlistSnapshot{}, err
	}
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM watchlist_symbols").Scan(&count); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("count watchlist symbols: %w", err)
	}
	if count >= MaxWatchlistSymbols {
		return WatchlistSnapshot{}, ErrWatchlistLimit
	}
	var position int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(position), -1) + 1 FROM watchlist_symbols").Scan(&position); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("find watchlist append position: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO watchlist_symbols(symbol, position) VALUES ($1, $2)", symbol, position); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WatchlistSnapshot{}, ErrWatchlistDuplicate
		}
		return WatchlistSnapshot{}, fmt.Errorf("insert watchlist symbol: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE watchlist_state SET revision = revision + 1 WHERE singleton = TRUE"); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("increment watchlist revision: %w", err)
	}
	return readAndCommitWatchlist(ctx, tx)
}

func (r *PostgresWatchlistRepository) Remove(ctx context.Context, rawSymbol string) (WatchlistSnapshot, error) {
	symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("begin remove watchlist symbol: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := lockWatchlistState(ctx, tx); err != nil {
		return WatchlistSnapshot{}, err
	}
	var position int
	if err := tx.QueryRowContext(ctx, "SELECT position FROM watchlist_symbols WHERE symbol = $1", symbol).Scan(&position); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			snapshot, readErr := readWatchlistSnapshot(ctx, tx)
			if readErr != nil {
				return WatchlistSnapshot{}, readErr
			}
			return snapshot, ErrWatchlistSymbolNotFound
		}
		return WatchlistSnapshot{}, fmt.Errorf("find watchlist symbol to remove: %w", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM watchlist_symbols").Scan(&count); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("count watchlist symbols: %w", err)
	}
	if count <= 1 {
		snapshot, readErr := readWatchlistSnapshot(ctx, tx)
		if readErr != nil {
			return WatchlistSnapshot{}, readErr
		}
		return snapshot, ErrWatchlistLastSymbol
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM watchlist_symbols WHERE symbol = $1", symbol); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("delete watchlist symbol: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE watchlist_symbols SET position = position - 1 WHERE position > $1", position); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("compact watchlist positions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE watchlist_state SET revision = revision + 1 WHERE singleton = TRUE"); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("increment watchlist revision: %w", err)
	}
	return readAndCommitWatchlist(ctx, tx)
}

func (r *PostgresWatchlistRepository) ImportLegacy(ctx context.Context, symbols []string) (WatchlistSnapshot, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("begin legacy watchlist import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	state, err := lockWatchlistState(ctx, tx)
	if err != nil {
		return WatchlistSnapshot{}, err
	}
	if !state.LegacyImportPending {
		return readAndCommitWatchlist(ctx, tx)
	}
	cleaned := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, raw := range symbols {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		cleaned = append(cleaned, symbol)
	}
	if len(cleaned) > 0 {
		if _, err := tx.ExecContext(ctx, "DELETE FROM watchlist_symbols"); err != nil {
			return WatchlistSnapshot{}, fmt.Errorf("clear seeded watchlist for legacy import: %w", err)
		}
		for position, symbol := range cleaned {
			if _, err := tx.ExecContext(ctx, "INSERT INTO watchlist_symbols(symbol, position) VALUES ($1, $2)", symbol, position); err != nil {
				return WatchlistSnapshot{}, fmt.Errorf("import legacy watchlist symbol: %w", err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE watchlist_state SET legacy_import_pending = FALSE, revision = revision + 1 WHERE singleton = TRUE"); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("finish legacy watchlist import: %w", err)
	}
	return readAndCommitWatchlist(ctx, tx)
}

func (r *PostgresWatchlistRepository) Reorder(ctx context.Context, expectedRevision int64, symbols []string) (WatchlistSnapshot, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("begin reorder watchlist: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	state, err := lockWatchlistState(ctx, tx)
	if err != nil {
		return WatchlistSnapshot{}, err
	}
	if state.Revision != expectedRevision {
		snapshot, readErr := readWatchlistSnapshot(ctx, tx)
		if readErr != nil {
			return WatchlistSnapshot{}, readErr
		}
		if err := tx.Commit(); err != nil {
			return WatchlistSnapshot{}, fmt.Errorf("commit watchlist conflict read: %w", err)
		}
		return snapshot, ErrWatchlistConflict
	}
	current, err := readWatchlistSnapshot(ctx, tx)
	if err != nil {
		return WatchlistSnapshot{}, err
	}
	if !sameMembership(current.Symbols, symbols) {
		return current, ErrWatchlistInvalidOrder
	}
	for position, symbol := range symbols {
		if _, err := tx.ExecContext(ctx, "UPDATE watchlist_symbols SET position = $1 WHERE symbol = $2", position, symbol); err != nil {
			return WatchlistSnapshot{}, fmt.Errorf("update watchlist order: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE watchlist_state SET revision = revision + 1 WHERE singleton = TRUE"); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("increment watchlist revision: %w", err)
	}
	return readAndCommitWatchlist(ctx, tx)
}

type watchlistQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readWatchlistSnapshot(ctx context.Context, queryer watchlistQueryer) (WatchlistSnapshot, error) {
	var snapshot WatchlistSnapshot
	if err := queryer.QueryRowContext(ctx, "SELECT revision, legacy_import_pending FROM watchlist_state WHERE singleton = TRUE").Scan(&snapshot.Revision, &snapshot.LegacyImportPending); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("read watchlist state: %w", err)
	}
	rows, err := queryer.QueryContext(ctx, "SELECT symbol FROM watchlist_symbols ORDER BY position, symbol")
	if err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("read ordered watchlist symbols: %w", err)
	}
	defer rows.Close()
	snapshot.Symbols = make([]string, 0)
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return WatchlistSnapshot{}, fmt.Errorf("scan watchlist symbol: %w", err)
		}
		snapshot.Symbols = append(snapshot.Symbols, symbol)
	}
	if err := rows.Err(); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("iterate watchlist symbols: %w", err)
	}
	return snapshot, nil
}

func lockWatchlistState(ctx context.Context, tx *sql.Tx) (WatchlistSnapshot, error) {
	var snapshot WatchlistSnapshot
	if err := tx.QueryRowContext(ctx, "SELECT revision, legacy_import_pending FROM watchlist_state WHERE singleton = TRUE FOR UPDATE").Scan(&snapshot.Revision, &snapshot.LegacyImportPending); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("lock watchlist state: %w", err)
	}
	return snapshot, nil
}

func readAndCommitWatchlist(ctx context.Context, tx *sql.Tx) (WatchlistSnapshot, error) {
	snapshot, err := readWatchlistSnapshot(ctx, tx)
	if err != nil {
		return WatchlistSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return WatchlistSnapshot{}, fmt.Errorf("commit watchlist transaction: %w", err)
	}
	return snapshot, nil
}

func sameMembership(current, proposed []string) bool {
	if len(current) != len(proposed) {
		return false
	}
	set := make(map[string]struct{}, len(current))
	for _, symbol := range current {
		set[symbol] = struct{}{}
	}
	for _, symbol := range proposed {
		if _, exists := set[symbol]; !exists {
			return false
		}
		delete(set, symbol)
	}
	return len(set) == 0
}
