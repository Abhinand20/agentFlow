# M10 - Cursor Binding & Capability Negotiation

- Milestone: M10
- Version: v0.2
- Status: **Partial — lean first cut done** ([PR #10](https://github.com/Abhinand20/agentFlow/pull/10))
- Spec: [§11 Host capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01), [§12 Runtime guarantees](../../spec/grammar.md#12-runtime-guarantees-by-target)
- Execution plan: [implementation-plans/2026-06-19-m10-cursor-binding.md](../../implementation-plans/2026-06-19-m10-cursor-binding.md)

## Goal

Add Cursor as a second host via shared render layer + Cursor vocabulary, and
implement capability negotiation so unsupported constructs warn with documented
fallbacks per the host matrix.

## Scope

### In scope (lean cut — done)

- Cursor binding: commands, rules (`.mdc`), mcp.json
- Cursor-local capability negotiation (`AF300`–`AF303`, `AF305`–`AF308`)
- Golden FS snapshots for `review.af`

### In scope (remaining)

- `.cursor/hooks.json` (beta)
- Negotiation framework shared by all bindings (`internal/binding/capability.go`)
- `BUILD-NOTES.md` or equivalent build-time fallback summary

### Out of scope (deferred)

- New language features.
- CLI wiring (`af build --target cursor`) — **M8**.

## Packages & files

| Path | Status |
|------|--------|
| `internal/binding/cursor/cursor.go` | Done |
| `internal/binding/cursor/vocabulary.go` | Done |
| `internal/binding/cursor/capabilities.go` | Done (cursor-local; not shared yet) |
| `internal/binding/capability.go` | Planned |
| `.cursor/hooks.json` emit | Planned |

## Tasks

- **Cursor Vocabulary:** ✅ rule-based agents, `$1` flow input, sequential parallel fallback, §9 output-protocol instructions.
- **Assemble:**
  - ✅ `.cursor/commands/<on>.md` — runbook from M6 + HTML metadata comment
  - ✅ `.cursor/rules/<name>.mdc` — agent prompt + frontmatter + agentflow metadata
  - ✅ `.cursor/mcp.json`
  - ⏳ `.cursor/hooks.json` (beta)
- **Negotiation framework** (§11 / §12):
  - ✅ `Binding.Capabilities()` cursor row
  - ✅ Diff program needs vs capabilities → `AF3xx` warnings (cursor-local)
  - ⏳ Extract shared framework for Claude (M7) and other bindings
  - ⏳ Document fallback in `BUILD-NOTES.md` or build stderr (M8 prints diags)

## Acceptance criteria

- [x] [examples/review.af](../../examples/review.af) emits Cursor artifacts (binding unit tests; CLI in M8).
- [x] `AF3xx` warnings for blocking gate + parallel on Cursor; artifacts still emitted.
- [x] Golden FS snapshots for Cursor target.
- [x] Same render text as default vocabulary where capabilities overlap (control-flow prose from M6).
- [ ] `.cursor/hooks.json` or documented permanent advisory fallback for gates.
- [ ] Shared `internal/binding/capability.go`.

## Dependencies

- M6 (render) — done. M7 binding pattern reused; M7 Claude still planned separately.
- M0 (`Capabilities()` on interface).

## Risks / notes

- Cursor hooks beta — matrix marks as partial; AF303 documents honest fallback.
- Implemented on branch `feat/m7-cursor-binding` before M7 Claude; see execution plan for PR stack.
