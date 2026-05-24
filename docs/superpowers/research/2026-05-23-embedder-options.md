# Embedder options for engram

Date: 2026-05-23. Companion to the tiered-memory design (specs/2026-05-14-tiered-memory-design.md), which currently leans Voyage `voyage-3-large` + `chromem-go`. Goal: decide which embedder engram should ship as the default in v2 and which alternates to support behind opt-in flags.

Constraints (assume these throughout):

- Pure-Go binary today. CGO is on the table only if the case is strong.
- Reads (`engram recall`) must be offline — no network at recall time.
- Writes (`engram update` / `engram learn` paths) may hit a network API.
- Single-user CLI; no required services. Opt-in services are fine.
- Vault content: 200–500-word English prose (principles, feedback, episode summaries). Not raw code.
- Scale: hundreds → 10⁴ today, design target 10⁵.

---

## TL;DR

- **The current spec's default (`voyage-3-large`) is the wrong SKU.** Voyage shipped voyage-4 in 2025; `voyage-4-large` is $0.12/M (vs voyage-3-large at $0.18/M), with the first 200M tokens free per account. voyage-3 family has no free tier. Same Matryoshka dims (256/512/1024/2048), same 32K context. Switch the default to `voyage-4-large`. At engram's scale (10⁵ notes × ~400 tokens ≈ 40M tokens) write-time embeddings cost **$0** indefinitely.
- **Pure-Go embedded inference is now actually feasible** via Hugot + GoMLX's `simplego` backend. Runs `all-MiniLM-L6-v2` in-process, no CGO, no native libs, single binary. The catch is ~9 MTEB-average points lower than voyage-4-large (~58 vs ~67) and small-batch only. This is a real pareto point for "no API key, no Ollama, no CGO." It's the right v2 fallback when API is unreachable.
- **fastembed-go is dead.** Archived January 2026, last release was January 2024. The ONNX-runtime-with-CGO path has narrowed to `yalue/onnxruntime_go` directly — and that requires shipping a native `.so`/`.dylib` per platform, which kills the single-binary story. Don't go here.
- **Ollama + `nomic-embed-text` is a viable opt-in** for users who want offline + better-than-MiniLM quality and don't mind running a daemon. nomic-embed-text-v1.5 is MTEB ~62 with native Matryoshka down to 64 dims. Setup cost is "user installs Ollama, runs `ollama pull nomic-embed-text`" — not zero, but well-understood.
- **Mycelium's stack tells you almost nothing useful.** It uses local fastembed (the archived Python package, not the Go fork) into pgvector inside AgensGraph. Multi-agent, hub-and-spoke, Postgres-required — orthogonal to engram's "single binary, filesystem markdown" design.
- **pgvector is a vector store, not an embedder, and requires Postgres.** Out of scope for engram's single-binary v2; revisit only if engram ever grows a server mode.

---

## Per-option assessment

Each option leads with the engram verdict, then the numbers.

### 1. Voyage AI

**Verdict: default. Use `voyage-4-large` at 1024 dims with Matryoshka headroom.** Best retrieval quality on the market in 2026 for general English prose; first 200M tokens free; engram lives inside the free tier forever at design scale.

Lineup as of 2026-05:

| Model | Default dims (Matryoshka opts) | Context | Price /1M | Free tier | MTEB notes |
|---|---|---|---|---|---|
| `voyage-4-large` | 1024 (256/512/2048) | 32K | $0.12 | 200M tokens | Current SOTA retrieval; quantization int8/binary supported |
| `voyage-4` | 1024 (256/512/2048) | 32K | $0.06 | 200M tokens | Within ~1pt of voyage-4-large on most benches |
| `voyage-4-lite` | 1024 (256/512/2048) | 32K | $0.02 | 200M tokens | Latency/cost-optimized |
| `voyage-3-large` *(prev gen)* | 1024 (256/512/2048) | 32K | $0.18 | none | The current spec target. Cited at MTEB avg 65.1 in some sources, 67.1 in others (discrepancy noted in refs). Superseded. |
| `voyage-3.5-lite` *(prev gen)* | 1024 (256/512/2048) | 32K | $0.02 | none | Acceptable budget option but voyage-4-lite dominates. |
| `voyage-3-lite` *(prev gen)* | 512 fixed (no Matryoshka) | 32K | retired-tier | none | Pareto-dominated by voyage-3.5-lite and voyage-4-lite. |

Voyage's own announcement claims voyage-3-large outperformed OpenAI's text-embedding-3-large by 9.74% across 100 datasets; voyage-4-large adds a further ~4pt edge on the announcement's internal benchmarks. April 2026 third-party leaderboards put voyage-3-large in the top tier of API embedders for retrieval specifically (NV-Embed-v2 wins overall MTEB average but is a hosted research model, not a product).

