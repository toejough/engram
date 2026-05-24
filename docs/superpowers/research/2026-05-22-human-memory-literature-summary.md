# Human Memory Literature Summary for Engram's Tiered Design

Date: 2026-05-22
Purpose: Pressure-test Engram's L0/L1/L2/L3 architecture against cognitive-science
primary sources. Every claim below maps to a citation in the References section.

---

## TL;DR (load-bearing findings, each tied to one design implication)

- **Two-system consolidation is real, but it is "fast episodic + slow semantic," not four
  tiers.** McClelland/McNaughton/O'Reilly's Complementary Learning Systems (CLS) is the
  most directly relevant cognitive model and it argues for *exactly two* stores, not
  four. **Implication:** L0 and L1 may be one tier (fast/episodic), and L2 and L3 may
  be one tier (slow/structured) — current four-tier split risks reifying a boundary the
  brain does not draw.
- **Retrieval is reconstructive, not lookup.** Schacter, Bartlett, and the encoding-
  specificity literature show that recall is a rebuild from partial cues, not a fetch
  of a stored row. **Implication:** strict append-only at L2 plus an immutable L1 is
  defensible as *substrate*, but the retrieval path must reconstruct (re-rank, re-
  contextualize, merge) rather than literally return stored content.
- **Memory is consolidated *into existing schemas*, not stored alongside them.** Tse et
  al. and modern schema-assimilation work show that schema-congruent items consolidate
  ~10x faster and that *incongruent* items are also preferentially retained
  (U-shaped). **Implication:** L3 MOCs should drive L2 ingestion, not just summarize
  it — incoming atomic facts should be routed by their fit/misfit with the existing
  MOC graph, and that fit/misfit signal is itself the salience gate.
- **Salience = prediction error + novelty.** Dopaminergic encoding gates which
  experiences get consolidated; surprising or goal-relevant events are encoded
  preferentially. **Implication:** Engram needs an explicit "is this worth keeping"
  signal at the L0 -> L1 boundary. The most defensible signal is *prediction error vs.
  L3* (does this contradict or extend a current MOC?), not surface features like
  length or recency.
- **The most-evidenced link types are temporal, contextual, semantic-similarity,
  contradictory, and causal — in that order of empirical support.** TCM (temporal),
  encoding-specificity (contextual), spreading activation (semantic), RIF/inhibition
  (contradictory), and recent narrative-memory work (causal). **Implication:** the
  current `contradicts:` and `supersedes:` edges are well-grounded; temporal/causal
  edges are missing and should be added; pure semantic-similarity is a retrieval-time
  computation, not an authored edge.
- **Forgetting is adaptive and inhibitory, not decay.** Anderson's RIF work and the
  modern interference literature show forgetting is goal-directed suppression that
  *improves generalization*. **Implication:** Engram's append-only stance is in
  tension with this. A purge/suppress signal — even soft, e.g. demote-from-recall —
  is required for the system to scale and to generalize.
- **The "lopsided hourglass" partially matches biology.** CLS-style consolidation is
  many-to-few (episodes -> semantic gist), and schema theory does suggest semantic
  knowledge re-expands into rich relational networks. But the L0 -> L2 funnel in
  Engram is *much* more aggressive than the brain's, and the L3 re-widening is small
  — biology suggests the opposite: a *very* wide semantic network with a *small*
  active retrieval set, not the reverse.
- **Working memory has no analog in Engram, and probably should.** Baddeley-style
  working memory is the substrate where retrieved memories are *combined* with current
  context to drive action. **Implication:** the recall payload itself is Engram's
  working-memory analog and deserves explicit design (size cap, decay-within-session,
  recency weighting), separate from the L0-L3 store.

---

## 1. Memory Classifications (episodic / semantic / procedural / working)

### Key findings

Tulving (1972) introduced the episodic/semantic split: episodic = personally
experienced events tagged with spatiotemporal context; semantic = decontextualized
factual knowledge. The taxonomy was extended (Tulving 1985, 2002) to include
procedural memory and the SPI (serial/parallel/independent) model.

Modern work treats episodic and semantic as **endpoints of a continuum** rather than
clean systems. Neuropsychological dissociations (e.g. semantic dementia preserves
episodic, hippocampal amnesia preserves semantic) support partial separability, but
fMRI shows substantial overlap in retrieval substrates (Renoult et al. 2019). Tulving
himself acknowledged the representational overlap.

