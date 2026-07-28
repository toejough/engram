# Symlink-discovery verification verdicts (tasks 1.1–1.3)

Probed 2026-07-27 via isolated homes (copied/keychain-extracted credentials, deleted after; real harness dirs untouched). Control-pair discipline: each harness ran with a real-dir control skill (`zzqctl`) AND a symlinked treatment skill (`zzqlnk` → a directory outside the skills dir); a symlink verdict counts only because the control was discovered in the same run.

| Harness | Control (real dir) | Skill via symlink | Command via symlink | Guidance via symlink | Deploy mode |
|---|---|---|---|---|---|
| Claude Code (headless `-p`, haiku, `CLAUDE_CONFIG_DIR` isolation) | discovered | **discovered** | n/a (none deployed) | readable (direct read; not a live `@import` run) | **symlink** |
| OpenCode 1.16.2 (`opencode run`, XDG+HOME isolation, `opencode/claude-haiku-4-5`) | discovered | **discovered** (frontmatter descriptions read through the link — SKILL.md content, not just the dir name) | **executes** (`--command` runs the symlinked file's body; missing-command baseline errors, a clean discriminator) | n/a (none deployed) | **symlink** |
| Pi v0.82.x (headless, HOME isolation, free local LM Studio model) | discovered | **discovered** | n/a (none deployed) | readable through symlink | **symlink** |

**Verdict for D7:** all three harnesses run in **symlink** deploy mode. The manifest fallback (task 6.1) ships as the spec-required capability but no current harness needs it.

Caveats carried forward: single-run (n=1) per harness at the versions above; Claude guidance verified by file-read semantics rather than a live import chain; symlink targets were same-volume. Full probe transcripts: workflow `wf_74fd0963-98d` journal.
