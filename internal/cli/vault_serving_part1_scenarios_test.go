package cli_test

// Scenario tests for the vault-serving product capability, part 1 (861c2zzf): the served command set and the help
// (792a20a, 792a20b), served commands behaving as the local ones (792a20c), the pending offer and its receipt
// (792a20d), and the identity, repo and vault name a served learn stamps (792a20e, 792a20f). Spec:
// docs/specs/product/vault-serving.md. The scenarios that need the real net/http mux (792a20a1a, 792a20a1b and
// 792a20a1c) live in cmd/engram/vault_serving_part1_scenarios_test.go.
//
// The scenarios run over the route handlers and the `engram serve`, `engram query` and `engram show` targets with the
// deps the neighbouring serve tests use: a vault in the test's own t.TempDir() (the recorded deviation, #860: the
// vault lock is a real flock, and no in-memory EdgeFS models the exclusive create and the directory listing a
// learn needs), a deterministic stub embedder, a fixed clock and a scripted git, so the server's own detection
// differs from every value a caller declares. No network and no real home.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a20b1c_ServeHelp_TheHelpStatesTheAddressRule verifies 792a20b1c (The help states the address rule):
// engram serve --help names the four served commands, says --addr is required and never defaults to 0.0.0.0, and
// lists --vault, --vault-name and --chunks-dir.
func Test792a20b1c_ServeHelp_TheHelpStatesTheAddressRule(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return vs1Home, nil }

	result, err := targ.Execute([]string{"engram", "serve", "--help"}, cli.Targets(deps)...)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	for _, command := range []string{"query", "show", "activate", "learn"} {
		g.Expect(result.Output).To(ContainSubstring(command), "the help names "+command)
	}

	for _, flag := range []string{"--addr", "--vault", "--vault-name", "--chunks-dir"} {
		g.Expect(result.Output).To(ContainSubstring(flag), "the help lists "+flag)
	}

	addrLine := vs1FlagLine(result.Output, "--addr")
	g.Expect(addrLine).To(ContainSubstring("required"), "the --addr line says the address is required: %s", result.Output)
	g.Expect(addrLine).To(ContainSubstring("never defaults to 0.0.0.0"), "the --addr line states the wildcard rule")
}

// Test792a20c1a_ServeQuery_AServedQueryMatchesTheLocalQuery verifies 792a20c1a (A served query matches the local
// query): over one vault, the payload of GET /query with a phrase is byte-identical to the payload of the local
// engram query with the same phrase.
func Test792a20c1a_ServeQuery_AServedQueryMatchesTheLocalQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault, chunksDir := t.TempDir(), t.TempDir()
	vs1PlantNotes(t, deps, vault)
	vs1PlantChunk(t, chunksDir, "chunk about the vault door")

	local, localStderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", vs1Phrase, "--vault", vault, "--chunks-dir", chunksDir},
		vs1Customize)
	g.Expect(localStderr).To(BeEmpty())
	g.Expect(local).To(ContainSubstring("served-alpha"), "the local payload holds the notes, so equality is not empty")
	g.Expect(local).To(ContainSubstring("chunk about the vault door"), "and the chunk")

	served := routeFor(t, cli.ServeRoutes(deps, vault, vs1ServerName, chunksDir), "/query").Serve(t.Context(),
		cli.ServeRequest{Query: map[string][]string{"phrase": {vs1Phrase}}})

	g.Expect(served.Status).To(Equal(vs1StatusOK), string(served.Body))
	g.Expect(string(served.Body)).To(Equal(local))
}

