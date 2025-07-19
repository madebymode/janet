# Janet Server - Single Binary Application

This is the new single binary version of Janet (Good Place Judge bot) that includes:

- **Slack Bot functionality** - All original karma tracking features
- **Web Management UI** - Password-protected admin interface at `/admin/`
- **Public Status Interface** - Public leaderboard and stats at `/`
- **Embedded Assets** - All web files are embedded in the binary
- **Configuration Management** - Web-based configuration instead of CLI flags

## Features

### Public Interface

- **Home** (`/`) - Overview with stats and recent leaderboard
- **Leaderboard** (`/leaderboard`) - Full leaderboard with search
- **Statistics** (`/stats`) - Detailed statistics and activity

### Admin Interface (Password Protected)

- **Dashboard** (`/admin/`) - System status and quick actions
- **Configuration** (`/admin/config`) - Bot settings, Slack tokens, personalities
- **Backfill** (`/admin/backfill`) - Import historical karma data from Slack
- **User Management** (`/admin/users`) - Manage users, blacklist, aliases

### Bot Features

- All original Janet karma tracking functionality
- Support for `@user++` and `@user--` commands
- Reactji support (emoji reactions for karma)
- Leaderboards and user queries
- Good/Bad Janet personalities

## Quick Start with Docker

1. Create a `.env` file:

```bash
JANET_SLACK_TOKEN=xoxb-your-bot-token
JANET_SLACK_SOCKET_TOKEN=xapp-your-socket-token
JANET_GOOD_PLACE_JUDGE_BOT_ID=B1234567890
JANET_WEB_PASSWORD=your-admin-password
```

2. Run with Docker Compose:

```bash
docker-compose -f docker-compose.janet-server.yml up -d
```

3. Access the interfaces:

- Public: http://localhost:8080
- Admin: http://localhost:8080/admin (password required)

## Configuration

Configuration can be provided via:

1. **Environment variables** (recommended for containers)
2. **JSON config file** (pass as argument: `janet-server config.json`)
3. **Web UI** (after initial setup)

### Environment Variables

| Variable                        | Description                        | Default                 |
|---------------------------------|------------------------------------|-------------------------|
| `JANET_SLACK_TOKEN`             | Slack bot token (xoxb-...)         | Required                |
| `JANET_SLACK_SOCKET_TOKEN`      | Slack socket mode token (xapp-...) | Required                |
| `JANET_GOOD_PLACE_JUDGE_BOT_ID` | Bot ID for thread management       | Required                |
| `JANET_WEB_PASSWORD`            | Admin interface password           | `admin`                 |
| `JANET_WEB_LISTEN_ADDR`         | Web server listen address          | `:8080`                 |
| `JANET_WEB_PUBLIC_URL`          | Public URL for links               | `http://localhost:8080` |
| `JANET_DATABASE_PATH`           | SQLite database file path          | `./db.sqlite3`          |

### Bot Behavior Settings

These can be configured via the web UI:

- **Max Points** - Maximum points per transaction (default: 5)
- **Leaderboard Limit** - Default leaderboard size (default: 10)
- **Reply Type** - How bot responds (thread/message/ephemeral)
- **Self Karma** - Allow users to give themselves points
- **Motivate Support** - Support for motivate.im syntax
- **Reactji Support** - Emoji reaction karma

## Building

### Local Build

```bash
go build -mod=mod -o janet-server ./janet-server
```

### Docker Build

```bash
docker build -f janet-server/Dockerfile -t janet-server .
```

## Migration from Original Janet

The new single binary is designed to be a drop-in replacement:

1. **Database**: Uses the same SQLite schema
2. **Slack Integration**: Same bot functionality and commands
3. **Configuration**: Migrates from CLI flags to environment variables/web UI

To migrate:

1. Copy your existing `db.sqlite3` database file
2. Set environment variables based on your current CLI flags
3. Start the new janet-server
4. Use the web UI for ongoing configuration management

## Differences from Original

| Feature       | Original Janet          | Janet Server                   |
|---------------|-------------------------|--------------------------------|
| Configuration | CLI flags               | Environment variables + Web UI |
| Web Interface | Optional separate setup | Built-in embedded              |
| Backfilling   | Separate binary         | Integrated in admin UI         |
| Management    | Command line tools      | Web-based admin interface      |
| Assets        | External files          | Embedded in binary             |
| Deployment    | Multiple binaries       | Single binary                  |

## Security

- Admin interface requires password authentication
- Sessions expire after 24 hours
- Sensitive config values are masked in API responses
- No sensitive data logged in debug mode

## API Endpoints

### Public API

- `GET /api/leaderboard` - Get leaderboard data
- `GET /api/stats` - Get basic statistics
- `GET /api/user/{username}` - Get user information

### Admin API (Authentication Required)

- `GET/POST /admin/api/config` - Manage configuration
- `POST /admin/api/backfill` - Run backfill operations
- `GET /admin/api/users` - Manage users
- `GET /admin/api/channels` - List Slack channels

## Troubleshooting

### Bot Not Connecting

1. Verify `JANET_SLACK_TOKEN` and `JANET_SLACK_SOCKET_TOKEN`
2. Check bot permissions in Slack app settings
3. Ensure socket mode is enabled

### Web Interface Issues

1. Check `JANET_WEB_LISTEN_ADDR` setting
2. Verify firewall/port access
3. Check container port mapping

### Database Issues

1. Ensure SQLite file is writable
2. Check `JANET_DATABASE_PATH` setting
3. Verify volume mounts in Docker

For more help, check the logs or visit the admin dashboard for system status.
