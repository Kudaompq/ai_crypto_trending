## 1. Shared preference behavior

- [x] 1.1 Add storage and chart-component tests proving study selections and parameters are shared across symbols while drawings remain isolated per symbol, including migration from the previous storage format.
- [x] 1.2 Run the focused frontend tests and confirm the new cross-symbol assertions fail against the current per-symbol implementation.

## 2. Storage and chart updates

- [x] 2.1 Implement the version 2 chart preference document with one shared study configuration and per-symbol drawings; migrate saved version 1 data without losing drawings.
- [x] 2.2 Update chart symbol switching to retain shared study settings, restore only the selected symbol's drawings, and keep OI requests scoped to the selected symbol and interval.
- [x] 2.3 Run the focused frontend tests and confirm cross-symbol settings, drawings, and migration behavior pass.

## 3. Regression and acceptance

- [x] 3.1 Run the full frontend test suite, TypeScript/Vite build, strict OpenSpec validation, and `git diff --check`.
- [x] 3.2 Verify in the running browser that changing indicators on one symbol updates another symbol, drawings remain isolated, and the settings survive reload and interval changes.
