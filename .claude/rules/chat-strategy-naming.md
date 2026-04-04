---
paths:
  - "pkg/usecase/chat/**"
---

# Chat Strategy Naming Convention

Chat strategies are named after wildflowers in alphabetical order: `aster` (A), `bluebell` (B), `clover` (C), `daisy` (D), etc. Each strategy is a separate package under `pkg/usecase/chat/<flower>/` implementing the `interfaces.ChatUseCase` interface. The current default strategy is `aster`. When adding a new strategy, use the next wildflower in alphabetical sequence.
