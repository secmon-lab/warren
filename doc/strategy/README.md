# Chat Strategies

Warren's chat system uses pluggable **strategies** to process user messages. Each strategy implements the `interfaces.ChatUseCase` interface and defines how the agent plans, executes, and responds.

## Strategy Selection

Use the `--chat-strategy` flag (or `WARREN_CHAT_STRATEGY` environment variable) to select a strategy:

```bash
warren serve --chat-strategy aster    # default
warren serve --chat-strategy bluebell
```

## Available Strategies

| Strategy | Description | Knowledge Service |
|----------|-------------|-------------------|
| [aster](aster.md) | Default security investigation with parallel task execution | Optional |
| [bluebell](bluebell.md) | Adaptive investigation with intent resolution from multiple prompts | **Required** |

## Naming Convention

Strategies are named after wildflowers in alphabetical order: **aster** (A), **bluebell** (B), **clover** (C), **daisy** (D), etc. When adding a new strategy, use the next letter in sequence.

## Common Architecture

All strategies share these components from `pkg/usecase/chat/`:

- **auth.go** — Policy-based authorization for agent execution
- **history.go** — Chat history persistence and loading
- **thread.go** — Slack thread context handling

Each strategy is a separate package under `pkg/usecase/chat/<name>/` with its own prompt templates, execution logic, and configuration options.

## Execution Flow Overview

Both current strategies follow a common high-level flow:

```
User Message → Authorization → Planning → Task Execution → Replanning → Final Response
```

The key differences lie in what happens *before* planning (intent resolution in bluebell) and how the system prompt is constructed.
