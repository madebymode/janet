# Janet

Janet is now a split Slack karma platform with three main binaries:

- `janet-bot`: listens to Slack events in Socket Mode and records live karma activity.
- `janet-server`: serves the analytics UI and JSON API from PostgreSQL-backed summary data.
- `janet-backfill`: imports historical Slack messages and reaction data into the database.

This branch replaces the older single-process bot/web setup with a database-first architecture aimed at long-lived analytics, historical backfills, and a richer web dashboard.

## What This Branch Adds

- Separates live bot traffic from the web UI and analytics server.
- Moves the app to PostgreSQL with a unified `karma_transactions` table plus cached summary tables.
- Adds a React dashboard with leaderboard, overview, analytics, activity, users, and popular-message views.
- Adds a backfill CLI for importing Slack history and emoji reactions.
- Adds cached Slack message enrichment for popular posts, including permalinks, author metadata, and attachment/image URLs.
- Adds Docker Compose stacks for local and production-style deployment.
- Adds a small maintenance command for filling missing `channel_id` values.

## Architecture

### Services

- `cmd/janet-bot`: live Slack bot service.
- `janet-server`: HTTP server for the dashboard and API.
- `cmd/janet-backfill`: ad hoc or bulk history import tool.
- `cmd/janet-maintenance`: one-off database maintenance tasks.

### Data Flow

1. `janet-bot` receives Slack message and reaction events through Socket Mode.
2. Karma transactions are written to PostgreSQL.
3. `janet-server` rebuilds summary tables on startup and serves analytics from those summaries.
4. `janet-backfill` can import older channel history and emoji reactions into the same database.
5. The popular-messages API lazily enriches records with Slack metadata and caches the result.

## Feature Summary

### Live Slack Karma

The bot still supports Janet-style karma commands:

- Upvote: `alice++`
- Downvote: `alice--`
- Multi-point vote: `alice++++`
- Reason text: `alice++ for fixing the deploy`
- Query points: `karma for alice`
- Throwback: `goodplace throwback alice`
- Leaderboard: `goodplace leaderboard`
- Larger leaderboard: `goodplace top 20`
- Year-specific leaderboard in chat: `goodplace highscores 20 2025`
- Motivate syntax: `alice motivate`

Slack mentions are also supported, for example:

- `<@U12345>++`
- `@alice ++`

Reaction-based karma is enabled as well. The bot treats configured positive reactions such as `:+1:`, `:thumbsup:`, `:joy:`, `:heart:`, and others as positive awards, and `:-1:` / `:thumbsdown:` as negative awards.

### Analytics Dashboard

The web app is a single-page dashboard with these sections:

- Overview: totals and high-level trend cards.
- Leaderboard: yearly or all-time ranking plus top givers.
- Popular: high-reaction messages with Slack enrichment and optional media filtering.
- Analytics: emoji usage, distribution, and time-based charts.
- Activity: paginated recent transaction feed with filtering.
- Users: per-user summaries and points-over-time views.

### Historical Import

The backfill command can:

- List accessible Slack channels.
- Import a specific time window from a channel.
- Optionally include emoji reactions.
- Run as a dry run before writing data.
- Limit the number of fetched messages for smaller test imports.

## Repository Layout

```text
cmd/janet-bot/          Live Slack bot
cmd/janet-backfill/     Historical Slack importer
cmd/janet-maintenance/  Database maintenance utility
database/               PostgreSQL access layer, migrations, summaries
janet-server/           Web server, handlers, templates, embedded assets
janet-server/web-react/ React source for the dashboard
docker-compose.yml      Local multi-service stack
docker-compose.production.yml  Production-style stack
```

## Requirements

- Go 1.25+
- Node 18+ if you want to rebuild the React app
- PostgreSQL 17+
- Slack bot token and app-level Socket Mode token
- Optional Slack user token for richer web enrichment and backfill behavior
- Docker and Docker Compose for containerized setup

## Environment Variables

Copy `.env.example` to `.env` and fill in the values you actually use.

### Required for Live Bot

- `JANET_SLACK_TOKEN`: bot token (`xoxb-...`)
- `JANET_SLACK_SOCKET_TOKEN`: app-level Socket Mode token (`xapp-...`)
- `JANET_GOOD_PLACE_JUDGE_BOT_ID`: Slack bot user ID
- `JANET_DATABASE_URL`: PostgreSQL connection string

