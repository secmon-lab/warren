# Chat Session Redesign Migration (Phase 7)

This document describes the production migration procedure for the
chat-session-redesign Phase 7 data move. All of the changes are driven
by the `warren migrate` CLI, with one `--job` per step. The jobs are
implemented in `pkg/usecase/migration/` and wired in
`pkg/cli/migrate_chat_session.go`.

## Prerequisites

- Firestore Admin write access to the target project/database.
- Cloud Storage write access to the bucket that holds
  `{prefix}/{schema}/ticket/{tid}/latest.json` and the new
  `{prefix}/{schema}/sessions/{sid}/history.json` layout.
- A recent Firestore PITR (point-in-time-recovery) boundary or export
  snapshot. Every step listed below is destructive at some level; the
  rollback strategy is "restore Firestore to $PITR − 1h" and reset GCS
  object versions.

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
`--storage-prefix` are only consulted by jobs that copy or delete GCS
objects (`history-scope`, `cleanup-legacy`); they are ignored otherwise.

## Execution order

Run each job with `--dry-run` first, inspect the log output
(`scanned / migrated / skipped / errors` counters), and only drop the
flag after review. **Execute in this order**:

1. `session-source-backfill` — stamp `Source` / `TicketIDPtr` on every
   pre-redesign `Session` row. Idempotent. Safe to re-run.
2. `comment-to-message` — rewrite each `ticket.Comment` as a
   `session.Message(type=user)` attached to the resolving Slack
   Session. Idempotent (de-duplicates by `(content, createdAt)`
   per Session). Leaves the original `Comment` documents in place.
3. `turn-synthesis` — synthesize one `Turn` per legacy Session and
   stamp the resulting `TurnID` onto every Message on that Session.
   Idempotent: skipped for any Session that already carries a Turn.
4. `history-scope` — copy
   `{prefix}/{schema}/ticket/{tid}/latest.json` into
   `{prefix}/{schema}/sessions/{sid}/history.json` for every Slack
   Session. Idempotent: destinations that already contain a non-empty
   history are skipped. Leaves the source file in place.
5. `cleanup-legacy` — **destructive**. Deletes every `ticket_comments`
   subdocument and every legacy `latest.json` object. Run only after
   steps 1–4 have completed successfully on the full dataset and the
   application has been verified against the new schema.

## Rollback

| Step | Rollback mechanism |
|---|---|
| 1–3 | Firestore PITR to the boundary snapshot taken before migration. |
| 4 | Re-copy from the legacy `latest.json` path (still present — step 4 does not delete). |
| 5 | Firestore PITR for comment documents; GCS object-version restore for deleted `latest.json`. |

Keep the PITR / object-version window open for at least 7 days after
running step 5 before deleting `ticket.Comment` / `ticket.History`
code paths from the repository.

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

The `migrate` command and the five jobs above remain part of the
application **indefinitely** — any fresh Firestore database created
before the redesign will need them. The following files therefore do
**not** get deleted after step 5:

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
reason, it should go through `migration.LegacyCommentStore` rather
than re-introduce the legacy methods on the main Repository.
