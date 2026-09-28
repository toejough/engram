package cli_test

// Served reads for exchange (design D7 H7, vault-serve-api "Served show
// SHALL offer a raw JSON envelope" and "Served query SHALL return dedupe
// keys on request"; tasks 4.4, 4.5).

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestServeQuery_DedupeKeysOnlyOnRequest: with dedupe-keys=1 the payload
// carries a top-level vault_id and, per note item, its exchange_hash and a
// non-empty aliases list; without it the payload is byte-identical to a
// local query with the same arguments.
func TestServeQuery_DedupeKeysOnlyOnRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	deps := serveTestDeps()

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "plain-note", Position: "top", Source: "local",
		Situation: "dedupe coverage plain", Subject: "engram", Predicate: "dedupes", Object: "notes",
	})
	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "renamed-note", Position: "top", Source: "local",
		Situation: "dedupe coverage renamed", Subject: "engram", Predicate: "dedupes", Object: "aliases",
	})

	renamed := noteFileWithSuffix(t, vault, "renamed-note.md")
	addFrontmatterLines(t, renamed, "aliases:\n  - 1.2026-01-01.old-name\n")

	query := map[string][]string{"phrase": {"dedupe coverage"}}
	routes := cli.ServeRoutes(deps, vault, "personal", "")

	plain := routeFor(t, routes, "/query").Serve(t.Context(), cli.ServeRequest{Query: query})
	g.Expect(plain.Status).To(Equal(200))

	var local bytes.Buffer

	localErr := cli.RunQuery(context.Background(), cli.QueryArgs{Phrases: []string{"dedupe coverage"}, VaultPath: vault},
		cli.ExportNewQueryDeps(deps), &local)
	g.Expect(localErr).NotTo(HaveOccurred())
	g.Expect(string(plain.Body)).To(Equal(local.String()), "without the parameter the payload is unchanged")
	g.Expect(string(plain.Body)).NotTo(ContainSubstring("vault_id"))
	g.Expect(string(plain.Body)).NotTo(ContainSubstring("exchange_hash"))

	withKeys := map[string][]string{"phrase": {"dedupe coverage"}, "dedupe-keys": {"1"}}
	keyed := routeFor(t, routes, "/query").Serve(t.Context(), cli.ServeRequest{Query: withKeys})
	g.Expect(keyed.Status).To(Equal(200))

	var payload struct {
		VaultID string `yaml:"vault_id"`
		Items   []struct {
			Path         string   `yaml:"path"`
			Kind         string   `yaml:"kind"`
			ExchangeHash string   `yaml:"exchange_hash"`
			Aliases      []string `yaml:"aliases"`
		} `yaml:"items"`
	}
	g.Expect(yaml.Unmarshal(keyed.Body, &payload)).To(Succeed())
	g.Expect(payload.VaultID).To(Equal(serverVaultID))
	g.Expect(payload.Items).NotTo(BeEmpty())

	seenRenamed := false

	for _, item := range payload.Items {
		g.Expect(item.ExchangeHash).To(Equal(fileExchangeHash(t, vault+"/"+item.Path)), item.Path)

		if strings.HasSuffix(item.Path, "renamed-note.md") {
			seenRenamed = true

			g.Expect(item.Aliases).To(Equal([]string{"1.2026-01-01.old-name"}))
		} else {
			g.Expect(item.Aliases).To(BeEmpty())
		}
	}

	g.Expect(seenRenamed).To(BeTrue())
	g.Expect(string(keyed.Body)).NotTo(MatchRegexp(`(?m)^\s+aliases: \[\]`), "an empty aliases list is omitted")
}

// TestServeQuery_DedupeKeysSkipUnhashableNote: a note whose exchange hash
// cannot be computed loses only its own keys (with a warning); the query
// still succeeds and every other note keeps its keys.
func TestServeQuery_DedupeKeysSkipUnhashableNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)

	var stderr bytes.Buffer

	deps := serveTestDeps()
	deps.Stderr = &stderr

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "good-note", Position: "top", Source: "local",
		Situation: "dedupe coverage good", Subject: "engram", Predicate: "hashes", Object: "notes",
	})
	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "odd-note", Position: "top", Source: "local",
		Situation: "dedupe coverage odd", Subject: "engram", Predicate: "hashes", Object: "oddly",
	})
	addFrontmatterLines(t, noteFileWithSuffix(t, vault, "odd-note.md"), "triggers: not-a-list\n")

	routes := cli.ServeRoutes(deps, vault, "personal", "")
	resp := routeFor(t, routes, "/query").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"phrase": {"dedupe coverage"}, "dedupe-keys": {"1"}},
	})
	g.Expect(resp.Status).To(Equal(200), string(resp.Body))

	var payload struct {
		Items []struct {
			Path         string `yaml:"path"`
			ExchangeHash string `yaml:"exchange_hash"`
		} `yaml:"items"`
	}
	g.Expect(yaml.Unmarshal(resp.Body, &payload)).To(Succeed())

	hashes := map[string]string{}
	for _, item := range payload.Items {
		hashes[item.Path] = item.ExchangeHash
	}

	g.Expect(hashes).To(HaveKeyWithValue(ContainSubstring("odd-note"), BeEmpty()))
	g.Expect(hashes).To(HaveKeyWithValue(ContainSubstring("good-note"), HavePrefix("xh1:")))
	g.Expect(stderr.String()).To(ContainSubstring("odd-note"))
}