// Test792a20c1c_ServeLearn_ConcurrentLocalAndServedWritersLoseNothing verifies 792a20c1c (Concurrent local and
// served writers lose nothing): a local learn and a served learn released together on one vault both complete, both
// notes are written under distinct Luhmann IDs and the vault holds no torn or stray file. The race is made
// deterministic through the filesystem seam: each writer's note create waits (with a timeout) until the other writer
// has reached its own, and a writer reaches it only after listing the vault to choose its ID. Under the shared vault
// lock the second writer cannot list until the first has written, so the wait times out and the writes run one
// after the other; without the lock both list the same state and pick the same ID, on every run.
func Test792a20c1c_ServeLearn_ConcurrentLocalAndServedWritersLoseNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	rendezvous := vs1NewRendezvousFS(deps.FS)
	deps.FS = rendezvous

	vault := t.TempDir()
	learnRoute := routeFor(t, cli.ServeRoutes(deps, vault, vs1ServerName, t.TempDir()), "/learn")

	body, marshalErr := json.Marshal(vs1FactArgs("served-writer", vs1CallerUser, vs1CallerRepo))
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	localArgs := vs1FactArgs("local-writer", "", "")
	localArgs.Vault, localArgs.VaultName = vault, vs1ServerName

	var (
		wg       sync.WaitGroup
		localErr error
		served   cli.ServeResponse
	)

	start := make(chan struct{})

	wg.Go(func() {
		<-start

		localErr = cli.ExportRunLearn(t.Context(), localArgs, cli.ExportNewLearnDeps(deps), io.Discard)
	})
	wg.Go(func() {
		<-start

		served = learnRoute.Serve(t.Context(), cli.ServeRequest{Body: body})
	})

	close(start)
	wg.Wait()

	g.Expect(rendezvous.arrivals()).To(Equal(vs1WriterCount), "both writers reached their note create")
	g.Expect(localErr).NotTo(HaveOccurred())
	g.Expect(served.Status).To(Equal(vs1StatusOK), string(served.Body))

	notes := noteFiles(t, vault)
	g.Expect(notes).To(HaveLen(vs1WriterCount), "both notes are written, neither is lost")

	ids := map[string]bool{}

	for _, note := range notes {
		id, found := cli.ExportExtractLuhmannFromFilename(filepath.Base(note))
		g.Expect(found).To(BeTrue(), "%s names a Luhmann ID", note)

		ids[id] = true
	}

	g.Expect(ids).To(HaveLen(vs1WriterCount), "the two notes hold distinct Luhmann IDs")

	for slug, pending := range map[string]bool{"served-writer": true, "local-writer": false} {
		content := readFileString(t, vs1NoteWithSlug(t, vault, slug))

		g.Expect(content).To(HavePrefix("---\n"), "%s is a whole note", slug)
		g.Expect(content).To(ContainSubstring("object: "+slug+" object"), "%s content", slug)
		g.Expect(strings.Contains(content, "pending: true")).To(Equal(pending), "%s marker", slug)
	}

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	for _, entry := range entries {
		g.Expect(entry.Name()).NotTo(ContainSubstring(".tmp"), "no temporary file is left")
	}
}

