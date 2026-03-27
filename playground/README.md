# Scoring Algorithm Playground

Interactive tool for tuning engram's memory surfacing algorithm against a curated ground-truth corpus.

## What It Does

- Implements BM25 + spreading activation + quality signals + tier boost + cross-project penalty in JS
- Tests 2000 ground-truth queries (should-surface and should-not-surface) against 100 curated memories
- Sliders for every tunable weight, with live sensitivity graphs
- Auto-optimizer that finds optimal weights via coordinate ascent across configurable score/project priorities

## Files

- `scoring-playground.html` — self-contained HTML playground (~700KB, mostly embedded corpus)
- `corpus/playground-corpus.json` — the curated test corpus (kept separately for reproducibility)
- `README.md` — this file

## How To Recreate This Experiment

### Step 1: Curate the corpus

Extract all memories to a flat JSON for analysis:

```python
import os, tomllib, json
data_dir = os.path.expanduser("~/.claude/engram/data/memories")
memories = []
for f in sorted(os.listdir(data_dir)):
    if not f.endswith(".toml"): continue
    with open(os.path.join(data_dir, f), "rb") as fh:
        rec = tomllib.load(fh)
    memories.append({
        "file": f,
        "principle": rec.get("principle", ""),
        "title": rec.get("title", ""),
        "confidence": rec.get("confidence", ""),
        "generalizability": rec.get("generalizability", 0),
        "surfaced_count": rec.get("surfaced_count", 0),
        "followed_count": rec.get("followed_count", 0),
        "irrelevant_count": rec.get("irrelevant_count", 0),
    })
with open("/tmp/engram-memories.json", "w") as fh:
    json.dump(memories, fh, indent=2)
```

Then use an LLM (Opus recommended) to:
1. Identify 4 distinct project domains by reading principles/titles
2. For each domain, select 10 most important + 10 least important memories (judged by principle quality only, not scores)
3. Select 10 most important + 10 least important general memories (gen 4-5)
4. For each memory, generate: 5 user queries that SHOULD surface it, 5 LLM statements that SHOULD surface it, 5 user queries that should NOT, 5 LLM statements that should NOT

**Key decision:** Judge importance solely by reading the principle text. Don't look at surfaced_count, followed_count, or other tracking data for selection. This creates an unbiased ground truth.

Output format per memory:
```json
{
  "file": "filename.toml",
  "principle": "the principle text",
  "confidence": "A",
  "generalizability": 2,
  "should_surface_user": ["query1", ...],
  "should_surface_llm": ["statement1", ...],
  "should_not_surface_user": ["query1", ...],
  "should_not_surface_llm": ["statement1", ...]
}
```

### Step 2: Enrich with tracking data

For each memory in the corpus, read its actual TOML file and add:
- `surfaced_count`, `followed_count`, `contradicted_count`, `ignored_count`, `irrelevant_count`
- `last_surfaced_at`, `updated_at`, `created_at` (ISO timestamps)
- `keywords` (array of strings — also improves BM25 recall)

### Step 3: Compute inter-memory links

Run the same algorithms as `internal/graph/graph.go`:
- **Concept overlap**: Jaccard similarity of keywords, threshold 0.15
- **Content similarity**: BM25 of each memory against all others, threshold 0.05, weight normalized by dividing raw score by 5.0 (capped at 1.0)

Add `links` array to each memory: `[{"target": "other.toml", "weight": 0.85, "basis": "content_similarity"}]`

### Step 4: Build the playground HTML

Single self-contained HTML file with:

**Scoring pipeline (JS implementation):**
1. BM25 (Okapi, k1=1.2, b=0.75) — searchable text = file slug words + principle + keywords
2. Irrelevance penalty: `bm25 *= K/(K + irrelevant_count)`
3. Spreading activation: for each BM25 match, walk its links, accumulate `bm25 * linkWeight` on targets, normalize by linker count
4. GenFactor: cross-project penalty based on generalizability (same-project/general = 1.0, cross-project scales by gen level)
5. Quality: `wEff*effectiveness + wRec*recency + wFreq*frequency + wTier*tierBoost`
6. Combined: `(bm25 * genFactor + alpha * spreading) * (1 + quality)`
7. Filters: BM25 floor, prompt limit, cold-start budget

**UI components:**
- Project selector (drives both scoring context and query visibility)
- Sliders for every weight with current-value display
- Sensitivity graph (canvas): click a slider to sweep it and plot all 4 scores. Y-axis auto-scales to data min/max. Color-coded vertical markers at each score's maximum.
- Scorecard: surface accuracy, suppression accuracy, cross-project isolation, overall
- Query detail table: paginated, filterable, expandable rows showing full formula breakdown per memory
- Auto-optimizer: configurable score weights + project weights, async coordinate ascent with progress display

**Performance optimizations:**
- Precompute `memoryIdxByFile` map (avoid O(N) indexOf in spreading loop)
- Cache BM25 scores per unique query text (Map)
- Cache spreading scores per unique query text (Map)
- Skip invisible queries in sweep (filter by project early)
- Clear caches when corpus changes, not when params change

**Key design decisions:**
- Project selector drives both scoring AND query visibility (don't decouple these)
- Y-axis auto-scales to data range (don't waste space on 0-100% when data is 75-95%)
- Show the actual formula in expanded rows: `(BM25×GenFactor + α×Spreading) × (1+Quality) = Combined`
- Auto-optimizer uses coarse steps (~8-10 per slider) for speed, user fine-tunes with sensitivity graph

### Step 5: Iterate

The playground is a tool for exploration, not a final answer. The workflow:
1. Select a project, click "Run All Queries" to see baseline
2. Click sliders to see sensitivity graphs — which params matter?
3. Use auto-optimizer to find a good starting point
4. Fine-tune by examining failing queries in the detail table
5. Test across all projects to ensure changes don't regress others

## Findings From First Run (2026-03-26)

- Tier boost (wTier) sweet spot around 0.5 for engram project
- Gen penalty at 1.0 is optimal (default was already right)
- Alpha (spreading) has no effect because no memories have links in production (graph builder not wired — filed as #390)
- Frequency/irrelevance/effectiveness weights only matter when tracking data varies across corpus — with fresh memories they're uniform multipliers
- Cross-project isolation jumps from 40% to 96% with gen penalty enabled
