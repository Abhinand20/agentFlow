# M10 — Cursor Binding (execution plan, lean first cut)

- **Date:** 2026-06-19 (lean cut landed 2026-06-20)
- **Milestone:** M10 (partial — lean first cut)
- **Design spec:** [plans/post-mvp/10-cursor-and-negotiation.md](../plans/post-mvp/10-cursor-and-negotiation.md)
- **Language spec:** [§11](../spec/grammar.md#11-host-capability-matrix-v01), [§12](../spec/grammar.md#12-runtime-guarantees-by-target)
- **Status:** **Partial — lean cut done** ([PR #10](https://github.com/Abhinand20/agentFlow/pull/10))
- **Depends on:** M5 (IR), M6 (render). Implements `binding.Binding` (M0).
- **Blocks:** M8 (`af build --target cursor`); full M10 negotiation framework

## 1. Goal

Assemble M6 render output into a working `.cursor/` configuration: slash commands,
agent rules, and MCP config. Cursor has no native named subagents — agents become
`.cursor/rules/*.mdc` files; parallel spawn and blocking gates fall back to advisory
wording with honest `AF3xx` warnings.

This is a **thin assembler** over M6 — no runbook logic in the binding.

> **Note:** Cursor was implemented before M7 (Claude) on a parallel track. Branch
> name `feat/m7-cursor-binding` is historical; PR title uses **M10**.

## 2. Lean cut — done (PR #10)

### Deliverables shipped

- `internal/binding/cursor/vocabulary.go` — Cursor `render.Vocabulary`
- `internal/binding/cursor/capabilities.go` — §11 row + capability-derived `Negotiate()`
- `internal/binding/cursor/cursor.go` — `Emit()` file assembly + registration
- `cmd/af/bindings.go` — blank import for `binding.Get("cursor")`
- Golden FS snapshots in `internal/binding/cursor/testdata/.cursor/**`

### Emitted file tree (`review.af`)

```
.cursor/
  commands/
    ship.md              # runbook body + HTML metadata comment (no YAML frontmatter)
  rules/
    build.mdc …          # one .mdc per reachable ir.Agent
  mcp.json               # mcpServers, transport defaults to stdio
```

### AF3xx codes (cursor-local)

| Code | When |
|------|------|
| AF300 | parallel used; Cursor runs sequential fallback |
| AF301 | bounded loop; advisory counter |
| AF302 | enum output parse; advisory |
| AF303 | blocking gate; hooks deferred |
| AF305 | model alias unmappable in rules |
| AF306 | tool refs metadata-only (MCP via mcp.json) |
| AF307 | permissions unmappable |
| AF308 | MCP JSON marshal failure |

`AF304` removed as per-program noise — agents-as-rules is a static binding caveat
(`StaticCaveat` in `capabilities.go`).

## 3. Remaining (full M10 / follow-up)

| Item | Milestone | Notes |
|------|-----------|-------|
| `.cursor/hooks.json` | M10 follow-up | Beta; blocking gates stay advisory until wired |
| `BUILD-NOTES.md` emit | M10 follow-up | Or stderr-only via M8 `af build` |
| `internal/binding/capability.go` | M10 follow-up | Shared negotiation framework for all bindings |
| DOT emitter | M8 / M7 | `internal/binding/dot/` for `af graph` |
| Manual smoke | M8 | `/ship TICKET-123` in Cursor IDE |
| CLI wiring | **M8** | `af build --target cursor`, pipeline, E2E |

## 4. Acceptance criteria

### Lean cut (done)

- [x] Golden FS snapshots for `review.af` pass
- [x] `reviewer.mdc` contains `agentflow-output` block
- [x] `ship.md` runbook: branch, loop, gate retry → `build`
- [x] `binding.Get("cursor")` resolves; `Capabilities()` matches §11 cursor row
- [x] AF300–AF303, AF305–AF306 for `review.af`; artifacts still emitted
- [x] Render frontmatter mapped (metadata comments + AF3xx for unmappable fields)

### Full M10 (remaining)

- [ ] `.cursor/hooks.json` for blocking gates (or permanent AF303 + docs)
- [ ] Shared negotiation in `internal/binding/capability.go`
- [ ] `BUILD-NOTES.md` or documented stderr fallback summary
- [ ] `af build --target cursor` E2E (M8)

## 5. Dependencies & PR stack

- **Hard dependency:** M6 render (PR #9). PR #10 stacks on `feat/m6-render`.
- **Retarget** PR #10 to `main` after M6 squash-merges.
- **M7 Claude binding** is independent; can land before or after M8 CLI.

## 6. Risks & notes

- Cursor command format has no YAML frontmatter — metadata uses HTML comments.
- Model/tools in rules are metadata comments + AF305/AF306, not host-enforced.
- Host-format drift confined to `internal/binding/cursor/` + regold.
