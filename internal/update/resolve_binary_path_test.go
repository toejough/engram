package update_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// TestResolveBinaryPath covers all three of resolveBinaryPath's resolution
// steps, in the go toolchain's own priority order: GOBIN, then GOPATH/bin,
// then ~/go/bin.
func TestResolveBinaryPath(t *testing.T) {
	t.Parallel()

	table := []struct {
		name string
		env  envVars
		want string
	}{
		{
			name: "GOBIN set wins",
			env:  envVars{"GOBIN": "/custom/gobin", "GOPATH": "/custom/gopath"},
			want: "/custom/gobin/engram",
		},
		{
			name: "GOPATH set, no GOBIN",
			env:  envVars{"GOPATH": "/custom/gopath"},
			want: "/custom/gopath/bin/engram",
		},
		{
			name: "neither set falls back to home/go/bin",
			env:  envVars{},
			want: "/home/joe/go/bin/engram",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := update.ExportResolveBinaryPath("/home/joe", tc.env)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// envVars is a minimal update.Env test double keyed purely by env var name
// — resolveBinaryPath is the only user of GOBIN/GOPATH in this package, and
// none of the other Env-consuming code paths this repo already tests need
// those two vars, so a tiny dedicated fake (rather than extending the
// shared fakeEnv) keeps this test isolated from unrelated behavior.
type envVars map[string]string

func (e envVars) Getenv(key string) string { return e[key] }

func (e envVars) Getwd() (string, error) { return "", nil }

func (e envVars) UserHomeDir() (string, error) { return e["HOME"], nil }
