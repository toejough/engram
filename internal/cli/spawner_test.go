package cli_test

import (
	"errors"
	"io"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func TestSpawner_PassesCallThroughOnSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var gotName string

	var gotArgs, gotEnv []string

	run := func(name string, args, extraEnv []string) (int, error) {
		gotName, gotArgs, gotEnv = name, args, extraEnv
		return 0, nil
	}

	code, err := spawnerOver(run).Run("engram", []string{"update"}, []string{"ENGRAM_UPDATE_REEXEC=1"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(0))
	g.Expect(gotName).To(Equal("engram"))
	g.Expect(gotArgs).To(Equal([]string{"update"}))
	g.Expect(gotEnv).To(Equal([]string{"ENGRAM_UPDATE_REEXEC=1"}))
}

func TestSpawner_PassesThroughNonZeroExitWithNilErr(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	run := func(_ string, _, _ []string) (int, error) { return 3, nil }

	code, err := spawnerOver(run).Run("engram", nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(3))
}

func TestSpawner_PassesThroughSpawnFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errSpawn := errors.New("spawn failed")
	run := func(_ string, _, _ []string) (int, error) { return -1, errSpawn }

	code, err := spawnerOver(run).Run("engram", nil, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(errSpawn))
	g.Expect(code).To(Equal(-1))
}

// spawnerOver builds the composed update.Spawner from a fake RunInherited
// primitive, through the real NewDeps wiring path (mirrors commanderOver).
func spawnerOver(run func(name string, args, extraEnv []string) (int, error)) update.Spawner {
	prims := cli.Primitives{Spawn: cli.SpawnPrims{RunInherited: run}}

	return cli.NewDeps(prims, io.Discard, io.Discard, nil).Spawner
}
