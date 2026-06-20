# M12 - Records & Multi-output Agents

- Milestone: M12
- Version: v0.3 — Language Level C (types)
- Status: Planned
- Spec: [§3.3 Level C](../../spec/grammar.md#33-level-c--post-mvp-syntax), [§4.8 Gather payloads](../../spec/grammar.md#48-parallel-and-gather-payloads)

## Goal

Extend the type system and data model: record types, multi-output agents, dotted
branch conditions, typed gather payloads, and multi-field output protocol.

## Scope

### In scope
- Record types + `outputs { }` on agents.
- Dotted conditions (`step.field == v`).
- Typed gather bundles.
- Backward compat: `out: Enum` sugar for `outputs { out: Enum }`.

### Out of scope (deferred)
- Generics.

## Packages & files

- `internal/parser`, `internal/model`, `internal/sema`, `internal/flowgraph`
- `internal/ir`, `internal/render/protocol.go`

## Tasks

- Grammar: `type Build = { diff: text, files: [text] }`.
- `outputs { review: Review }` supersedes single `out:` (sugar preserved).
- Branch: `branch reviewer.review.verdict { case approve -> ... }`.
- Gather payload becomes typed record in IR (extends §4.8).
- Output protocol §9 extension: multiple `key: value` lines in one fence; parse
  all declared output fields.
- Validation: field exists; enum fields used in conditions; structural `in:`/`out:` compat.

## Data shapes / snippets

```text
type Review = { verdict: Verdict, summary: text }

agent reviewer {
  model: opus
  outputs { review: Review }
}

branch reviewer.review.verdict {
  case approve -> deploy
}
```

## Acceptance criteria

- [examples/review.af](../../examples/review.af) still validates unchanged (enum sugar).
- Multi-output fixture routes on dotted field.
- Gather payload typed in IR JSON snapshot.

## Dependencies

- M2, M3, M4, M5, M6.

## Risks / notes

- Output protocol field names come from `outputs` keys, not hard-coded `out:`.