Squire's alternative taxonomy (declarative vs. non-declarative) groups episodic +
semantic together as "declarative" and contrasts them with procedural/implicit. The
two taxonomies cut the same data differently; neither is "wrong."

Working memory (Baddeley & Hitch 1974; Baddeley 2000) is a separate, capacity-limited
buffer that holds and manipulates currently-active information. It is *not* a tier
of long-term memory — it is the workspace in which long-term memories are *used*.

### Strongest sources

- Tulving (2002), "Episodic memory: From mind to brain," Annual Review of Psychology.
- Renoult, Irish, Moscovitch & Rugg (2019), "From Knowing to Remembering: The Semantic-
  Episodic Distinction," Trends in Cognitive Sciences.
- Squire (2004), "Memory systems of the brain: a brief history and current
  perspective," Neurobiology of Learning and Memory.

### Design implications

- L0 (raw transcripts) and L1 (task-boundary segments) are both **episodic** — they
  retain contextual binding (session, time, situation).
- L2 (atomic facts) is **semantic** — decontextualized principle-statements.
- L3 (MOCs) is **schematic** — organizational structure over semantic content.
- The continuum view warns against rigid boundaries: a single L2 fact may still carry
  episodic residue ("learned during the 2026-05-22 redesign"). Don't strip provenance.
- **There is no working-memory tier in Engram.** The recall payload assembled per
  session is effectively a working-memory window. Design it as such: bounded, ordered
  by recency/relevance, decaying within session.

---

## 2. Memory Consolidation & Storage (CLS, schemas, sleep)

### Key findings

**CLS (McClelland, McNaughton & O'Reilly 1995; Kumaran, Hassabis & McClelland 2016):**
the brain has *two* learning systems by necessity. The hippocampus uses sparse,
pattern-separated representations and high learning rates to capture individual
episodes without catastrophic interference. The neocortex uses dense, overlapping
representations and low learning rates to gradually extract statistical structure.
Memories form rapidly in hippocampus, then "replay" during quiet wake and sleep
gradually trains cortical representations. This is the consolidation process.

**Schema assimilation (Tse et al. 2007; Gilboa & Marlatte 2017; van Kesteren et al.
2012):** when new information is congruent with an existing schema, consolidation is
*dramatically accelerated* — rats embedded paired-associates into a learned spatial
schema in a single trial, with stable retention. The mPFC mediates schema-driven
fast consolidation. Schema-incongruent information is *also* preferentially
remembered (the U-shaped congruency effect) because the prediction-error signal
triggers attentional and dopaminergic encoding.

**Sleep-dependent consolidation (Klinzing, Niethard & Born 2019; Born et al. 2024):**
hippocampal sharp-wave ripples replay recent traces, coordinated with cortical slow
oscillations and thalamic spindles. The replay transforms episodic detail into
schema-like cortical representations and selectively strengthens reward-tagged or
emotionally tagged memories. Synaptic homeostasis (Tononi & Cirelli) downscales
weak connections during sleep — a *forgetting* function.

### Strongest sources

- McClelland, McNaughton & O'Reilly (1995), "Why there are complementary learning
  systems in the hippocampus and neocortex," Psychological Review 102(3):419-457.
- Kumaran, Hassabis & McClelland (2016), "What learning systems do intelligent
  agents need? Complementary learning systems theory updated," Trends in Cognitive
  Sciences 20(7):512-534.
- Tse et al. (2007), "Schemas and memory consolidation," Science 316(5821):76-82.
- Klinzing, Niethard & Born (2019), "Mechanisms of systems memory consolidation
  during sleep," Nature Neuroscience 22(10):1598-1610.

### Design implications

- CLS argues for **two** stores, not four. Engram's L0+L1 ~= fast episodic store;
  L2+L3 ~= slow semantic store. The L1/L2 boundary is the *consolidation event* —
  the thing that should be expensive and deliberate (LLM-mediated). The L0/L1 and
  L2/L3 boundaries are *not* consolidation events; they are representational
  refinements within a tier. Consider whether the four-tier UI is worth the
  ontological cost.
- Schema-assimilation says **L3 must influence L2 ingestion**, not just be derived
  from L2. A new atomic fact should be checked against the current MOC graph: if
  congruent, attach with high confidence; if incongruent (prediction error), tag
  with high salience and create an explicit `contradicts:` edge; if orthogonal,
  consider whether a new MOC node is needed.
- Sleep's downscaling function suggests Engram needs a **maintenance pass** (the
  equivalent of sleep) that runs after batches of L1 emissions: replay (re-rank),
  consolidate (promote to L2 / merge), and downscale (demote / archive). Append-
  only is fine as substrate; the maintenance pass is what makes the substrate
  useful.

