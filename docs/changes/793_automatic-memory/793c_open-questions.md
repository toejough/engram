# Automatic Memory - Open Questions

Each question needs Joe's input before `793a` is final. He was unavailable, so each has a default, an author decision in `793b` (the owner may override), which stands until he answers; his answer is then recorded here with its date and the entry stays in place.

## 793c1 How long may a prompt be held?

Default: 8 seconds (`793b17`). Alternatives: a shorter bound; a longer one; the owner's earlier rule, no longer than the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup on the same [vault](../../GLOSSARY.md#vault) and input (10.6 to 11.0 seconds on his vault, `old 793c7`). Open.

## 793c2 How soon after a turn must automatic memory have finished writing?

Default: 2 minutes (`793b18`), until a measured capture replaces it. Alternatives: 1 minute; 5 minutes. Open.

## 793c3 Is the volume of delivered notes limited?

Default: no number in the product spec; the technical spec limits delivery to what the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup returns (`793b19`). Alternative: a maximum stated in `793a2b`. Open.

## 793c5 Are lessons written straight into the vault or held for review?

Default: straight into the [vault](../../GLOSSARY.md#vault), marked automatic (`793b21`; the owner's earlier answer `old 793c4`). Alternative: [pending offers](../../GLOSSARY.md#offer) reviewed with the curate [skill](../../GLOSSARY.md#skill). Open.

## 793c6 Is the scope top-level sessions and notes only?

Default: the user's own [sessions](../../GLOSSARY.md#session) and [notes](../../GLOSSARY.md#permanent-note) only (`793b22`). Alternatives: add [subagents](../../GLOSSARY.md#subagent); add the chunk-index lookup. It reverses earlier owner answers, so it needs his confirmation. Open.

## 793c8 What about secrets in written notes?

Answered (Joe, 2026-10-11 in the earlier attempt, `old 793c7` item 10; the redaction approach and its residual risk ratified 2026-10-13, `old 793b324`): a [note](../../GLOSSARY.md#permanent-note) MUST NOT contain a credential, token, key or password that appeared in the [turn](./793a_product-spec.md#793a1a-terms). Applied in `793a2c` and `793b24`.

## 793c9 Does a turn with nothing to report show anything?

Default: nothing is shown for a [turn](./793a_product-spec.md#793a1a-terms) with nothing to report (`793b25`). Alternative: something on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on. Open.

## 793c10 Does "already has" survive a compaction?

Default: per [session](../../GLOSSARY.md#session), a [note](../../GLOSSARY.md#permanent-note) delivered once is not delivered again whatever the [agent](./793a_product-spec.md#793a1a-terms) still holds (`793b27`; it reverses the owner's earlier answer, row 11 of `793b36`). Alternative: deliver it again after a compaction. Open.

## 793c12 Is memory delivered at mid-task moments?

Default: the [prompt](./793a_product-spec.md#793a1a-terms) only (`793b8`). Alternatives: the prompt and tool results; the prompt, tool results and the four decision moments of 792a19a, which the owner approved in the earlier attempt (`old 793b187`, `old 793b240`) but which need a model judgement or a look at every tool result. Open.

## 793c13 Which earlier owner answers may the minimal set reverse?

Default: accept every reversal of `793b36` for the minimal set; any row can come back as a separate change. Alternative: keep named rows in the set (the owner names them by number). Open.

## 793c14 Is a turn that is followed by the end of the session covered?

Default: not covered; the promise holds while the [session](../../GLOSSARY.md#session) stays open for the 2 minutes (`793b37`). Alternative: covered, the work finishing after the session ends or in the next session. Open.

## 793c15 Should the activity be shown in the agent's reply, so that it reaches the claude.ai mobile and web view?

Parked (Joe, 2026-10-16, earlier attempt, `old 793c93`): "Nah, leave it for now." Default: the activity stays in the [session](../../GLOSSARY.md#session) only; alternative: the [agent](./793a_product-spec.md#793a1a-terms) ends its reply with one fixed activity line, so that it reaches the claude.ai mobile and web view. Revisit after `793a` has landed.

## 793c16 Are the ten scenarios of the first group the minimum?

Default: the ten scenarios as written (`793b40`). Alternative: fewer, the dry run and Pi folded into the first scenario of the update. Open.

## 793c17 Is the automatic behavior inside a session one mod deployed with the skills?

Default: one mod, `engram-memory`, in `agent-instructions/mods/`, deployed with the [skills](../../GLOSSARY.md#skill), running a new command for each [prompt](./793a_product-spec.md#793a1a-terms) (`793b41`). Alternative: a `UserPromptSubmit` hook in the user's `settings.json`. Open.

## 793c18 What does a delivered block say?

Default: a framing line and one line per [note](../../GLOSSARY.md#permanent-note) (file name, situation, body) (`793b42`). Alternatives: whole notes; a count and no text. Open.

## 793c19 Is the body of a note delivered whole, and on one line?

Default: whole, on one line (`793b44`). Alternatives: cut to a length, as the earlier attempt cut at 400 characters; whole with its lines kept, which needs a way to keep a line from imitating the framing (`793b43`). Open.

## 793c20 What does a session hold, so that a note is not delivered twice?

Default: the file names on the [note](../../GLOSSARY.md#permanent-note) lines of delivered blocks, plus each `engram query` and `engram show` the [agent](./793a_product-spec.md#793a1a-terms) ran, read from the [session](../../GLOSSARY.md#session)'s record (`793b45`). Alternatives: every tool result (the first version); a ledger kept by the mod. Open.

## 793c21 How long may the mod's command run, short of the hold?

Default: 7 seconds for the command, one second left (`793b46`). Alternatives: 8 seconds for the command; a limit inside the command. Open.

## 793c22 Does a dry run report the mod as a mod?

Default: the mod is listed in the Claude Code section with `agent-instructions/mods/<name>/` as its source (`793b47`). Alternatives: as before, under the [skills](../../GLOSSARY.md#skill); a report section for mods. Open.

## 793c23 Does the conformance check read this change's specs, and fail when they are gone?

Default: a test of the dev integration suite that fails, naming the cause, when the specs are missing (`793b48`). Alternatives: a target of its own; a skip when they are missing. Open.

## 793c24 How many acceptance tests run at once?

Default: at most four at once (`793b49`). Alternatives: the number of cores; one at a time. Open.

## 793c25 Does the mod use what a failed or unrecognised command printed?

Default: the output is used only when the command exited 0 and the output begins `[engram memory]` (`793b50`). Alternative: any non-empty output, as the first version did. Open.

## 793c26 Does the delivery read the records where Claude Code writes them, and does the sweep follow?

Default: the delivery reads `ENGRAM_TRANSCRIPT_DIR`, else `$CLAUDE_CONFIG_DIR/projects`, else `<home>/.claude/projects`; the sweep is unchanged (`793b51`). Alternatives: both use that order, which changes a canonical requirement of [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand); the delivery reads the home only. Open.

## 793c27 Does the delivery's lookup need imptest evidence of its arguments, or a waiver?

Default: the delivery takes the lookup as a function and `imptest` shows the arguments (`793b52`). Alternative: a waiver of the gate, with the arguments tested through the real engine. Open.

## 793c28 Does an acceptance test skip or fail when Claude Code or a login is missing?

Default: acceptance tests fail, naming the cause (`793b53`). Alternative: they skip and the target fails on any skip. Open.

## 793c29 Does the scenario of a note the agent looked up itself hold a second note?

Default: the [vault](../../GLOSSARY.md#vault) gains a second [note](../../GLOSSARY.md#permanent-note) after the [agent](./793a_product-spec.md#793a1a-terms)'s lookup, and the [prompt](./793a_product-spec.md#793a1a-terms) arrives with it (`793b54`). Alternative: the scenario stays as it was. Open.

## 793c30 Does a gate parse the mod's JavaScript, and what if node is missing?

Default: a test of the dev integration suite parses each file under `agent-instructions/mods/` with node and fails without node (`793b55`). Alternatives: the same, skipped without node; no gate. Open.

## 793c31 Do the 7 seconds bound the whole hook or the command?

Default: one 7-second timer for everything the mod does for a [prompt](./793a_product-spec.md#793a1a-terms) (`793b56`). Alternative: the command alone. Open.

## 793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?

Default: record the stall as an open risk, measure the cost of one delivery on an idle machine, optimise nothing (`793b49`). Alternatives: find and remove the stall in this change; accept the risk without recording it. Open.

## 793c33 Does the promise of automatic memory name a minimum version of Claude Code?

Default: the promise stays unconditional, and a [session](../../GLOSSARY.md#session) that cannot load the mod steps aside (row 13 of `793b36`). Alternatives: the README and the report of [`engram update`](../../GLOSSARY.md#engram-update) name the minimum Claude Code version; `engram update` checks the version. Open.

## 793c34 Does the delivery merge the parent vault, as `engram query` does?

Default: the local [vault](../../GLOSSARY.md#vault) only for now, a named node `793e6f` building the merge later (`793b57`). Alternative: build it in the first group. Open.

## 793c35 Does the mod tell a prompt the user sent from a prompt of another origin?

Default: every origin until `793e6b`, which takes the own-[session](../../GLOSSARY.md#session) clause and the filter (`793b58`). Alternative: filter now. Open.

## 793c36 Are the restated sentences of 792a2e kept as they are?

Default: the sentence of 792a2e kept verbatim (`793b60`). Alternative: reword it. Open.

## 793c37 Are the fourteen scenarios of the second group the minimum?

Default: the fourteen scenarios as written (`793b62`). Alternative: fewer, folding the two scenarios of `793a3c` or the three kinds that come from the [agent](./793a_product-spec.md#793a1a-terms)'s own words. Open.

## 793c38 Does automatic memory start sessions of its own?

Default: none; the judgement is one call of the mod's model API, which raises no [prompt](./793a_product-spec.md#793a1a-terms) or [turn](./793a_product-spec.md#793a1a-terms) (`793b63`). Alternative: a headless [session](../../GLOSSARY.md#session) of its own with a marker in its environment that makes the mod step aside. Open.

## 793c39 Which origins count as the prompts the user sends?

Default amended to (c): the delivery acts for `composer`, `bridge`, `slack-ping` and `sdk`, the judgement for `sdk` only when `ENGRAM_AUTO_CAPTURE_SDK` is on (`793b84`; the first default, all four, `793b64`). Alternative: no `sdk`, which would leave out the user's own `claude -p` runs. Open.

## 793c40 Is the window of the judgement the right one?

Default: the window as built (`793b65`). Alternatives: the whole [turn](./793a_product-spec.md#793a1a-terms) with tool results; the turn alone. Open.

## 793c41 Is the risk of a secret of another shape acceptable?

Default: the shapes as built, the rest a known risk (`793b69`). Alternatives: port the earlier attempt's patterns; write no [note](../../GLOSSARY.md#permanent-note) from a [turn](./793a_product-spec.md#793a1a-terms) in which the redaction found a credential (the next step if the owner finds the risk too large). Open.

## 793c42 Should the bound of 2 minutes be tightened?

Default: keep the 2 minutes (`793b70`); the stages together are 115 seconds (`793b85`). Alternatives: 30 seconds; 10 seconds. Open.

## 793c43 Should a missing vault end the capture before the model is asked?

Default amended to a failure that the writing records once, with no model call (`793b82`; the first default, as built, `793b71`). Alternative: record a skip with the reason and still run the judgement. Open.

## 793c44 Is the window between a note's file and its embedding a risk?

Default: accept the risk that the first [prompt](./793a_product-spec.md#793a1a-terms) after the first [note](../../GLOSSARY.md#permanent-note) of an empty [vault](../../GLOSSARY.md#vault) goes without its notes (`793b73`). Alternatives: write the sidecar before the note in the learn path; let the lookup skip a note with no embedding. Open.

## 793c45 Is the reading of a prompt in the record the rule of the transcript-reader?

Default: the rule of 792b15c3, the flag and the origin (`793b74`). Alternatives: the origin and the flag alone; a list of marks of the judgement's own. Open.

## 793c46 Is 2 seconds for a record of 1 MB the right bound for the judgement?

Default: 2 seconds for 1 MB (`793b75`). Alternatives: a tighter bound; no bound in the spec. Open.

## 793c47 Is the turn the one that began with the text the mod saw?

Default: the text the [turn](./793a_product-spec.md#793a1a-terms) started with, the last [prompt](./793a_product-spec.md#793a1a-terms) as the fallback (`793b76`). Alternatives: the last prompt of the record; the record's prompt ID, if the event gives it later. Open.

## 793c48 Is writing a < as &lt; the right defence of the window and the notes?

Default: write `<` as `&lt;` and say so in the instructions (`793b77`). Alternatives: a random delimiter in each request; neither. Open.

## 793c49 Should the promise that a note holds no credential stay absolute?

Default: the promise names the forms, the rest staying the risk of `793c41` (`793b78`). Alternatives: absolute, with no [note](../../GLOSSARY.md#permanent-note) from a [turn](./793a_product-spec.md#793a1a-terms) in which the redaction found a credential; absolute, with the model checking each note for a secret. Open.

## 793c50 Is a lesson of a cut reply worth writing?

Default: keep the whole lessons of a cut reply and record the cut (`793b79`). Alternatives: write nothing from a cut reply; raise the reply's cap. Open.

## 793c51 Should the delivered line mark a note written automatically?

Default: append the mark `(written automatically)` (`793b80`). Alternatives: no mark; the mark in the framing line, as a count. Open.

## 793c52 Is the redaction of a text written as one line the right order?

Default: write as one line, redact, then cut (`793b81`). Alternatives: redact first; redact before and after. Open.

## 793c53 Should a judgement without a record fail, and be recorded as a failure?

Default: a judgement without a record fails and is recorded as `failed` (`793b82`). Alternatives: a `skipped` record, which needs a second report from the mod; success with no trace. Open.

## 793c54 Should a failure that the writing recorded still fail the command?

Default: succeed once the failure is recorded (`793b83`). Alternatives: fail, and the mod does not report a failure of the writing; fail, and the writing does not record it. Open.

## 793c55 Is 9 passes in 10 runs the bar for a scenario that the judgement of a model decides?

Default: 9 passes in 10 runs (`793b86`). Alternatives: 10 in 10; a bar for each scenario by the cost of a miss. Open.

## 793c56 Is the bar of the learn skill the bar for a note written automatically?

Default: the learn [skill](../../GLOSSARY.md#skill)'s bar, with the exact names kept (`793b87`). Alternatives: a lower bar, every confirmation a lesson; the skill's bar to the letter, the principle without the instance. Open.

## 793c57 Should a reversal of the agent's own claim be written as a fact or as feedback?

Default: a reversal of the [agent](./793a_product-spec.md#793a1a-terms)'s own claim is a fact, the corrected conclusion (`793b65`). Alternatives: feedback, as the [skill](../../GLOSSARY.md#skill) writes it; both. Open.

## 793c58 Is the probe of the judgement the right way to change its instructions?

Default: the probe, with the live rows as the check (`793b88`). Alternatives: the live rows alone; a probe through the host's model call in a mod of its own. Open.

## 793c59 Should an automatic note be offered to the parent vault while the duplicate check ignores it?

Default: offer upward, check locally and let the parent's curation fold duplicates (`793b89`), then merge the parent [vault](../../GLOSSARY.md#vault) into the check once `793e6f` is built. Alternative: do not offer automatic [notes](../../GLOSSARY.md#permanent-note) upward. Open.

## 793c60 Should a request to remember be written when the agent says that it did not save it?

Default: write it (`793b90`). Alternatives: leave it to the [agent](./793a_product-spec.md#793a1a-terms)'s word; write it and say in the [note](../../GLOSSARY.md#permanent-note) that the agent refused. Open.

## 793c61 Are references the IDs of nodes, or links with their titles?

Default (a): the ID in a code span, because the titles in the labels would add about 5,600 words (`793b93`); a reference into `docs/specs` stays a link. Alternatives: (b) the "ID title" link that `docs/AGENTS.md` asks for, restored by a mechanical rewrite. Blocks: the reference check at promotion.

## 793c62 Does 793d5 restate the product scenarios?

Default (a): no, each mapping cites the scenario's ID, because the text is `793a`'s and a copy must follow every change to it (`793b94`). Alternatives: (b) restate each at its first mapping, as `docs/TECHNICAL-SPEC-GUIDE.md` asks (1,859 words). Blocks: the traceability check at promotion.

## 793c63 Is an open question one line, or context, options and recommendation?

Default (a): one line (`793b95`). Alternatives: (b) the sections `docs/AGENTS.md` lists. Blocks: the check of this file against `docs/AGENTS.md` at promotion.

## 793c64 Is the order of the groups a table or a graph?

Default (a): the table of `793e4`, with `793e9` the one graph (`793b96`). Alternatives: (b) a chain graph in `793e4` as well, as the plan guide has the graph carry the order. Blocks: the plan guide's rules for the graph at promotion.

## 793c65 Do the catalog rows keep their three commit columns?

Default (a): yes, as the record of step 5 (`793b97`). Alternatives: (b) drop them and leave the commits to git history. Blocks: the gate of `793e7` that every `Done` row has its three commits.

## 793c66 Does a requirement with two owning components stay whole?

Default (a): `793d4j` and `793d4m` stay one requirement each, because the shapes of a credential and the origins of a [prompt](./793a_product-spec.md#793a1a-terms) are stated once (`793b98`). Alternatives: (b) one requirement for each owner, as `docs/TECHNICAL-SPEC-GUIDE.md` asks. Blocks: the canonical component files at promotion.

## 793c67 Do the default-suite tests that cross the real filesystem keep their tag?

Default (a): deferred, not changed in this change (`793b99`). Alternatives: (b) tag them `integration`; (c) inject the filesystem so that they are pure. Blocks: the guide's rule that the default suite holds only pure tests, at promotion.
