## ADDED Requirements

### Requirement: Configure chart studies per trading pair
The system SHALL let users enable and disable moving average (MA), exponential moving average (EMA), and Bollinger Band studies on the price pane, and MACD on a secondary chart pane. Users SHALL be able to set each study's applicable parameters. The enabled studies and their parameters SHALL be saved separately for each trading pair in the current browser and SHALL be reused for that pair across all chart intervals.

#### Scenario: Select and configure price-pane studies
- **WHEN** the user enables MA, EMA, or Bollinger Bands and sets valid parameters
- **THEN** the selected studies are drawn on the price pane using those parameters

#### Scenario: Select and configure a secondary study
- **WHEN** the user enables MACD and configures its periods
- **THEN** MACD appears in a secondary pane below the price chart

#### Scenario: Restore studies for a trading pair
- **WHEN** the user switches from one symbol to another and later returns to the first symbol
- **THEN** the first symbol's enabled studies and parameters are restored

#### Scenario: Keep study settings across intervals
- **WHEN** the user changes the chart interval for the same symbol
- **THEN** the same studies and parameters remain enabled for that symbol

#### Scenario: Reject invalid study parameters
- **WHEN** the user enters an invalid parameter combination, such as a non-positive period or a MACD fast period that is not less than its slow period
- **THEN** the system SHALL reject the invalid settings and SHALL not apply them to the chart

### Requirement: Create and manage common chart drawings
The system SHALL provide common drawing tools including a horizontal line, trend line, ray, parallel channel, and Fibonacci retracement. Users SHALL be able to create, adjust, and remove drawings. Drawings SHALL be saved separately for each trading pair in the current browser and SHALL remain associated with their price and time coordinates across chart interval changes.

#### Scenario: Draw and adjust an object
- **WHEN** the user selects a drawing tool, places an object on the chart, and adjusts one of its anchors
- **THEN** the chart displays the object at the selected coordinates

#### Scenario: Restore drawings after reload
- **WHEN** the user reloads the page with a symbol selected
- **THEN** the chart restores that symbol's saved drawings at their original price and time coordinates

#### Scenario: Keep drawings isolated by symbol
- **WHEN** the user switches to a different trading pair
- **THEN** drawings belonging to the previous pair are not displayed on the current pair's chart

#### Scenario: Keep drawings across intervals
- **WHEN** the user changes the chart interval and then returns to a time range containing a saved drawing anchor
- **THEN** the drawing remains attached to the same price and time coordinates

#### Scenario: Remove a drawing
- **WHEN** the user removes a saved drawing from the chart
- **THEN** that drawing is removed from the current pair's saved drawings and stays absent after reload
