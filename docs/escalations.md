# Escalations

Review each escalation and update the **Status** field:
- `pending` - Not yet reviewed
- `resolved` - Add your answer in **Notes**
- `deferred` - Create an issue for later
- `issue` - Create an issue with your description in **Notes**

---

## ESC-001

**Category:** requirement
**Context:** TASK-052 memory query implementation uses mock embeddings instead of ONNX
**Question:** The acceptance criterion requires 'Uses local ONNX model for embeddings (no API calls)' but the current implementation uses hash-based mock embeddings. Options: (1) Download/bundle ONNX model (90+ MB) and implement real inference, (2) Update acceptance criteria to allow mock embeddings for initial release, (3) Use a lighter-weight embedding approach (e.g., BM25 or simpler vectors). Which approach should we take?

**Status:** resolved
**Notes:** Decision: Use e5-small ONNX model (~130MB, 384 dims, 16ms latency). Better accuracy than MiniLM-L6-v2 with same speed tier. Will download model on first use and implement real ONNX inference.

---

