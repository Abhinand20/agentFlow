# M7 - Claude Code Binding

- Milestone: M7
- Version: v0.1 (MVP) — Language Level A
- Status: Planned
- Spec: [§11 Host capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01), [§12 Runtime guarantees — native + hook-enforced](../../spec/grammar.md#12-runtime-guarantees-by-target)

## Goal

Assemble render-layer output into a working `.claude/` config. Typing `/ship` runs
the flow via native runbook (best-effort control flow) with hook-enforced blocking
gates where supported.

## Scope

### In scope
- Claude `render.Vocabulary`.
- Subagent files, command runbook, MCP, settings, gate hooks.
- Capability declaration + `AF3xx` warnings for advisory-only features.
- DOT emitter.

### Out of scope (deferred)
- Cursor (M10), SDK runtime (M15).

## Packages & files

- `internal/binding/claude/claude.go`
- `internal/binding/claude/vocabulary.go`
- `internal/binding/claude/capabilities.go`
- `internal/binding/dot/dot.go`

## Tasks

- **Claude Vocabulary:** Task-tool invocation, `$ARGUMENTS` for flow input (opaque
  `Ticket`), bash for gate scripts, `ReadOutput` / retry instructions per §9.3.
- **Subagents:** `ir.Agent` -> `.claude/agents/<name>.md` via `render.AgentPrompt`
  + frontmatter (`name`, `description`, `tools`, `model` from resolved provider).
- **Command:** `render.RunbookFromFlow` -> `.claude/commands/<on>.md` with
  frontmatter (`description`, `argument-hint`, `allowed-tools`).
- **MCP:** `kind: mcp` -> `.mcp.json`.
- **Models:** map `(provider, alias)` to Claude model id strings.
- **Permissions:** -> `.claude/settings.json`.
- **Blocking gates (hook-enforced tier):** settings hooks run gate script; block on
  non-zero exit when `behavior: blocking`.
- **Advisory tier warnings (`AF3xx`):** parallel spawn and loop bounds are best-effort
  in runbook; emit warnings documenting limitation vs §12 matrix.
- **Capabilities():** declare supported features per §11 row for `claude-code`.
- **DOT:** resolved flow graph for `af graph`.

## Acceptance criteria

- Golden FS snapshots in `testdata/` for [examples/review.af](../../examples/review.af).
- `reviewer.md` contains `agentflow-output` block with `out:` line.
- `ship.md` runbook: parses `code_review` `out:` for branch; loop on `review != revise`;
  gate `on-fail: retry` references control label `build`.
- Manual smoke: `/ship TICKET-123` in Claude Code.

## Dependencies

- M5 (IR), M6 (render). Implements `binding.Binding` from M0.

## Risks / notes

- Verify Claude hook event names at implementation time; pin in binding only.
- Thin assembler over render layer — no duplicate runbook logic here.
