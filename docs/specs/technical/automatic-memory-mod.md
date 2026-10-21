# Automatic Memory Mod - Technical Spec

## 793d3a Orientation

The `automatic-memory-mod` is the [mod](../../GLOSSARY.md#mod) `engram-memory`, a hooks module in `agent-instructions/mods/engram-memory/` (informational) that `engram update` deploys through the skills machinery ([793d4b Deploying the mods of the source](./harness-deploy-sync.md#793d4b-deploying-the-mods-of-the-source)). It owns the routing of the four workflows below, the placement of the block beside a [prompt](../../GLOSSARY.md#prompt-automatic-memory) (the lookup is the commands'), the model calls, the bound of every wait, the set of delivered IDs, the showing of the lines for the user and the report of a failure; the commands, not the mod, hold the logic. It holds no behavior requirement of its own: the behavior of each command is specified in the file of the component that owns it. Tested in a real Claude Code [session](../../GLOSSARY.md#session) against an `engram` that prints chosen results.

Every judgement is one `$.model.complete` call of the mod: its instructions in `system`, the untrusted text escaped in `prompt`, a bound: a `timeoutMs` in a capture, and in the flow of a prompt or a tool call the mod's own deadline of 2 minutes for each wait, which aborts a model call through the `signal` of `$.model.complete`. Every mechanical step is an `engram auto <step>` command that reads one JSON object on standard input and prints a result with a known prefix, which the mod checks with the exit code. A step that needs a judgement prints one JSON object `{"request":{...},"carry":{...}}`; the mod asks the model for the request exactly as printed and gives the reply and the carry, unread, to the next step. The carry is how a step hands what it found to the next one; no file holds it. The tiers are a cheap one (`haiku`, effort `low`) for the moment judgement and a mid one (`sonnet`) for the content judgements, the phrases among them; they are the current roster for a tier.

It registers functions for these events: `prompt.submit` (for each prompt, before it enters the session), `turn.start` and `turn.complete` (the ends of a turn), `tool.call`, `classic.PostToolUseFailure` and `classic.PostToolBatch` (the tool events of a turn).

The workflows it owns: [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow), [793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow), [793d2c Conversation sweep flow](./conversation-sweep-workflow.md#793d2c-conversation-sweep-flow) and [855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow).

Host facts it relies on:

- `prompt.submit` takes `context` entries, "What the model reads beside the [prompt](../../GLOSSARY.md#prompt-automatic-memory) and the user never sees", attached on the way down
- `tool.call`'s input is the tool-call envelope and `agentId`, present in a [subagent](../../GLOSSARY.md#subagent)'s loop only; it has no [session](../../GLOSSARY.md#session) ID
- A `tool.call` hook answers with the tool's result and `context`, "What the model reads after the tool's result and the user never sees"
- The session's ID is `$.session.id()`
- `classic.PostToolUseFailure` and `classic.PostToolBatch` answers carry `additionalContext`, "text handed to the model with the event"; their inputs carry `session_id` and `agent_id` (a subagent's only); a failure's input carries `is_interrupt`, and each call of a batch its `tool_use_id`
- `PostToolBatch` is "Fired once after every tool call in a batch has resolved, before the next model request"
- `$.model.complete` takes `model` (an alias), `prompt`, `system`, `maxTokens`, `effort`, `timeoutMs` ("default none") and, as an option, a `signal` that aborts the call; the result is `isAnswered` with `text`, or `reason` `api-error`, `empty-reply` or `aborted`
- `$.process.run(argv, { stdin, timeoutMs })` rejects when the command cannot start or is still running at the timeout: 30 seconds when absent, ten minutes at most; the result holds `exitCode`, `stdout`, `stderr`
- A hook's budget is 10 seconds of its own time; "the clock stops while a `next(e)` call or any `$` call of the hook's is in flight"
- A `classic.<Event>` function hook fires wherever the engine runs the classic hook, whether or not a settings hook is configured
- `$.clock.after(ms, fn)` runs `fn` later
- `$.ui.log(text)` appends one line the user sees and the model does not read

A mod in a linked skills folder loads in a headless `claude -p` session, and the context it adds is recorded in the session's [transcript](../../GLOSSARY.md#transcript) as a `hook_additional_context` row before the agent's first reply; `turn.complete` carries the turn's final text and ID and fires when the transcript already holds the turn's last row; `$.model.complete` answers `haiku` also from work left running after the hook that started it returned. `$.ui.log`, called in a hook or from a timer, appends one line to the conversation as the user sees it, "drawn like a system notice (dim; not sent to the model)" in the host's type file, and to no transcript of the session; a host that streams JSON receives it as a `system` event of the subtype `ui_log` with the mod's name, and plain text output of `claude -p` shows none. The transcript records a block delivered within a turn once, as an attachment of type `hook_additional_context` beginning `[engram memory]`, and a context put at `tool.call` reaches the model only at its next request, after the tool's result.

Claude Code's mod runtime (an external boundary, relied on and not tested here) loads a mod from `~/.claude/skills/<name>`, runs the functions above, runs the commands the mod asks for, makes the model calls it asks for with the user's own login, and carries the context it returns to the model: the `context` of a tool call's result and the `additionalContext` of the failure and batch events.
