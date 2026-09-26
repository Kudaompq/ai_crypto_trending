## Context

The current Pinia store composes preset symbols with browser-local custom-symbol, hidden-preset, and order keys. The backend has no database connection; its `backend-data` volume is already mounted at `/app/data` in Docker. There is no authentication or user identity, and the user chose one list shared by every client of this service. See `proposal.md` and the modified `custom-trading-pairs` requirements.

## Goals / Non-Goals

**Goals:**

- Make one ordered Watchlist authoritative across browsers and service restarts.
- Preserve the first browser's existing visible list through a one-time, race-safe migration when the database is first initialized.
- Keep database failures visible and avoid presenting failed mutations as committed.
- Keep market price REST/SSE subscriptions scoped to the current ordered list.

**Non-Goals:**

- Accounts, authentication, per-user or per-browser lists.
- Immediate push updates to already-open clients when another client edits the shared list; clients read the current state on load and after their own mutations.
- Storing chart studies, drawings, selected symbol, or other browser preferences in the database.

## Decisions

1. **Use SQLite with `database/sql` and `github.com/mattn/go-sqlite3`.** The application is self-hosted and needs one durable shared file. The existing Docker builder installs GCC and builds with CGO enabled; the runtime image already carries the data volume. The SQLite driver implements Go's `database/sql` interface and requires CGO/GCC, which matches the current container build. PostgreSQL would require a separate service and deployment configuration, while a JSON file would not provide transactional updates or database constraints.

2. **Store ordered symbols as rows, not a serialized list.** A `watchlist_symbols` table uses the normalized symbol as its primary key and an integer position as its order. Transactions protect add/remove/reorder updates, and the service enforces unique membership, valid symbols, a bounded list size, and at least one remaining symbol.

3. **Keep a singleton migration state.** Database initialization seeds the existing preset order and records that legacy import is pending. The first migration request atomically replaces that seed with the caller's current visible legacy list and marks migration complete. Later migration attempts return the database's current symbols without overwriting them. If no valid legacy list is available, the seeded presets remain. The frontend removes old localStorage Watchlist keys only after the server confirms the resulting state.

4. **Expose read and mutation endpoints.** Add `GET /api/watchlist`, a one-time legacy import endpoint, and endpoints to add a symbol, remove a symbol, and save an order. Add validates new symbols through the existing Binance `SymbolValidator`; duplicate adds and removal of the last symbol return explicit client errors. Reordering must submit exactly the current membership, so stale clients cannot silently erase another client's edits; a conflict response makes the frontend reload the current list.

5. **Make the database list the frontend source of truth.** The Pinia store loads the server list before using it as the active Watchlist, imports legacy localStorage only while the server reports migration pending, and applies mutations only after successful responses. On initial load failure, it may render the current browser's cached list as a temporary fallback but must display that it is unsynchronized and must not submit it as an implicit replacement.

6. **Use a data-volume path with an override.** Default the database path to `data/watchlist.db` relative to the backend working directory and allow `WATCHLIST_DB_PATH` to override it. This resolves to `backend/data/watchlist.db` for `start.sh` and `/app/data/watchlist.db` in the current Docker container. Keep it distinct from `data/opportunities.db`, which the retired-feature cleanup removes.

## Risks / Trade-offs

- **All clients share edits because the service has no identity.** This matches the chosen scope; introducing accounts later requires a schema migration that adds an owner key.
- **The first browser to complete migration selects the initial legacy list.** A transactional one-time flag prevents subsequent clients from overwriting it; deployments should keep the database file on persistent storage.
- **CGO makes local builds require a C compiler.** Docker already installs GCC; document the local compiler requirement and verify the existing macOS development build.
- **SQLite allows one writer at a time.** The list is small and writes are infrequent; use transactions and a bounded busy timeout, and keep database connections in one backend process.

## Migration Plan

1. Deploy the backend schema and APIs with the SQLite file located on the existing persistent volume.
2. The first frontend load imports the current browser's visible legacy list once; subsequent clients use the stored shared list.
3. Retain old browser keys until import succeeds, then remove them. If database loading fails, keep the old browser state visible with an unsynced warning so a retry can migrate it later.
4. Rollback can redeploy the prior frontend/backend; the old browser keys remain for clients that completed no successful migration. The new database file is separate from the retired opportunities database and must not be deleted by legacy cleanup.
