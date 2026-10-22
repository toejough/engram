package cli_test

// Scenario tests for the parent-vault-client technical component, part 1 (861c2a): the request to the parent
// (792b37c2), the merged query (792b37c4), the gated pull-down (792b37c11), the raw envelope (792b37c12) and the
// pulled copy (792b37c13). Spec: docs/specs/technical/parent-vault-client.md.
//
// The CLI journeys run over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860)
// and a fake parent behind Deps.Fetch, with no network and no real home. The four interaction scenarios drive
// Deps.Fetch through an imptest mock (MockPvc1Fetch, generated from pvc1FetchFunc).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

//go:generate impgen pvc1FetchFunc --dependency --name MockPvc1Fetch

// Test792b37c11a1_Activate_AFailedCheckPausesThePullDown verifies 792b37c11a1 (A failed check pauses the
// pull-down): a failed location check, an active backoff window, a self-parent or an unusable exchange state each
// stop the pull-down of several refs with its one warning, and nothing is fetched or written.
func Test792b37c11a1_Activate_AFailedCheckPausesThePullDown(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		setup   func(env *wiringEnv)
		warning string
	}{
		"a failed location check": {
			setup:   func(env *wiringEnv) { env.stampVault("/somewhere/else") },
			warning: "failed its location check",
		},
		"an active backoff window": {
			setup: func(env *wiringEnv) {
				env.stampVault("")
				env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"backoff_until":%q,"failures":2}`,
					env.parentURL, env.now().Add(time.Minute).Format(time.RFC3339)))
			},
			warning: "parent unreachable (retry after",
		},
		"a cached parent that is this vault": {
			setup: func(env *wiringEnv) {
				env.stampVault("")
				env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"vault_id":%q,"failures":0}`,
					env.parentURL, env.localID()))
			},
			warning: "the parent reports this vault's own ID",
		},
		"an exchange state that cannot be read": {
			setup: func(env *wiringEnv) {
				env.stampVault("")
				env.writeState("parent.json", "{not json")
			},
			warning: "warning: activate: parent cache: parent.json",
		},
		"a vault ID that cannot be stamped": {
			setup:   func(env *wiringEnv) { env.randFails = true },
			warning: "could not stamp the vault ID",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			first := env.parent.addNote(pulledFact("7.2026-09-01.paused-a", "c", ""))
			second := env.parent.addNote(pulledFact("8.2026-09-01.paused-b", "c", ""))
			testCase.setup(env)

			_, stderr := env.run("activate", "--note", first+".md", "--note", second+".md")

			g.Expect(strings.Count(stderr, testCase.warning)).To(Equal(1), stderr)
			g.Expect(env.parent.requests()).To(BeEmpty(), "nothing is fetched")
			g.Expect(env.noteFiles()).To(BeEmpty(), "nothing is written")
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
		})
	}
}

// Test792b37c11a2_Activate_AnOutagePausesTheLaterRefs verifies 792b37c11a2 (An outage pauses the later refs): the
// first fetch of an unreachable parent records one failure and prints one backoff warning with the queued count,
// and the later refs make no request (the mock allows exactly one call).
func Test792b37c11a2_Activate_AnOutagePausesTheLaterRefs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)

	// one offer queued while the parent is down, then the backoff of that outage has passed
	env.parent.setDown(true)
	env.learnFact("queued")
	env.parent.setDown(false)
	env.advance(time.Hour)

	failuresBefore := env.parentCache().Failures

	fetch, fetchCalls := MockPvc1Fetch(t)
	env.wrap = func(deps *cli.Deps) { deps.Fetch = fetch }

	script := pvc1Script(env, fetchCalls, pvc1Step{method: "GET", route: "/show?", fail: errPvc1Transport})

	_, stderr := pvc1RunWithin(t, env, script,
		"activate", "--note", "7.2026-09-01.one.md", "--note", "8.2026-09-01.two.md", "--note", "9.2026-09-01.three.md")

	g.Expect(env.parentCache().Failures).To(Equal(failuresBefore + 1))
	g.Expect(strings.Count(stderr, "engram: parent unreachable (retry after ")).To(Equal(1), stderr)
	g.Expect(stderr).To(ContainSubstring("1 offer(s) queued"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
}

// Test792b37c11a3_Activate_AnUnrecordableFailureNamesTheError verifies 792b37c11a3 (An unrecordable failure names
// the error): an outage whose failure cannot be written to the parent cache is warned with that error, not with a
// zero retry time.
func Test792b37c11a3_Activate_AnUnrecordableFailureNamesTheError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.stampVault("")
	env.parent.setDown(true)
	env.wrap = func(deps *cli.Deps) { deps.FS = parentCacheWriteFailsFS{EdgeFS: deps.FS} }

	_, stderr := env.run("activate", "--note", "7.2026-09-01.gone.md")

	g.Expect(stderr).To(ContainSubstring("the backoff could not be recorded: parent cache: " +
		errParentCacheWrite.Error()))
	g.Expect(stderr).NotTo(ContainSubstring("retry after"))
	g.Expect(stderr).NotTo(ContainSubstring("0001-01-01"))
}

