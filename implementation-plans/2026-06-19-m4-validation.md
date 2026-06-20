# M4 — Validation (execution plan)

- **Date:** 2026-06-19
- **Milestone:** M4
- **Design spec:** [plans/mvp/04-validation.md](../plans/mvp/04-validation.md)
- **Language spec:** [§4](../spec/grammar.md#4-execution-and-data-model), [§13.1](../spec/grammar.md#131-validate-v01-rule-set)
- **Status:** Done
- **Depends on:** M2 (`model.Program`), M3 (`flowgraph.Resolved`)
- **Blocks:** M5 (IR is only built from a valid program), M8 (`af validate`)

## 1. Goal

Run the v0.1 validation rule set (`AF200`–`AF213`) over the **resolved** flow plus the
model, producing precise, positioned diagnostics. Validation runs **after** inlining
(M3) so every rule sees fully expanded control/value labels and gather payloads. Rules
**aggregate** — never stop at the first error.

Validation is a pure function: `(model, resolved) → diagnostics`. It mutates nothing. The
pipeline (M8) treats `HasErrors()` as the gate for proceeding to IR.

## 2. Deliverables

- `internal/validate/validate.go` — `Validate(prog *model.Program, res *flowgraph.Resolved) diag.Diagnostics`.
- `internal/validate/af*.go` — one file per rule group.

> **Package note.** Validation lives in `internal/validate` (not `internal/sema`) to avoid
> an import cycle: `flowgraph` tests import `sema`, and validation needs both `model` and
> `flowgraph.Resolved`.
- Per-rule fixtures + tests under `internal/sema/testdata/validate/`.

## 3. Architecture

```go
package sema

type ruleCtx struct {
    Prog *model.Program
    Res  *flowgraph.Resolved
}

type Rule struct {
    Code string
    Fn   func(ruleCtx) diag.Diagnostics
}

var rules = []Rule{
    {"AF200", ruleDuplicateNames},
    {"AF201", ruleNodeResolution},
    {"AF202", ruleTypesExist},
    {"AF203", ruleConditionEnum},
    {"AF204", ruleBranchExhaustive},   // warning
    {"AF205", ruleCycleBounded},
    {"AF206", ruleToolRefs},
    {"AF207", ruleReachability},        // warning
    {"AF208", ruleDuplicateLabels},
    {"AF209", ruleReturnBinding},
    {"AF210", ruleBranchTerminalOut},
    {"AF211", rulePromptSource},
    {"AF212", ruleNoRecursion},         // M3 already emits; M4 re-checks marker (see §5)
    {"AF213", ruleValueTypeConsistency},// M3 already emits; M4 re-checks marker
}

func Validate(prog *model.Program, res *flowgraph.Resolved) diag.Diagnostics {
    var out diag.Diagnostics
    ctx := ruleCtx{prog, res}
    for _, r := range rules {
        out.Add(r.Fn(ctx)...)
    }
    sortDiagnostics(out) // by Pos (line, col), then Code, for stable output
    return out
}
```

> **AF212/AF213 ownership.** M3 emits these during resolution. To keep `af validate`'s
> output complete even if a caller runs `Validate` standalone, M4 either (a) trusts M3 to
> have emitted them and omits duplicates, or (b) re-derives from markers on `Resolved`.
> **Decision:** M3 is the single emitter; M4 does **not** duplicate `AF212`/`AF213`. They
> appear in the `rules` list only as documentation comments. The pipeline (M8)
> concatenates M3 diagnostics + M4 diagnostics, so the user sees them once. Drop the two
> no-op entries from the live slice; keep them in the doc table.

### 3.1 Determinism

`sortDiagnostics` sorts by `(Filename, Line, Column, Code)`. This makes multi-error output
stable for golden comparison regardless of rule execution order.

## 4. Rules (exact semantics + message shape)

Each message should name the offending symbol and, where useful, the fix. Positions come
from the relevant `model`/`flowgraph` node.

| Code | Sev | Rule | Message shape |
|------|-----|------|---------------|
| `AF200` | error | Duplicate declaration names across the program's decl namespace (agents/gates/flows/types/capabilities; collisions *between* namespaces also reported). | `duplicate declaration "reviewer" (also at L12)` |
| `AF201` | error | Every flow node reference resolves to an agent, gate, subflow, or terminal (`done`/`fail`). | `unknown step "deplyo" in flow ship` |
| `AF202` | error | `in`/`out` types are: a declared enum, builtin `text`, or (for `in` only) an opaque nominal. `out:` referencing an undeclared non-`text` type fails. | `agent reviewer: out type "Verdct" is not a declared enum` |
| `AF203` | error | `branch`/`loop until`/`repeat until` value label has an enum `out`, and each case/condition value is a member of that enum. | `branch reads "review" (Verdict); "approv" is not a member` |
| `AF204` | warn | Conditional branches are exhaustive over **reachable** enum values. | `branch on review may not handle: reject` |
| `AF205` | error | Any cycle (loop/repeat, or `goto`/`retry` back-edge) has a `max` bound. | `loop has no max bound` |
| `AF206` | error | Qualified tool refs (`github.get_pr`) name a capability that exists and lists that tool. | `tool "github.delete_pr" not provided by capability github` |
| `AF207` | warn | No orphan nodes unreachable from the entry flow. | `flow helper is never reached from entry` |
| `AF208` | error | No ambiguous duplicate **implicit** control/value labels in the same flow scope (M3 flags these). | `two steps named "reviewer" in flow X; use "as" to disambiguate` |
| `AF209` | error | `return:` value label exists & type matches flow `out:`; when omitted, the Rule 0 terminal producer exists and carries a typed/text output. | `flow code_review: return "reviw" is not a value label` / `flow X: no return and terminal step produces no output` |
| `AF210` | error | Branch-terminal flow (no `return:`): every leaf step's `out:` matches the flow `out:`. | `flow ship: branch leaf notify_author out "text" != flow out "Decision"` |
| `AF211` | error | Prompt source invalid: `prompt`+`prompt-file` together; or a prompt **path** (`prompt-file:` / `.md`-valued `prompt:`) that is absolute, escapes the source dir, is missing, unreadable, or invalid UTF-8. | `agent reviewer: prompt file "prompts/x.md" not found` |
| `AF212` | error | (Emitted by M3) recursive flow nesting. | listed for catalog completeness |
| `AF213` | error | (Emitted by M3) value label written with conflicting output types. | listed for catalog completeness |

### 4.1 Rule notes & edge cases

- **AF200 namespaces.** Agents, gates, and flows share one reference namespace (a step
  `reviewer` could be any of them). Therefore a name reused across these kinds is a
  duplicate. Types and capabilities are separate namespaces but still must be unique
  within themselves. `done`/`fail` are reserved: a user decl named `done` → `AF200`
  ("redefines reserved terminal"). (This resolves the M2 note about terminal precedence:
  reserved names win; redefinition is an error, not a warning.)
- **AF202 `text`.** `out: text` is always valid. `in:` may be any nominal (opaque). A
  flow `out:` that is omitted is fine (inferred); only an *explicit* bad type fails.
- **AF203 uses value labels**, not dotted field paths (those are M12). The value label's
  enum comes from `BranchNode.OutEnum` / the writer's `OutType` computed in M3.
- **AF204 reachability-aware exhaustiveness.** Only warn about enum members that could
  actually flow to the branch. For v0.1, "reachable" can be approximated as "all members
  of the enum" unless upstream narrows it (no narrowing in v0.1, so it is effectively
  "all members not covered"). Keep it a **warning**; `review.af`'s `ship` branch covers
  approve/revise/reject (exhaustive) → no warning.
