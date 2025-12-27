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

### Frontend Development
- `cd frontend && pnpm install` - Install frontend dependencies
- `pnpm run dev` - Start development server
- `pnpm run build` - Build frontend for production
- `pnpm run codegen` - Generate GraphQL types from schema

### Code Generation
- `go tool moq` - Generate mocks (handled by task commands)
- `go tool gqlgen generate` - Generate GraphQL resolvers and types

## Important Development Guidelines

### Implementation Completeness
- **NEVER leave incomplete implementations, TODOs, or placeholder code**
- **NEVER skip implementation because it's complex or lengthy**
- **ALWAYS complete the full implementation in one go**
- If a task seems too complex, break it down into smaller steps, but complete ALL steps
- Complexity is not an excuse - implement everything thoroughly
- Long code is acceptable - incomplete code is NOT

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Must wrap errors with `goerr.Wrap` to maintain error context
- Add helpful variables with `goerr.V` for debugging
- **NEVER check error messages using `strings.Contains(err.Error(), ...)`**
- **ALWAYS use `errors.Is(err, targetErr)` or `errors.As(err, &target)` for error type checking**
- Error discrimination must be done by error types, not by parsing error messages

### Testing with gt Package
- Use `github.com/m-mizutani/gt` package for type-safe testing
- Prefer Helper Driven Testing style over Table Driven Tests
- Use Memory repository from `pkg/repository` instead of mocks for repository testing
- Use mock implementations from `pkg/domain/mock`
- **NEVER comment out test assertions** - if a test doesn't work, fix it or delete it
- **NEVER use length-only checks** - always verify individual IDs/values explicitly
  - BAD: `gt.A(t, toDelete).Length(3)` with commented out ID checks
  - GOOD: Check each expected ID explicitly with `gt.True(t, deleteMap[id])`

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

#### Alert Processing Pipeline
- `pkg/domain/model/alert/` - Core alert model with metadata and embedding support
- `pkg/usecase/alert_pipeline.go` - Main alert processing pipeline
- Alerts are immutable and can be linked to at most one ticket
- Uses AI to generate titles, descriptions, and semantic embeddings

**Pipeline Stages**:
1. **Ingest Policy Evaluation** - Transform raw alert data into Alert objects
2. **Metadata Generation** - Fill missing titles/descriptions using LLM
3. **Enrich Policy Evaluation** - Execute enrichment tasks (query/agent)
4. **Triage Policy Evaluation** - Apply final metadata and determine publish type

**Pipeline Execution**:
- `ProcessAlertPipeline()` - Pure pipeline processing (no side effects)
- `HandleAlert()` - Complete alert handling including DB save and Slack posting
- All pipeline events are emitted through `Notifier` interface for real-time monitoring

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

#### Alert Clustering
- `pkg/domain/service/clustering/` - DBSCAN clustering algorithm implementation
- `pkg/usecase/clustering.go` - Clustering use case with caching
- Uses cosine distance on alert embeddings for similarity
- Configurable DBSCAN parameters (eps, minSamples)
- WebUI at `/clusters` for visualizing and managing alert clusters
- Supports creating tickets from clusters and binding clusters to existing tickets

#### Agent Memory Scoring (Experimental)
- `pkg/service/memory/scoring.go` - Quality-based memory ranking and pruning
- `pkg/service/memory/feedback.go` - LLM-based feedback collection
- `pkg/domain/model/memory/feedback.go` - Feedback model (Relevance/Support/Impact)

**Features**:
- **Quality Scoring**: Memories rated from -10 (harmful) to +10 (helpful)
- **Adaptive Search**: Re-ranks memories by similarity (50%) + quality (30%) + recency (20%)
- **LLM Feedback**: Automatically evaluates memory usefulness after each agent execution
- **EMA Updates**: Smooth score evolution using exponential moving average (alpha=0.3)
- **Conservative Pruning**: Strict deletion criteria to preserve memories
  - Critical (≤-8.0): Immediate deletion
  - Harmful (≤-5.0) + 90 days unused: Deletion
  - Moderate (≤-3.0) + 180 days unused: Deletion

**Configuration** (all parameters in `ScoringConfig`):
- Fully configurable weights, thresholds, and decay rates
- Default values optimized for gradual learning
- Easy to remove if experimental feature proves ineffective