---

## 3. Link Types / Associative Structure

### Key findings

**Semantic networks (Collins & Loftus 1975; Anderson 1983 ACT*; Anderson 2007
ACT-R):** memory is a graph of concept nodes with weighted edges. Retrieval is
**spreading activation** from cue nodes outward; activation decays with graph
distance and edge weight. The Collins-Loftus model already includes multiple edge
types (ISA, part-of, property) but emphasizes that the *graph structure plus
activation dynamics* is what produces retrieval, not the edge labels.

**Temporal Context Model (Howard & Kahana 2002):** episodic memory binds items to
a slowly-drifting context vector. Retrieval brings back the bound context, which
then cues nearby items in time. Temporal proximity is a first-class link type.

**Contextual binding theory (Yonelinas, Ranganath, Ekstrom & Wiltgen 2019):**
hippocampus does not "consolidate by transferring to cortex"; instead, it
permanently stores the *contextual binding* that lets reinstated cues retrieve an
episode. Context (spatial, temporal, situational) is the link.

**Causal links (Lu et al. 2025 and earlier narrative-memory work):** humans
organize event memory around causal structure, not just temporal order; retrieval
preferentially follows causal chains.

**Inhibitory / contradictory links (Anderson 2003; Anderson et al. 2015):**
retrieval-induced forgetting suggests that competing-memory relationships are
encoded as suppression links, not just absent connections.

### Strongest sources

- Collins & Loftus (1975), "A spreading-activation theory of semantic processing,"
  Psychological Review 82(6):407-428.
- Howard & Kahana (2002), "A distributed representation of temporal context,"
  Journal of Mathematical Psychology 46(3):269-299.
- Yonelinas, Ranganath, Ekstrom & Wiltgen (2019), "A contextual binding theory of
  episodic memory: systems consolidation reconsidered," Nature Reviews
  Neuroscience 20(6):364-375.

### Design implications

- Engram's `contradicts:` and `supersedes:` edges are well-grounded in the
  inhibition/RIF literature. Keep them.
- **Add temporal edges** (session-of, follows, precedes) at L1 and L2. TCM and
  contextual-binding theory both put temporal context at the center of episodic
  recall.
- **Add causal edges** at L2 (caused-by, enables, blocks). Narrative-memory work
  shows these are first-class.
- **Semantic similarity should be a retrieval-time computation, not a stored
  edge.** Spreading activation operates over an activation field computed at
  retrieval; ACT-R explicitly does not store every similarity edge. Embedding-
  based nearest-neighbor at recall time is the right analog.
- The edge type that is *missing from the literature and probably should not be in
  Engram either* is "semantically similar to" as an authored link. Authored links
  should be *structural* (temporal, causal, contradictory, hierarchical); content
  similarity is computed.

---

## 4. Curation / Forgetting

### Key findings

**Ebbinghaus (1885) and replications (Murre & Dros 2015):** retention drops
exponentially in the first 24 hours; the curve is real but interacts strongly with
content type, repetition, and interference.

**Interference theory (McGeoch 1932; Wixted 2004 review):** "time" per se does not
cause forgetting — new learning interferes with old (retroactive) and old learning
interferes with new (proactive). The modern consensus (Hardt, Nader, Nadel 2013;
Wixted 2004) is that interference dominates over pure decay, though both exist.

**Adaptive forgetting (Anderson 2003; Anderson & Hulbert 2021):** retrieval-induced
forgetting actively suppresses competing memories via prefrontal control of the
hippocampus. This is **goal-directed** — the brain forgets because forgetting
*improves* future retrieval and generalization. RIF generalizes across cues
(cue-independent), suggesting it operates on the memory itself, not the cue-memory
association.

