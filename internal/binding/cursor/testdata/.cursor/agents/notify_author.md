---
name: notify_author
description: "AgentFlow agent \"notify_author\" instructions"
model: inherit
---

Tell the author the change was rejected.

When you have finished, end your reply with exactly one fenced block in this form
(no other text after the closing fence):

```agentflow-output
out: <value>
```

`<value>` must be exactly one of: shipped, rejected.