// Test792a20c1d_Serve_ARequestCannotRedirectTheServer verifies 792a20c1d (A request cannot redirect the server):
// a served request that carries a vault, a vault name or a chunks directory of its own is answered from the
// server's own vault and chunk index, for each of the four routes, and the decoy named in the request is not read
// or written.
func Test792a20c1d_Serve_ARequestCannotRedirectTheServer(t *testing.T) {
	t.Parallel()

	redirecting := func(decoy, decoyChunks string) map[string][]string {
		return map[string][]string{
			"phrase": {vs1Phrase}, "vault": {decoy}, "vault-path": {decoy}, "vaultPath": {decoy},
			"vault-name": {"decoy-name"}, "chunks-dir": {decoyChunks}, "chunksDir": {decoyChunks},
		}
	}

	t.Run("query", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := vs1NewRedirectRig(t)
		query := routeFor(t, rig.routes, "/query")

		plain := query.Serve(t.Context(), cli.ServeRequest{Query: map[string][]string{"phrase": {vs1Phrase}}})
		g.Expect(plain.Status).To(Equal(vs1StatusOK), string(plain.Body))
		g.Expect(string(plain.Body)).To(ContainSubstring("served-alpha"))
		g.Expect(string(plain.Body)).To(ContainSubstring("the server's own chunk"))
		g.Expect(string(plain.Body)).NotTo(ContainSubstring("decoy"))

		redirected := query.Serve(t.Context(),
			cli.ServeRequest{Query: redirecting(rig.decoyVault, rig.decoyChunks)})
		g.Expect(redirected.Status).To(Equal(vs1StatusOK), string(redirected.Body))
		g.Expect(string(redirected.Body)).To(Equal(string(plain.Body)), "the request's own vault and chunks are ignored")
		g.Expect(vs1Snapshot(t, rig.decoyVault, rig.decoyChunks)).To(Equal(rig.decoyBefore), "the decoy is untouched")
	})

	t.Run("show", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := vs1NewRedirectRig(t)
		show := routeFor(t, rig.routes, "/show")

		plain := show.Serve(t.Context(), cli.ServeRequest{Query: map[string][]string{"note": {"1"}}})
		g.Expect(plain.Status).To(Equal(vs1StatusOK), string(plain.Body))
		g.Expect(string(plain.Body)).To(ContainSubstring("the server's own note"))

		redirected := show.Serve(t.Context(), cli.ServeRequest{Query: map[string][]string{
			"note": {"1"}, "vault": {rig.decoyVault}, "vault-path": {rig.decoyVault}, "vaultPath": {rig.decoyVault},
		}})
		g.Expect(redirected.Status).To(Equal(vs1StatusOK), string(redirected.Body))
		g.Expect(string(redirected.Body)).To(Equal(string(plain.Body)))
		g.Expect(string(redirected.Body)).NotTo(ContainSubstring("decoy"))
	})

	t.Run("activate", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := vs1NewRedirectRig(t)
		ownNote := vs1NoteWithSlug(t, rig.vault, "served-alpha")
		body := fmt.Sprintf(`{"notes":[%q],"vault":%q,"vaultPath":%q,"vault-path":%q}`,
			strings.TrimSuffix(filepath.Base(ownNote), ".md"), rig.decoyVault, rig.decoyVault, rig.decoyVault)

		resp := routeFor(t, rig.routes, "/activate").Serve(t.Context(), cli.ServeRequest{Body: []byte(body)})
		g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

		own := readFileString(t, embed.SidecarPath(ownNote))
		g.Expect(own).To(ContainSubstring(`"last_used":"`+vs1Date+`"`), "the server's own note is activated")
		g.Expect(vs1Snapshot(t, rig.decoyVault, rig.decoyChunks)).To(Equal(rig.decoyBefore), "the decoy is untouched")
	})

	t.Run("learn", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := vs1NewRedirectRig(t)
		args := vs1FactArgs("redirect-attempt", vs1CallerUser, vs1CallerRepo)

		raw, marshalErr := json.Marshal(args)
		g.Expect(marshalErr).NotTo(HaveOccurred())

		if marshalErr != nil {
			return
		}

		var fields map[string]any
		g.Expect(json.Unmarshal(raw, &fields)).To(Succeed())

		fields["vault"], fields["vaultPath"], fields["chunks-dir"] = rig.decoyVault, rig.decoyVault, rig.decoyChunks

		body, bodyErr := json.Marshal(fields)
		g.Expect(bodyErr).NotTo(HaveOccurred())

		if bodyErr != nil {
			return
		}

		resp := routeFor(t, rig.routes, "/learn").Serve(t.Context(), cli.ServeRequest{Body: body})
		g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

		g.Expect(noteFiles(t, rig.vault)).To(HaveLen(2), "the offer is written to the server's own vault")
		g.Expect(vs1NoteWithSlug(t, rig.vault, "redirect-attempt")).NotTo(BeEmpty())
		g.Expect(vs1Snapshot(t, rig.decoyVault, rig.decoyChunks)).To(Equal(rig.decoyBefore), "the decoy is untouched")
	})
}

// Test792a20d1a_ServeLearn_TheReceiptNamesThePendingNote verifies 792a20d1a (The receipt names the pending note):
// a served learn that creates pending note 1100.2026-09-27.x.md on a served vault with an ID starting 9a1e answers
// with status "offer received", luhmann "1100", basename "1100.2026-09-27.x", pending true and the vault's ID.
func Test792a20d1a_ServeLearn_TheReceiptNamesThePendingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, "1099.2026-01-01.prior.md", liveFactNote("1099", ""))

	resp := serveLearnWith(t, vs1ServerDeps(), vault, vs1FactArgs("x", vs1CallerUser, vs1CallerRepo))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Status).To(Equal("offer received"))
	g.Expect(receipt.Luhmann).To(Equal("1100"))
	g.Expect(receipt.Basename).To(Equal("1100.2026-09-27.x"))
	g.Expect(receipt.Pending).To(BeTrue())
	g.Expect(receipt.VaultID).To(Equal(serverVaultID))
	g.Expect(serverVaultID).To(HavePrefix("9a1e"))
	g.Expect(readFileString(t, filepath.Join(vault, "1100.2026-09-27.x.md"))).To(ContainSubstring("pending: true"))
}

