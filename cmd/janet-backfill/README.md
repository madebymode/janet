# Janet Backfill CLI

A command line tool for backfilling karma transactions from Slack message history. This provides an alternative to the
web-based backfill interface.

## Docker Setup (Recommended)

### Quick Start

1. **Copy environment template (from project root):**
   ```bash
   cp .env.backfill.example .env.backfill
   ```

2. **Edit `.env.backfill` file with your Slack token:**
   ```bash
   JANET_SLACK_TOKEN=xoxb-your-actual-slack-bot-token
   ```

3. **Start the services:**
   ```bash
   docker-compose -f docker-compose.backfill.yml --env-file .env.backfill up -d
   ```

4. **Run backfill commands:**
   ```bash
   # List available channels
   docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill -list-channels
   
   # Run a backfill
   docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill -channel C1234567890 -dry-run
   ```

5. **Stop services:**
   ```bash
   docker-compose -f docker-compose.backfill.yml down
   ```

### Docker Commands

```bash
# Build and start services
docker-compose -f docker-compose.backfill.yml --env-file .env.backfill up -d

# View logs
docker-compose -f docker-compose.backfill.yml logs -f janet-backfill

# Execute backfill commands
docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill [options]

# Stop services
docker-compose -f docker-compose.backfill.yml down

# Clean up (removes volumes)
docker-compose -f docker-compose.backfill.yml down -v
```

### Docker Examples

```bash
# List all available Slack channels
docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill -list-channels

# Dry run backfill for a specific channel
docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill -channel C1234567890 -dry-run

# Backfill last 7 days with emoji reactions
docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill \
  -channel C1234567890 \
  -since "2024-07-14" \
  -include-emoji

# Backfill with custom limits and debug logging
docker-compose -f docker-compose.backfill.yml exec janet-backfill ./janet-backfill \
  -channel C1234567890 \
  -limit 500 \
  -max-points 3 \
  -debug

# One-time backfill run (modify docker-compose.backfill.yml command)
# Change: command: ["-channel", "C1234567890", "-since", "2024-07-01"]
docker-compose -f docker-compose.backfill.yml --env-file .env.backfill up --abort-on-container-exit
```

## Local Development Setup

For local development without Docker:

### Prerequisites

- Go 1.21+
- PostgreSQL database
- Slack bot token

### Build and Run

```bash
# From the project root
go build ./cmd/janet-backfill

# Set environment variables
export JANET_SLACK_TOKEN=xoxb-your-token
export JANET_DATABASE_URL=postgres://user:pass@localhost:5432/janet?sslmode=disable

# Run the tool
./janet-backfill -list-channels
```

## Usage

```bash
# Build the tool
go build ./cmd/janet-backfill

# List available channels
./janet-backfill -list-channels

# Basic backfill for a channel
./janet-backfill -channel C1234567890

# Backfill with date range
./janet-backfill -channel C1234567890 -since "2024-01-01" -until "2024-01-31"

# Dry run to see what would be backfilled
./janet-backfill -channel C1234567890 -dry-run

# Include emoji reactions in backfill
./janet-backfill -channel C1234567890 -include-emoji

# Limit number of messages processed
./janet-backfill -channel C1234567890 -limit 500
```

## Options

- `-channel`: Slack channel ID to backfill (required)
- `-since`: Start timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)
- `-until`: End timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)
- `-dry-run`: Show what would be backfilled without making changes
- `-include-emoji`: Include emoji reactions in backfill
- `-limit`: Maximum number of messages to process (default: 1000)
- `-max-points`: Maximum points per karma transaction (default: 5)
- `-window`: Duplicate detection window in seconds (default: 10)
- `-list-channels`: List all available channels and exit
- `-debug`: Enable debug logging

## Environment Variables

The tool uses the same environment variables as the main Janet bot:

- `JANET_DATABASE_DRIVER`: Database driver (default: "postgres")
- `JANET_DATABASE_URL`: Database connection URL
- `JANET_SLACK_TOKEN`: Slack bot token (required)
- `JANET_DEBUG`: Enable debug mode ("true"/"false")

## Examples

### List all channels

```bash
./janet-backfill -list-channels
```

### Backfill last 30 days of a channel

```bash
./janet-backfill -channel C1234567890 -since "2024-06-01" -until "2024-06-30"
```

### Dry run with emoji reactions

```bash
./janet-backfill -channel C1234567890 -include-emoji -dry-run
```

### Backfill with custom limits

```bash
./janet-backfill -channel C1234567890 -limit 2000 -max-points 3
```

## Notes

- The tool respects Slack API rate limits with built-in delays
- Duplicate detection prevents processing the same karma twice
- User cache is preloaded to minimize API calls
- All operations are logged for visibility
- Supports both public and private channels (with appropriate permissions)
