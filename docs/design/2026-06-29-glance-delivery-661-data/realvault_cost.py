"""#662 de-risk — glance vs deep recall cost on a REAL-SCALE vault (per-rep fresh copy of the live vault).

Per rep: cp -R the live vault + chunk index to a fresh temp pair (isolated), run a recall-only claude with the
glance cfg vs the deep cfg against the copy, capture wall-time + $ + turns. Asserts both env vars before each
launch (else recall falls back to the LIVE vault). One-time payload-floor check confirms the copy reproduces
real scale. Densest-case Go task.
"""
import sys, os, json, time, subprocess, tempfile, shutil
TRAPS = "/Users/joe/repos/personal/engram/dev/eval/traps"
sys.path.insert(0, TRAPS)
from wrun import build_warm_cfg
from run import MODELS

SCRATCH = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"
LIVE_VAULT = os.path.expanduser("~/.local/share/engram/vault")
LIVE_CHUNKS = os.path.expanduser("~/.local/share/engram/chunks")
REPS = int(os.environ.get("REPS", "5"))

OVERRIDE = """
## GLANCE MODE (cheap rung) — read this first, it overrides the procedure below
You are running recall in **glance** mode (the cheap rung of the depth dial):
- **Step 1:** generate exactly **3** phrases (not 10) — the three most on-point for your situation.
- **Do normally:** Step 0/0.5, Step 2 query, Step 2.5A (read candidates), Step 2.5B (recency-resolve), Step 2.7 (activate used notes), Step 3 (synthesis + apply conventions as requirements).
- **SKIP the write side:** do NOT do Step 2.5C coverage writes, Step 2.6 linking, or Step 4 — make **no** `engram amend` or `engram learn` calls. Surface, recency-resolve, apply, then stop.
"""

def build_glance(dst):
    build_warm_cfg(dst)
    sk = os.path.join(dst, "skills", "recall", "SKILL.md")
    t = open(sk).read().replace("Always generate exactly **10**", "Always generate exactly **3**")
    lines = t.split("\n")
    for i, l in enumerate(lines):
        if l.startswith("# "):
            lines.insert(i + 1, OVERRIDE); break
    open(sk, "w").write("\n".join(lines))

CFGS = os.path.join(SCRATCH, "rv-cfgs"); shutil.rmtree(CFGS, ignore_errors=True); os.makedirs(CFGS)
DEEP = os.path.join(CFGS, "deep"); build_warm_cfg(DEEP)
GLANCE = os.path.join(CFGS, "glance"); build_glance(GLANCE)

PROMPT = ("Invoke your /recall skill to surface relevant project memory for the task below, then STOP and "
          "briefly report what you recalled. Do NOT write any code, do NOT build — recall only.\n\n"
          "Task: add an exported Go function that makes an HTTP GET request to a URL and returns the response "
          "body, with context cancellation and wrapped errors.")

def fresh_copy():
    base = tempfile.mkdtemp(prefix="rv-")
    v = os.path.join(base, "vault"); c = os.path.join(base, "chunks")
    subprocess.run(["cp", "-R", LIVE_VAULT, v], check=True)
    subprocess.run(["cp", "-R", LIVE_CHUNKS, c], check=True)
    return base, v, c

def time_one(cfg):
    base, v, c = fresh_copy()
    wd = tempfile.mkdtemp(prefix="rv-wd-")
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["ENGRAM_VAULT_PATH"] = v
    env["ENGRAM_CHUNKS_DIR"] = c
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", "rv")
    # ASSERT isolation: both set, non-empty, pointing at the COPY (not live)
    assert env["ENGRAM_VAULT_PATH"] == v and os.path.isdir(v) and v != LIVE_VAULT, "vault env not isolated"
    assert env["ENGRAM_CHUNKS_DIR"] == c and os.path.isdir(c) and c != LIVE_CHUNKS, "chunks env not isolated"
    args = ["claude", "-p", PROMPT, "--output-format", "json", "--model", MODELS["opus"],
            "--permission-mode", "bypassPermissions"]
    t0 = time.time()
    r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
    dt = time.time() - t0
    try:
        out = json.loads(r.stdout)
    except Exception:
        out = {}
    shutil.rmtree(base, ignore_errors=True); shutil.rmtree(wd, ignore_errors=True)
    return {"wall_s": dt, "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns")}

# --- payload-floor check (once): does the copy reproduce real scale? ---
fb_base, fb_v, fb_c = fresh_copy()
fenv = dict(os.environ); fenv["ENGRAM_VAULT_PATH"] = fb_v; fenv["ENGRAM_CHUNKS_DIR"] = fb_c
fphrases = ["building a command-line app in Go", "making an HTTP request in Go", "wrapping errors in Go",
            "Go context cancellation", "idiomatic Go error handling", "Go CLI conventions",
            "HTTP GET request body", "exported Go function", "Go code quality standards", "returning errors in Go"]
fcmd = ["engram", "query"] + sum([["--phrase", p] for p in fphrases], [])
fr = subprocess.run(fcmd, env=fenv, capture_output=True, text=True)
floor = {"payload_bytes": len(fr.stdout), "chunk_items": fr.stdout.count("kind: chunk"),
         "clusters": fr.stdout.count("- cluster_id:") or fr.stdout.count("cluster_id:")}
shutil.rmtree(fb_base, ignore_errors=True)
print(f"PAYLOAD FLOOR (10-phrase query on the copy): {floor}", flush=True)

results = {"deep": [], "glance": []}
for arm, cfg in (("deep", DEEP), ("glance", GLANCE)):
    for rep in range(REPS):
        m = time_one(cfg)
        results[arm].append(m)
        print(f"{arm:6} rep={rep} wall={m['wall_s']:.0f}s ${m['cost']:.3f} turns={m['turns']}", flush=True)

def stats(xs, k):
    vs = [x[k] for x in xs]; return sum(vs)/len(vs), min(vs), max(vs)
print("\n=== Real-vault recall-only: glance vs deep ===")
for arm in ("deep", "glance"):
    wm, wlo, whi = stats(results[arm], "wall_s"); cm, _, _ = stats(results[arm], "cost")
    tm, _, _ = stats(results[arm], "turns")
    print(f"  {arm:6}: wall {wm:.0f}s (range {wlo:.0f}-{whi:.0f}) | ${cm:.3f} | turns {tm:.0f}")
dw = stats(results["deep"], "wall_s"); gw = stats(results["glance"], "wall_s")
delta = dw[0]-gw[0]; spread = max(dw[2]-dw[1], gw[2]-gw[1])
print(f"\n  Δ wall = {delta:.0f}s | ratio = {dw[0]/gw[0]:.2f}x | within-arm spread = {spread:.0f}s")
print(f"  noise-floor check: Δ ({delta:.0f}s) {'>' if delta>spread else '<='} spread ({spread:.0f}s)")
out={"floor":floor,"results":results,"delta_s":delta,"ratio":dw[0]/gw[0],"spread_s":spread}
json.dump(out, open(os.path.join(SCRATCH,"realvault_cost_results.json"),"w"), indent=2)
print("REALVAULT_DONE")
