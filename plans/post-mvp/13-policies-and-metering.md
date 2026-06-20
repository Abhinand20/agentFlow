# M13 - Policies & Metering

- Milestone: M13
- Version: v0.4 — Language Level C
- Status: Planned
- Spec: [§12 Runtime guarantees](../../spec/grammar.md#12-runtime-guarantees-by-target), [§11 Capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01)

## Goal

Cross-cutting `policy` blocks with selectors, effective-policy resolution, and
hook/SDK enforcement mapped to the guarantees tier table.

## Scope

### In scope
- `policy` + agent `tags`.
- Effective policy per agent in IR.
- Hook enforcement (native/hook tiers) + `AF3xx` for unsupported policy aspects.
- Static `metering.json`.

### Out of scope (deferred)
- Live runtime metering (AgentFlow is not a runtime).

## Packages & files

- `internal/model/policy.go`, `internal/sema/policy.go`
- `internal/ir`, bindings

## Tasks

- `policy { select: all|tag(x)|name(x); budget; timeout; retry; fallback; guard; on_violation }`.
- Precedence: `name > tag > all`.
- Validation: fallback models resolve per §8; warn uncovered budget (`AF213`).
- IR: `EffectivePolicy` on each agent.
- Bindings:
  - **hook-enforced tier:** budget/timeout hooks where host supports
  - **native tier:** advisory runbook text only -> `AF3xx`
  - **SDK tier (M15):** hard enforcement in generated code
- `metering.json` — static description of tracked metrics.

## Acceptance criteria

- `tag(handles_user_data)` policy applies only to tagged agents.
- Claude binding emits hooks for budget when supported; Cursor warns `AF3xx`.
- Precedence tests pass.

## Dependencies

- M2, M4, M5, M7, M10.

## Risks / notes

- Policy fields interpreted semantically, not in grammar.
