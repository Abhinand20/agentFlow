# M7 — Claude Code Binding (execution plan)

- **Date:** 2026-06-19
- **Milestone:** M7
- **Design spec:** [plans/mvp/07-binding-claude-code.md](../plans/mvp/07-binding-claude-code.md)
- **Language spec:** [§11](../spec/grammar.md#11-host-capability-matrix-v01), [§12](../spec/grammar.md#12-runtime-guarantees-by-target)
- **Status:** Planned
- **Depends on:** M5 (IR), M6 (render). Implements `binding.Binding` (M0).
- **Blocks:** M8 (`af build --target claude-code`, E2E)

## 1. Goal

Assemble render-layer output into a working `.claude/` configuration so that typing
`/ship TICKET-123` in Claude Code runs the flow via a **native runbook** (best-effort
control flow) with **hook-enforced blocking gates** where supported. This is a **thin
assembler** over M6 — it must contain **no runbook logic**; it only chooses file layout,
frontmatter keys, host verbs (the Claude `Vocabulary`), and the model-id mapping, then
diffs program needs against host capabilities to emit honest `AF3xx` fallback warnings.

> Before implementation, **verify current Claude Code formats** (settings hooks schema,
> subagent/command frontmatter keys, `.mcp.json` shape, model id strings) against live
> docs — use the Context7 MCP (`/context7` for "Claude Code subagents / hooks / slash
> commands") or official Anthropic docs. The shapes below are the design intent; pin exact
> keys at implementation time and record them in this binding only.

## 2. Deliverables

- `internal/binding/claude/claude.go` — `Binding` impl + file assembly.
- `internal/binding/claude/vocabulary.go` — Claude `render.Vocabulary`.
- `internal/binding/claude/capabilities.go` — capability map + `AF3xx` negotiation.
- `internal/binding/claude/models.go` — `(provider, alias)` → Claude model id.
- `internal/binding/dot/dot.go` — DOT emitter for `af graph`.
- Golden FS snapshots in `internal/binding/claude/testdata/`.
- Self-registration so `binding.Get("claude-code")` works.

## 3. Emitted file tree

For `review.af`, `Emit` produces an `emit.FS` with (paths relative to `--out` dir):

```
.claude/
  agents/
    build.md
    lint.md
    security.md
    style.md
    reviewer.md
    deploy.md
    notify_author.md
  commands/
    ship.md
  settings.json
.mcp.json
```

- **Agents** — one file per reachable `ir.Agent`.
- **Command** — `.claude/commands/<trigger-basename>.md`. `on: "/ship"` → `ship.md` (strip
  leading `/`).
- **`.mcp.json`** — from `mcp` capabilities (project-scoped at repo root, per Claude
  convention; verify location).
- **`settings.json`** — permissions + gate hooks.

All paths registered via `emit.FS.Write` (sorted, deterministic). The binding returns
`(*emit.FS, diag.Diagnostics)`.

## 4. Claude `Vocabulary` (`vocabulary.go`)

Implements `render.Vocabulary` (M6 §3) with Claude phrasing:

| Method | Claude phrasing |
|--------|-----------------|
| `InvokeAgent(a)` | "Use the Task tool to invoke the `<name>` subagent." (verify Task-tool wording / subagent invocation convention). |
| `RunScript(g)` | "Run `<run>` with Bash." |
| `SpawnParallel(branches)` | "Launch the following subagents in parallel using multiple Task calls in one message: …" (advisory; §11 parallel = advisory). |
| `ReadOutput(vl)` | "Read the `out:` value from the `<vl>` subagent's `agentflow-output` block." |
| `ParseOutputProtocol(enum, retry)` | §9.3 table prose. |
| `GotoStep(label)` | "Return to step `<label>` and continue from there." |
| `Arg(name)` | `"$ARGUMENTS"` (Claude command argument substitution). |

The flow input (opaque `Ticket`) maps to `$ARGUMENTS` in the command runbook.

## 5. Agent files (`.claude/agents/<name>.md`)

Body = `render.AgentDocument(p, agent, claudeVocab).Body` (base prompt + output protocol).
Frontmatter (YAML) maps render's neutral `FMField`s to Claude keys (verify exact keys):

```yaml
---
name: reviewer
description: <agent description or generated default>
tools: <comma/space-separated host tool names from ir tools>
model: <hostModelID resolved from (provider, alias)>
---
<rendered prompt body, including agentflow-output protocol>
```

- `tools`: map `ir.ToolRef{Capability, Tool}` to the host tool reference Claude expects for
  MCP tools (e.g. `mcp__github__get_pr` — **verify the MCP tool naming convention**). If a
  binding cannot express a tool, emit `AF3xx` and omit.
- `model`: from §7 mapping. The IR's `HostModelID` is empty (M5 §3.1); the binding fills it
  here.

## 6. Command runbook (`.claude/commands/ship.md`)

Body = `render.RunbookDocument(p, claudeVocab).Body`. Frontmatter (verify keys):

```yaml
---
description: <entry flow description / generated>
argument-hint: <input type hint, e.g. "TICKET-123">
allowed-tools: <tools needed across the flow, e.g. Task, Bash, mcp__github__*>
---
<numbered runbook: §9 parse instructions, gate retry to `build`, loop, branch>
```

The runbook must include the §9.3 parse/retry/halt instructions and the gate `retry`
targeting control label `build` — all produced by M6; the binding only places the text.

## 7. Model id mapping (`models.go`)

```go
func HostModelID(provider, alias string) (string, bool)
```

A table mapping `(anthropic, opus|sonnet|haiku)` → Claude model id strings. **Verify
current ids before pinning** (they change); example shape:

| provider.alias | host model id (verify!) |
|----------------|-------------------------|
| anthropic.opus | `claude-opus-4-…` |
| anthropic.sonnet | `claude-sonnet-4-…` |
| anthropic.haiku | `claude-haiku-…` |

Unknown `(provider, alias)` → `AF3xx` warning ("no Claude model id for anthropic.foo;
emitting alias verbatim") and fall back to emitting the alias so the file is still
inspectable. Keep the table small and documented; this is the one place model ids live.

## 8. MCP (`.mcp.json`)

From `ir.Capability` with `Kind == "mcp"`. Shape (verify against Claude `.mcp.json`):

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "type": "stdio"
    }
  }
}
```

Map `Transport` → `type`, `Command`/`Args` straight through. Deterministic key ordering
(sort server names). `${NAME}` in args is left verbatim for Claude env substitution.

## 9. Settings + gate hooks (`.claude/settings.json`)

Two responsibilities:

1. **Permissions** — from agent `permissions:` values, mapped to Claude's permission
   model (verify schema). For MVP, a conservative default plus the union of needed tools.
2. **Blocking gates (hook-enforced tier, §12)** — for each gate with `Behavior ==
   "blocking"`, emit a hook entry that runs the gate script and **blocks on non-zero
   exit** (Claude hooks block on exit code 2 per §11 — verify). Hook event selection
   (e.g. `PostToolUse`/`Stop`/a custom point) must be verified; the gate semantics are
   "run script at the right point and block on failure".

Because precise hook wiring depends on host specifics, isolate it behind a small
`gateHook(g ir.Gate) hookEntry` function and pin its output in goldens. If a blocking gate
cannot be faithfully enforced by available hook events, emit `AF3xx` documenting the
advisory fallback (runbook still instructs the retry).

## 10. Capability negotiation (`capabilities.go`, §11/§12)

```go
func (b claude) Capabilities() map[binding.Capability]bool
```

Declare the `claude-code` row of the §11 matrix. Then, during `Emit`, **diff program needs
vs capabilities** and emit `AF3xx` **warnings** for advisory-only features the program
relies on:

| Need in program | Claude support | Diagnostic |
|-----------------|----------------|------------|
| `parallel` spawn | advisory (Task) | `AF300` "parallel spawn is advisory on claude-code; subagents may run sequentially" |
| loop/`repeat` `max` bound | advisory counter | `AF301` "loop bound is advisory; the host is instructed to self-count" |
| output protocol parse | advisory | `AF302` "output parsing is advisory; malformed output triggers re-invoke per runbook" |
| blocking gate where hook unavailable | fallback | `AF303` "gate '<name>' falls back to advisory; no enforcing hook" |

Define `binding.Capability` constants (the M0 type is a bare `string`; add named consts
here or in `binding`). Add `AF300`–`AF303` to the catalog. Warnings, not errors — `af
build` still succeeds.

## 11. DOT emitter (`internal/binding/dot/dot.go`)

```go
func Emit(p ir.Program) []byte
```

Render the resolved flow graph as Graphviz DOT for `af graph`:

- One node per control label (prefixed subflow labels included, e.g.
  `code_review.build`).
- Edges: sequential `->`, branch case edges (labeled with case values), loop back-edges
  (labeled `max N`), gather edges from each parallel branch into the gather node, gate
  on-fail edges (labeled `retry`/`goto` → target).
- Deterministic node/edge ordering (follow `ir.Flow.Order`).
- Not part of `.claude/` output; emitted to stdout by `af graph` (M8).

## 12. Registration

```go
func init() { binding.Register(claude{}) }
```

Ensure the package is imported for its side effect from `cmd/af` (blank import) so
`binding.Get("claude-code")` resolves. `Name()` returns `"claude-code"`.

## 13. Testing

Goldens: `internal/binding/claude/testdata/` mirroring the emitted tree (one golden file
per emitted path) for `review.af`. Test compares `emit.FS` paths + contents to goldens
with a `-update` flag.

### 13.1 FS snapshot tests

- Full tree for `review.af` matches goldens (paths + bytes).
- `reviewer.md` contains an `agentflow-output` block with the `out:` line and the three
  enum members.
- `ship.md` runbook: parses `code_review` `out:` for the branch; loop on `review != revise`
  (max 3, advisory); gate `on-fail: retry` references control label `build`.
- `.mcp.json` has the `github` server with `npx` command + args.
- `settings.json` has the `quality` gate hook (blocking) — pinned via `gateHook` golden.

### 13.2 Negotiation tests

- A program using `parallel` → `AF300` present.
- A program with a blocking gate but no enforceable hook path (simulate) → `AF303`.

### 13.3 Model mapping tests

- `(anthropic, opus)` → expected id; unknown alias → `AF3xx` + verbatim fallback.

### 13.4 DOT test

- `af graph review.af` DOT includes prefixed subflow labels (`code_review.build`) and
  gather edges (lint/security/style → reviewer). (DOT unit-tested here; CLI wiring in M8.)

### 13.5 Manual smoke (documented, not automated)

- `af build review.af --target claude-code --out /tmp/x && cd /tmp/x` then `/ship
  TICKET-123` in Claude Code; confirm the command appears and runs. Record steps in the
  progress log; not a CI gate.

## 14. Acceptance criteria

- [ ] Golden FS snapshots for `review.af` pass (full tree).
- [ ] `reviewer.md` contains the `agentflow-output` block with `out:` line.
- [ ] `ship.md` runbook parses `code_review` out for branch, loops on `review != revise`,
      gate retry → control label `build`.
- [ ] `.mcp.json` and `settings.json` generated with deterministic ordering.
- [ ] `AF3xx` advisory warnings emitted for parallel/loop/output-protocol/gate fallbacks.
- [ ] `binding.Get("claude-code")` returns the binding; `Capabilities()` matches §11.
- [ ] DOT emitter outputs prefixed labels + gather edges.
- [ ] No runbook logic duplicated in the binding (assembly only).

## 15. Commit plan

| # | Commit | Contents |
|---|--------|----------|
| 1 | `binding/claude: vocabulary + model id mapping` | `vocabulary.go`, `models.go` + tests |
| 2 | `binding/claude: agent + command file assembly` | `claude.go` agent/command emit |
| 3 | `binding/claude: mcp + settings + gate hooks` | `.mcp.json`, `settings.json`, hooks |
| 4 | `binding/claude: capability negotiation (AF3xx)` | `capabilities.go` + tests |
| 5 | `binding/dot: DOT emitter` | `dot/dot.go` + test |
| 6 | `binding/claude: FS goldens for review.af` | `testdata/**` + snapshot test |

## 16. Risks & notes

- **Host-format drift is the #1 risk.** Hook event names, frontmatter keys, MCP tool
  naming, and model ids change. Verify against live docs (Context7/Anthropic) at
  implementation time and **confine every host fact to this package**. The render layer
  (M6) stays neutral, so a format change is a localized edit here + a regold.
- **Blocking-gate fidelity.** If Claude hooks cannot enforce a gate at the exact flow
  point, prefer an honest `AF303` advisory fallback over a misleading "blocking" claim
  (§12 honesty principle).
- **Thin assembler discipline.** Resist re-implementing any control-flow prose here; if
  something reads awkwardly, fix M6 + its golden, not the binding.
- **Determinism.** Sort MCP servers, tool lists, and settings keys; never iterate maps for
  output. FS paths are already sorted by `emit.FS`.
