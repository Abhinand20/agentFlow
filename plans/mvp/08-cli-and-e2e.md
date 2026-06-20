# M8 - CLI & End-to-End

- Milestone: M8
- Version: v0.1 (MVP) — Language Level A complete
- Status: Planned
- Spec: [§14 Canonical golden program](../../spec/grammar.md#14-canonical-golden-program)

## Goal

Wire the pipeline behind `af`, ship the golden fixture, and prove MVP definition
of done with E2E tests. All snapshots derive from one canonical program.

## Scope

### In scope
- `af validate`, `af build`, `af graph`.
- Shared `pipeline.Compile`.
- Golden fixture + gate script stub.
- E2E test.

### Out of scope (deferred)
- `af fmt` (M11), `af test` (M14), registry (M11).

## Packages & files

- `cmd/af/main.go`, `validate.go`, `build.go`, `graph.go`
- `internal/pipeline/pipeline.go`
- [examples/review.af](../../examples/review.af) (canonical — do not fork)
- `examples/scripts/test.sh`
- `testdata/` — AST, IR, render, FS golden files from `review.af`

## Tasks

- `pipeline.Compile(path) (ir.Program, diag.Diagnostics)`:
  parse -> resolve -> inline/normalize -> validate -> IR.
- `af validate <file>` — diagnostics, non-zero on errors.
- `af graph <file>` — DOT to stdout.
- `af build <file> --target claude-code [--out dir]` — IR -> render -> binding -> FS flush.
- Add `examples/scripts/test.sh` (gate stub; exit 0 for happy path).
- E2E: build `review.af` to temp dir; assert file tree + key content goldens.

## Acceptance criteria

**MVP definition of done** (all must pass on [examples/review.af](../../examples/review.af)):

- `af validate` — zero errors.
- `af graph` — DOT includes prefixed subflow labels and gather edges.
- `af build --target claude-code` — produces `.claude/commands/ship.md`, agents, `.mcp.json`, settings.
- Runbook implements §9 output protocol instructions and §7.4.1 gate `retry` to control label `build`.
- E2E test green; all `testdata/` goldens tied to this file.

## Dependencies

- M0–M7.

## Risks / notes

- `pipeline.Compile` is the single entry for CLI, E2E, LSP (M16), and simulator (M14).
- Any spec/plan/example drift: update `review.af` first, then regold.
