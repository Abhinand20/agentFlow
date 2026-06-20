# M9 - Abstraction & std/patterns

- Milestone: M9
- Version: v0.2 — enables Language Level B
- Status: Planned
- Spec: [§3.2 Level B](../../spec/grammar.md#32-level-b--parsed-but-disabled-v02-target), [§5 Patterns as libraries](../../spec/grammar.md#5-the-kernel-primitives)

## Goal

Enable Language Level B: flow parameters, calls with arguments, trailing flow
blocks, `parallel each ... as`, and explicit `it`. Ship `std.patterns` as AgentFlow
source so architectures remain library code, not grammar keywords.

## Scope

### In scope
- Remove `AF150` rejection for Level B constructs.
- Real inlining with argument binding + `each` expansion.
- Embedded `std/patterns.af`.

### Out of scope (deferred)
- Record types (M12 — Level C).
- Policies (M13 — Level C).

## Packages & files

- `internal/sema/resolve.go`
- `internal/flowgraph/inline.go`
- `internal/stdlib/patterns.af` (`go:embed`)
- `internal/stdlib/stdlib.go`

## Tasks

- Flow parameters (`agent` / `flow` / `list` / enum); positional + named args;
  trailing flow block as last flow-typed arg.
- `parallel each <coll> as <x> { ... }` — expand static collections; dynamic
  fan-out metadata for advisory/SDK tiers.
- Explicit `it` in conditions — refers to latest output in call/callee scope (§4.5).
- Inlining: bind args, substitute callee body, preserve label prefixing from M3.
- Author `std/patterns.af`: `supervise`, `map`, `consensus`, `debate`, `handoff`.
- `use std.patterns as p` resolves to embedded source.

## Data shapes / snippets

```text
use std.patterns as p

type Route = billing | technical | account | resolved

flow support {
  p.supervise(triage, max: 6) {
    branch triage {
      case billing   -> billing_agent
      case technical -> tech_agent
      case account   -> account_agent
    }
  }
}
```

(No semicolons; `branch triage` not `branch triage.route`.)

## Acceptance criteria

- Supervisor example expands to same resolved tree as hand-written loop+branch.
- Validation, IR, render unchanged on expanded graph (data model §4 still applies).
- Arity/type mismatch -> `AF1xx`.
- Level B parse + resolve succeeds; Level A programs still validate.

## Dependencies

- M3 (inlining), M2, M4, M6. Golden base remains [examples/review.af](../../examples/review.af).

## Risks / notes

- Trailing block scoping rules must be documented and tested.
- `each` dynamic fan-out: native tier advisory per §12.