- **AF205 covers all cycles.** Loops/repeats always have a `max` slot (grammar allows it
  to be absent; if absent → `AF205`). Also: a gate `on-fail: retry`/`goto` whose target
  is *upstream* creates a cycle in the control graph — that back-edge must be bounded. For
  v0.1, gate retries are bounded by the gate's `retry:` (script retries) + the loop's
  `max` if inside a loop; a bare `retry` to an upstream label **outside** any loop with no
  bound → `AF205`. Document the back-edge detection: build a directed graph of control
  labels including gate on-fail edges; any cycle whose edges are not all `max`-bounded →
  error.
- **AF206** only applies to **qualified** tool refs. Unqualified tools (rare in v0.1) are
  not checked here.
- **AF207** is a warning; orphan = a declared flow/agent not reachable from entry. Agents
  used only inside an unreached flow are transitively orphaned (report the flow).
- **AF209/AF210** consume M3's `ReturnBinding`:
  - `Defaulted && unresolvable` → `AF209` "missing default return".
  - `ReturnExplicit` but value label absent → `AF209`.
  - `OutExplicit` and `Return` type ≠ `Out` type → `AF209`.
  - `BranchTerminal` → run `AF210` over leaves instead.
- **AF211** consumes M2's `PromptResolution` markers (reason → message). The `prompt` +
  `prompt-file` conflict reason maps to a distinct message. A `.md`-valued `prompt:` that
  does not resolve to a readable in-tree file is `AF211` (not a silent fallback to inline).

