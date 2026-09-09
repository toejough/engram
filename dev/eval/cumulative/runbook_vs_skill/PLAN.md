# Runbook-vs-Skill Headless Eval: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Measure whether a runbook note (procedural memory) or fact note (semantic memory) in the engram vault can replace a skill in headless trials, using a three-arm design (skill vs runbook vs fact) with a novel fictional task and Joe's parity/functionality decision frame.

**Architecture:** Three independent arms (S/R/F) execute the same multi-step task procedure written three ways:
- Arm S: Procedure as a real SKILL.md deployed to trial's `.claude/skills/` directory (discovered same way recall skill is discovered)
- Arm R: Procedure as a runbook note in a fixture vault (fields: situation, numbered steps, done_when)
- Arm F: Procedure as a fact note in the fixture vault (semantic framing of the same steps)

All arms receive identical project CLAUDE.md (with inlined recall guidance + validity marker), fresh headless `claude -p` per trial. Task: add a new sensor type to a fictional telemetry module with idiosyncratic procedural steps (registry entry in unusual file, codegen command, fixed ordering, numbered migration file, ledger-style update, validation command). Scoring: **FOUND** (tool_use from transcript: Skill for S, Bash `engram query` for R/F), **FOLLOWED** (mean k/N idiosyncratic steps performed, each verified from transcript or repo state), **END-STATE** (done_when checks pass), **COST**. One smoke (sonnet, 1 trial/arm) then paid opus run (5 trials/arm).

**Tech Stack:** Python 3.11+; Claude SDK (`claude -p`); engram binary (on PATH via isolation); bash for mechanical end-state checks; skills directory in trial project `.claude/`.

**Spec:** This plan implements the experiment confirmed with Joe (no redesign). Constraints: fictional domain (telemetry module, not real code); fixture repo has idiosyncratic steps no model can guess without instructions (registry entry in `lib/sensors/registry.txt` keyed by sensor ID + version tuple, codegen via `./scripts/sensors.py`, migration file in `migrations/NNNN_sensor_<name>.go` with exact format, ledger update in `TELEMETRY_CHANGELOG.log` with timestamp + author field, ordered dependency on migration before validation). Validity gate: marker in CLAUDE.md must appear in trial transcript; trials without marker are discarded before scoring (verify-treatment-delivery, never score 0).

**Source & Mechanics (file:line citations):**
- **Recall & learn skill deployment (all three arms):** The trial cfg is built via `dev/eval/cumulative/matrix.py::build_cfg_template(warm=True)` (lines 84–89), which copies the repo's real recall and learn skills from `agent-instructions/skills/recall` and `agent-instructions/skills/learn` into the trial cfg at `cfg/skills/recall` and `cfg/skills/learn`. These are NOT sourced from the operator's ~/.claude/skills/; they come from the repo itself, ensuring isolation. Fixture skill for Arm S lives additionally in trial project `.claude/skills/sensors-add-skill/`. Verify all three arms by checking transcripts for `"skill":"recall"` and `"skill":"learn"` JSON fields in session .jsonl files under cfg/projects/ (per `dev/eval/cumulative/underload_repro/run_underload_repro.py:_count_markers()` lines 159–174).
- **FOUND scoring from tool_use:** Parse session .jsonl files under `cfg/projects/` (created by `claude -p`). Search for JSON field `"skill":"<fixture-skill-name>"` for Arm S, and `"tool":"bash"` + `"engram query"` in tool call text for Arm R/F (per underload_repro.py:_count_markers(), lines 172–173). Arm R/F also verify tool_result contains the fixture note path.
- **Vault path resolution:** `internal/cli/ingest.go:DataDirFromHome()` sets XDG_DATA_HOME fallback; `isolation.py:isolated_env()` sets ENGRAM_VAULT_PATH, ENGRAM_CHUNKS_DIR, ENGRAM_TRANSCRIPT_DIR per-trial (isolation.py line 26 names ENGRAM_STATE_VARS); trials without these set are rejected by `isolation.assert_isolated()` (isolation.py line 88–100).
- **Runbook schema:** `openspec/specs/learn-runbook-capture/spec.md` — situation, done_when, body, source required; position/target for Luhmann hierarchy. Embed-on-write produces `.vec.json` sidecars.
- **Cost reference:** `dev/eval/LEDGER.md` row 39–40 (C1-C2 warm-vs-cold): clean measurement n=8/arm, "$3.08 costlier" total warm op, with build-phase +$1.00. These trials run tools + edit files (not single-response probe like endorse_cue), so budget $0.60–1.00 per opus trial.

## Global Constraints

