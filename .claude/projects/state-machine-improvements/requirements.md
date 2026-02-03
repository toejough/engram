# Requirements: state-machine-improvements

## REQ-001: State tracks repo directory

The state machine must track the repository root directory separately from the project directory, so precondition checks can locate code artifacts (tests, source files) in the correct location.

**Rationale:** Currently preconditions like `TestsExist` look in the project directory (`.claude/projects/<name>/`), but tests live in the repo source tree.

**Traces to:** ISSUE-038

---

## REQ-002: Auto-detect repo root

When initializing a project, the system should auto-detect the git repository root if `--repo-dir` is not explicitly provided.

**Rationale:** Reduces friction - users shouldn't need to specify repo root for typical single-repo workflows.

**Traces to:** ISSUE-038

---

## REQ-003: Preconditions use appropriate directory

Precondition checks must use the correct directory for each artifact type:
- Code artifacts (tests, source): repo directory
- Planning artifacts (requirements, design, tasks, AC): project directory

**Rationale:** This is the core fix that enables TDD phase transitions to work without `--force`.

**Traces to:** ISSUE-038

---

## REQ-004: Integration test for state task tracking

An integration test must verify that the state machine correctly tracks task completion across the full TDD cycle.

**Rationale:** Ensures task tracking works end-to-end, not just in unit tests.

**Traces to:** ISSUE-032

---

## REQ-005: Backward compatibility

Existing state.toml files without `repo_dir` should continue to work, defaulting to auto-detection behavior.

**Rationale:** Don't break existing projects.

**Traces to:** ISSUE-038
