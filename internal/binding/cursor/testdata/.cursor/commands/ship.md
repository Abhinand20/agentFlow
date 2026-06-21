<!-- agentflow: trigger=/ship flow=ship in=Ticket -->

1. Use the `build` subagent (`/build`) (step `code_review.build`).
2. Run `scripts/test.sh` in the terminal. If the gate fails, go back to step `build` and re-run from there. Retry the script up to `2` times first.
3. Launch the following subagents in parallel using multiple Task calls in one message: lint, security, style.
4. Collect the outputs from `lint`, `security`, `style` and give them to `reviewer`. Use the output from `code_review.build` as sequential context.
5. Repeat the following steps while `code_review.review` equals `revise`, at most `3` times (advisory — track your iteration count and stop at 3):
  - Use the `build` subagent (`/build`) (step `code_review.loop.build`) using the output from `code_review.reviewer`.
  - Run `scripts/test.sh` in the terminal. If the gate fails, go back to step `build` and re-run from there. Retry the script up to `2` times first.
  - Use the `reviewer` subagent (`/reviewer`) (step `code_review.loop.reviewer`) using the output from `code_review.loop.build`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: approve, revise, reject.
6. Read the `out:` value from `code_review`. Then:
  - if `code_review` is `approve`, do:
    - Use the `deploy` subagent (`/deploy`) using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
  - if `code_review` is `revise`, do:
    - Use the `notify_author` subagent (`/notify_author`) using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
  - if `code_review` is `reject`, do:
    - Use the `notify_author` subagent (`/notify_author`) using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
