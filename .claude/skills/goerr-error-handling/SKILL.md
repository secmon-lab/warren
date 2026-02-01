---
name: goerr-error-handling
description: Write Go error handling code using the goerr library. Use when creating errors, wrapping errors, adding context, or handling errors in Go code that uses goerr.
allowed-tools: Read, Bash(go:*)
---

# Error Handling with goerr

When writing Go error handling code, use the `goerr` library for enhanced errors with stack traces, contextual values, and structured logging support.

## Step 1: Locate goerr documentation

First, find where goerr is installed and read its documentation:

```bash
GOERR_DIR=$(go list -m -f '{{.Dir}}' github.com/m-mizutani/goerr/v2 2>/dev/null)
```

If goerr is available, read the relevant documentation from `$GOERR_DIR/`:

- `README.md` - Full feature overview, code examples, and API reference
- `docs/migration.md` - Migration guide from pkg/errors, standard errors, or goerr v1
- `examples/` - Working example programs for each feature

## Step 2: Choose the right function

| Situation | Use | NOT |
|-----------|-----|-----|
| Create a new error | `goerr.New("msg")` | `errors.New("msg")` or `fmt.Errorf("msg")` |
| Wrap an existing error | `goerr.Wrap(err, "msg")` | `fmt.Errorf("msg: %w", err)` |
| Add context without changing message | `goerr.With(err, goerr.V("k", v))` | `goerr.Wrap(err, err.Error(), goerr.V("k", v))` |
| Simple key-value context | `goerr.V("key", value)` | Building context into message string |
| Type-safe context | `goerr.TV(typedKey, value)` | `goerr.V()` when type safety matters |
| Categorize errors | `goerr.T(tagValue)` | Sentinel errors for categories |
| Shared context across errors | `goerr.NewBuilder(opts...)` | Repeating same options everywhere |
| Collect multiple errors | `goerr.Append(errs, err)` | Manual slice management |
| Define sentinel errors | `goerr.New("msg", goerr.ID("ERR_X"))` | `errors.New("msg")` |
| Extract goerr.Error | `goerr.Unwrap(err)` | `errors.As(err, &goErr)` |
| Extract goerr.Errors | `goerr.AsErrors(err)` | `errors.As(err, &goErrs)` |
| Check error tag | `goerr.HasTag(err, tag)` | Manual unwrap + check |

## Step 3: Apply best practices

### Error messages should describe what failed, not why

```go
// Good
goerr.Wrap(err, "failed to save user")

// Bad - "why" belongs in context values
goerr.Wrap(err, fmt.Sprintf("failed to save user %s due to timeout", userID))
```

### Use context values for structured data, not string interpolation

```go
// Good
goerr.Wrap(err, "query failed",
    goerr.V("table", "users"),
    goerr.V("user_id", userID))

// Bad
goerr.Wrap(err, fmt.Sprintf("query on table users failed for user %s", userID))
```

### Define typed keys at package level for critical values

```go
var (
    UserIDKey = goerr.NewTypedKey[string]("user_id")
    CountKey  = goerr.NewTypedKey[int]("count")
)

err := goerr.New("error", goerr.TV(UserIDKey, "user123"))
if userID, ok := goerr.GetTypedValue(err, UserIDKey); ok {
    // userID is guaranteed to be string
}
```

### Define tags and sentinel errors at package level

```go
var (
    TagNotFound   = goerr.NewTag("not_found")
    TagValidation = goerr.NewTag("validation")
)

var (
    ErrNotFound = goerr.New("not found", goerr.ID("ERR_NOT_FOUND"))
    ErrInvalid  = goerr.New("invalid input", goerr.ID("ERR_INVALID"))
)
```

### Use Builder for shared context in a scope

```go
func (s *Service) process() error {
    eb := goerr.NewBuilder(
        goerr.V("user_id", s.userID),
        goerr.V("request_id", s.reqID))

    if err := s.validate(); err != nil {
        return eb.Wrap(err, "validation failed")
    }
    if err := s.save(); err != nil {
        return eb.Wrap(err, "save failed")
    }
    return nil
}
```

### Use Unstack() for helper functions

```go
func newDomainError(msg string, opts ...goerr.Option) *goerr.Error {
    return goerr.New(msg, opts...).Unstack()
}
```

### Use slog integration for structured logging

```go
logger.Error("operation failed", slog.Any("error", err))
// Automatically includes message, values, tags, and stack trace
```

## Quick Reference

**Error creation**: `goerr.New`, `goerr.Wrap`, `goerr.With`

**Options**: `goerr.Value` / `goerr.V`, `goerr.TypedValue` / `goerr.TV`, `goerr.Tag` / `goerr.T`, `goerr.ID`

**Extraction**: `goerr.Unwrap`, `goerr.AsErrors`, `goerr.Values`, `goerr.TypedValues`, `goerr.Tags`, `goerr.HasTag`, `goerr.GetTypedValue`

**Multiple errors**: `goerr.Join`, `goerr.Append`, `(*Errors).ErrorOrNil`, `(*Errors).IsEmpty`, `(*Errors).Len`

**Builder**: `goerr.NewBuilder`, `(*Builder).With`, `(*Builder).New`, `(*Builder).Wrap`

**Type-safe keys**: `goerr.NewTypedKey[T]`, `goerr.TV`, `goerr.GetTypedValue`

**Tags**: `goerr.NewTag`, `goerr.T`, `goerr.HasTag`, `(*Error).HasTag`

**Stack control**: `(*Error).Unstack`, `(*Error).UnstackN`
