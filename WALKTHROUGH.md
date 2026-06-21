# AgentFlow MVP Walkthrough

> A friendly, high-level tour of what AgentFlow is, the milestones that build the
> MVP, what each component does, how they fit together, and where it goes after MVP.
>
> For the formal language definition see [spec/grammar.md](spec/grammar.md). For the
> architecture map see [OVERVIEW.md](OVERVIEW.md). For milestone detail see
> [plans/](plans/). This doc is the plain-English bridge between them.

---

## 1. What AgentFlow is (in one minute)

You write a single `.af` file describing a team of AI agents — who they are, what
tools they have, what typed output they produce, and how they hand work to each
other. AgentFlow **compiles** that file into the native configuration your host
needs, including a slash command like `/ship` that runs the whole team. Today the
**Cursor** target works end to end through the `af` CLI (`af build --target cursor`
emits native subagents, a slash command, and `mcp.json`); **Claude Code** is the
next binding.

The key mental model:

> **AgentFlow is a compiler, not a runtime.** It never calls a model or runs an
> agent. It generates instructions and config. The *host* (Claude Code) does the
> actual orchestration at runtime.

The one big idea behind the language:

> **Everything is a flow.** An agent is an atomic flow (`In -> Out`). A bigger flow
> is just a composition of smaller flows. Common patterns (supervisor, fan-out,
> critic loops) are *compositions*, not special keywords.

A tiny taste from [examples/review.af](examples/review.af):

```text
entry flow ship {
  on: "/ship"
  in: Ticket
  out: Decision
  code_review
  branch code_review {
    case approve -> deploy
    case revise  -> notify_author
    case reject  -> notify_author
  }
}
```

This says: when someone types `/ship`, run the `code_review` subflow, then branch
on its verdict — ship it, or notify the author.

---

## 2. The compiler pipeline (the spine everything hangs off)

Every milestone is a stage in one straight-line pipeline. Source text goes in the
left, host files come out the right:

```mermaid
flowchart LR
  src[".af source"] --> parse["parse (M1)"]
  parse --> resolve["resolve (M2)"]
  resolve --> inline["inline / normalize (M3)"]
  inline --> validate["validate (M4)"]
  validate --> ir["IR (M5)"]
  ir --> render["render (M6)"]
  render --> bind["host binding (M10 Cursor ✓ / M7 Claude)"]
  bind --> files["host config + markdown"]
  cli["af CLI (M8) ✓"] -.drives.-> parse
```

Each stage only consumes the output of the one before it. That's why the
milestones are numbered the way they are — they build the pipeline front to back.

**Where the build stands today:** the whole pipeline is **wired end to end for
Cursor.** M0–M6 (parse through render) are **done**, **M10 Cursor binding** is
**done** (native subagents, command, `mcp.json`), and **M8 CLI** is **done** —
`af validate`, `af graph`, and `af build --target cursor` run `examples/review.af`
through to a working `.cursor/` config, with golden + end-to-end tests green. The
remaining MVP item is **M7 Claude binding** (`af build --target claude-code`), which
reuses the same IR and render layers.

---

## 3. The MVP milestones (M0–M8)

The MVP is "Level A" of the language — the executable v0.1 subset. Here is each
milestone, what it builds, and why it matters. Status reflects the current repo.

### M0 — Foundations · ✅ Done

**What it builds:** the plumbing every later stage reuses — structured
diagnostics, an in-memory output file set, the binding interface, and a CLI
skeleton.

- `internal/diag/` — every compiler message is a structured `Diagnostic` with a
  code (`AF000`, `AF110`, …), severity, message, and source position. No raw
  `panic`; every pass returns diagnostics.
- `internal/emit/` — an ordered, in-memory set of files the binding will write,
  with deterministic sorted paths and a `Flush(dir)` to disk.
- `internal/binding/` — the `Binding` interface (`Name`, `Capabilities`, `Emit`)
  that hosts implement later.

**Why first:** nothing here is AgentFlow-specific logic, but everything downstream
depends on it.

### M1 — Lexer & Parser · ✅ Done

**What it builds:** turns `.af` text into an AST (abstract syntax tree) using the
`participle` parser library — with positioned errors and no panics on bad input.

- `internal/parser/` — lexer + parser. Entry point: `parser.Parse(filename, src)`.
- `internal/ast/` — the AST node types and their JSON snapshot form.

**Key trait:** the grammar is deliberately loose. Agents and gates are generic
"field bags," so unknown fields can be preserved rather than rejected. Note: there
are **no semicolons** — one step per line.

### M2 — Resolver & Semantic Model · ✅ Done

**What it builds:** lowers the loose AST into a typed **semantic model** with
symbol tables — interpreting each block according to its kind (agent, gate, type,
flow).

