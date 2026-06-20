# M0 — Foundations (execution plan)

- **Date:** 2026-06-19
- **Milestone:** M0
- **Design spec:** [plans/mvp/00-foundations.md](../plans/mvp/00-foundations.md)
- **Status:** In progress

## Goal

Shared plumbing: diagnostics, in-memory file set, binding interface, CLI skeleton.

## Tasks

- [x] Go module + package scaffold
- [ ] `internal/diag` — `Diagnostic`, `Diagnostics`, `Render` with golden tests
- [ ] `internal/emit` — `FS` with sorted paths, `Flush`, round-trip test
- [ ] `internal/ir` — empty `Program` stub (until M5)
- [ ] `internal/binding` — `Binding` interface + registry
- [ ] `cmd/af` — `validate`, `build`, `graph` subcommand stubs

## Acceptance

- `go build ./...` succeeds
- `go test ./...` green
- `af` prints usage; subcommands print "not implemented"
- Golden test for `diag.Render` (error + warning)
- `emit.FS.Flush` round-trip test

## Out of scope

Parsing, resolution, binding implementations, capability negotiation (M7/M10).
