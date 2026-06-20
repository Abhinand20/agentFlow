# M6 - Rendering Layer (IR to text)

- Milestone: M6
- Version: v0.1 (MVP) — Language Level A
- Status: Done
- Spec: [§9 Output protocol](../../spec/grammar.md#9-output-protocol-contract), [§4 Data model](../../spec/grammar.md#4-execution-and-data-model), [§12 Guarantees](../../spec/grammar.md#12-runtime-guarantees-by-target)

## Goal

Translate IR + resolved-flow constructs into the natural-language text that lands
in generated markdown: runbook steps, subagent prompts (with output protocol),
and neutral frontmatter values. Host-agnostic **"what to say"** layer; bindings
(M7/M10) handle **"where to put it"**.

## Scope

### In scope
- Step renderers for Level A kernel constructs.
- Output protocol injection (spec §9).
- Gate failure prose with explicit step targets (§7.4.1).
- Gather payload naming in prose (§4.8).
- Host `Vocabulary` hook.

### Out of scope (deferred)
- Host file layout (M7/M10).
- Multi-field output protocol / dotted branch (M12).

## Packages & files

- `internal/render/render.go`
- `internal/render/vocabulary.go`
- `internal/render/runbook.go`
- `internal/render/prompt.go`
- `internal/render/protocol.go`

## Tasks

- **`render.Vocabulary`:** `InvokeAgent`, `RunScript`, `SpawnParallel`, `ReadOutput(valueLabel)`,
  `ParseOutputProtocol`, `Arg(name)`, `GotoStep(label)`.
- **Sequence renderer:** numbered steps; pass previous step output in prose.
- **Branch renderer:** `branch <valueLabel> { case v -> ... }` — read `out:` enum via
  `ReadOutput(valueLabel)`; no dotted paths in v0.1.
- **Loop renderer:** `until <valueLabel> == v (max N)`; advisory bound wording per §12.
- **Repeat renderer:** do-while — render the body once, then "repeat the steps
  above until `<valueLabel> == v`, at most N times" (§4.7.1). Make explicit that
  the check happens **after** the body so the orchestrator never skips the first
  pass; same advisory `max` wording as loop.
- **Parallel/gather renderer:** spawn branches; pass gather payload keys explicitly
  ("outputs from lint, security, style") to gather step.
- **Gate renderer:** run script; on failure `GotoStep(onFailTarget)` per `halt|retry|goto|enter-loop`;
  never emit "bounce-back".
- **Prompt body:** use the agent's **already-resolved** prompt text from the model
  (M2 read any `prompt:`/`prompt-file:` path into `Prompt`); render never touches
  the filesystem, so file- and inline-sourced prompts render identically. `${NAME}`
  stays unexpanded for bindings.
- **Output protocol (§9):** append to agent prompts:
  - fence tag exactly `agentflow-output`
  - single line `out: <enum-member>`
  - list allowed values; instruct orchestrator parse algorithm §9.2
  - document retry/halt table §9.3 in runbook
- **`render.Document{ Frontmatter, Body }`** — deterministic ordering.

## Acceptance criteria

- Golden text snapshots per construct (seq/branch/loop/repeat/parallel/gate/protocol).
- `critic.af` repeat runbook: body rendered once, then "repeat until verdict == pass (max 3)".
- `reviewer` prompt golden includes `out:` protocol with `approve|revise|reject`.
- Runbook golden for [examples/review.af](../../examples/review.af): gate retry
  targets control label `build`; loop reads `review` `out:`; branch on `code_review`.

## Dependencies

- M5 (IR). Consumed by M7, M10.

## Risks / notes

- Runbook phrasing is runtime behavior for native tier (§12); pin with golden tests.
- All host-specific phrasing through `Vocabulary` only.