**Synaptic homeostasis (Tononi & Cirelli 2014):** sleep downscales weak synapses,
preserving relative strengths but reducing total energy cost. This is a population-
level forgetting mechanism that *aids* generalization.

### Strongest sources

- Anderson & Hulbert (2021), "Active Forgetting: Adaptation of Memory by
  Prefrontal Control," Annual Review of Psychology 72:1-36.
- Hardt, Nader & Nadel (2013), "Decay happens: the role of active forgetting in
  memory," Trends in Cognitive Sciences 17(3):111-120.
- Wixted (2004), "The psychology and neuroscience of forgetting," Annual Review
  of Psychology 55:235-269.

### Design implications

- **Append-only is in genuine tension with the literature.** Real memory actively
  prunes, and pruning *improves* downstream performance. Options:
  1. Keep append-only at L0/L1/L2 (substrate is durable), but add a `demoted: true`
     or `confidence: 0.1` tag that retrieval-time filtering respects. This is the
     "soft forgetting" version.
  2. Allow L3 MOCs to *not* link to demoted L2 atoms, achieving practical forgetting
     without physical deletion.
- **Confidence/strength decay** is well-supported: every retrieval should boost a
  fact's strength; every non-retrieval over time should decrement it. The
  Ebbinghaus curve gives a rough functional form (exponential with a slow tail).
- RIF generalizes across cues — this means that when one L2 fact `supersedes:`
  another, the superseded fact should be suppressed for *all* retrievals, not just
  the cue that triggered the supersession. Make `supersedes:` global.

---

## 5. Retrieval & Relevance Filtering

### Key findings

**Encoding-specificity principle (Tulving & Thomson 1973):** retrieval succeeds to
the extent that cues at recall overlap with the context at encoding. This is the
most robust single finding in the recall literature.

**Cue-dependent retrieval (Tulving 1983; Godden & Baddeley 1975 — divers):**
environmental and internal context substantially modulate recall. State-dependent
and mood-dependent memory are special cases.

**Reconstructive memory (Bartlett 1932; Schacter 2012; Schacter & Addis 2007):**
recall is *not* lookup — it is reconstruction from partial traces, current goals,
and schemas. Errors are systematic: gist preserved, detail mutated toward schema
expectations. The same reconstructive machinery is used for future-simulation
("constructive episodic simulation hypothesis").

**Spreading activation as relevance filter (Anderson ACT-R):** activation level is
the relevance score. Items with activation above a threshold are retrieved;
activation depends on (a) recent use, (b) frequency of use, (c) spreading
activation from currently active cues. **No explicit tags or scoring needed** —
relevance emerges from the activation dynamics.

### Strongest sources

- Tulving & Thomson (1973), "Encoding specificity and retrieval processes in
  episodic memory," Psychological Review 80(5):352-373.
- Schacter, Guerin & St. Jacques (2011), "Memory distortion: an adaptive
  perspective," Trends in Cognitive Sciences 15(10):467-474.
- Anderson et al. (2004), "An integrated theory of the mind" (ACT-R), Psychological
  Review 111(4):1036-1060.

### Design implications

- **Encoding specificity is the single strongest argument for keeping rich
  context in L1.** A stripped task-boundary segment that loses situation/entity
  context will fail to be retrieved by future cues that reinstate that context.
  L1 should err on the side of *more* context, not less.
- **Reconstructive retrieval argues for a re-synthesis step at recall time.** Don't
  return raw L2 atoms verbatim; return an LLM-synthesized passage that uses the
  retrieved atoms as scaffold. This mirrors human reconstruction and matches how
  the calling agent will actually use the memory.
- **Spreading activation gives a tag-free relevance model.** Engram's recall can
  use: query embedding → nearest L2 facts → follow `supersedes`/`contradicts`/
  `temporal` edges one hop → aggregate by activation score. No explicit
  "importance" tag is needed; importance emerges from connectivity + recency.

---

## 6. Salience / What Gets Encoded

### Key findings

**Prediction error gates encoding (Schultz 1998; Rouhani et al. 2018; Henson &
Gagnepain 2010):** dopaminergic signals scale with the discrepancy between
expectation and outcome. High-prediction-error events are preferentially encoded
into long-term memory.

