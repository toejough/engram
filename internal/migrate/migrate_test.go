package migrate_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/migrate"
)

func TestRunCLI_FailsOnMissingSrcDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	exitCode := migrate.RunCLI([]string{"--data-dir", dataDir})
	g.Expect(exitCode).To(Equal(1))
}

func TestRunCLI_SucceedsWithValidDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	exitCode := migrate.RunCLI([]string{"--data-dir", dataDir})
	g.Expect(exitCode).To(Equal(0))
}

func TestRun_ErrorOnAtomicRenameFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	// Fail the rename of temp file to destination (but not the backup rename).
	deps.Rename = func(old, _ string) error {
		// Temp files start with ".tmp-migrate-".
		if filepath.Base(old) != "memories" && filepath.Base(old) != "mem.toml" {
			return errInjectedAtomicRename
		}

		return nil
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(MatchError(ContainSubstring("renaming temp to destination")))
}

func TestRun_ErrorOnCreateTempFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()
	deps.CreateTemp = func(_, _ string) (*os.File, error) {
		return nil, errInjectedCreateTemp
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(MatchError(ContainSubstring("creating temp file")))
}

func TestRun_ErrorOnEncodeFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	// Return a read-only file from CreateTemp so encoding fails.
	dstDir := filepath.Join(dataDir, "memory", "feedback")

	mkErr := os.MkdirAll(dstDir, 0o750)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	deps.CreateTemp = func(dir, pattern string) (*os.File, error) {
		// Create a real temp file, then close and reopen read-only.
		tmpFile, tmpErr := os.CreateTemp(dir, pattern)
		if tmpErr != nil {
			return nil, tmpErr
		}

		tmpPath := tmpFile.Name()

		closeErr := tmpFile.Close()
		if closeErr != nil {
			return nil, closeErr
		}

		return os.Open(tmpPath)
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ErrorOnInvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	writeErr := os.WriteFile(
		filepath.Join(srcDir, "bad.toml"),
		[]byte("this is not valid { toml [[["),
		0o640,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(MatchError(ContainSubstring("migrating bad.toml")))
}

func TestRun_ErrorOnMissingSrcDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_ErrorOnMkdirFeedbackFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	deps := testDeps()
	deps.MkdirAll = func(_ string, _ os.FileMode) error {
		return errInjectedMkdir
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(MatchError(ContainSubstring("creating")))
}

func TestRun_ErrorOnRenameFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")
	dstDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(dstDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	// Inject failing rename for the final atomic write step.
	// Count calls: first rename is temp->dst (should succeed),
	// second is srcDir->backupDir (we make this fail to exercise the warning path).
	realRename := deps.Rename
	deps.Rename = func(oldPath, newPath string) error {
		// Fail only the backup rename.
		if filepath.Base(oldPath) == "memories" {
			return errInjectedRename
		}

		return realRename(oldPath, newPath)
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The warning should appear on stderr.
	stderrBuf, ok := deps.Stderr.(*bytes.Buffer)
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	g.Expect(stderrBuf.String()).To(ContainSubstring("warning"))
}

func TestRun_ErrorOnVerifyReadFail(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	// Fail ReadFile on temp files (validation step), but allow source reads.
	realReadFile := deps.ReadFile
	deps.ReadFile = func(path string) ([]byte, error) {
		if strings.Contains(filepath.Base(path), ".tmp-migrate-") {
			return nil, errInjectedVerifyRead
		}

		return realReadFile(path)
	}

	err := migrate.Run(dataDir, deps)
	g.Expect(err).To(MatchError(ContainSubstring("reading temp for validation")))
}

func TestRun_MigratesLegacyFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyTOML := `schema_version = 1
situation = "when running tests"
behavior = "using go test directly"
impact = "misses coverage"
action = "use targ test"
project_scoped = true
project_slug = "engram"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-02T00:00:00Z"
surfaced_count = 5
followed_count = 3
not_followed_count = 1
irrelevant_count = 0
missed_count = 2
initial_confidence = 0.8
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "test-mem.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Verify stdout contains OK message and summary.
	stdoutBuf, ok := deps.Stdout.(*bytes.Buffer)
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	stdout := stdoutBuf.String()
	g.Expect(stdout).To(ContainSubstring("OK: test-mem.toml"))
	g.Expect(stdout).To(ContainSubstring("Migrated: 1, Skipped: 0"))

	// Verify destination file exists and has correct format.
	dstPath := filepath.Join(dataDir, "memory", "feedback", "test-mem.toml")

	dstData, readErr := os.ReadFile(dstPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(dstData)
	g.Expect(content).To(ContainSubstring(`type = "feedback"`))
	g.Expect(content).To(ContainSubstring(`schema_version = 1`))
	g.Expect(content).To(ContainSubstring("[content]"))
	g.Expect(content).To(ContainSubstring(`behavior = "using go test directly"`))
	g.Expect(content).NotTo(ContainSubstring("pending_evaluations"))

	// Verify it round-trips via TOML decode.
	type checkContent struct {
		Behavior string `toml:"behavior"`
		Impact   string `toml:"impact"`
		Action   string `toml:"action"`
	}

	type checkRecord struct {
		SchemaVersion     int          `toml:"schema_version"`
		Type              string       `toml:"type"`
		Situation         string       `toml:"situation"`
		ProjectScoped     bool         `toml:"project_scoped"`
		ProjectSlug       string       `toml:"project_slug"`
		Content           checkContent `toml:"content"`
		CreatedAt         string       `toml:"created_at"`
		UpdatedAt         string       `toml:"updated_at"`
		SurfacedCount     int          `toml:"surfaced_count"`
		FollowedCount     int          `toml:"followed_count"`
		NotFollowedCount  int          `toml:"not_followed_count"`
		IrrelevantCount   int          `toml:"irrelevant_count"`
		MissedCount       int          `toml:"missed_count"`
		InitialConfidence float64      `toml:"initial_confidence"`
	}

	var decoded checkRecord

	_, decErr := toml.Decode(content, &decoded)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(decoded.SchemaVersion).To(Equal(1))
	g.Expect(decoded.Type).To(Equal("feedback"))
	g.Expect(decoded.Situation).To(Equal("when running tests"))
	g.Expect(decoded.Content.Behavior).To(Equal("using go test directly"))
	g.Expect(decoded.Content.Impact).To(Equal("misses coverage"))
	g.Expect(decoded.Content.Action).To(Equal("use targ test"))
	g.Expect(decoded.ProjectScoped).To(BeTrue())
	g.Expect(decoded.ProjectSlug).To(Equal("engram"))
	g.Expect(decoded.SurfacedCount).To(Equal(5))
	g.Expect(decoded.FollowedCount).To(Equal(3))
	g.Expect(decoded.NotFollowedCount).To(Equal(1))
	g.Expect(decoded.MissedCount).To(Equal(2))
	g.Expect(decoded.InitialConfidence).To(Equal(0.8))

	// Verify facts directory was created.
	factsDir := filepath.Join(dataDir, "memory", "facts")
	_, factsErr := os.Stat(factsDir)
	g.Expect(factsErr).NotTo(HaveOccurred())

	// Verify source was renamed to backup.
	_, backupErr := os.Stat(filepath.Join(dataDir, "memories.v1-backup"))
	g.Expect(backupErr).NotTo(HaveOccurred())

	_, srcErr := os.Stat(srcDir)
	g.Expect(os.IsNotExist(srcErr)).To(BeTrue())
}

func TestRun_SkipsExistingDestination(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")
	dstDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(dstDir, 0o750)).To(Succeed())

	legacyTOML := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "existing.toml"), []byte(legacyTOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Pre-create destination so it gets skipped.
	preErr := os.WriteFile(filepath.Join(dstDir, "existing.toml"), []byte("already here"), 0o640)
	g.Expect(preErr).NotTo(HaveOccurred())

	if preErr != nil {
		return
	}

	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdoutBuf, ok := deps.Stdout.(*bytes.Buffer)
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	g.Expect(stdoutBuf.String()).To(ContainSubstring("SKIP (exists): existing.toml"))
	g.Expect(stdoutBuf.String()).To(ContainSubstring("Migrated: 0, Skipped: 1"))
}

func TestRun_SkipsNonTOMLFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	// Create a non-TOML file that should be ignored.
	writeErr := os.WriteFile(filepath.Join(srcDir, "readme.txt"), []byte("not a memory"), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdoutBuf, ok := deps.Stdout.(*bytes.Buffer)
	g.Expect(ok).To(BeTrue())

	if !ok {
		return
	}

	g.Expect(stdoutBuf.String()).To(ContainSubstring("Migrated: 0, Skipped: 0"))
}

func TestRun_StripsPendingEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(srcDir, 0o750)).To(Succeed())

	legacyWithPending := `situation = "test"
behavior = "b"
impact = "i"
action = "a"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"

[[pending_evaluations]]
surfaced_at = "2026-01-01T00:00:00Z"
user_prompt = "test prompt"
session_id = "abc"
project_slug = "proj"
`

	writeErr := os.WriteFile(filepath.Join(srcDir, "pending.toml"), []byte(legacyWithPending), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	deps := testDeps()

	err := migrate.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	dstPath := filepath.Join(dataDir, "memory", "feedback", "pending.toml")

	dstData, readErr := os.ReadFile(dstPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(dstData)).NotTo(ContainSubstring("pending_evaluations"))
}

// unexported variables.
var (
	errInjectedAtomicRename = errors.New("injected atomic rename error")
	errInjectedCreateTemp   = errors.New("injected create temp error")
	errInjectedMkdir        = errors.New("injected mkdir error")
	errInjectedRename       = errors.New("injected rename error")
	errInjectedVerifyRead   = errors.New("injected verify read error")
)

// testDeps returns Deps backed by the real filesystem with captured stdout/stderr.
func testDeps() migrate.Deps {
	deps := migrate.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}
	deps.Stderr = &bytes.Buffer{}

	return deps
}
