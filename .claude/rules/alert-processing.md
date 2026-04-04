---
paths:
  - "pkg/usecase/alert_pipeline*"
  - "pkg/domain/model/alert/**"
---

# Alert Processing Rules

- **Alerts are immutable** and can be linked to at most one ticket
- `ProcessAlertPipeline()` is pure (no side effects); `HandleAlert()` includes DB save and Slack posting — new processing logic should respect this separation
- All pipeline events are emitted through `Notifier` interface for real-time monitoring
