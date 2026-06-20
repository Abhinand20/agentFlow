# M3 - Inline & Normalize

- Milestone: M3
- Version: v0.1 (MVP) — Language Level A
- Status: Done
- Spec: [§4 Execution/data model](../../spec/grammar.md#4-execution-and-data-model), [§13 Resolution](../../spec/grammar.md#13-resolution-and-lowering)

## Goal

Produce the **Resolved flow**: a tree of kernel-only constructs plus the data-model
artifacts (control labels, value labels, latest-output edges, gather payloads, return bindings) that
validation, IR, render, and DOT all consume. This is the one mandatory transform.

## Scope

### In scope
- Expand bare subflow references (nesting) with label prefixing.
- Cycle guard across flow nesting.
- Control/value label assignment (§4.1).
- Return binding wiring (§4.4).
- Gather payload construction (§4.8).
- Sequential latest-output edges (§4.5).

### Out of scope (deferred)
- Pattern-call substitution / `each` expansion (M9).

## Packages & files

- `internal/flowgraph/resolve.go`
- `internal/flowgraph/types.go`

## Tasks

- Inline subflow steps: splice callee body; prefix labels (`code_review.review`).
- Detect recursive nesting -> error (not stack overflow).
- Assign control labels and value labels; default = declaration name; explicit
  `as` writes to the named value label.
- Detect ambiguous duplicate implicit labels (also checked in M4 `AF208`).
- **Return binding:** subflow value output = inner flow's `return:` value label.
  When `return:` was defaulted (spec §4.4 Rule 0), resolve it here to the terminal
  producer's value label and fail with `AF209` if that step has no typed/text
  output or is a branch (ambiguous).
- **`repeat` normalization:** treat `repeat { body } until (cond, max N)` like
  `loop` for label prefixing (`repeat.generate`, `repeat.critic`), but record that
  the condition is evaluated **after** each iteration (body runs >= 1). Value
  labels read by `cond` may be written only inside the body; an unwritten label
  referenced in the body resolves to empty (spec §4.7.1).
- **Gather payload:** for `parallel { a b c } gather g`, build
  `{ a: latestOutput(a), b: ..., c: ... }` passed to gather step metadata.
- **Sequential edges:** record `prevStep -> nextStep` latest-output dependency for
  render/IR (no `it` in v0.1 source).
- Emit `flowgraph.Resolved{ Entry, Tree, Instances map[string]StepInstance }`.

## Data shapes / snippets

```go
type StepInstance struct {
    ControlLabel string
    ValueLabel   string
    Decl        string // agent/gate/flow name
    Kind        StepKind
    OutEnum     []string // if agent with enum out
    GatherPayload map[string]string // branch control label -> producing value label
}
type Resolved struct {
    Entry     string
    Tree      Node
    Instances map[string]*StepInstance
}
```

## Acceptance criteria

- Resolved-tree snapshot for [examples/review.af](../../examples/review.af).
- Subflow `code_review` inlined under `ship` with prefixed labels.
- Gather payload on `review` lists `lint`, `security`, `style`.
- `return: review` exposes `Verdict` on subflow value `code_review`.
- Recursive-flow cycle detected.
- `critic.af`: `repeat` body labels prefixed; `verdict` written inside body and
  read by `until`.
- `pipeline.af`: defaulted `return` resolves to `edit`.

## Dependencies

- M2 (model).

## Risks / notes

- Label prefix scheme must be stable once published (spec §16 open question).
- M9 plugs pattern-call inlining into this package without changing downstream.
