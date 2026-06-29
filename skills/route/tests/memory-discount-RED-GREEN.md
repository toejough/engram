# Route memory-discount rule — RED/GREEN (2026-06-28)

Tests the "Memory discounts the tier" rule (model-agnostic). Method: 6 memory-backed units (their answer is a
recallable convention/decision/lesson) that *surface*-look like they need a strong tier. Each unit routed by a
fresh sonnet agent in two arms: **control** (rubric without the rule) vs **treatment** (rubric + the rule).
Output = the tier picked (cheap/mid/deep).

| arm | tiers picked | mean tier | noticed memory-backed |
|---|---|--:|---|
| control (no rule) | mid, mid, cheap, cheap, mid, mid | 1.67 | 6/6 |
| treatment (+ rule) | cheap ×6 | 1.00 | 6/6 |

**RED:** the control recognized all 6 as memory-backed but still parked **4/6 at the mid tier** — it noticed
recallability but didn't act on it (over-provisioning). **GREEN:** the rule made the router discount one tier
(mid→cheap; cheap stays cheap), confirming the wording changes behavior.

**Honesty bound (note 122):** the measured #7 evidence is the deep→mid boundary (mid+memory == deep+memory).
The treatment dropped mid→cheap (one tier lower than measured) — so the rule applies the discount optimistically,
capped at one tier + floored at cheap, protected by the standing "upgrade if the cheaper fails" rule. Raw:
`memory-discount-microtest.json`.
