import json, glob, os, collections
PRICES = {
  "claude-opus-4-8":      dict(i=5e-6,  o=25e-6, cw=6.25e-6,  cr=0.5e-6),
  "claude-sonnet-4-6":    dict(i=3e-6,  o=15e-6, cw=3.75e-6,  cr=0.30e-6),
  "claude-haiku-4-5-20251001": dict(i=1e-6, o=5e-6, cw=1.25e-6, cr=0.10e-6),
}
def find_tx(sid):
    g=glob.glob(f"/tmp/cummatrix/cfgpool/*/projects/*/{sid}.jsonl")
    return g[0] if g else None
def tx_cost(tx):
    tot=collections.Counter(); model=None; seen=set()
    for line in open(tx):
        try:o=json.loads(line)
        except:continue
        m=o.get("message") or {}; u=m.get("usage"); md=m.get("model") or o.get("model")
        if md and md!="<synthetic>": model=md
        if not u: continue
        mid=m.get("id")
        if mid and mid in seen: continue
        if mid: seen.add(mid)
        tot["i"]+=u.get("input_tokens",0);tot["o"]+=u.get("output_tokens",0)
        tot["cw"]+=u.get("cache_creation_input_tokens",0);tot["cr"]+=u.get("cache_read_input_tokens",0)
    if not model or model not in PRICES: return None,None
    p=PRICES[model]
    return tot["i"]*p["i"]+tot["o"]*p["o"]+tot["cw"]*p["cw"]+tot["cr"]*p["cr"], tot

# MATCHED: only cells where transcript found; sum reported & recomputed over SAME cells
agg=collections.defaultdict(lambda: dict(rep=0.0, rec=0.0, n=0, tok=collections.Counter()))
for f in glob.glob("/tmp/cummatrix/results/*feeds*.json"):
    d=json.load(open(f))
    if d.get("round1_score") is None: continue
    model=os.path.basename(f).split("-")[0]; sid=d.get("session_id")
    tx=find_tx(sid) if sid else None
    if not tx: continue
    rc,tok=tx_cost(tx)
    if rc is None: continue
    a=agg[model]; a["rep"]+=d.get("total_cost",0) or 0; a["rec"]+=rc; a["n"]+=1; a["tok"]+=tok

print("MATCHED comparison (same cells both sides). reported=CLI total_cost_usd, recomputed=tokens×current prices")
print("%-8s %5s  %11s %12s %7s   avg cache-read/cell" % ("model","cells","reported$","recomputed$","ratio"))
for m in ["haiku","sonnet","opus"]:
    a=agg[m]
    if not a["n"]: continue
    r=a["rec"]/a["rep"] if a["rep"] else 0
    print("%-8s %5d  %11.2f %12.2f %6.2fx   %s" % (m,a["n"],a["rep"],a["rec"],r,f"{a['tok']['cr']//a['n']:,}"))
print()
print("Per-cell avg (matched cells):")
for m in ["haiku","sonnet","opus"]:
    a=agg[m]
    if not a["n"]: continue
    print("  %-7s reported $%.2f/cell  recomputed $%.2f/cell  (tokens: in=%d cw=%d cr=%d out=%d avg)" % (
        m, a["rep"]/a["n"], a["rec"]/a["n"], a["tok"]["i"]//a["n"], a["tok"]["cw"]//a["n"], a["tok"]["cr"]//a["n"], a["tok"]["o"]//a["n"]))
