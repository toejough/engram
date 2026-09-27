package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestCompareSkillOffers_AliasNeverOffers (task 2.3 property, postcondition):
// no register or refresh offer carries a hash some note already holds, and
// no two such offers share a hash; every offer's hash is the content of an
// enabled candidate with its key.
func TestCompareSkillOffers_AliasNeverOffers(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture := drawDedupeUniverse(rt)

		comparison, err := fixture.compareRapid()
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		noteHashes := map[string]bool{}
		for _, note := range fixture.notes {
			noteHashes[note.hash] = true
		}

		offeredHashes := map[string]string{}

		for _, offer := range comparison.Offers {
			if offer.Kind == cli.SkillOfferRemove {
				continue
			}

			if noteHashes[offer.Hash] {
				rt.Fatalf("offer %s %s repeats a note's skill_hash", offer.Kind, offer.Key)
			}

			if other, dup := offeredHashes[offer.Hash]; dup {
				rt.Fatalf("offers %s and %s share hash %s", other, offer.Key, offer.Hash)
			}

			offeredHashes[offer.Hash] = offer.Key

			if !fixture.hasEnabledCandidate(offer.Key, offer.Hash) {
				rt.Fatalf("offer %s has no enabled candidate with hash %s", offer.Key, offer.Hash)
			}
		}
	})
}

// TestCompareSkillOffers_Dedupe covers design D4: same-path collapse,
// same-key collapse and conflicts (no source is exempt), and aliases.
func TestCompareSkillOffers_Dedupe(t *testing.T) {
	t.Parallel()

	t.Run("entries resolving to one file collapse and both keys stay present", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		shared := "/real/x/SKILL.md"
		fixture := newOfferFixture()
		fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
		fixture.root(agentsUserRoot, cli.SkillRootFormAgentsUser, true)
		fixture.candidate(offerCand("claude:x", cli.SkillScopeClaudeUser, shared, "x body"))
		fixture.candidate(offerCand("agents:x", cli.SkillScopeAgentsUser, shared, "x body"))
		fixture.note("agents:x", "old-hash", "")

		comparison := fixture.compare(g)

		g.Expect(offerSummaries(comparison)).To(Equal([]string{"register claude:x " + shared}))
	})

	t.Run("same key with identical bytes collapses to one offer", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		for _, bucket := range []string{"b2", "b1"} {
			fixture.candidate(offerCand("anthropic-skills:pdf", cli.SkillScopeSynced,
				userSkillsRoot+"/synced/"+bucket+"/pdf/SKILL.md", "pdf"))
		}

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(BeEmpty())
		g.Expect(offerSummaries(comparison)).To(Equal([]string{
			"register anthropic-skills:pdf " + userSkillsRoot + "/synced/b1/pdf/SKILL.md",
		}))
	})

	t.Run("same key with different bytes is a conflict naming both paths", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/b/foo/SKILL.md", "foo b"))
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/a/foo/SKILL.md", "foo a"))
		fixture.candidate(offerCand("pi:bar", cli.SkillScopePiUser, piUserRoot+"/bar/SKILL.md", "bar"))

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(Equal([]string{
			"engram: skill key conflict: pi:foo at " + piUserRoot + "/a/foo/SKILL.md and " + piUserRoot + "/b/foo/SKILL.md",
		}))
		g.Expect(offerSummaries(comparison)).To(Equal([]string{"register pi:bar " + piUserRoot + "/bar/SKILL.md"}))
	})

	t.Run("three differing copies are named in one line", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, "/p/c/foo/SKILL.md", "c"))
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, "/p/a/foo/SKILL.md", "a"))
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, "/p/b/foo/SKILL.md", "b"))

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(Equal([]string{
			"engram: skill key conflict: pi:foo at /p/a/foo/SKILL.md, /p/b/foo/SKILL.md and /p/c/foo/SKILL.md",
		}))
		g.Expect(comparison.Offers).To(BeEmpty())
	})

	t.Run("a conflicted key's note is never offered for removal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/a/foo/SKILL.md", "foo a"))
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/b/foo/SKILL.md", "foo b"))
		fixture.note("pi:foo", "old", "")

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(HaveLen(1))
		g.Expect(comparison.Offers).To(BeEmpty())
	})

	t.Run("plugin name in two marketplaces: a loud line, no offers, no removals", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.sources.PluginManifestsRead = true
		fixture.sources.PluginConflicts = []cli.ClaudePluginConflict{
			{Plugin: "tools", Marketplaces: []string{"beta", "alpha"}},
		}
		fixture.sources.Plugins = []cli.ClaudePluginStatus{{Name: "tools"}}
		fixture.note("tools:lint", "h", "")

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(Equal([]string{"engram: plugin name conflict: tools in alpha, beta"}))
		g.Expect(comparison.Offers).To(BeEmpty())
	})

	t.Run("a diverged plugin skill and synced skill of one name are two skills", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("anthropic-skills:skill-creator", cli.SkillScopeSynced,
			"/s/skill-creator/SKILL.md", "synced"))
		fixture.candidate(offerCand("skill-creator:skill-creator", "plugin:skill-creator",
			"/p/skill-creator/SKILL.md", "plugin"))

		comparison := fixture.compare(g)

		g.Expect(offerSummaries(comparison)).To(Equal([]string{
			"register skill-creator:skill-creator /p/skill-creator/SKILL.md",
			"register anthropic-skills:skill-creator /s/skill-creator/SKILL.md",
		}))
	})

	t.Run("a copy of a higher-precedence candidate is an alias", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("pi:bar", cli.SkillScopePiUser, piUserRoot+"/bar/SKILL.md", "same"))
		fixture.candidate(offerCand("claude:foo", cli.SkillScopeClaudeUser, userSkillsRoot+"/foo/SKILL.md", "same"))

		comparison := fixture.compare(g)

		g.Expect(offerSummaries(comparison)).To(Equal([]string{"register claude:foo " + userSkillsRoot + "/foo/SKILL.md"}))
	})

	t.Run("a copy of any note's skill_hash is an alias", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.candidate(offerCand("pi:bar", cli.SkillScopePiUser, piUserRoot+"/bar/SKILL.md", "noted"))
		fixture.note("pi:other", cli.SkillContentHash([]byte("noted")), "")

		comparison := fixture.compare(g)

		g.Expect(comparison.Offers).To(BeEmpty())
	})

	t.Run("a Disabled candidate is never offered and is no alias source", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		disabled := offerCand("pi:ping", cli.SkillScopePiUser, piUserRoot+"/ping/SKILL.md", "ping")
		disabled.Disabled = true

		fixture := newOfferFixture()
		fixture.candidate(disabled)
		fixture.candidate(offerCand("agents:ping", cli.SkillScopeAgentsUser, agentsUserRoot+"/ping/SKILL.md", "ping"))

		comparison := fixture.compare(g)

		g.Expect(offerSummaries(comparison)).To(Equal([]string{
			"register agents:ping " + agentsUserRoot + "/ping/SKILL.md",
		}))
	})

	t.Run("a Disabled copy never conflicts with the enabled one", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		disabled := offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/a/foo/SKILL.md", "off")
		disabled.Disabled = true

		fixture := newOfferFixture()
		fixture.candidate(disabled)
		fixture.candidate(offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/b/foo/SKILL.md", "on"))

		comparison := fixture.compare(g)

		g.Expect(comparison.Conflicts).To(BeEmpty())
		g.Expect(offerSummaries(comparison)).To(Equal([]string{"register pi:foo " + piUserRoot + "/b/foo/SKILL.md"}))
	})
}

