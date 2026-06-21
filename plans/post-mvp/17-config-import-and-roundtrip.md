# M17 - Config Import & Round-Trip

- Milestone: M17
- Version: v0.6+ — tooling / adoption
- Status: Planned
- Spec: [§10 Bindings](../../spec/grammar.md#10-bindings-overview), [§11 Host capability matrix](../../spec/grammar.md#11-host-capability-matrix-v01)

## Goal

Reverse the compiler: read **pre-defined host agent configs** (an existing
`.cursor/` or `.claude/` tree) and reconstruct an AgentFlow `.af` file plus an IR
snapshot. This lets teams that already hand-wrote subagents adopt AgentFlow
without rewriting everything from scratch, and enables **round-trip** (`.af` →
host → `.af`) so a build can be diffed against its source.

> AgentFlow today is a one-way compiler: `.af` → host config. M17 adds the
> inverse direction so existing agents become a starting `.af` to refine.

## Motivation

- **Adoption on-ramp.** Most teams already have `.cursor/agents/*.md` or
  `.claude/commands/*.md`. Importing them produces an editable `.af` instead of
  a blank file.
- **Round-trip safety.** `af import` then `af build` should be idempotent for
  the supported subset; divergence is a diagnostic, not a surprise.
- **Migration between hosts.** Import Cursor config → emit Claude config (once
  M7 lands) by going through the shared IR.

## Scope

### In scope
- `af import --from cursor <dir>` and `af import --from claude <dir>`.
- Parse host artifacts that AgentFlow itself emits:
  - Cursor: `.cursor/agents/*.md` (name/description/model/readonly frontmatter +
    prompt body), `.cursor/commands/<on>.md` (runbook + `<!-- agentflow: ... -->`
    metadata comment), `.cursor/mcp.json`.
  - Claude (after M7): `.claude/agents/*.md`, `.claude/commands/<on>.md`,
    `.mcp.json`.
- Reconstruct: agents (model alias via reverse `HostModelID` lookup), `use`
  model-provider block, capabilities from MCP, and a best-effort `entry flow`.
- Emit `.af` source + an `af import --emit-ir` path that yields the same IR
  shape as `af build --emit-ir`.

### Out of scope (deferred)
- Importing arbitrary hand-authored configs that AgentFlow never generated
  (no metadata comment) — best-effort only, flagged with `AF4xx`.
- Recovering control flow that the host runbook lost (advisory loops/gates).
  Round-trip is exact only for constructs the binding preserves.
- Importing non-AgentFlow orchestration frameworks.

## Packages & files

- `cmd/af/import.go` — `af import` subcommand.
- `internal/binding/<host>/import.go` — per-host parser (inverse of `Emit`).
- `internal/ir/fromhost.go` — assemble `ir.Program` from parsed artifacts.
- `internal/unparse/` — IR/model → `.af` source printer (shares style rules with
  the M11 formatter).

## Tasks

- Define the round-trip contract: which IR fields the metadata comment must carry
  so import is lossless for the Level A subset (entry trigger, `in:`/`out:` types,
  agent models, value labels).
- Cursor importer: parse frontmatter + body + command metadata → `ir.Program`.
- IR → `.af` unparser reusing the formatter's no-semicolon style.
- `af import` CLI: `--from`, `--out`, `--emit-ir`, exit policy matching `build`.
- Round-trip test: `review.af` → `build --target cursor` → `import` → IR equals
  the original `build --emit-ir` for the preserved subset.

## Acceptance criteria

- `af import --from cursor` on an AgentFlow-generated tree reconstructs a `.af`
  that re-compiles to byte-identical agent/command files (golden round-trip).
- Constructs the host lost (advisory loop bounds, gate enforcement) are reported
  with `AF4xx` "not recoverable from host config" diagnostics, not silently dropped.
- Importing a hand-authored config with no `agentflow:` metadata still produces a
  usable skeleton `.af` (agents + entry flow), flagged as best-effort.

## Dependencies

- M5 (IR is the reconstruction target), M6/M10 (Cursor emit format is the parse
  target), M11 (formatter/printer reused by the unparser), M7 (Claude import).

## Risks / notes

- Lossy by nature for advisory tiers — the metadata comment is the contract that
  bounds how much is recoverable; keep it in sync with the binding's `Emit`.
- Hand-authored configs vary wildly; scope the guarantee to AgentFlow-emitted
  trees and treat everything else as best-effort with clear diagnostics.
