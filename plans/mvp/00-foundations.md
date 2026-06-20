# M0 - Foundations

- Milestone: M0
- Version: v0.1 (MVP) — Language Level A
- Status: Planned
- Spec: [§3 Language levels](../../spec/grammar.md#3-language-levels), [§13 Resolution](../../spec/grammar.md#13-resolution-and-lowering)

## Goal

Establish the shared infrastructure every later phase depends on: structured
diagnostics, an in-memory output file set, the binding interface, and a CLI
skeleton. Nothing here is AgentFlow-specific logic; it is the plumbing.

## Scope

### In scope
- Diagnostics type and renderer.
- In-memory file set abstraction.
- Binding interface and target registry (interface only, no implementations).
- CLI command dispatch skeleton.

### Out of scope (deferred)
- Any parsing, resolution, or emission logic.
- Capability negotiation (`Capabilities()` implementation — stub only; M10).

## Packages & files

- `internal/diag/diag.go`
- `internal/emit/fs.go`
- `internal/binding/binding.go`
- `cmd/af/main.go`

## Tasks

- `diag.Diagnostic{ Code string; Severity; Msg string; Pos lexer.Position }`,
  `Severity` enum (`Error`, `Warning`), `Diagnostics` slice with `HasErrors()` and
  `Add(...)`.
- `diag.Render(source string) string` prints `file:line:col: severity AFxxx: msg`
  plus the offending source line and a caret pointing at the column.
- Reserve code ranges (see [plans/README](../README.md#diagnostic-code-ranges)):
  - `AF0xx` lex/parse
  - `AF1xx` resolve (`AF110` ambiguous model, `AF150` Level B unsupported in v0.1)
  - `AF2xx` validate (`AF200`–`AF210` data-model rules)
  - `AF3xx` binding / capability negotiation
- `emit.FS` = ordered map of relative path to bytes, with `Write(path, []byte)`,
  `Get(path)`, `Paths()` (sorted), and `Flush(dir)` that writes to disk creating
  parent directories.
- `binding.Binding` interface and a registry:
  - `Name() string`
  - `Capabilities() map[Capability]bool` (stub empty map in M0; populated M7/M10)
  - `Emit(p ir.Program) (emit.FS, diag.Diagnostics)`
  - `binding.Register(b Binding)` / `binding.Get(name) (bool)`.
- CLI skeleton: subcommand dispatch for `validate`, `build`, `graph` (stubs that
  print "not implemented"); shared flag parsing; non-zero exit on error.

## Data shapes / snippets

```go
type Capability string // defined fully in M10; stub here

type Binding interface {
    Name() string
    Capabilities() map[Capability]bool
    Emit(p ir.Program) (emit.FS, diag.Diagnostics)
}
```

## Acceptance criteria

- `go build ./...` succeeds.
- Golden test for `diag.Render` (single error and a warning).
- `emit.FS.Flush` round-trip test (write, flush to temp dir, read back).
- `af` prints usage and dispatches subcommands.

## Dependencies

- None (`ir.Program` can start as an empty struct until M5).

## Risks / notes

- Keep `emit.FS` deterministic (sorted paths) so binding snapshots are stable.
- Golden fixture path: [examples/review.af](../../examples/review.af).
