# M1 - Lexer & Parser

- Milestone: M1
- Version: v0.1 (MVP) — parses Level A + Level B surface; semantics Level A only
- Status: Planned
- Spec: [§3 Language levels](../../spec/grammar.md#3-language-levels), [§6 Lexical](../../spec/grammar.md#6-lexical-structure), [§10 Grammar](../../spec/grammar.md#10-grammar-kernel)

## Goal

Turn `.af` source text into an AST using participle, with positioned diagnostics
and no panics on malformed input.

## Scope

### In scope
- Lexer rules (spec §6).
- AST node types (generic block model).
- `Parse()` entry point.
- Golden AST for [examples/review.af](../../examples/review.af).

### Out of scope (deferred)
- Semantic meaning of fields (M2).
- Level B semantics (M9) — parse only, reject at resolve with `AF150`.

## Packages & files

- `internal/parser/lexer.go`
- `internal/parser/parser.go`
- `internal/ast/ast.go`

## Tasks

- Lexer via `lexer.MustSimple` (longest-match first):
  - `Comment`, `String`, `Number`, `Ident` (hyphen-safe), `Op` (`->|==|!=`),
    `Punct`, `Whitespace`; elide Comment/Whitespace.
  - **No semicolon token** (spec §3.4).
- AST structs with embedded `lexer.Position`:
  - Generic `Block`, `Field`, `Value`; `TypeDecl`; `Flow` with `Entry`, `Params`, `Items`.
  - Step union: `Chain`, `Parallel`, `Branch`, `Loop`, `Repeat`, `Call`, `Ref`.
  - `Repeat` parses `repeat { ... } until ( cond [, max N] )` — condition **after**
    the body (do-while); `Loop` keeps the condition before. Both are Level A.
  - `Ref` / `Call` may carry `as <ident>` value-label aliases.
  - `Branch` head is `value-ref` (value label or `it` — `it` rejected at M2 until M9).
  - Flow header fields: `on`, `in`, `out`, `return` (all generic fields).
- `Parse(filename, src) (*ast.AST, diag.Diagnostics)` with `UseLookahead(2)`.
- Level B productions (`use ... as`, `params`, `call`, `each`, `it`) parse successfully; M2
  rejects with `AF150`.

## Acceptance criteria

- Golden AST snapshot for [examples/review.af](../../examples/review.af).
- Golden AST snapshots for the supplementary architecture fixtures
  [examples/pipeline.af](../../examples/pipeline.af),
  [examples/research.af](../../examples/research.af),
  [examples/critic.af](../../examples/critic.af) (cover `->` chain, `parallel`/`gather`,
  and `repeat ... until`).
- `a->b` lexes as `a`, `->`, `b`.
- `repeat { ... } until (verdict == pass, max 3)` parses into a `Repeat` node.
- Semicolon in source -> parse error or unexpected token (not valid syntax).
- Malformed inputs produce positioned `AF000` diagnostics; never panic.

## Dependencies

- M0 (`diag`).

## Risks / notes

- `field` vs `step`: disambiguate on `:` lookahead.
- AST carries no semantic decisions.