**Implementation**:
- Isolated in 3 files: `scoring.go`, `feedback.go`, `prompt/feedback.md`
- Minimal changes to existing code (re-ranking in `SearchRelevantAgentMemories`)
- Backward compatible (existing memories default to score=0.0)

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
- `interfaces.Notifier` - Event notification abstraction for alert pipeline events
- `clustering.Service` - Alert clustering service interface

#### Event Notification System
The alert processing pipeline uses an event-driven notification pattern:

- **Notifier Interface** (`pkg/domain/interfaces/notifier.go`):
  - Type-safe event handling with dedicated methods for each event type
  - Methods: `NotifyAlertPolicyResult`, `NotifyEnrichPolicyResult`, `NotifyCommitPolicyResult`, `NotifyEnrichTaskPrompt`, `NotifyEnrichTaskResponse`, `NotifyError`
  - No generic `Notify(event)` method - each event type has its own method signature

- **Event Types** (`pkg/domain/event/`):
  - `AlertPolicyResultEvent` - Alert policy evaluation results
  - `EnrichPolicyResultEvent` - Enrichment policy evaluation results
  - `CommitPolicyResultEvent` - Commit policy evaluation results
  - `EnrichTaskPromptEvent` - LLM task prompt being sent
  - `EnrichTaskResponseEvent` - LLM task response received
  - `ErrorEvent` - Error occurred during pipeline processing

- **Notifier Implementations**:
  - `ConsoleNotifier` (`pkg/service/notifier/console.go`) - Outputs events to console with color formatting
  - `SlackNotifier` (`pkg/service/notifier/slack.go`) - Posts events to Slack thread with formatted messages
  - Both implementations provide real-time visibility into alert pipeline processing

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

### Directory

- When you are mentioned about `tmp` directory, you SHOULD NOT see `/tmp`. You need to check `./tmp` directory from root of the repository.

### Exposure policy

In principle, do not trust developers who use this library from outside

- Do not export unnecessary methods, structs, and variables
- Assume that exposed items will be changed. Never expose fields that would be problematic if changed
- Use `export_test.go` for items that need to be exposed for testing purposes

### Check

When making changes, before finishing the task, always:
- Run `go vet ./...`, `go fmt ./...` to format the code
- Run `golangci-lint run ./...` to check lint error
- Run `gosec -exclude-generated -quiet ./...` to check security issue
- Run tests to ensure no impact on other code

### Language

All comment and character literal in source code must be in English

### Testing

- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` → `xyz_test.go`. Other test file names (e.g., `xyz_e2e_test.go`) are not allowed.
- Repository Tests Location:
  - NEVER create test files in `pkg/repository/firestore/` or `pkg/repository/memory/` subdirectories
  - ALL repository tests MUST be placed directly in `pkg/repository/*_test.go`
  - Use `runRepositoryTest()` helper to test against both memory and firestore implementations
- Repository Tests Best Practices:
  - Always use random IDs (e.g., using `time.Now().UnixNano()`) to avoid test conflicts
  - Never use hardcoded IDs like "msg-001", "user-001" as they cause test failures when running in parallel
  - Always verify ALL fields of returned values, not just checking for nil/existence
  - Compare expected values properly - don't just check if something exists, verify it matches what was saved
  - For timestamp comparisons, use tolerance (e.g., `< time.Second`) to account for storage precision
- Test Skip Policy:
  - **NEVER use `t.Skip()` for anything other than missing environment variables**
  - If a test requires infrastructure (like Firestore index), fix the infrastructure, don't skip the test
  - If a feature is not implemented, write the code, don't skip the test
  - The only acceptable skip pattern: checking for missing environment variables at the beginning of a test

### Test File Checklist (Use this EVERY time)
Before creating or modifying tests:
1. ✓ Is there a corresponding source file for this test file?
2. ✓ Does the test file name match exactly? (`xyz.go` → `xyz_test.go`)
3. ✓ Are all tests for a source file in ONE test file?
4. ✓ No standalone feature/e2e/integration test files?
5. ✓ For repository tests: placed in `pkg/repository/*_test.go`, NOT in firestore/ or memory/ subdirectories?
