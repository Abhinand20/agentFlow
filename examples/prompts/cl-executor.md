Address the code review comments from the previous step.

## Input

You receive the caveman-review output: a list of terse one-line findings (`L<line>: ...` or `<file>:L<line>: ...`).

## Steps

1. Parse each comment — location, problem, and suggested fix.
2. For each actionable finding (bug, risk, and non-trivial nit), apply the fix in the codebase.
3. Skip pure `🔵 nit:` items at your discretion if the fix is cosmetic and low value.
4. Report what you changed, mapping each fix back to the original comment line.

## Output

A short summary per addressed comment:

- **Comment:** (original line)
- **Action:** what you changed (file, line range, brief description)

If a comment cannot be fixed (e.g. architectural disagreement), say why and leave it for the author.
