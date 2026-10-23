package update_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// goEnvKey is the command line of the one go env call the resolver makes.
const goEnvKey = "go env GOBIN GOPATH"

// Test792b28c1a1_ResolveBinaryPath covers all three of resolveBinaryPath's resolution
// steps, in the go toolchain's own priority order: GOBIN, then GOPATH/bin,
// then ~/go/bin.
func Test792b28c1a1_ResolveBinaryPath(t *testing.T) {
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

			got := update.ExportResolveBinaryPath(t.Context(), "/home/joe", tc.env, &fakeCmd{})
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestResolveBinaryPath_GoEnvValues covers the values the go toolchain reports: its GOBIN wins over its GOPATH, the
// first GOPATH entry is used, and the environment answers when the toolchain reports nothing or fails.
func TestResolveBinaryPath_GoEnvValues(t *testing.T) {
	t.Parallel()

	table := []struct {
		name string
		cmd  *fakeCmd
		env  envVars
		want string
	}{
		{
			name: "go env GOBIN wins over go env GOPATH",
			cmd:  &fakeCmd{responses: map[string][]byte{goEnvKey: []byte("/gb\n/gp\n")}},
			env:  envVars{},
			want: "/gb/engram",
		},
		{
			name: "go env GOPATH, first entry only",
			cmd:  &fakeCmd{responses: map[string][]byte{goEnvKey: []byte("\n/gp1:/gp2\n")}},
			env:  envVars{},
			want: "/gp1/bin/engram",
		},
		{
			name: "go env reports nothing, the environment answers",
			cmd:  &fakeCmd{responses: map[string][]byte{goEnvKey: []byte("\n\n")}},
			env:  envVars{"GOBIN": "/eb"},
			want: "/eb/engram",
		},
		{
			name: "go env fails, the environment answers",
			cmd:  &fakeCmd{errs: map[string]error{goEnvKey: update.ErrGoNotFound}},
			env:  envVars{"GOPATH": "/ep"},
			want: "/ep/bin/engram",
		},
		{
			name: "go env fails, nothing set, the home directory answers",
			cmd:  &fakeCmd{errs: map[string]error{goEnvKey: update.ErrGoNotFound}},
			env:  envVars{},
			want: "/home/joe/go/bin/engram",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := update.ExportResolveBinaryPath(t.Context(), "/home/joe", tc.env, tc.cmd)
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