// TestCompareSkillOffers_OfferSetIndependentOfScanOrder (task 2.3 property,
// metamorphic): permuting the candidates, roots, plugin facts and the vault
// listing never changes the comparison.
func TestCompareSkillOffers_OfferSetIndependentOfScanOrder(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture := drawDedupeUniverse(rt)

		want, err := fixture.compareRapid()
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		shuffled := fixture.shuffled(rt)

		got, shuffledErr := shuffled.compareRapid()
		if shuffledErr != nil {
			rt.Fatalf("CompareSkillOffers (shuffled): %v", shuffledErr)
		}

		if fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", want) {
			rt.Fatalf("order changed the comparison:\nwant %+v\ngot  %+v", want, got)
		}
	})
}

// TestCompareSkillOffers_OffersCarryKeyScopeAndSource covers task 2.5:
// offers carry Key, ScopeID, SourcePath and Kind, sorted by (scope, key),
// and a removal's scope follows its key form.
func TestCompareSkillOffers_OffersCarryKeyScopeAndSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixture := newOfferFixture()
	fixture.home = fakeHome
	fixture.sources.PluginManifestsRead = true
	fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
	fixture.root(fakeHome+"/.claude/commands", cli.SkillRootFormClaudeCmd, true)
	fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
	fixture.root(agentsUserRoot, cli.SkillRootFormAgentsUser, true)
	fixture.root(piPromptsRoot, cli.SkillRootFormPiPrompt, true)
	fixture.root(userSkillsRoot+"/synced/b1", cli.SkillRootFormSynced, true, "anthropic-skills:")
	fixture.root(fakeHome+"/extra", cli.SkillRootFormPiSettings, true, "pi-settings:")
	fixture.root(piAgentDir+"/npm/node_modules/pk", cli.SkillRootFormPiPkg, true, "pi-pkg:pk:")
	fixture.root(projectTop+"/.claude/skills", cli.SkillRootFormProject, true, projectScope+":")

	fixture.candidate(offerCand("claude:zeta", cli.SkillScopeClaudeUser, userSkillsRoot+"/zeta/SKILL.md", "zeta"))
	fixture.candidate(offerCand("claude:alpha", cli.SkillScopeClaudeUser, userSkillsRoot+"/alpha/SKILL.md", "alpha"))
	fixture.note("claude:gone", "h1", "")
	fixture.note("claude:cmd:gone", "h2", "")
	fixture.note("pi:gone", "h3", "")
	fixture.note("agents:gone", "h4", "")
	fixture.note("pi-prompt:gone", "h5", "")
	fixture.note("anthropic-skills:gone", "h6", "~/.claude/skills/synced/b1/gone/SKILL.md")
	fixture.note("pi-settings:gone", "h7", "~/extra/gone/SKILL.md")
	fixture.note("pi-pkg:pk:gone", "h8", "~/.pi/agent/npm/node_modules/pk/skills/gone/SKILL.md")
	fixture.note(projectScope+":gone", "h9", projectTop+"/.claude/skills/gone/SKILL.md")
	fixture.note("ralph-loop:cmd:help", "h10", "")

	comparison := fixture.compare(g)

	type offerView struct {
		Kind                     cli.SkillOfferKind
		Key, ScopeID, SourcePath string
	}

	views := make([]offerView, 0, len(comparison.Offers))
	for _, offer := range comparison.Offers {
		views = append(views, offerView{offer.Kind, offer.Key, offer.ScopeID, offer.SourcePath})
	}

	g.Expect(views).To(Equal([]offerView{
		{cli.SkillOfferRemove, "agents:gone", "agents-user", ""},
		{cli.SkillOfferRemove, "claude:cmd:gone", "claude-cmd", ""},
		{cli.SkillOfferRegister, "claude:alpha", "claude-user", userSkillsRoot + "/alpha/SKILL.md"},
		{cli.SkillOfferRemove, "claude:gone", "claude-user", ""},
		{cli.SkillOfferRegister, "claude:zeta", "claude-user", userSkillsRoot + "/zeta/SKILL.md"},
		{cli.SkillOfferRemove, "pi-pkg:pk:gone", "pi-pkg:pk", "~/.pi/agent/npm/node_modules/pk/skills/gone/SKILL.md"},
		{cli.SkillOfferRemove, "pi-prompt:gone", "pi-prompt", ""},
		{cli.SkillOfferRemove, "pi-settings:gone", "pi-settings", "~/extra/gone/SKILL.md"},
		{cli.SkillOfferRemove, "pi:gone", "pi-user", ""},
		{cli.SkillOfferRemove, "ralph-loop:cmd:help", "plugin:ralph-loop", ""},
		{cli.SkillOfferRemove, projectScope + ":gone", projectScope, projectTop + "/.claude/skills/gone/SKILL.md"},
		{cli.SkillOfferRemove, "anthropic-skills:gone", "synced", "~/.claude/skills/synced/b1/gone/SKILL.md"},
	}))
}

// TestCompareSkillOffers_RemovalEligibilityPlugins covers design D5 for
// plugin notes: the manifest rule.
func TestCompareSkillOffers_RemovalEligibilityPlugins(t *testing.T) {
	t.Parallel()
	runRemovalCases(t, []removalCase{
		{
			name: "disabled plugin keeps its notes",
			setup: func(fixture *offerFixture) {
				fixture.sources.PluginManifestsRead = true
				fixture.sources.Plugins = []cli.ClaudePluginStatus{{Name: "hookify"}}
				fixture.note("hookify:writing-rules", "h", "")
			},
		},
		{
			name: "uninstalled plugin is offered for removal",
			setup: func(fixture *offerFixture) {
				fixture.sources.PluginManifestsRead = true
				fixture.sources.Plugins = []cli.ClaudePluginStatus{{Name: "hookify"}}
				fixture.note("ralph-loop:cmd:help", "h", "")
			},
			wantRemoved: []string{"ralph-loop:cmd:help"},
		},
		{
			name: "a scanned plugin's missing skill is removed",
			setup: func(fixture *offerFixture) {
				fixture.sources.PluginManifestsRead = true
				fixture.sources.Plugins = []cli.ClaudePluginStatus{{Name: "superpowers", Scanned: true}}
				fixture.note("superpowers:gone", "h", "")
			},
			wantRemoved: []string{"superpowers:gone"},
		},
		{
			name: "unread plugin manifests keep every plugin note",
			setup: func(fixture *offerFixture) {
				fixture.note("ralph-loop:cmd:help", "h", "")
			},
		},
		{
			name: "plugin conflict keeps the plugin's notes",
			setup: func(fixture *offerFixture) {
				fixture.sources.PluginManifestsRead = true
				fixture.sources.PluginConflicts = []cli.ClaudePluginConflict{{Plugin: "tools", Marketplaces: []string{"a", "b"}}}
				fixture.note("tools:lint", "h", "")
			},
		},
	})
}