- **Model tiers:** Smoke uses sonnet; paid run uses opus (vault note 258: haiku stalls on multi-step procedures)
- **Trial isolation:** Each trial has per-trial vault (ENGRAM_VAULT_PATH), chunks (ENGRAM_CHUNKS_DIR), transcripts (ENGRAM_TRANSCRIPT_DIR); shared global vault paths are forbidden (`isolation.py::assert_isolated()` line 88–100)
- **Skill deployment for Arm S:** Skill lives in trial project `.claude/skills/<fixture-skill-name>/` created at trial setup; `claude -p` resolves it the same way recall skill is resolved (cloudCode looks up skills from CLAUDE_CONFIG_DIR/.claude/skills/)
- **Recall/learn skills in all arms:** All three arms get recall and learn skills from `dev/eval/cumulative/matrix.py::build_cfg_template(warm=True)` (lines 84–89), which copies these from repo's `agent-instructions/skills/` into trial cfg/skills/ — never from operator's ~/.claude/
- **Marker validity gate:** Fixture CLAUDE.md contains a per-run unique marker `RUNBOOK-VS-SKILL-PROBE-<uuid>`; trials where marker is absent from transcript are discarded before scoring (marker_seen validity check; parallels endorse_cue design, never scored 0)
- **Runbook/fact creation:** Built with real engram binary via `engram learn runbook` and `engram learn fact` commands in fixture-vault setup; notes receive `.vec.json` sidecars on write (embed-on-write per spec)
- **Note-505 hygiene:** Procedure appears ONLY in skill/runbook/fact; never in README, Makefile comments, fixture documentation, or task scenario

---

## File Structure & Directory Layout

```
dev/eval/cumulative/runbook_vs_skill/
├── PLAN.md                              # This file
├── README.md                            # Results summary + parity/functionality frame
├── probe.py                             # Main harness: 3-arm fixture setup, trial spawning, transcript parsing
├── fixtures/                            # Fixture definitions (shared by all 3 arms)
│   ├── sensor-registration.txt          # Task scenario (identical across arms)
│   └── done_when_checks.sh              # Mechanical end-state validator (bash script)
├── fixture-vaults/                      # Pre-built vault directories (per-arm vault templates)
│   ├── skill/
│   │   └── sensors-add-skill/           # Skill as directory tree under .claude/skills/
│   │       ├── SKILL.md                 # The actual skill (frontmatter + description + steps)
│   │       └── references/              # Skill supporting files (if any)
│   ├── runbook/
│   │   └── vault/                       # Pre-built with runbook note + .vec.json sidecar
│   └── fact/
│       └── vault/                       # Pre-built with fact note + .vec.json sidecar
├── fixture-repo-template/               # Seeded sensor-system codebase
│   ├── .git/                            # Initialized git repo
│   ├── lib/sensors/
│   │   ├── registry.txt                 # Sensor registry (idiosyncratic format)
│   │   └── types.go                     # Sensor type definitions
│   ├── scripts/
│   │   └── sensors.py                   # Codegen script (must be run as part of procedure)
│   ├── migrations/                      # Migration directory (empty; populated during task)
│   ├── TELEMETRY_CHANGELOG.log          # Ledger file (pre-seeded template)
│   ├── Makefile                         # Contains `make validate` target
│   └── .gitignore
├── results/                             # Created at runtime
│   ├── smoke_results.jsonl             # 3 smoke trials (one per arm)
│   └── opus_results.jsonl              # 15 full trials (5 per arm)
└── .probe-marker-<uuid>.txt            # Transient: unique marker per run
```

## Fifth Recall Cue Draft

