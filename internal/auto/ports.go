package auto

import "time"

// Clock is the wall clock.
type Clock interface{ Now() time.Time }

// Deps are the impure capabilities of a job, injected at the edge: the commands it runs, the files it reads, the
// pending offers of the vault as the pending-offer bookkeeping tells them, the clock, and where the vault and the chunk
// index are, resolved as the engram commands resolve them.
type Deps struct {
	Runner    Runner
	Files     Files
	Offers    Offers
	Clock     Clock
	VaultPath string
	ChunksDir string
}

// Files is the file system the job reads transcripts and the chunk index from, and writes the queue and the backlog
// marks to.
type Files interface {
	Read(path string) (string, error)
	Write(path, text string) error
	Exists(path string) bool
}

// Offers lists the pending offers of a vault.
type Offers interface {
	// Pending lists, in name order, the file names of the notes of the vault that are pending offers.
	Pending(vault string) ([]string, error)
}

// Output is what a command printed.
type Output struct{ Stdout, Stderr string }

// Runner runs one engram (or system) command by its argument vector, with no shell, within the deadline. A command
// that exits with a code other than 0, cannot start or passes its deadline is an error whose text says so ("failed to
// start", "still running after").
type Runner interface {
	Run(argv []string, deadline time.Duration) (Output, error)
}