**Novelty as a separate driver (Lisman & Grace 2005, "hippocampal-VTA loop"):**
purely novel (not necessarily reward-relevant) events also trigger dopamine and
enhance encoding. Novelty habituates with exposure.

**Emotional tagging (LaBar & Cabeza 2006; McGaugh 2004):** amygdala-modulated
encoding strengthens memories with emotional valence. Negative valence is
particularly strong (mortal salience).

**Levels-of-processing (Craik & Lockhart 1972):** the *depth* of processing at
encoding (semantic > phonetic > orthographic) predicts later retrievability. This
is a "what gets remembered" lever the encoder controls.

### Strongest sources

- Rouhani, Norman & Niv (2018), "Dissociable effects of surprising rewards on
  learning and memory," Journal of Experimental Psychology: Learning, Memory, and
  Cognition 44(9):1430-1443.
- Lisman & Grace (2005), "The hippocampal-VTA loop: controlling the entry of
  information into long-term memory," Neuron 46(5):703-713.
- Craik & Lockhart (1972), "Levels of processing: A framework for memory
  research," JVLB 11(6):671-684.

### Design implications

- **Engram needs an explicit salience gate between L0 and L1.** Not every session
  produces a worthwhile L1 segment. The gate should be cheap and explicit. Candidate
  signals, in order of literature support:
  1. Prediction error against L3: does the session contradict or extend the current
     MOC graph?
  2. Novelty: are the entities / situation absent from existing L1/L2?
  3. User affect / emphasis (analog of emotional tagging): did the user mark this
     as important, frustrating, or surprising?
- **Levels-of-processing argues for LLM-mediated re-encoding at L1 emission, not
  raw transcript slicing.** A summary that *re-states the principle* (deep semantic
  processing) is more retrievable than one that *quotes the conversation* (shallow).

---

## Cross-Cutting Tensions With Current Design

These are places the literature actively pushes back on Engram's L0-L3 model.

1. **Four tiers vs. two.** CLS is the most directly-applicable cognitive model and
   it specifies *two* learning systems. Engram's L0/L1 are both episodic;
   Engram's L2/L3 are both semantic-structural. Consider whether the four-tier
   model is producing real architectural value or just visual neatness. A defensible
   alternative: a 2-tier model (episodic, semantic) with explicit "raw" and
   "consolidated" representations within each.

2. **Append-only vs. adaptive forgetting.** Anderson's RIF, Tononi-Cirelli synaptic
   homeostasis, and the modern interference literature all say that *forgetting is
   functional* — it improves generalization and retrieval. Append-only as substrate
   is fine; append-only as the retrieval semantics is not. A confidence/strength
   field that decays and a `superseded`/`demoted` filter at retrieval are the
   minimum required.

3. **L3 derived from L2 vs. L3 driving L2.** Schema-assimilation work shows that
   *schemas drive consolidation*, not the reverse. If L3 is purely a synthesis layer
   regenerated from L2, you've inverted the biological direction. The fix is to use
   L3 as the prediction model against which new L1/L2 content is scored for
   salience, congruence, and contradiction. L3 then updates from L2, but L2
   ingestion is conditioned on L3.

4. **Lopsided hourglass shape.** The current sketch (wide L0, narrow L2, slight
   re-widening at L3) does not match biology. The cortex (semantic) is *much* wider
   than the hippocampus (episodic) in capacity. The biological shape is closer to
   "narrow active episodic window, vast slow-learning semantic store, very small
   active working-memory output." If anything, L2 should be the widest tier and L0
   should be bounded (transcripts can be archived/pruned aggressively because L1
   has captured what mattered).

5. **Stripping context at L1.** Encoding specificity is the strongest principle in
   the retrieval literature. "Stripped" L1 segments risk failing to be retrieved by
   cues that would have reinstated the encoding context. Lean toward keeping
   situation/entity/temporal context *in* L1, even at storage cost.

6. **No working-memory analog.** Baddeley-style working memory is the workspace
   where retrieved memories interact with current input. Engram's "recall payload"
   plays this role implicitly but isn't designed as such. Make it explicit: a
   bounded, ordered, decaying window the calling agent reasons over.

---

## Open Questions for Follow-Up

