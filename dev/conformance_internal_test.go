//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test861b2b1a_Conformance_AScenarioOfAnyRootNeedsATest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	specText := "Scenario: 792a1a1a Old\nScenario: 793a2a1a Mid\nScenario: 855a2a1a New\nScenario: 861b2b1a Last\n" +
		"Scenario Outline: 792a5b1d Table\n"
	sources := []string{"package a_test\n\nfunc Test793a2a1a_Mid(t *testing.T) {}\n"}

	g.Expect(scenariosNeedingATest(scenarioIDs(specText), sources, nil)).
		To(Equal([]string{"792a1a1a", "855a2a1a", "861b2b1a", "792a5b1d"}))
}

func Test861b2b1b_Conformance_AnAllowlistedScenarioIsNotNamed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	ids := []string{"792a2d1b", "793a2a1a", "792b28c1a4", "855a2a1a"}
	sources := []string{"package a_test\n\nfunc Test793a2a1a_Mid(t *testing.T) {}\n"}

	g.Expect(scenariosNeedingATest(ids, sources, allowlistFixture())).To(Equal([]string{"855a2a1a"}))
}

func Test861b2b1c_Conformance_AnAllowlistEntryNamesItsReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		allowlist []allowlistEntry
		want      []string
	}{
		{name: "a decision row or an issue is a reason", allowlist: allowlistFixture(), want: nil},
		{
			name: "an empty reason and a reason naming neither are named",
			allowlist: []allowlistEntry{
				{id: "792a2d1b", reason: ""},
				{id: "792a2e1c", reason: "later"},
				{id: "792a2e1d", reason: "795a16 and 861d27"},
			},
			want: []string{"792a2d1b", "792a2e1c"},
		},
		{
			name: "an entry out of order or listed twice is named",
			allowlist: []allowlistEntry{
				{id: "792b3c4a4", reason: "861d27"},
				{id: "792a2d1b", reason: "861d27"},
				{id: "792a2d1b", reason: "861d27"},
			},
			want: []string{"792a2d1b", "792a2d1b"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(malformedAllowlistEntries(testCase.allowlist)).To(Equal(testCase.want))
		})
	}
}

func Test861b2b1d_Conformance_AStaleEntryIsNamed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ids     []string
		sources []string
		want    []string
	}{
		{
			name:    "an entry for a scenario with no test is current",
			ids:     []string{"792a2d1b", "792b28c1a4"},
			sources: nil,
			want:    nil,
		},
		{
			name:    "an entry for a scenario that has a test is named",
			ids:     []string{"792a2d1b", "792b28c1a4"},
			sources: []string{"package a_test\n\nfunc Test792a2d1b_Now(t *testing.T) {}\n"},
			want:    []string{"792a2d1b"},
		},
		{
			name:    "an entry for an ID no spec holds is named",
			ids:     []string{"792b28c1a4"},
			sources: nil,
			want:    []string{"792a2d1b"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(staleAllowlistEntries(testCase.ids, testCase.sources, allowlistFixture())).To(Equal(testCase.want))
		})
	}
}

func Test861b2b1e_Conformance_NoCountIsPinned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	specText := "Scenario: 792a1a1a One\nScenario: 793a2a1a Two\nScenario: 855a2a1a Three\n"
	sources := []string{
		"package a_test\n\nfunc Test792a1a1a_One(t *testing.T) {}\nfunc Test793a2a1a_Two(t *testing.T) {}\n" +
			"func Test855a2a1a_Three(t *testing.T) {}\nfunc Test861b2b1a_Four(t *testing.T) {}\n",
	}

	g.Expect(scenariosNeedingATest(scenarioIDs(specText), sources, nil)).To(BeEmpty())
	g.Expect(scenariosNeedingATest(scenarioIDs(specText+"Scenario: 861b2b1a Four\n"), sources, nil)).To(BeEmpty())
	g.Expect(scenariosNeedingATest(scenarioIDs("Scenario: 792a1a1a One\n"), sources, nil)).To(BeEmpty())
}

func TestScenarioIDs_ReadsTheScenariosOfBothChanges(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec string
		want []string
	}{
		{
			name: "the scenarios of a spec, in the order they appear",
			spec: "```gherkin\nScenario: 793a2a1a The next session has memory\n  Given a machine\n\n" +
				"Scenario: 793d4b1h A second update\n  Given a checkout\n```\n",
			want: []string{"793a2a1a", "793d4b1h"},
		},
		{
			name: "an ID that appears twice is listed once, where it first appears",
			spec: "Scenario: 793a2a1a One\nScenario: 793a2b1a Two\n### Mapping\nScenario: 793a2a1a One\n",
			want: []string{"793a2a1a", "793a2b1a"},
		},
		{
			name: "an indented scenario line is read",
			spec: "  Scenario: 793d4b1a Every file of a mod\n",
			want: []string{"793d4b1a"},
		},
		{
			name: "the scenarios of the product and technical specs of 855 are read beside those of 793",
			spec: "Scenario: 855a2a1a A glance\nScenario: 793a2a1a One\nScenario: 855d4b1m A cap\n",
			want: []string{"855a2a1a", "793a2a1a", "855d4b1m"},
		},
		{
			name: "an ID that appears again further on is counted once",
			spec: "Scenario: 793d2a1d Replaced\n### Modified\nScenario: 793d2a1d Replaced\nScenario: 855d2a1a New\n",
			want: []string{"793d2a1d", "855d2a1a"},
		},
		{
			name: "prose and headings that name an ID are left out",
			spec: "### Mapping of 793a2a1a\nThe line Scenario: 793a2a1a in prose\n",
			want: nil,
		},
		{
			name: "the scenarios of every root are read",
			spec: "Scenario: 792b28c1a1 Other\nScenario: 794b4a1a Third\nScenario: 861b2a1a Fourth\n",
			want: []string{"792b28c1a1", "794b4a1a", "861b2a1a"},
		},
		{
			name: "a scenario outline is read",
			spec: "Scenario Outline: 792a5b1d An unreadable named file fails\n",
			want: []string{"792a5b1d"},
		},
		{
			name: "a token that is not three digits, a letter and letters or digits is no ID",
			spec: "Scenario: 79a2a1a Short\nScenario: abc2a1a Letters\nScenario: 7922a1a Four digits\n" +
				"Scenario: 792 Bare\n",
			want: nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(scenarioIDs(testCase.spec)).To(Equal(testCase.want))
		})
	}
}

