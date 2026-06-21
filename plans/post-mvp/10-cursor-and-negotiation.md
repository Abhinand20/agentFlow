# M10 - Cursor Binding & Capability Negotiation

- Milestone: M10
- Version: v0.2
- Status: **Partial — lean cut + native subagents migration done** ([PR #10](https://github.com/Abhinand20/agentFlow/pull/10))
- Spec: [§11 Host capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01), [§12 Runtime guarantees](../../spec/grammar.md#12-runtime-guarantees-by-target)
- Execution plans: [implementation-plans/2026-06-19-m10-cursor-binding.md](../../implementation-plans/2026-06-19-m10-cursor-binding.md) (lean cut), [implementation-plans/2026-06-20-m10-cursor-subagents.md](../../implementation-plans/2026-06-20-m10-cursor-subagents.md) (native subagents migration)

## Goal

Add Cursor as a second host via shared render layer + Cursor vocabulary, and
implement capability negotiation so unsupported constructs warn with documented
fallbacks per the host matrix.

## Scope

### In scope (lean cut + subagents migration — done)

- Cursor binding: commands, native subagents (`.cursor/agents/`), mcp.json (lean cut
  shipped `.cursor/rules/*.mdc`; since migrated to native `.cursor/agents/*.md`)
- Cursor-local capability negotiation (`AF300`–`AF303`, `AF306`–`AF308`; `AF305` dropped
  once model moved into agent frontmatter)
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

- **Cursor Vocabulary:** ✅ native subagent invocation (`/<name>` / "use the `<name>`
  subagent"), advisory parallel via multiple Task calls in one message, `$1` flow input,
  §9 output-protocol instructions. (Lean cut shipped rule-based agents + sequential
  fallback; since migrated to native subagents.)
- **Assemble:**
  - ✅ `.cursor/commands/<on>.md` — runbook from M6 + HTML metadata comment
  - ✅ `.cursor/agents/<name>.md` — native Cursor subagent (YAML frontmatter `name`,
    `description`, `model` (`inherit` by default), optional `readonly`; body =
    `render.AgentDocument` prompt + §9 output protocol). Replaces the lean-cut
    `.cursor/rules/<name>.mdc` prompt library.
  - ✅ `.cursor/mcp.json`
  - ⏳ `.cursor/hooks.json` (beta)
- **Negotiation framework** (§11 / §12):
  - ✅ `Binding.Capabilities()` cursor row
  - ✅ Diff program needs vs capabilities → `AF3xx` warnings (cursor-local)
  - ⏳ Extract shared framework for Claude (M7) and other bindings
  - ⏳ Document fallback in `BUILD-NOTES.md` or build stderr (M8 prints diags)

## Acceptance criteria

- [x] [examples/review.af](../../examples/review.af) emits Cursor artifacts —
  `.cursor/agents/<name>.md` per agent, no `.cursor/rules/` (binding unit tests; CLI in M8).
- [x] `AF3xx` warnings for blocking gate + (advisory) parallel on Cursor; artifacts still
  emitted. Agent model is expressed in frontmatter, so no model-unmappable (`AF305`) warning.
- [x] Golden FS snapshots for Cursor target.
- [x] Same render text as default/Claude vocabulary where capabilities overlap (named
  subagents, parallel Task; control-flow prose from M6).
- [ ] `.cursor/hooks.json` or documented permanent advisory fallback for gates.
- [ ] Shared `internal/binding/capability.go`.

## Dependencies

- M6 (render) — done. M7 binding pattern reused; M7 Claude still planned separately.
- M0 (`Capabilities()` on interface).

## Risks / notes

- Cursor hooks beta — matrix marks as partial; AF303 documents honest fallback.
- Implemented on branch `feat/m7-cursor-binding` before M7 Claude; see execution plan for PR stack.