// Test792b37c11a4_Activate_APullDownDrainsTheOutboxOnceAfterTheLastRef verifies 792b37c11a4 (A pull-down drains the
// outbox): with an offer queued, a successful pull-down of two refs sends it once, after the last ref.
func Test792b37c11a4_Activate_APullDownDrainsTheOutboxOnceAfterTheLastRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	first := env.parent.addNote(pulledFact("7.2026-09-01.drain-a", "c", ""))
	second := env.parent.addNote(pulledFact("8.2026-09-01.drain-b", "c", ""))

	env.parent.setDown(true)
	env.learnFact("queued")
	env.parent.setDown(false)
	env.advance(time.Hour)

	before := len(env.parent.requests())

	env.run("activate", "--note", first+".md", "--note", second+".md")

	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.parent.offers()).To(HaveLen(1), "the offer is sent once")
	g.Expect(env.outboxEntries()).To(BeEmpty())

	requests := env.parent.requests()[before:]
	offerSent := slices.IndexFunc(requests, func(request recordedRequest) bool {
		return request.method == "POST" && strings.HasSuffix(request.url, "/learn")
	})
	lastPull := -1

	for index, request := range requests {
		if pvc1IsPullRequest(request) {
			lastPull = index
		}
	}

	g.Expect(lastPull).To(BeNumerically(">=", 0))
	g.Expect(offerSent).To(BeNumerically(">", lastPull), "the drain follows the last ref")
}

// Test792b37c12a1_Activate_AnOlderParentIsDetected verifies 792b37c12a1 (An older parent is detected): an answer
// that is a rendered note, or JSON without a vault_id, fails as the parent being too old and writes nothing.
func Test792b37c12a1_Activate_AnOlderParentIsDetected(t *testing.T) {
	t.Parallel()

	const basename = "7.2026-09-01.old"

	cases := map[string]func(t *testing.T, env *wiringEnv){
		"a rendered note": func(_ *testing.T, env *wiringEnv) { env.parent.showMode = "rendered" },
		"JSON with an empty vault_id": func(t *testing.T, env *wiringEnv) {
			t.Helper()

			env.wrap = func(deps *cli.Deps) {
				deps.Fetch = pvc1EnvelopeFetch(t, "", basename, pulledFact(basename, "c", "").content)
			}
		},
		"JSON that is not an object": func(_ *testing.T, env *wiringEnv) {
			env.wrap = func(deps *cli.Deps) { deps.Fetch = pvc1ShowAnswer(`["not","an","envelope"]`) }
		},
		"JSON with a basename and nothing else": func(_ *testing.T, env *wiringEnv) { env.parent.showMode = "no-vault-id" },
	}

	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.parent.addNote(pulledFact(basename, "c", ""))
			setup(t, env)

			_, stderr := env.run("activate", "--note", basename+".md")

			g.Expect(stderr).To(ContainSubstring("the parent is too old"))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.noteFiles()).To(BeEmpty())
		})
	}
}

