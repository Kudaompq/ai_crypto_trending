## MODIFIED Requirements

### Requirement: Display Binance open-interest history for the selected symbol
The system SHALL provide an open-interest study that can be enabled in a secondary chart pane and SHALL request historical samples from Binance USDⓈ-M for the currently selected trading pair and chart interval. The open-interest study selection SHALL be shared by all trading pairs in the current browser and SHALL be reused across all chart intervals. Samples SHALL be plotted only at timestamps and values returned by Binance; the system SHALL NOT interpolate, carry forward, or replace missing samples with another data source.

#### Scenario: Show available Binance samples
- **WHEN** the user enables open interest and Binance returns samples for the selected symbol, interval, and chart range
- **THEN** the secondary pane plots those samples at their returned timestamps

#### Scenario: Leave unsupported intervals empty
- **WHEN** the selected interval has no corresponding Binance open-interest history, including `1m` or `3m`
- **THEN** the open-interest pane contains no plotted values

#### Scenario: Leave unavailable history empty
- **WHEN** Binance has no open-interest samples for part or all of the requested chart range
- **THEN** the pane contains values only where samples were returned and leaves the unavailable range empty

#### Scenario: Share open-interest selection across symbols
- **WHEN** the user enables open interest for one symbol and switches to another symbol
- **THEN** open interest remains enabled for the other symbol while its samples are requested for that selected symbol

#### Scenario: Keep open-interest selection across intervals
- **WHEN** the user changes the chart interval
- **THEN** open interest remains enabled and the system requests samples for the newly selected interval

#### Scenario: Isolate samples when changing symbols
- **WHEN** the selected trading pair changes while open-interest data is loading
- **THEN** samples from the previous symbol are not displayed for the newly selected symbol

#### Scenario: Report a failed data request
- **WHEN** the Binance open-interest request fails
- **THEN** the chart keeps its other studies and K-line data, shows a recoverable open-interest load error, and does not present stale samples as current-symbol data
