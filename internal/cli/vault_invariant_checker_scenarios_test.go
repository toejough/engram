package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b26c1a4_RunCheck_ScanFailureIsReturnedBeforeAnyLine verifies 792b26c1a4
// (vault-invariant-checker.md): a scan failure is returned as "check: scan: <error>", nothing is
// printed, and neither the note reader nor the sidecar reader is consulted.
func Test792b26c1a4_RunCheck_ScanFailureIsReturnedBeforeAnyLine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errScan := errors.New("vault root unreadable")
	reads := 0
	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) { return nil, errScan },
		ReadNote: func(string) ([]byte, error) {
			reads++

			return nil, nil
		},
		ReadSidecar: func(string) ([]byte, error) {
			reads++

			return nil, nil
		},
	}

	out, err := vic1Run(t, deps)

	g.Expect(err).To(MatchError(errScan))
	g.Expect(err).To(MatchError(HavePrefix("check: scan: ")))
	g.Expect(out).To(BeEmpty())
	g.Expect(reads).To(BeZero())
}

// Test792b26c1a5_RunCheck_InvalidUTF8NoteIsChecked verifies 792b26c1a5 (vault-invariant-checker.md):
// a note whose body holds invalid UTF-8 bytes and links a missing note is scanned and its dangling
// link is reported, through the production scan over an in-memory vault.
func Test792b26c1a5_RunCheck_InvalidUTF8NoteIsChecked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := newVic1Vault(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: testing the checker",
			"body \xff\xfe\xc3\x28 before the link\n\nRelated to:\n- [[999]] — x.\n"),
	})

	out, err := vic1Run(t, cli.ExportNewCheckDeps(vault.edgeFS()))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("WARN  G3 dangling: 1 authored links target no note\n"))
	g.Expect(out).To(ContainSubstring("        " + vic1NoteBasename + " → [[999]]\n"))
}

// Test792b26c2a3_RunCheck_LinkToMissingNoteWarns verifies 792b26c2a3 (vault-invariant-checker.md):
// a note whose link [[999]] matches no note gets a G3 WARN line, the G0 line begins PASS, and no
// failure is reported.
func Test792b26c2a3_RunCheck_LinkToMissingNoteWarns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := newVic1Vault(map[string]string{
		vic1NoteFile: vic1NoteText(vic1FactType, "situation: testing the checker", "body\n\n- [[999]] — x.\n"),
	})

	out, err := vic1Run(t, cli.ExportNewCheckDeps(vault.edgeFS()))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(HavePrefix(vic1G0PassLine))
	g.Expect(out).To(ContainSubstring(
		"WARN  G3 dangling: 1 authored links target no note\n        " + vic1NoteBasename + " → [[999]]\n"))
	g.Expect(out).NotTo(ContainSubstring("FAIL"))
}

// Test792b26c2a4_RunCheck_BareIDMatchingNoNoteIsDangling verifies 792b26c2a4
// (vault-invariant-checker.md): a link 999 with no note whose basename begins "999." is counted under
// G3 and not under G0, even when notes with the longer IDs 99 and 9990 exist.
func Test792b26c2a4_RunCheck_BareIDMatchingNoNoteIsDangling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "1.2026-05-30.foo", Outgoing: []string{"999"}},
				{Basename: "99.2026-05-30.near"},
				{Basename: "9990.2026-05-30.far"},
			}, nil
		},
	}

	out, err := vic1Run(t, deps)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine +
		"WARN  G3 dangling: 1 authored links target no note\n        1.2026-05-30.foo → [[999]]\n"))
}

// Test792b26c2a5_RunCheck_NoG3LineWithoutADanglingLink verifies 792b26c2a5
// (vault-invariant-checker.md): notes whose links all resolve print the G0 PASS line and no G3 line.
func Test792b26c2a5_RunCheck_NoG3LineWithoutADanglingLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "A", Outgoing: []string{"B"}},
				{Basename: "B", Outgoing: []string{"A"}},
			}, nil
		},
	}

	out, err := vic1Run(t, deps)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine))
	g.Expect(out).NotTo(ContainSubstring("G3"))
}

