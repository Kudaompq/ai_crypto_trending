## 1. Contract tests first

- [ ] 1.1 Add SQLite repository tests for default seeding, ordered reads, add/remove/reorder persistence across reopen, one-time legacy import, and concurrent import attempts; run them and observe behavior failures.
- [ ] 1.2 Add service and handler tests for symbol validation, duplicate add, last-symbol protection, stale reorder conflicts, database errors, and the Watchlist HTTP request/response contract; observe failures before implementation.
- [ ] 1.3 Add frontend API/store tests for server-authoritative loading, one-time legacy import, database-backed add/remove/reorder, mutation failure handling, and unsynchronized fallback; observe failures before implementation.

## 2. Backend persistence and API

- [ ] 2.1 Add and pin the SQLite driver, configure a persistent database path with `WATCHLIST_DB_PATH` override, and initialize schema/default symbols without touching `opportunities.db`.
- [ ] 2.2 Implement transactional ordered Watchlist storage and the atomic one-time legacy import flag.
- [ ] 2.3 Implement Watchlist service validation and add, remove, and reorder conflict semantics.
- [ ] 2.4 Add `GET`, one-time import, add, remove, and reorder handlers/routes with explicit errors and response shapes; run focused Go tests.
- [ ] 2.5 Run `go test -count=1 ./backend/...` and `git diff --check`.

## 3. Frontend database source of truth

- [ ] 3.1 Add Watchlist read/import/add/remove/reorder API methods and client tests.
- [ ] 3.2 Replace active localStorage Watchlist composition with the server list; import legacy keys only while the backend reports migration pending and remove those keys only after a successful import.
- [ ] 3.3 Make Dashboard wait for Watchlist initialization before starting market requests, show database synchronization errors, and preserve selection and price-stream scope after server mutations.
- [ ] 3.4 Run frontend Watchlist/store/dashboard tests, the full `npm test` suite, and `npm run build`.

## 4. Documentation and acceptance

- [ ] 4.1 Update README persistence behavior and note that one service instance has one shared Watchlist.
- [ ] 4.2 Run the app against a temporary database and verify add, remove, reorder, legacy import, another browser reading the same list, and persistence after backend restart without modifying the user's active Watchlist database.
- [ ] 4.3 Verify Docker data-volume path and health/API behavior; record any environment limitation instead of claiming unverified persistence.
