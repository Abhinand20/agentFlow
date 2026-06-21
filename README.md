# AgentFlow

A small declarative language for describing multi-agent systems and compiling
them into working agents on hosts like Claude Code and Cursor.

## Why AgentFlow?

Multi-agent setups are usually hand-wired: prompts scattered across config files,
orchestration buried in markdown, and no shared model for how agents connect.
AgentFlow lets you describe your team once in a `.af` file — agents, typed
outputs, gates, and control flow — then validate and scaffold host-native
configuration. The host runs orchestration; AgentFlow is a **compiler**, not a
runtime.

## Example

```text
agent research {
  model: sonnet
  prompt: "Gather credible sources and notes on the topic."
}

agent write {
  model: sonnet
  prompt: "Draft the piece from the research."
}

flow publish {
  research -> write
}
```

## Status

The end-to-end MVP works today for **Cursor**: parse → resolve → inline →
validate → IR → render → bind, driven by the `af` CLI. Compiling a `.af` file
produces a working `.cursor/` config — native subagents, a slash command, and
`mcp.json`. Claude Code binding is the next target. See the
[roadmap](#roadmap--future-extensions).

## Try it

```bash
go build -o af ./cmd/af

af validate examples/review.af              # zero errors
af graph    examples/review.af              # resolved flow as DOT
af build    examples/review.af --target cursor --out .   # writes .cursor/
af build    examples/review.af --emit-ir    # print the binding-agnostic IR
```

A real, dogfooded example lives in [examples/cl-review.af](examples/cl-review.af)
— a Reviewer → Executor pipeline with Cursor model bindings
(`use cursor { kind: model-provider, models: [...] }`).

## Roadmap & future extensions

Near-term, on top of the working Cursor path:

- **Claude Code binding** (M7) — `af build --target claude-code` emits `.claude/`
  with hook-enforced gates.
- **Config import & round-trip** (M17) — `af import` reconstructs a `.af` file
  from a pre-defined `.cursor/`/`.claude/` tree, so existing agents become an
  editable AgentFlow source. See
  [plans/post-mvp/17-config-import-and-roundtrip.md](plans/post-mvp/17-config-import-and-roundtrip.md).
- **Abstraction & std/patterns** (M9), **registry + formatter** (M11),
  **SDK runtime** (M15), **LSP** (M16) — full list in
  [WALKTHROUGH.md §6](WALKTHROUGH.md#6-after-the-mvp-future-extensions).

## Learn more

- [OVERVIEW.md](OVERVIEW.md) — architecture, data model, and end-to-end flow
- [WALKTHROUGH.md](WALKTHROUGH.md) — plain-English tour, status, and roadmap
- [spec/grammar.md](spec/grammar.md) — language definition
- [plans/README.md](plans/README.md) — implementation roadmap
- [examples/review.af](examples/review.af) — canonical MVP fixture