// Test792b26c3a2_RunCheck_BlankSituationFails verifies 792b26c3a2 (vault-invariant-checker.md): a fact
// note whose situation is only whitespace fails M5, naming the note.
func Test792b26c3a2_RunCheck_BlankSituationFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := vic1SituationDeps(map[string]string{
		vic1NoteBasename: vic1NoteText(vic1FactType, `situation: "   "`, "body\n"),
	})

	out, err := vic1Run(t, deps)

	g.Expect(err).To(MatchError(cli.ErrCheckFailedForTest))
	g.Expect(out).To(Equal(vic1G0PassLine +
		"FAIL  M5 situation-presence: 1 note(s) missing a situation\n        " + vic1NoteBasename + "\n"))
}

// Test792b26c3a4_RunCheck_OtherTypesAreSkipped verifies 792b26c3a4 (vault-invariant-checker.md): a note
// of type moc with no situation is not judged, so M5 passes.
func Test792b26c3a4_RunCheck_OtherTypesAreSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := vic1SituationDeps(map[string]string{
		vic1NoteBasename: vic1NoteText("moc", "title: a map", "map of content\n"),
	})

	out, err := vic1Run(t, deps)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine + vic1M5PassLine))
}

// Test792b26c3a5_RunCheck_UnparseableNotesAreSkipped verifies 792b26c3a5 (vault-invariant-checker.md): a
// note with malformed YAML frontmatter (a syntax error, and a type mismatch that still decodes type: fact)
// and a note with no frontmatter are not judged, so M5 passes.
func Test792b26c3a5_RunCheck_UnparseableNotesAreSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		malformedBasename = "2.2026-05-30.bad"
		bareBasename      = "3.2026-05-30.raw"
		mismatchBasename  = "4.2026-05-30.mismatch"
	)

	deps := vic1SituationDeps(map[string]string{
		malformedBasename: "---\ntype: fact\nsituation: [unterminated\n---\nbody\n",
		bareBasename:      "just prose, no frontmatter\n",
		// A sequence where a string is expected: yaml.v3 reports a type error yet still decodes
		// type: fact, so only the parse-error guard keeps this note out of M5.
		mismatchBasename: "---\ntype: fact\nsituation: [a, b]\n---\nbody\n",
	})

	out, err := vic1Run(t, deps)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine + vic1M5PassLine))
}

// Test792b26c3a7_RunCheck_UnreadableNoteIsSkipped verifies 792b26c3a7 (vault-invariant-checker.md): a
// scanned note whose file cannot be read is not counted under M5, even when the failed read also
// handed back content that would otherwise violate it.
func Test792b26c3a7_RunCheck_UnreadableNoteIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errRead := errors.New("permission denied")
	deps := cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: vic1NoteBasename}}, nil
		},
		ReadNote: func(string) ([]byte, error) {
			return []byte(vic1NoteText(vic1FactType, `situation: ""`, "body\n")), errRead
		},
	}

	out, err := vic1Run(t, deps)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine + vic1M5PassLine))
}

// Test792b26c4a1_RunCheck_OldSchemaSidecarFails verifies 792b26c4a1 (vault-invariant-checker.md): a
// sidecar on the old single-vector schema fails S1, names the note, and a failure is reported.
func Test792b26c4a1_RunCheck_OldSchemaSidecarFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	oldSchema := `{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:x"}`

	out, err := vic1Run(t, vic1SidecarDeps(func(string) ([]byte, error) { return []byte(oldSchema), nil }))

	g.Expect(err).To(MatchError(cli.ErrCheckFailedForTest))
	g.Expect(out).To(Equal(vic1G0PassLine +
		"FAIL  S1 sidecar-schema: 1 sidecar(s) on an old/invalid schema (run `engram embed apply --force`)\n" +
		"        " + vic1NoteBasename + "\n"))
}

// Test792b26c4a2_RunCheck_CurrentSidecarPasses verifies 792b26c4a2 (vault-invariant-checker.md): a
// current sidecar prints the S1 PASS line and no error.
func Test792b26c4a2_RunCheck_CurrentSidecarPasses(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const dims = 4

	current := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             dims,
		SituationVector:  []float32{1, 0, 0, 0},
		BodyVector:       []float32{0, 1, 0, 0},
		ContentHash:      "sha256:x",
	})

	out, err := vic1Run(t, vic1SidecarDeps(func(string) ([]byte, error) { return current, nil }))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine + vic1S1PassLine))
}

