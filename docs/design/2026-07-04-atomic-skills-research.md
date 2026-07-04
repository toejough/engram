# Atomic skills exploration — consolidated research (S0, four beats)

Companion to `docs/design/2026-07-04-atomic-skills-options.md`; the plan's Step-1 deliverable.
**Provenance:** reconstructed at Gate C (2026-07-04) from the S0 beats' session task records —
the plan's Step-1 commit point was missed, so this is not contemporaneous capture; content is
the beats' reported findings, condensed.
Four parallel beats per the broad-sweep rule (note 157). Per-beat findings below are the beats'
own reported conclusions, condensed; labels: **measured / observed / analysis / null**.

## Beat 1 — Official Anthropic guidance (sonnet; local plugin docs + agentskills.io + platform docs)

- Skill scope is defined ONLY by size budgets — <500 lines / <5k tokens — across all four
  sources (writing-skills SKILL.md, anthropic-best-practices.md, agentskills.io spec, platform
  docs). **No official "one skill per function" concept exists** (observed).
- The single endorsed cross-skill mechanism: prose name-pointers (`REQUIRED SUB-SKILL: X`,
  `REQUIRED BACKGROUND: ...`), explicitly NOT @-imports ("force-loads files immediately,
  consuming 200k+ context" — writing-skills, quoted verbatim in the plan) (observed).
- Anti-duplication is explicit doctrine: "Don't repeat what's in cross-referenced skills"
  (observed).
- **Non-triggering descriptions: null across every source** — not endorsed, not modeled, not
  prohibited. The spec's only constraints (non-empty, describes what/when) are technically
  satisfiable by "when invoked by another skill." O-A's key device has no official backing;
  smoke evidence is the only evidence (null → smoke-load-bearing).

## Beat 2 — Shipped ecosystems (haiku; superpowers internals + 1,000+ community skills + agent frameworks)

- **No shipped system shares fragment FILES between skills — anywhere** (observed across
  superpowers 6.1.1's 14 skills [plan's "~20" corrected], anthropics/skills' 17, 20+
  awesome-lists, LangChain/AutoGen/CrewAI). The sharing unit is always the WHOLE named skill
  (prose pointer or frontmatter dependency), never an included fragment.
- Superpowers' own factoring policy (observed): discipline scaffolding is knowingly
  copy-pasted (the "Iron Law" no-X-without-Y-first blocks appear in 4 skills) so each skill
  stands alone; judgment-bearing procedures get ONE canonical home + adapt-in-place mapping
  tables elsewhere; MECHANICAL fill-in-the-blank content is extracted to templates/scripts
  WITHIN the owning skill. Exactly one references/ dir exists in the plugin (read-only data).
- Applied to engram: the 3-copy fact/feedback block is template-class (the category shipped
  systems extract); our duplicated discipline framing is duplicate-class (the category they
  deliberately keep) (analysis).
- "Instruction bleed" literature names the active-shared-component risk — supports O-C's park
  (observed).
- Marginal precedents: hermes' frontmatter `dependencies:` declarations + composition tests;
  chaparral's whole-skill symlinks. Both share whole skills, matching O-A's shape; a
  NON-TRIGGERING internal-only skill exists nowhere (null → O-A is a genuine first there).
- Vision-relevant, out of scope: dependency declarations, central versioned prompt registries,
  composition tests for skill pairs.

## Beat 3 — SE theory applied to prompt artifacts (sonnet; Metz, Dodds, Rainsberger, Yourdon-Constantine, Liu et al. 2307.03172, AGENTIF 2503.13657, + vault notes 89/137/145)

- Extraction pays iff (1) the indirection mechanism is RELIABLE and (2) the abstraction is
  CONTEXT-FREE (analysis). For prompt artifacts the "runtime" is an LLM whose dereferencing is
  probabilistic; failed indirection is SILENT (vs loud in code). AGENTIF: ~11.8% requirement-
  omission rate for condition-gated steps (measured, external).
