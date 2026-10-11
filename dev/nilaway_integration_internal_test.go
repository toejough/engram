//go:build targ && integration

package dev

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
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

	// The fix command guarantees only that the dependency is neither modified nor reported on. NilAway with -fix
	// applies suggested fixes and prints no diagnostics, so whether the Engram fault is reported is not asserted.
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

// entryTarget returns the target Engram registers under the named entry point.
func entryTarget(name string) *targ.Target {
	return map[string]*targ.Target{
		"check-nils-for-fail": checkNilsForFailTarget,
		"check-nils":          checkNilsTarget,
		"check":               checkTarget,
		"check-for-fail":      checkForFailTarget,
		"check-full":          checkFullTarget,
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
		t.Fatal("nilaway not on PATH; install it to run check-full")
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
	maps.Copy(all, files)
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

// reachesNilsForFail reports whether the target runs Engram's check-nils-for-fail target.
func reachesNilsForFail(target *targ.Target) bool {
	return reaches(target, checkNilsForFailTarget)
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

func Test794b4a1d_TargsOtherSharedTargetsStayAvailable(t *testing.T) {
	g := NewWithT(t)
	module, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/toejough/targ").Output()
	g.Expect(err).NotTo(HaveOccurred())

	declared := exportedTargetVariables(t, filepath.Join(strings.TrimSpace(string(module)), "dev"))
	g.Expect(declared).NotTo(BeEmpty())

	kept := make([]string, 0, len(sharedTargets))
	for name, target := range sharedTargets {
		kept = append(kept, name)

		g.Expect(target.GetSource()).To(Equal(targSourcePackage), "%s is not Targ's own target", name)
		g.Expect(target.GetName()).NotTo(BeEmpty(), name)
	}

	replaced := []string{
		"Check", "CheckForFail", "CheckFull", "CheckNils", "CheckNilsFix", "CheckNilsForFail",
	}

	g.Expect(slices.Concat(kept, replaced)).To(ConsistOf(declared),
		"Targ's shared targets changed: re-register each new one, or drop each removed one, in sharedTargets")
}

// exportedTargetVariables returns the names of the exported targets the package at dir declares.
func exportedTargetVariables(t *testing.T, dir string) []string {
	t.Helper()
	files, err := parser.ParseDir(token.NewFileSet(), dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0)
	for _, pkg := range files {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gen, isGen := decl.(*ast.GenDecl)
				if !isGen || gen.Tok != token.VAR {
					continue
				}

				for _, spec := range gen.Specs {
					value, isValue := spec.(*ast.ValueSpec)
					if !isValue || len(value.Values) != 1 || !buildsTarget(value.Values[0]) {
						continue
					}

					for _, name := range value.Names {
						if name.IsExported() {
							names = append(names, name.Name)
						}
					}
				}
			}
		}
	}

	return names
}

// buildsTarget reports whether the expression is a call chain rooted at targ.Targ(...).
func buildsTarget(expr ast.Expr) bool {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		return false
	}

	selector, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return false
	}

	if root, isIdent := selector.X.(*ast.Ident); isIdent {
		return root.Name == "targ" && selector.Sel.Name == "Targ"
	}

	return buildsTarget(selector.X)
}
