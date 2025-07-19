# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Warren is an AI agent and Slack-based security alert management tool. It processes security alerts, analyzes them using LLM (Gemini), and manages incident response through Slack integration.

## Common Development Commands

### Building and Testing
- `go build` - Build the main binary
- `go test ./...` - Run all tests
- `go test ./pkg/path/to/package` - Run tests for specific package
- `task` - Run default tasks (mock generation and GraphQL)
- `task mock` (alias: `task m`) - Generate all mock files
- `task graphql` - Generate GraphQL code from schema

### Code Generation
- `go tool moq` - Generate mocks (handled by task commands)
- `go tool gqlgen generate` - Generate GraphQL resolvers and types

## Important Development Guidelines

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Must wrap errors with `goerr.Wrap` to maintain error context
- Add helpful variables with `goerr.V` for debugging

### Testing with gt Package
- Use `github.com/m-mizutani/gt` package for type-safe testing
- Prefer Helper Driven Testing style over Table Driven Tests
- Use Memory repository from `pkg/repository` instead of mocks for repository testing
- Use mock implementations from `pkg/domain/mock`

### Code Visibility
- Do not expose unnecessary methods, variables and types
- Use `export_test.go` to expose items needed only for testing

## Architecture

### Core Structure
The application follows Domain-Driven Design (DDD) with clean architecture:

- `pkg/domain/` - Domain layer with business logic, interfaces, and models
- `pkg/service/` - Application services implementing business operations
- `pkg/controller/` - Interface adapters (HTTP, GraphQL, Slack)
- `pkg/adapter/` - Infrastructure adapters (storage, external APIs)
- `pkg/repository/` - Data persistence implementations
- `pkg/usecase/` - Application use cases orchestrating domain operations

### Key Components

#### Alert Processing
- `pkg/domain/model/alert/` - Core alert model with metadata and embedding support
- Alerts are immutable and can be linked to at most one ticket
- Uses AI to generate titles, descriptions, and semantic embeddings

#### Command System
- `pkg/service/command/` - Slack command processing (list, aggregate, ticket)
- Commands: `l`/`ls`/`list`, `a`/`aggr`/`aggregate`, `t`/`ticket`

#### LLM Integration
- Uses Vertex AI Gemini for alert analysis and metadata generation
- `pkg/service/llm/` - LLM service abstractions
- Implements gollem.LLMClient interface for AI operations

#### Storage
- Firestore for persistence in serve mode
- In-memory storage for testing/development
- `pkg/repository/` - Repository pattern implementations

### Application Modes
- `serve` - HTTP server mode with Slack integration, GraphQL API
- `run` - CLI mode for processing individual alerts
- `test` - Testing utilities
- `chat` - Interactive chat mode
- `tool` - Tool execution utilities

### Key Interfaces
- `interfaces.Repository` - Data persistence abstraction
- `interfaces.LLMClient` - AI/LLM client abstraction
- `interfaces.SlackClient` - Slack API client abstraction
- `interfaces.PolicyClient` - Policy evaluation using OPA
- `interfaces.StorageClient` - Cloud storage abstraction

### Tools Integration
External security tools integrated via `pkg/tool/`:
- BigQuery for data analysis
- VirusTotal, OTX, URLScan for threat intelligence
- AbuseChip, Shodan, IPDB for IP/domain analysis

## Configuration

The application is configured via CLI flags or environment variables. Key configurations include:
- Gemini/Vertex AI settings (project ID, location, model)
- Firestore database settings
- Slack integration (OAuth token, signing secret, channel)
- External tool API keys (OTX, URLScan, etc.)

## Testing

Test files follow Go conventions (`*_test.go`). The codebase includes:
- Unit tests for individual components
- Integration tests with mock dependencies
- Test data in `testdata/` directories
- Mock generation using `moq` tool

## Restrictions and Rules

- If you need to create a temporary file as mid-process, please create it in the `tmp` directory. No need to delete the temporary files.
- When you finish to edit source code, please run `go mod tidy` and `go fmt ./...` to update the dependencies and format the code. If you have any error, please fix it before complete the task.
