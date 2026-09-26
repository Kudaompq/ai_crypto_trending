## Why

Watchlist membership and ordering currently live in browser `localStorage`, so a browser reset loses them and separate clients can disagree. There is no login or user identity in the application; this change makes the self-hosted deployment's Watchlist a single shared, durable list.

## What Changes

- Store the ordered Watchlist in PostgreSQL on the backend and make it authoritative for all clients.
- Add backend read and mutation APIs for loading, adding, removing, and reordering symbols.
- Import the first connected browser's existing local Watchlist once when a new database is initialized, then use the database state for later clients.
- Keep current symbol validation and the rule that at least one symbol remains.
- Run PostgreSQL as a Compose service with a dedicated persistent volume so Watchlist data survives container restarts.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `custom-trading-pairs`: change Watchlist persistence from the current browser to one ordered, database-backed list shared by all visitors, with one-time import of legacy browser state.

## Impact

- Backend database initialization, Watchlist repository/service/handler, and API routes.
- Frontend API client, analysis store, Watchlist loading and mutation flows.
- Go module dependencies, Docker Compose, and the custom trading-pairs specification and user documentation.
- PostgreSQL connection configuration and a dedicated Docker volume; local development connects to a configured PostgreSQL instance.
