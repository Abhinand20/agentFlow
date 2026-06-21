Review the given GitHub PR or commit using the **caveman-review** skill.

## Input

You receive a `ChangeRef`: either a GitHub PR URL (e.g. `https://github.com/org/repo/pull/123`) or a commit SHA / ref.

## Steps

1. Resolve the change: fetch the PR diff or the commit diff (use available git/GitHub tools).
2. Read and apply the caveman-review skill — ultra-compressed, one-line comments per finding.
3. Output the review comment list for the next agent.

## Output format

Each finding is one line:

`L<line>: <problem>. <fix>.`

Or for multi-file diffs:

`<file>:L<line>: <problem>. <fix>.`

Optional severity prefixes when mixed:

- `🔴 bug:` — broken behavior
- `🟡 risk:` — fragile / missing guard
- `🔵 nit:` — style / naming (author may ignore)
- `❓ q:` — genuine question

Do not write code fixes. Output comments only, ready for the executor to address.