// Test792a20d1b_ServeLearn_TheOfferedNoteIsPendingNotLive verifies 792a20d1b (The offered note is pending, not
// live): after a served learn succeeded, engram show reads the note with the pending marker and engram query does
// not return it, while a live note of the same situation is returned, so the exclusion is the marker's doing.
func Test792a20d1b_ServeLearn_TheOfferedNoteIsPendingNotLive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault, chunksDir := newServeVault(t), t.TempDir()

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "live-twin", Position: "top", Source: "local",
		Situation: vs1Phrase, Subject: "engram", Predicate: "serves", Object: "live object",
	})

	args := vs1FactArgs("pending-twin", vs1CallerUser, vs1CallerRepo)
	args.Situation = vs1Phrase

	resp := serveLearnWith(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)

	shown, showStderr := executeCapturingBoth(t,
		[]string{"engram", "show", receipt.Basename, "--vault", vault}, vs1Customize)
	g.Expect(showStderr).To(BeEmpty())
	g.Expect(shown).To(ContainSubstring("pending: true"), "the note is read with its pending marker")

	queried, queryStderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", vs1Phrase, "--vault", vault, "--chunks-dir", chunksDir},
		vs1Customize)
	g.Expect(queryStderr).To(BeEmpty())
	g.Expect(queried).To(ContainSubstring("live-twin"), "the live note of the same situation is returned")
	g.Expect(queried).NotTo(ContainSubstring("pending-twin"), "the pending offer is not returned")
	g.Expect(queried).NotTo(ContainSubstring(receipt.Basename))
}

// Test792a20d1c_ServeLearn_TheReceiptCarriesNoContent verifies 792a20d1c (The receipt carries no content): the
// receipt of a succeeded served learn holds exactly the receipt fields, none of the note's content, the declared
// identity or a curation outcome, for a fact, a feedback and a runbook, and for an offer that resolves to a live
// note (which adds only for).
func Test792a20d1c_ServeLearn_TheReceiptCarriesNoContent(t *testing.T) {
	t.Parallel()

	const liveBasename = "3.2026-01-01.target"

	receiptKeys := []string{"basename", "luhmann", "pending", "status", "stored_hash", "vault_id"}

	cases := map[string]struct {
		args   cli.LearnArgs
		keys   []string
		tokens []string
	}{
		"a fact": {
			args: cli.LearnArgs{
				Type: "fact", Situation: "tokfsit", Subject: "tokfsub", Predicate: "tokfpred", Object: "tokfobj",
			},
			keys:   receiptKeys,
			tokens: []string{"tokfsit", "tokfsub", "tokfpred", "tokfobj"},
		},
		"a feedback": {
			args: cli.LearnArgs{
				Type: "feedback", Situation: "tokbsit", Behavior: "tokbbeh", Impact: "tokbimp", Action: "tokbact",
			},
			keys:   receiptKeys,
			tokens: []string{"tokbsit", "tokbbeh", "tokbimp", "tokbact"},
		},
		"a runbook": {
			args: cli.LearnArgs{
				Type: "runbook", Situation: "tokrsit", DoneWhen: "tokrdone", Body: "1. tokrstep",
				Triggers: []string{"tokrtrigger"}, RedFlags: []string{"tokrflag"},
			},
			keys:   receiptKeys,
			tokens: []string{"tokrsit", "tokrdone", "tokrstep", "tokrtrigger", "tokrflag"},
		},
		"an offer that resolves to a live note": {
			args: cli.LearnArgs{
				Type: "fact", Situation: "tokosit", Subject: "tokosub", Predicate: "tokopred", Object: "tokoobj",
				Offer: offerOf("", "", liveBasename),
			},
			keys:   append([]string{"for"}, receiptKeys...),
			tokens: []string{"tokosit", "tokosub", "tokopred", "tokoobj"},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			writeVaultNote(t, vault, liveBasename+".md", liveFactNote("3", ""))

			args := testCase.args
			args.Slug, args.Position, args.Source = "receipt-content", "top", "child"
			args.User, args.Repo = "toku@example.com", "tokrepo"

			resp := serveLearnWith(t, vs1ServerDeps(), vault, args)
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			var fields map[string]any
			g.Expect(json.Unmarshal(resp.Body, &fields)).To(Succeed())
			g.Expect(slices.Sorted(maps.Keys(fields))).To(Equal(slices.Sorted(slices.Values(testCase.keys))),
				"the receipt holds the receipt fields only: no content and no curation outcome")

			for _, token := range append([]string{"toku@example.com", "tokrepo"}, testCase.tokens...) {
				g.Expect(string(resp.Body)).NotTo(ContainSubstring(token), "the receipt does not echo %s", token)
			}
		})
	}
}

