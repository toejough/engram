package cli_test

// Activate re-checks a linked local note against the parent (ruling S31,
// final review F1): the merged query keeps a linked local copy in place of
// its parent note, so the next use of that local copy is where a changed
// parent note comes back down as a pending offer (design D8, D12).

import (
	"context"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestActivate_LinkedLocalNoteRecheck covers the recheck's outcomes: an
// unchanged parent note writes nothing, a declined changed version writes
// nothing, a link under another parent vault is never checked, and an
// unreachable parent leaves the local activate a success.
func TestActivate_LinkedLocalNoteRecheck(t *testing.T) {
	t.Parallel()

	t.Run("equal hash writes nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.steady", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)
		before := env.noteFiles()
		shownBefore := len(env.parent.shown())

		stdout, stderr := env.run("activate", "--note", local)
		g.Expect(stderr).To(BeEmpty())
		g.Expect(stdout).To(BeEmpty())
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal(before))
		g.Expect(env.parent.shown()[shownBefore:]).To(Equal([]string{parentNote + ".md"}), "one raw show per link")
		g.Expect(env.parent.activated()).To(HaveLen(1), "a recheck never signals use; only the first pull did")
	})

	t.Run("declined changed version writes nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.declined-update", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)

		env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
		env.run("activate", "--note", local)

		update := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == local })
		g.Expect(update).To(HaveLen(1))

		if len(update) != 1 {
			return
		}

		env.run("amend", "--target", strings.TrimSuffix(update[0], ".md"), "--discard")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.declined()).To(HaveLen(1))

		env.run("activate", "--note", local)
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{local}), "a declined version is not pulled again")
	})

	t.Run("link under another parent vault is not checked", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.elsewhere", "c", ""))
		env.stampVault("")
		env.writeState("parent.json", `{"url":"`+parentURL+`","vault_id":"`+parentVaultID+`","failures":0}`)
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(seqID(77),
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

		env.run("activate", "--note", "3.2026-09-20.local.md")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.parent.requests()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
	})

	t.Run("reported vault differs from the link's", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.other-vault", "c", ""))
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(seqID(77),
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

		env.run("activate", "--note", "3.2026-09-20.local.md")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
	})

	t.Run("backed-off parent is not contacted", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.backed-off", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)
		env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
		env.writeState("parent.json", `{"url":"`+parentURL+`","vault_id":"`+parentVaultID+
			`","backoff_until":"2026-09-28T11:00:00Z","failures":3}`)
		requestsBefore := len(env.parent.requests())

		_, stderr := env.run("activate", "--note", local)
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(stderr).To(ContainSubstring("parent unreachable (retry after"))
		g.Expect(env.parent.requests()).To(HaveLen(requestsBefore))
		g.Expect(env.noteFiles()).To(Equal([]string{local}))
	})

	t.Run("self-parent writes nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		env.stampVault("")
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.self", "c", ""))
		env.parent.vaultID = env.localID()
		env.plant("3.2026-09-20.local.md", []byte(localNoteWithLinks(env.localID(),
			"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

		_, stderr := env.run("activate", "--note", "3.2026-09-20.local.md")
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(stderr).To(ContainSubstring("own ID"))
		g.Expect(env.noteFiles()).To(Equal([]string{"3.2026-09-20.local.md"}))
	})

	t.Run("unreachable parent leaves the local activate a success", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newWiringEnv(t)
		parentNote := env.parent.addNote(pulledFact("7.2026-09-01.offline", "c", ""))
		local := acceptedPulledCopy(t, env, parentNote)
		env.plantSidecar(local, "2026-02-02")

		env.parent.setDown(true)
		env.run("activate", "--note", local)
		g.Expect(env.exitCodes()).To(BeEmpty())
		g.Expect(env.noteFiles()).To(Equal([]string{local}))
		g.Expect(env.lastUsed(local)).To(Equal("2026-09-28"))
		g.Expect(env.parentCache().Failures).To(Equal(1))
	})
}

// TestActivate_ParentChangeRoundTrip (final review F1, F11): pull a
// parent note, accept it, change it on the parent; the merged query still
// returns the local copy, and activating that local copy pulls the
// changed parent note down as a new pending offer.
func TestActivate_ParentChangeRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.queryNotes = true
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.round-trip", "c", ""))
	local := acceptedPulledCopy(t, env, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))

	stdout, _ := env.run("query", "--phrase", "when pulling", "--chunks-dir", t.TempDir())
	g.Expect(env.exitCodes()).To(BeEmpty())

	var payload struct {
		Items []struct {
			Path       string `yaml:"path"`
			FromParent *bool  `yaml:"from_parent"`
		} `yaml:"items"`
	}
	g.Expect(yaml.Unmarshal([]byte(stdout), &payload)).To(Succeed())

	paths := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		paths = append(paths, item.Path)
		g.Expect(item.FromParent).NotTo(BeNil())

		if item.FromParent != nil {
			g.Expect(*item.FromParent).To(BeFalse(), "the linked local copy stands in for the parent note")
		}
	}

	g.Expect(paths).To(ContainElement(local))
	g.Expect(paths).NotTo(ContainElement(parentNote + ".md"))

	pulled, stderr := env.run("activate", "--note", local)
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	update := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == local })
	g.Expect(update).To(HaveLen(1))

	if len(update) != 1 {
		return
	}

	g.Expect(pulled).To(ContainSubstring(update[0]))

	copied := decodePulledCopy(t, []byte(readFileString(t, env.vault+"/"+update[0])))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{
		{Note: parentNote, Via: "pulled", Hash: env.parent.hashOf(t, parentNote)},
	}))

	env.run("activate", "--note", local)
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(2), "re-activating pulls the update only once")
}

