<!-- agentflow: trigger=/ship flow=ship in=Ticket -->

1. Act as the `build` agent (step `code_review.build`) using the instructions in the `build` rule.
2. Run `scripts/test.sh` in the terminal. If the gate fails, Go back to step `build` and re-run from there. Retry the script up to `2` times first.
3. Run the following agents one after another (parallel execution is not available on Cursor): lint, security, style.
4. Collect the outputs from `lint`, `security`, `style` and give them to `reviewer`. Use the output from `code_review.build` as sequential context.
5. Repeat the following steps while `code_review.review` equals `revise`, at most `3` times (advisory — track your iteration count and stop at 3):
  - Act as the `build` agent (step `code_review.loop.build`) using the instructions in the `build` rule using the output from `code_review.reviewer`.
  - Run `scripts/test.sh` in the terminal. If the gate fails, Go back to step `build` and re-run from there. Retry the script up to `2` times first.
  - Act as the `reviewer` agent (step `code_review.loop.reviewer`) using the instructions in the `reviewer` rule using the output from `code_review.loop.build`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: approve, revise, reject.
6. Read the `out:` value from `code_review`. Then:
  - if `code_review` is `approve`, do:
    - Act as the `deploy` agent using the instructions in the `deploy` rule using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
  - if `code_review` is `revise`, do:
    - Act as the `notify_author` agent using the instructions in the `notify_author` rule using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
  - if `code_review` is `reject`, do:
    - Act as the `notify_author` agent using the instructions in the `notify_author` rule using the output from `code_review.loop.reviewer`. Read the last `agentflow-output` block. If it is missing or invalid, re-invoke the agent up to `0` times, then stop the flow. Allowed values: shipped, rejected.