## 5. Fixtures & testing

Each rule gets a **passing** and a **failing** fixture. Keep fixtures tiny and
single-purpose (one rule each), as inline strings or small `.af` files under
`internal/sema/testdata/validate/`. Avoid re-deriving `review.af`.

### 5.1 Failing-fixture table (assert exact code present, and that unrelated codes are absent)

| Code | Failing fixture sketch |
|------|------------------------|
| AF200 | two `agent reviewer { … }` |
| AF201 | flow references `nonexistent` |
| AF202 | `agent a { out: Undeclared }` |
| AF203 | `branch review { case notamember -> done }` |
| AF204 | branch covering only `approve` of `Verdict` (warning) |
| AF205 | `loop (until x == y) { … }` with no `max` |
| AF206 | `tools: [github.nope]` |
| AF207 | a second flow never referenced (warning) |
| AF208 | two bare `reviewer` in one scope (no `as`) |
| AF209 | `return: typo`; and a sequence ending in a gate with no `return:` |
| AF210 | branch leaf with `out` mismatching flow `out` |
| AF211 | `prompt` + `prompt-file`; missing `.md` file; `../escape.md`; absolute path |

### 5.2 Golden-program assertion

- `review.af` → **zero errors** (warnings allowed only if intended; `ship` branch is
  exhaustive so expect zero `AF204`).
- `ship` passes `AF210` (deploy/notify_author both `Decision`).
- `code_review` passes `AF209` (`return: review`, `out: Verdict`).
- `pipeline.af` passes `AF209` with defaulted return (`edit`).
- `docs.af` passes `AF211` (both prompt files resolve in-tree).

### 5.3 Aggregation test

A fixture that violates **three** rules at once must report all three (proves no
short-circuit), in deterministic sorted order.

## 6. Acceptance criteria

- [ ] Every `AF2xx` rule has a passing + failing fixture asserting the exact code.
- [ ] `review.af` validates with zero errors.
- [ ] Multi-error fixture reports all violations (aggregation), sorted deterministically.
- [ ] `AF209` distinguishes "bad explicit return", "type mismatch", and "missing default
      return".
- [ ] `AF211` covers conflict / missing / escape / absolute / non-utf8 reasons.
- [ ] `Validate` mutates nothing and never panics.

## 7. Commit plan

| # | Commit | Contents |
|---|--------|----------|
| 1 | `sema: validate harness + sorting + AF200/AF201` | dispatch, sorting, name/resolution rules |
| 2 | `sema: type + condition rules (AF202/AF203/AF206)` | type/enum/tool rules |
| 3 | `sema: structure rules (AF205/AF207/AF208)` | cycle bounds, reachability, dup labels |
| 4 | `sema: return rules (AF204/AF209/AF210)` | branch exhaustiveness + return/branch-terminal |
| 5 | `sema: prompt source rule (AF211)` | consume M2 markers |
| 6 | `sema: per-rule fixtures + golden-program clean` | testdata + aggregation test |

## 8. Risks & notes

- **Catalog drift.** M2 and M3 introduced `AF11x`–`AF13x`, `AF212`, `AF213`. Update
  [plans/README.md](../plans/README.md) and [plans/mvp/04-validation.md](../plans/mvp/04-validation.md)
  so the validation table is the authoritative catalog. (Handled in the index task.)
- **AF205 back-edge detection** is the trickiest rule — model gate `on-fail` edges as
  graph edges and reuse a single cycle-finder. Write it once, test loops, repeats, and
  gate-retry cycles.
- **Warnings vs errors.** `AF204` and `AF207` are warnings — they must not cause
  `HasErrors()` to trip, so `af validate` still exits 0 with only warnings (decide CLI
  exit policy in M8: warnings → exit 0, errors → exit 1).
- Keep each rule independent and side-effect free so they can run in any order / in
  parallel later if needed.