// Test792b37c12a2_Activate_AMalformedEnvelopeWritesNothing verifies 792b37c12a2 (A malformed envelope writes
// nothing): a vault_id that is not an exchange identifier, or a basename holding a separator or a pipe, writes
// nothing and the malformed ID is never cached.
func Test792b37c12a2_Activate_AMalformedEnvelopeWritesNothing(t *testing.T) {
	t.Parallel()

	const goodBasename = "7.2026-09-01.malformed"

	cases := map[string]struct {
		vaultID, basename string
		badVaultID        bool
	}{
		"a vault_id that is not hex":     {vaultID: "../../not-a-vault", basename: goodBasename, badVaultID: true},
		"a vault_id that is too short":   {vaultID: "9a1e", basename: goodBasename, badVaultID: true},
		"a basename with a pipe":         {vaultID: parentVaultID, basename: goodBasename + "|claim"},
		"a basename with a slash":        {vaultID: parentVaultID, basename: "../7.2026-09-01.malformed"},
		"a basename with a backslash":    {vaultID: parentVaultID, basename: `7.2026-09-01\malformed`},
		"a basename that is not Luhmann": {vaultID: parentVaultID, basename: "malformed"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			note := pulledFact(goodBasename, "c", "")
			env.wrap = func(deps *cli.Deps) {
				deps.Fetch = pvc1EnvelopeFetch(t, testCase.vaultID, testCase.basename, note.content)
			}

			_, stderr := env.run("activate", "--note", goodBasename+".md")

			g.Expect(stderr).To(ContainSubstring("is malformed"))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.noteFiles()).To(BeEmpty())

			if testCase.badVaultID {
				g.Expect(env.parentCache().VaultID).To(BeEmpty(), "a malformed ID is never cached")
			}
		})
	}
}

// Test792b37c12a3_Activate_AMismatchedHashWritesNothing verifies 792b37c12a3 (A mismatched hash writes nothing):
// an envelope whose exchange_hash does not match its content is refused and nothing is written.
func Test792b37c12a3_Activate_AMismatchedHashWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.mismatch", "c", ""))
	env.parent.showMode = "wrong-hash"

	_, stderr := env.run("activate", "--note", parentNote+".md")

	g.Expect(stderr).To(ContainSubstring("exchange hash differs"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty())
}

// Test792b37c12a4_Activate_ANoteOfAnotherTypeIsRefused verifies 792b37c12a4 (A note of another type is refused): a
// fetched note that is not a fact, feedback or runbook is refused, naming its type, and nothing is written.
func Test792b37c12a4_Activate_ANoteOfAnotherTypeIsRefused(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"moc", "vocab", "guide"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(fakeParentNote{
				basename: "7.2026-09-01.other",
				content: "---\ntype: " + noteType + "\nluhmann: \"7\"\ncreated: \"2026-09-01\"\nuser: bob\nvault: team\n" +
					"---\n\n# A note of another type\n",
			})

			_, stderr := env.run("activate", "--note", parentNote+".md")

			g.Expect(stderr).To(ContainSubstring("only fact, feedback and runbook notes are pulled down"))
			g.Expect(stderr).To(ContainSubstring(fmt.Sprintf("is a %q note", noteType)))
			g.Expect(env.exitCodes()).To(Equal([]int{1}))
			g.Expect(env.noteFiles()).To(BeEmpty())
		})
	}
}

// Test792b37c12a5_Activate_ACRLFEnvelopePullsAsLF verifies 792b37c12a5 (A CRLF envelope pulls as LF): an envelope
// whose content has CRLF line endings is written as an LF pending copy linked at the parent's hash.
func Test792b37c12a5_Activate_ACRLFEnvelopePullsAsLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lfNote := pulledFact("7.2026-09-01.crlf", "c", "")
	crlf := strings.ReplaceAll(lfNote.content, "\n", "\r\n")
	g.Expect(crlf).To(ContainSubstring("\r\n"), "the parent's content has CRLF line endings")

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(fakeParentNote{basename: lfNote.basename, content: crlf})
	hash := env.parent.hashOf(t, parentNote)

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	raw := readFileString(t, filepath.Join(env.vault, files[0]))
	copied := decodePulledCopy(t, []byte(raw))

	g.Expect(raw).NotTo(ContainSubstring("\r"), "the pending copy is LF")
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: hash}}))
}