// TestCompareSkillOffers_RemovalEligibilityProject covers design D5 for
// project notes: the exact project directory must be read and vouch for the key.
func TestCompareSkillOffers_RemovalEligibilityProject(t *testing.T) {
	t.Parallel()
	runRemovalCases(t, []removalCase{
		{
			name: "project note from another directory is kept",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note(projectScope+":openspec-propose", "h", projectSkills+"/openspec-propose/SKILL.md")
			},
		},
		{
			name: "project note's own folder read and skill gone is removed",
			setup: func(fixture *offerFixture) {
				fixture.root(projectSkills, cli.SkillRootFormProject, true, projectScope+":")
				fixture.note(projectScope+":openspec-propose", "h", projectSkills+"/openspec-propose/SKILL.md")
			},
			wantRemoved: []string{projectScope + ":openspec-propose"},
		},
		{
			name: "nested-only project skill is safe from the top level",
			setup: func(fixture *offerFixture) {
				fixture.root(projectSkills, cli.SkillRootFormProject, true, projectScope+":")
				fixture.root(projectTop+"/.claude/commands", cli.SkillRootFormProject, true, projectScope+":cmd:")
				fixture.note(projectScope+":foo", "h", projectTop+"/sub/.claude/skills/foo/SKILL.md")
			},
		},
		{
			name: "a project settings entry at the repo top never vouches for a Claude project note",
			setup: func(fixture *offerFixture) {
				fixture.root(projectTop, cli.SkillRootFormProject, true, projectScope+":pi-settings:")
				fixture.note(projectScope+":foo", "h", projectTop+"/sub/.claude/skills/foo/SKILL.md")
			},
		},
		{
			name: "a git repo nested in another project's .agents/skills keeps its notes",
			setup: func(fixture *offerFixture) {
				fixture.root(projectTop+"/.agents/skills", cli.SkillRootFormProject, true, projectScope+":agents:")
				fixture.note("project:github.com/x/inner:foo", "h",
					projectTop+"/.agents/skills/inner/.claude/skills/foo/SKILL.md")
			},
		},
		{
			name: "a read root of another scope nested inside the note's read root blocks it",
			setup: func(fixture *offerFixture) {
				fixture.root(projectSkills, cli.SkillRootFormProject, true, projectScope+":")
				fixture.root(projectSkills+"/foo", cli.SkillRootFormProject, true, projectScope+":pi-settings:")
				fixture.note(projectScope+":foo", "h", projectSkills+"/foo/SKILL.md")
			},
		},
		{
			name: "a package's gone prompt and a project's gone namespaced command are removed",
			setup: func(fixture *offerFixture) {
				fixture.root(piAgentDir+"/npm/node_modules/a", cli.SkillRootFormPiPkg, true,
					"pi-pkg:a:", "pi-pkg:a:pi-prompt:")
				fixture.root(projectTop+"/.claude/commands", cli.SkillRootFormProject, true, projectScope+":cmd:")
				fixture.note("pi-pkg:a:pi-prompt:p", "h1", piAgentDir+"/npm/node_modules/a/prompts/p.md")
				fixture.note(projectScope+":cmd:opsx:apply", "h2", projectTop+"/.claude/commands/opsx/apply.md")
			},
			wantRemoved: []string{"pi-pkg:a:pi-prompt:p", projectScope + ":cmd:opsx:apply"},
		},
		{
			name: "a skill root never vouches for a key with a further segment",
			setup: func(fixture *offerFixture) {
				fixture.root(projectSkills, cli.SkillRootFormProject, true, projectScope+":")
				fixture.note(projectScope+":pi:x", "h", projectSkills+"/x/SKILL.md")
			},
		},
		{
			name: "untrusted Pi project keeps its Pi project notes",
			setup: func(fixture *offerFixture) {
				fixture.root(projectSkills, cli.SkillRootFormProject, true, projectScope+":")
				fixture.note(projectScope+":pi:x", "h", projectTop+"/.pi/skills/x/SKILL.md")
			},
		},
	})
}

