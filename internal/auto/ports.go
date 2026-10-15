package auto

import "time"

// AmendRequest is an amend: the note it names, and what is done to it. Fields are the content fields to replace.
// Provenance is a line appended to the source of the note once, which is not a change of its content or of who wrote
// it (793b419). ExpectNote is the note as the job read it: when the note is no longer that version, nothing is written
// (the check `--expect-hash` makes, from the text read and not from a hash shown); ExpectHash is the hash an offer was
// shown with.
type AmendRequest struct {
	Target       string
	Into         string
	ExpectHash   string
	ExpectNote   string
	Provenance   string
	Fields       [][2]string
	Activate     bool
	ClearPending bool
	Discard      bool
}

// Clock is the wall clock.
type Clock interface{ Now() time.Time }

// Deps are the impure capabilities of a job, injected at the edge: the vault the job reads and writes, the files it
// reads, the pending offers of the vault as the pending-offer bookkeeping tells them, the clock, and where the vault
// and the chunk index are, resolved as the engram commands resolve them.
type Deps struct {
	Vault     Vault
	Files     Files
	Offers    Offers
	Clock     Clock
	VaultPath string
	ChunksDir string
}

// Files is the file system the job reads transcripts and the chunk index from, and writes the backlog marks to.
type Files interface {
	Read(path string) (string, error)
	Write(path, text string) error
	Exists(path string) bool
}

// LearnRequest is a note to write: its kind (`feedback` or `fact`), its slug, the source line, the situation and the
// values of the fields of its kind. The note is written at the top of the vault.
type LearnRequest struct {
	Kind, Slug, Source, Situation string
	Values                        [][2]string
}

// Offers lists the pending offers of a vault.
type Offers interface {
	// Pending lists, in name order, the file names of the notes of the vault that are pending offers.
	Pending(vault string) ([]string, error)
}

// Output is what an operation of the vault printed, and what it left on its standard error.
type Output struct{ Stdout, Stderr string }

// QueryRequest is a search: the text the user said, the phrases to match and how many items to return. WithChunks
// searches the chunk index too, and a chunk comes back as its id and score only; without it the search is of the notes
// of the vault (793b425).
type QueryRequest struct {
	Text       string
	Phrases    []string
	Limit      int
	WithChunks bool
}

// Vault is the memory of the vault as the job uses it (793b417): the operations the commands `engram query`,
// `engram show`, `engram learn`, `engram amend` and `engram ingest --auto` perform, called in the process by the code
// of the commands, with the deadline of the step. An operation that fails, or passes its deadline, is an error whose
// text says why ("still running after"); the text is what the failure is classified by.
type Vault interface {
	// Amend amends a note (or folds, clears, discards or activates it) as `engram amend` does.
	Amend(request AmendRequest, deadline time.Duration) error
	// Index brings the chunk index up to date as `engram ingest --auto` does.
	Index(deadline time.Duration) error
	// Learn writes a note as `engram learn` does, and returns what it printed: the path of the note.
	Learn(request LearnRequest, deadline time.Duration) (string, error)
	// Query searches the vault as `engram query` does, and returns its payload.
	Query(request QueryRequest, deadline time.Duration) (string, error)
	// Show reads a note as `engram show` does, and returns what it printed.
	Show(name string, deadline time.Duration) (string, error)
}
