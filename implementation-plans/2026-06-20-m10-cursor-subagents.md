# M10 — Cursor Native Subagents (execution plan)

- **Date:** 2026-06-20
- **Milestone:** M10 (follow-up to the lean first cut)
- **Design spec:** [plans/post-mvp/10-cursor-and-negotiation.md](../plans/post-mvp/10-cursor-and-negotiation.md)
- **Language spec:** [§11](../spec/grammar.md#11-host-capability-matrix-v01), [§12](../spec/grammar.md#12-runtime-guarantees-by-target)
- **Status:** Planned
- **Depends on:** M6 (render), the M10 lean first cut (PR #10). Implements `binding.Binding` (M0).
- **Supersedes:** the rule-based emission shipped in the first cut of PR #10.

## 1. Goal

Migrate the Cursor binding from emitting agents as `.cursor/rules/<name>.mdc` (a prompt
library the runbook tells the host to "act as") to emitting **native Cursor subagents** at
`.cursor/agents/<name>.md`, per [cursor.com/docs/subagents](https://cursor.com/docs/subagents).
This makes Cursor a near-peer of the Claude binding: one subagent file per agent, invoked
with `/name`, with parallel Task spawning. It is a **thin assembler** over M6 — no runbook
logic moves here; only file layout, frontmatter keys, host verbs, and negotiation change.

## 2. Why this changes

Cursor now supports first-class named subagents. A subagent is a markdown file with YAML
frontmatter (`name`, `description`, `model`, `readonly`, `is_background`) and is invoked via
`/name` or "use the X subagent". The original binding predates this and used rules as the
only available approximation; that assumption is now obsolete.

## 3. Emitted file tree (review.af)

```
.cursor/
  agents/
    build.md
    deploy.md
    lint.md
    notify_author.md
    reviewer.md
    security.md
    style.md
  commands/
    ship.md
  mcp.json
```

Was (first cut): `.cursor/rules/<name>.mdc`. Now: `.cursor/agents/<name>.md`. `commands/ship.md`
and `mcp.json` keep their shape; `ship.md` text changes (invocation + parallel wording).

## 4. Cursor `Vocabulary` (`vocabulary.go`)

| Method | New Cursor phrasing |
|--------|---------------------|
| `InvokeAgent(a)` | "Use the `<name>` subagent (`/<name>`)" + optional "(step `<label>`)" + " with $1" + " using the output from `<prev>`". |
| `SpawnParallel(branches)` | "Launch the following subagents in parallel using multiple Task calls in one message: …" |
| `RunScript` / `ReadOutput` / `ParseOutputProtocol` / `GotoStep` / `Arg` | unchanged (`$1` flow input). |

## 5. Subagent files (`.cursor/agents/<name>.md`)

Body = `render.AgentDocument(p, agent, cursorVocab).Body` (prompt + `agentflow-output`
protocol). Frontmatter:

```yaml
---
name: reviewer
description: <agent description or generated default>
model: inherit        # or a mapped Cursor model id (see models.go)
readonly: true        # only when the agent's permissions are read-only
---
<rendered prompt body, including agentflow-output protocol>
```

- Drop the old `alwaysApply: false` and `<!-- agentflow: model=… tools=… -->` comment.
- `model`: from `models.go` `HostModelID(provider, alias)`; defaults to `inherit`.
- Cursor subagents have **no per-agent tools allowlist**; tool refs stay metadata-only
  (`AF306`); MCP servers are configured in `.cursor/mcp.json`.

## 6. Model id mapping (`models.go`)

```go
func HostModelID(provider, alias string) (string, bool)
```

Extensible `(provider.alias) -> cursor model id` table; empty for now so everything resolves
to `inherit` (Cursor inherits the parent model unless an explicit id is known). One place for
model ids; no hard failure when unmapped.

## 7. Capability negotiation (`capabilities.go`, §11/§12)

`Capabilities()` Cursor row changes:

| Capability | Was | Now |
|-----------|-----|-----|
| `named-subagents` | false | **true** (`.cursor/agents/`) |
| `parallel-spawn` | false | **true** (advisory Task) |

`AF3xx` deltas:

| Code | Change |
|------|--------|
| `AF300` parallel | Keep as **advisory** ("parallel spawn is advisory on cursor; subagents may run sequentially"); emitted whenever the program uses parallel (special-cased like blocking gates, independent of the capability flag). |
| `AF301` loop bound | unchanged (advisory). |
| `AF302` output parse | unchanged (advisory). |
| `AF303` blocking gate | unchanged (advisory; hooks deferred). |
| `AF305` model unmappable | **removed** — model is now expressed in frontmatter. |
| `AF306` tools | kept, reworded ("tool refs are metadata-only on Cursor subagents; use `.cursor/mcp.json`"). |
| `AF307` permissions | only emitted when permissions do **not** map to `readonly: true`; reworded to "partially map (readonly)". |

`StaticCaveat` reworded to reflect native subagents.

## 8. Testing

- Regenerate goldens under `internal/binding/cursor/testdata/`: delete `.cursor/rules/*.mdc`,
  add `.cursor/agents/*.md`, update `commands/ship.md`. `mcp.json` unchanged.
- Update `cursor_emit_test.go`, `golden_test.go`, `capabilities_test.go`,
  `vocabulary_test.go` for the new paths, frontmatter, invocation/parallel wording, and the
  new negotiation set (AF300 advisory present, AF305 absent, `named-subagents`/`parallel-spawn`
  true).
- `go build ./... && go test ./...` green.

## 9. Acceptance criteria

- [ ] `review.af` emits `.cursor/agents/<name>.md` per reachable agent; no `.cursor/rules/`.
- [ ] Agent files have `name`/`description`/`model` frontmatter; `reviewer.md` contains the
      `agentflow-output` block with the three enum members.
- [ ] `ship.md` uses subagent invocation and advisory parallel wording; gate retry → `build`.
- [ ] `Capabilities()` reports `named-subagents: true`, `parallel-spawn: true`.
- [ ] `AF300` advisory present for `review.af`; `AF305` no longer emitted.
- [ ] Golden FS snapshots pass; no runbook logic duplicated in the binding.

## 10. Commit plan

| # | Commit | Contents |
|---|--------|----------|
| 1 | `binding/cursor: migrate agent emission from rules to .cursor/agents` | `cursor.go` + `formatAgentMD` + `models.go` |
| 2 | `binding/cursor: subagent invocation + parallel vocabulary` | `vocabulary.go` |
| 3 | `binding/cursor: capabilities for native subagents (AF3xx update)` | `capabilities.go` |
| 4 | `binding/cursor: regold review.af for .cursor/agents` | `testdata/**` + tests |
| 5 | `docs(m10): cursor native subagents (spec, overview, plans)` | spec/OVERVIEW/plans/this doc |
| 6 | `chore(m10): progress log — cursor subagents migration` | `progress/2026-06-20.md` |

## 11. Risks & notes

- **Host-format drift** stays the #1 risk; confine every Cursor fact (frontmatter keys, model
  ids, invocation verbs) to this package.
- **History hygiene:** the rule-based commits in PR #10 are already pushed/reviewed — layer
  this migration as new commits, do not rewrite/force-push. Refresh the PR title/body instead.
- **Thin assembler discipline:** any awkward runbook prose is fixed in M6 + its golden, not here.
