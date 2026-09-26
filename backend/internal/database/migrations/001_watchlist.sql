CREATE TABLE watchlist_state (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton = TRUE),
    revision BIGINT NOT NULL CHECK (revision > 0),
    legacy_import_pending BOOLEAN NOT NULL
);

CREATE TABLE watchlist_symbols (
    symbol TEXT PRIMARY KEY,
    position INTEGER NOT NULL CHECK (position >= 0)
);

CREATE INDEX watchlist_symbols_position_idx ON watchlist_symbols(position);

INSERT INTO watchlist_state(singleton, revision, legacy_import_pending)
VALUES (TRUE, 1, TRUE);

INSERT INTO watchlist_symbols(symbol, position) VALUES
    ('BTCUSDT', 0),
    ('ETHUSDT', 1),
    ('BNBUSDT', 2),
    ('SOLUSDT', 3),
    ('XRPUSDT', 4),
    ('ADAUSDT', 5),
    ('DOGEUSDT', 6),
    ('POLUSDT', 7);
