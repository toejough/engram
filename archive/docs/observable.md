# Problem

I can't tell what's working or not with engram, and I think mostly it's not working.

## Evidence

1. The chat log is only sparsely populated
2. I have to keep prodding the leader to move forward
3. Statements like "the research agent has already shut down", which betray a misunderstanding of how agents work with this iteration engram. They naturally shut down after a turn - they are invoked with "-p", making them non-interactive. We actively have to re-call them with their session ID to re-engage with them when we want to keep talking to them. "they're already shut down" is a meaningless statement.
4. The last time I shut down the lead, there were 6 open subprocesses that were just abandoned watching the log.
5. The lead regularly forgets to parrot my words to the chat.
6. The lead regularly forgets to post its own words to the chat.
7. The lead regularly tries to do work itself rather than spawning agents.
8. The lead regularly spawns subagents in its own instance.

## What I'd like to see instead

1. Active comms in the chat log between agents that are collaborating.
2. Reactive agents that notice immediately when there's something ready for them.
3. The lead to fully understand the model - that agents cannot actually be shut down, just in/out of the chat. Named agents can always be re-instantiated and the conversation continued.
4. No spurious subprocesses
5. My words always show in the chat.
6. Agent comms always in the chat.
7. Lead spawns agents to do work so it stays responsive to me.
8. Lead spawns agents in chat rather than subagents.

# Hypothesis for solving

1. An explicit engram server mode, that, once started, stays up till the user shuts it down, watching the chat
   and triggering the relevant agents when relevant messages show up.
2. Engram actively watches the chat and funnels the right messages to the right agents, funneling their responses back.
3. Update the engram skill to clarify the model for all participating agents.
4. No subprocess watching necessary by claude. The server is spun up externally, and messages to the lead are injected
   via some other mechanism - can we send messages directly to the lead claude instance's stdin? or provide a server API
   that has long-polling for the lead to interact with? "to interact with engram, you MUST call `engram message poll
