package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

var frontmatterDelim = []byte("---\n")

// ExtractBody returns the markdown body of a note with the leading YAML
// frontmatter block stripped. If the note has no leading frontmatter, it
// is returned unchanged.
//
// Frontmatter format: a leading "---\n" line, arbitrary lines (which may
// themselves be blank), and a closing "---\n" line. Anything after the
// closing delimiter is the body. A single leading blank line after the
// closing delimiter is also stripped so notes whose frontmatter blocks
// differ but whose bodies match produce identical hashes.
func ExtractBody(raw []byte) []byte {
	if !bytes.HasPrefix(raw, frontmatterDelim) {
		return raw
	}

	rest := raw[len(frontmatterDelim):]

	_, body, ok := bytes.Cut(rest, frontmatterDelim)
	if !ok {
		return raw
	}

	return bytes.TrimPrefix(body, []byte("\n"))
}

// ContentHash returns a sha256: prefixed hex digest of the note's body
// (frontmatter stripped). Used to detect stale sidecars when a note's
// body has changed.
func ContentHash(raw []byte) string {
	sum := sha256.Sum256(ExtractBody(raw))

	return "sha256:" + hex.EncodeToString(sum[:])
}