- `internal/sema/` — the resolver. Entry point: `sema.Resolve(ast, srcDir)`.
- `internal/model/` — the typed `Program`: maps of capabilities, types, agents,
  gates, flows, plus the entry flow.

This is where:
- models/providers get resolved (e.g. `opus` → the anthropic provider) — spec §8,
- prompt files (`prompt-file: "..."`) get read from disk,
- gate failure policy (`on-fail: retry`, `on-fail-target: build`) is parsed,
- flow `return:` bindings are recorded,
- Level B syntax (the v0.2 stuff) is rejected with `AF150`.

Unknown fields **warn but are kept** for forward compatibility.

### M3 — Inline & Normalize · ✅ Done

**What it builds:** the **Resolved flow** — a fully expanded tree of kernel-only
constructs plus the "data identity" metadata that the rest of the pipeline needs.

- `internal/flowgraph/` — entry point: `flowgraph.Resolve(program)`.

This single mandatory transform:
- **inlines subflows** (e.g. `code_review` gets expanded inline) and **prefixes
  labels** so names stay unique,
- assigns **control labels** (a stable runbook name per step) and **value labels**
  (`reviewer as review` names an output slot),
- builds **gather payloads** for `parallel { … } gather`,
- wires **return bindings** (which value a flow actually returns),
- detects cycles (`AF212`).

**Why it matters:** validation, IR, rendering, and the DOT graph all read this
resolved tree — not the raw AST. It's the load-bearing representation.

### M4 — Validation · ✅ Done

**What it builds:** the v0.1 rule set (`AF200`–`AF211`) over the resolved flow,
producing clear, actionable diagnostics — e.g. branch cases that don't match an
enum, missing producers, unreachable steps. Runs *after* inlining so it sees the
fully expanded graph.

- `internal/validate/` — rule engine over resolved flow + model.

### M5 — IR (Intermediate Representation) · ✅ Done

**What it builds:** a normalized, **binding-agnostic** IR with deterministic JSON.

- `internal/ir/` — `FromResolved`, JSON marshal/unmarshal, golden fixtures.

The IR is the **stable contract** between the front end and the back end.
Everything downstream (render, every host binding) consumes the IR — never the AST
or model. That decoupling is what lets new hosts be added later without touching
the parser.

### M6 — Rendering Layer · ✅ Done

**What it builds:** the host-agnostic "what to say" layer — turning IR into the
natural-language text that lands in generated markdown:

- runbook steps (the ordered instructions the host agent follows),
- subagent prompts, including the **output protocol** (the fenced
  `agentflow-output` block agents must emit, spec §9),
- neutral frontmatter values.

- `internal/render/` — `Vocabulary` interface, `runbook.go`, `prompt.go`, `protocol.go`.

Render decides *what to say*; bindings decide *where to put it*.

### M10 — Cursor Binding · ✅ Done (native subagents)

**What it builds:** assembles render output into `.cursor/commands/`, `.cursor/agents/*.md`
(native Cursor subagents), and `.cursor/mcp.json`. Each agent becomes a subagent file with
`name`/`description`/`model` (+ optional `readonly`) frontmatter; blocking gates use advisory
fallbacks with `AF3xx` warnings, and parallel spawn maps to advisory parallel Task wording.
(The lean first cut shipped `.cursor/rules/*.mdc`; it has since been migrated to native
subagents.)

- `internal/binding/cursor/` — **done**; wired to the CLI via `af build --target cursor`.

It also honors `use cursor { kind: model-provider, models: [...] }`, mapping symbolic
aliases (e.g. `opus-4-8`, `composer-2-5-fast`) to host model ids in the subagent
frontmatter (spec §8).

**Remaining polish (not blocking):** `.cursor/hooks.json` for hard gates and a shared
negotiation framework — see
[implementation-plans/2026-06-20-m10-cursor-subagents.md](implementation-plans/2026-06-20-m10-cursor-subagents.md).

### M8 — CLI & End-to-End · ✅ Done (Cursor path)

**What it builds:** wires the whole pipeline behind the `af` binary and proves the
MVP works end to end.

- `cmd/af/` — `af validate`, `af graph`, `af build --target cursor` (plus
  `--emit-ir` and `--out`); bindings self-register via `cmd/af/bindings.go`.
- `internal/pipeline.Compile(path)` — the single entry point running parse →
  resolve → inline → validate → IR, shared by the CLI and tests.

**MVP "definition of done" — met for Cursor** (all pass on `examples/review.af`):
- `af validate` — zero errors,
- `af graph` — DOT with prefixed subflow labels and gather edges,
- `af build --target cursor` — produces the `.cursor/` config,
- the runbook implements the §9 output protocol and gate retry-to-`build`,
- the E2E test is green (in-process golden tree + real-binary CLI contract), with
  all goldens tied to the canonical file.