// TestServeShow_ErrorStatuses: an empty ref is a 400; a listed note that
// cannot be read, or whose frontmatter cannot be parsed for its exchange
// hash, is a 500 — never a 404 or a partial envelope.
func TestServeShow_ErrorStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		query  map[string][]string
		status int
	}{
		{"empty raw ref", map[string][]string{"note": {""}, "raw": {"1"}}, 400},
		{"empty rendered ref", map[string][]string{"note": {""}}, 400},
		{"unreadable raw note", map[string][]string{"note": {"3.2026-01-03.unreadable"}, "raw": {"1"}}, 500},
		{"unreadable rendered note", map[string][]string{"note": {"3.2026-01-03.unreadable"}}, 500},
		{"unparseable raw note", map[string][]string{"note": {"4.2026-01-04.bad-yaml"}, "raw": {"1"}}, 500},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			unreadable := writeVaultNote(t, vault, "3.2026-01-03.unreadable.md", liveFactNote("3", ""))
			g.Expect(os.Chmod(unreadable, 0o000)).To(Succeed())
			writeVaultNote(t, vault, "4.2026-01-04.bad-yaml.md", "---\ntype: fact\nsituation: [unclosed\n---\n\nbody\n")

			routes := cli.ServeRoutes(serveTestDeps(), vault, "personal", "")
			resp := routeFor(t, routes, "/show").Serve(t.Context(), cli.ServeRequest{Query: testCase.query})

			g.Expect(resp.Status).To(Equal(testCase.status), string(resp.Body))
		})
	}
}

// TestServeShow_MissingNoteIsNotFound: a missing note is a 404 with an error
// body, raw or not — never a 500.
func TestServeShow_MissingNoteIsNotFound(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "1"} {
		t.Run("raw="+raw, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			query := map[string][]string{"note": {"404.2026-01-01.missing"}}

			if raw != "" {
				query["raw"] = []string{raw}
			}

			routes := cli.ServeRoutes(serveTestDeps(), vault, "personal", "")
			resp := routeFor(t, routes, "/show").Serve(t.Context(), cli.ServeRequest{Query: query})

			g.Expect(resp.Status).To(Equal(404))

			var errBody struct {
				Error string `json:"error"`
			}
			g.Expect(json.Unmarshal(resp.Body, &errBody)).To(Succeed())
			g.Expect(errBody.Error).To(ContainSubstring("not found"))
		})
	}
}

// TestServeShow_RawEnvelope: raw=1 returns {vault_id, basename, content,
// exchange_hash}, with content byte-identical to the note file (no
// red_flags preview cap, no outbound-links section), resolving a renamed
// note through its aliases to its current basename.
func TestServeShow_RawEnvelope(t *testing.T) {
	t.Parallel()

	for _, ref := range []string{"6.2026-01-06.many-flags", "6.2026-01-06.many-flags.md", "2.2026-01-02.old-name"} {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			path := writeVaultNote(t, vault, "6.2026-01-06.many-flags.md", manyRedFlagsRunbook)

			routes := cli.ServeRoutes(serveTestDeps(), vault, "personal", "")
			resp := routeFor(t, routes, "/show").Serve(t.Context(), cli.ServeRequest{
				Query: map[string][]string{"note": {ref}, "raw": {"1"}},
			})
			g.Expect(resp.Status).To(Equal(200))

			var envelope struct {
				VaultID      string `json:"vault_id"` //nolint:tagliatelle // design D7 envelope keys
				Basename     string `json:"basename"`
				Content      string `json:"content"`
				ExchangeHash string `json:"exchange_hash"` //nolint:tagliatelle // design D7 envelope keys
			}
			g.Expect(json.Unmarshal(resp.Body, &envelope)).To(Succeed())
			g.Expect(envelope.VaultID).To(Equal(serverVaultID))
			g.Expect(envelope.Basename).To(Equal("6.2026-01-06.many-flags"))
			g.Expect(envelope.Content).To(Equal(manyRedFlagsRunbook), "content is the file's bytes verbatim")
			g.Expect(envelope.ExchangeHash).To(Equal(fileExchangeHash(t, path)))
		})
	}
}

// unexported constants.
const (
	// manyRedFlagsRunbook has more red flags than engram show's preview cap,
	// a wikilink (which rendered show lists as an outbound link), and an
	// alias.
	manyRedFlagsRunbook = "---\ntype: runbook\nsituation: s\ndone_when: d\nred_flags:\n" +
		"  - flag one\n  - flag two\n  - flag three\n  - flag four\n  - flag five\n  - flag six\n" +
		"  - flag seven\n  - flag eight\n  - flag nine\n  - flag ten\n" +
		"luhmann: \"6\"\ncreated: 2026-01-06\nsource: agent\nuser: u\nvault: personal\n" +
		"aliases:\n  - 2.2026-01-02.old-name\n---\n\n1. see [[7.2026-01-07.other]]\n"
)

// addFrontmatterLines inserts lines just before path's closing frontmatter
// delimiter.
func addFrontmatterLines(t *testing.T, path, lines string) {
	t.Helper()

	content := readFileString(t, path)
	closing := strings.Index(content[len("---\n"):], "\n---\n") + len("---\n") + 1
	updated := content[:closing] + lines + content[closing:]

	writeVaultNote(t, "", path, updated)
}

// noteFileWithSuffix returns the one vault note whose name ends in suffix.
func noteFileWithSuffix(t *testing.T, vault, suffix string) string {
	t.Helper()

	for _, path := range noteFiles(t, vault) {
		if strings.HasSuffix(path, suffix) {
			return path
		}
	}

	t.Fatalf("no note ending in %s", suffix)

	return ""
}