// Test792b37c12a6_Activate_TheFetchHoldsNoLockAndAnAnswerResetsTheBackoff verifies 792b37c12a6 (The fetch holds no
// lock and an answer resets the backoff): the raw fetch is made with no vault lock held, and the answer leaves the
// failure count at zero and the parent's vault ID cached.
func Test792b37c12a6_Activate_TheFetchHoldsNoLockAndAnAnswerResetsTheBackoff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.nolock", "c", ""))

	// three failures behind it, the window already over
	env.stampVault("")
	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"backoff_until":%q,"failures":3}`,
		env.parentURL, env.now().Add(-time.Minute).Format(time.RFC3339)))
	g.Expect(env.parentCache().Failures).To(Equal(3))

	probe := &lockProbe{}
	env.wrap = probe.wrap

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	g.Expect(probe.fetched()).To(ContainElement(ContainSubstring("/show?note=" + parentNote + ".md&raw=1")))
	g.Expect(probe.fetchesUnderLock()).To(BeEmpty(), "no fetch is made under the vault lock")
	g.Expect(env.parentCache().Failures).To(BeZero())
	g.Expect(env.parentCache().VaultID).To(Equal(parentVaultID))
}

// Test792b37c12a7_Activate_APendingParentNoteIsNotPulled verifies 792b37c12a7 (A pending parent note is not
// pulled): a real served parent answers not found for its pending note, and nothing is written.
func Test792b37c12a7_Activate_APendingParentNoteIsNotPulled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parentVault := newServeVault(t)
	writeVaultNote(t, parentVault, "8.2026-01-08.pending-offer.md", pendingOfferFactNote("8", ""))

	routes := cli.ServeRoutes(serveTestDeps(), parentVault, "team", "")
	statuses := &pvc1StatusLog{}

	env := newWiringEnv(t)
	env.wrap = func(deps *cli.Deps) { deps.Fetch = statuses.record(servedFetch(routes)) }

	_, stderr := env.run("activate", "--note", "8.2026-01-08.pending-offer.md")

	g.Expect(statuses.of("/show?")).To(Equal([]int{404}), "the parent answers not found")
	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty())
}

// Test792b37c13a1_Activate_TheCopyIsPendingLinkedAndStripped verifies 792b37c13a1 (The copy is pending, linked and
// stripped): a parent note carrying its own aliases, xid, parent link, skill fields, sources, supersedes and tags
// is pulled as a copy that carries none of them and holds pending true and a via pulled primary link with the
// envelope's hash and the parent's authorship.
func Test792b37c13a1_Activate_TheCopyIsPendingLinkedAndStripped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.stripped", "c", pvc1ParentOnlyFields))
	hash := env.parent.hashOf(t, parentNote)

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	frontmatter := pvc1PulledFrontmatter(t, env)
	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, env.noteFiles()[0]))))

	for _, stripped := range []string{
		"aliases", "offer", "skill_hash", "skill_key", "skill_source", "sources", "supersedes", "vocab_version",
	} {
		g.Expect(frontmatter).NotTo(HaveKey(stripped))
	}

	g.Expect(copied.Tags).NotTo(ContainElement("vocab/parent-term"))
	g.Expect(copied.XID).To(MatchRegexp(`^[0-9a-f]{32}$`))
	g.Expect(copied.XID).NotTo(Equal(xidC), "the parent's own xid is not kept")
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: hash}}),
		"the parent's own parent link is replaced by the one primary via pulled link")
	g.Expect(copied.Parent.Author).To(Equal(pulledAuthor{Repo: "github.com/parent/repo", User: "bob", Vault: "team"}))
}

// Test792b37c13a2_Activate_TheIdentityIsStampedLocally verifies 792b37c13a2 (The identity is stamped locally): a
// note by another author is pulled with this vault's repo, user and vault, and the author in parent.author.
func Test792b37c13a2_Activate_TheIdentityIsStampedLocally(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.wrap = func(deps *cli.Deps) {
		deps.Commander = fakeGitCommander{remoteOut: "github.com/child/repo\n", configOut: "alice@example.com\n"}
	}
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.identity", "c", ""))

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	frontmatter := pvc1PulledFrontmatter(t, env)
	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, env.noteFiles()[0]))))

	g.Expect(frontmatter).To(HaveKeyWithValue("repo", "github.com/child/repo"))
	g.Expect(frontmatter).To(HaveKeyWithValue("user", "alice@example.com"))
	g.Expect(frontmatter).To(HaveKeyWithValue("vault", "personal"))
	g.Expect(copied.Parent.Author).To(Equal(pulledAuthor{Repo: "github.com/parent/repo", User: "bob", Vault: "team"}))
}

// Test792b37c13a3_Activate_AnUndetectableUserIsOmitted verifies 792b37c13a3 (An undetectable user is omitted): on
// a host where user detection resolves empty the copy has no user key, the vault is stamped, and one warning says
// detection resolved empty.
func Test792b37c13a3_Activate_AnUndetectableUserIsOmitted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.wrap = withoutUserDetection
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.nouser", "c", ""))

	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	frontmatter := pvc1PulledFrontmatter(t, env)

	g.Expect(frontmatter).NotTo(HaveKey("user"), "the parent's user is not kept either")
	g.Expect(frontmatter).To(HaveKeyWithValue("vault", "personal"))
	g.Expect(strings.Count(stderr, "user detection resolved empty")).To(Equal(1), stderr)
}

// Test792b37c13a4_Activate_TheCopyHashesLikeTheParentsNote verifies 792b37c13a4 (The copy hashes like the parent's
// note): the written copy's exchange hash equals the envelope's, and the copy is embedded on write.
func Test792b37c13a4_Activate_TheCopyHashesLikeTheParentsNote(t *testing.T) {
	t.Parallel()

	for name, parentNote := range map[string]fakeParentNote{
		"a fact":    pulledFact("7.2026-09-01.hashed", "c", pvc1ParentOnlyFields),
		"a runbook": pulledRunbook("8.2026-09-02.hashed-steps"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.parent.addNote(parentNote)
			hash := env.parent.hashOf(t, parentNote.basename)

			stdout, _ := env.run("activate", "--note", parentNote.basename+".md")
			g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

			files := env.noteFiles()
			g.Expect(files).To(HaveLen(1))

			if len(files) != 1 {
				return
			}

			g.Expect(env.exchangeHashOf(files[0])).To(Equal(hash))
			g.Expect(filepath.Join(env.vault, strings.TrimSuffix(files[0], ".md")+".vec.json")).To(BeAnExistingFile())
			g.Expect(env.embeds.Load()).To(BeNumerically(">", 0), "the copy is embedded on write")
			g.Expect(strings.TrimSpace(stdout)).To(Equal(filepath.Join(env.vault, files[0])), "its path is printed")
		})
	}
}

// Test792b37c13a5_Activate_TheParentsTagsAreNotKept verifies 792b37c13a5 (The parent's tags are not kept): a
// parent note carrying vocabulary tags is pulled as a copy that does not keep them.
func Test792b37c13a5_Activate_TheParentsTagsAreNotKept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.tagged", "c",
		"tags:\n    - vocab/parent-term\n    - vocab/other-term\nvocab_version: v3\n"))

	env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, files[0]))))

	g.Expect(copied.Tags).NotTo(ContainElements("vocab/parent-term", "vocab/other-term"))
	g.Expect(copied.VocabVersion).To(BeEmpty())
}

// Test792b37c2a3_FetchQueryPayload_ATransportErrorIsReportedAsTheParentUnreachable verifies 792b37c2a3 (A
// transport error is unreachable): when the connection fails, the request is sent once to the parent's route and
// the error says the parent is unreachable.
func Test792b37c2a3_FetchQueryPayload_ATransportErrorIsReportedAsTheParentUnreachable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const base = "http://parent-host:8420"

	fetch, fetchCalls := MockPvc1Fetch(t)
	done := make(chan error, 1)

	go func() {
		_, err := cli.ExportFetchQueryPayload(t.Context(), cli.Deps{Fetch: fetch}, base,
			cli.QueryArgs{Phrases: []string{"x"}})
		done <- err
	}()

	// the expectation runs apart, so that a call that does not match it, which this version of imptest leaves waiting,
	// cannot hold the test: the test waits for the outcome with a time limit
	go func() {
		fetchCalls.ArgsShould(match.BeAny, Equal("GET"), HavePrefix(base+"/query?"), match.BeAny).
			Return(cli.FetchResponse{}, errPvc1Transport)
	}()

	select {
	case err := <-done:
		g.Expect(err).To(MatchError(ContainSubstring("parent unreachable")))
		g.Expect(err).To(MatchError(errPvc1Transport), "the transport error stays its cause")
	case <-time.After(mockTimeout):
		t.Fatal("the request was not sent to the parent's /query route as a GET")
	}
}

// Test792b37c4a3_Targets_Query_Merged_ASelfParentAnswerReturnsTheLocalResult verifies 792b37c4a3 (A self-parent
// answer returns the local result): a parent that reports this vault's own ID is not merged; the local results are
// returned, as an ordinary local query, with the self-parent warning.
func Test792b37c4a3_Targets_Query_Merged_ASelfParentAnswerReturnsTheLocalResult(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	pvc1PlantLocalNote(t, env.vault)
	env.stampVault("")
	env.parent.vaultID = env.localID()
	env.parent.queryNotes = true
	env.parent.addNote(pulledFact("7.2026-09-01.parent-only", "c", ""))

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(strings.Count(stderr, "the parent reports this vault's own ID")).To(Equal(1), stderr)

	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) != 1 {
		return
	}

	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
	g.Expect(parsed.Items[0].FromParent).To(BeNil(), "an ordinary local query tags nothing from_parent")
}

// Test792b37c4a5_Targets_Query_Merged_AFailingParentLeavesTheLocalAnswerAndOneWarning verifies 792b37c4a5 (A
// failing parent leaves the local answer): when the parent fails, the local results and one warning are returned.
func Test792b37c4a5_Targets_Query_Merged_AFailingParentLeavesTheLocalAnswerAndOneWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	pvc1PlantLocalNote(t, env.vault)
	env.parent.setDown(true)

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(strings.Count(stderr, "warning: query: parent unavailable, returning local-only results")).To(Equal(1))
	g.Expect(strings.Count(strings.TrimSpace(stderr), "\n")).To(BeZero(), "one warning and nothing else: "+stderr)

	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) != 1 {
		return
	}

	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
	g.Expect(env.parentCache().Failures).To(Equal(1), "the outage is recorded for backoff")
}

// Test792b37c4a6_Targets_Query_Merged_ABackedOffParentIsNotAsked verifies 792b37c4a6 (A backed-off parent is not
// asked): inside a backoff window the local results are returned and the mock Deps.Fetch is never called (a call
// would wait for an expectation the test does not give it, and the command would not finish).
func Test792b37c4a6_Targets_Query_Merged_ABackedOffParentIsNotAsked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	pvc1PlantLocalNote(t, env.vault)
	env.stampVault("")
	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"backoff_until":%q,"failures":2}`,
		env.parentURL, env.now().Add(time.Minute).Format(time.RFC3339)))

	fetch, _ := MockPvc1Fetch(t)
	env.wrap = func(deps *cli.Deps) { deps.Fetch = fetch }

	stdout, stderr := pvc1RunWithin(t, env, nil, "query", "--phrase", "x", "--chunks-dir", t.TempDir())

	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) != 1 {
		return
	}

	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
	g.Expect(strings.Count(stderr, "engram: parent unreachable (retry after ")).To(Equal(1), stderr)
}

