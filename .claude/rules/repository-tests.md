---
paths:
  - "pkg/repository/**"
---

# Repository Test Rules

- NEVER create test files in `pkg/repository/firestore/` or `pkg/repository/memory/` subdirectories
- ALL repository tests MUST be placed directly in `pkg/repository/*_test.go`
- Use `runRepositoryTest()` helper to test against both memory and firestore implementations
- Always use random IDs (e.g., using `time.Now().UnixNano()`) to avoid test conflicts
- Never use hardcoded IDs like "msg-001", "user-001" as they cause test failures when running in parallel
- Always verify ALL fields of returned values, not just checking for nil/existence
- Compare expected values properly - don't just check if something exists, verify it matches what was saved
- For timestamp comparisons, use tolerance (e.g., `< time.Second`) to account for storage precision