**Integration in pure-Go:** Roll-your-own HTTP. There's no first-party Go SDK. The Voyage embeddings endpoint is `POST /v1/embeddings` with `{model, input, input_type}`. ~30 lines of Go using `net/http`. Engram already plans the HTTP-at-write-time pattern, so no new infrastructure.

**Gotchas:**

- `input_type` distinction matters. Voyage models behave noticeably better when documents are embedded with `"document"` and queries with `"query"`. Engram needs to pipe the difference through.
- Quotas: free tier is per-account, not per-key. If a user runs engram on two machines, they share the 200M pool. At 40M-tokens-at-design-scale this is a non-issue for years.
- Rate limits: 1M tokens/request for lite models, 320K for standard. Engram's per-write payloads are tiny; not load-bearing.
- API stability has been good. voyage-3 series wasn't deprecated when voyage-4 shipped — they coexist. Treat the model ID as a versioned constant in engram's spec.

### 2. OpenAI

**Verdict: alternate, not default.** Cheaper than Voyage on absolute price ($0.13 vs $0.12 for the flagships is a wash) but loses on retrieval quality at every public benchmark I found. Use it only as a fallback when the user has an OpenAI key handy and doesn't want a second vendor.

| Model | Default dims (Matryoshka opts) | Price /1M | MTEB avg | MTEB retrieval |
|---|---|---|---|---|
| `text-embedding-3-large` | 3072 (256–3072) | $0.13 | 64.6 | strong, but trails voyage-3-large |
| `text-embedding-3-small` | 1536 (256–1536) | $0.02 | 62.26 | the cheap-defensible-choice baseline |
| `text-embedding-ada-002` | 1536 fixed | retired | ~61 | **retired** — don't use |

Matryoshka on `-3-large` is the real feature here: you can drop to 1024 dims and lose only ~1–2 MTEB points, then store half the bytes per vector. (Voyage offers the same dim flexibility, so this is not a unique advantage anymore.) `text-embedding-3-small` at 1536 dims, $0.02/M, MTEB 62.26 is the historical defensible default for budget RAG — it's still fine, just not best.

**Integration:** First-party Go is unofficial but well-trodden (`sashabaranov/go-openai`); raw HTTP is also trivial. Pure-Go either way.

**Gotchas:**

- ada-002 is retired. Don't generate guidance suggesting it.
- The `-3-large` 3072 default is wasteful for engram — half-MB per vector at 10⁵ scale is ~50GB just for sidecars. Truncate to 1024 (Matryoshka API: `dimensions=1024`).

### 3. ONNX local (fastembed-go / yalue/onnxruntime_go)

**Verdict: don't go here.** The pure-Go binary constraint is incompatible with ONNX Runtime, which is a C++ library. Every path requires either CGO + bundled `.so`/`.dylib` per platform, or shipping a separate runtime the user has to install. fastembed-go (the most popular wrapper) was archived January 2026 — that's the canary.

State of the ecosystem:

- **fastembed-go (Anush008):** Archived 2026-01-15, read-only, last release v1.0.0 from January 2024. Supports BGE-small/base/large, MiniLM-L6-v2 via ONNX Runtime. Downloads models on first use. The archival is a strong signal: this niche has thinned.
- **yalue/onnxruntime_go:** Active (648 stars, 2026 commits). Generic ONNX bindings. Requires the `onnxruntime` shared library to be present on the target machine. The repo ships pre-built libraries for AMD64 Windows / ARM64 Linux / ARM64 Darwin only. Loads the lib at runtime — you cannot statically link it.

What you'd pay if you swallowed CGO:

- Model sizes: MiniLM-L6-v2 ~22 MB, BGE-small ~33 MB, BGE-base ~110 MB, BGE-large ~335 MB, nomic-embed ~274 MB, mxbai-embed-large ~335 MB.
- RAM at inference: 200 MB – 1.5 GB depending on model.
- ONNX runtime native lib: another ~10 MB binary to ship per platform.

**Quality reference:** BGE-large-en-v1.5 sits at MTEB avg 64.23 / retrieval 54.29. That's 1–3 points behind voyage-4-large on average and ~13 points behind on the retrieval subset specifically. BGE-M3 is competitive on retrieval (cited as best self-hosted quality/cost balance) but is multilingual and 568M params, larger again.

**Why it's a "don't":** Engram's whole identity is "single binary, no services, no native deps." Trading that for offline-capable inference made sense in 2024 when API embedders were sketchy. In 2026 voyage-4-large is so cheap that engram lives in the free tier forever, and the pure-Go option below has caught up enough to be the genuine fallback. The CGO path is pareto-dominated by both ends.

