//go:build targ && integration

package dev

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"
)

func Test794a2a1a_ExistingEntryPointsReportOnlyEngramModulePackages(t *testing.T) {
	for _, entry := range []string{"check-nils-for-fail", "check-nils", "check", "check-for-fail", "check-full"} {
		t.Run(entry, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(reachesNilsForFail(entryTarget(entry))).To(BeTrue(), "%s does not run check-nils-for-fail", entry)
		})
	}

	t.Run("check-nils-for-fail reports in Engram and not in the dependency", func(t *testing.T) {
		g := NewWithT(t)
		dir := newFixture(t, map[string]string{
			"thirdparty/go.mod":        "module example.com/thirdparty\n\ngo " + fixtureGoVersion + "\n",
			"thirdparty/thirdparty.go": "package thirdparty\n\nfunc Broken() { var value *int; println(*value) }\n",
		}, "require example.com/thirdparty v0.0.0\nreplace example.com/thirdparty => ./thirdparty\n",
			"import \"example.com/thirdparty\"\n\nfunc Dependent() { thirdparty.Broken() }\n")

		output, err := runEntry(t, dir, checkNilsForFail)

		g.Expect(err).To(HaveOccurred(), "known nil dereference produced no failure")
		g.Expect(output).To(ContainSubstring(fixtureEngineMarker))
		g.Expect(output).To(ContainSubstring(fixtureFaultMarker))
		g.Expect(output).NotTo(ContainSubstring("thirdparty.go:"))
	})
}

func Test794a2a1b_PrefixSiblingModuleIsNotReportedOn(t *testing.T) {
	g := NewWithT(t)
	dir := newFixture(t, map[string]string{
		"evaldep/go.mod":  "module github.com/toejough/engram-eval\n\ngo " + fixtureGoVersion + "\n",
		"evaldep/eval.go": "package eval\n\nfunc Broken() { var value *int; println(*value) }\n",
	}, "require github.com/toejough/engram-eval v0.0.0\nreplace github.com/toejough/engram-eval => ./evaldep\n",
		"import eval \"github.com/toejough/engram-eval\"\n\nfunc Dependent() { eval.Broken() }\n")

	output, err := runEntry(t, dir, checkNilsForFail)

	g.Expect(err).To(HaveOccurred(), "known nil dereference produced no failure")
	g.Expect(output).To(ContainSubstring(fixtureEngineMarker))
	g.Expect(output).To(ContainSubstring(fixtureFaultMarker))
	g.Expect(output).NotTo(ContainSubstring("eval.go:"))
}

func Test794a2a1c_FixCommandActsOnlyOnEngramModulePackages(t *testing.T) {
	g := NewWithT(t)
	dependencyPath := "thirdparty/thirdparty.go"
	dependencySource := "package thirdparty\n\nfunc Broken() { var value *int; println(*value) }\n"
	dir := newFixture(t, map[string]string{
		"thirdparty/go.mod": "module example.com/thirdparty\n\ngo " + fixtureGoVersion + "\n",
		dependencyPath:      dependencySource,
	}, "require example.com/thirdparty v0.0.0\nreplace example.com/thirdparty => ./thirdparty\n",
		"import \"example.com/thirdparty\"\n\nfunc Dependent() { thirdparty.Broken() }\n")

	// NilAway with -fix applies suggested fixes and prints no diagnostics, even for a known nil dereference, so
	// the scenario's "is reported" clause is not observable with the real analyzer. What is observable is that
	// the dependency is neither reported on nor modified.
	output, err := runEntry(t, dir, checkNilsFix)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(output).NotTo(ContainSubstring("thirdparty.go:"))
	after, err := os.ReadFile(filepath.Join(dir, dependencyPath))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(after)).To(Equal(dependencySource))
}

func Test794a2a1d_NestedModuleIsNotReportedOn(t *testing.T) {
	g := NewWithT(t)
	dir := newFixture(t, map[string]string{
		"nested/go.mod":    "module github.com/toejough/engram/nested\n\ngo " + fixtureGoVersion + "\n",
		"nested/nested.go": "package nested\n\nfunc Broken() { var value *int; println(*value) }\n",
	}, "require github.com/toejough/engram/nested v0.0.0\nreplace github.com/toejough/engram/nested => ./nested\n",
		"import \"github.com/toejough/engram/nested\"\n\nfunc Dependent() { nested.Broken() }\n")

	output, err := runEntry(t, dir, checkNilsForFail)

	g.Expect(err).To(HaveOccurred(), "known nil dereference produced no failure")
	g.Expect(output).To(ContainSubstring(fixtureEngineMarker))
	g.Expect(output).To(ContainSubstring(fixtureFaultMarker))
	g.Expect(output).NotTo(ContainSubstring("nested.go:"))
}

