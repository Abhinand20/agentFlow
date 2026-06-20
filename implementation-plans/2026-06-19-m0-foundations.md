# M0 — Foundations (execution plan)

- **Date:** 2026-06-19
- **Milestone:** M0
- **Design spec:** [plans/mvp/00-foundations.md](../plans/mvp/00-foundations.md)
- **Status:** Done

## Goal

Shared plumbing: diagnostics, in-memory file set, binding interface, CLI skeleton.

## Tasks

- [x] Go module + package scaffold
- [x] `internal/diag` — `Diagnostic`, `Diagnostics`, `Render` with golden tests
- [x] `internal/emit` — `FS` with sorted paths, `Flush`, round-trip test
- [x] `internal/ir` — empty `Program` stub (until M5)
- [x] `internal/binding` — `Binding` interface + registry
- [x] `cmd/af` — `validate`, `build`, `graph` subcommand stubs

## Acceptance

- [x] `go build ./...` succeeds
- [x] `go test ./...` green
- [x] `af` prints usage; subcommands print "not implemented"
- [x] Golden test for `diag.Render` (error + warning)
- [x] `emit.FS.Flush` round-trip test

## Commits

| Commit | Deliverable |
|--------|-------------|
| `e4dd0cf` | Bootstrap docs + go.mod |
| `6608c02` | internal/diag |
| `b5d733c` | internal/emit |
| `c09d1e5` | internal/binding + ir stub |
| `610f540` | cmd/af CLI |

## Out of scope

Parsing, resolution, binding implementations, capability negotiation (M7/M10).
