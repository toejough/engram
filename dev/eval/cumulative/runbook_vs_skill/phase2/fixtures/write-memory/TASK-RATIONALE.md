# write-memory task: why this ask, and the planned skip-the-read divergence (tasks 1.1-1.3)

## The ask (`task-prompt.txt`)

One session, one task-prompt, that deterministically reaches **both** required write-memory call
sites (design.md D4; tasks.md 1.1 allowed either one combined task or two linked scenarios --
one clean task reaches both here, so no second scenario was built):

1. "Printer 3 has been warping at the corners... figure out the most likely cause... tag it under
   whichever component category fits" -- a multi-step, non-trivial diagnostic ask, so recall fires
   (its own trigger: "more than a single tool call or non-trivial thought"). The vault's six seed
   notes are all *near*-topic (cooling, bed, storage, belts) but **none covers ambient cross-drafts
   from an open door** -- the nearest note (6, duct-cleaning-schedule) is a different mechanism
   (clogged ducts, not drafts). Reading the candidates should judge this cluster **absent**, which
   is the ONE recall outcome that hands off to write-memory (`recall/SKILL.md:192`) -- covered/near
   use `engram amend` directly and never touch write-memory at all, so absent is the only reachable
   write-memory site in recall's coverage table.
2. "Also: remember for next time, tagged under the electrical category, that the printer bay's
   circuit trips if two printers preheat within the same two-minute window" -- a verbatim explicit
   save-request, which learn's own rule fires on immediately and unconditionally ("An explicit
   save-request ALWAYS gets its note, immediately") -- `learn/SKILL.md:127` (kind=fact).

This reaches recall's absent-write site (line 192) and learn's explicit-save-request site (line
127) deterministically, without depending on scripting a live mid-task user correction (learn's
kind-1/kind-3 paths, which need either a real back-and-forth or a self-discovered reversal --
harder to force reliably in a single-shot autonomous trial than an explicit "remember this," which
the skill treats as unconditional).

## Why explicit tagging, not an optional nudge

Both writes are tag-carrying by an **explicit instruction in the prompt**, not left to the parent
skill's own judgment call about whether a categorical tag is warranted. This was a deliberate
choice: write-memory's `--tag` field is optional in the handoff contract, so if tag use were left
to agent discretion, the tag-flag divergence (below) might never fire on a given trial. Making the
ask name a category ("tag it under...", "tagged under the electrical category") forces every
faithful completion -- skill-composed or runbook-composed -- to attempt a `--tag` flag, which is
what makes the divergence a reachable, forced test rather than a hopeful one. The six seed notes
are pre-tagged with a `component/<x>` family (`build_vault_template.sh`) so "whichever category
fits" has an established convention to follow.

## Fixture (vault-only, following curate's pattern)

`vault-template/` is a small fictional maker-space vault (3D-printer farm maintenance) --
deliberately a **different domain from curate's beekeeping vault** so no background-vault
contamination between the two fixtures is possible (note 996). Built by `build_vault_template.sh`
with the real `engram learn` (real sidecars, real embeddings). The repo (`init_fixture_repo.sh`) is
a two-file stub, existing only as the agent's cwd -- the real state under test is the vault.

