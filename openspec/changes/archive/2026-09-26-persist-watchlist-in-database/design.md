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

1. **Use PostgreSQL through `database/sql` and `github.com/jackc/pgx/v5/stdlib`.** The shared Watchlist is small, but the project anticipates Agent integrations and may later run concurrent workers or multiple backend instances. PostgreSQL provides a central client/server database with transactional updates and row-level locking for conflicting writes. Agents will access data through backend APIs/tools; they will not receive direct database credentials. Keep connection settings in `WATCHLIST_DATABASE_URL`, supplied by the environment and never committed as credentials.

2. **Store ordered symbols as rows, not a serialized list.** A `watchlist_symbols` table uses the normalized symbol as its primary key and an integer position as its order. A singleton Watchlist state row carries a monotonically increasing revision. Transactions lock that state row while changing membership or order, then increment the revision. The service enforces unique membership, valid symbols, a bounded list size, and at least one remaining symbol.

3. **Use versioned PostgreSQL schema migrations and a singleton import state.** Backend startup applies pending schema migrations in order and fails clearly if PostgreSQL is unavailable or migration fails. The initial migration seeds the existing preset order and records that legacy import is pending. The first migration request locks the singleton state row and atomically replaces the seed with the caller's current visible legacy list, then marks migration complete. Concurrent or later migration attempts return the database's current symbols without overwriting them. If no valid legacy list is available, the seeded presets remain. The frontend removes old localStorage Watchlist keys only after the server confirms the resulting state.

4. **Expose read and mutation endpoints.** Add `GET /api/watchlist`, a one-time legacy import endpoint, and endpoints to add a symbol, remove a symbol, and save an order. Add validates new symbols through the existing Binance `SymbolValidator`; duplicate adds and removal of the last symbol return explicit client errors. Reordering submits the revision last observed by the client and the intended order. If another client or Agent has already changed the shared list, the server returns a conflict and the frontend reloads current state instead of silently overwriting that change.

5. **Make PostgreSQL the source of truth and keep access behind the backend.** The Pinia store loads the server list before using it as the active Watchlist, imports legacy localStorage only while the server reports migration pending, and applies mutations only after successful responses. On initial load failure, it may render the current browser's cached list as a temporary fallback but must display that it is unsynchronized and must not submit it as an implicit replacement. Agent-facing tools, if added later, must use the same validated backend service methods and must not connect directly to PostgreSQL.

6. **Run PostgreSQL as a first-class service.** Add a PostgreSQL service to Docker Compose with a named persistent volume, a health check, and backend startup ordering that waits for database readiness. Pass database credentials and the connection URL through untracked environment configuration; do not expose the PostgreSQL port publicly by default. Local `start.sh` development requires a running PostgreSQL instance and `WATCHLIST_DATABASE_URL`. Keep the database separate from `data/opportunities.db`, which the retired-feature cleanup removes.

## Risks / Trade-offs

- **All clients share edits because the service has no identity.** This matches the chosen scope; introducing accounts later requires a schema migration that adds a workspace or owner key and authorization rules.
- **The first browser to complete migration selects the initial legacy list.** A transactional one-time flag prevents subsequent clients from overwriting it; deployments must preserve the PostgreSQL volume.
- **PostgreSQL adds a service to self-hosted deployments.** Compose health checks and startup ordering make the dependency explicit; the database volume needs its own backup and restore procedure.
- **A shared global list can still have conflicting edits.** Use the Watchlist revision for optimistic concurrency, return conflicts explicitly, and keep transactions short. PostgreSQL row locks serialize writes to the singleton state while allowing unrelated reads and rows to proceed.

## Migration Plan

1. Deploy the PostgreSQL service, persistent volume, backend schema migrations, and APIs together through Docker Compose. For local development, configure `WATCHLIST_DATABASE_URL` to an isolated PostgreSQL database.
2. The first frontend load imports the current browser's visible legacy list once; subsequent clients use the stored shared list.
3. Retain old browser keys until import succeeds, then remove them. If database loading fails, keep the old browser state visible with an unsynced warning so a retry can migrate it later.
4. Rollback can redeploy the prior frontend/backend; the old browser keys remain for clients that completed no successful migration. Preserve the PostgreSQL volume during rollback; it is separate from the retired opportunities database and must not be deleted by legacy cleanup.