// Test792b26c4a3_RunCheck_MissingSidecarWarns verifies 792b26c4a3 (vault-invariant-checker.md): a note
// whose sidecar cannot be read gets a WARN line that counts it, the S1 PASS line follows, and no
// failure is reported. The sidecar read is the one beside the note.
func Test792b26c4a3_RunCheck_MissingSidecarWarns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var readPaths []string

	out, err := vic1Run(t, vic1SidecarDeps(func(path string) ([]byte, error) {
		readPaths = append(readPaths, path)

		return nil, fs.ErrNotExist
	}))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal(vic1G0PassLine +
		"WARN  S1 sidecar-schema: 1 note(s) missing a sidecar (run `engram embed apply`)\n" + vic1S1PassLine))
	g.Expect(readPaths).To(Equal([]string{filepath.Join(vic1VaultPath, vic1NoteBasename+".vec.json")}))
}

// Test792b26c4a4_RunCheck_MalformedSidecarFails verifies 792b26c4a4 (vault-invariant-checker.md): a
// sidecar that is not valid JSON fails S1 and names the note.
func Test792b26c4a4_RunCheck_MalformedSidecarFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	out, err := vic1Run(t, vic1SidecarDeps(func(string) ([]byte, error) { return []byte("{not json"), nil }))

	g.Expect(err).To(MatchError(cli.ErrCheckFailedForTest))
	g.Expect(out).To(Equal(vic1G0PassLine +
		"FAIL  S1 sidecar-schema: 1 sidecar(s) on an old/invalid schema (run `engram embed apply --force`)\n" +
		"        " + vic1NoteBasename + "\n"))
}

// Test792b26c6a1_RunCheck_CheckWritesNothing verifies 792b26c6a1 (vault-invariant-checker.md): a check
// over a vault that violates every invariant creates, changes and deletes no file. The in-memory vault
// records every write-capable EdgeFS call, and none is recorded.
func Test792b26c6a1_RunCheck_CheckWritesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		linkerBasename = "1.2026-05-30.linker"
		targetBasename = "2.2026-05-30.target"
	)

	oldSchema := `{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:x"}`
	vault := newVic1Vault(map[string]string{
		// G0 (bare id of an existing note), G3 (dangling), M5 (no situation), S1 FAIL (old schema).
		linkerBasename + ".md": vic1NoteText(vic1FactType, "subject: x",
			"body\n\n- [[2]] — bare id.\n- [[999]] — dangling.\n"),
		linkerBasename + ".vec.json": oldSchema,
		// S1 WARN: no sidecar at all.
		targetBasename + ".md": vic1NoteText(vic1FactType, "subject: y", "body\n"),
	})

	out, err := vic1Run(t, cli.ExportNewCheckDeps(vault.edgeFS()))

	g.Expect(err).To(MatchError(cli.ErrCheckFailedForTest))
	g.Expect(out).To(ContainSubstring("FAIL  G0 graph-resolution: 1 bare-id links that should resolve\n"))
	g.Expect(out).To(ContainSubstring("WARN  G3 dangling: 1 authored links target no note\n"))
	g.Expect(out).To(ContainSubstring("FAIL  M5 situation-presence: 2 note(s) missing a situation\n"))
	g.Expect(out).To(ContainSubstring("WARN  S1 sidecar-schema: 1 note(s) missing a sidecar"))
	g.Expect(out).To(ContainSubstring("FAIL  S1 sidecar-schema: 1 sidecar(s) on an old/invalid schema"))
	g.Expect(vault.writes).To(BeEmpty())
}

// unexported constants.
const (
	vic1FactType     = "fact"
	vic1G0PassLine   = "PASS  G0 graph-resolution: every authored link resolves by form\n"
	vic1M5PassLine   = "PASS  M5 situation-presence: every fact/feedback names a situation\n"
	vic1NoteBasename = "1.2026-05-30.foo"
	vic1NoteFile     = vic1NoteBasename + ".md"
	vic1S1PassLine   = "PASS  S1 sidecar-schema: every sidecar parses under the current schema\n"
	vic1VaultPath    = "v"
)