// Test792b37c4a7_Targets_Query_Merged_AParentOnlyRunbookIsReturnedThenShown verifies 792b37c4a7 (A parent-only
// runbook is returned and then shown): the merged query returns a runbook only the parent holds, tagged
// from_parent, and show then finds it through the parent fallback.
func Test792b37c4a7_Targets_Query_Merged_AParentOnlyRunbookIsReturnedThenShown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	pvc1PlantLocalNote(t, env.vault)

	runbook := pulledRunbook("8.2026-09-02.parent-only-runbook")
	env.parent.addNote(runbook)
	env.parent.queryNotes = true
	env.parent.showMode = "rendered"

	queried, queryStderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(queryStderr).To(BeEmpty())

	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(queried), &parsed)).To(Succeed())

	tagged := make(map[string]*bool, len(parsed.Items))
	for _, item := range parsed.Items {
		tagged[item.Path] = item.FromParent
	}

	g.Expect(tagged).To(HaveKeyWithValue(runbook.basename+".md", HaveValue(BeTrue())))
	g.Expect(tagged).To(HaveKeyWithValue("1.fact.md", HaveValue(BeFalse())))

	shown, showStderr := env.run("show", runbook.basename+".md")

	g.Expect(showStderr).To(BeEmpty())
	g.Expect(shown).To(Equal("# from_parent: true\n" + runbook.content))
}

