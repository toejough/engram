# Apply: generic-actionable learn + as-requirements recall

> **Paste this whole file to a fresh agent in the engram repo
> (`/Users/joe/repos/personal/engram`, branch `feat/eval-harness-m1`).**
> It is a self-contained task; do not assume prior conversation context.

## Goal

Apply two evidence-backed changes to the engram skills:

1. **`learn` skill — distill recurring conventions into *generic-actionable*
   notes.** When a convention recurs across episodes/sessions, write (or
   update) a **fact** that states the **general principle WITH its
   concrete actionable specifics attached** — not a bare abstraction, and
   not a rigid full code recipe.
2. **`recall` skill — surface memory *as explicit requirements*.** When
   recall reports surfaced memory to the consuming agent, frame each
   recalled convention/principle as a **requirement to implement**, not
   merely as background context.

## Why (the evidence — don't re-derive, just trust + verify)

A 5×5 matrix (learn abstraction-form × recall strategy), each cell a fresh
Go-CLI build scored on convention correctness (/17):

- Correctness is **non-monotonic** and **peaks in the middle** at
  **generic-actionable** (general principle + concrete actionable detail):
  12.4 avg, up to 15–16/17. Bare **generic** principles are the trough
  (5.4 ≈ no-memory); rigid full **recipes** fall off (8.8 — they prescribe
  form but crowd out feature breadth).
- Enriching generic with actionable detail roughly **triples** the score
  (5 → 15). The lost value was the *absence of specifics*, not abstraction.
- **`as-requirements` recall** (reframe surfaced memory as mandatory
  requirements) is the best extractor (col-avg 12.6); `expand` /
  `checklist-synth` paraphrase and *shed* items; `verbatim` is mediocre;
  `bounded`(top-1) is cheap but starves the agent.
- **Key negative result: recall cannot manufacture content the note never
  stored** — no recall strategy lifted bare-generic off 5–6/17.
  **Correctness is set at learn-time; recall amplifies, it does not
  create.**
- Cost: the winning pairing (generic-actionable + as-reqs) reached **15/17
  in one autonomous build (~27 turns, ~$0.82)**, vs ~$4.34 / ~61 turns of
  human-review-loop to reach the same bar with no memory. Memory's value =
  autonomous one-shot conventions instead of re-supplying them every time.

Durable records (read for grounding):
`docs/superpowers/specs/2026-05-30-cold-warm-todo-test.md` (Results +
Generalization + accumulation sections) and agent-memory vault notes
`Permanent/252`–`Permanent/255` (run `engram query "distilling agent
memory abstraction level and recall"` to surface them).

## MANDATORY process

Both edits are `SKILL.md` changes. Per the engram `CLAUDE.md`:
**ALWAYS use the `superpowers:writing-skills` skill when editing any
SKILL.md. No exceptions.** That means TDD: RED baseline (watch a fresh
agent fail to do the new behavior with the *current* skill) → GREEN
(minimal edit) → verify the behavioral change. The RED evidence already
exists (the matrix); still do a clean RED+GREEN verification per the
skill.

## Change 1 — `skills/learn/SKILL.md` (generic-actionable distillation)

Context: the learn skill already has (added earlier this session) a
"**Capture stated requirements and decisions — completely, consolidated**"
subsection under `## What to write`, an arc-based `### 6a` episode model,
and an "Atomicity is one coherent topic, not one micro-fact" quality bar.

**Add a cross-episode distillation directive.** Where the skill discusses
facts (and/or in a new short subsection), instruct: when a convention or
decision **recurs across multiple episodes/sessions**, distill it into a
**fact** written in **generic-actionable** form:

- State the **general principle** (so it retrieves for *any* matching task
  and isn't tied to one app/domain), AND
- Attach the **concrete actionable specifics** that make it implementable
  — exact interface shapes, file/layout names, exact patterns, the
  enumerated set — so a future agent can *act* on it without re-deriving.
- **Avoid both failure modes:** a bare abstraction (no specifics → the
  agent re-derives them → costly, low correctness) and a rigid full recipe
  (over-prescribes form → crowds out breadth). Aim for the middle:
  *principle + the directives needed to execute it.*
- When such a fact already exists, **update it** with new actionable
  detail rather than adding a redundant near-duplicate (redundant/weaker
  copies dilute and cost more — see `Permanent/253`).

Add a matching **Common mistakes** row (e.g. "Storing a recurring
convention as a bare abstraction" → "Attach the concrete actionable
specifics; principle + directives, not just the principle").

## Change 2 — `skills/recall/SKILL.md` (as-requirements framing)

Context: recall's user-/agent-facing synthesis is the `### 4b` section
("walk the plan, say confirmed/adjusted/contradicted/silent").

**Add framing instruction:** when recall surfaces load-bearing
conventions/principles relevant to what the agent is about to build, it
must present them **as explicit requirements to implement** — e.g. "Apply
these as requirements: …" — not merely as "memories that confirmed the
plan." The consuming agent should treat each surfaced convention as a
must-do, even when the memory states it generally. Keep it concise; this
is a framing change to how 4b reports actionable memory, not a new
section. Add a Red-flags / Common-mistakes note that surfacing memory as
passive background (rather than as requirements) is the weaker behavior
the matrix penalized.

## Verify (GREEN)

- **learn:** dispatch a fresh agent to follow the edited learn skill and
  distill a couple of build episodes; confirm it writes a **generic
  principle WITH concrete actionable specifics** (not a bare abstraction,
  not a full recipe).
- **recall:** dispatch a fresh agent to follow the edited recall skill on
  a vault of such facts; confirm its 4b output **frames the surfaced
  conventions as requirements to implement**.
- (Optional, costs ~$1–2) re-run one slice of the transfer test: build a
  Go CLI with a generic-actionable vault + as-reqs recall and confirm
  ~14–15/17 conventions, compiling. Reuse `/tmp/epitest/*` if it still
  exists, or the `dev/eval` harness.

## Land it

- Run **`targ check-full`** if any Go changed (these are skill/markdown
  edits, so likely just confirm nothing else broke).
- Commit with trailer `AI-Used: [claude]` (HEREDOC), referencing #642.
- **Propagate to installed skills:** `engram update` (copies
  `skills/{learn,recall}/` → `~/.claude/skills/` and OpenCode). Verify the
  installed copies contain the new directives.
- Run the closing `/learn` to capture the work.

## Conventions (engram `CLAUDE.md`)

- `targ` for all build/test/check (never raw `go test`/`go build`).
- Commit trailer `AI-Used: [claude]` (NOT Co-Authored-By).
- Editing SKILL.md → `superpowers:writing-skills` (no exceptions).
- `dev/eval` is the harness; `engram transcript --segments` lists arc
  boundaries; episodes embed by `situation`.
