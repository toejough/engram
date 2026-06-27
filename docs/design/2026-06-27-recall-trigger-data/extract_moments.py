#!/usr/bin/env python3
"""Scout pass: extract candidate correction/failure moments from engram session logs,
and count the denominators the over-fire math needs. NOT the analysis — just the work-list."""
import json, os, re, glob, collections

ENGRAM_DIR = os.path.expanduser("~/.claude/projects/-Users-joe-repos-personal-engram")
OUT = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"

# Strong user-correction signals (candidate extraction; agents filter false positives).
SIGNALS = [
    r"\bno,", r"\bno\.", r"\bdon'?t\b", r"\bstop\b", r"\bactually\b", r"that'?s not",
    r"that'?s wrong", r"\byou should(?:n'?t)?\b", r"why did you", r"why are you",
    r"instead of", r"\bi said\b", r"not what i", r"\brevert\b", r"\bundo\b",
    r"you missed", r"you forgot", r"\bincorrect\b", r"\bwrong\b", r"\bmistake\b",
    r"no need", r"didn'?t ask", r"\bi didn'?t\b", r"that'?s not right", r"you can'?t",
    r"never\b", r"you broke", r"that'?s a", r"you assumed", r"too eager", r"overcomplicat",
]
SIG_RE = re.compile("|".join(SIGNALS), re.IGNORECASE)

SR_RE = re.compile(r"<system-reminder>.*?</system-reminder>", re.DOTALL)

def is_genuine_user(t):
    """Exclude harness/injected pseudo-user messages; keep real human turns."""
    s = t.strip()
    if not s:
        return False
    if s.startswith("Base directory for this skill:"):
        return False
    if "<task-notification>" in s or "<task-id>" in s:
        return False
    if s.startswith("<command-name>") or "<local-command-stdout>" in s or "<command-message>" in s:
        return False
    if s.startswith("Caveat:"):
        return False
    if s.startswith("<system-reminder>"):
        return False
    if s.startswith("ARGUMENTS:") or "\nARGUMENTS:" in s[:80]:
        return False
    if s.startswith("Launching skill:") or s.startswith("Result of calling"):
        return False
    return True

def text_of(content):
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for b in content:
            if isinstance(b, dict):
                if b.get("type") == "text":
                    parts.append(b.get("text", ""))
                elif b.get("type") == "tool_result":
                    c = b.get("content", "")
                    parts.append(text_of(c) if not isinstance(c, str) else c)
        return " ".join(parts)
    return ""

def assistant_summary(content):
    tools, txt = [], []
    if isinstance(content, list):
        for b in content:
            if isinstance(b, dict):
                if b.get("type") == "tool_use":
                    tools.append(b.get("name", "?"))
                elif b.get("type") == "text":
                    txt.append(b.get("text", "")[:200])
    return tools, " ".join(txt)[:300]

files = sorted(glob.glob(os.path.join(ENGRAM_DIR, "*.jsonl")))
tool_counts = collections.Counter()
bash_cmd_counts = collections.Counter()
total_user_turns = 0
total_assistant_turns = 0
total_tool_uses = 0
candidates = []

for fp in files:
    last_asst = ("", [])  # (text, tools)
    with open(fp, errors="ignore") as fh:
        for ln, line in enumerate(fh, 1):
            line = line.strip()
            if not line:
                continue
            try:
                ev = json.loads(line)
            except Exception:
                continue
            typ = ev.get("type")
            msg = ev.get("message", {}) if isinstance(ev.get("message"), dict) else {}
            content = msg.get("content", "")
            if typ == "assistant":
                total_assistant_turns += 1
                tools, txt = assistant_summary(content)
                total_tool_uses += len(tools)
                for t in tools:
                    tool_counts[t] += 1
                    if t == "Bash" and isinstance(content, list):
                        for b in content:
                            if isinstance(b, dict) and b.get("type") == "tool_use" and b.get("name") == "Bash":
                                cmd = (b.get("input", {}) or {}).get("command", "")
                                head = cmd.strip().split()[0:2]
                                bash_cmd_counts[" ".join(head)] += 1
                last_asst = (txt, tools)
            elif typ == "user":
                # skip tool_result-only user events (not real user turns)
                is_tool_result = isinstance(content, list) and all(
                    isinstance(b, dict) and b.get("type") == "tool_result" for b in content) and content
                if is_tool_result:
                    continue
                utext = text_of(content)
                if not is_genuine_user(utext):
                    continue
                utext = SR_RE.sub("", utext).strip()  # strip appended system-reminders
                if not utext:
                    continue
                total_user_turns += 1
                if SIG_RE.search(utext):
                    m = SIG_RE.search(utext)
                    candidates.append({
                        "file": os.path.basename(fp), "line": ln,
                        "signal": m.group(0).lower(),
                        "user": utext[:400].replace("\n", " "),
                        "prior_asst_text": last_asst[0].replace("\n", " "),
                        "prior_asst_tools": last_asst[1],
                    })

with open(os.path.join(OUT, "candidates.json"), "w") as f:
    json.dump(candidates, f, indent=1)

print(f"=== CORPUS (engram top-level sessions: {len(files)} files) ===")
print(f"total user turns (real, non-tool-result): {total_user_turns}")
print(f"total assistant turns: {total_assistant_turns}")
print(f"total tool_use calls: {total_tool_uses}")
print(f"candidate correction moments: {len(candidates)}")
print(f"\n=== DENOMINATORS: tool_use by name (top 20) ===")
for t, c in tool_counts.most_common(20):
    print(f"  {c:6d}  {t}")
print(f"\n=== Bash first-2-words (top 25 — for git push / commit etc denominators) ===")
for t, c in bash_cmd_counts.most_common(25):
    print(f"  {c:5d}  {t}")
print(f"\n=== correction-signal breakdown (top 25) ===")
sigc = collections.Counter(c["signal"] for c in candidates)
for s, c in sigc.most_common(25):
    print(f"  {c:4d}  {s!r}")
