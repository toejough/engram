"""Compounding-eval fixtures — 2-level EMERGENT synthesis ladders (corrected design).

The earlier chain version (zx1 triggers zx2 ...) was a degenerate linked-list traversal: the terminal
was a stored literal, so it tested lookup-and-follow, not synthesis. This version ladders GENUINE
emergent conclusions, matching the synthesis taxonomy (compositional join / transitive composition /
analogical transfer):

  Level 1:  A + B  ->  C   (emergent; C is stated in NO note)
  Level 2:  C + D  ->  E   (emergent; E needs the level-1 C AND a new fact D)

Arms (both warm, same skill; differ only in per-trial vault contents):
  no-persist : vault = {A, B, D}        -> must RE-DERIVE C from A,B, then compose with D to reach E.
  persist    : vault = {A, B, D, C*}    -> C* = the ORACLE stored level-1 conclusion; only does C+D->E.
               (Oracle stored C = best case for persistence; this RED is an upper bound.)

All facts are idiosyncratic invented tokens so cold opus can't shortcut and the agent must combine.
The metric judges whether the answer reaches E (a conclusion, not a token) — use the LLM judge.
"""
import os
import shutil
import subprocess

# Each type: raw notes A,B,D (for no-persist); stored_C (added for persist, the emergent level-1
# conclusion); the level-2 task; and E (the emergent level-2 conclusion, for the judge).
TYPES = {
    "join": {
        # compositional join: security(A) x ops(B) joined by the vault-7 / "-7" key.
        "notes": [
            ("vault7-reader", "the vault-7 service account",
             "is the only identity allowed to read", "the production secrets store"),
            ("drill-suspends", "the Tuesday failover drill",
             "suspends", "every service account whose name ends in -7"),
            ("backup-needs-secrets", "the nightly compliance backup runs every Tuesday at 02:00 and",
             "must read", "the production secrets store"),
        ],
        "stored_C": ("during-drill-no-secrets", "during the Tuesday failover drill",
                     "the production secrets store becomes unreadable",
                     "because vault-7, its only reader, is suspended"),
        "task": ("Is the Tuesday 02:00 nightly compliance backup at risk of failing? Using ONLY your "
                 "recalled memory, explain precisely why or why not."),
        "E": ("YES — the Tuesday 02:00 compliance backup is at risk: the Tuesday failover drill suspends "
              "vault-7 (the only account that can read the production secrets store), so prod secrets are "
              "unreadable during the drill window, and the backup must read them at 02:00 — so it fails if "
              "it overlaps the drill."),
    },
    "transitive": {
        # transitive composition: flag->pipeline->table (C), then table cap (D) -> overflow (E).
        "notes": [
            ("qm-routes-v2", "enabling the qm-rollout flag",
             "routes all checkout traffic through", "the v2 pipeline"),
            ("v2-writes-shard-d", "the v2 pipeline",
             "writes every transaction to", "the ledger-shard-D table"),
            ("shard-d-cap", "the ledger-shard-D table",
             "sheds load above", "500 writes per second"),
        ],
        "stored_C": ("qm-writes-shard-d", "enabling the qm-rollout flag",
                     "causes transaction writes to", "the ledger-shard-D table"),
        "task": ("We are about to enable the qm-rollout flag during peak checkout traffic. Using ONLY your "
                 "recalled memory, what specific failure should we watch for?"),
        "E": ("Enabling qm-rollout routes checkout through v2, which writes every transaction to "
              "ledger-shard-D; at peak traffic those writes can exceed ledger-shard-D's 500/sec cap, so "
              "ledger-shard-D will shed load — watch for dropped/shed transaction writes."),
    },
    "analogical": {
        # analogical transfer: a pattern established in A,B (-> C), transferred to a new problem D (-> E).
        "notes": [
            ("payments-bloom", "the payments service double-charge bug",
             "was fixed by", "rejecting any request whose trace-id was already in the seen-traces bloom filter"),
            ("webhooks-bloom", "the webhooks service later reused",
             "the same trace-id seen-traces bloom-filter dedupe", "to stop duplicate webhook deliveries"),
            ("billing-dup", "the new billing-export service",
             "currently double-sends", "whenever an export is retried"),
        ],
        "stored_C": ("bloom-is-standard", "the trace-id seen-traces bloom-filter dedupe",
                     "is our standard idempotency pattern", "for stopping duplicate sends across services"),
        "task": ("The billing-export service is double-sending on retries and we need a fix. Using ONLY "
                 "your recalled memory of how we solved this elsewhere, propose the fix."),
        "E": ("Apply our standard trace-id seen-traces bloom-filter dedupe (the same pattern used for "
              "payments and webhooks) to billing-export: check each export's trace-id against the bloom "
              "filter and skip any retry already present."),
    },
}


def _learn(vault, slug, subj, pred, obj):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(
        ["engram", "learn", "fact", "--slug", slug, "--position", "top",
         "--source", f"compound fixture: {slug}",
         "--situation", f"{subj} {pred} {obj}",
         "--subject", subj, "--predicate", pred, "--object", obj],
        env=env, check=True, capture_output=True, text=True)


def build(stype, persist, dst, scatter=0):
    if stype not in TYPES:
        raise ValueError(stype)
    spec = TYPES[stype]
    if os.path.exists(dst):
        shutil.rmtree(dst)
    os.makedirs(dst)
    for slug, subj, pred, obj in spec["notes"]:           # A, B, D
        _learn(dst, slug, subj, pred, obj)
    if persist:                                            # + oracle stored level-1 C
        slug, subj, pred, obj = spec["stored_C"]
        _learn(dst, slug, subj, pred, obj)
    for j in range(scatter):                              # distractor pad
        _learn(dst, f"distract-{j:03d}", f"qd{j}", "is unrelated to", f"qe{j}")
    missing = [n for n in os.listdir(dst)
               if n.endswith(".md") and not os.path.exists(os.path.join(dst, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"compound fixture {stype}: missing sidecars for {missing}")
    return spec