// Test792b37c4a8_Targets_Query_Merged_AMergedAnswerDrainsTheOutbox verifies 792b37c4a8 (A merged answer drains the
// outbox): with an offer queued, a successful merged query asks the parent for its answer, prints the merged
// answer, and only then sends the offer (the mock Deps.Fetch is driven in that order).
func Test792b37c4a8_Targets_Query_Merged_AMergedAnswerDrainsTheOutbox(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	pvc1PlantLocalNote(t, env.vault)

	env.parent.setDown(true)
	env.learnFact("queued")
	env.parent.setDown(false)
	env.advance(time.Hour)

	g.Expect(env.outboxEntries()).To(HaveLen(1), "an offer is queued")

	fetch, fetchCalls := MockPvc1Fetch(t)
	answer := &pvc1AnswerWriter{}

	env.wrap = func(deps *cli.Deps) {
		deps.Fetch = fetch
		answer.inner = deps.Stdout
		deps.Stdout = answer
	}

	var answeredBeforeOffer bool

	script := pvc1Script(env, fetchCalls,
		pvc1Step{method: "GET", route: "/query?"},
		pvc1Step{method: "POST", route: "/learn", seen: func() { answeredBeforeOffer = answer.wasWritten() }})

	stdout, stderr := pvc1RunWithin(t, env, script, "query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(ContainSubstring("1.fact.md"))
	g.Expect(answeredBeforeOffer).To(BeTrue(), "the offer is sent after the answer")
	g.Expect(env.parent.offers()).To(HaveLen(1))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// unexported constants.
const (
	// pvc1ParentOnlyFields adds the skill fields to parentOnlyFields: every field a pulled copy must not carry.
	pvc1ParentOnlyFields = parentOnlyFields + "skill_hash: abc123\nskill_source: ~/.claude/skills/x/SKILL.md\n"
)

// unexported variables.
var (
	errPvc1Transport = errors.New("dial tcp: connection refused")
)

// pvc1AnswerWriter wraps a command's stdout and remembers whether anything was written to it.
type pvc1AnswerWriter struct {
	inner   io.Writer
	mu      sync.Mutex
	written bool
}

func (w *pvc1AnswerWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	w.written = true
	w.mu.Unlock()

	return w.inner.Write(data)
}

func (w *pvc1AnswerWriter) wasWritten() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.written
}

// pvc1FetchFunc is the signature of cli.Deps.Fetch; impgen generates MockPvc1Fetch from it.
type pvc1FetchFunc func(ctx context.Context, method, url string, body []byte) (cli.FetchResponse, error)

// pvc1StatusEntry is one answer in a pvc1StatusLog.
type pvc1StatusEntry struct {
	url    string
	status int
}

// pvc1StatusLog records the status of every answer a fetch function gave, with the URL it answered.
type pvc1StatusLog struct {
	mu      sync.Mutex
	entries []pvc1StatusEntry
}

// of lists the statuses of the answers to the URLs containing route, in the order given.
func (l *pvc1StatusLog) of(route string) []int {
	l.mu.Lock()
	defer l.mu.Unlock()

	statuses := make([]int, 0, len(l.entries))

	for _, entry := range l.entries {
		if strings.Contains(entry.url, route) {
			statuses = append(statuses, entry.status)
		}
	}

	return statuses
}

// record wraps fetch so that each answer is logged.
func (l *pvc1StatusLog) record(
	fetch func(context.Context, string, string, []byte) (cli.FetchResponse, error),
) pvc1FetchFunc {
	return func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
		resp, err := fetch(ctx, method, target, body)

		l.mu.Lock()
		l.entries = append(l.entries, pvc1StatusEntry{url: target, status: resp.Status})
		l.mu.Unlock()

		return resp, err
	}
}

