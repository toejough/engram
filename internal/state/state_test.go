package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"pgregory.net/rapid"
)

func fixedTime() time.Time {
	return time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

func TestInit(t *testing.T) {
	t.Run("creates state file with correct initial state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal("test-project"))
		g.Expect(s.Project.Phase).To(Equal("init"))
		g.Expect(s.Project.Created).To(Equal(fixedTime()))
		g.Expect(s.History).To(HaveLen(1))
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[0].Timestamp).To(Equal(fixedTime()))

		// Verify file exists on disk
		_, err = os.Stat(filepath.Join(dir, state.StateFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("errors if state file already exists", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create initial state
		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Try again — should fail
		_, err = state.Init(dir, "test-project", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	t.Run("errors if directory does not exist", func(t *testing.T) {
		g := NewWithT(t)

		_, err := state.Init("/nonexistent/path/that/does/not/exist", "test-project", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	})

	t.Run("state file is valid TOML readable by Get", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		original, err := state.Init(dir, "roundtrip-test", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Name).To(Equal(original.Project.Name))
		g.Expect(loaded.Project.Phase).To(Equal(original.Project.Phase))
		g.Expect(loaded.History).To(HaveLen(1))
	})
}

func TestInitProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		name := rapid.StringMatching(`[a-z][a-z0-9-]{1,30}`).Draw(rt, "name")
		dir := t.TempDir()

		s, err := state.Init(dir, name, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal(name))
		g.Expect(s.Project.Phase).To(Equal("init"))

		// Roundtrip: read back should match
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Name).To(Equal(name))
	})
}

func TestGet(t *testing.T) {
	t.Run("errors if state file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Get(dir)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})

	t.Run("reads valid state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "read-test", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal("read-test"))
		g.Expect(s.Project.Phase).To(Equal("init"))
	})
}
