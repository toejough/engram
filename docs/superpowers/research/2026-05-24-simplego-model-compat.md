# Hugot simplego embedding-model compatibility (verified shortlist)

Date: 2026-05-24. Follow-up to `2026-05-23-embedder-options.md`, which picked Hugot's `simplego` backend as the offline fallback. Goal here: figure out which embedding model that path can actually run.

## TL;DR

- **Only `sentence-transformers/all-MiniLM-L6-v2` is verified end-to-end** in Hugot's pure-Go path. It is the only model exercised by Hugot's go-backend feature-extraction test (`tests/utils.go`, `KnightsAnalytics_all-MiniLM-L6-v2`).
- **Every standard BERT-based sentence encoder in the candidate list ("BERT-clone")** — MiniLM-L12-v2, BGE-small/base-en-v1.5, Snowflake-arctic-embed-xs/s/m, GTE-small/base, E5-small/base-v2 — *should* work: same ONNX inputs (`input_ids`/`token_type_ids`/`attention_mask`), absolute positional embeddings, op surface already covered by `onnx-gomlx`. None has a Hugot test run against it. **"Should-work, untested."**
- **Nomic v1.5 is unknown.** RoPE op exists in onnx-gomlx (added for Gemma), but nomic's ONNX export is a custom `nomic_bert` architecture and the upstream maintainer was explicitly unsure whether it would convert ("there may be operations not implemented yet").
- **Jina v2 is unsupported.** `position_embedding_type: alibi` — no ALiBi case exists in onnx-gomlx's op dispatch.
- **The honest recommendation is: stay on MiniLM-L6-v2 until a one-evening verification spike clears MiniLM-L12-v2 or arctic-embed-xs.** Don't ship a swap without that.

## Per-model verdict

| # | Model | Arch | Status | Notes |
|---|---|---|---|---|
| 1 | sentence-transformers/all-MiniLM-L6-v2 | BERT, absolute pos, 256 max ctx | **verified** | Hugot `tests/utils.go::FeatureExtractionPipeline` runs it. onnx-gomlx README: "has been working perfectly." |
| 2 | sentence-transformers/all-MiniLM-L12-v2 | BERT, absolute pos, 512 max ctx | **should-work** | Identical input/op surface to L6. ONNX export ships `onnx/model.onnx`. Safest upgrade target. |
| 3 | BAAI/bge-small-en-v1.5 | BERT, absolute pos, 512 max ctx | **should-work** | onnx-gomlx maintainer said "most components are there" but didn't confirm; shipped a safetensors example, not an ONNX-via-Hugot test (onnx-gomlx issue #80). |
| 4 | BAAI/bge-base-en-v1.5 | BERT, absolute pos, 512 max ctx | **should-work** | Same arch as bge-small, just deeper. 110MB. |
| 5 | Snowflake/snowflake-arctic-embed-xs | BERT, absolute pos, 512 max ctx | **should-work** | onnx-gomlx PR #51 explicitly added `ReduceL2` op "for Snowflake Arctic embedding models." 6 layers / 384-dim. Operators present; not end-to-end tested through Hugot. |
| 6 | Snowflake/snowflake-arctic-embed-s | BERT, absolute pos, 512 max ctx | **should-work** | Same arch family. |
| 7 | Snowflake/snowflake-arctic-embed-m | BERT, absolute pos, 512 max ctx | **should-work** | Same arch family, 110MB. |
| 8 | thenlper/gte-small | BERT, absolute pos, 512 max ctx | **should-work** | Standard BERT. ONNX export at `onnx/model.onnx`. |
| 9 | thenlper/gte-base | BERT, absolute pos, 512 max ctx | **should-work** | Standard BERT. |
| 10 | intfloat/e5-small-v2 | BERT, absolute pos, 512 max ctx | **should-work, packaging gotcha** | Only ships `onnx/model_O4.onnx` and `model_qint8_avx512_vnni.onnx` — no plain `model.onnx`. Hugot's `FeatureExtractionConfig.OnnxFilename` must be set to `model_O4.onnx`. O4 graph optimisations are runtime-agnostic; quantized variant is AVX-512 specific. |
| 11 | intfloat/e5-base-v2 | BERT, absolute pos, 512 max ctx | **should-work** | Has `onnx/model.onnx`. |
| 12 | nomic-ai/nomic-embed-text-v1.5 | nomic_bert (custom), RoPE, 2048 max ctx | **unknown** | Custom `NomicBertModel` arch with `rotary_emb_fraction: 1.0`, `flash_attn`, fused ops. onnx-gomlx has `RotaryEmbedding` op (added for Gemma), but nomic's ONNX export may use fused/custom ops not yet covered. The maintainer's stance was "not sure if it would work" for BGE — that caution applies harder to a fully custom arch. To prove: download `onnx/model.onnx`, point Hugot at it, observe whether parsing/CallGraph succeeds. |
| 13 | jinaai/jina-embeddings-v2-base-en | BERT-shell + ALiBi positional, 8192 max ctx | **unsupported** | `position_embedding_type: alibi`. No ALiBi case in onnx-gomlx's op dispatch (verified by reading `internal/onnxgomlx/graph.go`). Will fail to convert. |

## Architecture compatibility notes

The simplego backend works by parsing an ONNX file (`gomlx/onnx-gomlx`) and executing it on the GoMLX `simplego` runtime. Two filter layers determine model compatibility:

1. **ONNX input names** (Hugot's `backends/model_gomlx.go::createInputTensorsGoMLX`): the model is required to take only `input_ids`, `token_type_ids`, `attention_mask`, and/or `position_ids`. Anything else returns `unknown input meta name`. Every BERT-clone in the candidate list uses exactly these. Nomic v1.5 may or may not export only these — needs verification.

2. **ONNX op coverage** (`gomlx/onnx-gomlx/internal/onnxgomlx/graph.go`). The op dispatch enumerates: `Abs, Add, And, ArgMax, ArgMin, AveragePool, BatchNormalization, Cast, Ceil, Clip, Concat, Constant, ConstantOfShape, Conv, Cos, CumSum, DequantizeLinear, Div, DynamicQuantizeLinear, Einsum, Equal, Erf, Exp, Expand, FastGelu, Flatten, Floor, Gather, GatherElements, GatherND, Gelu, Gemm, GlobalAveragePool, Greater, GroupQueryAttention, Identity, If, LayerNormalization, Less, Log, LSTM, MatMul, MatMulInteger, Max, MaxPool, Min, Mod, Mul, MultiHeadAttention, Neg, NonZero, Not, Or, Pad, Pow, QLinearMatMul, QuantizeLinear, Range, Reciprocal, ReduceL2, ReduceMax, ReduceMean, ReduceMin, ReduceProd, ReduceSum, Relu, Reshape, Resize, RotaryEmbedding, ScatterND, Shape, Sigmoid, SimplifiedLayerNormalization, Sin, Size, Slice, Softmax, Split, Sqrt, Squeeze, Sub, Tanh, Tile, TopK, Transpose, Trilu, Unsqueeze, Where, Xor`. Notably **absent: ALiBi** (rules out Jina v2). Notably **present**: `RotaryEmbedding` (Gemma path; potentially nomic), `SimplifiedLayerNormalization` (Gemma RMSNorm), `ReduceL2` (Snowflake Arctic).

3. **Sequence-length config**: Hugot v0.4.2 (2025-06) added `max_position_embeddings` enforcement in the tokenizer (CHANGELOG, issue #73). All 512-ctx models will respect 512 tokens; MiniLM-L6 stays at its trained 256.

4. **Prompt-prefix model contracts** (writer-visible): E5 family requires `"query: "` / `"passage: "` prefixes on inputs; nomic v1.5 requires `"search_query: "` / `"search_document: "`; BGE recommends a short query instruction ("Represent this sentence for searching relevant passages:"); MiniLM, GTE, Snowflake Arctic, Jina embed raw text. Engram's writer would need a per-model prefix policy if engram ever ships anything but MiniLM as default.

Sources: `tests/utils.go`, `backends/model_gomlx.go`, `gomlx/onnx-gomlx/internal/onnxgomlx/graph.go`, `gomlx/onnx-gomlx` issue #51 (Snowflake Arctic ops added), `gomlx/onnx-gomlx` issue #80 (BGE — closed as "use safetensors path"), Hugot `CHANGELOG.md`.

## Recommendation

**Pick MiniLM-L6-v2. It's the only one any human has driven through Hugot+simplego end-to-end.** Engram's offline-fallback design accepts a quality hit for "no API, no daemon, single binary," and the only verified path delivers exactly that.

If you want a better-than-MiniLM offline path, do a one-evening verification spike before committing. The two cleanest candidates to spike:

- **MiniLM-L12-v2** — safest. Same arch class as L6, same tokenizer, same input surface. ~1–2 MTEB points up, ~50% larger model (33MB). If anything fails here, the L6 path is also broken.
- **Snowflake-arctic-embed-xs** — best quality bet at the same size. 22MB, MTEB ~62 (vs L6 ~57), 6-layer BERT, and onnx-gomlx already added the one op (`ReduceL2`) that distinguishes Arctic from vanilla BERT.

Verification recipe: download `onnx/model.onnx` and the matching tokenizer to a local dir, point `hugot.FeatureExtractionConfig{ModelPath: ..., OnnxFilename: "model.onnx"}` at it from a `NewGoSession`, embed a handful of sentences, compare cosine similarities against the same sentences embedded by `sentence-transformers` in Python (10-line script). If the cosines agree to ~3 decimals on a dozen pairs, the model is verified.

For engram's specific shape — 200–500-word LLM-voiced notes, 10⁴-scale vault, single-binary commitment, no API at recall — the marginal value of moving from MiniLM-L6 (~57 MTEB avg) to Arctic-xs (~62) is real but not dramatic on personal-vault retrieval (the retrieval gap on short personal prose is typically narrower than on the general MTEB average). The dominant decision driver is verification cost, not quality delta. **Recommended path: ship MiniLM-L6 as the offline default in v2; file an issue to spike Arctic-xs once the rest of the offline pipeline is wired.**

Do not ship nomic or jina as the offline default. Nomic is unverified custom-arch territory; Jina is provably unsupported (no ALiBi op).

## What would change the answer

Concretely:
- A Hugot integration test (`hugot/tests/go/`) that loads BGE-small or Arctic-xs and asserts embedding cosines vs a reference would move that model to **verified** and flip the recommendation.
- An onnx-gomlx commit adding ALiBi op coverage would lift Jina v2 from **unsupported** to **should-work**.
- A nomic-specific success report on the gomlx Slack or in onnx-gomlx issues would lift Nomic v1.5 to **should-work**. (Today's evidence is silence.)
