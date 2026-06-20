# M2 - Resolver & Semantic Model

- Milestone: M2
- Version: v0.1 (MVP) — Language Level A
- Status: Planned
- Spec: [§4 Execution/data model](../../spec/grammar.md#4-execution-and-data-model), [§7 Declarations](../../spec/grammar.md#7-declarations), [§8 Model resolution](../../spec/grammar.md#8-model-and-provider-resolution)

## Goal

Lower the generic AST into a typed semantic model with symbol tables, by
interpreting each block's fields according to its kind. Unknown fields warn but
are preserved for forward-compatibility.

## Scope

### In scope
- Typed `model.Program` and its node types.
- Field interpretation per kind (use / type / agent / gate / flow).
- Symbol tables and implicit terminals.
- Level B rejection (`AF150`).
- Model/provider resolution (`AF110`).
- Gate `on-fail` parsing (spec §7.4.1).
- Flow `return:` binding field (spec §4.4).

### Out of scope (deferred)
- Inlining / normalization (M3).
- Validation rules (M4).

## Packages & files

- `internal/model/model.go`
- `internal/sema/resolve.go`

## Tasks

- Define `model` types:
  - `Capability{ Name, Kind; Models, Tools []string; Transport, Command string; Args []string; Raw }`
  - `EnumType{ Name; Values []string }`
  - `Agent{ Name, Model, ModelProvider string; In, Out; Permissions, Prompt; Tools []ToolRef; Retry int; Raw }`
  - `Gate{ Name, Run; OnFail GateFailAction; OnFailTarget string; Behavior; ScriptRetry int; Raw }`
  - `GateFailAction` enum: `halt | retry | goto | enter-loop`
  - `Flow{ Name; Entry bool; On, In, Out, Return string; Body []Step; Params []Param }`
  - `Step` mirrors AST step union with resolved references and optional value alias.
- Resolver:
  - interpret `Block.Fields` by known keys; unknown -> `AF1xx` warning, kept in `Raw`.
  - structural checks: `use kind: mcp` needs `command`; gate `on-fail: retry` needs `on-fail-target`.
- Build symbol tables: `types`, `agents`, `gates`, `capabilities`, `flows`.
- Register implicit terminals `done` and `fail`.
- **Level B rejection:** flow `Params`, `Call` args, `parallel each`, explicit `it`
  -> `AF150 unsupported in v0.1` (spec §3.2).
- **Model resolution (§8):** unqualified alias -> exactly one provider or `AF110`;
  qualified `provider.alias`; alias must be in provider `models:` list.
- **Flow header:** `in:` opaque nominal OK without `type` decl; `out:` enum or
  `text` — **optional** (inferred from the return value when omitted); `return:`
  value label — **optional**, defaults to the terminal producer (spec §4.4 Rule 0;
  validated in M4). Record on `Flow` whether `Return`/`Out` were explicit or
  defaulted so render/IR are deterministic.
- **Gate on-fail:** reject `bounce-back`; parse `halt`, `retry`, `goto`,
  `enter-loop`; require `on-fail-target` for `retry`/`goto`.
- Exactly one `entry flow`; zero or multiple -> diagnostic.

## Data shapes / snippets

```go
type GateFailAction int
const (
    FailHalt GateFailAction = iota
    FailRetryStep
    FailGotoStep
    FailEnterLoop
)

type Flow struct {
    Name, On, In, Out, Return string
    Entry bool
    Body  []Step
}
```

## Acceptance criteria

- Model snapshot for [examples/review.af](../../examples/review.af).
- Unknown-field warning test (field retained in `Raw`).
- Level B construct -> `AF150` (flow with params).
- Ambiguous model alias -> `AF110`.
- `bounce-back` in gate -> error.
- Gate `on-fail: retry` + `on-fail-target: build` parsed with target `build`.

## Dependencies

- M1 (AST), M0 (`diag`).

## Risks / notes

- Keep field interpretation table-driven per kind.
- `Return` on `ship` flow is empty (branch-terminal); `Return` on `code_review` is `review`.
- Default-return sugar (spec §4.4 Rule 0): `pipeline.af` `content` resolves
  `Return = edit` implicitly; `critic.af` `sql` must keep explicit `return: draft`
  because the terminal step is the critic, not the draft.
