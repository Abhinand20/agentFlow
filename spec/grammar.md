# AgentFlow Language Specification

- **Status:** Draft / v0.1 (kernel)
- **File extension:** `.af`
- **Pronounced:** "agent-flow"
- **Canonical MVP program:** [examples/review.af](../examples/review.af)

AgentFlow is a small declarative language for describing multi-agent systems
(agents, their typed contracts, and the flows that compose them) and compiling
them into platform-native configuration for hosts like Claude Code and Cursor.

This document specifies the language: its goals, principles, language levels,
the execution/data model, the kernel grammar, and how a program is resolved and
lowered.

---

## 1. Goals

1. **One source of truth for an agent team.** You describe agents, capabilities,
   and how they collaborate in a single `.af` file. Everything a host needs is
   generated from it.
2. **Compiler, not a runtime.** AgentFlow does not execute agents and never makes
   network calls. It parses, validates, and emits configuration/instructions.
3. **A tiny, stable kernel that is theoretically complete.** Control flow uses
   sequence, selection, iteration, concurrency, and abstraction. Architectures
   such as supervisor and fan-out are library flows, not keywords.
4. **Typed contracts between agents.** Routing reads declared enum outputs via a
   formal output protocol. **Data identity through the graph is the load-bearing
   design constraint** — control labels, value labels, latest outputs, flow I/O, and gather payloads
   are defined precisely below.
5. **Portability with honesty.** The same program targets multiple hosts. Unsupported
   constructs warn and emit documented fallbacks (capability negotiation).
6. **Ergonomic and readable.** A trivial pipeline is a few lines; advanced
   composition does not contaminate the simple case.

### Non-goals (for the kernel)

- No general-purpose expression language (no arithmetic, string ops, user-defined
  predicates). AgentFlow expresses **composition, not computation**.
- No mutable program state in v0.1 (blackboard deferred).
- No host-specific concepts in the core grammar.

---

## 2. Design principles

1. **Small typed kernel, open periphery.**
2. **Composition is the primitive** — agents and flows compose; subflows nest.
3. **Patterns are libraries, not syntax.**
4. **Theoretically complete control flow** — sequence + selection + iteration +
   concurrency + abstraction.
5. **Typed contracts over stringly data** — enum outputs + output protocol in v0.1.
6. **Declarative and denotational** — a program denotes a resolved flow graph.
7. **Composition, not computation** — flat conditions only (`step == value`).
8. **Capability negotiation** — bindings declare support; compiler diffs and warns.
9. **Determinism is a spectrum** — native runbook (best-effort), hooks (partial
   enforcement), SDK runtime (deterministic). See section 12.
10. **Tooling-first** — diagnostics-first compiler; grammar stays LL-friendly.

---

## 3. Language levels

AgentFlow uses one grammar surface, but **three semantic levels**. This removes
roadmap ambiguity: the parser may accept more than v0.1 executes.

### 3.1 Level A — v0.1 semantic subset (MVP)

**Executable today.** The compiler must accept, validate, lower, and emit for
these constructs:

| Category | v0.1 constructs |
|----------|-----------------|
| Declarations | `use`, `type` (enum only), `agent`, `gate`, `flow`, `entry flow` |
| Flow header fields | `on`, `in`, `out` (optional — inferred), `return` (optional — defaults to last producer) |
| Steps | bare ref (agent/gate/subflow), `ref as value`, `a -> b`, `parallel { ... } gather g as value`, `branch value { case v -> ... }`, `loop (until value == v, max N) { ... }`, `repeat { ... } until (value == v, max N)` |
| Conditions | `value == enum` / `value != enum` (value = value label in scope) |
| Types | declared enums; opaque nominal types for unstructured `in:` (e.g. `Ticket`) |
| Capabilities | inline `use { kind: ... }` only |
| Gates | `on-fail: halt \| retry \| goto \| enter-loop`; `on-fail-target:` required for `retry` / `goto` |

**Not executable in v0.1** (even if parsed): flow `params`, `call(...)`, trailing
flow blocks, `parallel each`, `it` (explicit keyword — sequential binding uses
previous step implicitly in lowering).

