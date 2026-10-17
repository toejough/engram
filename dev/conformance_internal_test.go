//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestScenarioIDs_ListsEachScenarioOnceInOrder(t *testing.T) {
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
			name: "a scenario restated in a mapping is listed once, where it first appears",
			spec: "Scenario: 793a2a1a One\nScenario: 793a2b1a Two\n### 793d5a Mapping\nScenario: 793a2a1a One\n",
			want: []string{"793a2a1a", "793a2b1a"},
		},
		{
			name: "an indented scenario line is read",
			spec: "  Scenario: 793d4b1a Every file of a mod\n",
			want: []string{"793d4b1a"},
		},
		{
			name: "prose and headings that name an ID, and the scenarios of other changes, are left out",
			spec: "### 793d5a Mapping of 793a2a1a\nThe line Scenario: 793a2a1a in prose\nScenario: 792b28c1a1 Other\n",
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
			sources: []string{"func Test793d4b1h_Else(t *testing.T) {}\n"},
			want:    []string{"793a2a1a", "793d4b1g"},
		},
		{
			name:    "a comment or a call that names a test is not a test",
			ids:     []string{"793a2a1a"},
			sources: []string{"// Test793a2a1a_Something covers it\nfunc helper() { Test793a2a1a_Something(t) }\n"},
			want:    []string{"793a2a1a"},
		},
		{
			name:    "the test of a longer ID is not the test of a shorter one",
			ids:     []string{"793a2a1a"},
			sources: []string{"func Test793a2a1ab_Something(t *testing.T) {}\n"},
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