// TestCompareSkillOffers_RemovalEligibilitySourceRooted covers design D5 for
// synced, pi-settings and pi-pkg notes: the root holding skill_source must be read and vouch for the key.
func TestCompareSkillOffers_RemovalEligibilitySourceRooted(t *testing.T) {
	t.Parallel()
	runRemovalCases(t, []removalCase{
		{
			name: "two roots at one path must both be read, and one must vouch",
			setup: func(fixture *offerFixture) {
				fixture.root("/home/joe/same", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.root("/home/joe/same", cli.SkillRootFormPiSettings, false, "pi-settings:pi-prompt:")
				fixture.root("/home/joe/twin", cli.SkillRootFormPiSettings, true, "pi-settings:pi-prompt:")
				fixture.root("/home/joe/twin", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.note("pi-settings:x", "h1", "/home/joe/same/x/SKILL.md")
				fixture.note("pi-settings:y", "h2", "/home/joe/twin/y/SKILL.md")
			},
			wantRemoved: []string{"pi-settings:y"},
		},
		{
			name: "one failing synced bucket keeps its notes; the parsed bucket's note is removed",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(userSkillsRoot+"/synced", "", true)
				fixture.root(syncedA, cli.SkillRootFormSynced, true, "anthropic-skills:")
				fixture.root(syncedB, cli.SkillRootFormSynced, false, "anthropic-skills:")
				fixture.note("anthropic-skills:in-b", "h1", syncedB+"/in-b/SKILL.md")
				fixture.note("anthropic-skills:in-a", "h2", syncedA+"/in-a/SKILL.md")
			},
			wantRemoved: []string{"anthropic-skills:in-a"},
		},
		{
			name: "a bucket no longer listed is not its note's read root",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(userSkillsRoot+"/synced", "", true)
				fixture.root(syncedA, cli.SkillRootFormSynced, true, "anthropic-skills:")
				fixture.note("anthropic-skills:old", "h", userSkillsRoot+"/synced/gone/old/SKILL.md")
			},
		},
		{
			name: "no synced manifest parsed keeps every synced note",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(userSkillsRoot+"/synced", "", false)
				fixture.note("anthropic-skills:pdf", "h", syncedA+"/pdf/SKILL.md")
			},
		},
		{
			name: "pi-settings note is tied to its own unreadable entry",
			setup: func(fixture *offerFixture) {
				fixture.root("/home/joe/first", cli.SkillRootFormPiSettings, false, "pi-settings:")
				fixture.root("/home/joe/second", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.note("pi-settings:x", "h1", "/home/joe/first/x/SKILL.md")
				fixture.note("pi-settings:y", "h2", "/home/joe/second/y/SKILL.md")
			},
			wantRemoved: []string{"pi-settings:y"},
		},
		{
			name: "a failing entry nested in a read entry still keeps its note",
			setup: func(fixture *offerFixture) {
				fixture.root("/home/joe/a", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.root("/home/joe/a/b", cli.SkillRootFormPiSettings, false, "pi-settings:")
				fixture.note("pi-settings:x", "h", "/home/joe/a/b/x/SKILL.md")
			},
		},
		{
			name: "a pi-settings entry that is the skill file itself counts as its root",
			setup: func(fixture *offerFixture) {
				fixture.root("/home/joe/solo.md", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.note("pi-settings:solo", "h", "/home/joe/solo.md")
			},
			wantRemoved: []string{"pi-settings:solo"},
		},
		{
			name: "a package dropped from settings leaves an orphan",
			setup: func(fixture *offerFixture) {
				fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
				fixture.note("pi-pkg:x:foo", "h", piAgentDir+"/npm/node_modules/x/skills/foo/SKILL.md")
			},
		},
		{
			name: "a root of another form never proves a note absent",
			setup: func(fixture *offerFixture) {
				fixture.root("/work", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note(projectScope+":foo", "h1", projectSkills+"/foo/SKILL.md")
				fixture.note("anthropic-skills:bar", "h2", syncedA+"/bar/SKILL.md")
			},
		},
		{
			name: "a dropped package nested in another package's root is an orphan",
			setup: func(fixture *offerFixture) {
				fixture.root(piAgentDir+"/npm/node_modules/a", cli.SkillRootFormPiPkg, true,
					"pi-pkg:a:", "pi-pkg:a:pi-prompt:")
				fixture.note("pi-pkg:b:x", "h", piAgentDir+"/npm/node_modules/a/node_modules/b/skills/x/SKILL.md")
			},
		},
		{
			name: "a source-rooted note without skill_source is kept",
			setup: func(fixture *offerFixture) {
				fixture.root(syncedA, cli.SkillRootFormSynced, true, "anthropic-skills:")
				fixture.note("anthropic-skills:pdf", "h", "")
			},
		},
		{
			name: "a ~-relative skill_source without a home matches no root",
			setup: func(fixture *offerFixture) {
				fixture.home = ""
				fixture.root(fakeHome+"/extra", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.note("pi-settings:fmt", "h", "~/extra/fmt/SKILL.md")
			},
		},
		{
			name: "a ~-relative skill_source expands against home",
			setup: func(fixture *offerFixture) {
				fixture.root(fakeHome+"/extra", cli.SkillRootFormPiSettings, true, "pi-settings:")
				fixture.note("pi-settings:fmt", "h", "~/extra/fmt/SKILL.md")
			},
			wantRemoved: []string{"pi-settings:fmt"},
		},
	})
}

// TestCompareSkillOffers_RemovalEligibilityUserRoots covers design D5 for
// single-root user forms, aliases, disabled candidates, --skills-dir and declines.
func TestCompareSkillOffers_RemovalEligibilityUserRoots(t *testing.T) {
	t.Parallel()
	runRemovalCases(t, []removalCase{
		{
			name: "unreadable ~/.claude/skills keeps claude: notes",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, false)
				fixture.note("claude:c4", "h", "")
			},
		},
		{
			name: "readable ~/.claude/skills removes a gone claude: note",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note("claude:c4", "h", "")
			},
			wantRemoved: []string{"claude:c4"},
		},
		{
			name: "an alias keeps its note",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
				fixture.candidate(offerCand("claude:ping", cli.SkillScopeClaudeUser, userSkillsRoot+"/ping/SKILL.md", "ping"))
				fixture.candidate(offerCand("pi:ping", cli.SkillScopePiUser, piUserRoot+"/ping/SKILL.md", "ping"))
				fixture.note("pi:ping", "h", "")
			},
		},
		{
			name: "a Pi-disabled skill keeps its note",
			setup: func(fixture *offerFixture) {
				fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
				disabled := offerCand("pi:ping", cli.SkillScopePiUser, piUserRoot+"/ping/SKILL.md", "ping")
				disabled.Disabled = true
				fixture.candidate(disabled)
				fixture.note("pi:ping", "h", "")
			},
		},
		{
			name: "harness not installed keeps its notes",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note("pi:ping", "h", "")
			},
		},
		{
			name: "unreadable ~/.pi/agent/skills does not block a claude: removal",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(piUserRoot, cli.SkillRootFormPiUser, false)
				fixture.note("claude:route", "h", "")
			},
			wantRemoved: []string{"claude:route"},
		},
		{
			name: "unreadable ~/.agents/skills does not block a claude: removal",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.root(agentsUserRoot, cli.SkillRootFormAgentsUser, false)
				fixture.note("claude:route", "h", "")
			},
			wantRemoved: []string{"claude:route"},
		},
		{
			name: "unreadable ~/.claude/skills does not block a pi: or agents: removal",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, false)
				fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
				fixture.root(agentsUserRoot, cli.SkillRootFormAgentsUser, true)
				fixture.note("pi:route", "h", "")
				fixture.note("agents:route", "h2", "")
			},
			wantRemoved: []string{"pi:route", "agents:route"},
		},
		{
			name: "--skills-dir makes no removal offer",
			setup: func(fixture *offerFixture) {
				fixture.noRemovals = true
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note("claude:c4", "h", "")
			},
		},
		{
			name: "a declined removal at the note's hash is not re-offered",
			setup: func(fixture *offerFixture) {
				fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
				fixture.note("claude:c4", "h", "")
				fixture.declined["claude:c4"] = "h"
			},
		},
	})
}

// TestCompareSkillOffers_RemovalOnlyWhenOwnRootReadAndKeyAbsent (task 2.4
// property, model-based): each drawn note gets an intended root state and,
// for source-rooted forms, maybe a read decoy root of the same form but
// another scope or segment, around or inside its own root; a removal is
// offered exactly when its own root was read, no decoy hides it, and no
// candidate, alias or disabled one included, holds the note's key.
func TestCompareSkillOffers_RemovalOnlyWhenOwnRootReadAndKeyAbsent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture, expected := drawRemovalUniverse(rt)

		comparison, err := fixture.compareRapid()
		if err != nil {
			rt.Fatalf("CompareSkillOffers: %v", err)
		}

		got := removedKeys(comparison)
		slices.Sort(got)

		if strings.Join(got, ",") != strings.Join(expected, ",") {
			rt.Fatalf("removals: want %v, got %v", expected, got)
		}
	})
}

// TestCompareSkillOffers_ResolvedPluginsAndSources runs the spec scenarios for
// plugins, synced buckets, Pi settings entries and packages end to end: a fake home through ResolveSkillSources, then
// CompareSkillOffers against a fake vault.
func TestCompareSkillOffers_ResolvedPluginsAndSources(t *testing.T) {
	t.Parallel()

	t.Run("an uninstalled plugin's note is offered for removal; the installed one's kept", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		notes := []offerNote{
			{key: "ralph-loop:cmd:help", hash: "h1"},
			{key: "tools:tl", hash: cli.SkillContentHash([]byte("tl"))},
		}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"ralph-loop:cmd:help"}))
	})

	t.Run("a disabled plugin keeps its notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		writePluginManifestsAt(g, fsys, fakeHome+"/.claude",
			[]pluginInstall{{key: "tools@m", installPath: resolverPluginCache + "/m/tools/1.0"}},
			map[string]bool{"tools@m": false},
		)
		notes := []offerNote{{key: "tools:tl", hash: "h"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("a plugin in two marketplaces is loud and keeps its notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		writePluginManifestsAt(g, fsys, fakeHome+"/.claude",
			[]pluginInstall{
				{key: "tools@alpha", installPath: resolverPluginCache + "/m/tools/1.0"},
				{key: "tools@beta", installPath: resolverPluginCache + "/m/tools/1.0"},
			},
			map[string]bool{"tools@alpha": true, "tools@beta": true},
		)
		notes := []offerNote{{key: "tools:gone", hash: "h"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(comparison.Conflicts).To(Equal([]string{"engram: plugin name conflict: tools in alpha, beta"}))
		g.Expect(removedKeys(comparison)).To(BeEmpty())
		g.Expect(offeredKeys(comparison)).NotTo(ContainElement(HavePrefix("tools:")))
	})

	t.Run("one failing synced bucket keeps its notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).file(userSkillsRoot+"/synced/b2/manifest.json", "not json")
		notes := []offerNote{
			{key: "anthropic-skills:in-b2", hash: "h1", source: userSkillsRoot + "/synced/b2/in-b2/SKILL.md"},
			{key: "anthropic-skills:in-b1", hash: "h2", source: "~/.claude/skills/synced/b1/in-b1/SKILL.md"},
		}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"anthropic-skills:in-b1"}))
	})

	t.Run("a pi-settings note is tied to its own entry", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).
			file(piAgentDir+"/settings.json", `{"skills": ["~/first", "~/extra-skills"]}`).
			dir(fakeHome + "/first").
			failRead(fakeHome + "/first")
		notes := []offerNote{
			{key: "pi-settings:x", hash: "h1", source: "~/first/x/SKILL.md"},
			{key: "pi-settings:y", hash: "h2", source: "~/extra-skills/y/SKILL.md"},
		}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"pi-settings:y"}))
	})

	t.Run("a failed settings entry reached through a symlink still guards its notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).
			file(piAgentDir+"/settings.json", `{"skills": ["~/outer", "~/lnk"]}`).
			file(fakeHome+"/outer/keep/SKILL.md", "keep").
			link(fakeHome+"/lnk", fakeHome+"/outer/inner")
		notes := []offerNote{
			{key: "pi-settings:x", hash: "h1", source: "~/outer/inner/x/SKILL.md"},
			{key: "pi-settings:y", hash: "h2", source: "~/outer/y/SKILL.md"},
		}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"pi-settings:y"}))
	})

	t.Run("a package dropped from settings leaves an orphan", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).file(piAgentDir+"/settings.json", `{"skills": ["~/extra-skills"]}`)
		notes := []offerNote{{key: "pi-pkg:pk:gone", hash: "h", source: pkGoneSource}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("a listed package's gone skill is offered for removal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		notes := []offerNote{{key: "pi-pkg:pk:gone", hash: "h", source: pkGoneSource}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"pi-pkg:pk:gone"}))
	})
}

