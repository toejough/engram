import sys, os, json, time, subprocess, shutil, uuid, re, threading, concurrent.futures as cf
P2="/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/phase2"
sys.path.insert(0,P2); sys.path.insert(0,os.path.dirname(P2))
import probe_phase2 as pp
import probe as p1
S1=os.path.dirname(os.path.abspath(__file__))
OUT=S1+"/results.jsonl"; RUN=S1+"/run"
TASKS={
 "rename":("kelp","In the kelp inventory CLI, rename the --shelf flag to --bay everywhere it appears. It is a hard rename, no alias for the old spelling. Get it done properly in this repo."),
 "feature":("lighthouse","Add a CSV export option to the lighthouse-log report tool, with tests and docs, and get it committed."),
 "bugfix":("beacon","Fix the crash in the beacon scheduler when the interval is 0, and update the docs to describe the new behavior."),
 "refactor":("marmot","Refactor the marmot config loader by splitting it into separate parser and validator modules without changing behavior."),
 "docs":("quill","Overhaul the quill CLI docs so they match the current commands and flags."),
}
def make_repo(path,name):
    os.makedirs(path+"/docs",exist_ok=True); os.makedirs(path+"/tests",exist_ok=True)
    open(path+f"/{name}.py","w").write(f'''import argparse
def load(path):
    return [l.strip() for l in open(path)]
def run(items, interval=1):
    return [x for x in items][::interval]
def main():
    ap=argparse.ArgumentParser(prog="{name}")
    ap.add_argument("--shelf"); ap.add_argument("--interval",type=int,default=1)
    a=ap.parse_args()
    print(run(load(a.shelf or "in.txt"),a.interval))
if __name__=="__main__": main()
''')
    open(path+"/README.md","w").write(f"# {name}\nUse `python {name}.py --shelf FILE --interval N`.\n")
    open(path+"/docs/usage.md","w").write(f"# {name} usage\n`--shelf` selects the input; `--interval` sets step.\n")
    open(path+"/tests/test_basic.py","w").write(f"import {name}\ndef test_run():\n    assert {name}.run(['a','b'])==['a','b']\n")
    subprocess.run(f"cd {path} && git init -q && git add -A && git -c user.email=e@x -c user.name=e commit -qm init",shell=True,check=True)
PRICE=dict(i=3.0,o=15.0,cr=0.3,cw=3.75)  # $/Mtok estimate
def q_cmds(events):
    return [e for e in events if e["kind"]=="tool_use" and e["name"]=="Bash" and "engram query" in (e["input"].get("command") or "")]
def first_query_phrases(cmd):
    return re.findall(r"--phrase[= ]+(\"[^\"]*\"|'[^']*'|\S+)",cmd)
