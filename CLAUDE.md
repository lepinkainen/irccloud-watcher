# IRCCloud Watcher - AI Development Guide

## Project Overview
IRCCloud Watcher is a Go application that monitors IRC channels via IRCCloud's WebSocket API, stores messages in SQLite, and generates AI-powered daily summaries on a cron schedule. **Status: WORKING** - Authentication and WebSocket connection issues have been resolved.

## Architecture Components

### Core Components
- **`cmd/irccloud-watcher/main.go`** - Entry point with CLI flags, cron scheduling, and graceful shutdown
- **`internal/api/websocket.go`** - IRCCloud WebSocket client with working authentication and dynamic URL handling
- **`internal/api/websocket_test.go`** - Comprehensive unit tests for authentication and WebSocket functionality
- **`internal/storage/sqlite.go`** - SQLite database wrapper using modernc.org/sqlite (no cgo)
- **`internal/config/config.go`** - Viper-based YAML configuration loader with validation and environment variable support
- **`internal/summary/generator.go`** - Daily summary generator (placeholder for AI integration)

### Data Flow
1. Configuration loaded with validation (supports environment variables)
2. Two-step authentication: formtoken request → login with credentials
3. Dynamic WebSocket URL extracted from login response (e.g., wss://api-2.irccloud.com/websocket/2)
4. WebSocket connection established to load-balanced IRCCloud servers
5. Messages received, parsed, and stored in SQLite with full metadata
6. Cron job triggers daily summary generation from previous day's messages
7. Graceful shutdown handling for production deployments

## Development Conventions

### Build System
- **Always use `task build`** instead of `go build` - ensures tests pass and linting runs
- **Build artifacts go to `build/` directory** - follows project tech stack guidelines
- **Run `gofmt -w .` before builds** - automatically handled by Taskfile

### Key Libraries (Following Project Standards)
- **Configuration**: `github.com/spf13/viper` - YAML config management
- **Database**: `modernc.org/sqlite` + `github.com/jmoiron/sqlx` - No cgo dependency
- **WebSocket**: `github.com/gorilla/websocket` - IRCCloud API connection
- **Scheduling**: `github.com/robfig/cron/v3` - Daily summary generation

### Configuration
- Uses `config.yaml` (copy from `config.yaml.example`)
- Structure: email, password, channels list, database_path, summary_output_path, summary_time (cron format)
- **Environment Variable Support**: `IRCCLOUD_EMAIL` and `IRCCLOUD_PASSWORD` override config file
- **Validation**: Ensures all required fields are present with helpful error messages
- Config loaded via Viper with mapstructure tags

### Database Schema
Messages table: id, channel, timestamp, sender, message, date (YYYY-MM-DD format)

## Testing Requirements
- **Comprehensive test suite** covering authentication, WebSocket, and error handling
- **Unit tests**: Mock-based tests for all core functionality in `websocket_test.go`
- **Integration tests**: Real API tests in `websocket_integration_test.go` (skipped in CI)
- **Tests must pass before build completes** - enforced by Taskfile dependencies
- **Use `task test` for development, `task test-ci` for CI** with coverage
- Skip CI-specific tests with `//go:build !ci` tags

## Development Workflow
1. **Research phase**: Use `go run /path/to/llm-shared/utils/gofuncs/gofuncs.go` to explore functions
2. **Make changes**: Follow existing patterns in `internal/` packages
3. **Test**: `task test` runs all tests
4. **Build**: `task build` runs tests + lint + build
5. **No commits without explicit request** - preserve user control

## Key Patterns
- **Error handling**: Standard Go patterns with wrapped errors
- **Logging**: Standard library `log` package for simplicity
- **CLI flags**: Standard library `flag` package (not Kong for this simple case)
- **Package structure**: `cmd/` for binaries, `internal/` for private code
- **Database access**: Repository pattern with `*storage.DB` wrapper

## Current Status ✅
- **Authentication**: ✅ WORKING - Fixed missing headers (`Content-Length: 0`, `x-auth-formtoken`)
- **WebSocket Connection**: ✅ WORKING - Now uses dynamic URLs from login response (e.g., `wss://api-2.irccloud.com/websocket/2`)
- **Message Processing**: ✅ WORKING - Enhanced IRC message parsing with complete field support
- **Error Handling**: ✅ ROBUST - Comprehensive error parsing and debug logging
- **Testing**: ✅ COMPLETE - Full test coverage including integration tests

## Security Configuration
- **Environment Variables**: Use `IRCCLOUD_EMAIL` and `IRCCLOUD_PASSWORD` environment variables
- **Config Format**: `config.yaml` uses `${IRCCLOUD_EMAIL}` syntax for variable substitution
- **No Hardcoded Secrets**: All sensitive data loaded from environment
- **Debug Logging**: `IRCCLOUD_DEBUG=true` enables request/response logging (excludes sensitive headers)

## Extension Points
- **Summary generation**: Currently placeholder - ready for LLM integration
- **Message filtering**: Add logic in WebSocket handler for channel-specific rules
- **Export formats**: Extend summary generator for multiple output formats
- **Reconnection logic**: Add automatic reconnection on WebSocket failures
- **Performance monitoring**: Add metrics for connection health and message throughput

## Usage
```bash
# Build the application
task build

# Run with default config
./build/irccloud-watcher

# Run with custom config
./build/irccloud-watcher -config /path/to/config.yaml

# Generate summary only
./build/irccloud-watcher -generate-summary

# Run with debug logging
IRCCLOUD_DEBUG=true ./build/irccloud-watcher

# Test with real credentials (requires environment variables)
IRCCLOUD_EMAIL=your@email.com IRCCLOUD_PASSWORD=yourpass ./build/irccloud-watcher
```

## Configuration Management
- Copy `config.yaml.example` to `config.yaml` and customize
- Set `IRCCLOUD_EMAIL` and `IRCCLOUD_PASSWORD` environment variables
- Ensure channels, database_path, and summary paths are configured

## Memories

### Project Safety
- **DO NOT TOUCH config.yaml** - critical configuration file that must not be modified directly