func Test794a2c1a_SuppliedSettingsArePassedThrough(t *testing.T) {
	entries := map[string]func(context.Context) error{
		"check-nils-for-fail": checkNilsForFail,
		"check-nils-fix":      checkNilsFix,
	}
	for _, scenario := range []struct {
		entry    string
		supplied map[string]string
	}{
		{"check-nils-for-fail", map[string]string{"GOMEMLIMIT": "4GiB"}},
		{"check-nils-fix", map[string]string{"GOMEMLIMIT": "off", "GOGC": "off", "GOMAXPROCS": "8"}},
		{"check-nils-for-fail", map[string]string{"GOMEMLIMIT": "", "GOGC": "", "GOMAXPROCS": ""}},
	} {
		t.Run(scenario.entry+" "+formatSettings(scenario.supplied), func(t *testing.T) {
			g := NewWithT(t)
			capture := standInNilaway(t, 0)
			setResourceVariables(t, scenario.supplied)

			g.Expect(entries[scenario.entry](context.Background())).To(Succeed())

			g.Expect(readStandInResources(t, capture)).To(Equal(scenario.supplied))
			g.Expect(currentResourceVariables()).To(Equal(scenario.supplied))
		})
	}
}

func Test794a2c1b_AbsentSettingsStayAbsent(t *testing.T) {
	entries := map[string]func(context.Context) error{
		"check-nils-for-fail": checkNilsForFail,
		"check-nils-fix":      checkNilsFix,
	}
	for entry, run := range entries {
		t.Run(entry, func(t *testing.T) {
			g := NewWithT(t)
			capture := standInNilaway(t, 0)
			setResourceVariables(t, nil)

			g.Expect(run(context.Background())).To(Succeed())

			g.Expect(readStandInResources(t, capture)).To(BeEmpty())
			g.Expect(currentResourceVariables()).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	fixtureDiagnosticFile = "diagnostics"
	fixtureEngineMarker   = "engram.go:"
	fixtureEnginePackage  = `package engram

func Broken() { var value *int; println(*value) }
`
	fixtureFaultMarker = "Potential nil panic"
	fixtureGoVersion   = "1.21"
	fixtureWrapper     = "#!/bin/sh\nexec \"$ENGRAM_NILAWAY_REAL_BINARY\" -pretty-print=false \"$@\" " +
		"> \"$ENGRAM_NILAWAY_CAPTURE\" 2>&1\n"
)

// entryTarget returns the shared development target the named entry point resolves to.
func entryTarget(name string) *targ.Target {
	return map[string]*targ.Target{
		"check-nils-for-fail": targdev.CheckNilsForFail,
		"check-nils":          targdev.CheckNils,
		"check":               targdev.Check,
		"check-for-fail":      targdev.CheckForFail,
		"check-full":          targdev.CheckFull,
	}[name]
}

// newFixture writes a hermetic checkout whose root module is Engram's module path. extraModule is appended to its
// go.mod, and importBlock plus a nil dereference in the root package are written to engram.go.
func newFixture(t *testing.T, files map[string]string, extraModule, importBlock string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture requires a POSIX shell")
	}
	binary, err := exec.LookPath("nilaway")
	if err != nil {
		t.Fatal("NilAway is required for the explicit dev integration suite")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	all := map[string]string{
		"go.mod":    "module github.com/toejough/engram\n\ngo " + fixtureGoVersion + "\n\n" + extraModule,
		"engram.go": strings.Replace(fixtureEnginePackage, "\nfunc Broken", "\n"+importBlock+"\nfunc Broken", 1),
		"nilaway":   fixtureWrapper,
	}
	for name, contents := range files {
		all[name] = contents
	}
	for name, contents := range all {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ENGRAM_NILAWAY_REAL_BINARY", binary)
	t.Setenv("ENGRAM_NILAWAY_CAPTURE", filepath.Join(dir, fixtureDiagnosticFile))
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	return dir
}

// reachesNilsForFail reports whether the target runs the configured check-nils-for-fail target.
func reachesNilsForFail(target *targ.Target) bool {
	if target == targdev.CheckNilsForFail {
		return true
	}
	for _, dependency := range target.GetDeps() {
		if reachesNilsForFail(dependency) {
			return true
		}
	}
	return false
}

// runEntry runs an entry point from the fixture directory and returns the analyzer's diagnostics.
func runEntry(t *testing.T, dir string, entry func(context.Context) error) (string, error) {
	t.Helper()
	t.Chdir(dir)
	err := entry(context.Background())
	contents, readErr := os.ReadFile(filepath.Join(dir, fixtureDiagnosticFile))
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(contents), err
}
