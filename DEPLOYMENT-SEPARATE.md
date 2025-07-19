# Janet Separate Services Deployment

This document describes how to deploy Janet with **separate services** for the Slack bot and web UI, which is the recommended production deployment method.

## 🏗️ Architecture Overview

### Separate Services Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   janet-bot     │    │   janet-web     │    │   PostgreSQL    │
│                 │    │                 │    │                 │
│ • Slack Bot     │    │ • Web UI        │    │ • Database      │
│ • Socket Mode   │    │ • Admin Panel   │    │ • Statistics    │
│ • Event Proc.   │    │ • API Server    │    │ • Transactions  │
│ • Karma Logic   │    │ • Auth System   │    │ • User Data     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
                   ┌─────────────────┐
                   │    Network      │
                   │   (Docker)      │
                   └─────────────────┘
```

### Benefits of Separate Services

✅ **Independent Scaling**: Scale bot and web UI separately based on load  
✅ **Better Reliability**: If one service fails, the other continues working  
✅ **Easier Maintenance**: Update/restart services independently  
✅ **Resource Optimization**: Allocate resources based on specific service needs  
✅ **Development Flexibility**: Different teams can work on different services  

## 🚀 Quick Start

### 1. Setup Environment

Copy the environment configuration:
```bash
cp .env.separate.example .env
```

Edit `.env` with your Slack tokens:
```bash
# Required Slack Configuration
JANET_SLACK_TOKEN=xoxb-your-bot-token-here
JANET_SLACK_SOCKET_TOKEN=xapp-your-socket-token-here
JANET_GOOD_PLACE_JUDGE_BOT_ID=your-bot-user-id-here

# Web UI Password
JANET_WEB_PASSWORD=your-secure-admin-password
```

### 2. Start Services

Start all services with Docker Compose:
```bash
docker-compose up -d
```

> **Note**: If you see warnings about missing environment variables (JANET_SLACK_TOKEN, etc.), that's normal until you create your .env file with the actual values.

### 3. Verify Deployment

Check service status:
```bash
docker-compose ps
```

Access the web UI at: http://localhost:8080

## 📋 Services Overview

### **janet-bot** - Dedicated Slack Bot Service
- **Purpose**: Handles all Slack interactions and karma operations
- **Features**:
  - Socket Mode Slack integration
  - Real-time event processing
  - Karma calculations and storage
  - User interaction handling
- **Container**: `janet-bot` (Alpine Linux)
- **Health Check**: Process monitoring
- **Restart Policy**: `unless-stopped`

### **janet-web** - Web UI Service
- **Purpose**: Provides web interface and API endpoints
- **Features**:
  - React-based admin dashboard
  - Statistics and charts
  - User management
  - API for external integrations
- **Container**: `janet-web` (based on janet-server with bot disabled)
- **Ports**: `8080:8080`
- **Health Check**: HTTP endpoint monitoring

### **postgres** - PostgreSQL Database
- **Purpose**: Stores all karma data, transactions, and statistics
- **Features**:
  - PostgreSQL 15 (Alpine)
  - Automatic schema migration
  - Persistent data storage
- **Container**: `janet-postgres`
- **Ports**: `5432:5432` (for development)
- **Volumes**: `postgres_data`

### **migrate** - Database Migration
- **Purpose**: One-time setup of database schema and SQLite migration
- **Container**: `janet-migrate`
- **Restart Policy**: `no` (runs once)

## 🔧 Configuration

### Environment Variables

#### Bot Service (`janet-bot`)
```bash
# Database
JANET_DATABASE_DRIVER=postgres
JANET_DATABASE_URL=postgres://janet:password@postgres:5432/janet?sslmode=disable

# Slack
JANET_SLACK_TOKEN=xoxb-...
JANET_SLACK_SOCKET_TOKEN=xapp-...
JANET_GOOD_PLACE_JUDGE_BOT_ID=U...

# Bot Behavior
JANET_MAX_POINTS=5
JANET_LEADERBOARD_LIMIT=10
JANET_REPLY_TYPE=message
JANET_DEBUG=false
JANET_SELF_KARMA=false
JANET_MOTIVATE=true
JANET_REACTJI_ENABLED=true
```

#### Web Service (`janet-web`)
```bash
# Database
JANET_DATABASE_DRIVER=postgres
JANET_DATABASE_URL=postgres://janet:password@postgres:5432/janet?sslmode=disable

