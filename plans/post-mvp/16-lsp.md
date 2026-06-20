# M16 - LSP

- Milestone: M16
- Version: v0.5+ — tooling
- Status: Planned
- Spec: [§3 Language levels](../../spec/grammar.md#3-language-levels), [§4 Data model](../../spec/grammar.md#4-execution-and-data-model)

## Goal

Editor support via LSP: diagnostics, go-to-definition, formatting — reusing
`pipeline.Compile` so behavior matches CLI exactly.

## Scope

### In scope
- Push diagnostics (all `AFxxx` codes including `AF150`, `AF208`–`AF210`).
- Go-to-definition: agents, flows, types, capabilities, control labels, and value labels in scope.
- Formatting via M11 printer.

### Out of scope (deferred)
- Completion, hover, rename.

## Packages & files

- `cmd/af-lsp/main.go`
- `internal/lsp/server.go`

## Tasks

- `textDocument/didOpen|didChange` -> `pipeline.Compile` -> publish diagnostics.
- `textDocument/definition`:
  - declaration names -> agent/flow/type/use block
  - step ref in flow body -> agent/flow declaration
  - `return:` -> value label definition; gate `on-fail-target` -> control label definition in resolved scope (after inline pass in pipeline, or best-effort pre-inline label)
- `textDocument/formatting` -> M11 printer (preserves no-semicolon style).
- UTF-16 position mapping for LSP.

## Acceptance criteria

- Live diagnostics match `af validate` on [examples/review.af](../../examples/review.af).
- Go-to-def on `reviewer as review` in loop body -> `agent reviewer` block and value label `review`.
- Go-to-def on `on-fail-target: build` in gate -> `agent build` block.
- Format matches `af fmt`.

## Dependencies

- M8 (`pipeline.Compile`), M11 (printer), M2 (symbols), M0 (diag).

## Risks / notes

- Thin adapter only — no duplicate compiler logic.
- Level B files show `AF150` until M9 enabled.