// Test792a20e1a_ServeLearn_TheDeclaredIdentityIsStampedAsGiven verifies 792a20e1a (The declared identity is stamped
// as given): a served learn whose body declares user alice@example.com writes a pending note whose user: is
// alice@example.com, not the identity the server's own git and OS user resolve to.
func Test792a20e1a_ServeLearn_TheDeclaredIdentityIsStampedAsGiven(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)

	resp := serveLearnWith(t, vs1ServerDeps(), vault, vs1FactArgs("stamped", "alice@example.com", vs1CallerRepo))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	note := readFileString(t, filepath.Join(vault, decodeReceipt(t, resp).Basename+".md"))
	g.Expect(note).To(MatchRegexp(`(?m)^user: alice@example\.com$`))
	g.Expect(note).To(ContainSubstring("pending: true"))
	g.Expect(note).NotTo(ContainSubstring(vs1ServerUser), "the server's own identity is not stamped")
}

// Test792a20e1b_ServeLearn_AnEmptyIdentityIsRejected verifies 792a20e1b (An empty identity is rejected): a served
// learn whose declared user is empty, whether sent as "" or left out, is answered 400 with "serve: user: must be
// non-empty" and writes nothing.
func Test792a20e1b_ServeLearn_AnEmptyIdentityIsRejected(t *testing.T) {
	t.Parallel()

	bodies := map[string]func(args cli.LearnArgs) ([]byte, error){
		"user sent as the empty string": func(args cli.LearnArgs) ([]byte, error) {
			args.User = ""

			return json.Marshal(args)
		},
		"user left out of the body": func(args cli.LearnArgs) ([]byte, error) {
			raw, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("marshaling the body: %w", err)
			}

			var fields map[string]any

			err = json.Unmarshal(raw, &fields)
			if err != nil {
				return nil, fmt.Errorf("decoding the body: %w", err)
			}

			delete(fields, "user")

			return json.Marshal(fields)
		},
	}

	for name, build := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			before := snapshotVault(t, vault)

			body, buildErr := build(vs1FactArgs("no-identity", vs1CallerUser, vs1CallerRepo))
			g.Expect(buildErr).NotTo(HaveOccurred())

			if buildErr != nil {
				return
			}

			resp := routeFor(t, cli.ServeRoutes(vs1ServerDeps(), vault, vs1ServerName, t.TempDir()), "/learn").
				Serve(t.Context(), cli.ServeRequest{Body: body})
			g.Expect(resp.Status).To(Equal(vs1StatusBadRequest), string(resp.Body))

			var failure struct {
				Error string `json:"error"`
			}
			g.Expect(json.Unmarshal(resp.Body, &failure)).To(Succeed())
			g.Expect(failure.Error).To(Equal("serve: user: must be non-empty"))
			g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")
		})
	}
}

// Test792a20f1a_ServeLearn_RepoReflectsTheCallersDetection verifies 792a20f1a (repo reflects the caller's
// detection): a served learn that declares repo "r" writes a note whose repo: is "r", although the server's own
// git resolves another repository.
func Test792a20f1a_ServeLearn_RepoReflectsTheCallersDetection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)

	resp := serveLearnWith(t, vs1ServerDeps(), vault, vs1FactArgs("repo-stamp", vs1CallerUser, "r"))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

	note := readFileString(t, filepath.Join(vault, decodeReceipt(t, resp).Basename+".md"))
	g.Expect(note).To(MatchRegexp(`(?m)^repo: r$`))
	g.Expect(note).NotTo(ContainSubstring(vs1ServerRepo), "the server's own repository is not substituted")
}