- Cohesion audit (analysis): recall = sequential cohesion (pipeline; second-best tier);
  learn/route/please = functional. The charter's write-memory atom is high-cohesion ONLY if
  scoped to mechanical flag execution; carrying covered/near/absent branching makes it the
  wrong abstraction (Metz). read-memory is NOT cleanly extractable (recall's write step is
  sequentially coupled to its judgment).
- Rule of Three (analysis over the F2 census): triggers ONLY for the 3-copy fact/feedback
  block; the 2-copy qa and 2-copy ingest blocks do not trigger (AHA: don't extract yet;
  Rainsberger: the ingest duplication is essential, not accidental).
- Lost-in-the-Middle (Liu et al.) + vault note 137 (action-blending → skipped action,
  measured 3/5) jointly predicted O-A's risk mode: mid-body atom invocations silently skipped.
- Five testable predictions, pre-stated before the smokes ran (outcomes at haiku n=3):
  1. **O-A reliability** — O-A arms will show a higher step-skip rate than O-B arms (the atom
     invocation silently skipped mid-skill). **REFUTED** — 0 skip incidents across all O-A
     arms; the invocation was reliably followed regardless of position in the body.
  2. **O-A wrong-abstraction signal** (Metz, smoke S3) — branching errors appear if the atom
     carries covered/near/absent judgment. **REFUTED** — the atom text excludes judgment by
     construction and no branching errors were observed.
  3. **O-B partial win on S2** — the prose pointer, staying within the calling skill's body,
     will match or beat O-A's cross-skill invocation on qa capture. **INVERTED** — O-B 0/3
     (confabulated flags) vs O-A 3/3.
  4. **O-A vs O-B net tradeoff** — O-A ≥ O-B on maintainability (1 edit vs 3 for a flag
     change), O-B ≥ O-A on reliability. This was a pre-registered decision RULE, not a separate
     measurement: "if O-A passes all scenarios (refuting predictions 1 and 2), O-A is the
     better choice." **RESOLVED in O-A's favor** — O-A passed every scenario, and O-B's
     predicted reliability edge did not merely fail to appear, it reversed (see prediction 3).
  5. **O-D dominated** — zero cross-skill single-source-of-truth gain (N copies = N maintenance
     surfaces); classified as needing no smoke. **CONFIRMED the park** (unrun by design).
- Literature gap flagged honestly: all modular-prompt literature assumes an EXTERNAL
  template/injection mechanism; none covers a skill body instructing the agent to invoke a
  sub-skill mid-execution (null).

## Beat 4 — Failure modes of decomposition (haiku; vault + repo + external)

Seven-mode catalog (each: mechanism → evidence → which option it threatens → mitigation):
1. Under-fire / silent step loss — threatens O-A primarily (mitigated: checkpoint smokes;
   outcome: not observed at haiku n=3).
2. Over-fire / competing descriptions — the ~147×–380× mechanical over-fire history
   (vault note 139; the plan's P1 cited note 144 for this figure — a misattribution corrected
   in the plan) — parks O-C.
3. Judgment-seam fracture (note 78) — bounds the atom's scope.
4. Anti-amnesia heaviness core (note 100: 8/8→0/8; presence-only failed 83%) — please is
   untouchable in every option.
5. Dispatch overhead not worth it (note 80: −14% rollback) — every new skill must pay for
   itself.
6. Cross-reference drift — threatens O-B (superseded by O-B's smoke elimination).
7. Context-coupled duplication — bounds which blocks are extractable at all.

## Synthesis (what the beats jointly imply)

Official guidance and shipped ecosystems both endorse whole-named-skill sharing and reject
fragment sharing; SE theory endorses extracting exactly ONE of our duplications (the 3-copy
mechanical block) under a strict judgment-free scope; the vault's own history bounds
everything else (please untouchable, active-description atoms parked, dispatch must pay).
The one genuinely uncharted element — a non-triggering description — has no evidence anywhere
except our own smokes, which is why the options doc carries the deployed-sandbox non-fire test
as a ship-gate rather than treating the inline-fixture smokes as sufficient.
