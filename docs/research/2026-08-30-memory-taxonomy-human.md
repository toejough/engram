> Landscape research (docs/README.md `design/`-charter sibling rule: conclusions graduate to ADR/specs/ROADMAP; file is deletable once extracted). Produced 2026-08-30 by the memory-taxonomy research workflow (4 parallel researchers + critic), explore session on #735 → memory north star. Human memory science briefing.

# Human Memory Systems: A Briefing for LLM-Agent Memory Design

**Audience:** an engineer building an agent memory system (vault of notes + embedding retrieval + skills that read/write it) that deliberately mirrors human memory categories.
**Stance:** cognitive science is used here as a *source of design predictions*, not as a metaphor to decorate the README. Each section ends with what the science actually predicts for the agent system. A final section separates load-bearing analogies from decorative ones.

---

## 1. The taxonomy: declarative vs nondeclarative (Squire), episodic vs semantic (Tulving)

**Consensus definitions.** Squire's taxonomy (1987; 2004) divides long-term memory by whether contents are consciously accessible and verbally reportable:

- **Declarative (explicit)** — hippocampus/medial-temporal-lobe dependent; flexible, one-shot capable, reportable.
  - **Episodic** (Tulving 1972; 1983): memory for specific events with spatiotemporal context — *what happened, where, when*, bound into a single occasion, with "autonoetic" self-involvement ("I was there").
  - **Semantic**: decontextualized knowledge — facts, concepts, meanings — with no memory of the learning episode ("Paris is the capital of France," but you don't remember learning it).
- **Nondeclarative (implicit)** — a grab-bag united only by *not* being declarative: **procedural memory** (skills/habits; basal ganglia, cerebellum), priming (neocortex), classical conditioning (amygdala/cerebellum), habituation. Expressed through *performance*, not recollection. Amnesic patient H.M. could learn mirror-drawing across days while denying he'd ever done the task — the canonical dissociation.

**The modern view: the boxes leak.** Three qualifications every design should internalize:

1. **Episodic → semantic consolidation ("semanticization").** Repeated or schema-consistent episodes lose their contextual detail and become semantic knowledge. The modern framing (Winocur & Moscovitch's Trace Transformation Theory; Moscovitch & Gilboa's 2024 Oxford Handbook review) is that a *gist/schematic version* of an episodic memory forms in neocortex while a detailed version can persist hippocampally; which one dominates depends on retrieval demands. It is a *transformation*, not a file transfer — both traces can coexist and interact bidirectionally, lifelong.
2. **Episodic memory bootstraps semantic learning.** New facts are typically acquired *through* episodes; semantic memory is largely the residue of many episodes with the episode-specific features averaged out (this is also exactly how complementary-learning-systems models describe cortical learning — see §5).
3. **Declarative knowledge scaffolds procedural acquisition** (ACT-R, §2). The taxonomy describes *storage systems*, not walls between them; in normal function they operate simultaneously on the same experience.

**Predicts for agent design:**
- A two-type note store — *episodic* notes (session-bound, timestamped, full context: what was tried, what failed, verbatim error text) and *semantic* notes (decontextualized rules/standards) — is well-motivated, **but only if there is a consolidation path between them.** A vault where episodes never get distilled into rules mirrors amnesia-adjacent pathology, not healthy memory.
- Keep **provenance links** from semantic notes back to source episodes (the human system effectively does this early in consolidation, and detail-on-demand is what Trace Transformation says the hippocampus keeps providing). When a rule is challenged, the episodes are the evidence.
- Expect and design for **coexisting redundancy**: the gist-rule and the detailed episode both live in the store, and retrieval demands decide which surfaces. Deduplication that deletes episodes once a rule exists throws away the high-resolution trace humans demonstrably keep.

---

## 2. Procedural memory: how skills are actually acquired — and why a runbook is not one

**Fitts & Posner (1967) three stages:**
1. **Cognitive stage** — the skill is represented *declaratively*: verbal instructions, worked examples, explicit rules. Performance is slow, serial, error-prone, and consumes attention; learners literally talk themselves through it.
2. **Associative stage** — errors are detected and pruned; components get associated; verbal mediation drops away.
3. **Autonomous stage** — performance is fast, parallel-capable, low-attention, and *no longer verbally accessible*. Experts often cannot state what they do (and forcing them to introspect degrades performance — "choking," Beilock).

**ACT-R's mechanism (Anderson 1982, 1987, 1993):** *knowledge compilation.* The learner starts by holding declarative facts (the recipe) in working memory and interpreting them via general-purpose productions — slow and load-heavy. With practice, **proceduralization** builds task-specific production rules that no longer reference the declarative trace, and **composition** collapses multi-step rule chains into single rules. The result follows the **power law of practice** (Newell & Rosenblatt): speedup is large early, then asymptotic. Crucially, after compilation the declarative recipe can be *forgotten entirely* while the skill persists — the skill is a different representation, not a faster read of the same one.

**Ryle (1949): knowing-that vs knowing-how.** Ryle's argument was precisely that knowing-how is not reducible to knowing-that: applying a proposition itself requires know-how, so an infinite regress looms if all skill is propositional. Cognitive science largely vindicated the functional distinction (dissociable neural systems, amnesia dissociations), though there's a live philosophical literature (Stanley & Williamson's intellectualism) arguing the conceptual boundary is fuzzier than Ryle claimed.

**The honest point you asked for:** a written runbook that an agent retrieves and interprets is **declarative memory about a procedure** — it is the *cognitive stage*, permanently. Human data says the cognitive stage has a specific, measurable performance signature: slow, serial, attention-hungry, error-prone at step boundaries, fragile under concurrent load. An agent interpreting a retrieved runbook inherits every one of those properties, in token-cost and instruction-following-failure form. And with frozen weights, there is *no within-inference path* to the associative or autonomous stage: the agent re-enters the cognitive stage on every retrieval, forever.

**Does the distinction matter functionally?** Yes, in four measurable ways: (1) **speed/cost** — interpretation costs context tokens and serial reasoning steps each time; compiled skill is amortized; (2) **load robustness** — declarative interpretation collapses under concurrent demands (§8), compiled skill doesn't; (3) **failure mode** — cognitive-stage errors are omissions and mis-orderings of steps; autonomous-stage errors are context-inappropriate habit intrusions (strong-habit capture, e.g. driving toward work on a Saturday). Agents running runbooks will show the *former* profile — plan on step-skipping and ordering bugs, not habit intrusions; (4) **transfer** — declarative knowledge transfers flexibly to novel variants; compiled skill is narrow. The runbook's weakness is also its strength: it stays inspectable, editable, and general.

**Predicts for agent design:**
- **Stop expecting runbooks to behave like skills.** They are permanent novice-stage scaffolding. Their measured value should be highest exactly where the model lacks priors (idiosyncratic, project-specific procedure) and near zero where the base model is already "autonomous" from pretraining — the pretrained weights *are* the agent's procedural memory, compiled by gradient descent instead of practice. (This matches the engram-verified finding that memory clean-wins only on idiosyncratic content.)
- **Build the analog of compilation anyway, at the artifact level:** practice should *compress the note*. After N successful executions, a 40-line runbook should be amended toward a short parameterized checklist (composition), and its verbose rationale demoted to linked episodic provenance. Frequency-of-use should drive **migration up the loading hierarchy**: hot procedures move from retrieved-on-demand → always-loaded skill/system-prompt text (the closest inference-time analog of automaticity: zero retrieval latency, always "in mind"), and — the only true compilation available — into **fine-tuning/RL data** when the platform allows.
- **Preserve the cognitive-stage advantages deliberately:** because agent procedures never automatize, they also never become uninspectable. Keep runbooks the *editable source of truth* and treat any compiled/compressed form as derived.
- Runbook format should respect cognitive-stage failure modes: numbered steps with explicit ordering constraints and *verification substeps* (the associative stage's error-pruning, done by the artifact since the agent can't learn it).

---

## 3. Prospective memory: remembering to remember → when to fire recall

**Definitions.** Prospective memory (PM) = forming an intention now, executing it later at the right moment, while engaged in an ongoing task. Two kinds: **event-based** (act when a cue occurs: "give Joe the message when you see him") and **time-based** (act at/after a time: "call at 3pm"). Time-based PM is reliably harder — no environmental cue; requires self-initiated clock-checking.

**McDaniel & Einstein's multiprocess theory** (2000; still the consensus frame): intentions get retrieved via two routes:
- **Spontaneous/automatic retrieval** — the cue itself triggers the intention with no ongoing monitoring cost. Works well when the cue is **focal** (processed as part of the ongoing task), distinctive, and strongly associated with the intended action.
- **Strategic monitoring** — actively maintaining the intention and scanning for cues. Works for nonfocal/weak cues but imposes measurable ongoing-task costs (slowed responses on *every* trial, whether or not the cue appears) and collapses under load or delay.

**Why PM fails:** (a) the cue occurs but isn't *noticed* (nonfocal cue, attention absorbed elsewhere); (b) the cue is noticed but the intention isn't retrieved (weak cue–intention association); (c) the intention is retrieved but execution is deferred and lost ("I'll do it after this paragraph" — deferred-execution failures are rampant); (d) **output-monitoring failure** — forgetting you already did it, causing repetition or omission. The **intention-superiority effect** (Goschke & Kuhl 1993): uncompleted intentions are held at heightened activation and pop into mind more readily than neutral material — the mind subsidizes pending intentions (related: Zeigarnik effect). Completion cancels the boost.

**The strongest applied result: implementation intentions** (Gollwitzer 1999). Encoding an intention in explicit *if-situation-then-action* form ("**When** I close the PR, **then** I run /learn") dramatically improves cue-driven execution over goal-form intentions ("I should capture lessons"), with medium-to-large effect sizes across hundreds of studies. Mechanism: it pre-forms the cue→action association so retrieval becomes automatic rather than monitored.

**Predicts for agent design (this is your when-to-fire-recall problem, isomorphically):**
- **Strategic monitoring is the wrong architecture.** "Check every turn whether recall is warranted" is nonfocal monitoring: it taxes every step and still misses cues under load. (Engram's measured 147x over-fire for "before tool calls" triggers is exactly the monitoring-cost signature the human literature predicts.)
- **Engineer focal cues instead.** Make the recall trigger *part of the processing the agent is already doing at that moment*: hooks on tool-call sites, error events, "about to declare done," task-init — points where the cue is processed en route, costing nothing when absent. Deterministic hooks are better-than-human here: a hook literally cannot fail to notice its cue. Use them wherever the cue is mechanically detectable.
- **Write ambient guidance as implementation intentions.** "When X happens, run /recall glance" (if-then, concrete cue named) — not "remember to consult memory." Engram's recall.md/learn.md cue lists already have this shape; the literature says the *if-then concreteness of the cue wording* is the active ingredient, so keep cues perceptual/situational ("a check failed and you can't explain it"), not dispositional ("when uncertain").
- **Close the loop on completion** (output monitoring): record that recall fired for this cue-instance, or you get both repetition (re-firing, cost) and false belief that it fired (omission). An intention ledger with explicit completion-marking mirrors the activation-then-cancellation dynamics humans use.
- **Beware deferred execution.** "I'll capture that lesson after the fix" is the canonical PM failure; the learn.md rule "capture before applying the fix" is the correct mitigation — execute-on-retrieval, never defer.

---

## 4. Encoding specificity and transfer-appropriate processing → how to key notes

**Encoding specificity (Tulving & Thomson 1973):** retrieval succeeds to the degree that cues available at retrieval match the properties encoded with the trace. Not "good cues" in the abstract — cues that were *part of the encoding context*. Striking demonstrations: recognition can fail where cued recall succeeds if the recognition probe mismatches encoding context; context-dependent memory (Godden & Baddeley 1975: divers who learned words underwater recalled them better underwater); state- and mood-dependent variants.

**Transfer-appropriate processing (Morris, Bransford & Franks 1977):** the deeper principle — memory performance depends on the *match between the processing done at encoding and the processing required at retrieval*. Semantic encoding beats phonemic encoding for a semantic test but *loses* to phonemic encoding on a rhyme test. There is no context-free "good encoding"; there is only encoding matched to the anticipated retrieval situation.

**Predicts for agent design:**
- **Key notes by the retrieval moment, not the content topic.** The future retrieval cue is *the situation the agent will be in*: the symptom, the error string, the task phrasing, the file being touched — not the name of the solution. A note titled/embedded around "flaky check-nils failures with `signal: killed`" is retrievable at the moment of need; the same lesson keyed as "tmpfs cleanup in testbinary_test.go" is not, because at retrieval time the agent doesn't know that vocabulary yet. Write the *symptom-side* vocabulary into the note explicitly.
- **Embed the query-side text, retrieve against situation snapshots.** With embedding retrieval, encoding specificity translates to: the note's embedded surface should contain the same *kind* of text that will be in the context window at retrieval time (error output, user phrasing, tool names). This is why concrete idiosyncratic tokens proved load-bearing in engram's question-anchoring eval — abstracting them away is *destroying encoding-specific cues*, and the human literature predicted that result.
- **Encode multiple retrieval routes** (Anderson's elaboration work): one lesson, several cue-framings (symptom, task-type, artifact name). Cheap at write time; each added route is an independent chance of a cue-match later.
- **Procedure notes specifically** should open with their *firing conditions* ("Use when: ...") in situation vocabulary — the same reason skill descriptions trigger correctly or not. TAP also warns against evaluating retrieval with topic-worded probes: test with realistic *situation* probes (transcript-shaped), or the eval measures the wrong match. 

---

## 5. Consolidation and reconsolidation → replay jobs and update-on-use

**Systems consolidation.** The standard model (Squire & Alvarez; McClelland, McNaughton & O'Reilly 1995): the hippocampus rapidly binds an episode; over time (sleep replay is the leading mechanism), repeated reinstatement *teaches* neocortex a distributed, schema-integrated version, and the memory becomes progressively hippocampus-independent. The computational rationale — **Complementary Learning Systems (CLS)** — is directly relevant to you: a fast one-shot store and a slow statistical learner are *both* necessary because interleaved slow learning avoids **catastrophic interference** while extracting structure, and the fast store lets you act on single experiences meanwhile. (Kumaran, Hassabis & McClelland 2016 explicitly updated CLS as a blueprint for artificial agents.) Two modern refinements: (a) **Multiple Trace / Trace Transformation Theory** (Moscovitch, Winocur, Gilboa — 2024 Oxford Handbook treatment): consolidation is a *transformation* to a gist/schematic trace, with detailed hippocampal traces persisting for vivid episodic memories and ongoing bidirectional hippocampal–neocortical collaboration — not a handover with deletion. (b) **Schema-accelerated consolidation** (Tse et al. 2007): when new information fits an existing schema, cortical integration is dramatically faster. Prior structure is the rate-limiter, not time per se.

**Reconsolidation.** Nader, Schafe & LeDoux (2000): a reactivated (retrieved) memory can return to a labile state and must be re-stabilized ("reconsolidated"), during which it can be modified, strengthened, weakened, or updated. Honest status per the current literature: the *core phenomenon* is well-established in animal models; **human clinical translation is fragile**. Destabilization is not guaranteed by mere retrieval — it appears to require **prediction error** at reactivation (and there's evidence for both a lower and upper bound on how much PE, dependent on prior learning), old/strong memories resist, and human propranolol interventions have notable replication failures alongside positive PTSD trials. The safe consensus: *retrieval is an opportunity for updating, gated by surprise/mismatch* — not an automatic rewrite.

**Predicts for agent design:**
- **Update-on-use, gated by prediction error.** When a recalled note is *used* and the outcome mismatches it (the note said X, reality did Y), that is exactly the destabilization condition — amend the note then, in that turn, while the evidence is in context. When a recalled note is used and simply *confirmed*, strengthen (bump activation/confidence) but don't rewrite — mere reactivation without mismatch shouldn't churn content (the human boundary conditions are effectively a guard against noise-driven corruption; adopt the same guard).
- **Run the replay job.** An explicit offline consolidation pass — batch-summarizing accumulated episodic notes into/against semantic notes, *interleaved with existing notes* rather than overwriting — is the CLS prescription, and it's the piece most vault designs skip. Engram's `ingest --auto` sweep at cycle-close is this job; the science says its key property must be interleaving (integrate new lessons with retrieved old ones, checking for conflict) rather than append-only.
- **Schema effects cut both ways:** lessons consistent with existing vault structure are cheap to integrate (link and file fast); *schema-inconsistent* lessons — the ones contradicting an existing note — are precisely the valuable, expensive ones. Surface contradictions to a judging step (curate-style) instead of auto-merging; humans mis-consolidate schema-inconsistent material by distorting it toward the schema (Bartlett, §6), and an auto-merger will do the same.
- **Don't delete episodes after distillation** (§1): transformation, not handover.

---

## 6. Working memory, chunking, and schemas → context-window discipline

**Baddeley's model** (Baddeley & Hitch 1974; episodic buffer added 2000): a limited-capacity **central executive** (attention control) coordinating a **phonological loop** (speech-based rehearsal, ~2 seconds' worth), a **visuospatial sketchpad**, and an **episodic buffer** binding WM contents with LTM into integrated chunks. Capacity: Miller's 7±2 was about *chunks*, and the modern estimate is **Cowan's ~4 chunks** of attended information. Contents decay without rehearsal and are hostage to attention.

**Chunking is the capacity multiplier.** Chase & Simon (1973): chess masters recall realistic board positions vastly better than novices — but *only* for legal positions; random boards erase the advantage. Expertise = a vocabulary of learned chunks that lets 4 slots carry huge structured content. **Long-term working memory** (Ericsson & Kintsch 1995): experts bypass WM limits by encoding into LTM with *retrieval structures* — stable cue systems that make LTM contents accessible as if they were in WM.

**Schemas** (Bartlett 1932; modern schema theory): organized prior knowledge structures that (a) guide encoding (schema-relevant details are kept), (b) enable massive compression (store only deviations from the default), and (c) drive **reconstructive distortion** — Bartlett's "War of the Ghosts" retellings normalized unfamiliar material toward the readers' cultural schemas. Memory is reconstruction from gist + schema defaults, not playback; schema-based intrusions feel exactly like memories.

**Predicts for agent design:**
- The context window is the agent's only working memory, and while it's ~5 orders of magnitude bigger than 4 chunks, it shares the *functional* limits: attention over it is uneven (primacy/recency; "lost-in-the-middle" degradation is the empirical LLM analog), and stuffing it has a real performance cost — so recall payload budgets are a WM-hygiene issue, not just a token-cost issue.
- **Chunk before injecting.** A recall payload of 12 raw episodes competes for attention 12 ways; the same content pre-consolidated into one structured note is one chunk. The Chase–Simon lesson: the compression only works when the structure matches the domain's real regularities — generic summarization is a random-board chunk.
- **LTWM is the design pattern for the vault interface:** keep a *small always-loaded retrieval structure* (index/description layer — cue phrases pointing at vault notes) and fetch bodies on demand. That's Ericsson & Kintsch implemented. Caveat from engram's own lazy-chunks eval: it only wins if the agent actually fetches when needed — the retrieval-structure cues must be strong enough to drive the fetch (§4).
- **Schema compression is lossy in a biased way.** Summarization-at-consolidation will normalize idiosyncratic details toward the model's priors — which deletes exactly the idiosyncratic content that carries the vault's verified value. Rule: compress *structure*, pin *tokens* (keep verbatim error strings, flag values, names inside the compressed note).

---

## 7. Forgetting: interference, retrieval-induced forgetting, spacing, retrieval practice → decay/activation design

**Interference, not decay, is the dominant cause of forgetting** (McGeoch 1932 onward): **proactive** (old learning impairs new) and **retroactive** (new impairs old). The unifying principle is **cue overload** (Watkins): a cue's effectiveness is divided among everything attached to it — forgetting is mostly *retrieval competition*, with the trace still present.

**Retrieval-induced forgetting** (Anderson, Bjork & Bjork 1994): practicing retrieval of some items *actively suppresses* related unpracticed competitors — beyond mere lack of practice. Selective retrieval reshapes the store.

**New theory of disuse** (Bjork & Bjork 1992) — the most design-ready framework: every trace has **storage strength** (how well learned; effectively never decreases) and **retrieval strength** (current accessibility; drops with disuse and competition). Key interactions: retrieval practice boosts both, and the boost to storage strength is *larger when retrieval strength is low* — hard, spaced retrievals build durable memory ("desirable difficulties").

**Spacing effect** (Ebbinghaus 1885; Cepeda et al. 2006 meta-analysis): distributed practice beats massed, robustly; optimal gap scales with retention interval. **Testing effect** (Roediger & Karpicke 2006): retrieval practice beats restudy for long-term retention — *use is the strengthener, not re-reading*.

**ACT-R's rational analysis** (Anderson & Schooler 1991) ties it together: base-level activation ≈ log of summed, power-decaying uses — recency + frequency — which provably tracks the environmental *probability that the item will be needed now*. Human forgetting is a need-probability estimator, not a defect.

**Predicts for agent design:**
- **Two-variable bookkeeping, not one decay scalar.** A single confidence×0.9/session decay conflates storage and retrieval strength. Keep them separate: *storage strength* (validated correctness — should essentially never decay on its own; a verified lesson from 2025 is still true) and *retrieval strength/activation* (recency+frequency of use — this is what ranking should use, ACT-R-style: log-sum of power-decayed uses). Prune on *invalidation or supersession*, demote on disuse.
- **Retrieval is a write.** Every surfaced-and-used note should get an activation bump (testing effect: use, not re-storage, is what strengthens). A note retrieved 10 times and never used is signal too — cue matches but content doesn't help: rewrite or re-key it (§4), don't just decay it.
- **Cue overload is the vault's scaling disease.** As near-duplicate notes accumulate on the same cue vocabulary, each retrieval's effectiveness divides among them. The fix is consolidation (merge competitors into one strong note, §5), which is *also* the mitigation for the retrieval-induced-forgetting analog: always surfacing top-k strengthens winners and structurally buries near-miss competitors. Occasionally audit what sits just below the cutoff for high-traffic cues.
- **Interference predicts the failure mode of stale procedure notes:** the old runbook proactively interferes with the new one *sharing its cues*. On supersession, don't just add the new note — explicitly mark/attach-to/retire the old one, or retrieval will split between them indefinitely.

---

## 8. Composing multiple procedures at once → hierarchy, automatization, and the delegation analog

**What's known about doing two things at once:**
- **Automaticity is the enabler.** Dual-task costs are severe when both tasks are attention-demanding and near zero when one is autonomous-stage (§2). Driving-while-talking works until driving demands spike, and conversation measurably *pauses* — the executive reclaims the channel. A cognitive-stage skill essentially cannot be dual-tasked.
- **There is a central serial bottleneck.** The psychological refractory period (Pashler): response *selection* — deciding what to do — is queued one-at-a-time even when perception and action can overlap. Task-set switching (Rogers & Monsell 1995) costs time on every switch, plus residual "task-set inertia" — the just-abandoned task's settings interfere with the new one.
- **Skills are hierarchies, not chains.** Lashley (1951) demolished stimulus-chaining accounts of serial behavior; the consensus is hierarchical plan structures (Miller, Galanter & Pribram's TOTE units; ACT-R goal stacks; hierarchical RL "options" as the computational descendant). A cook running two recipes doesn't interleave step-by-step at random — they hold a scheduling plan over *subroutines*, where each subroutine (chop, sauté) is automatized and, crucially, has known durations and interruption-safe boundaries. Expertise in composition = knowing where the safe suspension points are.
- **Interleaving in *learning*** (contextual interference, Shea & Morgan 1979): practicing skills interleaved rather than blocked hurts immediate performance but improves retention and transfer — a desirable difficulty consistent with §7.

**Predicts for agent design:**
- **Two retrieved runbooks in one context = two cognitive-stage tasks = the worst dual-task case.** The literature flatly predicts interleaving errors: cross-talk between step-sets, task-set inertia (applying runbook A's convention inside runbook B's step), and losses at every switch. Where the agent must follow two procedures, expect and check for exactly these — step-skipping at interleave points and convention bleed.
- **The human solution translates directly: hierarchize and delegate.** Compose at the *plan* level over subroutine boundaries, and make subroutines "automatic" by giving each to a context that runs *only it* — a subagent with one runbook is the architectural analog of an automatized subroutine: it consumes no executive/context attention in the orchestrator, cannot cross-talk with the sibling procedure, and reports back at a designed suspension point. The delegate-everything doctrine is, in cognitive-science terms, the *only* mechanism an LLM agent has for manufacturing automaticity's dual-task benefits, since it can't compile skills.
- **Design suspension points into runbooks** (interruption-safe boundaries with explicit resumption state), because the orchestrator's serial bottleneck is real: it will handle subagent returns one at a time, and a runbook that can't be safely suspended mid-flight will lose state exactly the way human PM loses deferred intentions (§3).

---

## 9. Where the human analogy breaks for LLM agents

Be explicit about the disanalogies, because several design temptations come from ignoring them:

1. **No weight updates at inference.** Human memory's substrate is plasticity everywhere, continuously. The agent's weights are frozen: there is no associative stage, no automatization, no true consolidation into the "cortex." Every mechanism in §§2,5 must be re-implemented *at the artifact level* (note compression, loading-tier migration, fine-tune pipelines) or it doesn't exist.
2. **The pretrained model is already a colossal semantic-plus-procedural memory.** No human enters a task with the equivalent of the internet consolidated. This inverts the economics: the vault should not store what the model already knows (it will only add interference and cost — confirmed empirically by engram's C1–C6 ledger: value concentrates entirely in *idiosyncratic* content). Human-memory-inspired designs that dutifully record everything are solving a problem the base model already solved.
3. **Working memory is re-readable and enormous, but attention-limited.** Human WM decays in seconds and is capacity-4; the context window persists for the session, is randomly addressable, and holds 10⁵–10⁶ tokens. But the *attention* over it degrades (§6), so WM discipline survives as payload budgeting rather than rehearsal.
4. **Retrieval is the only bridge, and it's a different retrieval.** Human retrieval is content-addressable spreading activation with the current mental state as an automatic, free, always-on cue. Agent retrieval is an explicit, costly *action* over embedding similarity — which is why prospective memory (§3) is the load-bearing problem for agents in a way it never is for the always-on human system: the agent must *decide* to remember.
5. **No autonomous offline consolidation.** Sleep replay is free and mandatory for humans. The agent's replay is a batch job someone must schedule, pay for, and gate (§5) — skip it and you get an append-only episodic pile, which is not a memory system, it's a log.
6. **Forgetting is a policy choice.** Human forgetting is enforced by the substrate and is (per Anderson & Schooler) adaptive need-probability tracking. The agent could keep everything forever; the *reasons* to forget are retrieval competition and cue overload (§7) — so implement forgetting as ranking demotion and consolidation, not deletion-by-decay-schedule.
7. **No continuous self.** Sessions die; there is no single experiencing agent accumulating skill across them. "Practice" only exists if the artifacts carry it (use-counts, amendments). Any design that implicitly assumes the same agent gets better must route that improvement through the store.

---

## 10. Where the analogy is load-bearing vs decorative

**Load-bearing** (the human result transfers as an engineering prediction, and skipping it costs measured performance):

| Analogy | Why it bears load |
|---|---|
| **Episodic/semantic split + consolidation pipeline** (§1,5) | CLS gives the *computational* argument (fast store + slow interleaved integration avoids interference while extracting structure) — it's a theorem-shaped claim about any learning system, not a biology fact. |
| **Encoding specificity / TAP** (§4) | Directly governs embedding-retrieval hit rates: key notes by the retrieval-moment's vocabulary. Engram's own evals (idiosyncratic tokens load-bearing; matched-note floor) landed where this literature points. |
| **Multiprocess PM theory** (§3) | Predicts the exact cost structure of recall-triggering: monitoring taxes every step (measured 147x over-fire); focal cues + implementation-intention wording + deterministic hooks are the fix. |
| **Storage vs retrieval strength; testing effect; cue overload** (§7) | Gives the correct two-variable decay/activation design and names the vault's scaling disease (duplicate notes splitting cue effectiveness). |
| **Cognitive-stage signature of interpreted runbooks** (§2) | Predicts runbook failure modes (step-skipping, ordering, load fragility) and where runbooks pay (idiosyncratic content only). |
| **Automaticity/hierarchy for composition → delegation** (§8) | Dual-task literature predicts two-runbooks-one-context interference; subagent-per-procedure is the available automaticity substitute. |
| **Reconsolidation's PE gate → update-on-use** (§5) | Update on mismatch, strengthen on confirmation, guard against churn on mere reactivation. |

**Decorative** (fine as naming, but don't let it drive design):

- **Neuroanatomy mapping** (hippocampus = vault, cortex = weights, basal ganglia = skills): evocative, but no design decision should be justified "because that's what the brain region does" — only by the computational rationale behind it.
- **"Procedural memory" as a label for runbook notes**: actively misleading if believed — they are declarative artifacts about procedures (§2). Keep the name if convenient; keep the §2 predictions in view.
- **Seven-plus-or-minus-two / any literal capacity number**: the *chunking and attention-competition* principles transfer; the constants do not.
- **Autonoetic consciousness, mental time travel, emotional modulation (flashbulb memories, amygdala effects)**: rich human phenomena with no agent counterpart yet; importing them adds vocabulary, not function.
- **Decay-per-session constants mimicking forgetting curves**: human-shaped decay schedules are decorative; the load-bearing version is need-probability ranking (recency+frequency) with storage strength kept separate.

---

### Spot-checked sources (recent/moving literature)

- [Moscovitch & Gilboa, "Systems Consolidation, Transformation, and Reorganization: MTT, TTT, and Their Competitors," Oxford Handbook of Human Memory (2024)](https://academic.oup.com/edited-volume/57928/chapter/475478430) — modern consolidation-as-transformation consensus.
- [Winocur & Moscovitch, "Memory Transformation and Systems Consolidation"](https://www.semanticscholar.org/paper/Memory-Transformation-and-Systems-Consolidation-Winocur-Moscovitch/c1f6c35717ae3999b3d68d82fbd357b816078de1) — trace transformation theory.
- [Demarcating the boundary conditions of memory reconsolidation: an unsuccessful replication (Sci Rep 2022)](https://www.nature.com/articles/s41598-022-06119-5) — PE necessary-not-sufficient; lower and upper PE bounds; replication fragility.
- [Propranolol PTSD reconsolidation meta-analysis (J Psychiatr Res 2022)](https://www.sciencedirect.com/science/article/abs/pii/S0022395622001741) and [failure to engage destabilisation (Neuroscience 2021)](https://www.sciencedirect.com/science/article/abs/pii/S0306452221005571) — mixed human clinical evidence; destabilization is the gate.
- [Hippocampal connectivity after propranolol reactivation, randomized fMRI study (2025)](https://www.tandfonline.com/doi/full/10.1080/20008066.2025.2466886) — ongoing positive-side evidence in PTSD.

Everything else in this briefing (Squire taxonomy; Tulving 1972/1983; Tulving & Thomson 1973; Morris et al. 1977; Anderson's ACT-R 1982–1993; Fitts & Posner 1967; Ryle 1949; McDaniel & Einstein 2000; Goschke & Kuhl 1993; Gollwitzer 1999; McClelland/McNaughton/O'Reilly 1995; Kumaran/Hassabis/McClelland 2016; Nader et al. 2000; Tse et al. 2007; Baddeley & Hitch 1974/2000; Cowan 2001; Chase & Simon 1973; Ericsson & Kintsch 1995; Bartlett 1932; Anderson/Bjork/Bjork 1994; Bjork & Bjork 1992; Cepeda et al. 2006; Roediger & Karpicke 2006; Anderson & Schooler 1991; Pashler PRP; Rogers & Monsell 1995; Lashley 1951; Shea & Morgan 1979) is stable, decades-settled literature reported from training knowledge.