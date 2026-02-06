# Summary: state-machine-improvements

## Problem Solved

State machine precondition checks (like `TestsExist`) were looking in the project directory (e.g., `.claude/projects/path-fixes/`) for code artifacts, but tests actually live in the repository root. This required using `--force` to bypass preconditions during TDD phases.

## Solution Implemented

Added `RepoDir` field to track the repository root separately from the project directory:

1. **RepoDir in state** - `Project` struct now has `repo_dir` field persisted to state.toml
2. **FindRepoRoot utility** - Detects git repository root via `git rev-parse --show-toplevel`
3. **CLI integration** - `projctl state init` accepts `--repo-dir` flag with auto-detection fallback
4. **Wiring to preconditions** - `TransitionWithChecker` populates `RepoDir` from state and passes it to code-related preconditions

## Key Files Changed

- `internal/state/state.go` - Added RepoDir to Project struct and InitOpts/TransitionOpts
- `internal/state/repo.go` - New file with FindRepoRoot function
- `cmd/projctl/state.go` - Added --repo-dir flag with auto-detection
- `internal/state/state_integration_test.go` - Integration test validating TDD cycle

## Testing

- Unit tests for RepoDir persistence and FindRepoRoot
- Integration test creates temp git repo, runs full TDD transition cycle
- Verified tdd-green succeeds when tests exist in repo dir (not project dir)

## Closes

- ISSUE-38: State machine should track repo dir separately from project dir
