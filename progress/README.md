# AgentFlow Progress Log

Daily record of **what was done** during implementation.

- **Execution plans:** [`implementation-plans/`](../implementation-plans/)
- **Milestone specs:** [`plans/`](../plans/)

## Naming

One file per calendar day worked: `YYYY-MM-DD.md`

Each daily file links to the active implementation plan at the top.

## Git workflow

Implementation changes go through **feature branches and pull requests** — not direct commits to `main`. See [`.cursor/rules/git-workflow.mdc`](../.cursor/rules/git-workflow.mdc).

### Branch ownership (required on active milestone days)

At the top of each daily file, record who owns active branches so parallel worktrees do not collide:

```markdown
## Branch ownership

| Branch | Worktree / location | PR | Status |
|--------|---------------------|-----|--------|
| `feat/m1-lexer-parser` | `/path/to/worktree` | #2 | active |

Notes: <what remains, e.g. "commits 1–2 done; finish parser + goldens">
```

Rules:

- **One branch, one worktree** — do not implement the same remote branch from two paths.
- Before starting work, run `git worktree list` and check this table.
- Clear or mark **merged** when the PR lands.

### After each PR (required)

Update this daily file whenever a PR is opened, updated, or merged:

| Event | Action |
|-------|--------|
| **PR opened** | Add or refresh the **Branch ownership** row (`active`, PR link) |
| **PR updated** (new commits pushed) | Update **Completed**, **Verification**, and ownership notes |
| **PR merged** | Set status `merged`; record what shipped; set **Next session** |

Progress commits belong on the same feature branch as the PR when possible. See [`.cursor/rules/git-workflow.mdc`](../.cursor/rules/git-workflow.mdc).
