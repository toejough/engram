#!/usr/bin/env python3
"""Convergence loop for ONE arm.

Starting from the existing round-1 build, repeatedly: score the workspace
against the architecture rubric; if arch >= BAR stop; else assemble the FIXED
spec-free hints for the currently-failing items (identical wording across all
arms) and resume the session for another round. Caps at MAX_ROUNDS.

Emits a RESULT line: converged?, rounds, human_interactions, turns, cost.
Run per arm; reuses run-layer-resume.sh for the actual --resume build.
"""
import sys, os, re, json, subprocess, tempfile

REPO = '/Users/joe/repos/personal/engram'
RUN = REPO + '/dev/eval/.layer-run'
RESUME = REPO + '/dev/eval/run-layer-resume.sh'
CFG = os.environ.get('CLAUDE_CFG_DIR', RUN + '/cfg')
BAR, MAX_ROUNDS = 9, 5

HINTS = {
 'A1': "I want to unit-test the core logic in isolation — the tests must not create real files on disk or read the real current time.",
 'A2': "The business logic, the file storage, and the command-parsing/printing are tangled together. Separate those concerns so the core logic stands alone.",
 'A3': "Adding a test case is repetitive and awkward — make the tests data-driven so a new case is one row.",
 'A4': "When a contact isn't found or an action is invalid, a caller can't tell the error kind — it's just a string. Make the failure kinds programmatically distinguishable.",
 'A5': "Behavior depends on shared package-level mutable state; remove the global state.",
 'A6': "Drop the third-party dependencies — the standard library covers the CLI parsing and output you need here.",
 'A7': "Two persistence problems: the data file isn't in the conventional per-user data directory, and if the process is interrupted mid-save the file can be left truncated or corrupt.",
 'A8': "The default listing is hard to scan (columns don't line up), and there's no machine-readable output mode for scripting.",
 'A9': "Output has no visual emphasis; and if you add it, it must switch off automatically when output isn't a terminal or when NO_COLOR is set.",
 'A10': "Some expected operations are missing — I need to add, list, show, edit, remove, and search contacts.",
}
OPS = [('add', ['add']), ('list', ['list', 'ls']), ('show', ['show', 'view', 'get']),
       ('edit', ['edit', 'update']), ('del', ['rm', 'remove', 'delete']), ('search', ['search', 'find'])]


def load(ws):
    code = tests = gomod = ''
    for root, _, fs in os.walk(ws):
        for f in fs:
            if f.endswith('.go'):
                t = open(os.path.join(root, f), errors='ignore').read()
                if f.endswith('_test.go'):
                    tests += t
                else:
                    code += t
            if f == 'go.mod':
                gomod = open(os.path.join(root, f)).read()
    return code, tests, gomod


def score(ws):
    c, t, gm = load(ws)
    a = c + t
    R = lambda p, s=a: bool(re.search(p, s, re.I | re.M))
    cmds = sum(1 for _, syns in OPS if any(re.search(r'"%s"' % s, a) for s in syns))
    chk = {
     'A1': R(r'type \w*store\w* interface') and R(r'io\.Writer') and R(r'func New\w*App|NewApp\('),
     'A2': R(r'type \w*store\w* interface') and (R(r'core\.go|type App struct') or R(r'package store')),
     'A3': R(r'\[\]struct\s*\{', t) and R(r'mem|inmem|TempDir|fake|stub', t),
     'A4': (R(r'var Err\w+|errors\.New')) and (R(r'errors\.Is') or ': %w' in a),
     'A5': not R(r'^var (store|db|state|cfg) ', c),
     'A6': not re.search(r'\n\s*require ', gm) and 'require (' not in gm,
     'A7': (R(r'XDG_DATA_HOME') or R(r'\.local/share')) and R(r'os\.Rename|\.tmp'),
     'A8': (R(r'tabwriter') or R(r'%-\d')) and (R(r'json\.Marshal') and (R(r'"json"') or R(r'--json') or R(r'asJSON|jsonOut|outputJSON'))),
     'A9': R(r'NO_COLOR'),
     'A10': cmds >= 5,
    }
    return sum(chk.values()), [k for k, v in chk.items() if not v]


def totals(arm, upto):
    tt = tc = 0
    for i in range(1, upto + 1):
        p = '%s/%s-r%d.json' % (RUN, arm, i)
        if os.path.exists(p):
            d = json.load(open(p))
            tc += d.get('total_cost_usd', 0) or 0
            tt += d.get('num_turns', 0) or 0
    return tt, tc


def main():
    arm = sys.argv[1]
    rnd = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    ws = open('%s/%s-r%d.workspace' % (RUN, arm, rnd)).read().strip()
    sc, failing = score(ws)
    print('%s round %d: arch=%d/10 failing=%s' % (arm, rnd, sc, failing), flush=True)
    while sc < BAR and rnd < MAX_ROUNDS:
        fb = ("I reviewed your contact manager and ran it. It works, but please address "
              "these before it's done:\n\n" + '\n'.join('- ' + HINTS[k] for k in failing if k in HINTS) +
              "\n\nKeep going autonomously until done; do not write any memory notes.")
        fh = tempfile.NamedTemporaryFile('w', suffix='.txt', delete=False)
        fh.write(fb)
        fh.close()
        prev, new = 'r%d' % rnd, 'r%d' % (rnd + 1)
        env = dict(os.environ, CLAUDE_CFG_DIR=CFG)
        r = subprocess.run(['bash', RESUME, arm, prev, new, fh.name], env=env, capture_output=True, text=True)
        if r.returncode != 0:
            print('%s round %d RESUME FAILED: %s' % (arm, rnd + 1, r.stderr[-300:]), flush=True)
            break
        rnd += 1
        ws = open('%s/%s-r%d.workspace' % (RUN, arm, rnd)).read().strip()
        sc, failing = score(ws)
        print('%s round %d: arch=%d/10 failing=%s' % (arm, rnd, sc, failing), flush=True)
    tt, tc = totals(arm, rnd)
    print('RESULT %s converged=%s rounds=%d human_interactions=%d turns=%d cost=$%.3f'
          % (arm, sc >= BAR, rnd, rnd - 1, tt, tc), flush=True)


if __name__ == '__main__':
    main()
