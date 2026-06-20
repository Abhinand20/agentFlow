# M11 - Registry, Formatter, Diagnostics Polish

- Milestone: M11
- Version: v0.2 — Level B registry + tooling
- Status: Planned
- Spec: [§3.3 Level C registry item](../../spec/grammar.md#33-level-c--post-mvp-syntax) (registry lands here as Level B convenience)

## Goal

Versioned capability registry, canonical formatter, and richer diagnostics — without
changing the v0.1 data model or golden fixture semantics.

## Scope

### In scope
- Local capability registry + `use pkg@semver as alias`.
- `af fmt` (no semicolons; one step per line).
- Carets + did-you-mean suggestions.

### Out of scope (deferred)
- Remote/network registries.

## Packages & files

- `internal/registry/registry.go`
- `cmd/af/fmt.go`
- `internal/parser/printer.go`
- `internal/diag/suggest.go`

## Tasks

- Registry descriptors (same shape as inline `use`); search:
  `./.agentflow/registry` -> `~/.agentflow/registry` -> embedded.
- `use github@^2 as gh` with caret matching; `AF111` unresolved, `AF112` version mismatch.
- `af fmt`: idempotent; formats [examples/review.af](../../examples/review.af) as canonical style reference.
- Diagnostics: multi-column carets; Levenshtein suggestions over symbol tables
  (control labels, value labels, agents, types, capabilities).

## Acceptance criteria

- `use github@^2 as gh` resolves from registry file.
- `fmt(fmt(review.af)) == fmt(review.af)`.
- Suggestion on typo `reviewr` -> `reviewer`.

## Dependencies

- M2, M1, M0.

## Risks / notes

- Registry schema = inline `use` body (spec §7.1).