### 3.2 Level B — parsed but disabled (v0.2 target)

Syntax is in the grammar; the resolver rejects with `AF150 unsupported in v0.1`
until M9 lands:

- `flow F(a, b) { ... }` — parameters
- `F(args) { trailing block }` — calls with arguments
- `parallel each items as x { ... }` — dynamic fan-out
- `it` as an explicit value reference in conditions
- `use std.patterns as p` — requires abstraction + embedded stdlib

### 3.3 Level C — post-MVP syntax

Scheduled after v0.2; not in the parser until their milestone:

- Record types and `outputs { field: Type }`
- Dotted conditions (`step.field == v`)
- `policy { ... }`, agent `tags`
- `test "..." { stub; run; expect }`
- `use pkg@semver as alias` registry resolution
- Linear Plan IR consumers (`af test`, `--runtime sdk`)

### 3.4 Statement separators

**There is no statement separator.** Steps are separated by newlines (or whitespace).
Semicolons are **not** part of the grammar. Examples must not use `;`.

---

## 4. Execution and data model

This section is normative. The compiler, IR, render layer, and bindings must agree
on these identities.

### 4.1 Names and scopes

- **Declaration name** — the identifier on `agent`, `gate`, or `flow` (e.g.
  `reviewer`, `code_review`).
- **Step instance** — one occurrence of an agent, gate, or subflow inside a flow
  body.
- **Control label** — the unique name of a step instance within a flow scope, used
  for runbook step numbers and gate targets.
- **Value label** — the name of the latest output slot written by a step instance,
  used by conditions, `return:`, and parent-flow branches.

**Label assignment (v0.1):**

- Default control label = declaration name.
- Default value label = control label for agents and subflows with output.
- `ref as name` keeps the step instance's control label but writes its latest
  output to value label `name`.
- A value label may be written more than once in a sequential or loop lineage if
  every writer has the same output type. This models "latest verdict" patterns
  such as `reviewer as review` before and inside a retry loop.
- Two step instances in the same flow scope must not have the same control label
  unless at least one uses `as`; the normalizer then assigns stable occurrence
  labels for runbook/DOT output (for example `reviewer`, `reviewer#2`, or a
  deterministic equivalent). `AF208` reports ambiguous implicit labels.
- After inlining a subflow, inner control/value labels are prefixed:
  `code_review.build`, `code_review.review`, etc.

### 4.2 Values and latest output

- Every agent with `out: SomeEnum` produces a **latest output** value after it
  runs successfully (parsed from the output protocol; see section 9).
- Agents without `out:` produce unstructured **text** output (not used in enum
  routing in v0.1).
- Gates produce no typed output; they succeed or fail.
- A subflow step instance exposes the subflow's **return binding** as its latest
  output (section 4.4).

**Latest output** of value label `V` is written `V` in conditions
(`review == approve`).

### 4.3 Flow input

- `in: TypeName` on a flow declares the type of the flow's external input.
- For the entry flow, input comes from the trigger argument (e.g. `/ship
  TICKET-123` → input bound as flow input).
- **Opaque nominal types** (e.g. `Ticket`) need not have a `type` declaration in
  v0.1; they are unstructured payloads passed through to prompts/runbook text.
- An agent step whose `in:` matches the enclosing flow's `in:` receives the flow
  input implicitly (plus any sequential context from prior steps).

### 4.4 Flow output binding

Flow header fields:

- `in: TypeName` — input type (enum, opaque nominal, or `text`).
- `out: TypeName` — output type the flow exposes to its caller. **Optional**; if
  omitted it is inferred from the resolved return value's type.
- `return: valueLabel` — value label whose latest output becomes the flow's
  output. **Optional**; defaults to the terminal producer (see Rule 0).

**Rules:**

0. **Default return (sugar).** If `return:` is omitted and the flow body is a
   sequence/loop/`repeat` (not branch-terminal), the flow returns the latest
   output of its **last step instance** — the terminal producer. If that step
   produces no typed/text output (e.g. a gate), or it is not the value you want
   (e.g. a critic-refiner whose final step is the critic but whose answer is the
   draft), you **must** write `return:` explicitly. This default removes ceremony
   from linear pipelines without introducing a "last step wins" surprise: it
   applies only to the unambiguous terminal-producer case and is otherwise an
   error (`AF209`).