// TestCompareSkillOffers_ResolvedProject runs the spec scenarios for
// project sources end to end (conflicts across chain levels, removals
// scoped to the exact project directory): a fake home through
// ResolveSkillSources, then CompareSkillOffers against a fake vault.
func TestCompareSkillOffers_ResolvedProject(t *testing.T) {
	t.Parallel()

	t.Run("same project skill name at two levels is a conflict from sub/", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newFakeSkillFS().dir(fakeHome+"/.claude").
			file(projectTop+"/.claude/skills/foo/SKILL.md", "top foo").
			file(projectTop+"/sub/.claude/skills/foo/SKILL.md", "sub foo").
			file(projectTop+"/.claude/skills/bar/SKILL.md", "bar")

		comparison := resolveAndCompare(g, fsys, projectTop+"/sub", resolverDeps(fsys), nil)

		g.Expect(comparison.Conflicts).To(Equal([]string{
			"engram: skill key conflict: " + projectScope + ":foo at " + projectTop + "/.claude/skills/foo/SKILL.md and " +
				projectTop + "/sub/.claude/skills/foo/SKILL.md",
		}))
		g.Expect(offeredKeys(comparison)).To(Equal([]string{projectScope + ":bar"}))
	})

	t.Run("nested-only project note is kept when run from the top level", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newFakeSkillFS().dir(fakeHome+"/.claude").file(projectTop+"/.claude/skills/bar/SKILL.md", "bar")
		notes := []offerNote{{key: projectScope + ":foo", hash: "h", source: projectTop + "/sub/.claude/skills/foo/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, projectTop, resolverDeps(fsys), notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("a gone project skill in the read folder is offered for removal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newFakeSkillFS().dir(fakeHome+"/.claude").file(projectTop+"/.claude/skills/bar/SKILL.md", "bar")
		notes := []offerNote{{key: projectScope + ":foo", hash: "h", source: projectTop + "/.claude/skills/foo/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, projectTop, resolverDeps(fsys), notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{projectScope + ":foo"}))
	})

	t.Run("update outside a repository keeps project notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		notes := []offerNote{{key: projectScope + ":foo", hash: "h", source: projectTop + "/.claude/skills/foo/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("an untrusted Pi project keeps its Pi project notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).remove(piAgentDir + "/trust.json")
		notes := []offerNote{{key: projectScope + ":pi:gone", hash: "h", source: projectTop + "/.pi/skills/gone/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, projectTop, resolverDeps(fsys), notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("a trusted Pi project's gone skill is offered for removal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		notes := []offerNote{{key: projectScope + ":pi:gone", hash: "h", source: projectTop + "/.pi/skills/gone/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, projectTop, resolverDeps(fsys), notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{projectScope + ":pi:gone"}))
	})

	t.Run("a project settings entry of .. never exposes a nested-only project note", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).
			file(projectTop+"/.pi/settings.json", `{"skills": [".."]}`).
			file(projectTop+"/sub/.claude/skills/foo/SKILL.md", "foo")
		notes := []offerNote{{key: projectScope + ":foo", hash: "h", source: projectTop + "/sub/.claude/skills/foo/SKILL.md"}}

		comparison := resolveAndCompare(g, fsys, projectTop, resolverDeps(fsys), notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})
}

// TestCompareSkillOffers_ResolvedUserAndPi runs the spec scenarios for
// the user roots and the Pi harness end to end: a fake home through ResolveSkillSources, then
// CompareSkillOffers against a fake vault.
func TestCompareSkillOffers_ResolvedUserAndPi(t *testing.T) {
	t.Parallel()

	t.Run("unreadable ~/.claude/skills keeps claude: notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).failRead(userSkillsRoot)
		notes := []offerNote{{key: "claude:gone", hash: "h"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})

	t.Run("readable ~/.claude/skills offers a gone claude: note for removal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g)
		notes := []offerNote{{key: "claude:gone", hash: "h"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(Equal([]string{"claude:gone"}))
	})

	t.Run("a Pi-disabled skill makes no offer of any kind", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newFakeSkillFS().
			file(piUserRoot+"/ping/SKILL.md", "ping").
			file(piAgentDir+"/settings.json", `{"skills": ["!ping"]}`)
		notes := []offerNote{{key: "pi:ping", hash: "stale"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(comparison.Offers).To(BeEmpty())
	})

	t.Run("no Pi harness keeps pi notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := resolverFixture(g).remove(fakeHome + "/.pi")
		notes := []offerNote{{key: "pi:ping", hash: "h"}}

		comparison := resolveAndCompare(g, fsys, "/tmp", cli.SkillSourceDeps{FS: fsys, Commander: scriptedGit{}}, notes)

		g.Expect(removedKeys(comparison)).To(BeEmpty())
	})
}

// TestReportSkillOfferProblems: warnings and conflict lines are printed, and
// any conflict is a failure; without one it is silent success.
func TestReportSkillOfferProblems(t *testing.T) {
	t.Parallel()

	t.Run("conflicts print and fail", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		err := cli.ReportSkillOfferProblems(&out, cli.SkillOfferComparison{
			Conflicts: []string{"engram: skill key conflict: x at /a and /b"},
			Warnings:  []string{"engram: w"},
		})

		g.Expect(err).To(MatchError(cli.ErrSkillOfferConflictForTest))
		g.Expect(out.String()).To(Equal("engram: w\nengram: skill key conflict: x at /a and /b\n"))
	})

	t.Run("warnings alone print and succeed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		err := cli.ReportSkillOfferProblems(&out, cli.SkillOfferComparison{Warnings: []string{"engram: w"}})

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.String()).To(Equal("engram: w\n"))
	})

	t.Run("nothing to report is silent", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var out bytes.Buffer

		g.Expect(cli.ReportSkillOfferProblems(&out, cli.SkillOfferComparison{})).To(Succeed())
		g.Expect(out.String()).To(BeEmpty())
	})
}

// TestResolveSkillPathBestEffort: a root that cannot be resolved still gets
// the real location it points into, up to the failing component.
func TestResolveSkillPathBestEffort(t *testing.T) {
	t.Parallel()

	fsys := newFakeSkillFS().
		dir("/real/ok").
		link("/home/joe/ok", "/real/ok").
		link("/home/joe/lnk", "/real/gone/inner").
		dir("/real/denied").failLstat("/real/denied").
		link("/home/joe/via", "/real/denied").
		link("/loop/a", "/loop/b").link("/loop/b", "/loop/a")

	testCases := map[string]string{
		"/home/joe/ok":      "/real/ok",
		"/home/joe/lnk":     "/real/gone/inner",
		"/home/joe/via/x":   "/real/denied/x",
		"relative/path":     "relative/path",
		"/loop/a/skills":    "/loop/a/skills",
		"/home/joe/missing": "/home/joe/missing",
	}

	for path, want := range testCases {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.ExportResolveSkillPathBestEffort(fsys, path)).To(Equal(want))
		})
	}
}

