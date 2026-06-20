# M3 — Inline & Normalize (execution plan)

- **Date:** 2026-06-19
- **Milestone:** M3
- **Design spec:** [plans/mvp/03-inline-and-normalize.md](../plans/mvp/03-inline-and-normalize.md)
- **Language spec:** [§4](../spec/grammar.md#4-execution-and-data-model), [§13](../spec/grammar.md#13-resolution-and-lowering)
- **Status:** Planned
- **Depends on:** M2 (`model.Program`)
- **Blocks:** M4 (validation runs on the resolved graph), M5 (IR), M7 (DOT)

## 1. Goal

Produce the **Resolved flow**: a tree of kernel-only constructs annotated with the
data-model artifacts every downstream pass needs:

- **control labels** (unique per flow scope; runbook/gate targets) — §4.1
- **value labels** (latest-output slots read by conditions/`return`/branches) — §4.1
- **subflow inlining** with stable label prefixing + recursion guard — §4.1
- **return binding** resolution, including the **default terminal producer** (Rule 0) — §4.4
- **gather payloads** (branch-control-label → value bundle) — §4.8
- **sequential latest-output edges** — §4.5
- **repeat normalization** (do-while; body runs ≥ 1; condition after) — §4.7.1

This is the one mandatory structural transform. After M3, M4/M5/M6/M7 operate purely on
`flowgraph.Resolved` + `model.Program` and never re-derive flow structure.

> M3 detects some structural defects (recursion, missing default return target). It
> emits the diagnostics it is uniquely positioned to find (`AF209` default-return
> resolution, recursion). The full validation rule set is M4. Where a check is shared
> (e.g. ambiguous labels), M3 computes the data and M4 reports — see §6.

## 2. Deliverables

- `internal/flowgraph/types.go` — `Resolved`, `Node`, `StepInstance`, label types.
- `internal/flowgraph/resolve.go` — `Resolve(prog *model.Program) (*Resolved, diag.Diagnostics)`.
- `internal/flowgraph/labels.go` — label allocation + prefixing helpers.
- Resolved-tree snapshot goldens in `internal/flowgraph/testdata/`.

## 3. Data shapes (`internal/flowgraph/types.go`)

```go
package flowgraph

type StepKind int
const (
    KindAgent StepKind = iota
    KindGate
    KindTerminal // done/fail
    // composite kinds carried on Node, not StepInstance:
)

// StepInstance is one runnable occurrence of an agent/gate/terminal in the
// fully-inlined graph (subflows are gone — their bodies are spliced in).
type StepInstance struct {
    ControlLabel string   // unique within the entry flow's inlined scope
    ValueLabel   string   // output slot name; "" if step has no output (gate/terminal)
    Decl         string   // agent/gate name (after inlining, the original decl)
    Kind         StepKind
    OutType      string   // enum type name or "text" or ""
    OutEnum      []string // enum members if OutType is an enum; nil otherwise
    Pos          lexer.Position
}

// Node is the kernel control-flow tree. Exactly one of the pointers is set.
type Node struct {
    Seq      *SeqNode
    Step     *StepInstance   // leaf (agent/gate/terminal occurrence)
    Branch   *BranchNode
    Loop     *LoopNode
    Parallel *ParallelNode
}

type SeqNode struct{ Steps []*Node }

type BranchNode struct {
    ValueLabel string        // value label read (already prefixed if from subflow)
    OutEnum    []string      // enum members of that value label's type
    Cases      []*BranchCase
}
type BranchCase struct {
    Values []string // enum members this case matches
    Body   *Node
}

type LoopNode struct {
    DoWhile    bool   // repeat => true
    Cond       *Cond  // nil only if (illegal) no until — guarded by AF205 in M4
    Max        int
    HasMax     bool
    Body       *Node
    ScopePrefix string // e.g. "loop" / "repeat" for child labels
}
type Cond struct{ ValueLabel string; Op string; Enum string } // Op "==" | "!="

type ParallelNode struct {
    Branches    []*Node       // each typically a single Step
    Gather      *StepInstance // gather agent occurrence; nil if absent
    GatherBody  *Node         // the gather as a node (for label consistency)
    Payload     GatherPayload
}
type GatherPayload struct {
    GatherControlLabel string
    // ordered: branch control label -> producing value label
    Branches []GatherBranch
}
type GatherBranch struct{ ControlLabel, ValueLabel string }

type Resolved struct {
    Entry     string                   // entry flow name
    EntryIn   string                   // entry flow input type
    Tree      *Node                    // fully inlined kernel tree
    Instances map[string]*StepInstance // by control label
    Order     []string                 // control labels in document/runbook order
    // Return binding of the ENTRY flow and each (pre-inline) flow, post-defaulting:
    Returns   map[string]ReturnBinding // flow name -> binding
}

type ReturnBinding struct {
    Flow        string
    ValueLabel  string // resolved value label (after Rule 0 defaulting)
    OutType     string // resolved/declared output type
    Defaulted   bool   // true if from Rule 0
    BranchTerminal bool // true if flow ends in a branch with no return (§4.4 Rule 2)
}
```

> Ordered slices (`Order`, `GatherPayload.Branches`) instead of maps wherever output
> ordering matters — snapshots and runbooks must be deterministic.

## 4. Algorithm

`Resolve` builds the tree for the **entry flow**, inlining subflows on the way. (Non-entry
flows are also normalized individually for validation/snapshot purposes, but the canonical
`Resolved.Tree` is the entry flow with everything spliced in.)

### 4.1 Inlining subflows (§4.1)

Walk the entry flow body. When a step reference resolves to a **flow** (not agent/gate),
splice that flow's normalized body in place of the reference:

- Maintain a `stack []string` of flow names currently being inlined. Before descending
  into flow `F`, if `F ∈ stack` → emit `AF205`-adjacent recursion error. **Decision: use
  a dedicated code `AF212` "recursive flow nesting: a -> b -> a"** (cycle in the *flow
  call* graph, distinct from `AF205` which is about *unbounded loops*). Add to catalog.
  Abort inlining that branch (do not stack-overflow), continue elsewhere.
- Prefix every inner control/value label with `F.` (e.g. `code_review.build`,
  `code_review.review`). The prefix scheme is `<flowname> + "." + innerLabel`, applied
  recursively (nested subflows compose: `a.b.step`). This is spec §16's open question —
  **decide and pin: dot-separated, left-to-right, recursive.** Document in `labels.go`.
- The subflow's **value output** (its return binding, §4.4) becomes the value label that
  the parent sees for the subflow reference. Concretely: a parent step `code_review`
  (with no `as`) exposes value label `code_review`, whose latest output is the inner
  return value label `code_review.review`. Record this mapping so a parent
  `branch code_review { … }` reads the right enum.

### 4.2 Label allocation (§4.1)

Within a single flow scope (before prefixing):

- Default **control label** = declaration name.
- Default **value label** = control label, for agents/subflows that produce output;
  gates/terminals get value label `""`.
- `ref as name` keeps the control label = decl name but sets value label = `name`.
- **Duplicate implicit control labels**: if two instances in the same scope share a decl
  name and neither uses `as`, allocate stable occurrence labels: first stays `reviewer`,
  second becomes `reviewer#2`, etc. (deterministic, document the scheme). Record that the
  duplication was *implicit* so M4 can emit `AF208`. (Spec §4.1: "the normalizer then
  assigns stable occurrence labels … `AF208` reports ambiguous implicit labels.")
  - Note the golden program reuses `build`/`quality`/`reviewer` across the pre-loop
    sequence and the loop body — but those are in **different scopes** (loop body is its
    own scope with a `loop.` prefix), so they do **not** collide. Verify the scoping rule:
    a loop/repeat body is a nested scope; labels there get the loop prefix and are
    de-duplicated independently.

### 4.3 Value-label lineage (§4.1, §4.7)

A value label may be **rewritten** across a sequential/loop lineage if every writer has
the same output type. Track, per value label, the set of writer output types:

- `review` is written by `gather reviewer as review` (type `Verdict`) and by the loop
  body `reviewer as review` (type `Verdict`) — same type, OK. The branch in `ship` reads
  the subflow's `review` (via `code_review` return), consistent.
- If two writers disagree on type → record a conflict for M4. **Decision: new code
  `AF213` "value label 'x' written with conflicting output types: A vs B".** Add to
  catalog. (Not currently in the M4 table; add it there too.)

### 4.4 Return binding + default (Rule 0, §4.4) — **the subtle one**

For each flow, compute its `ReturnBinding`:

1. If `Flow.ReturnExplicit`: `ValueLabel = Flow.Return`. The label must exist as a value
   label in the (normalized, pre-inline) flow scope — if not, that's `AF209` (reported in
   M4; M3 records existence). `OutType` = the type of that value label's latest writer.
2. If **not** explicit and the flow body is **branch-terminal** (last structural item is a
   `branch` with no trailing producer): `BranchTerminal = true`, `ValueLabel = ""`. The
   per-leaf output match is `AF210` (M4).
3. If **not** explicit and not branch-terminal (sequence/loop/repeat): apply **Rule 0**.
   The flow returns the **terminal producer** = the last step instance in the flow's
   top-level sequence that produces a typed/text output.
   - Determine the "last step instance": for a trailing `loop`/`repeat`, the terminal
     producer is the loop body's last producing step (e.g. `pipeline.af` ends in `edit`;
     `critic.af` ends in a `repeat` whose last body step is `critic` — but `critic.af`
     sets `return: draft` **explicitly** precisely because the terminal producer would be
     the critic, not the draft).
   - If the terminal producer has no typed/text output (e.g. a gate) → `Defaulted = true`
     but mark **unresolvable**; M4 emits `AF209` "missing default return". M3 records the
     reason.
   - Otherwise `ValueLabel = <terminal producer value label>`, `OutType = its type`,
     `Defaulted = true`.