1. When present, `return:` names a value label in the flow whose latest output
   becomes this flow's output. Types must match: value label's enum type must
   equal flow `out:` (when `out:` is declared).
2. If the flow ends in a `branch` with no `return:`, each branch leaf step (or
   subflow) must produce `out:` matching the flow's `out:` type. Example: `ship`
   branches to `deploy` and `notify_author`, both `out: Decision`.
3. Referencing a subflow's output from a parent: use the subflow value label in
   `branch code_review { case approve -> ... }` — the value is the subflow's
   return output.

There is **no** "last step wins" rule: the Rule 0 default is the *terminal
producer*, which is well-defined only for non-branch flows and is reported as an
error when ambiguous.

### 4.5 Sequential data flow

Within a sequence (including loop bodies):

- The **previous step's latest output** is available to the next step implicitly.
- The render layer/runbook passes it in prose ("using the output from Step N").
- No `it` keyword in v0.1 source; the compiler may introduce an internal name
  during lowering.

### 4.6 Branch conditions

```text
branch <valueLabel> {
  case approve -> deploy
  case reject  -> notify_author
}
```

- `<valueLabel>` must refer to a value label whose latest output is an enum.
- Each `case` value must be a member of that enum.
- Exhaustiveness over reachable enum values is recommended (warning `AF204`).

### 4.7 Loop scoping

```text
loop (until review != revise, max 3) {
  build
  reviewer as review
}
```

- `until` is checked **before** each iteration and again after each iteration.
  If the condition is already true, the loop runs zero times.
- The condition uses value labels visible before the loop or written inside the
  loop body. Rewriting the same value label inside the body updates the value
  tested by the next iteration.
- `max` is a hard upper bound on iterations; enforced as advisory in native
  runbook, hook-assist where available, deterministic in SDK runtime.
- If `max` is reached while `until` is still false, the loop exits with the latest
  value labels as-is. Downstream branches must handle those values.
- Steps inside the loop body get control labels such as `loop.build`,
  `loop.reviewer` after normalization (exact prefix scheme is implementation-defined
  but stable).

#### 4.7.1 `repeat` (do-while)

```text
repeat {
  generate as draft
  critic as verdict
} until (verdict == pass, max 3)
```

`repeat` is `loop` with the condition checked **after** the body instead of
before. It is pure sugar over iteration — it adds no new data semantics — but it
removes the most common wart in v0.1: the "run once, then loop" prologue
duplication where a generator/critic pair must be written both before and inside
a `loop`.