// TestResolveSkillPathBestEffort_FollowsLinkChainsToAMissingTarget
// (property): a chain of symlinks ending at a missing path resolves to that
// path, whatever the chain's length and link forms.
func TestResolveSkillPathBestEffort_FollowsLinkChainsToAMissingTarget(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		hops := rapid.IntRange(0, 5).Draw(rt, "hops")
		leaf := "/real/" + rapid.StringMatching(`[a-z]{1,6}`).Draw(rt, "leaf") + "/SKILL.md"
		fsys := newFakeSkillFS().dir("/real")
		target := leaf

		for hop := hops; hop > 0; hop-- {
			link := fmt.Sprintf("/links/l%d", hop)
			fsys.link(link, target)
			target = link
		}

		if got := cli.ExportResolveSkillPathBestEffort(fsys, target); got != leaf {
			rt.Fatalf("best effort of %s: got %s, want %s", target, got, leaf)
		}
	})
}

// TestResolveSkillSources_StampsRootForms: every read root carries the
// removal-eligibility form of the notes it can prove absent, and each
// source-rooted root the key prefixes it vouches for; the synced directory
// itself and plugin roots carry neither.
func TestResolveSkillSources_StampsRootForms(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(resolverFixture(g)))
	g.Expect(err).NotTo(HaveOccurred())

	forms := map[string]cli.SkillRootForm{}
	for _, root := range resolved.Roots {
		forms[root.Path] = root.Form
	}

	g.Expect(forms).To(Equal(map[string]cli.SkillRootForm{
		userSkillsRoot:                              cli.SkillRootFormClaudeUser,
		fakeHome + "/.claude/commands":              cli.SkillRootFormClaudeCmd,
		userSkillsRoot + "/synced":                  "",
		userSkillsRoot + "/synced/b1":               cli.SkillRootFormSynced,
		piUserRoot:                                  cli.SkillRootFormPiUser,
		agentsUserRoot:                              cli.SkillRootFormAgentsUser,
		piPromptsRoot:                               cli.SkillRootFormPiPrompt,
		fakeHome + "/extra-skills":                  cli.SkillRootFormPiSettings,
		fakeHome + "/extra-prompts":                 cli.SkillRootFormPiSettings,
		piAgentDir + "/npm/node_modules/pk":         cli.SkillRootFormPiPkg,
		resolverPluginCache + "/m/tools/1.0":        "",
		resolverPluginCache + "/m/tools/1.0/skills": "",
		projectTop + "/.claude/skills":              cli.SkillRootFormProject,
		projectTop + "/.claude/commands":            cli.SkillRootFormProject,
		projectTop + "/.pi/skills":                  cli.SkillRootFormProject,
		projectTop + "/.agents/skills":              cli.SkillRootFormProject,
		projectTop + "/.pi/prompts":                 cli.SkillRootFormProject,
		projectTop + "/.pi/extra":                   cli.SkillRootFormProject,
		projectTop + "/.pi/extra-prompts":           cli.SkillRootFormProject,
		projectTop + "/.pi/npm/node_modules/ppk":    cli.SkillRootFormProject,
	}))

	prefixes := map[string][]string{}

	for _, root := range resolved.Roots {
		if root.KeyPrefixes != nil {
			prefixes[root.Path] = root.KeyPrefixes
		}
	}

	project := projectScope + ":"
	g.Expect(prefixes).To(Equal(map[string][]string{
		userSkillsRoot + "/synced/b1":            {"anthropic-skills:"},
		fakeHome + "/extra-skills":               {"pi-settings:"},
		fakeHome + "/extra-prompts":              {"pi-settings:pi-prompt:"},
		piAgentDir + "/npm/node_modules/pk":      {"pi-pkg:pk:", "pi-pkg:pk:pi-prompt:"},
		projectTop + "/.claude/skills":           {project},
		projectTop + "/.claude/commands":         {project + "cmd:"},
		projectTop + "/.pi/skills":               {project + "pi:"},
		projectTop + "/.agents/skills":           {project + "agents:"},
		projectTop + "/.pi/prompts":              {project + "pi-prompt:"},
		projectTop + "/.pi/extra":                {project + "pi-settings:"},
		projectTop + "/.pi/extra-prompts":        {project + "pi-settings:pi-prompt:"},
		projectTop + "/.pi/npm/node_modules/ppk": {project + "pi-pkg:ppk:", project + "pi-pkg:ppk:pi-prompt:"},
	}))
}

// unexported constants.
const (
	removalStateRead = iota
	removalStateFailed
	removalStateAbsent
	removalStateCount
)

// unexported constants.
const (
	pkGoneSource  = "~/.pi/agent/npm/node_modules/pk/skills/gone/SKILL.md"
	projectSkills = projectTop + "/.claude/skills"
	syncedA       = userSkillsRoot + "/synced/a"
	syncedB       = userSkillsRoot + "/synced/b"
)

// offerFixture builds a CompareSkillOffers input: resolved sources plus a
// fake vault of keyed skill notes.
type offerFixture struct {
	sources    cli.ResolvedSkillSources
	notes      []offerNote
	declined   map[string]string
	home       string
	noRemovals bool
}

func (fixture *offerFixture) candidate(candidate cli.SkillCandidate) {
	fixture.sources.Candidates = append(fixture.sources.Candidates, candidate)
}

func (fixture *offerFixture) compare(g Gomega) cli.SkillOfferComparison {
	comparison, err := fixture.compareRapid()
	g.Expect(err).NotTo(HaveOccurred())

	return comparison
}

func (fixture *offerFixture) compareRapid() (cli.SkillOfferComparison, error) {
	vault := newSkillregFixtureVault()
	names := make([]string, 0, len(fixture.notes))

	for _, note := range fixture.notes {
		// The basename depends only on the key, so permuting the notes never
		// renames one.
		name := fmt.Sprintf("1.2026-09-26.%s.md", cli.SkillKeySlug(note.key))
		vault.put(name, sourcedSkillNote(note))
		names = append(names, name)
	}

	return cli.CompareSkillOffers(cli.SkillOfferInput{
		Vault:      skillregFixtureVaultRoot,
		Names:      names,
		ReadFile:   vault.readFile,
		Declined:   fixture.declined,
		Home:       fixture.home,
		Sources:    fixture.sources,
		NoRemovals: fixture.noRemovals,
	})
}