### 4. Ollama / llama-server HTTP

**Verdict: opt-in, not default.** A real option for users who want offline + better-than-pure-Go-MiniLM quality and accept running Ollama. Engram stays pure-Go (HTTP client to `localhost:11434`). Quality is reasonable but model warmup, daemon lifecycle, and "user must install Ollama" are friction.

Common embedding models served via Ollama:

| Model | Size | Dims | Context | MTEB avg | Notes |
|---|---|---|---|---|---|
| `nomic-embed-text` (v1.5) | 274 MB | 768 (Matryoshka 64/128/256/512/768) | 8192 | 62.28 | Apache 2.0; **requires task prefix** (`search_document:` for docs, `search_query:` for queries). Skipping this hurts quality. |
| `mxbai-embed-large` | 335 MB | 1024 | 512 | ~64 | Apache 2.0; smaller context than nomic. |
| `bge-m3` | 568 MB | 1024 | 8192 | competitive on retrieval | Multilingual; overkill for engram's English-only vault. |

**Integration:** `POST http://localhost:11434/api/embeddings` with `{model, prompt}`. Pure-Go HTTP. Easy.

**Gotchas:**

- **Model warmup.** Ollama unloads idle models. First request after idle takes 1–5 seconds. Engram's writes can absorb this; engram's recalls (online lookup of a query embedding) cannot — but recall is offline-by-spec, so this only bites if you also use Ollama for the query embedding at recall time. That's the actual reason "embed at write, store sidecar" matters.
- **Daemon lifecycle.** User has to keep `ollama serve` running. Engram should probe at startup and degrade clearly.
- **Task prefixes for nomic.** Engram's writer code has to know to prefix `search_document: ` and the query code has to prefix `search_query: `. Easy to get wrong silently — it doesn't error, it just retrieves worse.
- **The nomic v1.5 context of 8192 tokens** is fine for engram's 200–500-word notes (~1000 tokens) but if engram ever embeds whole transcripts you'd want it.

### 5. Pure-Go embedders that actually exist (Hugot + GoMLX)

**Verdict: this is the surprise. Use as the offline fallback / no-API-key path.** Hugot's `simplego` backend (powered by GoMLX) runs `all-MiniLM-L6-v2` in-process today with no CGO, no native libraries, no daemon. It's slower than ONNX and tops out at small models, but for a vault of 10⁴–10⁵ notes that you embed on write, the speed cost is invisible.

What exists:

- **Hugot (knights-analytics/hugot)**: 606 stars, 53 releases, latest v0.7.3 from May 2026, actively maintained. Wraps ONNX models behind a uniform Go API; has both ORT (CGO) and `simplego` (pure-Go) backends. The pure-Go backend explicitly supports the `featureExtraction` pipeline and calls out `all-MiniLM-L6-v2` as the canonical small-model target.
- **GoMLX (gomlx/gomlx)**: 1.4k stars, 57 releases, latest v0.27.3 from April 2026. The `simplego` backend has no C/C++ deps. Supports SIMD (AVX2/AVX512) and fused operations.

**Quality:** all-MiniLM-L6-v2 is MTEB average ~56–58 (older numbers cite 56.26, newer ones a touch higher). That's roughly 9 points below voyage-4-large on the same average. Retrieval-specifically the gap is smaller on short personal notes than on a generic web corpus, but you should expect "good enough for similarity, not best-in-class."

**Constraints from the docs:**

- "Designed for simpler workloads, environments that disallow cgo, and smaller models such as all-MiniLM-L6-v2."
- "Best with small batches of roughly 32 inputs per call."
- "If you have performance requirements, please move to a C backend such as XLA or ORT."

For engram this is fine: writes happen one note at a time or in small task-boundary batches. Recall sends one query embedding. Throughput isn't the binding constraint.

**Packaging:** Models are HuggingFace downloads (you cache a ~22 MB ONNX file in `~/.cache/engram` on first use) or you bundle the model in the binary if you don't mind the size. No native libraries either way. This is the option that genuinely keeps engram a single binary.

**The other "pure-Go" projects** (`sugarme/transformer`, `andrewyang17/transformer`, `MufidJamaluddin/transformer`) are either pre-1.0 unmaintained or thin sugarme/gotch derivatives that need libtorch. None of them are production-grade today. Hugot+GoMLX is the only one I'd ship.

### 6. Mycelium

**Verdict: not relevant.** Different problem, different constraints. Worth a paragraph below for the record; nothing to borrow.

