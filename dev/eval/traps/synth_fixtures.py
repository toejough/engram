"""Synthesis-layer C6 fixtures (per the 2026-06-23-synthesis-layer.md spec).

Each fixture seeds a vault with notes A and B such that:
  - cross-domain (different functional areas);
  - the task's answer C = compose(A, B), and C is stated in NO note;
  - A and B are IDIOSYNCRATIC invented facts, so cold opus cannot know them and the
    warm agent cannot recite C from a single note — it must COMBINE A and B.

Three relational kinds: compositional join, transitive chain, analogical transfer.

build(kind, dst) writes the notes (real engram fact notes + sidecars).
FIXTURES[kind] also carries the task prompt + the emergent conclusion C (for the judge).
"""
import os
import shutil
import subprocess

# (slug, subject, predicate, object)
_NOTES = {
    # compositional join: security(A) x ops(B), joined by the "-7" account suffix.
    "synth-join": [
        ("vault7-secret-reader", "the vault-7 service account",
         "is the only identity allowed to read", "the production secrets store"),
        ("drill-suspends-7", "the Tuesday failover drill",
         "temporarily suspends", "every service account whose name ends in -7"),
    ],
    # transitive chain: flag(A) -> flow -> table(B); ask the A->C consequence.
    "synth-chain": [
        ("qmflag-gates-checkout", "the qm-rollout feature flag",
         "gates", "the new checkout flow"),
        ("checkout-writes-ledger", "the new checkout flow",
         "writes its records to", "the ledger-v2 table"),
    ],
    # analogical transfer: an idiosyncratic fix in domain1(A); an analogous bug in domain2(B).
    "synth-transfer": [
        ("payments-dedupe-fix", "the payments service double-charge bug",
         "was fixed by", "rejecting any request whose trace-id was already in the seen-traces bloom filter"),
        ("notifications-dup-bug", "the notifications service",
         "currently double-sends emails", "whenever a delivery is retried"),
    ],
}

FIXTURES = {
    "synth-join": {
        "task": ("During the Tuesday failover drill, production can't read its secrets and nobody knows "
                 "why. Using ONLY your recalled memory, explain the cause and what to do."),
        # C = vault-7 (the sole prod-secrets reader) gets suspended by the drill -> secrets unreadable.
        "C": ("The Tuesday drill suspends accounts ending in -7, which includes vault-7 — the ONLY account "
              "that can read the production secrets store — so prod loses secret access during the drill. "
              "Fix: exempt vault-7 from the drill (or pre-stage the secrets)."),
        "bridge_slugs": ["vault7-secret-reader", "drill-suspends-7"],
    },
    "synth-chain": {
        "task": ("A teammate is about to enable the qm-rollout feature flag and asks which database table "
                 "will start receiving new writes as a result. Using ONLY your recalled memory, answer."),
        # C = ledger-v2 (qm-rollout -> new checkout flow -> writes ledger-v2).
        "C": "The ledger-v2 table (qm-rollout enables the new checkout flow, which writes to ledger-v2).",
        "bridge_slugs": ["qmflag-gates-checkout", "checkout-writes-ledger"],
    },
    "synth-transfer": {
        "task": ("The notifications service is double-sending emails on retries and we need a fix. Using "
                 "ONLY your recalled memory of how we solved a similar problem elsewhere, propose the fix."),
        # C = transfer the payments fix: reject retries whose trace-id is already in a seen-traces bloom filter.
        "C": ("Reuse the payments-service approach: dedupe by checking each delivery's trace-id against a "
              "seen-traces bloom filter and skipping any retry already present."),
        "bridge_slugs": ["payments-dedupe-fix", "notifications-dup-bug"],
    },
}


def _learn(vault, slug, subj, pred, obj):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(
        ["engram", "learn", "fact", "--slug", slug, "--position", "top",
         "--source", f"synth fixture: {slug}",
         "--situation", f"{subj} {pred} {obj}",
         "--subject", subj, "--predicate", pred, "--object", obj],
        env=env, check=True, capture_output=True, text=True)


def build(kind, dst):
    if kind not in _NOTES:
        raise ValueError(kind)
    if os.path.exists(dst):
        shutil.rmtree(dst)
    os.makedirs(dst)
    for slug, subj, pred, obj in _NOTES[kind]:
        _learn(dst, slug, subj, pred, obj)
    missing = [n for n in os.listdir(dst)
               if n.endswith(".md") and not os.path.exists(os.path.join(dst, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"fixture {kind}: missing sidecars for {missing}")
    return sorted(n for n in os.listdir(dst) if n.endswith(".md"))