// drawPluginState records plugin as scanned (read), unscanned or in a
// name conflict (failed), or uninstalled (absent).
func (fixture *offerFixture) drawPluginState(rt *rapid.T, plugin string, state int) {
	switch state {
	case removalStateRead:
		fixture.sources.Plugins = append(fixture.sources.Plugins, cli.ClaudePluginStatus{Name: plugin, Scanned: true})
	case removalStateFailed:
		if rapid.Bool().Draw(rt, "conflict:"+plugin) {
			fixture.sources.PluginConflicts = append(fixture.sources.PluginConflicts,
				cli.ClaudePluginConflict{Plugin: plugin, Marketplaces: []string{"a", "b"}})

			return
		}

		fixture.sources.Plugins = append(fixture.sources.Plugins, cli.ClaudePluginStatus{Name: plugin})
	}
}

// drawRemovalNote adds note index of a drawn key form and root state, and
// reports its key and whether the model says its root was read.
func (fixture *offerFixture) drawRemovalNote(rt *rapid.T, index int, fixedRead map[string]bool) (string, bool) {
	formNames := []string{
		"claude", "claude-cmd", "pi", "agents", "pi-prompt", "synced", "pi-settings", "pi-pkg", "project", "plugin",
	}
	formName := rapid.SampledFrom(formNames).Draw(rt, fmt.Sprintf("form%d", index))
	state := rapid.IntRange(0, removalStateCount-1).Draw(rt, fmt.Sprintf("state%d", index))
	name := fmt.Sprintf("n%d", index)

	if formName == "plugin" {
		plugin := fmt.Sprintf("plg%d", index)
		fixture.drawPluginState(rt, plugin, state)
		fixture.note(plugin+":"+name, "h"+name, "")

		return plugin + ":" + name, fixture.sources.PluginManifestsRead && state != removalStateFailed
	}

	if spec, fixed := removalFixedForms()[formName]; fixed {
		fixture.note(spec.prefix+name, "h"+name, "")

		return spec.prefix + name, fixedRead[formName]
	}

	spec := removalSourcedForms()[formName]
	outer := fmt.Sprintf("/r/%d", index)
	own := outer + "/own"
	inner := own + "/d"

	if state != removalStateAbsent {
		fixture.root(own, spec.form, state == removalStateRead, spec.prefix)
	}

	// A decoy: a READ root of the same form vouching for another scope or
	// segment, around the note's own root or inside it. It never grants
	// eligibility, and inside it hides the own root.
	decoy := rapid.SampledFrom([]string{"none", "outer", "inner"}).Draw(rt, fmt.Sprintf("decoy%d", index))
	if decoy != "none" && spec.otherPrefix != "" {
		fixture.root(map[string]string{"outer": outer, "inner": inner}[decoy], spec.form, true, spec.otherPrefix)
	}

	fixture.note(spec.prefix+name, "h"+name, inner+"/"+name+"/SKILL.md")

	return spec.prefix + name, state == removalStateRead && (decoy != "inner" || spec.otherPrefix == "")
}

func (fixture *offerFixture) hasEnabledCandidate(key, hash string) bool {
	for _, candidate := range fixture.sources.Candidates {
		if candidate.Key == key && !candidate.Disabled && cli.SkillContentHash(candidate.Content) == hash {
			return true
		}
	}

	return false
}

func (fixture *offerFixture) note(key, hash, source string) {
	fixture.notes = append(fixture.notes, offerNote{key: key, hash: hash, source: source})
}

// root records a read (or failed) root of form vouching for keyPrefixes.
func (fixture *offerFixture) root(path string, form cli.SkillRootForm, scanned bool, keyPrefixes ...string) {
	resolved := ""
	if scanned {
		resolved = path
	}

	fixture.sources.Roots = append(fixture.sources.Roots, cli.ScannedRoot{
		Path: path, Resolved: resolved, Scanned: scanned, Form: form, KeyPrefixes: keyPrefixes,
	})
}

// shuffled returns a copy with every input list permuted.
func (fixture *offerFixture) shuffled(rt *rapid.T) *offerFixture {
	out := *fixture
	out.sources.Candidates = rapid.Permutation(fixture.sources.Candidates).Draw(rt, "candidates")
	out.sources.Roots = rapid.Permutation(fixture.sources.Roots).Draw(rt, "roots")
	out.sources.Plugins = rapid.Permutation(fixture.sources.Plugins).Draw(rt, "plugins")
	out.notes = rapid.Permutation(fixture.notes).Draw(rt, "notes")

	return &out
}

// offerNote is one skill note in an offerFixture's vault.
type offerNote struct {
	key, hash, source string
}

// removalCase is one design D5 removal scenario over hand-built sources.
type removalCase struct {
	name        string
	setup       func(fixture *offerFixture)
	wantRemoved []string
}

// removalFormSpec is one key form's prefix and root for drawRemovalUniverse.
type removalFormSpec struct {
	prefix string
	root   string
	form   cli.SkillRootForm
	// otherPrefix is a key prefix of the same form but another scope or
	// segment, for decoy roots (empty: the form has only one prefix).
	otherPrefix string
}

// drawDedupeUniverse draws candidates over a small pool of keys, scopes,
// paths and contents (so same-path, same-key, alias and installed engram
// copy cases all occur), plus notes over the same keys and hashes. A path
// always has one content, as a real file does.
func drawDedupeUniverse(rt *rapid.T) *offerFixture {
	type slot struct {
		key, scope, path string
	}

	slots := []slot{
		{"claude:a", cli.SkillScopeClaudeUser, userSkillsRoot + "/a/SKILL.md"},
		{"claude:a", cli.SkillScopeClaudeUser, engramClaudeSkills + "/a/SKILL.md"},
		{"pi:a", cli.SkillScopePiUser, engramPiSkills + "/a/SKILL.md"},
		{"pi:a", cli.SkillScopePiUser, piUserRoot + "/x/a/SKILL.md"},
		{"pi:a", cli.SkillScopePiUser, piUserRoot + "/y/a/SKILL.md"},
		{"agents:a", cli.SkillScopeAgentsUser, agentsUserRoot + "/a/SKILL.md"},
		{"agents:b", cli.SkillScopeAgentsUser, piUserRoot + "/x/a/SKILL.md"},
		{"anthropic-skills:b", cli.SkillScopeSynced, userSkillsRoot + "/synced/1/b/SKILL.md"},
		{"anthropic-skills:b", cli.SkillScopeSynced, userSkillsRoot + "/synced/2/b/SKILL.md"},
		{"tools:b", "plugin:tools", "/cache/tools/skills/b/SKILL.md"},
		{projectScope + ":b", projectScope, projectTop + "/.claude/skills/b/SKILL.md"},
		{projectScope + ":b", projectScope, projectTop + "/sub/.claude/skills/b/SKILL.md"},
	}
	contents := []string{"one", "two", "three"}
	contentByPath := map[string]string{}

	fixture := newOfferFixture()

	count := rapid.IntRange(0, 10).Draw(rt, "candidateCount")
	for index := range count {
		picked := rapid.SampledFrom(slots).Draw(rt, fmt.Sprintf("slot%d", index))

		content, known := contentByPath[picked.path]
		if !known {
			content = rapid.SampledFrom(contents).Draw(rt, fmt.Sprintf("content%d", index))
			contentByPath[picked.path] = content
		}

		candidate := offerCand(picked.key, picked.scope, picked.path, content)
		candidate.Disabled = rapid.IntRange(0, 4).Draw(rt, fmt.Sprintf("disabled%d", index)) == 0
		fixture.candidate(candidate)
	}

	noteKeys := []string{"claude:a", "pi:a", "agents:a", "agents:b", "anthropic-skills:b", "tools:b", projectScope + ":b"}
	for _, key := range noteKeys {
		if !rapid.Bool().Draw(rt, "hasNote:"+key) {
			continue
		}

		hash := cli.SkillContentHash([]byte(rapid.SampledFrom(append(contents, "stale")).Draw(rt, "noteContent:"+key)))
		fixture.note(key, hash, "")
	}

	fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
	fixture.root(piUserRoot, cli.SkillRootFormPiUser, true)
	fixture.root(agentsUserRoot, cli.SkillRootFormAgentsUser, rapid.Bool().Draw(rt, "agentsRead"))
	fixture.sources.PluginManifestsRead = true
	fixture.sources.Plugins = []cli.ClaudePluginStatus{{Name: "tools", Scanned: true}, {Name: "other"}}

	return fixture
}