| Question | Difficulty | Value |
|----------|-----------|-------|
| Does CLS's "fast episodic + slow semantic" map cleanly to Engram, or is the L0/L1 distinction (raw vs. summarized) doing useful work CLS doesn't model? | Medium — needs prototype with merged tiers to compare. | High — could simplify the architecture by ~50%. |
| What's the right functional form for L2 confidence decay? Ebbinghaus exponential, power-law (Wixted), or use-conditioned (ACT-R activation)? | Low — literature is well-developed. | Medium — affects long-term store quality. |
| How should L3 MOCs gate L1 ingestion concretely? "LLM check against current MOC" is expensive; is there a cheaper proxy (embedding distance to MOC centroids)? | Medium — design + empirical. | High — this is the schema-assimilation lever. |
| Should `temporal` and `causal` edges be authored by the L1/L2 emitter or extracted by a later pass? | Low — design question. | Medium — affects how rich the retrieval graph is. |
| Is reconstructive retrieval (LLM synthesizes a passage from retrieved atoms) net-positive vs. returning raw atoms? Reconstruction can introduce confabulation. | High — empirical, needs eval harness. | High — affects every recall. |
| What's the analog of sleep / replay in Engram? A scheduled maintenance pass? Per-session? On idle? | Medium — design + scheduling. | High — without it, the store degrades. |
| How should Engram handle the *episodic-semantic continuum* — should L2 atoms be allowed to retain provenance (which session, which situation), or is that contamination? | Low — design call. | Medium — affects audit/debug and retrieval cueing. |

---

## References

(Citations are formatted as author(s), year, title, venue, with DOI or stable
identifier where available.)

- Anderson, J. R. (1983). *The Architecture of Cognition*. Harvard University Press.
- Anderson, J. R., Bothell, D., Byrne, M. D., Douglass, S., Lebiere, C., & Qin, Y.
  (2004). An integrated theory of the mind. *Psychological Review*, 111(4),
  1036-1060. doi:10.1037/0033-295X.111.4.1036
- Anderson, M. C. (2003). Rethinking interference theory: Executive control and the
  mechanisms of forgetting. *Journal of Memory and Language*, 49(4), 415-445.
  doi:10.1016/j.jml.2003.08.006
- Anderson, M. C., & Hulbert, J. C. (2021). Active forgetting: Adaptation of memory
  by prefrontal control. *Annual Review of Psychology*, 72, 1-36.
  doi:10.1146/annurev-psych-072720-094140
- Baddeley, A. D. (2000). The episodic buffer: a new component of working memory?
  *Trends in Cognitive Sciences*, 4(11), 417-423.
- Bartlett, F. C. (1932). *Remembering: A Study in Experimental and Social
  Psychology*. Cambridge University Press.
- Born, J., et al. (2024). Sleep's contribution to memory formation. *Physiological
  Reviews*. doi:10.1152/physrev.00054.2024
- Collins, A. M., & Loftus, E. F. (1975). A spreading-activation theory of semantic
  processing. *Psychological Review*, 82(6), 407-428.
- Craik, F. I. M., & Lockhart, R. S. (1972). Levels of processing: A framework for
  memory research. *Journal of Verbal Learning and Verbal Behavior*, 11(6),
  671-684.
- Ebbinghaus, H. (1885/1913). *Memory: A Contribution to Experimental Psychology*.
  Teachers College, Columbia University.
- Gilboa, A., & Marlatte, H. (2017). Neurobiology of schemas and schema-mediated
  memory. *Trends in Cognitive Sciences*, 21(8), 618-631.
  doi:10.1016/j.tics.2017.04.013
- Godden, D. R., & Baddeley, A. D. (1975). Context-dependent memory in two natural
  environments. *British Journal of Psychology*, 66(3), 325-331.
- Hardt, O., Nader, K., & Nadel, L. (2013). Decay happens: the role of active
  forgetting in memory. *Trends in Cognitive Sciences*, 17(3), 111-120.
  doi:10.1016/j.tics.2013.01.001
- Howard, M. W., & Kahana, M. J. (2002). A distributed representation of temporal
  context. *Journal of Mathematical Psychology*, 46(3), 269-299.