### Required for Web Server

- `JANET_DATABASE_URL`
- `JANET_WEB_LISTEN_ADDR`
- `JANET_BOT_ENABLED=false` if running the server in web-only mode

### Recommended for Web Enrichment and Admin Flows

- `JANET_WEB_TOKEN`

`JANET_WEB_TOKEN` can be:

- A user token (`xoxp-...`) with `search:read`, `channels:history`, and `groups:history` for best results.
- A bot token (`xoxb-...`) with history scopes, if you only need enrichment where channel IDs are already known.

### Optional Bot Settings

- `JANET_MAX_POINTS`
- `JANET_LEADERBOARD_LIMIT`
- `JANET_REPLY_TYPE`
- `JANET_DEBUG`
- `JANET_SELF_KARMA`
- `JANET_MOTIVATE`
- `JANET_REACTJI_ENABLED`
- `JANET_GOOD_USERNAME`
- `JANET_GOOD_ICON_URL`
- `JANET_BAD_USERNAME`
- `JANET_BAD_ICON_URL`

### Optional Web Settings

- `JANET_ATTACHMENTS_DIR`
- `JANET_WEB_PUBLIC_URL`
- `JANET_WEB_RATE_LIMIT_RPS`
- `JANET_WEB_RATE_LIMIT_BURST`
- `JANET_RUN_CHANNELID_BACKFILL`

### OAuth2 Proxy

The Compose stacks front `janet-web` with `oauth2-proxy`. Configure:

- `OAUTH2_PROXY_CLIENT_ID`
- `OAUTH2_PROXY_CLIENT_SECRET`
- `OAUTH2_PROXY_COOKIE_SECRET`
- `OAUTH2_PROXY_REDIRECT_URL`
- `OAUTH2_PROXY_EMAIL_DOMAINS`

## Running with Docker Compose

### Local Stack

```bash
cp .env.example .env
docker compose up --build
```

This starts:

- `postgres`
- `janet-bot`
- `janet-web`
- `oauth2-proxy`
- `janet-backfill` as an interactive container entrypoint

The public entrypoint is `http://localhost:8080`, which hits `oauth2-proxy`, then forwards to `janet-web`.

### Running a Backfill in Docker

Start the stack, then run the backfill container with flags:

```bash
docker compose run --rm janet-backfill -list-channels
docker compose run --rm janet-backfill -channel C0123456789 -since 2025-01-01 -until 2025-12-31 -include-emoji
docker compose run --rm janet-backfill -channel C0123456789 -since 2025-01-01 -dry-run
```

### Production-Style Compose

```bash
docker compose -f docker-compose.production.yml up -d
```

That stack expects prebuilt images such as `troyxmccall/janet:bot` and `troyxmccall/janet:web`.

## Running Locally Without Docker

### 1. Start PostgreSQL

Create a database and apply [`database/migration.sql`](/Users/tdmccall/projects/janet/database/migration.sql).

### 2. Start the Live Bot

```bash
export JANET_DATABASE_URL='postgres://janet:password@localhost:5432/janet?sslmode=disable'
export JANET_SLACK_TOKEN='xoxb-...'
export JANET_SLACK_SOCKET_TOKEN='xapp-...'
export JANET_GOOD_PLACE_JUDGE_BOT_ID='U0123456789'
go run ./cmd/janet-bot
```

### 3. Start the Web Server

```bash
export JANET_DATABASE_URL='postgres://janet:password@localhost:5432/janet?sslmode=disable'
export JANET_WEB_LISTEN_ADDR=':8080'
export JANET_BOT_ENABLED='false'
export JANET_WEB_TOKEN='xoxp-...'
go run ./janet-server
```

### 4. Run a Backfill

```bash
export JANET_DATABASE_URL='postgres://janet:password@localhost:5432/janet?sslmode=disable'
export JANET_SLACK_TOKEN='xoxb-...'
go run ./cmd/janet-backfill -list-channels
go run ./cmd/janet-backfill -channel C0123456789 -since 2025-01-01 -until 2025-03-31 -include-emoji
```

### 5. Run the Maintenance Utility