// pvc1Step is one request a pvc1Script expects from the code under test, in order: the method, a substring of the
// URL, an optional hook run when the request arrives, and an optional transport failure to answer with (without
// one the fake parent of the env answers).
type pvc1Step struct {
	method string
	route  string
	seen   func()
	fail   error
}

// pvc1EnvelopeFetch answers GET /show?raw=1 with an envelope holding the given vault ID, basename and content, and
// the exchange hash of the content; it answers POST /activate with ok.
func pvc1EnvelopeFetch(
	t *testing.T, vaultID, basename, content string,
) pvc1FetchFunc {
	t.Helper()

	hash, hashErr := cli.ExportExchangeHash([]byte(content))
	if hashErr != nil {
		t.Fatal(hashErr)
	}

	body, marshalErr := json.Marshal(map[string]string{
		"vault_id": vaultID, "basename": basename, "content": content, "exchange_hash": hash,
	})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}

	return func(_ context.Context, _, target string, _ []byte) (cli.FetchResponse, error) {
		if strings.Contains(target, "/show?") {
			return cli.FetchResponse{Status: 200, Body: body}, nil
		}

		return cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok"}`)}, nil
	}
}

// pvc1IsPullRequest reports whether a request to the parent belongs to the pull of a ref: its raw fetch or its
// signal of use.
func pvc1IsPullRequest(request recordedRequest) bool {
	return strings.Contains(request.url, "/show?") || strings.HasSuffix(request.url, "/activate")
}

