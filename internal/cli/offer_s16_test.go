package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestTargets_B1NearFoldBouncesUpOnce (D12 B1): a near fold of a pulled
// note into L is offered once; after its receipt, a bookkeeping write and
// an equal-hash rewrite of L queue nothing.
func TestTargets_B1NearFoldBouncesUpOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.plant("3.2026-09-27.pulled.md", offerTestNote{
		pending: true, parentVault: parentVaultID, linkVia: "pulled", linkHash: "xh1:p",
	}.render(t))
	env.plant("4.2026-09-27.local.md",
		[]byte(strings.Replace(string(offerTestNote{}.render(t)), xidA, xidC, 1)))

	env.parent.setDown(true)
	env.run("amend", "--target", "4.2026-09-27.local", "--object", "folded claim")
	env.run("amend", "--target", "3.2026-09-27.pulled", "--discard")

	folded := filepath.Join(env.vault, "4.2026-09-27.local.md")
	env.parent.setStoredHash(mustHash(t, []byte(readFileString(t, folded))))
	env.parent.setDown(false)
	env.advance(time.Minute)
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())
	g.Expect(env.outboxEntries()).To(BeEmpty())

	env.run("amend", "--target", "4.2026-09-27.local", "--activate")
	env.run("amend", "--target", "4.2026-09-27.local", "--object", "folded claim")
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Object).To(Equal("folded claim"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// TestTargets_Learn_UnmintableXIDQueuesNothing (S16 item 3): when no xid
// can be minted the note is still written, and nothing is queued.
func TestTargets_Learn_UnmintableXIDQueuesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.randFails = true

	notePath := env.learnFact("no-xid")
	g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("xid:"))
	g.Expect(env.lastStderr).To(ContainSubstring("could not mint an xid"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(env.parent.requests()).To(BeEmpty())
}

// TestTargets_M14OriginIsParentWithdrawnAtSendTime (S16 item 2, D12): a
// served offer accepted while the parent's ID is unknown is queued, and
// withdrawn at send time once the drain learns its origin is the parent.
func TestTargets_M14OriginIsParentWithdrawnAtSendTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.plant("5.2026-09-27.served.md", offerTestNote{
		pending: true, origin: parentVaultID + ":" + xidB, path: []string{parentVaultID},
	}.render(t))

	env.run("amend", "--target", "5.2026-09-27.served", "--clear-pending")
	g.Expect(env.parent.offers()).To(BeEmpty())
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// TestTargets_NotOfferedPathsQueueNothing (S16 item 6): every CLI write
// path design D5 keeps local queues no offer and sends nothing.
func TestTargets_NotOfferedPathsQueueNothing(t *testing.T) {
	t.Parallel()

	chunkLine := `{"source":"s","anchor":"a","content_hash":"sha256:x","text":"t","vector":[1,0,0,0]}` + "\n"

	cases := map[string]func(env *wiringEnv){
		"--activate": func(env *wiringEnv) {
			env.plant("1.2026-09-27.note.md", offerTestNote{}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--activate")
		},
		"--clear-pending of a non-served note": func(env *wiringEnv) {
			env.plant("1.2026-09-27.note.md", offerTestNote{pending: true}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--clear-pending")
		},
		"--discard": func(env *wiringEnv) {
			env.plant("1.2026-09-27.note.md", offerTestNote{}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--discard")
		},
		"supersedes only": func(env *wiringEnv) {
			env.plant("1.2026-09-27.note.md", offerTestNote{}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--supersedes", "2.2026-09-27.old|updates|c")
		},
		"chunk-source only": func(env *wiringEnv) {
			chunks := env.tempDir()
			env.mustWrite(filepath.Join(chunks, "index.jsonl"), chunkLine)
			env.plant("1.2026-09-27.note.md", offerTestNote{}.render(env.t))
			env.run("amend", "--target", "1.2026-09-27.note", "--chunk-source", "s#a", "--chunks-dir", chunks)
			env.expectOutput("1.2026-09-27.note.md")
		},
		"a pending learn": func(env *wiringEnv) {
			env.runLearn(cli.LearnArgs{Pending: true})
		},
		"learn qa": func(env *wiringEnv) {
			env.run("learn", "qa", "--slug", "qa", "--question", "why?", "--answer", "because", "--source", "t")
			env.expectOutput(".md")
		},
		"identity backfill": func(env *wiringEnv) {
			raw := strings.Replace(string(offerTestNote{}.render(env.t)), "user: alice\nvault: personal\n", "", 1)
			env.plant("1.2026-09-27.note.md", []byte(raw))

			stamped, err := cli.ExportBackfillIdentity(
				context.Background(), env.vault, cli.ExportNewIdentityDeps(env.deps()), false)
			if err != nil || stamped != 1 {
				env.t.Fatalf("backfill stamped %d: %v", stamped, err)
			}
		},
		"a registration note": func(env *wiringEnv) {
			env.runLearn(cli.LearnArgs{SkillHash: "abc", SkillKey: "k", SkillSource: "~/s"})
		},
		"an amend whose content equals the parent link hash": func(env *wiringEnv) {
			env.learnFact("seed") // caches the parent's vault ID
			env.parent.reset()
			env.resetOutbox()
			env.plant("7.2026-09-27.linked.md",
				offerTestNote{parentVault: parentVaultID, linkVia: "offered", linkMatches: true}.render(env.t))
			env.run("amend", "--target", "7.2026-09-27.linked", "--object", "c")
			env.expectOutput("7.2026-09-27.linked.md")
		},
	}

	for name, write := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			write(env)

			g.Expect(env.lastStderr).NotTo(ContainSubstring("error"))
			g.Expect(env.parent.requests()).To(BeEmpty())
			g.Expect(env.outboxEntries()).To(BeEmpty())
		})
	}
}

// TestTargets_ParentIDProbeFailures: the drain's vault-ID probe keeps the
// offer queued and warns once when the parent answers 4xx, reports no ID
// (too old), or accepts the probe but is unreachable for the offer.
func TestTargets_ParentIDProbeFailures(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		configure func(*recordingParent)
		warning   string
	}{
		"4xx":               {func(p *recordingParent) { p.queryStatus = 400 }, "bad probe"},
		"no vault id":       {func(p *recordingParent) { p.queryNoID = true }, "too old"},
		"offer unreachable": {func(p *recordingParent) { p.learnDown = true }, "parent unreachable"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			testCase.configure(env.parent)

			env.learnFact("probe")
			g.Expect(env.parent.offers()).To(BeEmpty())
			g.Expect(env.outboxEntries()).To(HaveLen(1))
			g.Expect(nonEmptyLines(env.lastStderr)).To(HaveLen(1))
			g.Expect(env.lastStderr).To(ContainSubstring(testCase.warning))
		})
	}
}

// TestTargets_ParentSwitchIgnoresOldLinks (S16 item 1): after ENGRAM_PARENT
// changes, the drain learns the new parent's ID first; a write whose
// content equals the OLD parent's link hash is still offered, with no
// offer.for and no supersedes translated through old links.
func TestTargets_ParentSwitchIgnoresOldLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	oldParent := seqID(71)
	env := newWiringEnv(t)
	env.parent.vaultID = oldParent
	env.learnFact("seed") // caches the old parent's ID

	superseded := offerTestNote{parentVault: oldParent, linkVia: "offered", linkHash: "xh1:o"}.render(t)
	env.plant("8.2026-09-27.super.md", []byte(strings.Replace(string(superseded), xidA, xidD, 1)))

	linked := offerTestNote{parentVault: oldParent, linkVia: "offered", linkMatches: true}.render(t)
	linked = []byte(strings.Replace(string(linked), "xid: ",
		"supersedes:\n  - note: 8.2026-09-27.super\n    type: updates\n    claim: c\nxid: ", 1))
	env.plant("7.2026-09-27.linked.md", linked)

	env.parentURL = "http://new-parent:9"
	env.parent = &recordingParent{vaultID: seqID(72)}

	env.run("amend", "--target", "7.2026-09-27.linked", "--object", "c")

	requests := env.parent.requests()
	g.Expect(requests).NotTo(BeEmpty())
	g.Expect(requests[0].url).To(ContainSubstring("/query"))
	g.Expect(requests[0].url).To(ContainSubstring("dedupe-keys"))

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Offer.For).To(BeEmpty())
	g.Expect(offers[0].Supersedes).To(BeEmpty())
	g.Expect(readFileString(t, filepath.Join(env.vault, "7.2026-09-27.linked.md"))).
		To(ContainSubstring("vault: " + seqID(72)))
}