Worked examples (must match goldens):

| Flow | Return | OutType | Source |
|------|--------|---------|--------|
| `code_review` (review.af) | `review` | `Verdict` | explicit |
| `ship` (review.af) | `""` (branch-terminal) | `Decision` | Rule 2 (leaves `deploy`/`notify_author`) |
| `content` (pipeline.af) | `edit` | `text` | Rule 0 default (terminal producer) |
| `sql` (critic.af) | `draft` | `text` | explicit (terminal producer is critic, not draft) |
| `research` (research.af) | `report` | `Report` | explicit? No — `gather … as report`; **default** would pick the gather producer `report`. It is the terminal producer, so Rule 0 resolves to `report`. Verify against the fixture (research.af has `out: Report` but no `return:`). |
| `docs` (docs.af) | `draft` | `text` | Rule 0 default (terminal producer `draft`) |

> Action item for the implementer: confirm `research.af`'s gather step value label is the
> terminal producer for Rule 0. The gather agent `synthesize as report` is the last
> producing instance, so `Return = report`. Pin this in the golden.

### 4.5 Gather payload (§4.8)

For each `parallel { a b c } gather g`:

- `Payload.Branches` = ordered `[{a, valueLabel(a)}, {b, …}, {c, …}]` keyed by branch
  **control label** → its producing **value label** (text if no enum out).