// Test792a20f1b_ServeLearn_VaultFallsBackToTheServersName verifies 792a20f1b (vault falls back to the server's
// name): with engram serve started with --vault-name personal, a served learn that sends no vault name writes a
// note whose vault: is personal; started with another name, the note carries that name instead, so the fallback is
// the started name and not a constant.
func Test792a20f1b_ServeLearn_VaultFallsBackToTheServersName(t *testing.T) {
	t.Parallel()

	for _, vaultName := range []string{"personal", "lab-notebook"} {
		t.Run(vaultName, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newServeVault(t)
			body, marshalErr := json.Marshal(vs1FactArgs("name-fallback", vs1CallerUser, vs1CallerRepo))
			g.Expect(marshalErr).NotTo(HaveOccurred())

			if marshalErr != nil {
				return
			}

			var (
				handlers = map[string]cli.ServeHandler{}
				resp     cli.ServeResponse
			)

			stderr := executeForTestWithDeps(t,
				[]string{
					"engram", "serve", "--addr", vs1Addr, "--vault", vault, "--vault-name", vaultName,
					"--chunks-dir", t.TempDir(),
				},
				func(deps *cli.Deps) {
					vs1Customize(deps)

					deps.Commander = fakeGitCommander{remoteOut: vs1ServerRepo, configOut: vs1ServerUser}
					deps.Username = func() (string, error) { return vs1ServerUser, nil }
					deps.NewServeMux = func() cli.RawServeMux { return "mux" }
					deps.RegisterRoute = func(_ cli.RawServeMux, method, pattern string, handler cli.ServeHandler) {
						handlers[method+" "+pattern] = handler
					}
					deps.ListenAndServe = func(ctx context.Context, _ cli.RawServeMux, _ string) error {
						resp = handlers["POST /learn"].Serve(ctx, cli.ServeRequest{Body: body})

						return nil
					}
				})

			g.Expect(handlers).To(HaveKey("POST /learn"), "the running server serves learn: %s", stderr)
			g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))

			note := readFileString(t, filepath.Join(vault, decodeReceipt(t, resp).Basename+".md"))
			g.Expect(note).To(MatchRegexp(`(?m)^vault: ` + regexp.QuoteMeta(vaultName) + `$`))
		})
	}
}

// Test792a20f1c_ServeLearn_AnInPlaceRewriteIsStampedFromItsOwnRequest verifies 792a20f1c (An in-place rewrite is
// stamped from its own request): a pending offer written with repo "r" by one request, and a same-origin served
// learn that declares no repo and another user, rewrite the pending note in place under the same basename and the
// rewritten note carries no repo: and the user: of the new request.
func Test792a20f1c_ServeLearn_AnInPlaceRewriteIsStampedFromItsOwnRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := vs1ServerDeps()
	vault := newServeVault(t)

	first := vs1FactArgs("rewritten", "first@example.com", "r")
	first.Offer = offerOf(childOrigin, "key-one", "")

	firstResp := serveLearnWith(t, deps, vault, first)
	g.Expect(firstResp.Status).To(Equal(vs1StatusOK), string(firstResp.Body))

	firstReceipt := decodeReceipt(t, firstResp)
	written := readFileString(t, filepath.Join(vault, firstReceipt.Basename+".md"))
	g.Expect(written).To(MatchRegexp(`(?m)^repo: r$`), "the pending offer was written with repo r")
	g.Expect(written).To(MatchRegexp(`(?m)^user: first@example\.com$`))

	second := vs1FactArgs("rewritten", "second@example.com", "")
	second.Object = "second object"
	second.Offer = offerOf(childOrigin, "key-two", "")

	secondResp := serveLearnWith(t, deps, vault, second)
	g.Expect(secondResp.Status).To(Equal(vs1StatusOK), string(secondResp.Body))

	secondReceipt := decodeReceipt(t, secondResp)
	g.Expect(secondReceipt.Basename).To(Equal(firstReceipt.Basename), "the pending note is rewritten in place")
	g.Expect(noteFiles(t, vault)).To(HaveLen(1), "no second pending note exists")

	rewritten := readFileString(t, filepath.Join(vault, secondReceipt.Basename+".md"))
	g.Expect(rewritten).To(ContainSubstring("object: second object"), "the note holds the new request's content")
	g.Expect(rewritten).To(ContainSubstring("pending: true"))
	g.Expect(rewritten).To(MatchRegexp(`(?m)^user: second@example\.com$`))
	g.Expect(rewritten).NotTo(MatchRegexp(`(?m)^repo:`), "the rewritten note carries no repo:")
	g.Expect(rewritten).NotTo(ContainSubstring("first@example.com"))
	g.Expect(rewritten).NotTo(ContainSubstring(vs1ServerRepo), "the server's own repository is not substituted")
}

