# Task 7.2 real-binary verification (U3, 2026-10-01)

Per ruling V5: no `go install`. Built to a scratch `$S/bin/engram` and ran every command
from a cwd outside any repo (`/tmp`), with `XDG_DATA_HOME=$S/data` and either `--vault $S/vault`
or `ENGRAM_VAULT_PATH=$S/vault` (the flag `engram update` itself doesn't take). The real vault
(`~/.local/share/engram/vault`) and `~/.claude` were never referenced by any of these commands.

```
S=$(mktemp -d)
mkdir -p "$S/bin" "$S/data" "$S/vault" "$S/home"
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill
go build -o "$S/bin/engram" ./cmd/engram   # exit 0
```

`targ test` and `targ check-full` were used for all Go test/lint/check operations elsewhere in
this unit; this section is the real-binary scratch-vault check task 7.2 asks for separately.

## #772 — show never truncates; query's marker names a count and the real basename

```
$ENGRAM learn runbook --vault "$S/vault" --position top --slug ccells-772-verify \
    --source u3-verification-772 \
    --situation "verifying show never truncates red_flags for task 772" \
    --done-when "all 15 red_flags are present with no omission marker" \
    --red-flag "verification red flag number 01 ..." ... (15 total, each long enough that
    15 together exceed the 1200-byte preview budget)
-> /…/vault/1.2026-10-01.ccells-772-verify.md
```

`engram show 1.2026-10-01.ccells-772-verify`: all 15 `red_flags` entries present, 0 `OMITTED`
occurrences (`grep -c "verification red flag number"` = 15, `grep -c OMITTED` = 0).

`engram query --phrase "verifying show never truncates red_flags for task 772"` returned the note
with:

```yaml
red_flags:
    - "[10 EARLIER RED_FLAGS OMITTED — run engram show 1.2026-10-01.ccells-772-verify for all 15]"
    - verification red flag number 11 ...
    - verification red flag number 12 ...
    - verification red flag number 13 ...
    - verification red flag number 14 ...
    - verification red flag number 15 ...
```

The marker names the count (10 omitted), the total (15), and the real basename. Running the
marker's exact command (`engram show 1.2026-10-01.ccells-772-verify`) again prints all 15 entries
and no marker — confirmed by the grep counts above. **PASS.**

## #776 — amend never blanks `vault:`, and the OS-username fallback masks the empty-user case on macOS

```
$ env -u ENGRAM_VAULT_NAME $ENGRAM amend --vault "$S/vault" \
    --target 1.2026-10-01.ccells-772-verify --object x
```

Resulting note: `vault: personal` (never empty), with neither `--vault-name` nor
`ENGRAM_VAULT_NAME` set. **PASS.**

For the user case: hand-set `user: alice` on the note, then:

```
$ GIT_CONFIG_GLOBAL=/dev/null HOME="$S/home" $ENGRAM amend --vault "$S/vault" \
    --target 1.2026-10-01.ccells-772-verify --object y
```

Result: `user: joe` (the real OS account), not `user: alice` and not `user: ""`. As task 7.2
anticipates: on macOS `os/user.Current()` still resolves (`whoami` → `joe`) even with git's
global config pointed at `/dev/null`, so true empty-string detection is **not reachable from the
real binary** on this machine — the OS-username fallback always succeeds. The empty-detection
path (`DetectUser` returning `""`, `user: alice` kept, `LogWarning` fired once) is exercised only
by the mocked unit test in `amend_identity_test.go` (task 2.1), which is the evidence for that
branch. **Recorded, as instructed — not a binary-level failure.**

## #770 — the derive payload defers to learn's batch mode

Created two near-duplicate top-level fact notes (`2.*.dup-note-a`, `3.*.dup-note-b`), then:

```
$ ENGRAM_VAULT_PATH="$S/vault" $ENGRAM update --reparent-luhmann
```

```json
{
  "candidates": [ {"note": "2", "target": "3", "similarity": 0.9852609}, ... ],
  "instruction": "Hand this payload to the learn skill's batch mode (\"Batch mode — Luhmann re-eval answers\"), which holds the continuation (a deeper elaboration of target) / sibling (a related but distinct point next to target) / top (unrelated — no change) disposition test. Write an answers file: {...}, exactly one entry per distinct candidate \"note\". Hand the finished answers file back to the user or orchestrating agent: preview it first with `next_command` plus --dry-run, then apply it by running `next_command` again without --dry-run. The answering pass itself does not run apply.",
  "next_command": "engram update --reparent-luhmann --answers <path-to-answers-file>",
  "fingerprint": "n3-2d0760d3369758e63e724753b911f6a379883017fe08c428a03aecf1c3bc6387"
}
```

Names "learn skill's batch mode", `--dry-run`, "exactly one entry per distinct candidate", and
never tells the answering pass to run apply. **PASS.**

## #780(a) — anchored candidate is refused untouched

Hand-wrote `9.2026-10-01.ccells-anchor-candidate.md` with `luhmann: &a "9"` / `issue: *a`, took a
pre-apply `ls $S/vault` snapshot, derived a fresh fingerprint, and answered `{"note":"9",
"position":"continuation","target":"3"}`:

```
$ ENGRAM_VAULT_PATH="$S/vault" $ENGRAM update --reparent-luhmann --answers answers.json --dry-run
update: update --reparent-luhmann (dry-run): rename: rewritten frontmatter does not decode: 9.2026-10-01.ccells-anchor-candidate: parsing frontmatter: yaml: unknown anchor 'a' referenced

$ ENGRAM_VAULT_PATH="$S/vault" $ENGRAM update --reparent-luhmann --answers answers.json
update: update --reparent-luhmann: applying renames: rename: rewritten frontmatter does not decode: 9.2026-10-01.ccells-anchor-candidate: parsing frontmatter: yaml: unknown anchor 'a' referenced
```

Both the `--dry-run` pre-flight and the real apply refuse, naming the note; `ls $S/vault` before
and after are byte-identical (`diff` empty). **PASS.**

## #780(b) — CRLF candidate + CRLF referrer convert to LF; untouched CRLF note is unaffected

Built, via `engram learn` (LF) then a byte-level `\n` → `\r\n` conversion on disk (simulating a
hand-/externally-authored CRLF note — the same thing the spec's CRLF scenarios describe):

- `4.2026-10-01.ccells-crlf-candidate.md` (CRLF, to be renamed)
- `5.2026-10-01.ccells-crlf-referrer.md` (CRLF, `supersedes:` names note 4)
- `6.2026-10-01.ccells-crlf-untouched.md` (CRLF, touched by nothing)

```
$ ENGRAM_VAULT_PATH="$S/vault" $ENGRAM update --reparent-luhmann --answers answers.json --dry-run
update --reparent-luhmann (dry-run): would rename
  4.2026-10-01.ccells-crlf-candidate -> 3a.2026-10-01.ccells-crlf-candidate
  references rewritten in: 5.2026-10-01.ccells-crlf-referrer.md

$ ENGRAM_VAULT_PATH="$S/vault" $ENGRAM update --reparent-luhmann --answers answers.json
update --reparent-luhmann: renamed 1 note(s)
...
update --reparent-luhmann: 2 further candidate(s) found — run `engram update --reparent-luhmann` again
```

Afterwards:
- `3a.2026-10-01.ccells-crlf-candidate.md`: 0 `\r` bytes (`grep -c $'\r'` = 0); `luhmann: "3a"`.
- `5.2026-10-01.ccells-crlf-referrer.md`: 0 `\r` bytes; `supersedes: - note:
  3a.2026-10-01.ccells-crlf-candidate` (rewritten to the new basename).
- `6.2026-10-01.ccells-crlf-untouched.md`: still 16 `\r` occurrences, and its sha256 is
  byte-for-byte identical before and after the apply
  (`5e3dc19926ff15761c8d0f41a28672176cbd13be2beca6798a8fb41636afdb73`).
- `engram embed status`: `total: 6, with-embeddings: 5, without: 0, stale: 1, incompatible: 0,
  broken: 0`. The one stale entry is note 6 (`engram embed apply --stale --dry-run` names it) —
  **this is a harness artifact, not a regression**: note 6's sidecar was embedded when the file
  was still LF (via `engram learn`), and it was then converted to CRLF on disk *outside* engram
  (bypassing embed-on-write entirely) to simulate "a pre-existing CRLF note", which is exactly
  the kind of out-of-band edit that always leaves a sidecar stale regardless of this change. The
  CRLF notes the apply *did* touch (3a, 5) both show `embed status` fresh (confirmed via
  `md_mtime == vec_mtime`), matching the spec's "every converted note has its sidecar rebuilt"
  guarantee. **PASS** (requirement scenarios: CRLF renamed note converted+rewritten, CRLF referrer
  rewritten, untouched CRLF note stays byte-identical, sidecar rebuilt for renamed/rewritten
  notes).

## #780(c) — adopt and resituate keep unmodeled keys

Adopt: wrote `~/.claude`-equivalent fixture `$S/home/.claude/skills/ccells-adopt/SKILL.md`, a
runbook note `7.2026-10-01.ccells-adopt-target.md` with a hand-added `luhmann_old: "12"`, then:

```
$ HOME="$S/home" $ENGRAM register-skills --vault "$S/vault" --adopt "claude:ccells-adopt=7"
-> /…/vault/7.2026-10-01.skill-claude-ccells-adopt.md
```

The renamed note still carries `luhmann_old: "12"` alongside the new `skill_hash`, `skill_key`,
`skill_source` and `aliases: [7.2026-10-01.ccells-adopt-target]`. **PASS.**

Resituate: wrote a fact note `8.2026-10-01.ccells-resituate-target.md` with a hand-added
`luhmann_old: "13"`, then:

```
$ $ENGRAM resituate --vault "$S/vault" --note 8 \
    --situation "verifying resituate preserves an unmodeled key, now resituated"
```

The note's `situation:` changed and `luhmann_old: "13"` survived unchanged. **PASS.**

### Extra check (per U3's brief): resituate's anchored-key refusal

Hand-wrote a fact note `14.2026-10-01.ccells-resituate-anchor.md` with `situation: &s "..."` and
`object: *s`:

```
$ $ENGRAM resituate --vault "$S/vault" --note 14 --situation "a new situation that would orphan the alias"
resituate: frontmatter key the edit replaces carries a YAML anchor: situation
```

Exit 1; the note's sha256 is identical before and after
(`98aa795504f8f17d640499de0d89a0109610e1eacd35d381abc0ca3c2527b40a`). **PASS.** (Adopt and
refresh share the same `applySkillNoteBody`/`refuseAnchoredKeys` code path exercised indirectly
by every other adopt check above; a dedicated refresh anchor trial was not run separately since
task 7.2 names only reparent-apply and resituate for the real-binary pass, and refresh's refusal
is unit-tested directly in `skillreg_accept_test.go`.)

## Real-vault confirmation

```
$ git -C ~/.local/share/engram/vault status --short
?? 1075a.2026-09-30.sandbox-write-probe-must-reach-seatbelt.md
?? 1075a.2026-09-30.sandbox-write-probe-must-reach-seatbelt.vec.json
?? 1075b.2026-10-01.state-confinement-in-every-dispatch-that-may-run-headless-claude.md
?? 1075b.2026-10-01.state-confinement-in-every-dispatch-that-may-run-headless-claude.vec.json
?? 1080a.2026-10-01.check-what-a-fixture-actually-seeded-before-citing-it-as-evidence.md
?? 1080a.2026-10-01.check-what-a-fixture-actually-seeded-before-citing-it-as-evidence.vec.json
?? 1081.2026-09-29.llm-match-judge-over-large-candidate-lists-over-matches.md
?? 1081.2026-09-29.llm-match-judge-over-large-candidate-lists-over-matches.vec.json
?? 1082.2026-09-29.openspec-validate-does-not-check-modified-targets-exist.md
?? 1082.2026-09-29.openspec-validate-does-not-check-modified-targets-exist.vec.json
?? 1083.2026-09-30.claude-p-stream-json-omits-subagent-tool-calls.md
?? 1083.2026-09-30.claude-p-stream-json-omits-subagent-tool-calls.vec.json
?? 1084.2026-09-30.agent-definition-disallowedtools-pattern-removes-whole-tool.md
?? 1084.2026-09-30.agent-definition-disallowedtools-pattern-removes-whole-tool.vec.json
```

**Concern:** task 7.2 named 1075a, 1080a, 1081, 1082, 1083 (plus sidecars) as the only expected
uncommitted files. Two more are present — `1075b` and `1084` (plus sidecars) — that weren't on
that list. None of this U3 work ever invoked `engram` without an explicit `--vault`/
`ENGRAM_VAULT_PATH` pointing at the scratch vault `$S/vault` (every command above is pasted in
full), so these two extra notes were not written by this session. They are most likely later
controller/concurrent-session lesson captures made after the ruling's note list was written.
Flagging for the controller rather than acting on it — the real vault is explicitly off-limits to
touch or reconcile from here.

## Scratch cleanup

`$S` (the scratch `mktemp -d` root: binary, vault, data, home) was removed after this section was
written; nothing under it was ever referenced by any committed artifact.

## Summary

| Check | Result |
|---|---|
| #772 show never truncates; query marker has count+basename; marker command returns all 15 | PASS |
| #776 vault: never blank | PASS |
| #776 user: empty-detection kept | Not reachable from real binary on macOS (OS-user fallback always resolves); covered by `amend_identity_test.go` (2.1) |
| #770 instruction names learn batch mode, `--dry-run`, one-entry-per-candidate, no apply | PASS |
| #780(a) anchored candidate refused, vault unchanged | PASS |
| #780(b) CRLF candidate+referrer convert to LF, luhmann/supersedes rewritten, sidecars rebuilt | PASS |
| #780(b) untouched CRLF note stays byte-identical | PASS (sha256 match) |
| #780(c) adopt keeps `luhmann_old` | PASS |
| #780(c) resituate keeps `luhmann_old` | PASS |
| resituate refuses an anchored key (extra check) | PASS |
| Real vault unchanged by this session | PASS, with 2 untracked notes (1075b, 1084) beyond the ruling's list — not from this session, flagged for the controller |

# Task 7.2 re-run for amend and resituate (final-review fix wave, ruling V6, 2026-10-01)

After the final-review fixes (amend moved onto the YAML-node edit; refresh and resituate convert CRLF),
the amend and resituate checks were re-run against a fresh scratch build of the working tree on top of
`49cfc120`. As before, nothing was `go install`ed. Every command ran from `/tmp` under
`env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data`, with `--vault $S/vault`, so
`ENGRAM_VAULT_NAME` and `ENGRAM_PARENT` were unset. No command named the real vault or `~/.claude`.

```
S=$(mktemp -d …/scratchpad/v72.XXXX); mkdir -p $S/bin $S/data $S/vault $S/home
go build -o $S/bin/engram ./cmd/engram   # exit 0
```

The fixtures were hand-written fact notes (`user: alice`, `vault: personal`, quoted `created:`).

| # | Check | Command | Result |
|---|---|---|---|
| A1 | Content amend keeps unmodeled keys; `vault:` defaults | `engram amend --target 1 --object sprocket` on a note with `luhmann_old: "12"` and `provenance: {origin: import, steps: [a, b]}` | exit 0. `object: sprocket`; `luhmann_old: "12"` and the whole `provenance` map intact; `vault: personal`. `user:` re-stamped to the OS user (`joe`), as expected on macOS. **PASS** |
| A2 | An anchor on an unedited key survives | `engram amend --target 2 --object sprocket` on a note with `source: &src test` and `x_src: *src` | exit 0. `source: &src test` and `x_src: *src` are byte-preserved. **PASS** |
| A3 | An anchored edited key is refused untouched | `engram amend --target 3 --object sprocket` on a note with `object: &o gear` and `x_obj: *o` | exit 1: `amend: frontmatter key the edit replaces carries a YAML anchor: object`. sha256 `cd577c4a…5c9d23` is the same before and after. **PASS** |
| A4 | A supersedes-only amend keeps unmodeled keys | `engram amend --target 4 --supersedes "1.2026-10-01.amend-unknown\|narrows\|old claim"` | exit 0. `supersedes:` was added; `luhmann_old: "14"` is intact. **PASS** |
| R1 | Resituate converts a CRLF note and keeps unmodeled keys | `engram resituate --note 5 --situation "re-verifying resituate"` on an all-CRLF note (16 CR lines) with `luhmann_old: "15"` | exit 0. 0 CR lines after; `situation:` changed; `luhmann_old: "15"` is intact. **PASS** |
| E | Sidecars | `engram embed status` | `stale: 0`, `broken: 0`. 3 with embeddings: the three content writes. The 2 without are the refused note and the supersedes-only note, which amend does not re-embed (provenance-only, D3). **PASS** |

Exchange-hash parity with the pre-change amend:
- A binary built from `49cfc120` (amend unchanged there) amended the same 26 inputs that need no chunk index (fact, feedback and runbook; content, supersedes, red-flags, triggers, body and clear-pending kinds).
- `engram show`'s `# exchange_hash:` matched the committed goldens on all 26. The 6 `--chunk-source` cases are pinned in-process only.
- The new code reproduces every golden byte-for-byte (`TestRunAmend_ExchangeHashParityWithPreChangeAmend`, 32 cases).

The scratch root `$S` was removed afterwards.
