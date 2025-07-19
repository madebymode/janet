# Janet Architecture Overview

## 🏗️ Modern Unified Architecture

Janet now uses a **single-container architecture** that eliminates the dual-bot problem and provides a cohesive experience.

### Current Architecture (Recommended)

```
┌─────────────────────────────────────────┐
│             janet-server                │
│  ┌─────────────────┬─────────────────┐  │
│  │   Slack Bot     │    Web UI       │  │
│  │                 │                 │  │
│  │ • Socket Mode   │ • Admin Panel   │  │
│  │ • Karma Logic   │ • Leaderboards  │  │
│  │ • Event Proc.   │ • Statistics    │  │
│  │ • Services      │ • Auth System   │  │
│  └─────────────────┴─────────────────┘  │
│           │                             │
│           ▼                             │
│  ┌─────────────────────────────────────┐│
│  │        PostgreSQL V2 DB            ││
│  │                                     ││
│  │ • Transactions  • User Summaries   ││
│  │ • Emoji Stats   • Monthly Reports  ││
│  │ • Leaderboards  • Audit Trails     ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

### Key Components

#### 🤖 **janet-server** (`janet-server/main.go`)
- **Single unified service** running bot + web UI
- **Socket Mode Slack integration** for real-time events
- **Embedded web assets** (no external file dependencies)
- **JSON configuration** with environment variable support
- **Authentication system** for admin features
- **Health checks** and proper container lifecycle

#### 🗄️ **PostgreSQL V2 Database** 
- **Partitioned by year** for performance
- **Rich analytics** with summary tables
- **Emoji statistics** and leaderboards  
- **Monthly aggregations** for reporting

#### 🛠️ **Service Layer**
- **TransactionService**: Handles karma operations
- **UserService**: User validation and parsing
- **KarmaService**: Business logic coordination
- **ErrorHandler**: Centralized error management

#### 🔧 **Supporting Tools**
- **Migration service**: One-time data migration
- **janetctl**: Administrative commands
- **Docker Compose**: Complete environment setup

## 📜 Legacy Architecture (Deprecated)

### ❌ Previous Problematic Setup
```
┌─────────────┐    ┌─────────────┐
│ cmd/janet   │    │janet-server │
│             │    │             │
│ • Bot only  │    │ • Bot + UI  │  ← CONFLICT!
│ • PostgreSQL│
│ • CLI flags │    │ • JSON cfg  │
└─────────────┘    └─────────────┘
       │                   │
       └─────┬─────────────┘
             ▼
    ⚠️ DUPLICATE SLACK BOTS ⚠️
    • Race conditions
    • Duplicate processing  
    • Configuration conflicts
```

### 🗑️ What Was Removed/Deprecated

1. **`cmd/janet/main.go`** → `cmd/janet-legacy/main.go`
   - Old CLI-based bot launcher
   - SQLite database support
   - Separate web UI process

2. **`database/database.go`** → ❌ **Deleted**
   - Old SQLite implementation
   - Legacy Points/User structs
   - Synchronous operations

3. **Legacy Docker service** → ❌ **Removed**
   - Separate janet container in compose
   - SQLite volume mounts  
   - Manual coordination

## 🚀 Migration Benefits

### ✅ **Solved Problems**
- **No more dual bots** - Single service, single Slack connection
- **Simplified deployment** - One container with everything
- **Better performance** - PostgreSQL with optimized queries
- **Easier maintenance** - Unified configuration and logging
- **Rich web interface** - Embedded assets, no external dependencies

### ✅ **Improved Architecture**
- **Service-oriented design** with clear separation of concerns
- **Type-safe interfaces** throughout the application
- **DRY principles** - No duplicate business logic
- **Testable components** - Services can be mocked easily
- **Container-first** - Designed for modern deployment

## 🐳 Deployment

### Recommended: Docker Compose
```bash
# Single command to start everything
docker-compose up

# Services started:
# - PostgreSQL database
# - Janet-server (bot + web UI)  
# - Migration (one-time SQLite → PostgreSQL)
```

### Alternative: Direct Build
```bash
cd janet-server
go build && ./janet-server
```

## 🔧 Development

The new architecture makes development easier:

- **Single entry point**: `janet-server/main.go`
- **Service layer**: Easy to test individual components
- **Embedded assets**: No need to manage separate web files
- **Environment config**: Docker-friendly configuration
- **Database migrations**: Automatic on startup

This unified architecture provides a much cleaner, more maintainable, and more reliable Janet deployment.