// unexported constants.
const (
	vs1Addr             = "127.0.0.1:0"
	vs1CallerRepo       = "git@github.com:example/caller.git"
	vs1CallerUser       = "caller@example.com"
	vs1Date             = "2026-09-27"
	vs1Home             = "/home/vs1"
	vs1Phrase           = "the vault door of a served node"
	vs1RendezvousWait   = 2 * time.Second
	vs1ServerName       = "personal"
	vs1ServerRepo       = "git@github.com:example/server-own.git"
	vs1ServerUser       = "server-own@example.com"
	vs1StatusBadRequest = 400
	vs1StatusOK         = 200
	vs1WriterCount      = 2
)

// vs1RedirectRig is one served vault with notes, a chunk index and a decoy vault and chunk index a request names.
type vs1RedirectRig struct {
	vault       string
	routes      []cli.ServeRoute
	decoyVault  string
	decoyChunks string
	decoyBefore map[string]string
}

// vs1RendezvousFS holds each writer's note create until the other writer has reached its own, or vs1RendezvousWait
// has passed. A writer reaches its note create only after listing the vault to choose its Luhmann ID, so two writers
// that are not serialized by the vault lock both list the same state before either writes.
type vs1RendezvousFS struct {
	cli.EdgeFS

	mutex   sync.Mutex
	arrived int
	met     chan struct{}
}

// WriteFileExcl waits for the other writer when path is a note, then creates the file.
func (f *vs1RendezvousFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if strings.HasSuffix(path, ".md") {
		f.arrive()
	}

	err := f.EdgeFS.WriteFileExcl(path, data, perm)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}

	return nil
}

func (f *vs1RendezvousFS) arrivals() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.arrived
}

func (f *vs1RendezvousFS) arrive() {
	f.mutex.Lock()

	f.arrived++
	if f.arrived == vs1WriterCount {
		close(f.met)
	}

	f.mutex.Unlock()

	timer := time.NewTimer(vs1RendezvousWait)
	defer timer.Stop()

	select {
	case <-f.met:
	case <-timer.C:
	}
}

// vs1Customize wires the deterministic embedder, the fixed clock and an empty environment into a test's deps.
func vs1Customize(deps *cli.Deps) {
	deps.Embed = stubEmbedder{modelID: "test-model@4", dims: 4}
	deps.Now = vs1Now
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return vs1Home, nil }
}

// vs1FactArgs is a served learn body for a fact: a top-level note declared by user and repo.
func vs1FactArgs(slug, user, repo string) cli.LearnArgs {
	return cli.LearnArgs{
		Type: "fact", Slug: slug, Position: "top", Source: "child",
		Situation: "a served write about " + slug, Subject: "engram", Predicate: "serves", Object: slug + " object",
		Repo: repo, User: user,
	}
}

// vs1FlagLine returns the help line that describes flag: the first line of text that begins with the flag.
func vs1FlagLine(text, flag string) string {
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), flag+" ") {
			return line
		}
	}

	return ""
}