// vic1Vault is an in-memory vault behind a cli.EdgeFS. Every write-capable call is recorded in
// writes instead of applied, so a test can prove a read-only command did none.
type vic1Vault struct {
	files  map[string][]byte
	writes []string
}

// edgeFS returns the EdgeFS view of the vault: directory listing and file reads over files, every
// write-capable call recorded.
func (v *vic1Vault) edgeFS() fakeEdgeFS {
	return fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			content, ok := v.files[filepath.Clean(path)]
			if !ok {
				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
			}

			return content, nil
		},
		readDir: func(dir string) ([]fs.DirEntry, error) {
			names := make([]string, 0, len(v.files))

			for path := range v.files {
				if filepath.Dir(path) == filepath.Clean(dir) {
					names = append(names, filepath.Base(path))
				}
			}

			sort.Strings(names)

			entries := make([]fs.DirEntry, 0, len(names))
			for _, name := range names {
				entries = append(entries, fakeDirEntry{name: name})
			}

			return entries, nil
		},
		writeFile: func(path string, _ []byte, _ fs.FileMode) error {
			v.record("WriteFile", path)

			return nil
		},
		writeFileAtomic: func(path string, _ []byte, _ fs.FileMode) error {
			v.record("WriteFileAtomic", path)

			return nil
		},
		writeFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
			v.record("WriteFileExcl", path)

			return nil
		},
		mkdirAll: func(path string, _ fs.FileMode) error {
			v.record("MkdirAll", path)

			return nil
		},
		mkdirTemp: func(dir, _ string) (string, error) {
			v.record("MkdirTemp", dir)

			return dir, nil
		},
		remove: func(path string) error {
			v.record("Remove", path)

			return nil
		},
		removeAll: func(path string) error {
			v.record("RemoveAll", path)

			return nil
		},
		rename: func(oldPath, _ string) error {
			v.record("Rename", oldPath)

			return nil
		},
		symlink: func(_, link string) error {
			v.record("Symlink", link)

			return nil
		},
	}
}

// record notes one write-capable call.
func (v *vic1Vault) record(op, path string) {
	v.writes = append(v.writes, op+" "+path)
}

// newVic1Vault builds a vic1Vault from file names (relative to the vault directory) and contents.
func newVic1Vault(files map[string]string) *vic1Vault {
	vault := &vic1Vault{files: make(map[string][]byte, len(files))}
	for name, content := range files {
		vault.files[filepath.Join(vic1VaultPath, name)] = []byte(content)
	}

	return vault
}

// vic1NoteText renders a note with a type, one more frontmatter line and a body.
func vic1NoteText(noteType, frontmatterLine, body string) string {
	return "---\ntype: " + noteType + "\n" + frontmatterLine + "\n---\n" + body
}

// vic1Run runs the check over deps with the shared vault path and returns what it printed and the
// error it returned.
func vic1Run(t *testing.T, deps cli.CheckDeps) (string, error) {
	t.Helper()

	var out bytes.Buffer

	err := cli.RunCheck(t.Context(), cli.CheckArgs{VaultPath: vic1VaultPath}, deps, &out)

	return out.String(), err
}

// vic1SidecarDeps returns CheckDeps over one scanned fact note that run only the graph and sidecar
// checks, reading sidecars through readSidecar.
func vic1SidecarDeps(readSidecar func(string) ([]byte, error)) cli.CheckDeps {
	return cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: vic1NoteBasename}}, nil
		},
		ReadSidecar: readSidecar,
	}
}

// vic1SituationDeps returns CheckDeps that scan one note per key of texts (the key is the basename)
// and read each note's text, running the graph and situation checks.
func vic1SituationDeps(texts map[string]string) cli.CheckDeps {
	basenames := make([]string, 0, len(texts))
	for basename := range texts {
		basenames = append(basenames, basename)
	}

	sort.Strings(basenames)

	return cli.CheckDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			notes := make([]vaultgraph.Note, 0, len(basenames))
			for _, basename := range basenames {
				notes = append(notes, vaultgraph.Note{Basename: basename})
			}

			return notes, nil
		},
		ReadNote: func(path string) ([]byte, error) {
			basename := filepath.Base(path)
			basename = basename[:len(basename)-len(".md")]

			text, ok := texts[basename]
			if !ok {
				return nil, fs.ErrNotExist
			}

			return []byte(text), nil
		},
	}
}