- The gather step instance `g` receives, in render prose: (1) the sequential context =
  latest output of the step immediately *before* the `parallel` (e.g. `quality`/`build`
  in review.af — note `quality` is a gate with no output, so the relevant sequential
  predecessor producer is `build`), and (2) the payload bundle.
- Record `Payload.GatherControlLabel = g`'s control label.

### 4.6 Sequential edges (§4.5)

For each sequence, record `prevProducer -> next` latest-output dependency. Only
**producers** (agents/subflows with output) are sources; gates/terminals are skipped as
sources but remain in the runbook order. Store these as part of `SeqNode` ordering plus an
explicit `Instances` predecessor lookup if render needs "the output from step N". Simplest:
render can recompute from `Order`; but provide a helper `PrevProducer(controlLabel) string`
to avoid duplicating the skip-gate logic in M6.

### 4.7 Repeat normalization (§4.7.1)

- `repeat { body } until (cond, max N)` → `LoopNode{DoWhile: true, ScopePrefix: "repeat"}`.
- Body labels prefixed `repeat.generate`, `repeat.critic`.
- `cond` value labels may be written only inside the body; an unwritten label referenced
  inside the body resolves to "empty/absent" (do not error here; this is allowed §4.7.1).
- Record `DoWhile` so M6 renders "run once, then repeat" and M4 still enforces `max`
  (`AF205`).

## 5. `Resolve` orchestration

```go
func Resolve(prog *model.Program) (*Resolved, diag.Diagnostics) {
    // 1. Normalize each flow body independently (labels, value lineage, returns)
    //    -> per-flow normalized tree + ReturnBinding (pre-inline).
    // 2. Build entry tree by inlining subflows (recursion guard, prefixing).
    // 3. Compute gather payloads + sequential edges on the inlined tree.
    // 4. Populate Instances/Order from the inlined tree (document order).
    // 5. Return Resolved + diagnostics (AF212 recursion, AF213 type conflict,
    //    AF209 default-return-unresolvable markers passed forward).
}
```

Keep `Resolve` total. On recursion (`AF212`), produce a best-effort partial tree (cut the
back-edge) so later passes still run and surface more diagnostics.

## 6. Diagnostics owned vs deferred