// TestActivate_ServedParentPendingAndRecheck (final review F8, F1): against
// a real served parent (cli.ServeRoutes over a vault), a pending parent
// note is never pulled down, a live one is, and after the child accepts
// its copy and the parent's note changes, activating the local copy
// pulls the change down through the raw route.
func TestActivate_ServedParentPendingAndRecheck(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parentVault := newServeVault(t)
	writeVaultNote(t, parentVault, "8.2026-01-08.pending-offer.md", pendingOfferFactNote("8", ""))
	livePath := writeVaultNote(t, parentVault, "9.2026-01-09.live.md", liveFactNote("9", ""))

	env := newWiringEnv(t)
	routes := cli.ServeRoutes(serveTestDeps(), parentVault, "team", "")
	env.wrap = func(deps *cli.Deps) { deps.Fetch = servedFetch(routes) }

	_, stderr := env.run("activate", "--note", "8.2026-01-08.pending-offer.md")
	g.Expect(env.exitCodes()).To(Equal([]int{1}), "a pending parent note is not found")
	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(env.noteFiles()).To(BeEmpty())

	env.run("activate", "--note", "9.2026-01-09.live.md")
	g.Expect(env.exitCodes()).To(BeEmpty())

	copies := env.noteFiles()
	g.Expect(copies).To(HaveLen(1))

	if len(copies) != 1 {
		return
	}

	env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--clear-pending")
	g.Expect(env.exitCodes()).To(BeEmpty())

	writeVaultNote(t, "", livePath, strings.Replace(liveFactNote("9", ""), "object: c", "object: curated", 1))

	env.run("activate", "--note", copies[0])
	g.Expect(env.exitCodes()).To(BeEmpty())

	after := env.noteFiles()
	g.Expect(after).To(HaveLen(2))

	update := slices.DeleteFunc(after, func(name string) bool { return name == copies[0] })
	if len(update) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, update[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Vault).To(Equal(serverVaultID))
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{
		{Note: "9.2026-01-09.live", Via: "pulled", Hash: fileExchangeHash(t, livePath)},
	}))
}

// TestCurateNearOrder_PulledOffer (final review F10) traces the curate
// skill's near row for a pulled offer O folded into a live local note E
// that has no primary link. Amending E first drains at once with E still
// unlinked, so the bounce goes up as a learn-offer and E's primary becomes
// the parent's new pending note; folding first gives E O's pulled link as
// its primary, so the amend goes up as an amend-offer for the pulled
// note's parent counterpart (design D12 B1).
func TestCurateNearOrder_PulledOffer(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		foldFirst bool
		wantFor   string
	}{
		"amend then fold": {foldFirst: false, wantFor: ""},
		"fold then amend": {foldFirst: true, wantFor: "7.2026-09-01.near"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			parentNote := env.parent.addNote(pulledFact("7.2026-09-01.near", "c", ""))
			env.plant("3.2026-09-20.existing.md", []byte("---\ntype: fact\ntier: L2\nsituation: when local\n"+
				"subject: l\npredicate: m\nobject: n\nluhmann: \"3\"\ncreated: \"2026-09-20\"\nsource: test\n"+
				"user: alice\nvault: personal\n---\n\nInformation learned: when local, l m n.\n"))

			env.run("activate", "--note", parentNote+".md")

			offers := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool {
				return name == "3.2026-09-20.existing.md"
			})
			g.Expect(offers).To(HaveLen(1))

			if len(offers) != 1 {
				return
			}

			offer := strings.TrimSuffix(offers[0], ".md")
			amend := func() {
				env.run("amend", "--target", "3.2026-09-20.existing", "--object", "n plus the pulled claim")
			}
			fold := func() {
				env.run("amend", "--target", offer, "--discard", "--into", "3.2026-09-20.existing",
					"--expect-hash", env.exchangeHashOf(offers[0]))
			}

			if testCase.foldFirst {
				fold()
				amend()
			} else {
				amend()
				fold()
			}

			g.Expect(env.exitCodes()).To(BeEmpty())

			sent := env.parent.offers()
			g.Expect(sent).To(HaveLen(1))

			if len(sent) == 1 {
				g.Expect(sent[0].Offer.For).To(Equal(testCase.wantFor))
			}
		})
	}
}

// acceptedPulledCopy pulls parentNote down, clears the copy's pending
// marker, and returns the accepted local copy's file name.
func acceptedPulledCopy(t *testing.T, env *wiringEnv, parentNote string) string {
	t.Helper()

	env.run("activate", "--note", parentNote+".md")

	copies := env.linkedCopies(parentNote)
	if len(copies) != 1 {
		t.Fatalf("pulling %s wrote %v", parentNote, copies)
	}

	env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--clear-pending")

	if codes := env.exitCodes(); len(codes) != 0 {
		t.Fatalf("accepting %s: exit %v: %s", copies[0], codes, env.lastStderr)
	}

	return copies[0]
}

// servedFetch routes a child's parent requests straight to a served
// parent's route handlers (no network): the URL's path picks the route and
// its query string becomes the request's query.
func servedFetch(routes []cli.ServeRoute) func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
	return func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
		parsed, parseErr := url.Parse(target)
		if parseErr != nil {
			return cli.FetchResponse{}, parseErr
		}

		for _, route := range routes {
			if route.Method == method && route.Pattern == parsed.Path {
				resp := route.Handler.Serve(ctx, cli.ServeRequest{Query: parsed.Query(), Body: body})

				return cli.FetchResponse(resp), nil
			}
		}

		return cli.FetchResponse{Status: 404, Body: []byte(`{"error":"no route"}`)}, nil
	}
}