# Web Server
JANET_WEB_LISTEN_ADDR=0.0.0.0:8080
JANET_WEB_PASSWORD=your-secure-password
JANET_WEB_PUBLIC_URL=http://localhost:8080

# Bot Control
JANET_BOT_ENABLED=false  # Disables bot in web service

# Slack (needed for API calls)
JANET_SLACK_TOKEN=xoxb-...  # For API integration only
```

## 🐳 Docker Configuration

### Building Images Locally

Build the bot service:
```bash
docker build -f cmd/janet-bot/Dockerfile -t janet-bot:local .
```

Build the web service:
```bash
docker build -f janet-server/Dockerfile -t janet-web:local .
```

### Using Pre-built Images

The services use pre-built images from Docker Hub:
- `troyxmccall/janet-bot:latest`
- `troyxmccall/janet:latest`

## 📊 Monitoring and Health Checks

### Service Health Checks

**Bot Service**:
```bash
docker exec janet-bot pgrep janet-bot
```

**Web Service**:
```bash
curl -f http://localhost:8080/api/stats
```

**Database**:
```bash
docker exec janet-postgres pg_isready -U janet -d janet
```

### Logs

View service logs:
```bash
# Bot logs
docker-compose logs -f janet-bot

# Web logs  
docker-compose logs -f janet-web

# Database logs
docker-compose logs -f postgres
```

## 🔄 Operations

### Restart Services

Restart individual services:
```bash
# Restart bot only
docker-compose restart janet-bot

# Restart web UI only
docker-compose restart janet-web

# Restart all services
docker-compose restart
```

### Update Services

Update to latest versions:
```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d
```

### Backup Database

Backup PostgreSQL data:
```bash
docker exec janet-postgres pg_dump -U janet janet > janet_backup_$(date +%Y%m%d).sql
```

Restore from backup:
```bash
cat janet_backup_20240101.sql | docker exec -i janet-postgres psql -U janet janet
```

## 🚨 Troubleshooting

### Common Issues

**Bot not responding to Slack messages:**
1. Check bot container is running: `docker ps | grep janet-bot`
2. Verify Slack tokens in environment
3. Check bot logs: `docker logs janet-bot`
4. Ensure Socket Mode is enabled in Slack app

**Web UI not accessible:**
1. Check web container is running: `docker ps | grep janet-web`
2. Verify port 8080 is not in use: `lsof -i :8080`
3. Check web logs: `docker logs janet-web`

**Database connection errors:**
1. Check postgres container: `docker ps | grep janet-postgres`
2. Verify database URL in environment
3. Test connection: `docker exec janet-postgres psql -U janet janet`

### Debug Mode

Enable debug logging for bot:
```bash
docker-compose stop janet-bot
docker-compose run --rm -e JANET_DEBUG=true janet-bot
```

## 🔒 Security Considerations

1. **Use strong passwords**: Set `JANET_WEB_PASSWORD` to a secure value
2. **Network isolation**: Services communicate through Docker network
3. **Database access**: PostgreSQL only accessible within Docker network
4. **Slack tokens**: Store securely in `.env` file with restricted permissions
5. **Regular updates**: Keep Docker images updated for security patches

## 📈 Scaling

### Horizontal Scaling

**Bot Service**: Can run multiple instances with shared database
```yaml
janet-bot:
  # ... existing config
  deploy:
    replicas: 3
```

**Web Service**: Can run multiple instances behind load balancer
```yaml
janet-web:
  # ... existing config
  ports:
    - "8080-8082:8080"  # Multiple ports
```

### Resource Limits

Set resource limits in docker-compose:
```yaml
janet-bot:
  # ... existing config
  deploy:
    resources:
      limits:
        memory: 256M
        cpus: '0.5'
      reservations:
        memory: 128M
        cpus: '0.25'
```

## 📋 Maintenance Checklist

### Daily
- [ ] Check service health status
- [ ] Monitor log files for errors
- [ ] Verify bot responsiveness in Slack

### Weekly  
- [ ] Review resource usage
- [ ] Check disk space usage
- [ ] Update Docker images if needed

### Monthly
- [ ] Backup database
- [ ] Review and rotate logs
- [ ] Security updates
- [ ] Performance optimization review

This separate services architecture provides a robust, scalable solution for production Janet deployments while maintaining the simplicity of Docker Compose management.