func TestScenariosWithoutTest_NamesThoseNoTestSourceDeclares(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ids     []string
		sources []string
		want    []string
	}{
		{
			name: "every scenario has a test",
			ids:  []string{"793a2a1a", "793d4b1h"},
			sources: []string{
				"package a_test\n\nfunc Test793a2a1a_Something(t *testing.T) {}\n",
				"package b_test\n\nfunc Test793d4b1h_Else(t *testing.T) {}\n",
			},
			want: nil,
		},
		{
			name:    "a scenario no test declares is named, in the order of the IDs",
			ids:     []string{"793a2a1a", "793d4b1h", "793d4b1g"},
			sources: []string{"package a_test\n\nfunc Test793d4b1h_Else(t *testing.T) {}\n"},
			want:    []string{"793a2a1a", "793d4b1g"},
		},
		{
			name: "a comment or a call that names a test is not a test",
			ids:  []string{"793a2a1a"},
			sources: []string{
				"package a_test\n\n// Test793a2a1a_Something covers it\nfunc helper() { Test793a2a1a_Something(t) }\n",
			},
			want: []string{"793a2a1a"},
		},
		{
			name:    "the test of a longer ID is not the test of a shorter one",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\nfunc Test793a2a1ab_Something(t *testing.T) {}\n"},
			want:    []string{"793a2a1a"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(scenariosWithoutTest(testCase.ids, testCase.sources)).To(Equal(testCase.want))
		})
	}
}

// TestScenariosWithoutTest_OnlyADeclaredFunctionIsATest: the text of a declaration is not a declaration. The check's
// own fixtures hold such text in string literals, and renaming the real tests left it green.
func TestScenariosWithoutTest_OnlyADeclaredFunctionIsATest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ids     []string
		sources []string
		want    []string
	}{
		{
			name:    "a declaration inside a string literal is not a test",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\nvar fixture = \"func Test793a2a1a_Something(t *testing.T) {}\\n\"\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name:    "a declaration inside a raw string literal is not a test",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\nvar fixture = `\nfunc Test793a2a1a_Something(t *testing.T) {}\n`\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name:    "a method named like a test is not a test",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\ntype suite struct{}\n\nfunc (suite) Test793a2a1a_Something() {}\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name:    "a function literal named like a test is not a test",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\nvar Test793a2a1a_Something = func(t *testing.T) {}\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name:    "a source that does not parse declares no test",
			ids:     []string{"793a2a1a"},
			sources: []string{"package a_test\n\nfunc Test793a2a1a_Something(t *testing.T) {\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name: "a declared test counts beside a fixture that names another",
			ids:  []string{"793a2a1a", "793d4b1h"},
			sources: []string{
				"package a_test\n\nvar fixture = \"func Test793d4b1h_Else(t *testing.T) {}\"\n\n" +
					"func Test793a2a1a_Something(t *testing.T) {}\n",
			},
			want: []string{"793d4b1h"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(scenariosWithoutTest(testCase.ids, testCase.sources)).To(Equal(testCase.want))
		})
	}
}

// TestScenariosWithoutTest_OnlyTheIDsGivenAreChecked: a scenario with no test is named only when its ID is among those
// given.
func TestScenariosWithoutTest_OnlyTheIDsGivenAreChecked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	specWithTests := "Scenario: 793a2a1a One\n"
	specWithoutTest := "Scenario: 855a2a1a A glance\nScenario: 855d4b1m A cap\n"
	sources := []string{
		"package a_test\n\nfunc Test793a2a1a_One(t *testing.T) {}\nfunc Test855a2a1a_Glance(t *testing.T) {}\n",
	}

	g.Expect(scenariosWithoutTest(scenarioIDs(specWithTests), sources)).To(BeEmpty(), "an ID not given is not checked")
	g.Expect(scenariosWithoutTest(scenarioIDs(specWithTests+specWithoutTest), sources)).To(Equal([]string{"855d4b1m"}))
}

// allowlistFixture is a sorted allowlist, each reason naming a decision row or an issue.
func allowlistFixture() []allowlistEntry {
	return []allowlistEntry{
		{id: "792a2d1b", reason: "861d27: removed by change 796"},
		{id: "792b28c1a4", reason: "#831: Run runs no go env"},
	}
}
