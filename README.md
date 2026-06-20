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

## Learn more

- [OVERVIEW.md](OVERVIEW.md) — architecture, data model, and end-to-end flow
- [spec/grammar.md](spec/grammar.md) — language definition
- [plans/README.md](plans/README.md) — implementation roadmap
- [examples/review.af](examples/review.af) — canonical MVP fixture