func (p *recordingParent) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log = nil
}

func (e *wiringEnv) expectOutput(suffix string) {
	e.t.Helper()

	if !strings.Contains(e.lastStdout, suffix) {
		e.t.Fatalf("command wrote %q (stderr %q), want a path ending %s", e.lastStdout, e.lastStderr, suffix)
	}
}

func (e *wiringEnv) mustWrite(path, content string) {
	e.t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		e.t.Fatal(err)
	}
}

func (e *wiringEnv) resetOutbox() {
	e.t.Helper()

	err := os.Remove(filepath.Join(e.vault, ".engram", "outbox.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		e.t.Fatal(err)
	}
}

// runLearn runs RunLearn through the production learn deps (for writes no
// CLI flag can produce: pending and registration notes).
func (e *wiringEnv) runLearn(args cli.LearnArgs) {
	e.t.Helper()

	args.Type, args.Slug, args.Vault, args.Position = "runbook", "direct", e.vault, "top"
	args.Situation, args.DoneWhen, args.Body, args.Source = "when direct", "done", "1. step", "t"

	var stdout, stderr strings.Builder

	deps := e.deps()
	deps.Stderr = &stderr

	err := cli.RunLearn(context.Background(), args, cli.ExportNewLearnDeps(deps), &stdout)
	if err != nil {
		e.t.Fatalf("learn: %v", err)
	}

	e.lastStdout, e.lastStderr = stdout.String(), stderr.String()
}
