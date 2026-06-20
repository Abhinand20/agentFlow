# M14 - Plan IR & Simulator

- Milestone: M14
- Version: v0.5 — Language Level C
- Status: Planned
- Spec: [§4 Data model](../../spec/grammar.md#4-execution-and-data-model), [§9 Output protocol](../../spec/grammar.md#9-output-protocol-contract), [§13 Resolution](../../spec/grammar.md#13-resolution-and-lowering) (Plan IR note)

## Goal

Introduce linear Plan IR and static simulator — first consumers that need flat
control flow with explicit labels. Add `test` / `expect` and `af test`.

## Scope

### In scope
- `lower(resolved) -> plan` with explicit control/value labels matching M3/M5.
- Simulator honoring stubs, gather payloads, gate `on-fail` targets, loop `max`.
- `test` blocks (Level C syntax).

### Out of scope (deferred)
- Real agent execution (M15 SDK).

## Packages & files

- `internal/plan/plan.go`, `internal/plan/lower.go`
- `internal/sim/sim.go`
- `cmd/af/test.go`

## Tasks

- Opcodes: `RUN`, `BRANCH`, `SPAWN`, `JOIN`, `GATE`, `LOOP_IF`, `GOTO` — args
  include control labels, value labels, gather payload keys, gate fail targets (§7.4.1).
- Lower from `flowgraph.Resolved` + `ir.Program` (not AST).
- Simulator:
  - stub agent -> enum value for `out:` (§9)
  - track loop counts vs `max`
  - gate fail -> `retry|goto|halt|enter-loop` behavior
  - gather payload assembly per §4.8
- Language: `test "name" { stub reviewer -> approve; run ship(...); expect ... }`.
- Assertions: `terminates(loops<=N)`, `cost < usd(x)`, `policy P applies_to(agent)`.
- `af test` on [examples/review.af](../../examples/review.af) fixtures.

## Acceptance criteria

- Plan snapshot for `review.af` lowerer.
- Simulator catches loop exceeding `max 3`.
- Simulator models gate `on-fail: retry` + `on-fail-target: build` correctly.
- `af test` passes on golden tests.

## Dependencies

- M3, M5, M13 (cost assertions).

## Risks / notes

- Plan labels must match render runbook Step N mapping for traceability.
