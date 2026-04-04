# CLAUDE.md

Warren is an AI agent and Slack-based security alert management tool that processes security alerts using LLM (Gemini) and manages incident response through Slack.

## Common Development Commands

### Building and Testing
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

## Architecture Overview

The application follows Domain-Driven Design (DDD) with clean architecture:

- `pkg/domain/` - Domain layer with business logic, interfaces, and models
- `pkg/service/` - Application services implementing business operations
- `pkg/controller/` - Interface adapters (HTTP, GraphQL, Slack)
- `pkg/adapter/` - Infrastructure adapters (storage, external APIs)
- `pkg/repository/` - Data persistence implementations
- `pkg/usecase/` - Application use cases orchestrating domain operations

### Alert Processing Pipeline
Pipeline stages in `pkg/usecase/alert_pipeline.go`:
1. **Ingest Policy Evaluation** - Transform raw alert data into Alert objects
2. **Tag Conversion** - Convert tag names to tag IDs
3. **Metadata Generation** - Fill missing titles/descriptions using LLM
4. **Enrich Policy Evaluation** - Execute enrichment tasks (query/agent)
5. **Triage Policy Evaluation** - Apply final metadata and determine publish type

### Application Modes
`serve` (HTTP/Slack/GraphQL), `run` (CLI), `chat` (interactive), `tool` (utilities), `test` (testing)

## Development Rules

### Implementation Completeness
- **NEVER leave incomplete implementations, TODOs, or placeholder code**
- **NEVER skip implementation because it's complex or lengthy**
- **ALWAYS complete the full implementation in one go**
- If a task seems too complex, break it down into smaller steps, but complete ALL steps
- Complexity is not an excuse - implement everything thoroughly
- Long code is acceptable - incomplete code is NOT

### Test Requirements
- **EVERY code change MUST be accompanied by tests that verify the change**
- When adding new functionality, write tests that cover the new behavior
- When fixing a bug, write a test that reproduces the bug and verifies the fix
- When refactoring, ensure existing tests still pass and add tests if coverage gaps are found
- Do NOT consider a task complete until tests are written and passing

### Frontend Test Requirements
- **EVERY frontend feature addition or modification MUST include corresponding E2E tests**
- E2E tests are located in `frontend/e2e/tests/` using Playwright
- Page Objects are in `frontend/e2e/pages/` — add or update them as needed
- Use semantic locators (`getByRole`, `getByText`, `data-testid`) instead of CSS class selectors
- Do NOT use `waitForLoadState("networkidle")` — rely on Playwright's auto-waiting
- Do NOT consider a frontend task complete until E2E tests covering the new/changed behavior are written

### Frontend Date Format
- ALWAYS use `YYYY/MM/DD` format. NEVER use `MM/DD/YYYY` or locale-dependent formats like `toLocaleDateString()`
  - Use: `date.toISOString().split('T')[0].replace(/-/g, '/')`
  - Do NOT use: `date.toLocaleDateString()`

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Must wrap errors with `goerr.Wrap` to maintain error context
- Add helpful variables with `goerr.V` for debugging
- **NEVER check error messages using `strings.Contains(err.Error(), ...)`**
- **ALWAYS use `errors.Is(err, targetErr)` or `errors.As(err, &target)` for error type checking**
- Error discrimination must be done by error types, not by parsing error messages
- Tag errors with `goerr.T(errutil.TagXxx)` from `pkg/utils/errutil` where appropriate (see existing code for examples)
- **Use `errutil.Handle(ctx, err)` for error logging in background goroutines and fire-and-forget contexts** — it logs the error and sends it to Sentry in one call
  - BAD: `logger.Error("failed to do X", "error", err)`
  - GOOD: `errutil.Handle(ctx, goerr.Wrap(err, "failed to do X", goerr.V("id", id)))`

### Resource Cleanup
- **ALWAYS use `safe.Close(ctx, closer)` from `pkg/utils/safe`** to close `io.Closer` resources
- **NEVER use `_ = x.Close()` or bare `x.Close()`** — use `safe.Close` instead for nil-safe, error-logged cleanup
  - BAD: `defer func() { _ = client.Close() }()`, `defer client.Close()`
  - GOOD: `defer safe.Close(ctx, client)`

### Testing Conventions
- Use `github.com/m-mizutani/gt` package for type-safe testing
- Prefer Helper Driven Testing style over Table Driven Tests
- Use Memory repository from `pkg/repository` instead of mocks for repository testing
- Use mock implementations from `pkg/domain/mock`
- **NEVER comment out test assertions** - if a test doesn't work, fix it or delete it
- **NEVER use length-only checks** - always verify individual IDs/values explicitly
  - BAD: `gt.A(t, toDelete).Length(3)` with commented out ID checks
  - GOOD: Check each expected ID explicitly with `gt.True(t, deleteMap[id])`
- Test files should have `package {name}_test`. Do not use same package name
- Test file name convention is: `xyz.go` -> `xyz_test.go`. Other test file names (e.g., `xyz_e2e_test.go`) are not allowed
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
1. Is there a corresponding source file for this test file?
2. Does the test file name match exactly? (`xyz.go` -> `xyz_test.go`)
3. Are all tests for a source file in ONE test file?
4. No standalone feature/e2e/integration test files?
5. For repository tests: placed in `pkg/repository/*_test.go`, NOT in firestore/ or memory/ subdirectories?

### Code Visibility
- Do not expose unnecessary methods, structs, and variables
- Assume that exposed items will be changed. Never expose fields that would be problematic if changed
- Use `export_test.go` for items that need to be exposed for testing purposes

### Alert Processing Rules
- **Alerts are immutable** and can be linked to at most one ticket
- `ProcessAlertPipeline()` is pure (no side effects); `HandleAlert()` includes DB save and Slack posting — new processing logic should respect this separation
- All pipeline events are emitted through `Notifier` interface for real-time monitoring

### Notifier Design Rule
- Notifier uses **type-safe event methods** — each event type has its own dedicated method (e.g., `NotifyIngestPolicyResult`, `NotifyError`)
- **Do NOT add a generic `Notify(event)` method** — always add a new typed method for new event types

### Chat Strategy Naming Convention
Chat strategies are named after wildflowers in alphabetical order: `aster` (A), `bluebell` (B), `clover` (C), `daisy` (D), etc. Each strategy is a separate package under `pkg/usecase/chat/<flower>/` implementing the `interfaces.ChatUseCase` interface. The current default strategy is `aster`. When adding a new strategy, use the next wildflower in alphabetical sequence.

### Pre-commit Checks
When making changes, before finishing the task, always:
- Run `go vet ./...`, `go fmt ./...` to format the code
- Run `golangci-lint run ./...` to check lint error
- Run `gosec -exclude-generated -quiet ./...` to check security issue
- Run tests to ensure no impact on other code

**NEVER run `go build` to verify code.** Use `go vet ./...` instead to check for compile errors.

### Language
All comment and character literal in source code must be in English

### Directory
- When you are mentioned about `tmp` directory, you SHOULD NOT see `/tmp`. You need to check `./tmp` directory from root of the repository.