// drawRemovalUniverse draws notes of every key form, each with an intended
// state for its eligibility root (read, failed, absent) and, sometimes, a
// candidate (possibly disabled) holding its key. It returns the fixture and
// the sorted keys the model says must be offered for removal.
func drawRemovalUniverse(rt *rapid.T) (*offerFixture, []string) {
	fixture := newOfferFixture()
	fixture.home = fakeHome
	fixture.root("/r", "", true) // an untagged root holding every source: it must never grant eligibility
	fixture.sources.PluginManifestsRead = rapid.Bool().Draw(rt, "manifestsRead")

	fixedRead := map[string]bool{}

	for _, name := range slices.Sorted(maps.Keys(removalFixedForms())) {
		spec := removalFixedForms()[name]

		state := rapid.IntRange(0, removalStateCount-1).Draw(rt, "fixedState:"+name)
		if state != removalStateAbsent {
			fixture.root(spec.root, spec.form, state == removalStateRead)
		}

		fixedRead[name] = state == removalStateRead
	}

	var expected []string

	count := rapid.IntRange(0, 8).Draw(rt, "noteCount")
	for index := range count {
		key, eligible := fixture.drawRemovalNote(rt, index, fixedRead)

		if rapid.IntRange(0, 2).Draw(rt, fmt.Sprintf("present%d", index)) == 0 {
			present := offerCand(key, cli.SkillScopeClaudeUser, fmt.Sprintf("/elsewhere/%d/SKILL.md", index), key)
			present.Disabled = rapid.Bool().Draw(rt, fmt.Sprintf("presentDisabled%d", index))
			fixture.candidate(present)

			continue
		}

		if eligible {
			expected = append(expected, key)
		}
	}

	slices.Sort(expected)

	return fixture, expected
}

func newOfferFixture() *offerFixture {
	return &offerFixture{declined: map[string]string{}}
}

// offerCand is a keyed candidate whose Name is the key's last segment.
func offerCand(key, scope, sourcePath, content string) cli.SkillCandidate {
	name := key[strings.LastIndex(key, ":")+1:]

	return cli.SkillCandidate{
		Key:        key,
		Name:       name,
		ScopeID:    scope,
		SourcePath: sourcePath,
		Kind:       cli.SkillSourceKindSkill,
		Content:    []byte(content),
	}
}

// offerSummaries renders each offer as "<kind> <key> <source>".
func offerSummaries(comparison cli.SkillOfferComparison) []string {
	summaries := make([]string, 0, len(comparison.Offers))
	for _, offer := range comparison.Offers {
		summaries = append(summaries, fmt.Sprintf("%s %s %s", offer.Kind, offer.Key, offer.SourcePath))
	}

	return summaries
}

func offeredKeys(comparison cli.SkillOfferComparison) []string {
	keys := make([]string, 0, len(comparison.Offers))
	for _, offer := range comparison.Offers {
		if offer.Kind != cli.SkillOfferRemove {
			keys = append(keys, offer.Key)
		}
	}

	return keys
}

// removalFixedForms are the single-root user key forms and their roots.
func removalFixedForms() map[string]removalFormSpec {
	return map[string]removalFormSpec{
		"claude":     {prefix: "claude:", root: userSkillsRoot, form: cli.SkillRootFormClaudeUser},
		"claude-cmd": {prefix: "claude:cmd:", root: fakeHome + "/.claude/commands", form: cli.SkillRootFormClaudeCmd},
		"pi":         {prefix: "pi:", root: piUserRoot, form: cli.SkillRootFormPiUser},
		"agents":     {prefix: "agents:", root: agentsUserRoot, form: cli.SkillRootFormAgentsUser},
		"pi-prompt":  {prefix: "pi-prompt:", root: piPromptsRoot, form: cli.SkillRootFormPiPrompt},
	}
}

// removalSourcedForms are the source-rooted key forms.
func removalSourcedForms() map[string]removalFormSpec {
	return map[string]removalFormSpec{
		"synced": {prefix: "anthropic-skills:", form: cli.SkillRootFormSynced},
		"pi-settings": {
			prefix: "pi-settings:", form: cli.SkillRootFormPiSettings, otherPrefix: "pi-settings:pi-prompt:",
		},
		"pi-pkg":  {prefix: "pi-pkg:pk:", form: cli.SkillRootFormPiPkg, otherPrefix: "pi-pkg:other:"},
		"project": {prefix: projectScope + ":", form: cli.SkillRootFormProject, otherPrefix: projectScope + ":agents:"},
	}
}

func removedKeys(comparison cli.SkillOfferComparison) []string {
	keys := make([]string, 0, len(comparison.Offers))
	for _, offer := range comparison.Offers {
		if offer.Kind == cli.SkillOfferRemove {
			keys = append(keys, offer.Key)
		}
	}

	return keys
}

// resolveAndCompare resolves fsys's sources from cwd and compares them with
// a vault holding notes.
func resolveAndCompare(
	g Gomega, fsys cli.SkillSourceFS, cwd string, deps cli.SkillSourceDeps, notes []offerNote,
) cli.SkillOfferComparison {
	deps.FS = fsys

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, cwd, deps)
	g.Expect(err).NotTo(HaveOccurred())

	fixture := newOfferFixture()
	fixture.home = fakeHome
	fixture.sources = resolved
	fixture.notes = notes

	return fixture.compare(g)
}

// runRemovalCases runs each case against a fresh fixture (home fakeHome).
func runRemovalCases(t *testing.T, cases []removalCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fixture := newOfferFixture()
			fixture.home = fakeHome
			testCase.setup(fixture)

			comparison := fixture.compare(g)

			g.Expect(removedKeys(comparison)).To(ConsistOf(stringsOrEmpty(testCase.wantRemoved)))
		})
	}
}

// sourcedSkillNote renders a runbook skill note with skill_key and, when
// set, skill_source.
func sourcedSkillNote(note offerNote) string {
	source := ""
	if note.source != "" {
		source = "skill_source: \"" + note.source + "\"\n"
	}

	return "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_hash: \"" + note.hash +
		"\"\nskill_key: \"" + note.key + "\"\n" + source + "---\n\nbody\n"
}

func stringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