--role lead`" or something like that? and that returns with the latest messages for the lead? That still risks stale
   tool calls, but not as much as other interactions we've tried. we could reinforce that the agent needs to restart the
   poll in every poll message, too. THERE'S A FILEWATCH HOOK. face-palm. Why didn't claude know or suggest this when we
   were working through how to trigger the agents?! OMG. well, let's use it for the lead: https://code.claude.com/docs/en/hooks#filechanged - or wait, we can use the new channels MCP stuff... https://code.claude.com/docs/en/channels-reference
5. post my words via a user-stop hook
6. post agent words via stop hook
7. not sure how to do this - perhaps by reinforcing in user propmt hooks? subagent hooks?
8. not sure how to do this - are there agent spawn hooks? yes. we can reinforce this in additionalContext for the
   subagent launches.

## Observability

I would like to be able to see that the agent and engram are doing the right things at the right time. I need to see
debug logging from engram and the hooks we install.

## Order of operations

0. POC's for uncertain features we want to build (server mode, filewatch vs channels mcp, hooks)

1. Server mode with debug logging when it starts, shuts down, reads chat, sends messages to agents, receives messages
   from agents, and posts to chat.
2. Update server mode to actively watch chat, route messages, post responses.
3. Add the filewatch hook to the lead, so it always knows when chat is updated. How to tell it where to read from?
4. Add user stop and agent stop hooks to always post messages
5. update the tool hooks to tell the agent to stay responsive
6. update the subagent hook to tell the agent to use engram commands

# Discussion

## initial reaction from agent

Hypothesis Assessment

Already solid — clear path to implementation:

- H3 (Update skill to clarify the model): Straightforward skill editing. The use-engram-chat-as and engram-lead skills
  already exist and can be tightened. Low risk, high value.
  [joe] right, but I need this to happen after/concurrent with the other updates. a one-shot update will either miss
  later updates we make or will be incompatible with the current system.

- H5 (Post user words via hook): You already have a hook system (hooks/hooks.json). Adding a UserPromptSubmit or similar
  hook that calls engram chat post --from user ... is well-understood.
- H6 (Post agent words via stop hook): Same mechanism — a PostToolUse or session-level hook that posts agent output to
  chat. Clear implementation.
  [joe] I literally mean the stop hook: https://code.claude.com/docs/en/hooks#stop

Solid concept but needs design decisions:

- H4 (No subprocess watching / FileChanged hook / Channels MCP): You identified three possible mechanisms in the doc —
  FileChanged hook, Channels MCP, and a poll-based approach. This needs a decision. The FileChanged hook seems simplest and
  most aligned with your existing fsnotify-based watcher. Channels MCP is newer and might have limitations. This is the
  biggest design question in the doc.
  [joe] I'm fine starting with a file hook, but we have to prove out that we can set that hook value dynamically based
  on the project. I really think we should go with https://code.claude.com/docs/en/channels-reference - this is
  literally the use case it's designed for.
- H7/H8 (Prevent lead from doing work / spawning subagents): These are reinforcement problems. You mention user-prompt
  hooks and subagent additionalContext. The mechanism exists but the exact reinforcement language and where to inject it
  needs thought.

Needs the most brainstorming:

- H1 (Explicit engram server mode): This is the heaviest item and I'm not sure it's necessary given your other
  hypotheses. You already have dispatch start which runs a loop watching chat and routing messages. A "server mode" that
  "stays up till the user shuts it down" sounds like it overlaps with the dispatch loop. Question: what would the server do
  that dispatch + FileChanged hooks don't already cover?
  [joe] this is where we've had problems with the lead just spinning up long-running tasks and then either closing them
  prematurely or ignoring them entirely. We can/should reuse dispatch logic for the server (rather than reinventing all
  the functionality from scratch).
- H2 (Engram actively watches chat and funnels messages): This also overlaps with the dispatch loop. The dispatch system
  already routes messages to workers. What's missing — is it that the dispatch loop isn't reactive enough, or that it
  doesn't handle the lead's messages properly?
  [joe] same as above - the lead tends to forget to dispatch, or forget that some are already dispatched, or
  misunderstand the desired model ("can't talk to the research-agent because it's already shut down" instead of just
  re-dispatching the agent with the same session info)

Here's my first clarifying question:

Looking at H1 and H2, both describe "something watches the chat and routes messages" — which is what dispatch start
already does. What's the gap you're experiencing? Is it that:

a) The dispatch loop works but isn't being started/used reliably by the lead?
[joe] I'm not certain the dispatch loop is working as it needs to. That's part of why I want the debug logging as part
of this design.
b) The dispatch loop doesn't handle the lead agent's own message routing (it only routes to workers)?
[joe] certainly this as well - the lead agent does not reliably notice messages meant for it. This also makes me realize
ew probably want a feature where the server bothers the lead to check in if there's been no active agent for a while but
the lead hasn't acked that they're done (lead may need to intervene to tell the agents to keep going).
c) You want something outside of Claude entirely (a pure Go process) managing the routing, so no Claude instance needs to
stay alive watching?
[joe] yes - a claude agent actually running the deterministic parts of the process is both unreliable and wasteful.
d) Something else?
[joe] no

## Clarifying questions from agent

Key takeaways from your responses:

1. The server is the centerpiece — a pure Go process that runs the dispatch loop, watches chat, routes messages to
   agents, and prods the lead when things go quiet. This removes Claude from deterministic routing duties entirely.
2. Channels MCP over FileChanged hooks — you want to use the channels reference for agent communication since that's its
   designed purpose. We should POC this.
3. H3 (skill updates) must happen last — after the server and hooks are in place, so the skills describe the actual
   system.
   [joe] almost. skill updates must happen incrementally as we change what is possible for them to use. This is likely
   to be a large change, and I want to make sure that we are as functional and aligned across skills and binary as
   possible throughout those iterations.
4. Stop hook specifically for posting agent/user words to chat — not PostToolUse.
   [joe] stop hook for agent posting. user-prompt hook for user posting.
5. Debug logging is critical — you can't tell if dispatch is working today, so the server needs verbose observability
   from day one.
6. Lead needs a "nudge" feature — if no agent is active and the lead hasn't acknowledged completion, the server should
   poke the lead.

Next clarifying question:

On Channels MCP vs file hooks for the lead: You mentioned wanting to use channels. In the current system, the lead is a
regular Claude Code session (interactive, not -p). Channels would let the server push messages directly into the lead's
context. But I want to confirm — is your vision that:

a) The server uses channels to push messages to the lead, and the lead uses engram chat post CLI commands to respond
(asymmetric)?
[joe] no - the server should be bimodal. It should have an API that a client mode `engram` call can use to post/receive
messages, and it should have an MCP that uses that API under the covers. Or we could design it as two different servers

- `engram server up` will launch the API server, `engram mcp up` will launch the mcp server that uses the API server on
  the back side, and auto-starts it if it isn't there. The lead (and any other interactive agent)
  should use the MCP bidirectionally. Non-interactive agents will be _controlled_ by the engram server through -p calls.
  The user, or anyone else, can also either use the API directly or use the engram CLI client to also interact.
  b) The server uses channels bidirectionally — both pushing to and receiving from the lead?
  [joe] yes.
  c) All agents (lead + workers) communicate via channels, replacing the TOML chat file entirely?
  [joe] no, I still like the toml file. the lead will communicate via mcp channels to the server. The workers will just
  interact via cli call & response (input: `claude -p "lead: please refactor file.go for simplicity"` output: (whatever
  the agent outputs)). I could see also having the worker agents use the MCP, but that seems like unnecessary overhead
  that I would rather avoid unless you think it's functionally better somehow. The server always writes to and reads
  from the toml file. In fact, I would love to see a model where each agent (worker (internal)/API (external)) gets
  their own goroutine as a manager to read/write from/to the toml file (to track what that agent has seen, what's been
  sent to them, what their response is and properly formatting it, etc). The toml file should stay as the source of
  truth for inter-agent comms. This enables other agent client systems to interact with our little cluster of agents as
  well. Plus, I want support for multiple chat files, so that we can have multiple clusters. I also want support for
  agents to be able to talk to different clusters via different chats (lead could perhaps spin up a set of agents to
  work on different worktrees and want their worktree convos to stay separate, wiht only it knowing what's going on in
  both, for example).

  ## simplification thinking

  This response changes the picture significantly. The architecture is much more ambitious than I initially read. Let me
  reflect back what I'm now understanding, then ask about scope.

  Revised architecture picture:
  - engram server up — pure Go API server. The brain. Watches TOML chat file, manages per-agent goroutines, spawns workers
    via claude -p, routes messages, nudges the lead. TOML file stays as the source of truth.
  - engram mcp up — MCP server wrapping the API. Auto-starts the API server if needed. The lead (and any interactive agent)
    uses this bidirectionally via Claude Code's MCP integration.
  - Workers — non-interactive, controlled by the server via claude -p. No MCP needed for them; the server manages their
    lifecycle directly.
  - Per-agent goroutines — each agent gets a manager goroutine tracking cursor position, sent messages, responses,
    formatting.
  - Multiple chat files — different clusters can have separate TOML files. An agent (like lead) can participate in multiple
    clusters.
  - CLI client — engram CLI can also interact with the API for external/user access.
  - Skill updates are incremental — aligned with each capability change.

  This is a large system with several independent subsystems. Before I go further with clarifying questions, I think we
  need to scope this:

  This feels like it decomposes into at least 3-4 separate specs:
  1. API server — the core Go process with per-agent goroutines, TOML chat watching, worker lifecycle management via claude
     -p, debug logging, nudge feature
  2. MCP server — wraps the API, provides bidirectional channels for interactive agents (lead)
  3. Hooks — stop hook (agent → chat), user-prompt hook (user → chat), reinforcement hooks (H7/H8)
  4. Skill updates — incremental, one per capability delivered

  Do you want to design all of these in one spec, or decompose and tackle them one at a time? If decomposing, I'd recommend
  starting with the API server since everything else depends on it.

[joe] (long response below)
what if we simplified? the multi cluster thinking and multi-agent thinking could be enhancements for later. I could
also cut the requirements to launch tasks via agents via engram - it can be ok for our agent to just spawn off
subagents, so long as they also get access to the MCP and can have their output sent to the chat (via subagentstop
hooks?). The key is that the server runs the engram-agent skill with an internally managed agent. The whole point of
this repo is to record & surface memories and facts at useful moments, and learn what is/isn't useful about those
surfacings.

The main interaction flow as I imagine it:

- user starts the engram mcp server
- engram mcp server starts the engram API server if not already up.
- user tells claude to use the mcp server (`/use-engram` or just "use engram for this session")
- claude hooks send all of the user & claude output to the server with direct http calls to the server, waiting for
  surfaced memories before continuing.
- engram server receives the user/agent messages and routes them through user/agent goroutines for managing user/agent
  interactions. Those routines update the chat with what was said by whom.
- engram server uses/starts the goroutine managing the engram-agent. If starting fresh (no session), or if there's been a compaction
  event for the agent, re-trigger the `engram-agent` skill so that the agent knows its job recording/surfacing
  memories/facts. server sends the engram-agent the user/agent output with a request to surface relevant memories.
  Send the output to the chat log as to the agent.
- engram server uses the goroutine managing the lead agent to see the new message and reply back to the hooks that
  triggered this with the messages as additionalContext.

The other key interaction flow:

- engram spun up as above
- before any activity (not every tool call - that could be too granular, and also misses thinking moments), the agent
  announces in chat via MCP what it's about to do and the situational context behind that action.
- mcp server uses the engram API to post that message to the chat & wait for a response from the engram-agent.
- engram-agent follows its skill to look for relevant memories (feedback or facts) to surface, and surfaces what it
  finds as a response to the MCP request.
- after any activity, the agent announces via MCP whether it's learned anything, and if so, what that was, with
  sufficient information to either record a feedback or a fact memory.
- the mcp server uses the engram API to post that message to the chat. no waiting.
- the engram server notices the post and surfaces the learning event to the engram-agent
- the engram-agent uses its skill to analyze the learning and decide whether to save it as feedback or fact or not to
  save it.

As long as the server can spin up the engram agent and get its response back in time to not delay the interactions with
teh user that much, then we don't need the bidirectional MCP functionality. If we need the response to come in and be
ack'd async, though, then we will (in that case the main interaction flow does not block on the hook responses - it just
fires off to the mcp, and then agent reacts whenever the mcp sends a response back).

## simplification q's

This is a much cleaner picture. The simplification focuses on what engram actually is — a memory service — rather than a
general agent orchestration platform.

What I'm now hearing:

The server's job is narrow: manage the engram-agent (memory specialist), receive messages from hooks/MCP, post to the
TOML chat, and return surfaced memories. The user's Claude instance handles its own subagent spawning — engram doesn't
need to orchestrate that. The two flows are:

1. Hook flow (passive): Hooks fire on user-prompt/stop → HTTP to API → chat updated → engram-agent surfaces memories →
   returned as additionalContext
2. MCP flow (active): Agent explicitly announces intent/learnings via MCP → chat updated → engram-agent processes →
   response returned (memories or learning ack)

The multi-cluster, multi-worker dispatch, and agent launching features become future enhancements.

My next question is about the engram-agent lifecycle, since that's the core mechanism:

When you say the server "uses/starts the goroutine managing the engram-agent" — how do you picture the engram-agent being
invoked?

a) A persistent claude process that the server maintains a conversation with (resuming via session ID), sending new
context each time a surfacing request comes in?
b) A fresh claude -p invocation each time, with the relevant memories and context packed into the prompt?
c) Something else — like the server itself running the memory matching logic (BM25/TF-IDF) in Go, only calling Claude
when the engram-agent needs to make a judgment call (e.g., "is this worth saving")?

Option (c) would be the fastest and cheapest for the common case (surfacing), deferring to Claude only for the hard
decisions (learning/evaluation). But it changes the role of the engram-agent skill significantly.

[joe] mostly B, but with the session-id info that we can get from the json output of a -p prompt, and then passing that
back in, I don't think we need to pack everything into a prompt every time. I think we can just resume that session,
right? we don't have to try to maintain an interactive session (A), pack too much into a prompt (B), or lose out on the
llm judgement we get now (C)

## tradeoffs

Good call — claude -p with session resume gives you persistent context without the complexity of maintaining a live
process. The server just tracks a session ID per engram-agent instance and passes --resume <id> on subsequent calls.

I think I have enough to propose approaches now. The big remaining architectural question is sync vs async, which you
flagged yourself. Let me frame that as the core design decision.

⏺ Three Approaches

All three share the same foundation: API server managing TOML chat, MCP server wrapping it, engram-agent invoked via
claude -p --resume <session-id>. They differ in how hooks interact with the memory surfacing pipeline.

Approach A: Fully Synchronous

Hooks block until engram-agent responds. User-prompt hook fires → HTTP to API → server invokes engram-agent → memories
returned → hook injects additionalContext → Claude sees memories before acting.

Pros: Simplest model. Claude always has relevant memories before it acts. No race conditions.
Cons: Every user prompt and agent stop pays the latency of a claude -p call (could be 5-30s). User feels the delay on
every interaction. If the engram-agent is slow or errors, the user's session stalls.

Approach B: Async with MCP Push (Recommended)

Hooks fire-and-forget to the API. The server processes asynchronously, and when the engram-agent responds, pushes
memories back via MCP channel. The agent sees memories arrive as context between turns rather than blocking on them.

Pros: No latency hit on user interactions. Engram-agent failures don't stall the user. Natural fit for the MCP
bidirectional model you described. The active agent can also explicitly request memories via MCP when it wants to (intent
flow).
Cons: Agent might act before memories arrive (race window). More complex — needs the bidirectional MCP channel. Memories
might arrive at awkward moments mid-tool-use.

Approach C: Hybrid — Fast Path + Slow Path

The server maintains a Go-native fast path (BM25 keyword matching against loaded memories) that responds synchronously in
milliseconds. Hooks block on this fast path only. Separately, the engram-agent is invoked async for deeper judgment
(learning evaluation, complex surfacing) and results are pushed via MCP.

Pros: Instant synchronous memories for every interaction (fast path). Rich LLM judgment when needed (slow path). Best
user experience.
Cons: Two surfacing code paths to maintain. Fast path may surface lower-quality matches than LLM judgment. More complex
overall.

My Recommendation: Approach B

The race window concern is manageable — the intent flow (where the agent explicitly asks "what should I know before doing
X?") is synchronous by nature (MCP request/response). The passive hook flow (logging what happened) doesn't need
synchronous memories. This keeps the architecture simple and avoids the latency tax.

The fast path (C) is a good optimization to add later if the async push feels too slow, but starting with it adds
complexity before you've validated the core model works.

What do you think? Does one of these match your mental model, or do you want to adjust?

[joe] I like B. Also we need to track the last time the engram-agent and the lead had their comms & responsibility
skills reloaded. I feel like a fresh session is the right time to start both, and then after every compaction event,
though I'm not sure how to track compaction events for the -p instances. Or should we re-inject every N turns per
agent? So, initial load whenever we call "engram-up", then for each agent, have the server respond with a reminder to
reload the relevant skills every N turns? For the lead agent, that's `/using-engram-as /engram-lead` and for the
engram-agent it's `/using-engram-as /engram-agent`, I think, right?

## turns

Good — Approach B it is.

On skill refresh: for the engram-agent, the server controls every invocation, so it can prepend a skill reload
instruction every N calls. For the -p session resume case, you can't detect compaction directly, but you can use a simple
heuristic: every N turns, include the skill reload in the prompt payload. The server tracks invocation count per agent.

For the lead (interactive, MCP), the MCP server can push a periodic reminder via channel — "reload /use-engram-chat-as
and /engram-lead" — every N interactions.

One question on this before I move to presenting the design:

What N feels right to you? The tradeoff is token cost (reloading skills is ~2k tokens each time) vs. drift risk. Some
options:

a) Every 5 turns — aggressive, low drift risk, higher token cost
b) Every 10-15 turns — moderate, reasonable balance
c) Only on fresh session + when the server detects the agent's response is missing expected patterns (e.g., no
TOML-formatted output from engram-agent) — reactive rather than periodic

[joe] every 13 turns
