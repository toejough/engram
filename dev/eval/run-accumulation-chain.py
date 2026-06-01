#!/usr/bin/env python3
"""Accumulation-chain orchestrator: 3 layers x 3 app-orders x 3 stages.

Each (layer, order) is an independent chain with its own accumulating vault.
Within a chain the 3 stages run STRICTLY sequentially (the vault accumulates:
stage 0 cold -> learns; stage 1 recalls stage-0 memory + learns; stage 2 recalls
both, no learn). The 3 cyclic orders put each app in each position once, so each
app is measured cold / +1-prior / +2-prior. Chains run sequentially here for cfg
safety (they share CLAUDE_CONFIG_DIR); ~27 headless builds total.

Usage: python3 dev/eval/run-accumulation-chain.py [LAYER ...] [--orders N]
       (default: all layers, all 3 orders)
"""
import subprocess, os, sys

REPO = "/Users/joe/repos/personal/engram"
STAGE = REPO + "/dev/eval/run-chain-stage.sh"
ORDERS = [
    ["todo", "bookmarks", "contacts"],
    ["bookmarks", "contacts", "todo"],
    ["contacts", "todo", "bookmarks"],
]
LAYERS_ARG = [a for a in sys.argv[1:] if a in ("L1", "L2", "L3")]
LAYERS = LAYERS_ARG or ["L1", "L2", "L3"]
NORDERS = 3
if "--orders" in sys.argv:
    NORDERS = int(sys.argv[sys.argv.index("--orders") + 1])

fails = []
for layer in LAYERS:
    for oi, order in enumerate(ORDERS[:NORDERS], 1):
        chain = "/tmp/chain-%s-o%d" % (layer, oi)
        vault = chain + "/vault"
        subprocess.run(["rm", "-rf", chain])
        os.makedirs(vault, exist_ok=True)
        # Refresh the cfg credential from the Keychain at each chain start so a
        # token TTL can't kill a multi-hour run partway through.
        subprocess.run(["bash", "-c",
            'security find-generic-password -s "Claude Code-credentials" -w '
            '> %s/dev/eval/.layer-run/cfg/.credentials.json && '
            'chmod 600 %s/dev/eval/.layer-run/cfg/.credentials.json' % (REPO, REPO)])
        for s, app in enumerate(order):
            learn = "no" if s == 2 else "yes"
            print("======== %s o%d s%d %s (learn=%s) ========" % (layer, oi, s, app, learn), flush=True)
            r = subprocess.run(["bash", STAGE, layer, app, vault, str(s), learn])
            if r.returncode != 0:
                fails.append("%s-o%d-s%d-%s" % (layer, oi, s, app))
                print("  !! FAILED %s o%d s%d %s — continuing chain" % (layer, oi, s, app), flush=True)
print("### CHAIN COMPLETE ### failures: %s" % (fails or "none"), flush=True)
