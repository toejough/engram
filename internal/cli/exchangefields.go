package cli

// exchangeFrontmatter is the exchange frontmatter every fact, feedback and
// runbook note may carry (design D4, vault-note-identity). Every field is
// omitempty, so a note that never takes part in exchange serializes exactly
// as it did before exchange existed. Only the exchange paths (offer receipt,
// pull-down, `amend --discard --into`, a rename's alias, served learn) set or
// change these; every other frontmatter rewrite carries them through
// unchanged. The json:"-" tags keep any JSON decoding from populating them.
type exchangeFrontmatter struct {
	// XID is the note's own exchange ID: random, stamped lazily the first
	// time the note is queued, pulled or served-received, never backfilled,
	// never changed.
	XID string `json:"-" yaml:"xid,omitempty"`
	// Parent links the note to notes in the configured parent vault.
	Parent parentLinks `json:"-" yaml:"parent,omitempty"`
	// Aliases lists basenames this note answers to in its own vault.
	Aliases []string `json:"-" yaml:"aliases,omitempty"`
	// Offer records a served offer's origin, idempotency key, target and
	// path; kept after acceptance.
	Offer offerRecord `json:"-" yaml:"offer,omitempty"`
}

// noteAuthor is a pulled note's authorship in the parent vault.
type noteAuthor struct {
	Repo  string `json:"-" yaml:"repo,omitempty"`
	User  string `json:"-" yaml:"user,omitempty"`
	Vault string `json:"-" yaml:"vault,omitempty"`
}

// offerRecord is a served offer's bookkeeping (design D7): Origin is
// "<origin vault id>:<origin xid>", Key the idempotency key, For the
// basename an amend-offer targets, and Path the vault IDs the offer has
// passed through. PriorKeys holds the keys an in-place rewrite superseded
// (the most recent maxPriorOfferKeys, oldest dropped), so a late retry of
// a superseded key is recognized and never reverts the note (ruling S10).
// Only served learn's in-place rewrite sets it; it rides under the
// not-offered "offer" key, so it never enters the exchange hash.
type offerRecord struct {
	Origin    string   `json:"-" yaml:"origin,omitempty"`
	Key       string   `json:"-" yaml:"key,omitempty"`
	PriorKeys []string `json:"-" yaml:"prior_keys,omitempty"`
	For       string   `json:"-" yaml:"for,omitempty"`
	Path      []string `json:"-" yaml:"path,omitempty"`
}

// parentLink is one link to a parent note: Note is the parent basename, Via
// is offered, pulled or covered, and Hash is the exchange hash last
// exchanged.
type parentLink struct {
	Note string `json:"-" yaml:"note,omitempty"`
	Via  string `json:"-" yaml:"via,omitempty"`
	Hash string `json:"-" yaml:"hash,omitempty"`
}

// parentLinks is the note's links to the parent vault, keyed by that vault's
// ID. At most one link is primary (via offered or pulled).
type parentLinks struct {
	Vault  string       `json:"-" yaml:"vault,omitempty"`
	Links  []parentLink `json:"-" yaml:"links,omitempty"`
	Author noteAuthor   `json:"-" yaml:"author,omitempty"`
}