### 7. pgvector

**Verdict: not in scope.** pgvector is a Postgres extension that adds vector columns and indexes (HNSW, IVFFlat). It is not an embedder — it stores vectors someone else generated. It requires running Postgres. Engram's design promise is "single binary, no services," so pgvector is out by definition. The natural place to revisit it is if engram ever grows a server mode that already requires a database; until then, `chromem-go`'s flat index covers the design target.

---

## Mycelium read

(https://github.com/mycelium-io/mycelium)

**What they do.** Multi-agent coordination layer. 3+ peer agents negotiating in shared "rooms" that are backed by markdown on disk and synced via PostgreSQL. CognitiveEngine state machine drives structured negotiation.

**Stack.**

- **Embedder:** local `fastembed` (the Python package, not the archived Go fork) producing 384-dim vectors.
- **Vector store:** `pgvector` running inside **AgensGraph** (Postgres 16 fork with graph extensions).
- **Backend:** FastAPI + Docker.
- **Deployment:** Hub-and-spoke. By default everything runs locally on one machine; teams elect one machine as the hub.

**Constraints they designed under:**

- Multi-agent semantics (negotiation, consensus, deduplication of work). Hierarchy-free.
- Need graph operations (CognitiveEngine traversal, room relationships) — hence AgensGraph rather than vanilla Postgres.
- Local-first but multi-machine-capable. The hub-and-spoke topology assumes one person elects a host.

**What's relevant for engram:** Almost nothing. The shared design element is "memories are markdown files on disk." That's an idea engram already has (the vault). Everything else — Postgres dependency, Docker, graph DB, negotiation state machines, multi-agent rooms — is orthogonal to "single-user CLI, single binary, no services."

**What's instructive negatively:** Mycelium chose `fastembed` (a Python package) because they already ship a Docker-managed Python backend. Once you have a backend service, local embeddings are cheap. Engram has the opposite constraint: it explicitly refuses to ship a backend. That's why the embedder question is harder for engram than for Mycelium — and why "embed at write via API" is the cleanest answer.

---

## Comparison table

| Option | Quality (MTEB avg) | Offline read | CGO | Setup for user | Cost at engram's scale | Engram fit |
|---|---|---|---|---|---|---|
| **voyage-4-large (API)** | ~67 (top tier) | sidecars yes; writes online | no | get API key | **$0** (40M < 200M free) | **default** |
| voyage-4 (API) | ~66 | same | no | get API key | $0 in free tier | budget default if quality tolerated |
| voyage-4-lite (API) | ~64 | same | no | get API key | $0 in free tier | budget alt |
| voyage-3-large (API) | ~65–67 (disputed) | same | no | get API key | $7/year at 40M | superseded |
| OpenAI text-embedding-3-large | 64.6 | same | no | get API key | $5/year at 40M | alternate |
| OpenAI text-embedding-3-small | 62.26 | same | no | get API key | $0.80/year at 40M | budget alt |
| **Hugot + MiniLM (pure-Go)** | ~57 | yes | **no** | nothing | $0 | **offline fallback** |
| Hugot + larger ONNX (CGO) | up to ~64 | yes | yes | install runtime | $0 | not worth CGO cost |
| fastembed-go | ~64 with BGE-large | yes | yes | install runtime | $0 | **archived — skip** |
| **Ollama + nomic-embed** | 62.28 | yes | no | install + run Ollama | $0 | opt-in alt |
| Ollama + mxbai-embed-large | ~64 | yes | no | install + run Ollama | $0 | opt-in alt |
| pgvector | n/a (store, not embedder) | yes if local PG | no | run Postgres | $0 | **out of scope** |

**Pareto-dominated options to remove from consideration:**

- `voyage-3-large` — superseded by voyage-4-large (better quality, $0.06/M cheaper, 200M free).
- `voyage-3-lite` — superseded by voyage-3.5-lite (same price, Matryoshka) and voyage-4-lite (free tier).
- `text-embedding-ada-002` — retired by OpenAI.
- `fastembed-go` — archived January 2026; ONNX-runtime-via-CGO is the same cost path through `yalue/onnxruntime_go` if you really want it.
- ONNX-with-CGO generally — dominated by Hugot+MiniLM (no CGO, single binary) at the offline-quality end and by voyage-4-large (better quality, free) at the online-quality end.

**The non-dominated frontier:** voyage-4-large (best quality), Hugot+MiniLM (best for "no API, no services, single binary"), Ollama+nomic-embed (middle ground: offline + better-than-MiniLM if user accepts a daemon).

---

## Recommendation

Ship two paths, default to one, design for swap.

1. **Default: Voyage `voyage-4-large`, 1024 dims, Matryoshka headroom to 2048.** Engram's design scale (10⁵ notes × ~400 tokens) is 40M tokens — comfortably inside the 200M-token free tier. Best retrieval quality available. Standard HTTP call at write time, sidecar `.vec.json`. Tag every sidecar with `embedding_model_id: "voyage-4-large@1024"` so a future model swap forces a re-embed migration rather than silent drift. This replaces the current spec's `voyage-3-large` choice directly — same shape, better numbers, cheaper, free at scale.

2. **Offline fallback: Hugot + `all-MiniLM-L6-v2` via the GoMLX pure-Go backend.** Triggered by `--offline` flag or by retry exhaustion against Voyage. Lower quality (~57 MTEB avg) but truly zero-setup, no API key, no daemon, no native libraries — the binary just works. Embed the model in the binary (~22 MB) or download to `~/.cache/engram` on first use. Tag sidecars `embedding_model_id: "minilm-l6-v2@384"` so the migration story still applies.

3. **Opt-in alternate: Ollama + `nomic-embed-text`.** Document it as `engram config set embedder=ollama` for users who want offline + better-than-MiniLM quality and already run Ollama. Handles the gap between (1) "user wants no API call ever" and (2) "user accepts a quality hit." Don't make it the default — the daemon-lifecycle and task-prefix gotchas push it out of zero-config.

**Tradeoffs being accepted:**

- Voyage as default means engram inherits Voyage's product lifecycle. If Voyage retires voyage-4 series someday, we re-cut. The current `embedding_model_id` tagging makes that survivable.
- MiniLM as fallback means the offline path is measurably worse. For a vault of 10⁴ personal notes, this is probably invisible at the top-3 results; for tail queries it'll show. We accept the quality cost in exchange for the "single binary, no setup" promise.
- We do not commit to a vector store change. `chromem-go` flat at 40ms/100K is fine for the design target, but the last release was September 2024 and the maintainer's roadmap (HNSW, IVFFlat) is unstarted. Flag for revisit before scale, not now.

**Strongest single takeaway:** **the spec's current default model is one generation behind and on the wrong pricing tier.** Switching from `voyage-3-large` to `voyage-4-large` is a same-shape change (same dims, same context, same API surface) that improves quality and moves engram into the free tier for the foreseeable future. Do that in the spec before anything else.

---

## References

- Voyage AI embeddings overview: https://docs.voyageai.com/docs/embeddings
- Voyage AI pricing (200M free tier on voyage-4 family): https://docs.voyageai.com/docs/pricing
- Voyage 3 Large announcement (claims 9.74% lift over OpenAI text-embedding-3-large on 100 datasets): https://blog.voyageai.com/2025/01/07/voyage-3-large/
- pecollective embedding comparison (voyage-3-large MTEB avg cited as 67.1 here vs 65.1 in some Hacker News and aggregator threads — discrepancy unresolved, doesn't change the decision): https://pecollective.com/tools/text-embedding-models-compared/
- TokenMix text-embedding-3-small spec (MTEB 62.26, Matryoshka 256–1536): https://tokenmix.ai/blog/text-embedding-3-small-developer-guide-2026
- MTEB leaderboard, April 2026 snapshot (NV-Embed-v2 / Gemini Embedding 001 lead overall; voyage-3-large and -3.1-large lead retrieval among API products): https://awesomeagents.ai/leaderboards/embedding-model-leaderboard-mteb-april-2026/
- nomic-embed-text-v1.5 model card (Matryoshka 64–768, MTEB 62.28, requires task prefixes): https://huggingface.co/nomic-ai/nomic-embed-text-v1.5
- Ollama nomic-embed-text page (REST endpoint, 274 MB, 8192 ctx): https://ollama.com/library/nomic-embed-text
- BGE-large-en-v1.5 model card (MTEB avg 64.23, retrieval 54.29, MIT): https://huggingface.co/BAAI/bge-large-en-v1.5
- fastembed-go repository (archived 2026-01-15): https://github.com/Anush008/fastembed-go
- onnxruntime_go (requires native onnxruntime library at runtime, no single-binary): https://github.com/yalue/onnxruntime_go
- Hugot (pure-Go backend via GoMLX, v0.7.3 May 2026): https://github.com/knights-analytics/hugot
- GoMLX (simplego pure-Go backend, v0.27.3 April 2026): https://github.com/gomlx/gomlx
- chromem-go (pure-Go flat index, last release September 2024): https://github.com/philippgille/chromem-go
- Mycelium (local fastembed Python + pgvector inside AgensGraph): https://github.com/mycelium-io/mycelium
