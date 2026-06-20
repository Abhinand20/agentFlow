# M4 - Validation

- Milestone: M4
- Version: v0.1 (MVP) — Language Level A
- Status: Planned
- Spec: [§4 Execution/data model](../../spec/grammar.md#4-execution-and-data-model), [§13.1 Validate](../../spec/grammar.md#131-validate-v01-rule-set)

## Goal

Run the v0.1 validation rule set over the Resolved flow and the model, producing
clear `AF2xx` diagnostics. Validation runs **after** inlining (M3) so it sees the
fully expanded graph with control/value labels and gather payloads.

## Scope

### In scope
- Twelve validation rules below (`AF200`–`AF211`; + `AF110` resolver fixture documented here).
- One passing and one failing fixture per rule.
- Golden fixture [examples/review.af](../../examples/review.af) validates clean.

### Out of scope (deferred)
- Record structural checks (M12).
- Policy coverage (M13).
- Call-argument type checks (M9).

## Packages & files

- `internal/sema/validate.go`
- `internal/sema/rules/*.go`

## Tasks

Implement each rule as `func(*model.Program, *flowgraph.Resolved) diag.Diagnostics`:

| Code | Rule |
|------|------|
| `AF200` | Duplicate declaration names |
| `AF201` | Every flow node resolves to agent / gate / subflow / terminal |
| `AF202` | `in`/`out` types: declared enum, builtin `text`, or opaque nominal for `in` only |
| `AF203` | `branch`/`loop until`: value label has enum `out` containing case value |
| `AF204` | (warning) Conditional branches exhaustive over reachable enum values |
| `AF205` | Any cycle requires `max` bound |
| `AF206` | Qualified tool refs exist on capability |
| `AF207` | (warning) Orphan nodes unreachable from entry |
| `AF208` | Ambiguous duplicate implicit control/value labels in same flow scope |
| `AF209` | `return:` value label exists and matches flow `out:`; when omitted, the default terminal producer (§4.4 Rule 0) exists and carries a typed/text output (else ambiguous/missing default return) |
| `AF210` | Branch-terminal flow (no `return:`): each leaf step `out:` matches flow `out:` |
| `AF211` | Prompt source: `prompt` + `prompt-file` together, or a prompt path (`prompt-file:` / `.md`-valued `prompt:`) that is absolute, escapes the source dir, is missing, unreadable, or invalid UTF-8 (spec §7.3.1) |

Also cover resolver `AF110` (ambiguous model) in M2 fixtures; list here for the
full diagnostic catalog.

Aggregate all rule outputs; do not stop at the first error.

## Data shapes / snippets

```go
var rules = []Rule{
    ruleDuplicateNames,      // AF200
    ruleNodeResolution,      // AF201
    ruleTypesExist,          // AF202
    ruleConditionEnum,       // AF203
    ruleBranchExhaustive,    // AF204
    ruleCycleBounded,        // AF205
    ruleToolRefs,            // AF206
    ruleReachability,        // AF207
    ruleDuplicateLabels,     // AF208
    ruleReturnBinding,       // AF209
    ruleBranchTerminalOut,   // AF210
    rulePromptSource,        // AF211
}
```

## Acceptance criteria

- For each `AF2xx` rule: passing + failing fixture asserting exact code.
- [examples/review.af](../../examples/review.af) validates with zero errors.
- `ship` flow passes `AF210` (deploy/notify_author both `out: Decision`).
- `code_review` passes `AF209` (`return: review`, `out: Verdict`).
- `pipeline.af` passes `AF209` with defaulted `return` (terminal producer `edit`).
- A sequence flow ending in a gate (no typed output) and no explicit `return:`
  fails `AF209` (missing default return).
- `docs.af` passes `AF211`: `outline` (`prompt:` `.md` path) and `draft`
  (`prompt-file:`) both resolve to in-tree files.
- `AF211` failing fixtures: `prompt` + `prompt-file` together; `prompt: "x.md"`
  pointing at a missing file; a path escaping the source dir.

## Dependencies

- M2 (model), M3 (resolved flow with labels/payloads).

## Risks / notes

- `AF203` uses value labels (`review != revise`), not dotted paths (M12).
- `AF204` exhaustiveness: warn-only initially; account for upstream routing.