The one remaining target is `af build --target claude-code`, which lands with M7.

### M7 — Claude Code Binding · ⏳ Planned (next)

**What it builds:** assembles the render output into a working `.claude/`
directory:

- `internal/binding/claude/` — `.claude/commands/ship.md`, agent files,
  `.mcp.json`, settings/hooks.

Typing `/ship` then runs the flow via the native runbook (best-effort control
flow), with **hook-enforced blocking gates** where the host supports them. Because
render (M6) and IR (M5) are host-agnostic, M7 is a new *binding* consuming the same
IR — no front-end changes. Cursor (M10) shipped first; Claude completes the MVP's
original primary target.

---

## 4. How it all fits together (the worked example)

Walk `examples/review.af` through the finished pipeline to see the pieces connect:

1. **Author** writes the `.af`: agents (`build`, `lint`, `reviewer`, `deploy`…), a
   `quality` gate, a `code_review` flow, and an `entry flow ship` with `on: "/ship"`.
2. **Parse (M1)** → AST. (Level B syntax would parse but get rejected later.)
3. **Resolve (M2)** → semantic model: `opus`/`sonnet`/`haiku` resolve to the
   anthropic provider, `reviewer`'s prompt is read from `prompts/reviewer.md`, the
   gate's `on-fail: retry → build` policy is parsed.
4. **Inline (M3)** → resolved flow: `code_review` is inlined into `ship`, labels
   are assigned, the `parallel { lint, security, style } gather reviewer as review`
   produces a gather payload, and the `loop (until review != revise, max 3)` is
   wired.
5. **Validate (M4)** → confirms branch cases (`approve`/`revise`/`reject`) match
   the `Verdict` enum, every reference resolves, etc.
6. **IR (M5)** → a JSON description of agents, the flow graph, and data-flow
   metadata.
7. **Render (M6)** → runbook prose + agent prompts with `agentflow-output` blocks.
8. **Bind (M10 Cursor today; M7 Claude next)** → host config (`.cursor/` or, later,
   `.claude/`).
9. **CLI (M8)** → `af build --target cursor` ties it together.

**At runtime (outside AgentFlow):** a developer types `/ship TICKET-123` in the
host (Cursor today). The host follows the generated runbook: runs `build`, runs the `quality`
gate (retrying back to `build` on failure), fans out the three reviewers in
parallel, gathers a verdict, loops up to 3 times while the verdict is `revise`,
then branches to `deploy` or `notify_author`. Gates are enforced by hooks where
supported (advisory on Cursor today; hook-enforced once the Claude binding lands).

### The same kernel, many shapes

The four example files show that common architectures are just compositions of the
same small kernel — no new keywords:

| Architecture | Example | Kernel used |
|--------------|---------|-------------|
| Sequential pipeline | [examples/pipeline.af](examples/pipeline.af) | `a -> b -> c`, default `return:` |
| Supervisor / worker fan-out | [examples/research.af](examples/research.af) | `parallel { … } gather` |
| Generator / critic | [examples/critic.af](examples/critic.af) | `repeat { … } until` |
| Review + ship (all of it) | [examples/review.af](examples/review.af) | subflow, gate, branch, loop |
| CL review (dogfooded, Cursor models) | [examples/cl-review.af](examples/cl-review.af) | `a -> b`, `use cursor` model-provider, `prompt-file` |

`review.af` is the **canonical golden fixture**: the spec, all snapshot tests, and
the MVP acceptance criteria reference it. Change it → re-gold the tests.

---

## 5. The data-identity backbone (why this works at all)

Control flow is useless unless you know *what value* each step produces and *who
receives it*. The MVP makes this explicit (spec §4), and it's why M3 exists as its
own milestone:

- **Control labels** — a stable runbook name for each agent/gate/subflow occurrence.
- **Value labels** — `ref as value` names the latest-output slot used by branches,
  loops, and `return:`.
- **Latest output** — agents with `out: Enum` emit a parsed enum; conditions are
  `review == approve`, never free-form expressions (composition, not computation).
- **Flow I/O** — `in:` / `out:` declare types; `return: valueLabel` binds the
  flow's output explicitly (no "last step silently wins").
- **Gather payload** — parallel branches produce a labeled bundle passed to the
  gather step.
- **Output protocol** — agents end with a fenced `agentflow-output` block; a parse
  failure retries, then halts.

---

## 6. After the MVP (future extensions)

With the Cursor path working end to end, AgentFlow grows along a few axes. There's
one grammar with three semantic levels: **A** (MVP, today), **B** (parses now,
enabled in M9), **C** (post-MVP new syntax).

