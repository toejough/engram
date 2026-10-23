package update_test

import (
	"context"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

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

			got := update.ExportResolveBinaryPath(t.Context(), "/home/joe", tc.env, nil)
			g.Expect(got).To(Equal(tc.want))
		})
	}
}

// TestCommanderGoEnv_Values covers how the lines `go env` prints become one value per name asked, and that a failed
// command is an error.
func TestCommanderGoEnv_Values(t *testing.T) {
	t.Parallel()

	table := []struct {
		name   string
		stdout string
		want   []string
	}{
		{name: "both set", stdout: "/gb\n/gp\n", want: []string{"/gb", "/gp"}},
		{name: "the first unset", stdout: "\n/gp1:/gp2\n", want: []string{"", "/gp1:/gp2"}},
		{name: "windows line ends", stdout: "/gb\r\n/gp\r\n", want: []string{"/gb", "/gp"}},
		{name: "a line missing", stdout: "/gb\n", want: []string{"/gb", ""}},
		{name: "no output", stdout: "", want: []string{"", ""}},
		{name: "extra lines are ignored", stdout: "/gb\n/gp\n/extra\n", want: []string{"/gb", "/gp"}},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			cmd := &fakeCmd{responses: map[string][]byte{"go env GOBIN GOPATH": []byte(tc.stdout)}}

			got, err := update.CommanderGoEnv{Cmd: cmd}.Values(t.Context(), "GOBIN", "GOPATH")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tc.want))
			g.Expect(cmd.calls).To(Equal([][]string{{"go", "env", "GOBIN", "GOPATH"}}))
			g.Expect(cmd.dirs).To(Equal([]string{""}))
		})
	}

	t.Run("a failed command is an error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cmd := &fakeCmd{errs: map[string]error{"go env GOBIN GOPATH": update.ErrCommandNotFound}}

		_, err := update.CommanderGoEnv{Cmd: cmd}.Values(t.Context(), "GOBIN", "GOPATH")
		g.Expect(err).To(MatchError(update.ErrCommandNotFound))
	})
}

// TestResolveBinaryPath_GoEnvValues covers the values the go toolchain reports: its GOBIN wins over its GOPATH, the
// first GOPATH entry is used, and the environment answers when the toolchain reports nothing or fails.
func TestResolveBinaryPath_GoEnvValues(t *testing.T) {
	t.Parallel()

	table := []struct {
		name  string
		goEnv *fakeGoEnv
		env   envVars
		want  string
	}{
		{
			name:  "go env GOBIN wins over go env GOPATH",
			goEnv: &fakeGoEnv{settings: map[string]string{"GOBIN": "/gb", "GOPATH": "/gp"}},
			env:   envVars{},
			want:  "/gb/engram",
		},
		{
			name:  "go env GOPATH, first entry only",
			goEnv: &fakeGoEnv{settings: map[string]string{"GOPATH": "/gp1:/gp2"}},
			env:   envVars{},
			want:  "/gp1/bin/engram",
		},
		{
			name:  "go env reports nothing, the environment answers",
			goEnv: &fakeGoEnv{},
			env:   envVars{"GOBIN": "/eb"},
			want:  "/eb/engram",
		},
		{
			name:  "go env fails, the environment answers",
			goEnv: &fakeGoEnv{err: update.ErrGoNotFound},
			env:   envVars{"GOPATH": "/ep"},
			want:  "/ep/bin/engram",
		},
		{
			name:  "go env fails, nothing set, the home directory answers",
			goEnv: &fakeGoEnv{err: update.ErrGoNotFound},
			env:   envVars{},
			want:  "/home/joe/go/bin/engram",
		},
		{
			name:  "go env answers with the wrong count, the home directory answers",
			goEnv: &fakeGoEnv{short: true, settings: map[string]string{"GOBIN": "/gb"}},
			env:   envVars{},
			want:  "/home/joe/go/bin/engram",
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got := update.ExportResolveBinaryPath(t.Context(), "/home/joe", tc.env, tc.goEnv)
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

// fakeGoEnv is an update.GoEnv over fixed settings that records what it was asked. An unset setting is empty; short
// drops the last value; err is returned in place of any value.
type fakeGoEnv struct {
	settings map[string]string
	err      error
	short    bool
	asked    [][]string
}

func (f *fakeGoEnv) Values(_ context.Context, names ...string) ([]string, error) {
	f.asked = append(f.asked, slices.Clone(names))

	if f.err != nil {
		return nil, f.err
	}

	values := make([]string, 0, len(names))

	for _, name := range names {
		values = append(values, f.settings[name])
	}

	if f.short {
		values = values[:len(values)-1]
	}

	return values, nil
}