def run_trial(arm,task,trial,cfg,model="sonnet5"):
    name,prompt=TASKS[task]
    marker=f"PROBE-{uuid.uuid4().hex[:10]}"
    td=f"{RUN}/{arm}-{task}-{trial}"; shutil.rmtree(td,ignore_errors=True); os.makedirs(td)
    repo=td+"/repo"; make_repo(repo,name)
    shim=open(S1+f"/shim_{arm}.md").read()
    open(repo+"/CLAUDE.md","w").write(shim.rstrip()+f"\n\nPROBE-TOKEN: {marker}\n")
    subprocess.run(f"cd {repo} && git add CLAUDE.md && git -c user.email=e@x -c user.name=e commit -qm cfg",shell=True,check=True)
    p1.matrix.refresh_creds(cfg)
    env=pp.trial_env_phase2(cfg,td,repo)
    v=env["ENGRAM_VAULT_PATH"]; shutil.rmtree(v,ignore_errors=True); shutil.copytree(pp.REAL_VAULT,v)
    pp.remove_eval_session_notes(v,pp.EXCLUDE_LUHMANN_MIN)
    args=["claude","-p",prompt,"--output-format","stream-json","--verbose","--model",p1.MODELS[model],"--permission-mode","bypassPermissions"]
    t0=time.time(); usage=dict(i=0,o=0,cr=0,cw=0); killed=False; nlines=0; seen_q=False; final=None; degenerate=None
    proc=subprocess.Popen(args,cwd=repo,env=env,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
    timer=threading.Timer(300,proc.kill); timer.start()
    kill_at=None; slog=open(td+'/stream.jsonl','w'); tools=[]
    try:
        for line in proc.stdout:
            slog.write(line); slog.flush()
            try: o=json.loads(line)
            except Exception: continue
            if o.get("type")=="assistant":
                u=(o.get("message") or {}).get("usage") or {}
                usage["i"]+=u.get("input_tokens",0); usage["o"]+=u.get("output_tokens",0)
                usage["cr"]+=u.get("cache_read_input_tokens",0); usage["cw"]+=u.get("cache_creation_input_tokens",0)
                for b in (o.get("message") or {}).get("content") or []:
                    if b.get('type')=='tool_use': tools.append((b.get('name'),(b.get('input') or {}).get('command') or json.dumps(b.get('input'))[:200]))
                    if b.get("type")=="tool_use" and b.get("name")=="Bash" and "engram query" in (b.get("input") or {}).get("command",""):
                        seen_q=True; kill_at=time.time()
            if o.get("type")=="result": final=o
            if seen_q and kill_at and time.time()-kill_at>=0:  # kill right after first query issued (score phrases only)
                proc.kill(); killed=True; break
    finally:
        timer.cancel(); 
        try: proc.kill()
        except Exception: pass
        proc.wait()
    cost=(usage["i"]*PRICE["i"]+usage["o"]*PRICE["o"]+usage["cr"]*PRICE["cr"]+usage["cw"]*PRICE["cw"])/1e6
    if final and final.get("total_cost_usd"): cost=final["total_cost_usd"]
    slog.close()
    tp=p1.discover_transcript_paths(cfg,repo); raw=p1.transcript_raw_text(tp); ev=p1.parse_transcript_events(tp)
    qs=[t for t in tools if t[0]=='Bash' and 'engram query' in t[1]]
    first_tool=next((e for e in ev if e["kind"]=="tool_use"),None)
    rec=dict(arm=arm,task=task,trial=trial,marker_seen=marker in raw,
      shim_recheck_in_transcript=("Re-check phrase two" in raw),
      n_queries=len(qs),first_cmd=(qs[0][1] if qs else None),
      first_tool=(tools[0] if tools else None),
      killed_after_first_query=killed,final_is_error=bool(final and final.get("is_error")),
      cost_est=round(cost,4),wall_s=round(time.time()-t0,1),transcripts=tp,
      other_shim_hits=raw.count("Before your first tool call, on every request"))
    shutil.rmtree(v,ignore_errors=True)
    return rec
if __name__=="__main__":
    arms=sys.argv[1].split(","); tasks=sys.argv[2].split(","); n=int(sys.argv[3]); workers=int(sys.argv[4])
    os.makedirs(RUN,exist_ok=True)
    done=set()
    if os.path.exists(OUT):
        for l in open(OUT): r=json.loads(l); done.add((r["arm"],r["task"],r["trial"]))
    pool=p1.build_cfg_pool if False else pp.build_cfg_pool_noskills_phase2(RUN,workers)
    import queue; cq=queue.Queue(); [cq.put(c) for c in pool]
    jobs=[(a,t,i) for i in range(n) for t in tasks for a in arms if (a,t,i) not in done]
    lock=threading.Lock()
    def work(j):
        c=cq.get()
        try: r=run_trial(*j,c)
        except Exception as e: r=dict(arm=j[0],task=j[1],trial=j[2],error=str(e))
        finally: cq.put(c)
        with lock:
            open(OUT,"a").write(json.dumps(r)+"\n")
        print(j,r.get("cost_est"),r.get("n_queries"),r.get("error"),flush=True)
    with cf.ThreadPoolExecutor(workers) as ex: list(ex.map(work,jobs))
