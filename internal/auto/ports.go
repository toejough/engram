package auto

import "time"

// Clock is the wall clock.
type Clock interface{ Now() time.Time }

// Deps are the impure capabilities of a job, injected at the edge.
type Deps struct {
	Runner Runner
	Files  Files
	Env    Env
	Clock  Clock
}

// Env reads the environment of the process.
type Env interface{ Get(key string) string }

// Files is the file system the job reads transcripts, the chunk index and the vault from, and writes the backlog
// marks to.
type Files interface {
	Read(path string) (string, error)
	Write(path, text string) error
	Exists(path string) bool
	// Walk lists the regular files below root.
	Walk(root string) ([]string, error)
}

// Output is what a command printed.
type Output struct{ Stdout, Stderr string }

// Runner runs one engram (or system) command by its argument vector, with no shell, within the deadline. A command
// that exits with a code other than 0, cannot start or passes its deadline is an error whose text says so ("failed to
// start", "still running after").
type Runner interface {
	Run(argv []string, deadline time.Duration) (Output, error)
}
