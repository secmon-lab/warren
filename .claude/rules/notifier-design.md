---
paths:
  - "pkg/domain/interfaces/notifier*"
  - "pkg/domain/event/**"
  - "pkg/service/notifier/**"
---

# Notifier Design Rule

- Notifier uses **type-safe event methods** — each event type has its own dedicated method (e.g., `NotifyIngestPolicyResult`, `NotifyError`)
- **Do NOT add a generic `Notify(event)` method** — always add a new typed method for new event types
