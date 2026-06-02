import json, glob, os, collections
PRICES = {  # current Anthropic API rates, $/Mtok (verified 2026-06-02 from platform.claude.com)
  "haiku":  dict(name="Claude Haiku 4.5",  i=1.0,  o=5.0,  cw=1.25,  cr=0.10),
  "sonnet": dict(name="Claude Sonnet 4.6", i=3.0,  o=15.0, cw=3.75,  cr=0.30),
  "opus":   dict(name="Claude Opus 4.8",   i=5.0,  o=25.0, cw=6.25,  cr=0.50),
}
MK={"haiku":"claude-haiku-4-5-20251001","sonnet":"claude-sonnet-4-6","opus":"claude-opus-4-8"}
def find_tx(sid):
    g=glob.glob(f"/tmp/cummatrix/cfgpool/*/projects/*/{sid}.jsonl"); return g[0] if g else None
def tx_tokens(tx):
    tot=collections.Counter(); seen=set()
    for line in open(tx):
        try:o=json.loads(line)
        except:continue
        m=o.get("message") or {}; u=m.get("usage")
        if not u: continue
        mid=m.get("id")
        if mid and mid in seen: continue
        if mid: seen.add(mid)
        tot["i"]+=u.get("input_tokens",0);tot["o"]+=u.get("output_tokens",0)
        tot["cw"]+=u.get("cache_creation_input_tokens",0);tot["cr"]+=u.get("cache_read_input_tokens",0)
    return tot
agg=collections.defaultdict(lambda: dict(tok=collections.Counter(), rep=0.0, n=0))
for f in glob.glob("/tmp/cummatrix/results/*feeds*.json"):
    d=json.load(open(f))
    if d.get("round1_score") is None: continue
    model=os.path.basename(f).split("-")[0]; sid=d.get("session_id")
    tx=find_tx(sid) if sid else None
    if not tx: continue
    agg[model]["tok"]+=tx_tokens(tx); agg[model]["rep"]+=d.get("total_cost",0) or 0; agg[model]["n"]+=1
print("PER-CELL AVERAGE TOKENS BY TYPE (mean over cells with retained transcripts)")
print("%-8s %4s %10s %10s %12s %10s | %9s %9s" % ("model","n","input","cacheWr","cacheRd","output","recompute$","reported$"))
for m in ["haiku","sonnet","opus"]:
    a=agg[m]; n=a["n"] or 1; t=a["tok"]; p=PRICES[m]
    rc=(t["i"]*p["i"]+t["o"]*p["o"]+t["cw"]*p["cw"]+t["cr"]*p["cr"])/1e6
    print("%-8s %4d %10d %10d %12d %10d | %8.2f  %8.2f" % (
        m, a["n"], t["i"]//n, t["cw"]//n, t["cr"]//n, t["o"]//n, rc/n, a["rep"]/n))
print()
print("PRICE SHEET ($/Mtok): " + " | ".join(f"{m}: in {PRICES[m]['i']} out {PRICES[m]['o']} cacheWr {PRICES[m]['cw']} cacheRd {PRICES[m]['cr']}" for m in ["haiku","sonnet","opus"]))
print("cost = (input*in + cacheWr*cw + cacheRd*cr + output*out) / 1e6")
