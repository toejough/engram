#!/usr/bin/env python3
"""Smoke mini-chain: notes -> links -> feeds (blended, sonnet), to validate
learn->recall fidelity and the beta accumulation signal end-to-end, and to
measure real per-cell cost. notes teaches alpha; links recalls alpha + teaches
beta; feeds is built cold / +notes / +notes+links and we compare round-1 beta.
"""
import subprocess, os, shutil
CUM = "/Users/joe/repos/personal/engram/dev/eval/cumulative"
SMOKE = "/tmp/smoke"
VAULT = SMOKE + "/vault"
CFG_WARM = "/tmp/todo-coldwarm/cfg-warm"
CFG_COLD = "/tmp/todo-coldwarm/cfg-cold"

def refresh(cfg):
    subprocess.run(["bash", "-c",
        f'security find-generic-password -s "Claude Code-credentials" -w > {cfg}/.credentials.json '
        f'&& chmod 600 {cfg}/.credentials.json'])

def cell(app, regime, vin, vout, cfg, spec, tag, learn):
    wd = f"{SMOKE}/ws-{tag}"; out = f"{SMOKE}/chain-{tag}.json"
    subprocess.run(["rm", "-rf", wd])
    print(f"==== cell {tag}: app={app} regime={regime} vin={os.path.basename(vin)} ====", flush=True)
    subprocess.run(["python3", f"{CUM}/harness.py", "--app", app, "--model", "sonnet",
        "--regime", regime, "--vault-in", vin, "--vault-out", vout, "--cfg", cfg,
        "--workdir", wd, "--spec", f"{CUM}/{spec}", "--out", out, "--max-rounds", "6", "--learn", learn])
    print(f"==== {tag} done -> {out} ====", flush=True)

refresh(CFG_WARM); refresh(CFG_COLD)
shutil.rmtree(VAULT, ignore_errors=True); os.makedirs(VAULT, exist_ok=True)

cell("notes", "cold", "none", VAULT, CFG_WARM, "notes_spec.json", "notes", "yes")
shutil.rmtree(SMOKE + "/vault_notes", ignore_errors=True)
shutil.copytree(VAULT, SMOKE + "/vault_notes")

cell("links", "blended", VAULT, VAULT, CFG_WARM, "links_spec.json", "links", "yes")
shutil.rmtree(SMOKE + "/vault_noteslinks", ignore_errors=True)
shutil.copytree(VAULT, SMOKE + "/vault_noteslinks")

cell("feeds", "cold", "none", "none", CFG_COLD, "feeds_spec.json", "feeds-cold", "no")
cell("feeds", "blended", SMOKE + "/vault_notes", "none", CFG_WARM, "feeds_spec.json", "feeds-notes", "no")
cell("feeds", "blended", SMOKE + "/vault_noteslinks", "none", CFG_WARM, "feeds_spec.json", "feeds-noteslinks", "no")
print("SMOKE_CHAIN_DONE", flush=True)