// pvc1PlantLocalNote puts one local fact, with its sidecar, in the vault.
func pvc1PlantLocalNote(t *testing.T, vault string) {
	t.Helper()

	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})
}

// pvc1PulledFrontmatter is the frontmatter, as a map, of the one note the vault holds after a pull.
func pvc1PulledFrontmatter(t *testing.T, env *wiringEnv) map[string]any {
	t.Helper()

	files := env.noteFiles()
	if len(files) != 1 {
		t.Fatalf("the vault holds %v, want the one pulled copy", files)
	}

	raw, readErr := os.ReadFile(filepath.Join(env.vault, files[0]))
	if readErr != nil {
		t.Fatal(readErr)
	}

	frontmatter, _, found := embed.SplitFrontmatter(raw)
	if !found {
		t.Fatalf("%s has no frontmatter", files[0])
	}

	var fields map[string]any

	unmarshalErr := yaml.Unmarshal(frontmatter, &fields)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}

	return fields
}

// pvc1RunWithin runs `engram <args>` over env in a goroutine and waits for it, and for the script when one is
// given, with a time limit: a request the script does not expect leaves the command waiting on the mock, and a
// request the script expects but never gets leaves the script waiting.
func pvc1RunWithin(t *testing.T, env *wiringEnv, script <-chan struct{}, args ...string) (string, string) {
	t.Helper()

	type output struct{ stdout, stderr string }

	done := make(chan output, 1)

	go func() {
		stdout, stderr := env.run(args...)
		done <- output{stdout: stdout, stderr: stderr}
	}()

	var result output

	select {
	case result = <-done:
	case <-time.After(mockTimeout):
		t.Fatal("the command did not finish: it made a request to the parent that the script does not expect")
	}

	if script != nil {
		select {
		case <-script:
		case <-time.After(mockTimeout):
			t.Fatal("the parent did not get every request the script expects")
		}
	}

	return result.stdout, result.stderr
}

// pvc1Script drives the mock Deps.Fetch through steps, in order, and closes the returned channel when the last one
// is answered. Each request is answered by the env's fake parent, or fails when the step says so.
func pvc1Script(env *wiringEnv, calls *Pvc1FetchMockMethod, steps ...pvc1Step) <-chan struct{} {
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		for _, step := range steps {
			call := calls.ArgsShould(match.BeAny, Equal(step.method), ContainSubstring(step.route), match.BeAny)

			if step.seen != nil {
				step.seen()
			}

			if step.fail != nil {
				call.Return(cli.FetchResponse{}, step.fail)

				continue
			}

			args := call.GetArgs()
			resp, err := env.parent.fetch(args.Ctx, args.Method, args.Url, args.Body)
			call.Return(resp, err)
		}
	}()

	return finished
}

// pvc1ShowAnswer answers every request with 200 and body.
func pvc1ShowAnswer(body string) pvc1FetchFunc {
	return func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
		return cli.FetchResponse{Status: 200, Body: []byte(body)}, nil
	}
}
