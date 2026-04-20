# Chat Session Redesign Migration (Phase 7)

This document describes the production migration procedure for the
chat-session-redesign Phase 7 data move. All of the changes are driven
by the `warren migrate` CLI, with one `--job` per step. The jobs are
implemented in `pkg/usecase/migration/` and wired in
`pkg/cli/migrate_chat_session.go`.

**All jobs in this PR are non-destructive.** No job deletes pre-redesign
Firestore documents or Cloud Storage objects. Legacy `ticket.Comment`
rows and legacy `{prefix}/{schema}/ticket/{tid}/latest.json` objects
stay in place indefinitely — operators can decide separately, outside
the scope of this migration, when (if ever) to discard them.

## Prerequisites

- Firestore Admin write access to the target project/database.
- Cloud Storage write access to the bucket that holds
  `{prefix}/{schema}/ticket/{tid}/latest.json` and the new
  `{prefix}/{schema}/sessions/{sid}/history.json` layout.
- A recent Firestore PITR (point-in-time-recovery) boundary or export
  snapshot. The jobs here only *write* new data, but keeping a PITR
  boundary is still the cheapest rollback for any operator mistake
  (wrong project, partial interruption, etc.).

## Required CLI flags

```sh
warren migrate \
  --firestore-project-id $FIRESTORE_PROJECT \
  --firestore-database-id $FIRESTORE_DATABASE \
  --storage-bucket $STORAGE_BUCKET \
  --storage-prefix $STORAGE_PREFIX \
  --job <job-name> \
  [--dry-run]
```

The Firestore flags are always required. `--storage-bucket` /
`--storage-prefix` are only consulted by the job that copies GCS
objects (`history-scope`); they are ignored otherwise.

## Execution order

Run each job with `--dry-run` first, inspect the log output
(`scanned / migrated / skipped / errors` counters), and only drop the
flag after review. **Execute in this order**:

1. `session-source-backfill` — stamp `Source` / `TicketIDPtr` on every
   pre-redesign `Session` row. Idempotent. Safe to re-run.
2. `session-consolidate` — materialize the canonical `slack_<hash>`
   Session for every Slack-origin Ticket so the Conversation sidebar
   has a single survivor per thread. Non-destructive; legacy UUID
   Session rows stay in Firestore and are filtered out at read time.
3. `comment-to-message` — rewrite each `ticket.Comment` as a
   `session.Message(type=user)` attached to the resolving Slack
   Session. Idempotent (de-duplicates by `(content, createdAt)`
   per Session). Leaves the original `Comment` documents in place.
4. `turn-synthesis` — synthesize one `Turn` per legacy Session and
   stamp the resulting `TurnID` onto every Message on that Session.
   Idempotent: skipped for any Session that already carries a Turn.
5. `history-scope` — **server-side copy** of
   `{prefix}/{schema}/ticket/{tid}/latest.json` into
   `{prefix}/{schema}/sessions/{sid}/history.json` for every Slack
   Session. The rewrite runs entirely inside GCS (see
   `storage.Service.CopyLatestHistoryToSession`); no payload bytes
   traverse the migrate binary, so egress stays at zero regardless of
   dataset size. Idempotent: destinations that already exist are
   skipped. Source files are left in place.

## Rollback

| Step | Rollback mechanism |
|---|---|
| 1–4 | Firestore PITR to the boundary snapshot taken before migration. Nothing is deleted, so in most cases it is enough to re-run the forward job after fixing whatever drove the rollback. |
| 5 | Delete the `{prefix}/{schema}/sessions/{sid}/history.json` destination objects (or overwrite them); the legacy `latest.json` path is untouched. |

## Verification

After every step, verify:

- `go test ./pkg/usecase -run TestSlackGolden` is still green in the
  deployed image (no regression in user-visible Slack behaviour).
- Spot-check a handful of tickets in the Web UI under `Conversation`
  (the new `TicketConversation` card) — the messages should show both
  the legacy Slack comment history (step 2 output) and any Web/CLI
  session messages authored after deploy.
- `warren_get_ticket_session_messages` returns the expected rows for a
  test ticket.

## Legacy code surface

The `migrate` command and the jobs above remain part of the
application **indefinitely** — any fresh Firestore database created
before the redesign will need them. The following files therefore do
**not** get deleted:

- `pkg/domain/model/ticket/comment.go`
- `pkg/domain/model/ticket/history.go`
- `pkg/usecase/migration/*`
- `pkg/cli/migrate_chat_session.go`
  (`firestoreCommentSource` etc.)

Instead, the post-redesign codebase **confines** Comment access to the
migration package. The rest of the application does not know Comments
exist — `interfaces.Repository` no longer exposes Comment CRUD, the
Firestore / Memory repositories no longer implement it, and the
GraphQL / Web UI / agent tool surfaces have been rewritten against
`SessionMessage`. If a future PR needs to read a Comment for any
reason, it should go through `migration.CommentSource` rather than
re-introduce the legacy methods on the main Repository.