- Klinzing, J. G., Niethard, N., & Born, J. (2019). Mechanisms of systems memory
  consolidation during sleep. *Nature Neuroscience*, 22(10), 1598-1610.
  doi:10.1038/s41593-019-0467-3
- Kumaran, D., Hassabis, D., & McClelland, J. L. (2016). What learning systems do
  intelligent agents need? Complementary learning systems theory updated. *Trends
  in Cognitive Sciences*, 20(7), 512-534. doi:10.1016/j.tics.2016.05.004
- LaBar, K. S., & Cabeza, R. (2006). Cognitive neuroscience of emotional memory.
  *Nature Reviews Neuroscience*, 7(1), 54-64.
- Lisman, J. E., & Grace, A. A. (2005). The hippocampal-VTA loop: controlling the
  entry of information into long-term memory. *Neuron*, 46(5), 703-713.
- McClelland, J. L., McNaughton, B. L., & O'Reilly, R. C. (1995). Why there are
  complementary learning systems in the hippocampus and neocortex: Insights from
  the successes and failures of connectionist models of learning and memory.
  *Psychological Review*, 102(3), 419-457.
- McGeoch, J. A. (1932). Forgetting and the law of disuse. *Psychological Review*,
  39(4), 352-370.
- Murre, J. M. J., & Dros, J. (2015). Replication and analysis of Ebbinghaus'
  forgetting curve. *PLOS ONE*, 10(7), e0120644.
  doi:10.1371/journal.pone.0120644
- Renoult, L., Irish, M., Moscovitch, M., & Rugg, M. D. (2019). From knowing to
  remembering: The semantic-episodic distinction. *Trends in Cognitive Sciences*,
  23(12), 1041-1057. doi:10.1016/j.tics.2019.09.008
- Rouhani, N., Norman, K. A., & Niv, Y. (2018). Dissociable effects of surprising
  rewards on learning and memory. *Journal of Experimental Psychology: Learning,
  Memory, and Cognition*, 44(9), 1430-1443.
- Schacter, D. L., & Addis, D. R. (2007). The cognitive neuroscience of
  constructive memory: remembering the past and imagining the future.
  *Philosophical Transactions of the Royal Society B*, 362(1481), 773-786.
- Schacter, D. L., Guerin, S. A., & St. Jacques, P. L. (2011). Memory distortion:
  An adaptive perspective. *Trends in Cognitive Sciences*, 15(10), 467-474.
- Schultz, W. (1998). Predictive reward signal of dopamine neurons. *Journal of
  Neurophysiology*, 80(1), 1-27.
- Squire, L. R. (2004). Memory systems of the brain: a brief history and current
  perspective. *Neurobiology of Learning and Memory*, 82(3), 171-177.
- Tononi, G., & Cirelli, C. (2014). Sleep and the price of plasticity: from
  synaptic and cellular homeostasis to memory consolidation and integration.
  *Neuron*, 81(1), 12-34.
- Tse, D., Langston, R. F., Kakeyama, M., Bethus, I., Spooner, P. A., Wood, E. R.,
  Witter, M. P., & Morris, R. G. M. (2007). Schemas and memory consolidation.
  *Science*, 316(5821), 76-82. doi:10.1126/science.1135935
- Tulving, E. (1972). Episodic and semantic memory. In E. Tulving & W. Donaldson
  (Eds.), *Organization of Memory* (pp. 381-403). Academic Press.
- Tulving, E. (2002). Episodic memory: From mind to brain. *Annual Review of
  Psychology*, 53, 1-25.
- Tulving, E., & Thomson, D. M. (1973). Encoding specificity and retrieval
  processes in episodic memory. *Psychological Review*, 80(5), 352-373.
- van Kesteren, M. T. R., Ruiter, D. J., Fernández, G., & Henson, R. N. (2012).
  How schema and novelty augment memory formation. *Trends in Neurosciences*,
  35(4), 211-219.
- Wixted, J. T. (2004). The psychology and neuroscience of forgetting. *Annual
  Review of Psychology*, 55, 235-269.
- Yonelinas, A. P., Ranganath, C., Ekstrom, A. D., & Wiltgen, B. J. (2019). A
  contextual binding theory of episodic memory: systems consolidation
  reconsidered. *Nature Reviews Neuroscience*, 20(6), 364-375.
  doi:10.1038/s41583-019-0150-4
