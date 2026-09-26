## MODIFIED Requirements

### Requirement: Configure chart studies per trading pair
The system SHALL let users enable and disable moving average (MA), exponential moving average (EMA), and Bollinger Band studies on the price pane, and MACD on a secondary chart pane. Users SHALL be able to set each study's applicable parameters. The enabled studies and their parameters SHALL be shared by all trading pairs in the current browser and SHALL be reused across all chart intervals.

#### Scenario: Select and configure price-pane studies
- **WHEN** the user enables MA, EMA, or Bollinger Bands and sets valid parameters
- **THEN** the selected studies are drawn on the price pane using those parameters

#### Scenario: Select and configure a secondary study
- **WHEN** the user enables MACD and configures its periods
- **THEN** MACD appears in a secondary pane below the price chart

#### Scenario: Share studies across trading pairs
- **WHEN** the user changes the selected studies or parameters for one trading pair and switches to another pair
- **THEN** the same studies and parameters remain enabled for the other pair

#### Scenario: Keep study settings across intervals
- **WHEN** the user changes the chart interval
- **THEN** the same studies and parameters remain enabled

#### Scenario: Migrate existing study settings
- **WHEN** the browser contains saved per-pair study settings from the previous version
- **THEN** the settings for the currently selected pair are used as the shared settings when present, otherwise the first saved study settings are used, while drawings remain associated with their original trading pairs

#### Scenario: Reject invalid study parameters
- **WHEN** the user enters an invalid parameter combination, such as a non-positive period or a MACD fast period that is not less than its slow period
- **THEN** the system SHALL reject the invalid settings and SHALL not apply them to the chart
