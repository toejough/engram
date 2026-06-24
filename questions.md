1. The synthesize-l2 path. so when querying, you're saying that the recency decay on chunks is not enabled? ifso, that's
   a mistake. I want _every_ query to consider recent events, more if there aren't any L2's that pass muster, less if
   there are a lot, but always some recent events.

2. I'm more worried about note capping being too limiting than being too open. What does the empirical curve look like
   for total N? can we always have 10 phrases with the top 100 matches and still be done in sub-second timing?

3. ranking over the whole vault is absurd. we should only nominate L2's from within the cluster.
4. this looks fine.

Other questions:

1. you said Phase A filters to L1+L2, and not L2. My understanding is all we have as notes now are L2's. Is that
   correct? If so, let's just start calling them notes. L1 & L3 language and code and documentation should be removed or
   moved to the design history sections only, and all L2 references should just be "notes".
2. Phase B: I want chunks to have recency applied to them when evaluating cosine matches & the taking up to the limit of
   the best ones - really old memories don't mean as much.
3. there's no mention of surfacing recent chunks as "here's what we've been up to lately". That should be happening. I
   want recent chunks surfaced to the caller so that they remmember what they did in the last 24 hours of working time
   (that is, if I leave and come back a month later, I want the LLM to remember what we were doing before I left, not
   just shrug and go "we haven't done anything in 24 hours of wall clock time")
4. what's with the phase D >= 3 notes? how about we just always say "the top 5"?
5. what's with the baseScore >= 0.5 limit? if the memory was used, it should be activated. If we're trying to cut out
   low-scoring, bad matches, we should do that before returning them to the agent. Top 100 matches per query, filtered
   to only those with a recency-biased score above 0.25, + the last 24 active-hours of chunks, for example.
