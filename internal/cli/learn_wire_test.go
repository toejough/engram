package cli_test

// Wire-safety tests for served learn (design D7 "Wire safety",
// vault-note-identity "Exchange fields SHALL NOT be settable over the served
// API"; task 3.7).

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestLearnArgs_DecodesOfferButNoExchangeOrSkillFields: a served-learn body
// carrying xid, parent, aliases and the skill_* fields, in camelCase and in
// PascalCase, populates none of them, while offer decodes.
func TestLearnArgs_DecodesOfferButNoExchangeOrSkillFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`{"type":"fact","situation":"s",` + forgedExchangeKeys + `,` +
		`"offer":{"origin":"o:x","key":"k","for":"12.2026-01-01.t","path":["p"]}}`)

	var args cli.LearnArgs

	g.Expect(json.Unmarshal(body, &args)).To(Succeed())
	g.Expect(args.XID).To(BeEmpty())
	g.Expect(args.Parent).To(BeZero())
	g.Expect(args.Aliases).To(BeEmpty())
	g.Expect(args.SkillHash).To(BeEmpty())
	g.Expect(args.SkillKey).To(BeEmpty())
	g.Expect(args.SkillSource).To(BeEmpty())
	g.Expect(args.Offer).To(Equal(cli.LearnOffer{
		Origin: "o:x", Key: "k", For: "12.2026-01-01.t", Path: []string{"p"},
	}))
}

// TestLearnArgs_WireKeysAreTheKnownSetPlusOffer pins the JSON keys a served
// learn can set: the pre-exchange keys plus offer, and nothing else. A new
// LearnArgs field without json:"-" fails here.
func TestLearnArgs_WireKeysAreTheKnownSetPlusOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	want := []string{
		"action", "behavior", "body", "chunkSources", "doneWhen", "impact", "issue", "object", "offer",
		"pending", "position", "predicate", "project", "redFlags", "repo", "situation", "slug", "source",
		"subject", "supersedes", "tags", "target", "tier", "triggers", "type", "user", "vault", "vaultName",
	}

	got := make([]string, 0, len(want))

	for field := range reflect.TypeFor[cli.LearnArgs]().Fields() {
		if !field.IsExported() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "-" {
			got = append(got, name)
		}
	}

	slices.Sort(got)
	g.Expect(got).To(Equal(want))
}

// unexported constants.
const (
	forgedExchangeKeys = `"xid":"forgedxidcamel","Xid":"forgedxidpascal","XID":"forgedxidupper",` +
		`"parent":{"vault":"forgedparentcamel","links":[{"note":"forgedlinkcamel","via":"offered"}]},` +
		`"Parent":{"Vault":"forgedparentpascal","Links":[{"Note":"forgedlinkpascal","Via":"offered"}]},` +
		`"aliases":["forgedaliascamel"],"Aliases":["forgedaliaspascal"],` +
		`"skillHash":"forgedskillhash","SkillHash":"forgedskillhash",` +
		`"skillKey":"forgedskillkey","SkillKey":"forgedskillkey",` +
		`"skillSource":"forgedskillsource","SkillSource":"forgedskillsource"`
)
