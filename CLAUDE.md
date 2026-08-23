# Claude Code Guidance

Read [AGENTS.md](AGENTS.md) completely before working in this repository. It is
the canonical guide for Apollo's architecture, implementation workflow, review
standards, validation matrix, and contributor handoff. This file only adds a
small Claude Code operating checklist so the two guides do not drift.

## Operating Checklist

1. Inspect `git status --short` before editing. Preserve unrelated tracked and
   untracked changes, including work created by another developer or agent.
2. Build context from the current source and tests. Apollo v2 uses `apollo.go`,
   `backend/`, and gouroboros ledger types; ignore advice that refers to the
   removed `backend/maestro/` package or the v1 `ApolloBuilder.go`,
   `serialization/`, `txBuilding/`, and `crypto/` trees.
3. Search for all callers and implementations before changing an interface,
   response shape, CBOR representation, wallet behavior, or fluent builder
   method.
4. For a bug fix, add or identify a focused regression test and verify that it
   fails for the intended reason without the fix. Test malformed and boundary
   inputs as well as the happy path.
5. Keep edits narrow. Do not create durable plan files, rewrite unrelated code,
   expose secrets, contact live providers, submit transactions, or publish
   artifacts unless the task explicitly requires and authorizes it.
6. Run the narrowest relevant test first, then the broader checks selected from
   `AGENTS.md`. Read command output and exit status before reporting success.
7. Review the final diff for scope, API compatibility, deterministic encoding,
   32-bit integer behavior, documentation drift, and accidental generated or
   dependency changes.

## Review-Only Tasks

Do not modify code when asked only to review. Report findings first, ordered by
severity. Each finding needs a path and line or symbol, a triggering condition,
the impact, and evidence. Treat bot comments as claims to verify, not facts. If
no findings remain, say so and list the validation performed and any checks
that could not be run.

## Completion

Finish with a concise evidence-based handoff: changed paths, behavior affected,
commands and exit codes, skipped checks with reasons, and remaining risks. The
module currently requires Go 1.25.13 or newer; verify `go.mod` rather than
relying on this sentence if the toolchain floor is relevant to the task.
