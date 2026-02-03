# Retrospective: state-machine-improvements

## What Went Well

**W1: Clean task decomposition**
The 5-task breakdown (RepoDir field → FindRepoRoot → CLI flag → wire to preconditions → integration test) allowed parallel development of TASK-001 and TASK-002, followed by clear dependency flow.

**W2: Integration test caught real behavior**
The integration test validates the full TDD cycle with actual git repo creation, proving the fix works end-to-end rather than just unit test coverage.

**W3: macOS symlink handling**
Discovered and handled `/var` → `/private/var` symlink issue on macOS using `filepath.EvalSymlinks` in tests.

## What Could Be Improved

**I1: Path change ripple effects**
The earlier path-fixes project changed default DocsDir to empty string, which broke coverage tests. These should have been caught and fixed in that project, not discovered later.

**I2: No explicit test for TestsFail precondition**
The integration test validates TestsExist but relies on a mock for TestsFail. A more thorough integration test would actually run failing tests.

## Action Items

- [ ] Consider adding pre-commit hooks to run full test suite (not just affected packages)
- [ ] Future path changes should grep for hardcoded path strings in test files

## Process Observations

The state machine transitions work correctly with the new repo dir separation. The key insight is that project artifacts (requirements, design, tasks) belong in the project dir while code artifacts (tests, implementation) belong in the repo dir.
