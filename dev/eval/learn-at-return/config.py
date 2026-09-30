"""Frozen constants for the learn-at-subagent-return eval (openspec change learn-at-subagent-return,
design D5). Every value here is quoted from design.md / tasks.md; change them only with the design."""

# The commit every non-variable installed text is pinned to (tasks 1.1, design D5).
PIN = "67911117"

# The four production guidance files, in the order the real ~/.claude/CLAUDE.md imports them.
GUIDANCE = ("recall", "delegate", "learn", "shim")
IMPORT_LINES = tuple(f"@~/.claude/engram/{name}.md" for name in GUIDANCE)

# One plain marker line per guidance file (design D5, delivery gate).
FILE_MARKERS = {
    "recall": "LAR-RECALL-2H6W",
    "delegate": "LAR-DELEGATE-9C3N",
    "learn": "LAR-LEARN-8J2D",
    "shim": "LAR-SHIM-5V1R",
}

# The per-arm token appended to learn.md only. A revised GREEN wording gets a new token (D5 item 3).
ARM_TOKENS = {
    "RED": "LAR-RED-3M8T",
    "GREEN": "LAR-GREEN-7Q4K",
}

MARKER_PREFIX = "Session tracking token: "

# All six engram skills, installed in both arms at PIN.
SKILLS = ("recall", "learn", "please", "route", "curate", "write-memory")

# Layer 1 (design D5).
ALLOWED_TOOLS = ("Bash(engram:*)", "Read", "Glob", "Grep", "Skill", "Agent")
DISALLOWED_TOOLS = ("WebFetch", "WebSearch", "Bash(git:*)", "Bash(curl:*)", "Bash(security:*)")

# The probe arm adds exactly this to the allowlist (design D5, per-batch write probe).
PROBE_EXTRA_TOOL = "Bash(touch:*)"

# The fixture subagent's name and model (design D5).
FIXTURE_AGENT = "unit-worker"
FIXTURE_MODEL = "haiku"

# Tool names the scorer keys on; the smoke (task 1.5) confirms them against the installed claude.
AGENT_TOOL_NAMES = ("Agent", "Task")
SKILL_TOOL_NAME = "Skill"
BASH_TOOL_NAME = "Bash"
