## 1. Watchlist order

- [x] 1.1 Add store and Dashboard tests for pointer-based reordering, preserving selection, reload persistence, and adding/removing symbols around a saved order.
- [x] 1.2 Run the focused frontend tests and confirm failures are caused by missing reorder behavior; establish only the minimal test seam needed if an interface is not yet available.
- [x] 1.3 Implement per-browser Watchlist ordering, stale/duplicate entry reconciliation, and drag handling without changing the selected symbol.
- [x] 1.4 Re-run focused tests and verify the order survives a browser reload in the actual page.

## 2. Binance open-interest endpoint

- [x] 2.1 Add repository tests using a local HTTP transport for Binance response parsing, query parameters, supported periods, empty responses, and upstream errors.
- [x] 2.2 Add service and handler tests for symbol validation, interval mapping, empty unsupported periods, time-range limits, and distinguishable upstream failures.
- [x] 2.3 Run the new backend tests and confirm behavior assertions fail before implementation.
- [x] 2.4 Implement the read-only open-interest repository/service/handler route using the pinned Binance SDK and no alternate data source.
- [x] 2.5 Run the focused backend tests and `go test -count=1 ./backend/...`.
- [x] 2.6 Verify the public Binance endpoint with controlled samples for an available period and for `1m`/`3m`; record the observed coverage and empty-data behavior.

## 3. Per-symbol chart studies

- [x] 3.1 Add tests for saving/restoring enabled MA, EMA, Bollinger Bands, and MACD settings per symbol, sharing settings across intervals, and rejecting invalid parameters.
- [x] 3.2 Run the focused frontend tests and confirm the intended behavior assertions fail before implementation.
- [x] 3.3 Implement chart study controls, price-pane and secondary-pane creation, parameter validation, and versioned per-symbol browser persistence.
- [x] 3.4 Run the focused frontend tests and verify studies restore after symbol and interval changes.

## 4. Open-interest chart study

- [x] 4.1 Add frontend API and chart tests for returned samples, empty unsupported periods/ranges, recoverable failures, cancellation, and stale-response isolation after symbol or interval changes.
- [x] 4.2 Run the focused frontend tests and confirm the intended behavior assertions fail before implementation.
- [x] 4.3 Implement the optional OI secondary pane using Binance samples only, leaving absent timestamps empty and displaying request errors separately from no-data results.
- [x] 4.4 Verify supported and unsupported intervals, a range outside Binance's retention window, reload persistence, and switching symbols in the browser.

## 5. Drawing tools and persistence

- [x] 5.1 Add chart tests for creating, adjusting, restoring, isolating by symbol, preserving time/price anchors across intervals, and deleting drawings.
- [x] 5.2 Run the focused frontend tests and confirm the intended behavior assertions fail before implementation.
- [x] 5.3 Implement the horizontal line, trend line, ray, parallel channel, and Fibonacci retracement controls with per-symbol persistence.
- [x] 5.4 Run the focused frontend tests and verify saved drawings in the browser after reload and interval changes.

## 6. Regression and acceptance

- [x] 6.1 Run `npm test` and `npm run build` in `frontend/`, plus `go test -count=1 ./backend/...` and `git diff --check` from the repository root.
- [x] 6.2 Exercise the complete browser flow: reorder the watchlist, change study settings and drawings for two symbols, switch intervals, reload, and confirm each symbol restores only its own state.
- [x] 6.3 Record any failed, skipped, unavailable, or unverified checks and leave the change incomplete until required behavior and acceptance checks pass.
