# Coding Standards

## Dependency Injection

No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O flows through injected interfaces. Wire dependencies at the edges (`internal/cli`, `cmd/engram/main.go`).

Test hard-to-test code by refactoring for DI, not by writing integration tests around I/O.

## Pure Go, No CGO

TF-IDF and BM25 for text similarity. External embedding API if vector similarity is needed. No CGO dependencies.

## TDD for All Changes

Full red/green/refactor cycle for every artifact change:

1. **Red**: Write a failing test first.
2. **Green**: Write the minimum code to pass.
3. **Refactor**: Clean up while tests stay green.

Never skip phases.

## Build System

Use `targ` for all build/test/check operations. Never run `go test`, `go vet`, or `go build` directly.

```bash
targ test          # Run tests
targ check-full    # Lint + coverage (gets ALL errors at once)
targ check         # Stops early -- avoid for comprehensive checks
targ build         # Build binary
```

## Test Categorization

**Unit tests** verify business logic via DI + mocks (imptest pattern). These are the default.

**Integration tests** verify wiring of thin I/O wrappers with real dependencies.

If a function has business logic AND I/O, refactor to separate them -- don't write an integration test around the whole thing.

## Test Conventions

- `t.Parallel()` on every test and subtest.
- No shared mutable state between parallel subtests -- each gets its own data.
- Blackbox tests use `package foo_test`.
- Entry points (`main.go`, root module files) are excluded from coverage.

## Nilaway + Gomega Compatibility

nilaway doesn't recognize gomega assertions as nil guards. Required patterns:

```go
// After asserting no error, add explicit nil guard
g.Expect(err).NotTo(HaveOccurred())
if err != nil { return }

// Use MatchError instead of err.Error()
g.Expect(err).To(MatchError("expected message"))

// Nil-check pointers after asserting
g.Expect(result).NotTo(BeNil())
if result == nil { return }
```

## Code Style

- **No magic numbers.** Name constants: `const maxRetries = 3`.
- **Descriptive variable names.** `memory`, `pattern`, `score` -- not `m`, `p`, `s`.
- **Wrap errors with context.** `fmt.Errorf("finding similar: %w", err)`.
- **Sentinel errors.** `var ErrNotFound = errors.New(...)` not inline `fmt.Errorf(...)`.
- **Line length under 120 characters.**
- **Use `crypto/rand`** not `math/rand`.
- **Use `http.NewRequestWithContext`** not `http.Get`.
- **Use `make([]T, 0, capacity)`** when size is known.

## Directory Structure

```
engram/
├── engram.go           # Public API at root
├── internal/           # Non-public implementation
├── cmd/engram/         # CLI binary entry point
├── hooks/              # Claude Code hook scripts
├── skills/             # Interactive skill definitions
└── dev/                # Build tooling
```

Root = public API, `internal/` = implementation. Shallow nesting (2-3 levels max).
