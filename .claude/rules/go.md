---
globs: "*.go"
---

## Nilaway + Gomega Compatibility

nilaway doesn't recognize gomega assertions as nil guards. Required patterns:
- After `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` before accessing values
- Use `g.Expect(err).To(MatchError(...))` instead of `err.Error()`
- Add explicit nil guards before field access on pointers
- For test helpers returning `(*T, error)`, nil-check the pointer after asserting no error

## Known Model Defaults to Avoid

Generate code that passes linters on first commit:
- Name constants instead of magic numbers: `const maxRetries = 3`, not bare `3`
- Descriptive variable names: `memory`, `pattern`, `score` — not `m`, `p`, `s`
- Wrap errors with context: `fmt.Errorf("finding similar: %w", err)` not bare `return err`
- Use sentinel errors: `var ErrNotFound = errors.New(...)` not inline `fmt.Errorf(...)`
- Add `t.Parallel()` to every test and subtest (with no shared mutable state)
- Use `http.NewRequestWithContext` not `http.Get`
- Use `crypto/rand` not `math/rand`
- Line length under 120 chars
