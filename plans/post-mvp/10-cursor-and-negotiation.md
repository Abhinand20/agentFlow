# M10 - Cursor Binding & Capability Negotiation

- Milestone: M10
- Version: v0.2
- Status: Planned
- Spec: [§11 Host capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01), [§12 Runtime guarantees](../../spec/grammar.md#12-runtime-guarantees-by-target)

## Goal

Add Cursor as a second host via shared render layer + Cursor vocabulary, and
implement capability negotiation so unsupported constructs warn with documented
fallbacks per the host matrix.

## Scope

### In scope
- Cursor binding (commands, rules, mcp, hooks beta).
- Negotiation framework shared by all bindings.

### Out of scope (deferred)
- New language features.

## Packages & files

- `internal/binding/cursor/cursor.go`
- `internal/binding/cursor/vocabulary.go`
- `internal/binding/capability.go`

## Tasks

- **Cursor Vocabulary:** ad-hoc Task subagents, `$ARGUMENTS`, same output-protocol
  instructions as Claude (§9).
- **Assemble:**
  - `.cursor/commands/<on>.md` — runbook from M6
  - `.cursor/rules/<name>.mdc` — agent context + subagent prompt library
  - `.cursor/mcp.json`
  - `.cursor/hooks.json` (beta)
- **Negotiation framework** (§11 / §12):
  - `Binding.Capabilities()` per host row: command trigger, named subagents, MCP,
    hooks, parallel spawn, blocking gates, output parse enforcement.
  - Diff program needs vs capabilities -> `AF3xx` warnings + fallback behavior:
    - blocking gate -> advisory runbook step on Cursor
    - parallel -> sequential fallback wording
    - loop bounds -> advisory counter text
  - Document fallback in emitted `BUILD-NOTES.md` or build stderr.

## Acceptance criteria

- [examples/review.af](../../examples/review.af) builds for `cursor` target.
- `AF3xx` warnings for blocking gate + parallel on Cursor; artifacts still emitted.
- Golden FS snapshots for Cursor target.
- Same render text as Claude where capabilities overlap.

## Dependencies

- M6 (render), M7 (binding pattern), M0 (`Capabilities()` on interface).

## Risks / notes

- Cursor hooks beta — matrix marks as partial; negotiation must be honest.
