<!-- agentflow: trigger=/cl-review flow=cl_review in=ChangeRef -->

1. Use the `reviewer` subagent (`/reviewer`).
2. Use the `executor` subagent (`/executor`) using the output from `reviewer`.