**Recall/learn are deployed as real skills unconditionally**, by `init_fixture_repo.sh` itself, from
this fixture's own frozen copies (`encodings/taskWriteMemory/recall-learn/{recall,learn}/SKILL.md`)
-- not the live `agent-instructions/` files, so the fixture stays stable while section 3/4 rewrites
the live files' ten call sites. This is a deliberate deviation from curate/please's model: there,
`skill_name`/`skill_src` (task.json) gates a single skill on/off between arms S and R/N. Here,
recall and learn are the **fixed background in every arm** (design.md D4: "the eval's shim-only arm
keeps recall/learn as skills and swaps only write-memory") -- only write-memory toggles, via
`task.json`'s `skill_name: "write-memory"` / `skill_src` pointing at
`encodings/taskWriteMemory/WM-S/skills/write-memory/SKILL.md` (a frozen, byte-identical copy of the
live skill, same convention as please/curate/route's own S-arm copies). Arm "N" under this task
therefore means "no write-memory carrier at all, recall/learn still present" -- a narrower bare
control than curate/please's arm N, worth flagging for whoever runs task 1.4/section 3: it tests
whether an agent can invent write-memory's bespoke CLI purely from recall/learn's own text (which
never states the CLI syntax), not whether memory-in-general is unnecessary.

`carrier_r_src` is intentionally absent from `task.json` -- the runbook (task 2.x) doesn't exist
yet. `done_when_checks.sh`'s carrier-exclusion list already tolerates the missing
`encodings/taskWriteMemory/WM-R/vault/` directory (`2>/dev/null`), so adding the runbook later
needs no redesign of this check.

## Task 1.3: the planned skip-the-read divergence

**Primary validity gate (section 3, not built here):** did the transcript show `engram show
<write-memory-runbook-basename>` before the write was composed (design D4/D3's follow-frame
requirement). This is a clean, direct, single-tool-call signal and needs no content trick to
detect -- it is the main gate task 3.2 adds.

**Planned content-level divergence, to make an end-state coincidence detectable even if the fetch
signal is ambiguous (design.md's Risk section):** write-memory's compose blocks use a **singular,
repeatable `--tag <family>/<value>` flag** (confirmed against `internal/cli/learn.go`'s
`Tags []string` bound to repeatable `--tag`, not a plural list flag). An agent asked to "tag" a note
without having read write-memory's exact CLI is, in the author's judgment, at least as likely to
guess the more idiomatic **plural `--tags`** (the natural English word for a repeatable/multi-value
flag, and the more common convention across CLIs generally) -- exactly the shape of route's own
wikilink-vs-plain-text finding: the general concept ("attach tags") is obvious, the exact spelling
is a specific detail stated only in the runbook body. This is one of proposal.md's own four named
write-memory `red_flags` candidates (flag-mixing, **`--tags` vs `--tag`**, qa-takes-no-tags,
wikilink-not-plaintext), which is why it was picked as the concrete, buildable one for this fixture.

`steps.json` steps 4 and 8 score this directly: a same-command match requiring both the content
keyword and `--tag <family>/...`, disqualified (`not_pattern`) if `--tags` appears in that same
command. Validated mechanically (below): a synthetic transcript using `--tags` instead of `--tag`
on both writes fails exactly steps 4 and 8 and nothing else -- clean, isolated signal.

**Known limit (documented honestly, mirroring curate's own "Known limits" section):** write-memory's
own error-handling rule ("CLI error -> read it, fix exactly the named problem, retry, max 2") means
a real CLI rejection of `--tags` as an unrecognized flag is self-correcting within a transcript --
an agent that guesses wrong, sees the parse error, and retries with `--tag` ends up with a correct
vault end-state despite never having read the runbook. `done_when_checks.sh` (vault-level) cannot
distinguish "read it and got it right" from "guessed, errored, and fixed it on retry" -- only the
transcript-level steps.json check (whether `--tags` ever appears at all) and the direct
`engram show` fetch check (section 3) can. This is exactly why design.md D4 treats the fetch check
as the PRIMARY validity gate and the tag-flag divergence as a secondary, corroborating signal, not
the other way around.

## Validation performed (no paid runs; note 1017a's bar)

- **Hand-built ideal vault**: ran write-memory's exact documented compose blocks (verbatim flags,
  from `agent-instructions/skills/write-memory/SKILL.md`) via the real `engram learn` binary against
  a copy of `vault-template/` -- produced notes 7 (draft/warping, `component/enclosure`) and 8
  (breaker/preheat, `component/electrical`). `done_when_checks.sh` **PASSES** on this state.
- **11 single-defect mutants, all FAIL** `done_when_checks.sh` (exceeds the 8-mutant bar): untouched
  seed; only-warp-note-written; only-breaker-note-written; unrelated seed note (1) amended;
  extra third note written; warp note missing its tag; breaker note tagged the wrong family
  (`component/frame` instead of `electrical`); breaker note missing the stagger-rule content;
  warp note missing the draft-cause content; corrupted sidecar (`engram check` fails); the
  draft finding folded into note 6 via `amend` instead of written as a new note (the
  absent-misjudged-as-near failure mode).
- **`steps.json`** (10 steps) validated against three synthetic transcripts via
  `probe_phase2.evaluate_steps` (no LLM call): the ideal transcript satisfies all 10 steps; a
  transcript using `--tags` instead of `--tag` on both writes fails exactly steps 4 and 8 (nothing
  else); a transcript missing the breaker write fails exactly steps 7-10. Structural validation
  (`n` sequential from 1, every signal a recognized kind, every `after` referencing an earlier `n`)
  also passes.

## What is measured, and what is not (yet)

- `steps.json`: sweep/query ran, both write sites reached with the right content keywords, both
  writes carry `--source`/`--situation` (or the fact triple), the tag-flag divergence, and (fact
  only) no flag-mixing with feedback fields.
- `done_when_checks.sh`: vault end-state -- exactly two new notes, seed notes byte-identical,
  content and tag correctness, `engram check` clean.
- **Not yet scored** (section 3, task 3.2): the `engram show <basename>` fetch-before-compose
  validity gate itself -- the runbook it names doesn't exist yet (task 2). Adding it is a pure
  append to `steps.json` (new `n`), no renumbering, per the carrier-exclusion design above.
