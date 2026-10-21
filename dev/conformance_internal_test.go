//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

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
			name: "prose and headings that name an ID, and the scenarios of other changes, are left out",
			spec: "### Mapping of 793a2a1a\nThe line Scenario: 793a2a1a in prose\nScenario: 792b28c1a1 Other\n",
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