### Near-term extensions

These build directly on the working compiler and are the immediate priorities:

- **Claude Code binding (M7)** — `af build --target claude-code` emits a `.claude/`
  tree (commands, agents, `.mcp.json`, settings/hooks) from the *same* IR and
  render layers Cursor already uses, adding **hook-enforced blocking gates**. This
  is a new binding, not a front-end change.
- **Config import & round-trip (M17)** — *rebuild AgentFlow configs from
  pre-defined agents.* Today the compiler is one-way (`.af` → host). `af import
  --from cursor <dir>` (and `--from claude` after M7) parses an existing
  `.cursor/`/`.claude/` tree back into a `.af` file via the IR, so teams that
  already hand-wrote subagents get an editable AgentFlow source instead of a blank
  file — and a build can be diffed against its source. See
  [plans/post-mvp/17-config-import-and-roundtrip.md](plans/post-mvp/17-config-import-and-roundtrip.md).
- **Hard gates on Cursor** — emit `.cursor/hooks.json` so blocking gates stop
  being advisory, plus a shared `internal/binding/capability.go` negotiation
  framework reused by every binding.
- **Multi-host build** — one `af build` pass that emits several targets and a
  build-notes summary of each host's advisory fallbacks.

### Roadmap

| # | Milestone | Version | What it adds |
|---|-----------|---------|--------------|
| M7 | Claude Code Binding | v0.1 | `af build --target claude-code`, hook-enforced gates |
| M9 | Abstraction & std/patterns | v0.2 (Level B) | flow parameters, calls, `each`, `it`, a `std.patterns` library |
| M11 | Registry, Formatter, Diagnostics | v0.2 | a component registry, `af fmt`, richer diagnostics |
| M12 | Records & Multi-output | v0.3 (Level C) | record types, agents with multiple outputs, dotted branches |
| M13 | Policies & Metering | v0.4 | execution policies and usage metering |
| M14 | Plan IR & Simulator | v0.5 | a Plan IR, a simulator, and `af test` |
| M15 | SDK Runtime | v0.5+ | opt-in `--runtime sdk`: a generated deterministic orchestrator with *hard* control flow, gates, parallelism, and loop bounds |
| M16 | LSP | v0.5+ | editor support — diagnostics, go-to-def, formatting |
| M17 | Config Import & Round-Trip | v0.6+ | `af import` reconstructs `.af` from a host config; round-trip diffing |

**The throughline — determinism is a spectrum by target.** The Cursor binding
follows the runbook *best-effort* with advisory gates; the Claude binding (M7)
adds hook-enforced gates; the later SDK runtime (M15) generates a program that
drives the flow *deterministically*. Same `.af` source; stronger guarantees as the
target improves:

| Capability | Cursor (shipped) | Claude Code (M7) | SDK (M15) |
|------------|------------------|------------------|-----------|
| Slash command | yes | yes | CLI |
| Blocking gates | advisory | hooks | hard |
| Deterministic control flow | no | no | yes |

**Why the architecture extends cleanly:** the IR (M5) is the stable, binding-agnostic
contract. New hosts (Cursor, SDK) are new *bindings* that consume the same IR;
new language features (Level B/C) extend the front end and IR. The parser never
needs to know about hosts, and hosts never need to know about syntax.

---

## 7. Quick reference

**Pipeline:** parse → resolve → inline → validate → IR → render → bind.

**Status:** MVP **working end to end for Cursor** (M0–M6, M8, M10 done) · M7 Claude
binding is the next target · M9+ / M17 are post-MVP.

**Code map:**

| Stage | Package | Status |
|-------|---------|--------|
| Diagnostics / emit / binding iface | `internal/diag`, `internal/emit`, `internal/binding` | ✅ M0 |
| Parser / AST | `internal/parser`, `internal/ast` | ✅ M1 |
| Resolver / model | `internal/sema`, `internal/model` | ✅ M2 |
| Inline / normalize | `internal/flowgraph` | ✅ M3 |
| Validation | `internal/validate` | ✅ M4 |
| IR | `internal/ir` | ✅ M5 |
| Render | `internal/render` | ✅ M6 |
| DOT graph | `internal/dot` | ✅ M8 |
| Cursor binding | `internal/binding/cursor` | ✅ M10 |
| CLI / pipeline | `cmd/af`, `internal/pipeline` | ✅ M8 |
| Claude binding | `internal/binding/claude` | ⏳ M7 |

**Docs:** [README.md](README.md) (intro) · [OVERVIEW.md](OVERVIEW.md) (architecture) ·
[spec/grammar.md](spec/grammar.md) (language) · [plans/](plans/) (milestone detail) ·
[examples/review.af](examples/review.af) (canonical program).
