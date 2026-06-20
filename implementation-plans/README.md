# AgentFlow Implementation Plans

Execution plans for building AgentFlow. Milestone **design specs** live in
[`plans/`](../plans/); daily **work logs** live in [`progress/`](../progress/).

## Naming

`YYYY-MM-DD-<milestone-slug>.md`

## Index

| Date | File | Milestones | Status |
|------|------|------------|--------|
| 2026-06-19 | [2026-06-19-mvp-m0-m8.md](2026-06-19-mvp-m0-m8.md) | M0–M8 (umbrella) | Active |
| 2026-06-19 | [2026-06-19-m0-foundations.md](2026-06-19-m0-foundations.md) | M0 | Done |
| 2026-06-19 | [2026-06-19-m1-parser.md](2026-06-19-m1-parser.md) | M1 | Done |
| 2026-06-19 | [2026-06-19-m2-resolver.md](2026-06-19-m2-resolver.md) | M2 | Done |
| 2026-06-19 | [2026-06-19-m3-inline-normalize.md](2026-06-19-m3-inline-normalize.md) | M3 | Done |
| 2026-06-19 | [2026-06-19-m4-validation.md](2026-06-19-m4-validation.md) | M4 | Done |
| 2026-06-19 | [2026-06-19-m5-ir.md](2026-06-19-m5-ir.md) | M5 | Planned |
| 2026-06-19 | [2026-06-19-m6-render.md](2026-06-19-m6-render.md) | M6 | Planned |
| 2026-06-19 | [2026-06-19-m7-claude-binding.md](2026-06-19-m7-claude-binding.md) | M7 | Planned |
| 2026-06-19 | [2026-06-19-m8-cli-e2e.md](2026-06-19-m8-cli-e2e.md) | M8 | Planned |

## How these relate

- **Design specs** (`plans/mvp/`) — *what* each milestone must deliver, grounded in the
  language spec.
- **Execution plans** (this folder, per milestone) — *how* to build it: concrete Go types,
  algorithms, file layouts, test tables, commit sequence. Hand-off ready.
- **Work logs** (`progress/`) — *what actually happened*, day by day.

Read order for a milestone: design spec → execution plan → spec sections it cites.

## Status legend

`Draft` | `Active` | `Planned` | `Superseded` | `Done`