**Context:** ADR-0027 §D1 prospective-memory gap (`#735: cue framed as situation-recognition, broadened`) requires implementation-intention wording naming the action, broadened "this or a similar situation" recognition (not task-specific), positive example of recurring routine inside the cue, no worded value gate, no negative examples. Style: match four existing bullets in `agent-instructions/guidance/recall.md` lines 9–16 (action is recalling, not judging; emphasis on cheap fire). **REWRITTEN to avoid task-echo**: cue now keys on general "multi-step task" and task-start situation, not on release/tag/push (which would match the fixture's own task and create measurement-invalid echo).

**Drafted fifth cue (to inline into fixture CLAUDE.md only — DO NOT edit the repo's guidance file):**

```markdown
- **Before you start a multi-step task** — one you may have done before, in this or a similar situation (the usual test/tag/push dance, a dependency bump, a cleanup pass) — 
  **run `/recall glance` first** (the action is recalling the situation, not reasoning from scratch) — the vault may hold a runbook for this or a similar situation.
```

**Justification:**
- Situation-recognition: "Before you start a multi-step task" + "in this or a similar situation" (general, broadened, not fixture-specific)
- Implementation intention: names action explicitly (`run /recall glance first`)
- Recurring routine examples: test/tag/push dance, dependency bump, cleanup pass (multiple exemplars, generic)
- No value gate: doesn't say "cheaper" or "avoid mistakes"
- No negative examples: doesn't name failures or anti-patterns
- No task-echo: keyed to task-start situation, not to "sensor registration" or "telemetry" (would be fixture-echo)
- Matches style: "the action is recalling the situation" parallels existing bullets' "action is X not Y" framing

---

## Task 1: Fixture Design & Repo Template (Multi-Step Sensor Registration)

**Files:**
- Create: `dev/eval/cumulative/runbook_vs_skill/fixtures/sensor-registration.txt`
- Create: `dev/eval/cumulative/runbook_vs_skill/fixtures/done_when_checks.sh`
- Create: `dev/eval/cumulative/runbook_vs_skill/fixture-repo-template/` (seeded git repo)

**Interfaces:**
- Produces: A seeded sensor-system git repo and task scenario that probe.py will clone/instantiate per trial

**Acceptance:**
- Fixture repo contains 6+ idiosyncratic, mechanically-checkable steps no model can guess without instructions:
  1. **Registry entry format:** `lib/sensors/registry.txt` uses tab-separated tuples `<sensor_id>:<version>\t<sensor_name>\t<date_added>` (not JSON, not YAML)
  2. **Codegen requirement:** Must run `./scripts/sensors.py` after editing registry (produces type stubs)
  3. **Migration file naming:** `migrations/NNNN_sensor_<name>.go` where NNNN is 4-digit zero-padded sequence number, must contain `func init() { registerSensor(...) }` boilerplate (checked by validator)
  4. **Ledger entry format:** `TELEMETRY_CHANGELOG.log` requires line: `[YYYY-MM-DD HH:MM:SS] <operator> Added sensor <id>:<version>` (exact timestamp + author field)
  5. **Fixed ordering:** Registry entry MUST come before running codegen, codegen MUST complete before migration file is created, migration MUST exist before `make validate` can pass
  6. **Validation command:** `make validate` must pass (checks all four artifacts exist and have correct format)
- Task prompt is natural and NOT echo-y: "Register a new sensor type (pressure_v2) in the telemetry system" (not "add to registry" or "run migration")
- Procedure appears ONLY in skill/note definitions, never in README/Makefile comments, fixture docs, or scenario
- Task scenario file contains only the scene-setting and task goal; all steps are in the procedure (skill/note)

**Steps:**

- [ ] **Step 1: Create the task scenario fixture**

Write `fixtures/sensor-registration.txt`:

```
You are a software engineer working on a telemetry system for IoT sensors.
The codebase lives in /tmp/trial-sensor-repo (which has been cloned for you).

The task: Register a new sensor type (pressure_v2) in the telemetry system.

Here's what you should know:
- The sensor system tracks different sensor types in a registry.
- There is a codegen tool to update type definitions.
- There is a migration system for schema changes.
- You can run `make validate` to verify the registration is complete.
- You have git, Python, make, and standard CLI tools available.

Go ahead and register the sensor. You must complete all steps, but you do NOT need 
to call any tools beyond those named here. Your transcript will be checked for 
evidence of each step being performed.
```

- [ ] **Step 2: Create the end-state validator script**

Write `fixtures/done_when_checks.sh`:

```bash
#!/bin/bash
set -e

REPO_PATH="${1:-.}"
cd "$REPO_PATH"

# Check 1: Registry entry exists with correct format
if ! grep -q "pressure_v2:[0-9.]*[[:space:]]*pressure_v2[[:space:]]*[0-9]" lib/sensors/registry.txt; then
  echo "FAIL: pressure_v2 registry entry not found or malformed"
  exit 1
fi

# Check 2: Migration file exists
MIGRATION=$(ls migrations/*_sensor_pressure_v2.go 2>/dev/null | head -1)
if [ -z "$MIGRATION" ]; then
  echo "FAIL: Migration file for pressure_v2 not found"
  exit 1
fi

# Check 3: Migration file has init function
if ! grep -q "func init().*registerSensor" "$MIGRATION"; then
  echo "FAIL: Migration file missing registerSensor init function"
  exit 1
fi

# Check 4: Changelog entry exists
if ! grep -q "Added sensor pressure_v2" TELEMETRY_CHANGELOG.log; then
  echo "FAIL: Changelog entry missing"
  exit 1
fi

# Check 5: Changelog has timestamp + author format
if ! grep "Added sensor pressure_v2" TELEMETRY_CHANGELOG.log | grep -q "\[[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}.*\].*Added sensor"; then
  echo "FAIL: Changelog entry missing proper timestamp format"
  exit 1
fi

# Check 6: make validate passes
if ! make validate >/dev/null 2>&1; then
  echo "FAIL: make validate failed"
  exit 1
fi

echo "PASS: All end-state checks successful"
exit 0
```

Run: `chmod +x fixtures/done_when_checks.sh`

- [ ] **Step 3: Create fixture git repo template**

Create `fixture-repo-template/` directory structure:

```bash
mkdir -p fixture-repo-template/{lib/sensors,scripts,migrations}
cd fixture-repo-template
git init
```

Create `lib/sensors/registry.txt`:

```
temp_v1	temperature	2024-01-15
humidity_v1	humidity	2024-01-16
```

Create `lib/sensors/types.go`:

```go
// Sensor type definitions — generated by sensors.py
package sensors

// Registered sensors
var (
	SensorTypes = []string{
		"temp_v1",
		"humidity_v1",
	}
)
```

Create `scripts/sensors.py`:

```python
#!/usr/bin/env python3
import os
import sys

# Read registry and generate types.go
registry_file = "lib/sensors/registry.txt"
output_file = "lib/sensors/types.go"

if not os.path.exists(registry_file):
    print(f"ERROR: {registry_file} not found")
    sys.exit(1)

sensors = []
with open(registry_file) as f:
    for line in f:
        parts = line.strip().split('\t')
        if len(parts) >= 2:
            sensors.append(parts[0])

# Generate types.go
with open(output_file, 'w') as f:
    f.write("// Sensor type definitions — generated by sensors.py\n")
    f.write("package sensors\n\n")
    f.write("// Registered sensors\n")
    f.write("var (\n")
    f.write("\tSensorTypes = []string{\n")
    for sensor in sensors:
        f.write(f'\t\t"{sensor}",\n')
    f.write("\t}\n)\n")

print(f"Generated {output_file} with {len(sensors)} sensor types")
```

Create `Makefile`:

```makefile
.PHONY: validate

validate:
	@echo "Validating sensor registry..."
	@test -f lib/sensors/registry.txt || (echo "ERROR: registry.txt missing" && exit 1)
	@test -d migrations || (echo "ERROR: migrations dir missing" && exit 1)
	@ls migrations/*_sensor_*.go >/dev/null 2>&1 || (echo "ERROR: no migration files found" && exit 1)
	@grep -q "Added sensor" TELEMETRY_CHANGELOG.log || (echo "ERROR: changelog missing entry" && exit 1)
	@echo "Validation passed"
```

Create `TELEMETRY_CHANGELOG.log`:

```
# Telemetry System Changelog

[2024-01-15 09:30:00] system Added sensor temp_v1
[2024-01-16 10:15:00] system Added sensor humidity_v1
```

Create `.gitignore`:

```
*.pyc
__pycache__/
.DS_Store
```

Initialize repo and commit:

```bash
cd fixture-repo-template
git add .
git commit -m "Initial commit: sensor system v1"
```

**Acceptance check:** Directory exists with all files; `git log` shows initial commit; `cd fixture-repo-template && make validate` currently fails (expected; repo is incomplete until task is run).

---

## Task 2: Build Fixture Vaults & Skill (Arm S, R, F Preparation)

**Files:**
- Create: `dev/eval/cumulative/runbook_vs_skill/fixture-vaults/skill/sensors-add-skill/SKILL.md`
- Create: `dev/eval/cumulative/runbook_vs_skill/fixture-vaults/runbook/vault/` (with runbook note + sidecar)
- Create: `dev/eval/cumulative/runbook_vs_skill/fixture-vaults/fact/vault/` (with fact note + sidecar)

**Interfaces:**
- Produces: Skill definition (for trial `.claude/skills/` deployment), pre-built runbook and fact notes with `.vec.json` sidecars

**Acceptance:**
- Arm S skill is a valid SKILL.md with frontmatter, description, all 6 numbered procedure steps (exact same steps as runbook/fact)
- Arm R runbook note has: situation = "when registering a new sensor type in a telemetry module", done_when = "validated: registry entry exists in registry.txt with correct tab format, codegen script has been run, migration file exists with correct naming and init function, changelog has timestamp+author entry, make validate passes", body = 6 numbered procedure steps
- Arm F fact note has same 6 steps reframed as semantic knowledge ("The sensor registration procedure involves six ordered operations...")
- Both notes have source attribution, `.vec.json` sidecars (created by real engram binary)

**Steps:**

- [ ] **Step 1: Write the fixture skill for Arm S**

Create `fixture-vaults/skill/sensors-add-skill/SKILL.md`:

```markdown
# Add Sensor Type to Telemetry System Skill

## Overview

Procedure for registering a new sensor type in the telemetry system's registry and migration system.

## When to use

When you need to register a new sensor type (e.g., pressure_v2) in the telemetry system.

## The procedure

1. Add a new entry to lib/sensors/registry.txt in tab-separated format: <sensor_id>:<version>\t<sensor_name>\t<date>.
   Example line: `pressure_v2:1.0	pressure	2024-02-01`

2. Run the codegen script to update type definitions: `python3 scripts/sensors.py`

3. Create a migration file in migrations/ with naming pattern NNNN_sensor_<name>.go where NNNN is the next 4-digit sequence number.
   The migration must contain a func init() block that calls registerSensor(<sensor_id>).

4. Add an entry to TELEMETRY_CHANGELOG.log with exact format: `[YYYY-MM-DD HH:MM:SS] <operator> Added sensor <id>:<version>`

5. Verify all changes are staged in git (do not commit yet).

6. Run `make validate` to verify the sensor registration is complete and correct.

## Done when

- Registry entry exists in lib/sensors/registry.txt with correct tab-separated format
- Codegen script has been executed (scripts/sensors.py ran without error)
- Migration file exists with correct naming convention (NNNN_sensor_<name>.go)
- Migration file contains proper init function with registerSensor call
- Changelog has entry with timestamp, operator field, and sensor addition note
- make validate returns success (exit code 0)
```

**Acceptance check:** File is valid markdown; contains all 6 numbered steps; matches the procedure in runbook/fact.

- [ ] **Step 2: Create runbook note for Arm R**

```bash
TEMP_VAULT="/tmp/runbook-vault-$$"
mkdir -p "$TEMP_VAULT"

export ENGRAM_VAULT_PATH="$TEMP_VAULT"
export XDG_DATA_HOME="/tmp/engram-xdg-$$"

engram learn runbook \
  --slug sensor-registration-procedure \
  --situation "when registering a new sensor type in a telemetry module" \
  --done-when "verified: registry entry exists in lib/sensors/registry.txt with tab-separated <id>:<version> format, codegen script (scripts/sensors.py) has been executed, migration file NNNN_sensor_<name>.go exists with correct naming, migration contains func init() with registerSensor call, changelog has [YYYY-MM-DD HH:MM:SS] operator timestamp and sensor addition note, make validate passes with exit code 0" \
  --body "1. Add new entry to lib/sensors/registry.txt in tab-separated format: <sensor_id>:<version>\t<sensor_name>\t<date>. Example: pressure_v2:1.0\tpressure\t2024-02-01
2. Run the codegen script to update type definitions: python3 scripts/sensors.py
3. Create migration file in migrations/ with naming pattern NNNN_sensor_<name>.go (NNNN = next 4-digit sequence). Must contain func init() { registerSensor(...) }
4. Add entry to TELEMETRY_CHANGELOG.log with format: [YYYY-MM-DD HH:MM:SS] <operator> Added sensor <id>:<version>
5. Verify all changes are staged in git (do not commit).
6. Run make validate to verify sensor registration complete." \
  --source "runbook-vs-skill probe fixture (ADR-0027 C4 procedural-memory evaluation)" \
  --position top

mkdir -p fixture-vaults/runbook
cp -r "$TEMP_VAULT" fixture-vaults/runbook/vault

rm -rf "$TEMP_VAULT" "/tmp/engram-xdg-$$"
```

**Acceptance check:** `fixture-vaults/runbook/vault/` contains one `.md` file with frontmatter (type: runbook, situation, done_when, date, luhmann) and `.vec.json` sidecar.

- [ ] **Step 3: Create fact note for Arm F**

```bash
TEMP_VAULT="/tmp/fact-vault-$$"
mkdir -p "$TEMP_VAULT"

export ENGRAM_VAULT_PATH="$TEMP_VAULT"
export XDG_DATA_HOME="/tmp/engram-xdg-$$"

engram learn fact \
  --slug sensor-registration-semantic \
  --situation "sensor type registration procedures in telemetry systems" \
  --body "Registering a new sensor type in a telemetry system requires six ordered operations. First, the new sensor is declared in a registry file using tab-separated format: identifier, version tuple, human name, and date. Second, a codegen tool is invoked to generate or update type stubs from the registry. Third, a migration file is created following a strict naming pattern (four-digit sequence number prefix, sensor name suffix, Go language). The migration contains initialization code that registers the sensor with the runtime. Fourth, a changelog or audit log is updated with a timestamped entry including the operator name and sensor details. Fifth, all changes are prepared for commit without yet committing (allowing for review). Sixth, a validation tool is run to verify all artifacts exist, have correct formats, and are in consistent state. The steps are order-dependent: registry before codegen, codegen before migration, migration before validation." \
  --source "runbook-vs-skill probe fixture (ADR-0027 C4 semantic-memory evaluation)" \
  --position top

mkdir -p fixture-vaults/fact
cp -r "$TEMP_VAULT" fixture-vaults/fact/vault

rm -rf "$TEMP_VAULT" "/tmp/engram-xdg-$$"
```

**Acceptance check:** `fixture-vaults/fact/vault/` contains one `.md` file with frontmatter (type: fact) and `.vec.json` sidecar.

---

## Task 3: Write the Harness (probe.py) — Tool_Use Parsing & Transcript Analysis

**Files:**
- Create: `dev/eval/cumulative/runbook_vs_skill/probe.py`

**Interfaces:**
- Consumes: Fixture definitions (Task 1), vaults & skill (Task 2)
- Produces: Three JSONL result files (smoke + opus); scored by FOUND/FOLLOWED/END-STATE/COST

**Acceptance:**
- Harness spawns trials with isolated environments (per isolation.py requirements)
- Arm S: Skill is deployed to trial project `.claude/skills/sensors-add-skill/`; verified in transcripts by `"skill":"sensors-add-skill"` JSON field in session .jsonl
- Arm R/F: Fixture vault is set via ENGRAM_VAULT_PATH; verified by Bash tool_use running `engram query` with result containing fixture note path
- FOUND scoring: Parse session .jsonl files under CLAUDE_CONFIG_DIR/projects/ (per underload_repro.py:_count_markers, lines 159–174); search for `"skill":"sensors-add-skill"` (Arm S) or `"tool":"bash"` + `engram query` in tool result (Arm R/F)
- FOLLOWED: Each step verified from transcript tool_use or repo state; scored as k/N where N=6 idiosyncratic steps
- END-STATE: Mechanical check via done_when_checks.sh script
- Validity gate: marker must appear in transcript; marker_seen=false trials discarded before scoring

**Steps:**

- [ ] **Step 1: Write probe.py harness**

Create `probe.py` (comprehensive harness, ~400 lines). See appendix below for full implementation. Key points:
- Trial setup: creates fresh project cwd with `.claude/skills/sensors-add-skill/` for Arm S
- Isolation: per-trial ENGRAM_VAULT_PATH, ENGRAM_CHUNKS_DIR, ENGRAM_TRANSCRIPT_DIR (never operator's real vault)
- Transcript parsing: reads `.jsonl` files under cfg/projects/ (per underload_repro.py pattern)
- FOUND: Arm S searches for `"skill":"sensors-add-skill"` JSON field; Arm R/F search for bash tool running engram query
- FOLLOWED: Parses transcript for 6 key steps (registry edit, codegen run, migration creation, changelog update, git staging, validation command); each step verified from tool_use or stderr
- Validity: marker echoed in at least one transcript line before scoring

---

## Task 4: Create README with Parity & Functionality Decision Frame

**Files:**
- Create: `dev/eval/cumulative/runbook_vs_skill/README.md`

**Interfaces:**
- Produces: Harness documentation, smoke test instructions, pre-registered decision frame (Joe's two questions)

**Acceptance:**
- README explains parity scoring: "indistinguishable" if within 1 trial, "worse" if 2+ below, "better" if 2+ above on FOUND/FOLLOWED/END-STATE
- README explains functionality frame: R exceeds F only if R is 2+ trials ahead on FOLLOWED or END-STATE; otherwise "can't distinguish"
- Smoke test acceptance stated (marker_seen≥3/3, ≥1 end_state=true, cost≤$0.80 sonnet)
- Full run cost estimate as range with source citations (LEDGER row 39–40: $3.08 reference, trials edit files + run tools, budget $0.60–1.00/opus trial)
- Pre-register that Arm S below 3/5 END-STATE makes parity uninterpretable; triggers fixture/spec fix before re-run

**Steps:**

- [ ] **Step 1: Write README**

Create `README.md`:

```markdown
# runbook_vs_skill — Three-Arm Procedural-Memory Eval

Headless measurement of whether a runbook note or fact note in the engram vault can replace a skill in guiding an agent through a multi-step task (fictional sensor-type registration).

## Design

Three arms, identical task (register pressure_v2 sensor), procedure written three ways:
- **Arm S (skill):** Procedure as a SKILL.md in trial project `.claude/skills/sensors-add-skill/` (discovered same way recall skill is)
- **Arm R (runbook):** Procedure as a runbook note in fixture vault (situation, steps, done_when)
- **Arm F (fact):** Procedure as a semantic fact note in fixture vault

All arms receive identical recall guidance (5 cues) inlined into CLAUDE.md + validity marker.
Each trial is fresh `claude -p`, isolated vault (never touches operator's real memory).
Scoring: **FOUND** (tool_use from transcript: Skill for S, Bash `engram query` for R/F), **FOLLOWED** (mean k/6 steps, each verified), **END-STATE** (done_when checks pass mechanically), **COST**.

## Smoke Test (Validation)

Runs 1 trial per arm with sonnet to validate plumbing (marker delivery, skill loading, vault setup, transcript parsing, end-state checks).

```bash
cd dev/eval/cumulative/runbook_vs_skill
python3 probe.py --model sonnet --n 1 --out smoke_results.jsonl
```

Expected: 3 results, all marker_seen=true, ≥1 end_state=true.

**Pre-registered smoke bars:**
- Marker delivery: 3/3 marker_seen=true (validity gate works)
- Plumbing: ≥1/3 end_state=true (at least one arm registers sensor)
- Recall delivery: Arm R transcript contains `"skill":"recall"` JSON field (proves recall+learn skills copied from repo into cfg/skills/, not operator's ~/.claude/)
- Cost: ≤$0.80 total (sonnet baseline, 3 trials)

## Full Run (Paid Opus Trials)

Runs 5 trials per arm with opus.

```bash
cd dev/eval/cumulative/runbook_vs_skill
python3 probe.py --model opus --n 5 --out opus_results.jsonl
```

## Joe's Decision Frame

Two questions, n=5 honest (no rounding; report exact k/5 per arm):

### (a) PARITY: Is runbook indistinguishable from skill?

Per metric (FOUND, FOLLOWED mean k/6, END-STATE):

| Outcome | Definition |
|---------|-----------|
| Indistinguishable | Runbook within 1 trial of skill (e.g., skill 4/5, runbook 3–5/5) |
| Worse | Runbook 2+ below skill (e.g., skill 4/5, runbook ≤2/5) |
| Better | Runbook 2+ above skill (e.g., skill 2/5, runbook ≥4/5) |
| Can't distinguish | Gap ≤1 trial — NOT called "tie" |

**If Arm S END-STATE < 3/5:** Baseline too hard; parity uninterpretable. Triggers fixture/spec fix before re-run.

### (b) FACT > FACT: Does runbook exceed fact functionally?

Runbook exceeds fact ONLY if R is 2+ trials ahead of F on FOLLOWED or END-STATE; otherwise "can't distinguish."

## Cost Estimate

These trials run tools (Bash, Skill invocations) and edit files, unlike endorse_cue's single-response probe.

**Source:** `dev/eval/LEDGER.md` row 39–40 (C1-C2 warm-vs-cold): clean n=8/arm measurement shows "$3.08 costlier" total warm op with tool/file operations. These are tool-rich trials; endorse_cue (single response) is $0.40/opus trial. Scaling from the LEDGER reference (tool-based, file-edit operations), budget **$0.60–1.00 per opus trial**.

**Estimate:**
- Smoke (sonnet, 3 trials): 3 × $0.20 = $0.60
- Full run (opus, 15 trials): 15 × $0.80 = $12.00 (mid-range estimate)
- **Range: $12.60–$22.00** (smoke + full at $0.60–1.00/opus trial)
- **Smoke separately:** $0.60 (commit-point after validation)
- **Full run separately:** $12.00–15.00 (commit after results aggregated)

## Fixture & Vault Isolation

Each trial gets:
- `.claude/skills/sensors-add-skill/` (trial project)
- `ENGRAM_VAULT_PATH=/tmp/trial-<uuid>/vault` (per-trial, never operator's real vault)
- `ENGRAM_CHUNKS_DIR=/tmp/trial-<uuid>/chunks`
- `ENGRAM_TRANSCRIPT_DIR=/tmp/trial-<uuid>/transcripts`
- `CLAUDE_CONFIG_DIR=/tmp/runbook-vs-skill-probe/cfg` (shared cold config)

Validity gate: unique marker in CLAUDE.md; trials without marker in transcript are discarded (marker_seen=false, never scored 0). Verify by reading session .jsonl under cfg/projects/.

## Reuse Notes

- Probe design is easily adapted for other skills/runbooks: only fixture repo, task scenario, done_when checks, and procedure encodings are task-specific.
- Fixture repo template (sensor system with idiosyncratic registry/migration/ledger format) can seed other procedural-memory evals.
- FOUND/FOLLOWED/END-STATE/COST scoring framework reusable; decision frame (parity + functionality questions) generalizable to other note-vs-skill comparisons.
```

**Acceptance check:** README exists; parity frame explained (1-trial gap is indistinguishable, 2+ is worse/better); functionality frame explained (2+ advantage only); cost as range with source; Arm S <3/5 baseline stated as uninterpretable.

---

## Task 5: Smoke Test Execution & Validation

**Files:** (None created; runs probe.py)

**Acceptance:**
- Smoke test runs without error
- All 3 trials marker_seen=true
- At least 1 trial end_state=true
- Spend ≤$0.80
- Transcripts show Arm S skill tool_use, Arms R/F engram query bash tool_use

**Steps:**

- [ ] **Step 1: Run smoke test**

```bash
cd dev/eval/cumulative/runbook_vs_skill
python3 probe.py --model sonnet --n 1 --out smoke_results.jsonl --workers 1
```

- [ ] **Step 2: Validate results**

```bash
jq -s '[.[] | select(.marker_seen == true)] | length' smoke_results.jsonl
# Expected: 3

jq -s '[.[] | select(.end_state == true)] | length' smoke_results.jsonl
# Expected: ≥1

jq -s 'map(.cost) | add' smoke_results.jsonl
# Expected: ≤$0.80
```

---

## Task 6: Full Opus Run & Results Aggregation

**Files:** (None created; results only)

**Acceptance:**
- All 15 trials marker_seen=true
- Results reported against parity frame (gap ≤1 = indistinguishable, 2+ = worse/better)
- Functionality frame applied (R exceeds F only if 2+ ahead)
- Spend $12–15 (within budget range)
- Decision stated: can distinguish or can't distinguish (never "tie")

**Steps:**

- [ ] **Step 1: Run full opus evaluation**

```bash
cd dev/eval/cumulative/runbook_vs_skill
python3 probe.py --model opus --n 5 --out opus_results.jsonl
```

- [ ] **Step 2: Aggregate results and report**

Script to produce summary per parity frame (see README for exact table format).

---

## Appendix: Probe.py Implementation Sketch

Key functions:
- `_parse_transcripts(cfg)` — scan .jsonl files under cfg/projects/ for skill/query markers (per underload_repro.py:_count_markers)
- `score_found(arm, transcripts)` — Arm S: `"skill":"sensors-add-skill"` in JSON; Arm R/F: `"tool":"bash"` with engram query + note path in result
- `score_followed(arm, transcripts, repo_path)` — parse for 6 steps; verify each from tool_use or repo state
- `check_end_state(repo_path)` — run done_when_checks.sh
- `run_one(arm, ...)` — spawn one trial: clone fixture repo, deploy skill (Arm S), set vault (Arm R/F), run claude -p, parse transcripts, score, cleanup

---

**Deliverables Checklist:**

- [ ] PLAN.md exists with all 7 blocking changes applied
- [ ] Item 1: Arm S is a REAL skill in `.claude/skills/`; cite file:line for cold_cfg structure; explain skill discovery mechanism; verify via `"skill":"sensors-add-skill"` in transcripts
- [ ] Item 2: FOUND scored from tool_use (JSON field in .jsonl); cite underload_repro.py:_count_markers for mechanism; Arm S = skill tool_use, Arm R/F = bash + engram query + note path in result
- [ ] Item 3: Fifth cue rewritten to avoid task-echo; general task-start situation, multiple recurring routine examples, no fixture-specific keywords
- [ ] Item 4: Fixture procedure changed to sensor registration (6 steps: registry, codegen, migration, changelog, git stage, validate); note-505 hygiene enforced
- [ ] Item 5: FOLLOWED scored separately from END-STATE; FOLLOWED = k/6 steps, END-STATE = script passes
- [ ] Item 6: Parity frame replaced (indistinguishable if within 1, worse if 2+ below, better if 2+ above); functionality frame (R 2+ ahead of F only); Arm S <3/5 baseline triggers retest
- [ ] Item 7: Cost as range $12.60–$22.00 with source (LEDGER row 39–40); $0.60–1.00/opus trial budget cited
- [ ] All mechanism claims cite file:line from actual code

LESSONS: Tool-use parsing from .jsonl transcripts (not text) requires reading session files under cfg/projects/; cold CLAUDE_CONFIG_DIR has NO skills by default, so Arm S skill must be in trial project `.claude/skills/`; fixture procedure must avoid echo-matching the recall cue (sensor registration ≠ "sensor registration" routine example).
