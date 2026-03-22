package toolgate_test

import (
	"maps"
	"testing"

	"engram/internal/toolgate"

	. "github.com/onsi/gomega"
)

func TestCommandKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{name: "two tokens subcommand", cmd: "go test ./...", want: "go test"},
		{name: "targ subcommand", cmd: "targ check-full", want: "targ check-full"},
		{name: "flag second token dropped", cmd: "grep -r foo src/", want: "grep"},
		{name: "leading env var stripped", cmd: "FOO=bar git push origin main", want: "git push"},
		{name: "multiple env vars stripped", cmd: "A=1 B=2 npm install", want: "npm install"},
		{name: "single token command", cmd: "ls", want: "ls"},
		{name: "flag only second token", cmd: "ls -la", want: "ls"},
		{name: "empty string", cmd: "", want: ""},
		{name: "whitespace only", cmd: "   ", want: ""},
		{name: "env var only", cmd: "FOO=bar", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(toolgate.CommandKey(tt.cmd)).To(Equal(tt.want))
		})
	}
}

func TestExtractBashCommand(t *testing.T) {
	t.Parallel()

	t.Run("extracts command field from valid JSON", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		result := toolgate.ExtractBashCommand(`{"command":"grep foo bar"}`)
		g.Expect(result).To(Equal("grep foo bar"))
	})

	t.Run("returns empty string for invalid JSON", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		result := toolgate.ExtractBashCommand("not json")
		g.Expect(result).To(Equal(""))
	})

	t.Run("returns empty string when command field absent", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		result := toolgate.ExtractBashCommand(`{"description":"do something"}`)
		g.Expect(result).To(Equal(""))
	})
}

func TestGate_FirstCall_AlwaysSurfaces(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	shouldSurface, err := gate.Check("go test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}

func TestGate_HighCount_SkipsWhenRollExceedsProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	// P(100) ≈ 0.18, roll of 0.5 > 0.18 → should skip.
	shouldSurface, err := gate.Check("grep")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeFalse())
}

func TestGate_HighCount_SurfacesWhenRollBelowProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.1 })

	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	shouldSurface, err := gate.Check("grep")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}

func TestGate_SeparateKeys_IndependentCounters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	store := newStubStore()
	gate := toolgate.NewGate(store, func() float64 { return 0.5 })

	for range 100 {
		_, err := gate.Check("grep")
		g.Expect(err).NotTo(HaveOccurred())
	}

	shouldSurface, err := gate.Check("go test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(shouldSurface).To(BeTrue())
}

func TestSurfaceProbability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// count 0 → 1.0
	g.Expect(toolgate.SurfaceProbability(0)).To(BeNumerically("~", 1.0, 0.001))

	// count 1 → 1/(1+ln(2)) ≈ 0.59
	g.Expect(toolgate.SurfaceProbability(1)).To(BeNumerically("~", 0.59, 0.01))

	// count 10 → 1/(1+ln(11)) ≈ 0.29
	g.Expect(toolgate.SurfaceProbability(10)).To(BeNumerically("~", 0.29, 0.01))

	// count 100 → 1/(1+ln(101)) ≈ 0.18
	g.Expect(toolgate.SurfaceProbability(100)).To(BeNumerically("~", 0.18, 0.01))

	// monotonically decreasing
	prev := toolgate.SurfaceProbability(0)
	for _, count := range []int{1, 2, 5, 10, 50, 100, 1000} {
		probability := toolgate.SurfaceProbability(count)
		g.Expect(probability).To(BeNumerically("<", prev), "probability should decrease with count")
		prev = probability
	}
}

// stubStore is an in-memory CounterStore for testing.
type stubStore struct {
	data map[string]toolgate.CounterEntry
}

func (s *stubStore) Load() (map[string]toolgate.CounterEntry, error) {
	out := make(map[string]toolgate.CounterEntry, len(s.data))
	maps.Copy(out, s.data)

	return out, nil
}

func (s *stubStore) Save(entries map[string]toolgate.CounterEntry) error {
	s.data = make(map[string]toolgate.CounterEntry, len(entries))
	maps.Copy(s.data, entries)

	return nil
}

func newStubStore() *stubStore {
	return &stubStore{data: make(map[string]toolgate.CounterEntry)}
}
