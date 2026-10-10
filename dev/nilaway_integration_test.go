//go:build targ && integration

package dev

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func Test794a2a1a_RealAnalyzerScope(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture requires a POSIX shell")
	}
	binary, err := exec.LookPath("nilaway")
	if err != nil {
		t.Fatal("NilAway is required for the explicit dev integration suite")
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":                   "module github.com/toejough/engram/nilawayfixture\n\ngo 1.26\n\nrequire example.com/nilawaydependency v0.0.0\nreplace example.com/nilawaydependency => ./dependency\n",
		"main.go":                  "package main\nimport dependency \"example.com/nilawaydependency\"\nfunc main() { var value *int; println(*value); dependency.Broken() }\n",
		"dependency/go.mod":        "module example.com/nilawaydependency\n\ngo 1.26\n",
		"dependency/dependency.go": "package dependency\nfunc Broken() { var value *int; println(*value) }\n",
		"nilaway":                  "#!/bin/sh\nexec \"$ENGRAM_NILAWAY_REAL_BINARY\" \"$@\" > \"$ENGRAM_NILAWAY_CAPTURE\" 2>&1\n",
	}
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	capture := filepath.Join(dir, "diagnostics")
	t.Chdir(dir)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ENGRAM_NILAWAY_REAL_BINARY", binary)
	t.Setenv("ENGRAM_NILAWAY_CAPTURE", capture)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	settings := map[string]string{"GOMEMLIMIT": "2GiB", "GOGC": "50", "GOMAXPROCS": "2"}
	for _, scoped := range []bool{false, true} {
		args := []string{"-pretty-print=false", "-include-errors-in-files=" + dir}
		if scoped {
			args = append(args, nilawayArgs(false)...)
		} else {
			args = append(args, "./...")
		}
		args = append(args, "example.com/nilawaydependency")
		if err := executeNilaway(context.Background(), args, settings); err == nil {
			t.Fatal("known first-party nil dereference produced no failure")
		}
		contents, err := os.ReadFile(capture)
		if err != nil {
			t.Fatal(err)
		}
		output := string(contents)
		if !strings.Contains(output, "main.go:") || !strings.Contains(output, "Potential nil panic") {
			t.Fatalf("missing first-party nil diagnostic: %s", output)
		}
		if strings.Contains(output, "dependency.go:") == scoped {
			t.Fatalf("scoped=%v: unexpected dependency analysis: %s", scoped, output)
		}
	}
}

func Test794a2b1a_AcceptanceDefaults(t *testing.T) {
	for _, fix := range []bool{false, true} {
		t.Run(map[bool]string{false: "check", true: "fix"}[fix], func(t *testing.T) {
			capture := nilawayCapture(t)
			if err := executeNilaway(context.Background(), nilawayArgs(fix), nilawayEnvironment(os.LookupEnv)); err != nil {
				t.Fatal(err)
			}
			want := []string{"2GiB", "50", "2", "inherited", "-include-pkgs=github.com/toejough/engram"}
			if fix {
				want = append(want, "-fix")
			}
			assertNilawayCapture(t, capture, append(want, "./..."))
			for _, key := range []string{"GOMEMLIMIT", "GOGC", "GOMAXPROCS"} {
				if _, present := os.LookupEnv(key); present {
					t.Fatalf("parent environment gained %s", key)
				}
			}
		})
	}
}

func Test794a2c1a_AcceptanceOverrides(t *testing.T) {
	for _, supplied := range []map[string]string{
		{"GOMEMLIMIT": "4GiB"},
		{"GOMEMLIMIT": "off", "GOGC": "off", "GOMAXPROCS": "8"},
		{"GOMEMLIMIT": "", "GOGC": "", "GOMAXPROCS": ""},
	} {
		for _, fix := range []bool{false, true} {
			t.Run(map[bool]string{false: "check", true: "fix"}[fix], func(t *testing.T) {
				capture := nilawayCapture(t)
				for key, value := range supplied {
					t.Setenv(key, value)
				}
				if err := executeNilaway(context.Background(), nilawayArgs(fix), nilawayEnvironment(os.LookupEnv)); err != nil {
					t.Fatal(err)
				}
				want := []string{"2GiB", "50", "2", "inherited", "-include-pkgs=github.com/toejough/engram"}
				for index, key := range []string{"GOMEMLIMIT", "GOGC", "GOMAXPROCS"} {
					if value, present := supplied[key]; present {
						want[index] = value
					}
				}
				if fix {
					want = append(want, "-fix")
				}
				assertNilawayCapture(t, capture, append(want, "./..."))
				for _, key := range []string{"GOMEMLIMIT", "GOGC", "GOMAXPROCS"} {
					value, suppliedPresent := supplied[key]
					if got, present := os.LookupEnv(key); present != suppliedPresent || got != value {
						t.Fatalf("parent %s changed", key)
					}
				}
			})
		}
	}
}

func Test794b4b1c_ChildEnvironmentIsolation(t *testing.T) {
	capture := nilawayCapture(t)
	settings := map[string]string{"GOMEMLIMIT": "2GiB", "GOGC": "50", "GOMAXPROCS": "2"}
	if err := executeNilaway(context.Background(), []string{"-include-pkgs=github.com/toejough/engram", "./..."}, settings); err != nil {
		t.Fatal(err)
	}
	assertNilawayCapture(t, capture, []string{"2GiB", "50", "2", "inherited", "-include-pkgs=github.com/toejough/engram", "./..."})
	for key := range settings {
		if _, present := os.LookupEnv(key); present {
			t.Fatalf("parent environment gained %s", key)
		}
	}
}

func assertNilawayCapture(t *testing.T, capture string, want []string) {
	t.Helper()
	contents, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(contents), "\n"), "\n")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("child values = %#v, want %#v", got, want)
	}
}

func nilawayCapture(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	capture := filepath.Join(dir, "capture")
	script := "#!/bin/sh\n: \"${GOMEMLIMIT?}\" \"${GOGC?}\" \"${GOMAXPROCS?}\"\nprintf '%s\\n' \"$GOMEMLIMIT\" \"$GOGC\" \"$GOMAXPROCS\" \"$ENGRAM_NILAWAY_MARKER\" > \"$ENGRAM_NILAWAY_CAPTURE\"\nprintf '%s\\n' \"$@\" >> \"$ENGRAM_NILAWAY_CAPTURE\"\n"
	if err := os.WriteFile(filepath.Join(dir, "nilaway"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ENGRAM_NILAWAY_CAPTURE", capture)
	t.Setenv("ENGRAM_NILAWAY_MARKER", "inherited")
	for _, key := range []string{"GOMEMLIMIT", "GOGC", "GOMAXPROCS"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	return capture
}
