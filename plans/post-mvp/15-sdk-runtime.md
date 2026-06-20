# M15 - SDK Runtime Binding

- Milestone: M15
- Version: v0.5+ — SDK deterministic tier (§12)
- Status: Planned
- Spec: [§12 Runtime guarantees — SDK runtime row](../../spec/grammar.md#12-runtime-guarantees-by-target), [§11 Matrix](../../spec/grammar.md#11-host-capability-matrix-v01)

## Goal

Opt-in `--runtime sdk`: generate a deterministic orchestrator from Plan IR with
**hard** control flow, output parsing, gates, parallelism, and loop bounds per §12.

## Scope

### In scope
- Codegen from Plan IR (M14).
- Slash command or CLI entry invoking generated program.
- Full §9 output protocol parser in code (not advisory).

### Out of scope (deferred)
- Replacing native runbook default.

## Packages & files

- `internal/binding/sdk/sdk.go`
- `internal/binding/sdk/codegen.go`
- templates/ (TS or Python Agent SDK)

## Tasks

- Codegen mapping:
  - `RUN` -> SDK agent call + parse `agentflow-output` block (§9.2–9.3)
  - `BRANCH`/`LOOP_IF` -> real conditionals on parsed enum values
  - `SPAWN`/`JOIN` -> concurrent calls + gather payload struct (§4.8)
  - `GATE` -> subprocess; implement `on-fail: halt|retry|goto|enter-loop` (§7.4.1)
  - `GOTO` -> jump to labeled step
- Wire policies from IR when M13 landed.
- `af build --runtime sdk --target claude-code` (command shells out to orchestrator).
- **`Capabilities()`:** declare SDK tier — all §12 "hard" columns true.

## Acceptance criteria

- [examples/review.af](../../examples/review.af) loop respects `max 3` in generated code.
- Invalid `out:` enum halts after agent `retry` (§9.3).
- Gate `on-fail: retry` + `on-fail-target: build` re-executes from `build` deterministically.
- Plan-to-code golden snapshots.

## Dependencies

- M14 (Plan IR), M5 (agent/capability facts), M13 (optional policies).

## Risks / notes

- Generated code readable + snapshot-tested.
- Behavioral alignment with native tier where enforcement differs (document in BUILD-NOTES).