// vs1NewRedirectRig plants the server's vault (one served-alpha note and a chunk) and a decoy vault of the same
// shape, and returns the routes over the server's own.
func vs1NewRedirectRig(t *testing.T) vs1RedirectRig {
	t.Helper()

	deps := vs1ServerDeps()
	vault, chunksDir := t.TempDir(), t.TempDir()
	decoyVault, decoyChunks := t.TempDir(), t.TempDir()

	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "served-alpha", Position: "top", Source: "local",
		Situation: vs1Phrase, Subject: "engram", Predicate: "serves", Object: "the server's own note",
	})
	vs1PlantChunk(t, chunksDir, "the server's own chunk")

	learnLocal(t, deps, decoyVault, cli.LearnArgs{
		Type: "fact", Slug: "decoy-note", Position: "top", Source: "local",
		Situation: vs1Phrase, Subject: "engram", Predicate: "hides", Object: "the decoy note",
	})
	vs1PlantChunk(t, decoyChunks, "the decoy chunk")

	return vs1RedirectRig{
		vault:       vault,
		routes:      cli.ServeRoutes(deps, vault, vs1ServerName, chunksDir),
		decoyVault:  decoyVault,
		decoyChunks: decoyChunks,
		decoyBefore: vs1Snapshot(t, decoyVault, decoyChunks),
	}
}

// vs1NewRendezvousFS wraps inner so that note creates rendezvous.
func vs1NewRendezvousFS(inner cli.EdgeFS) *vs1RendezvousFS {
	return &vs1RendezvousFS{EdgeFS: inner, met: make(chan struct{})}
}

// vs1NoteWithSlug returns the path of the one note file of vault whose name ends in .<slug>.md.
func vs1NoteWithSlug(t *testing.T, vault, slug string) string {
	t.Helper()

	var found []string

	for _, note := range noteFiles(t, vault) {
		if strings.HasSuffix(note, "."+slug+".md") {
			found = append(found, note)
		}
	}

	if len(found) != 1 {
		t.Fatalf("want one note for slug %q in %s, got %v", slug, vault, found)
	}

	return found[0]
}

// vs1Now is the fixed clock of the scenarios.
func vs1Now() time.Time {
	return time.Date(2026, time.September, 27, 12, 0, 0, 0, time.UTC)
}

// vs1PlantChunk writes a one-record chunk index under dir whose vector is orthogonal to every note and query vector,
// so the chunk reaches a payload through the recency channel only.
func vs1PlantChunk(t *testing.T, dir, text string) {
	t.Helper()

	records := []chunk.Record{{
		Source: "/sessions/s1.jsonl", Anchor: "turn-1", ContentHash: chunk.HashText(text), Text: text,
		Vector: []float32{0, 1, 0, 0}, IngestedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	}}

	data, encodeErr := chunk.EncodeRecords(records)
	if encodeErr != nil {
		t.Fatalf("encoding chunks: %v", encodeErr)
	}

	const indexPerm = 0o600

	writeErr := os.WriteFile(filepath.Join(dir, "s1.jsonl"), data, indexPerm)
	if writeErr != nil {
		t.Fatalf("writing the chunk index: %v", writeErr)
	}
}

// vs1PlantNotes learns three notes into vault through the local learn path.
func vs1PlantNotes(t *testing.T, deps cli.Deps, vault string) {
	t.Helper()

	for _, slug := range []string{"served-alpha", "served-beta", "served-gamma"} {
		learnLocal(t, deps, vault, cli.LearnArgs{
			Type: "fact", Slug: slug, Position: "top", Source: "local",
			Situation: vs1Phrase + " " + slug, Subject: "engram", Predicate: "serves", Object: slug + " object",
		})
	}
}

// vs1ServerDeps is the served node's deps: real files and locks, the stub embedder, the fixed clock and a scripted
// git and OS user that resolve to the server's own identity, never to a caller's.
func vs1ServerDeps() cli.Deps {
	deps := serveTestDeps()
	deps.Now = vs1Now
	deps.Getenv = func(string) string { return "" }
	deps.Commander = fakeGitCommander{remoteOut: vs1ServerRepo, configOut: vs1ServerUser}
	deps.Username = func() (string, error) { return vs1ServerUser, nil }

	return deps
}

// vs1Snapshot maps every file under the given directories to its content.
func vs1Snapshot(t *testing.T, dirs ...string) map[string]string {
	t.Helper()

	merged := map[string]string{}

	for _, dir := range dirs {
		maps.Copy(merged, snapshotVault(t, dir))
	}

	return merged
}
