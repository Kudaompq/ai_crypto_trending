## 1. Contract tests first

- [x] 1.1 Add PostgreSQL repository integration tests for default seeding, ordered reads, add/remove/reorder persistence, revision conflicts, one-time legacy import, and concurrent import attempts against an isolated test database; run them and observe behavior failures.
- [x] 1.2 Add service and handler tests for symbol validation, duplicate add, last-symbol protection, stale reorder conflicts, database errors, and the Watchlist HTTP request/response contract; observe failures before implementation.
- [x] 1.3 Add frontend API/store tests for server-authoritative loading, one-time legacy import, database-backed add/remove/reorder, mutation failure handling, and unsynchronized fallback; observe failures before implementation.

## 2. Backend persistence and API

- [x] 2.1 Add and pin `github.com/jackc/pgx/v5/stdlib`, configure `WATCHLIST_DATABASE_URL`, and add versioned PostgreSQL schema migrations and default symbols without touching `opportunities.db`.
- [x] 2.2 Add a PostgreSQL Compose service with a health check, private application network, credentials supplied through untracked environment configuration, and a dedicated named persistent volume; configure backend startup to wait for database readiness.
- [x] 2.3 Implement transactional ordered Watchlist storage, revision-based optimistic concurrency, and the atomic one-time legacy import state using PostgreSQL row locks.
- [x] 2.4 Implement Watchlist service validation and add, remove, and reorder conflict semantics.
- [x] 2.5 Add `GET`, one-time import, add, remove, and reorder handlers/routes with explicit errors and response shapes; run focused Go tests.
- [x] 2.6 Run repository integration tests against an isolated PostgreSQL instance, `go test -count=1 ./backend/...`, and `git diff --check`.

## 3. Frontend database source of truth

- [x] 3.1 Add Watchlist read/import/add/remove/reorder API methods and client tests.
- [x] 3.2 Replace active localStorage Watchlist composition with the server list; import legacy keys only while the backend reports migration pending and remove those keys only after a successful import.
- [x] 3.3 Make Dashboard wait for Watchlist initialization before starting market requests, show database synchronization errors, and preserve selection and price-stream scope after server mutations.
- [x] 3.4 Run frontend Watchlist/store/dashboard tests, the full `npm test` suite, and `npm run build`.

## 4. Documentation and acceptance

- [x] 4.1 Update README with PostgreSQL setup, `WATCHLIST_DATABASE_URL`, Compose persistence and backup behavior, and the one-shared-Watchlist scope.
- [x] 4.2 Run the app against an isolated PostgreSQL database and verify add, remove, reorder, legacy import, another client reading the same list, conflict handling, and persistence after backend restart without modifying the user's active database.
- [x] 4.3 Verify Compose database readiness, dedicated volume persistence, backend health/API behavior, and local `start.sh` configuration. Readiness and volume persistence passed with PostgreSQL 16; the configured `postgres:16-alpine` image pull was stopped because it advanced only about 2 MB in 30 seconds, and this image-specific limitation is recorded in the implementation report.
