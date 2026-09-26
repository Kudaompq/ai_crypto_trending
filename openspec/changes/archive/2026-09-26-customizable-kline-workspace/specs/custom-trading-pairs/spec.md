## ADDED Requirements

### Requirement: Reorder and restore the watchlist
The system SHALL let users drag a visible watchlist symbol to a new position. The resulting order SHALL be saved in the current browser and restored after reload. Reordering SHALL not change the selected trading pair. Newly added symbols SHALL be appended to the saved order, and removed symbols SHALL no longer appear in it.

#### Scenario: Move a symbol in the watchlist
- **WHEN** the user drags a symbol before or after another visible symbol
- **THEN** the watchlist displays the new order without changing the selected symbol

#### Scenario: Restore the chosen order
- **WHEN** the user reloads the page after reordering symbols
- **THEN** the watchlist restores the same order

#### Scenario: Add or remove a symbol after reordering
- **WHEN** the user adds a symbol or removes a visible symbol
- **THEN** the new symbol is appended to the watchlist order and a removed symbol is discarded from that order
