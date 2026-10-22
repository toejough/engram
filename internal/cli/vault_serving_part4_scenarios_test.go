package cli_test

// Scenario tests for the vault-serving product capability, part 4 (861c2zzi): the receiving vault tags a served
// offer from its own vocabulary (792a20o). Spec: docs/specs/product/vault-serving.md.
//
// The scenarios run over the registered /learn route with the deps the neighbouring serve tests use: a vault in the
// test's own t.TempDir() (the recorded deviation, #860: the vault lock is a real flock and no in-memory EdgeFS models
// the exclusive create and the directory listing a learn needs), a fixed clock and a scripted git. The vault's
// vocabulary is planted the way the vocab commands leave it (term definition notes with sidecars). The embedder maps
// a text to the count of each cue word in it, so the term a note is nearest to is known from the words it holds. A
// pending offer and its rewrite are produced by the served learn itself. No network and no real home.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a20o1a_ServeLearn_AServedOfferIsTaggedByTheReceivingVault verifies 792a20o1a (A served offer is tagged by
// the receiving vault): a served learn that sends no tags, over a vault with a vocabulary, writes a pending offer
// whose tags are the vocab/ tags the vault's vocabulary assigns to its content: the terms near it, nearest first,
// and not the term it is far from. The term files are listed in the reverse of that order, so the order is the
// vocabulary's ranking and not the listing's.
func Test792a20o1a_ServeLearn_AServedOfferIsTaggedByTheReceivingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps, vault := vs4Server(t)

	// Two alpha cues and one bravo cue: nearest to alpha, near bravo, far from charlie.
	args := offeredFact("tagged", vs4Cue(vs4Alpha, vs4Alpha, vs4Bravo), vs4Offer("k1"))
	g.Expect(args.Tags).To(BeEmpty(), "the served learn sends no tags")

	resp, receipt := vs2Offer(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(receipt.Pending).To(BeTrue())

	notePath := vs4NotePath(vault, receipt)
	g.Expect(vs3Frontmatter(t, notePath)).To(HaveKeyWithValue("pending", true))
	g.Expect(vs4Tags(t, notePath)).To(Equal([]string{"vocab/" + vs4Alpha, "vocab/" + vs4Bravo}))
}

// Test792a20o1b_ServeLearn_ARewriteIsReTagged verifies 792a20o1b (A rewrite is re-tagged): a pending offer tagged
// for one subject, rewritten in place by a same-origin served learn whose content is about another, keeps its
// basename and carries the vocab/ tags the vocabulary assigns to the new content, with none of the old ones.
func Test792a20o1b_ServeLearn_ARewriteIsReTagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps, vault := vs4Server(t)

	_, first := vs2Offer(t, deps, vault, offeredFact("retagged", vs4Cue(vs4Alpha), vs4Offer("k1")))
	notePath := vs4NotePath(vault, first)
	g.Expect(vs4Tags(t, notePath)).To(Equal([]string{"vocab/" + vs4Alpha}), "the offer is first tagged for its content")

	resp, second := vs2Offer(t, deps, vault, offeredFact("retagged", vs4Cue(vs4Bravo), vs4Offer("k2")))
	g.Expect(resp.Status).To(Equal(vs1StatusOK), string(resp.Body))
	g.Expect(second.Basename).To(Equal(first.Basename), "the rewrite is in place")
	g.Expect(vs2PendingNotes(t, vault)).To(HaveLen(1))
	g.Expect(vs3Frontmatter(t, notePath)).To(HaveKeyWithValue("object", vs4Cue(vs4Bravo)), "the new content is written")
	g.Expect(vs4Tags(t, notePath)).To(Equal([]string{"vocab/" + vs4Bravo}), "the old term is gone, the new one assigned")
}

// unexported constants.
const (
	vs4Alpha   = "alpha"
	vs4Bravo   = "bravo"
	vs4Charlie = "charlie"
	vs4Dims    = 4
)

// vs4Embedder embeds a text as the count of each cue word in it: alpha, bravo and charlie, and a fourth, empty axis.
type vs4Embedder struct{}

// Dims is the vector length.
func (vs4Embedder) Dims() int { return vs4Dims }

// Embed counts the cue words in text.
func (vs4Embedder) Embed(_ context.Context, text string) ([]float32, error) {
	lower := strings.ToLower(text)

	return []float32{
		float32(strings.Count(lower, vs4Alpha)),
		float32(strings.Count(lower, vs4Bravo)),
		float32(strings.Count(lower, vs4Charlie)),
		0,
	}, nil
}

// ModelID matches the model the planted term sidecars are stamped with.
func (vs4Embedder) ModelID() string { return "test-model@4" }

// vs4Cue is the object of an offered fact: the cue words, which place the note on the vocabulary's axes.
func vs4Cue(words ...string) string {
	return strings.Join(words, " ")
}

// vs4NotePath is the path of the note a receipt names.
func vs4NotePath(vault string, receipt receiptBody) string {
	return filepath.Join(vault, receipt.Basename+".md")
}

// vs4Offer is the offer record of the child vault of childOrigin, with a key from tag.
func vs4Offer(tag string) cli.LearnOffer {
	return cli.LearnOffer{Origin: childOrigin, Key: vs2Key(tag), Path: []string{childVaultID}}
}

// vs4Server is the served node's deps with the cue embedder, and a vault with its ID and a vocabulary of three terms
// on the three axes. The files are named so that the listing runs bravo, alpha, charlie.
func vs4Server(t *testing.T) (cli.Deps, string) {
	t.Helper()

	deps := vs1ServerDeps()
	deps.Embed = vs4Embedder{}
	vault := newServeVault(t)

	vhs1PlantVocabTerm(t, vault, "5.2026-01-05.vocab-"+vs4Bravo+"-definition", []float32{0, 1, 0, 0})
	vhs1PlantVocabTerm(t, vault, "6.2026-01-06.vocab-"+vs4Alpha+"-definition", []float32{1, 0, 0, 0})
	vhs1PlantVocabTerm(t, vault, "7.2026-01-07.vocab-"+vs4Charlie+"-definition", []float32{0, 0, 1, 0})

	return deps, vault
}

// vs4Tags is the tags: list of the note at path.
func vs4Tags(t *testing.T, path string) []string {
	t.Helper()

	entries, isList := vs3Frontmatter(t, path)["tags"].([]any)
	if !isList {
		return nil
	}

	tags := make([]string, 0, len(entries))

	for _, entry := range entries {
		tag, isString := entry.(string)
		if !isString {
			t.Fatalf("%s has a tag that is not a string: %v", path, entry)
		}

		tags = append(tags, tag)
	}

	return tags
}
