# Design Rationale: Memory Types and Structures

## Why two types: feedback and fact

Memories split into **feedback** and **fact** because they answer different questions and surface at different moments.

- **Feedback** answers *"how should I behave here?"* — a correction or pattern you want applied or avoided. Only useful if you can tell *when* to apply it and *what* to do differently.
- **Fact** answers *"what's true about this project or environment?"* — declarative knowledge the agent would otherwise have to rediscover.

Mixing them under a single schema forces every memory to either lose structure ("freeform blob") or pretend to be one type. Keeping them separate means recall can weight them differently and `/learn` / `/remember` can guide you through the right fields for each kind.

This split mirrors a long-standing distinction in cognitive psychology. Tulving (1972)¹ distinguished **semantic memory** (general facts about the world) from **episodic memory** (personal experiences); Graf & Schacter (1985)² formalized the **explicit/implicit** divide; Squire (2004)³ consolidated these into the modern taxonomy of long-term memory systems (declarative, with semantic and episodic subtypes, vs. non-declarative, including procedural memory). The distinction also appears in computational cognitive architectures: Anderson's **ACT-R**⁴ treats declarative facts (chunks) and procedural knowledge (productions) as separate memory stores with different retrieval and learning rules.

Engram's **fact** type maps to semantic memory — declarative knowledge about a project or environment. Engram's **feedback** type is closer to an *explicit behavioral rule*: a procedure stated as a rule the agent can read and apply, rather than truly implicit procedural memory (which is unconscious skill acquired through practice and not directly readable).

> ¹ Tulving, E. (1972). "Episodic and semantic memory." In *Organization of Memory*, ed. Tulving & Donaldson. Academic Press.
> ² Graf, P. & Schacter, D. L. (1985). "Implicit and explicit memory for new associations in normal and amnesic subjects." *Journal of Experimental Psychology: Learning, Memory, and Cognition*, 11(3), 501–518.
> ³ [Squire, L. R. (2004). "Memory systems of the brain: A brief history and current perspective." *Neurobiology of Learning and Memory*, 82(3), 171–177.](http://whoville.ucsd.edu/PDFs/384_Squire_%20NeurobiolLearnMem2004.pdf)
> ⁴ [Anderson, J. R., Bothell, D., Byrne, M. D., Douglass, S., Lebiere, C., & Qin, Y. (2004). "An integrated theory of the mind." *Psychological Review*, 111(4).](https://act-r.psy.cmu.edu/about/) See the Carnegie Mellon ACT-R project for a current overview of the declarative/procedural split.

## Why SBIA for feedback?

SBIA = **S**ituation, **B**ehavior, **I**mpact, **A**ction.

- **Situation** — the task the agent would be doing *before* this lesson is known (e.g. "writing async Go tests"). This is what the recall query matches against. If it describes the diagnosed problem instead ("after a race condition in test X"), the memory never surfaces for a fresh attempt.
- **Behavior** — what was actually done that needs changing.
- **Impact** — the consequence. Without impact, the agent has no reason to prefer the new action over the old one.
- **Action** — what to do instead.

SBIA is the minimum structure that makes a behavioral correction generalize. Without Situation you can't retrieve it; without Impact the agent rationalizes around it; without Action it isn't actionable.

SBIA builds on the **SBI model** (Situation-Behavior-Impact™), a feedback framework developed by the [Center for Creative Leadership](https://www.ccl.org/articles/leading-effectively-articles/closing-the-gap-between-intent-vs-impact-sbii/)⁵ and canonicalized in Weitzel (2000)⁶. The **A**ction extension has been applied in medical education — notably in analyzing narrative feedback from clinical preceptors in Entrustable Professional Activities (EPAs) e-portfolio systems — where structured action steps are needed alongside the observational triad (Hsu et al., 2021⁷). SBI's three-part structure was designed for live human conversations: pin the moment, describe the observable behavior, name its impact. That's enough in person because the receiver can ask "what should I do differently?" A memory file has no such feedback channel, so engram uses the SBIA form with Action written down at capture time. Engram didn't invent the "SBI + Action" shape — it adopts an existing research-grade framework for the same reason medical educators did: written feedback without an action step isn't actionable.

> ⁵ [Center for Creative Leadership. "Use SBI to Understand Intent vs. Impact."](https://www.ccl.org/articles/leading-effectively-articles/closing-the-gap-between-intent-vs-impact-sbii/)
> ⁶ Weitzel, S. R. (2000). *Feedback That Works: How to Build and Deliver Your Message*. Center for Creative Leadership.
> ⁷ [Hsu, T.-C. et al. (2021). "A Study to Analyze Narrative Feedback Record of an Emergency Department." *International Journal of Environmental Research and Public Health* 18(12): 6265.](https://pmc.ncbi.nlm.nih.gov/articles/PMC8238687/) — uses the SBIA framework to evaluate clinical preceptor feedback in EPA-based e-portfolios.
> ⁸ [Mindtools. "The Situation-Behavior-Impact™ Feedback Tool."](https://www.mindtools.com/ay86376/the-situation-behavior-impact-feedback-tool/)

## Why subject/predicate/object for facts?

Facts use SPO because declarative knowledge is naturally a triple: *some entity* has *some relationship* to *some value*. "engram uses targ", "the reorder-decls linter requires alphabetical test functions", "the auto-memory path derives from the git main repo root".

The triple form forces you to name the entity explicitly, which keeps facts from drifting into freeform prose that's hard to search and impossible to update. It also makes duplicate detection tractable — two facts with the same subject and predicate are candidates for merging.

Situation still matters for facts — it's *when* the fact is relevant, not *what* the fact says. A fact with no situation surfaces for every query; a fact with a sharp situation surfaces only when it matters.

This shape is the **RDF triple** — the atomic data unit in the W3C Resource Description Framework⁹, introduced in RDF 1.0 (Lassila & Swick, 1999)¹⁰ and refined as RDF 1.1 (2014)¹¹. Triples are also the foundation of Berners-Lee, Hendler & Lassila's (2001) vision of the Semantic Web¹². A triple makes a claim: the relationship indicated by the *predicate* holds between the *subject* and the *object*. Engram doesn't build an RDF graph, but it borrows the shape because the same constraint — you have to name the entity explicitly to say anything about it — keeps facts retrievable rather than drifting into freeform paragraphs.

> ⁹ [Semantic triple — Wikipedia](https://en.wikipedia.org/wiki/Semantic_triple) for an overview of how triples are used in knowledge graphs.
> ¹⁰ [Lassila, O. & Swick, R. R., eds. (1999). *Resource Description Framework (RDF) Model and Syntax Specification*. W3C Recommendation.](https://www.w3.org/TR/1999/REC-rdf-syntax-19990222/)
> ¹¹ [W3C (2014). *RDF 1.1 Concepts and Abstract Syntax*. W3C Recommendation.](https://www.w3.org/TR/rdf11-concepts/)
> ¹² Berners-Lee, T., Hendler, J. & Lassila, O. (2001). "The Semantic Web." *Scientific American*, 284(5), 34–43.