- The body runs **at least once**; `until` is evaluated after each iteration.
- Value labels read by `until` (e.g. `verdict`) need not exist before the first
  iteration — they are written by the body. A value label referenced inside the
  body but not yet written (e.g. a critic's feedback on the first pass) resolves
  to **empty/absent** and is passed through as "no prior value."
- `max` is the hard upper bound, enforced exactly as in `loop` (advisory native,
  hook-assist, deterministic SDK). On reaching `max` with `until` still false, the
  flow continues with the latest value labels as-is.
- Body steps get stable control labels under a `repeat`-scoped prefix, analogous
  to `loop`.
- The critic's textual output flows to the next iteration's generator as ordinary
  sequential context (§4.5); only the enum drives `until`. This is why a separate
  "feedback" output slot is **not** required in v0.1.

`loop` (check-first / while) and `repeat` (check-after / do-while) are the two
iteration forms; both are bounded by `max`.

### 4.8 Parallel and gather payloads

```text
parallel {
  lint
  security
  style
} gather reviewer as review
```

- Each branch runs independently; each branch step produces a latest output
  (text if no enum `out:`).
- **Gather payload** — a fixed record keyed by branch control label:
  `{ lint: <text>, security: <text>, style: <text> }`.
- The gather step (`reviewer as review`) receives:
  1. Sequential context: latest output of the step immediately before `parallel`
     (`build` in the golden program).
  2. The gather payload (all branch outputs).
- v0.1 does not surface gather payload in the type system (no record types); the
  runbook and gather agent prompt describe the bundle explicitly. Record types
  (Level C) will type this bundle.

### 4.9 Terminals

`done` and `fail` are implicit terminals for explicit early exit (optional in
v0.1 examples). They produce no output.

---

## 5. The kernel primitives

| # | Primitive | Surface form | Models |
|---|-----------|--------------|--------|
| 1 | Sequence | `a -> b` or ordered steps | pipeline |
| 2 | Selection | `branch value { case v -> ... }` | routing |
| 3 | Iteration | `loop (until …, max N) { ... }` (while) or `repeat { ... } until (…, max N)` (do-while) | bounded loops |
| 4 | Concurrency | `parallel { ... } gather g` | fan-out |
| 5 | Reference | bare `name` | agent / subflow |
| 6 | Abstraction | Level B — `flow F(params)`, calls | patterns library |

Patterns (supervisor, consensus, etc.) decompose into primitives; see OVERVIEW.

---

## 6. Lexical structure

```ebnf
comment      = "#" { any-char-except-newline } ;
string       = '"' { "\\" any-char | any-char-except-quote } '"' ;
number       = digit { digit } [ "." digit { digit } ] ;
boolean      = "true" | "false" ;
ident        = letter { letter | digit | "_" } { "-" ( letter | digit | "_" ) { letter | digit | "_" } } ;
qual-name    = ident { "." ident } ;
op           = "->" | "==" | "!=" ;
```

Lexer ordering: `->`, `==`, `!=` before single-char punct. Identifiers never end
with `-` (so `a->b` lexes correctly). **No semicolon token.**

`${NAME}` in strings is resolved by bindings at emit time, not by the parser.

---

## 7. Declarations

### 7.1 `use` — capabilities (v0.1: inline only)

```text
use anthropic {
  kind: model-provider
  models: [opus, sonnet, haiku]
}
```

See section 8 for model resolution.

### 7.2 `type` — enums (v0.1)

```text
type Verdict = approve | revise | reject
```

Enums used in routing must be declared. Flow/agent `in:` may reference **opaque
nominal** types without a declaration (unstructured input).

Builtin: `text` (unstructured output).

### 7.3 `agent`

```text
agent reviewer {
  model: opus
  in: Ticket          # optional; opaque nominal OK
  out: Verdict        # required for routing; enum only in v0.1
  tools: [github.get_pr]
  permissions: supervised
  retry: 2            # output-protocol parse retries (section 9)
  prompt-file: "prompts/reviewer.md"
}
```

Known fields interpreted in v0.1: `model`, `in`, `out`, `tools`, `permissions`,
`retry`, `prompt`, `prompt-file`, `description`. Unknown fields warn and are preserved.

#### 7.3.1 Prompt source

An agent prompt can be supplied three ways. `prompt:` is overloaded: it accepts
either inline text **or** a path to a markdown file.

| Field | Meaning |
|-------|---------|
| `prompt: "Implement the change."` | Inline prompt text, useful for short agents and examples. |
| `prompt: "prompts/reviewer.md"` | A path to a UTF-8 markdown prompt file (see detection rule below). The file content becomes the prompt. |
| `prompt-file: "prompts/reviewer.md"` | Explicit file form — always a path; never interpreted as inline text. |

**Path detection for `prompt:`** — a `prompt:` value is treated as a **file path**
when it ends with `.md` (case-insensitive). Any other value is inline text. This
is the only heuristic; it keeps the common case (inline text) unambiguous while
making the file case ergonomic without a second field name. Use `prompt-file:`
when you want to force file semantics regardless of the value.

Rules:

1. `prompt` and `prompt-file` are mutually exclusive (`AF211`).
2. A prompt **path** — whether `prompt-file:` or a `.md`-valued `prompt:` — must be
   a relative path, must stay under the source file's directory, and must exist
   and be valid UTF-8 at compile time (`AF211`). A `.md`-valued `prompt:` that does
   **not** resolve to a readable in-tree file is an `AF211` error, not a silent
   fallback to inline text.
3. Path resolution is relative to the `.af` file that declares the agent.
4. The resolved prompt text (inline or file content) is copied into the semantic
   model/IR before rendering; generated host artifacts do not depend on the
   original file at runtime.
5. Output-protocol instructions (section 9) are appended after resolving the
   final prompt text, whatever its source.
6. Environment references such as `${NAME}` inside prompt text or files are
   preserved for bindings; AgentFlow does not expand them during parsing or
   resolution.

### 7.4 `gate`

```text
gate quality {
  run: "scripts/test.sh"
  on-fail: retry          # halt | retry | goto | enter-loop
  on-fail-target: build   # required for retry/goto
  behavior: blocking     # blocking | advisory
  retry: 2                # script exit-code retries before on-fail action
}
```

#### 7.4.1 Gate failure policy (`on-fail`)

| Fields | Meaning |
|--------|---------|
| `on-fail: halt` | Stop the flow; report failure. |
| `on-fail: retry` + `on-fail-target: <controlLabel>` | Re-run the named control label and its downstream sequence until the gate is hit again. |
| `on-fail: goto` + `on-fail-target: <controlLabel>` | Jump orchestration to the named control label without counting as a full retry of intermediate steps. |
| `on-fail: enter-loop` | Re-enter the innermost enclosing `loop` body. Valid only if the gate is inside a loop. |

`bounce-back` is **not** valid. Use `retry` or `goto` with `on-fail-target`.

`behavior: advisory` — failure is reported but does not block (native runbook only;
blocking requires `behavior: blocking` + host hook support).

### 7.5 `flow`

```text
entry flow ship {
  on: "/ship"
  in: Ticket
  out: Decision
  code_review
  branch code_review {
    case approve -> deploy
    case revise  -> notify_author
    case reject  -> notify_author
  }
}
```

Header fields (all optional except one `entry` flow per program):

| Field | Role |
|-------|------|
| `on:` | Trigger pattern (slash command) |
| `in:` | Input type |
| `out:` | Output type (optional — inferred from the return value when omitted) |
| `return:` | Value label whose latest output is this flow's output (optional — defaults to the terminal producer, §4.4 Rule 0) |

---

## 8. Model and provider resolution

- Agent `model:` is a **symbolic alias** (e.g. `opus`, `sonnet`) or a
  **qualified** reference `provider.alias` (e.g. `anthropic.opus`).
- Resolution searches `use` blocks with `kind: model-provider` and a `models:`
  list containing the alias.
- **Unqualified alias** must resolve to exactly one provider; otherwise `AF110`
  ambiguous model (list candidates in diagnostic).
- **Qualified** form selects that provider; error if alias not in its `models`.
- Bindings map resolved `(provider, alias)` to host-native model id strings
  (e.g. `claude-opus-4-20250514`). The `.af` file keeps symbolic names.
- Aliases in `models:` are **required** for every model an agent references; there
  is no implicit global alias table in v0.1.

---

## 9. Output protocol contract

Agents with `out: SomeEnum` must emit a machine-readable block so the orchestrator
can route `branch` and `loop`.

### 9.1 Prompt injection (compiler-generated)

Appended to the resolved agent prompt text (inline `prompt` or `prompt-file`):

```text
When you have finished, end your reply with exactly one fenced block in this form
(no other text after the closing fence):

```agentflow-output
out: <value>
```

`<value>` must be exactly one of: <comma-separated enum members>.
```

The output **field name is always `out`** in v0.1 (the single output slot). Level C
multi-output will use declared field names.

### 9.2 Orchestrator parse algorithm

1. Take the **last** ```agentflow-output fenced block in the subagent's final
   message (search from bottom).
2. Parse a single line `out: <ident>` (optional whitespace); ignore `#` comments.
3. `<ident>` must be a member of the agent's declared enum type.
4. On success, write the enum value to the step instance's value label.

### 9.3 Parse failure and retries

| Event | Behavior |
|-------|----------|
| Missing block | Parse error |
| Malformed line | Parse error |
| Unknown enum member | Parse error |
| Parse error | Re-invoke the same agent up to `retry:` times (default `0`); then **halt** the flow with an orchestration error |
| Success after retry | Continue with parsed value |

Bindings must instruct the orchestrator to follow this table in the runbook
("If the block is missing or invalid, re-invoke up to N times, then stop.").

### 9.4 Invalid enum at runtime

Treated as parse error (not silently coerced). No default branch is taken.

---

## 10. Grammar (kernel)

```ebnf
file        = { decl } ;
decl        = use | type-decl | agent | gate | flow ;

use         = "use" ident "{" { field } "}" | "use" qual-name "as" ident ;  (* alias form = Level B *)
type-decl   = "type" ident "=" ident { "|" ident } ;
agent       = "agent" ident "{" { field } "}" ;
gate        = "gate"  ident "{" { field } "}" ;

flow        = [ "entry" ] "flow" ident [ params ] "{" { flow-item } "}" ;
params      = "(" param { "," param } ")" ;       (* Level B *)
param       = ident [ ":" type-ref ] ;
type-ref    = ident ;

flow-item   = field | step ;
field       = ident ":" value ;

step        = chain | parallel | branch | loop | repeat | call | ref ;
chain       = atom { "->" atom } [ edge-attr ] ;
atom        = call | ref ;
ref         = qual-name [ alias ] ;
call        = qual-name "(" [ args ] ")" [ "{" { step } "}" ] [ alias ] ;  (* call = Level B *)
alias       = "as" ident ;

parallel    = "parallel" [ "each" qual-name "as" ident ]
              "{" { step } "}" [ "gather" atom ] ;
branch      = "branch" value-ref "{" { case } "}" ;
case        = "case" ident { "," ident } "->" step ;
loop        = "loop" "(" [ "until" cond ] [ [","] "max" number ] ")"
              "{" { step } "}" ;
repeat      = "repeat" "{" { step } "}"
              "until" "(" cond [ [","] "max" number ] ")" ;

edge-attr   = "[" [ "when" cond ] [ [","] "max" number ] "]" ;
cond        = value-ref ( "==" | "!=" ) ident ;
value-ref   = qual-name | "it" ;                (* "it" = Level B *)

value       = string | number | boolean | qual-name | list ;
list        = "[" [ value { "," value } ] "]" ;
```

Parsing notes: `field` vs `step` disambiguated by `:` lookahead; `call` vs `ref`
by `(`. Level B productions parse in v0.1 but fail resolution with `AF150`.

---

## 11. Host capability matrix (v0.1)

Bindings declare which features they support. The compiler diffs program needs vs
this matrix and emits `AF3xx` warnings.

| Capability | claude-code (MVP target) | cursor (M10) | sdk (M15) |
|------------|--------------------------|--------------|-------------|
| Command trigger (`on:`) | yes — `.claude/commands/` | yes — `.cursor/commands/` | yes — CLI entry |
| Named subagent files | yes — `.claude/agents/` | yes — `.cursor/agents/` | yes — SDK agents |
| MCP config emission | yes — `.mcp.json` | yes — `.cursor/mcp.json` | yes |
| Lifecycle hooks | yes — settings hooks | beta — `.cursor/hooks.json` | N/A (in-process) |
| Parallel subagent spawn | yes — Task (advisory) | advisory — Task (multiple calls in one message) | yes — async |
| Blocking gates | yes — hook exit 2 | advisory fallback | yes — exit code |
| Output protocol parse | advisory — runbook instructs | advisory | deterministic |
| File layout stability | `.claude/**` | `.cursor/**` | generated project |

---

## 12. Runtime guarantees by target

| Target | Control flow | Gates | Loop bounds | Output parse | Parallelism |
|--------|--------------|-------|-------------|--------------|-------------|
| **Native runbook** (default) | Best-effort — host LLM follows markdown steps | Blocking only where hooks exist; else advisory text | Advisory counter in runbook | Advisory re-invoke per section 9 | Advisory "spawn together" wording |
| **Hook-enforced** (native + hooks) | Same + gate script enforced | **Hard** — hook blocks on non-zero exit | Partial — counter file / hook | Same | Same |
| **SDK runtime** (`--runtime sdk`, M15) | **Deterministic** — generated code | **Hard** — subprocess exit code | **Hard** — for-loop | **Hard** — parser in code | **Hard** — async/await |

The spec's resolved flow semantics are identical across targets; only enforcement
differs. Bindings document fallbacks in emitted README or build warnings.

---

## 13. Resolution and lowering

```
source
  -> parse            (tokens -> AST)
  -> resolve          (AST -> model; reject Level B with AF150)
  -> inline/normalize (expand subflows; assign labels; gather payloads)
  -> validate         (rules AF2xx)
  -> IR               (binding-agnostic)
  -> render           (IR -> instruction text)
  -> bind             (per-target file layout)
  -> write
```

### 13.1 Validate (v0.1 rule set)

- `AF200` duplicate declaration names
- `AF201` node resolution
- `AF202` `in`/`out` types: enum declared or opaque nominal / `text` allowed for `in`
- `AF203` branch/loop conditions: value label has enum `out` containing case value
- `AF204` (warning) branch exhaustiveness
- `AF205` cycles require `max`
- `AF206` qualified tool refs exist
- `AF207` (warning) orphan nodes
- `AF208` ambiguous duplicate implicit control/value labels in same flow scope
- `AF209` `return:` value label exists and output type matches flow `out:`; when
  `return:` is omitted, the default terminal producer (§4.4 Rule 0) must exist and
  carry a typed/text output, else `AF209` (ambiguous or missing default return)
- `AF210` branch-terminal flows: each leaf output type matches flow `out:`
- `AF211` prompt source invalid (`prompt` + `prompt-file` together; a prompt path
  — `prompt-file:` or a `.md`-valued `prompt:` — that is absolute, escapes the
  source directory, is missing, unreadable, or invalid UTF-8)

Validation runs **after** inlining.

### 13.2 IR and bindings

- **Declarative facts** → agent files, MCP, settings.
- **Resolved flow + data model** → render layer → runbook markdown.
- See [plans/mvp/06-rendering-layer.md](../plans/mvp/06-rendering-layer.md).

---

## 14. Canonical golden program

The single authoritative MVP fixture is
[examples/review.af](../examples/review.af). The spec, plans, golden AST/IR/render
snapshots, and E2E tests must all use this file unchanged unless the spec version
bumps.

Properties it demonstrates:

- Inline capabilities, enum types, opaque `Ticket` input
- Agent prompt file via `prompt-file: "prompts/reviewer.md"`
- Sequence, parallel/gather, gate with `on-fail: retry` + `on-fail-target: build`, bounded loop
- Subflow with `return: review`; parent `branch code_review`
- Entry flow `on: "/ship"` with branch-terminal outputs (`Decision`)

**Supplementary architecture examples** (also Level A; exercise `repeat` and
default `return:`, but `review.af` remains the regold anchor):

- [examples/pipeline.af](../examples/pipeline.af) — sequential pipeline
  (Research → Analyze → Write → Edit) with `->` chaining and default `return:`.
- [examples/research.af](../examples/research.af) — supervisor/worker fan-out
  with `parallel { ... } gather`.
- [examples/critic.af](../examples/critic.af) — generator/critic refinement with
  `repeat { ... } until`.
- [examples/docs.af](../examples/docs.af) — prompts loaded from markdown files via
  both the `prompt:` path form and explicit `prompt-file:` (§7.3.1).

---

## 15. Deferred (by language level)

See section 3. Mapped to plans:

| Level | Feature | Plan |
|-------|---------|------|
| B | Abstraction, std/patterns | M9 |
| B | Cursor + negotiation | M10 |
| B | Registry, fmt, diagnostics | M11 |
| C | Records, multi-output | M12 |
| C | Policies | M13 |
| C | Plan IR, simulator, `af test` | M14 |
| C | SDK runtime | M15 |
| C | LSP | M16 |

---

## 16. Open questions

- Exact inner label prefix scheme (`code_review.reviewer` vs `code_review/reviewer`)
  — must be stable once published.
- Gather payload typing when record types land (M12).