| Concern | M3 (this pass) | M4 |
|---------|----------------|-----|
| Recursive subflow nesting | **emits `AF212`** | — |
| Conflicting value-label writer types | **emits `AF213`** | — |
| Default return unresolvable (terminal has no output) | records reason | **emits `AF209`** |
| Ambiguous implicit labels | allocates `#2` + records flag | **emits `AF208`** |
| Cycle without `max` | builds loop node | **emits `AF205`** |
| Branch enum membership | builds branch node w/ `OutEnum` | **emits `AF203`** |

> Add `AF212`, `AF213` to the catalog (and the M4 doc's "see also" list).

## 7. Testing

Goldens: `internal/flowgraph/testdata/<fixture>.resolved.json` (position-stripped,
ordered). Test file `internal/flowgraph/resolve_test.go`.

### 7.1 Resolved-tree goldens

- `review.af`:
  - `code_review` body inlined under `ship` with prefixed labels
    (`code_review.build`, `code_review.quality`, `code_review.reviewer`, loop body
    `code_review.loop.build` etc. — confirm exact scheme and pin).
  - Gather payload on `code_review.review` lists branch control labels
    `code_review.lint`, `code_review.security`, `code_review.style`.
  - `code_review` return = `review` (Verdict) exposed as value label `code_review`.
  - `ship` is branch-terminal; `branch code_review` reads enum `Verdict`.
  - Loop `until review != revise, max 3` → `LoopNode{DoWhile:false, Cond{review != revise}, Max:3}`.
- `pipeline.af`: defaulted return resolves to `edit` (`text`).
- `critic.af`: `repeat` body labels prefixed; `verdict` written inside body and read by
  `until`; `DoWhile == true`; explicit `return: draft` preserved.
- `research.af`: gather payload (`market`, `competitor`, `financial`); Rule 0 return =
  `report`.

### 7.2 Targeted unit tests

- Recursive flow (`flow a { b } flow b { a }`, with one entry) → `AF212`, no overflow.
- `pipeline.af` ending swapped to a gate with no `return:` → records default-return-
  unresolvable (M4 will emit `AF209`).
- Two `reviewer` refs in the same scope without `as` → second labeled `reviewer#2`, flag
  set for `AF208`.
- Conflicting value lineage (`x as r` of type Verdict then `y as r` of type Decision) →
  `AF213`.
- Label prefix scheme stable across nested subflows (`a.b.step`).

## 8. Acceptance criteria

- [ ] Resolved-tree goldens pass for review/pipeline/critic/research.
- [ ] Subflow `code_review` inlined under `ship` with documented prefix scheme.
- [ ] Gather payload on `review` lists lint/security/style (ordered).
- [ ] `return: review` exposes `Verdict` on subflow value `code_review`.
- [ ] Recursive nesting → `AF212`, never a stack overflow.
- [ ] `critic.af` repeat normalized (DoWhile, prefixed labels, verdict lineage).
- [ ] `pipeline.af` default return resolves to `edit`.
- [ ] All ordered outputs are deterministic across runs.

## 9. Commit plan

| # | Commit | Contents |
|---|--------|----------|
| 1 | `flowgraph: types + label allocation` | `types.go`, `labels.go` + label unit tests |
| 2 | `flowgraph: per-flow normalize + value lineage` | sequence/branch/loop/repeat normalization |
| 3 | `flowgraph: subflow inlining + recursion guard (AF212)` | inlining + prefixing |
| 4 | `flowgraph: return binding + Rule 0 default` | return resolution + AF209 markers |
| 5 | `flowgraph: gather payloads + sequential edges` | parallel/gather, PrevProducer |
| 6 | `flowgraph: resolved-tree goldens` | `testdata/*.resolved.json` |

## 10. Risks & notes

- **Label prefix scheme is a published contract** (spec §16). Once goldens land, changing
  it is a breaking regold. Pin dot-separated recursive scheme and document loudly.
- **Loop body scoping.** Treat loop/repeat bodies as nested label scopes so the golden
  program's reuse of `build`/`reviewer` inside the loop does not trigger `AF208`. Get this
  right early — it affects every downstream golden.
- **Rule 0 terminal-producer definition** is the most error-prone semantic. Encode it as a
  single well-tested function `terminalProducer(node *Node) (*StepInstance, ok bool)` and
  unit-test it directly against all five fixtures plus the gate-terminal negative case.
- **Map iteration** — same caveat as M2: emit through `Order`/ordered slices only.
- M9 will plug pattern-call inlining into this package; keep `inline()` parameterized over
  "how to obtain a body for a reference" so the call path slots in without refactoring.