```bash
export JANET_DATABASE_URL='postgres://janet:password@localhost:5432/janet?sslmode=disable'
go run ./cmd/janet-maintenance -fill-channel-ids
```

## Frontend Development

The deployed server embeds static assets from `janet-server/web/static`, and the Docker build regenerates those assets from the React app in `janet-server/web-react`.

If you are actively editing the dashboard UI:

```bash
cd janet-server/web-react
npm ci
npm run dev
```

To build the static bundle:

```bash
cd janet-server/web-react
npm ci
npm run build
```

## API Overview

All API routes are served from `/api`.

### Leaderboards

- `GET /api/leaderboard?limit=25`
- `GET /api/leaderboard/current`
- `GET /api/leaderboard/2025?limit=100`

Example:

```bash
curl 'http://localhost:8080/api/leaderboard/2025?limit=10'
```

### Summary Stats

- `GET /api/stats`
- `GET /api/stats?year=2025`
- `GET /api/stats/detailed?year=2025`
- `GET /api/stats/years`

Example:

```bash
curl 'http://localhost:8080/api/stats/detailed?year=2025'
```

### Top Givers and Emoji Stats

- `GET /api/stats/top-givers?limit=10`
- `GET /api/stats/top-givers/2025?limit=10`
- `GET /api/stats/emojis?year=2025&limit=15`
- `GET /api/stats/karma-distribution?year=2025`

### Time-Series Analytics

- `GET /api/stats/activity-timeline?year=2025`
- `GET /api/stats/points-over-time?year=2025`
- `GET /api/stats/recent-activity?limit=50&offset=0&from=alice&to=bob&year=2025`

Example:

```bash
curl 'http://localhost:8080/api/stats/recent-activity?limit=20&from=alice&year=2025'
```

### Popular Messages

- `GET /api/stats/popular-messages?year=2025`
- `GET /api/stats/popular-messages?year=2025&user=alice`
- `GET /api/stats/popular-messages?year=2025&min_reactions=5&has_media=1`
- `GET /api/stats/popular-messages?year=2025&funny_bias=1&include_meta=1`

Example:

```bash
curl 'http://localhost:8080/api/stats/popular-messages?year=2025&min_reactions=8&has_media=1'
```

### User Drill-Down

- `GET /api/user/alice`
- `GET /api/user/alice/2025`
- `GET /api/user/alice/2025/points-over-time`
- `GET /api/user/alice/points-over-time/all`

Example:

```bash
curl 'http://localhost:8080/api/user/alice/2025'
```

### Health

- `GET /api/status`

## Example Workflows

### Fresh Start for a New Workspace

1. Copy `.env.example` to `.env`.
2. Fill in Slack tokens and OAuth settings.
3. Run `docker compose up --build`.
4. Visit `http://localhost:8080`.
5. Run `docker compose run --rm janet-backfill -list-channels`.
6. Backfill the channels you care about.

### Import a Year of History for One Channel

```bash
go run ./cmd/janet-backfill \
  -channel C0123456789 \
  -since 2025-01-01 \
  -until 2025-12-31 \
  -include-emoji
```

### Test a Small Import Before a Full Backfill

```bash
go run ./cmd/janet-backfill \
  -channel C0123456789 \
  -since 2025-01-01 \
  -limit 100 \
  -dry-run
```

### Fetch Analytics Data for a Reporting Script

```bash
curl 'http://localhost:8080/api/leaderboard/2025?limit=25'
curl 'http://localhost:8080/api/stats/top-givers/2025?limit=10'
curl 'http://localhost:8080/api/stats/popular-messages?year=2025&min_reactions=10'
```

## Operational Notes

- `janet-server` rebuilds summary tables on startup to keep read-heavy endpoints fast.
- `janet-server` also attempts to reset database sequences on startup after imports.
- Popular-message enrichment is cached in `popular_message_cache` to reduce Slack API calls.
- Attachments can be cached to `JANET_ATTACHMENTS_DIR` and served from `/attachments/...`.
- API responses filter out bots and deleted Slack users by default unless `show_all=true` is used where supported.

## Known Branch Differences from `master`

- The old single `cmd/janet` binary is gone.
- The old `janetctl` CLI is gone.
- The old server-rendered web UI under `www/` has been replaced by `janet-server` plus a React SPA.
- PostgreSQL is now the primary storage model in the checked-in Compose setup.
