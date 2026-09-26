## Why

The watchlist cannot be arranged around a user's preferred workflow, and the K-line chart offers no way to select the studies or drawing tools needed for individual analysis. Users need these chart choices to follow each trading pair across interval changes, while unavailable Binance open-interest history must remain visibly empty instead of being fabricated or silently substituted.

## What Changes

- Allow users to drag watchlist symbols into a preferred order and restore that order after reloading the browser.
- Add configurable price-pane studies (MA, EMA, Bollinger Bands), subchart studies (including MACD and open interest), and common chart drawing tools.
- Save study selections, parameters, and drawings per trading pair in browser storage; reuse them across all intervals for that pair.
- Load open-interest history from Binance USDⓈ-M data where available. Leave the OI pane without values when the selected interval or requested range has no Binance data.
- Clarify that chart-attached study panes and user-created drawings are supported while the retired standalone analysis cards, automatic support/resistance annotations, and trading-opportunity UI remain absent.

## Capabilities

### New Capabilities
- `open-interest-chart`: Display Binance USDⓈ-M open-interest history as a chart-attached study and leave unavailable periods empty.

### Modified Capabilities
- `custom-trading-pairs`: Persist and restore the user-defined order of watchlist symbols.
- `market-kline-chart`: Configure chart studies and drawing tools, with study settings and drawings scoped per symbol across intervals.
- `chart-analysis-display`: Keep retired standalone analysis panels removed while allowing user-selected studies attached to the K-line chart.

## Impact

- Frontend watchlist and analysis state, chart controls, KLineCharts integration, and browser-local persistence.
- A backend or frontend data-access path to Binance USDⓈ-M open-interest history, subject to the source's available periods and history window.
- OpenSpec specs and automated frontend/backend coverage for persistence, interval switching, data availability, and chart behavior.
