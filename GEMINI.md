# IRCCloud Watcher - AI Agent Guide

This document outlines essential knowledge for AI coding agents working on the `irccloud-watcher` Go project.

## 1. Project Overview
A Go application monitoring IRC channels via IRCCloud's WebSocket API, storing messages in SQLite, and generating AI-powered daily summaries.

## 2. Architecture Overview
- **Entry Point**: `cmd/irccloud-watcher/main.go` (CLI, cron, graceful shutdown).
- **WebSocket Client**: `internal/api/websocket.go` (IRCCloud API, authentication, dynamic URLs).
- **Data Storage**: `internal/storage/sqlite.go` (SQLite, `modernc.org/sqlite` + `sqlx`).
- **Configuration**: `internal/config/config.go` (Viper-based YAML, env vars).
- **Summary Generation**: `internal/summary/generator.go` (placeholder for AI).
- **Data Flow**: Config -> Auth (formtoken, login) -> Dynamic WS URL -> WS Connect -> Message Parse & Store (SQLite) -> Cron Summary.

## 3. Critical Developer Workflows
- **Build**: `task build` (runs tests, lint, then builds to `build/irccloud-watcher`).
- **Test**: `task test` (runs all Go tests). For CI, use `task test-ci` (with coverage).
- **Lint**: `task lint` (runs `go fmt`, `go vet`, and `golangci-lint`).
- **Clean**: `task clean` (removes `build/` directory).
- **Run**: `./build/irccloud-watcher` (after building).
- **Debug**: `IRCCLOUD_DEBUG=true ./build/irccloud-watcher` for detailed logging.

## 4. Project-Specific Conventions & Patterns
- **Build System**: Always use `task` commands. `gofmt -w .` is integrated into `build-go`.
- **Configuration**: Uses `config.yaml` (from `config.yaml.example`). Sensitive data via `IRCCLOUD_EMAIL`, `IRCCLOUD_PASSWORD` environment variables.
- **Database**: `internal/storage/sqlite.go` manages `messages` table (`id`, `channel`, `timestamp`, `sender`, `message`, `date`).
- **Error Handling**: Standard Go patterns with wrapped errors.
- **Package Structure**: `cmd/` for executables, `internal/` for private application code.

## 5. Integration Points & Dependencies
- **IRCCloud API**: WebSocket (`wss://api-2.irccloud.com/websocket/2`) for real-time data.
- **Key Libraries**:
    - `github.com/spf13/viper` for configuration.
    - `modernc.org/sqlite` + `github.com/jmoiron/sqlx` for database.
    - `github.com/gorilla/websocket` for WebSocket client.
    - `github.com/robfig/cron/v3` for scheduling.

## 6. Testing Strategy
- **Unit Tests**: Mock-based, e.g., `internal/api/websocket_test.go`.
- **Integration Tests**: Real API tests, e.g., `internal/api/websocket_integration_test.go` (skipped in CI).
- **Enforcement**: Tests must pass before `task build` completes.

## 7. Security Configuration
- **Secrets**: Loaded from environment variables (`IRCCLOUD_EMAIL`, `IRCCLOUD_PASSWORD`) or `config.yaml` with variable substitution.
- **No Hardcoded Secrets**.

## 8. Extension Points
- **Summary Generation**: `internal/summary/generator.go` is a placeholder for LLM integration.
- **Message Filtering**: Extend WebSocket handler.
- **Reconnection Logic**: Add to WebSocket client.
