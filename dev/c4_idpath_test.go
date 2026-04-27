//go:build targ

package dev

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

func TestT1_IDPath_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		segments []string
		level    int // 1=S, 2=SN, 3=SNM, 4=SNMP
		ok       bool
	}{
		{"S1", []string{"S1"}, 1, true},
		{"S2-N3", []string{"S2", "N3"}, 2, true},
		{"S2-N3-M5", []string{"S2", "N3", "M5"}, 3, true},
		{"S2-N3-M5-P12", []string{"S2", "N3", "M5", "P12"}, 4, true},
		{"E27", nil, 0, false},             // legacy flat
		{"S2-M5", nil, 0, false},           // skipped N
		{"N1", nil, 0, false},              // missing S
		{"S2-N3-M5-P12-X1", nil, 0, false}, // too deep
		{"s2-n3", nil, 0, false},           // wrong case
		{"S2-N0", nil, 0, false},           // zero-numbered
		{"S2-N", nil, 0, false},            // letter without number
		{"S2-N3a", nil, 0, false},          // letter suffix
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			path, err := ParseIDPath(test.input)
			if !test.ok {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}
			g.Expect(path.Segments).To(Equal(test.segments))
			g.Expect(path.Level).To(Equal(test.level))
		})
	}
}

func TestT2_IDPath_IsAncestorOf(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent, _ := ParseIDPath("S2-N3")
	child, _ := ParseIDPath("S2-N3-M5")
	sibling, _ := ParseIDPath("S2-N4")
	g.Expect(parent.IsAncestorOf(child)).To(BeTrue())
	g.Expect(child.IsAncestorOf(parent)).To(BeFalse())
	g.Expect(sibling.IsAncestorOf(child)).To(BeFalse())
	g.Expect(parent.IsAncestorOf(parent)).To(BeFalse())
}

func TestT3_IDPath_String(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	path, err := ParseIDPath("S2-N3-M5")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(path.String()).To(Equal("S2-N3-M5"))
}

func TestT4_IDPath_IsAncestorOrEqual(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent, _ := ParseIDPath("S2-N3")
	child, _ := ParseIDPath("S2-N3-M5")
	sibling, _ := ParseIDPath("S2-N4")
	g.Expect(parent.IsAncestorOrEqual(child)).To(BeTrue())
	g.Expect(child.IsAncestorOrEqual(parent)).To(BeFalse())
	g.Expect(sibling.IsAncestorOrEqual(child)).To(BeFalse())
	g.Expect(parent.IsAncestorOrEqual(parent)).To(BeTrue()) // reflexive
}

func TestT5_IDPath_Append(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	parent, _ := ParseIDPath("S2-N3")
	child, err := parent.Append("M", 5)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(child.String()).To(Equal("S2-N3-M5"))

	// Wrong letter for level should fail
	_, err = parent.Append("P", 1) // L3 expects M, not P
	g.Expect(err).To(HaveOccurred())
}

func TestT6_LocalLetter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level  int
		letter string
		ok     bool
	}{
		{1, "S", true},
		{2, "N", true},
		{3, "M", true},
		{4, "P", true},
		{0, "", false},
		{5, "", false},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("level-%d", test.level), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			letter, err := LocalLetter(test.level)
			if !test.ok {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}
			g.Expect(letter).To(Equal(test.letter))
		})
	}
}

func TestT7_Anchor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(Anchor("S2-N3-M5", "Recall")).To(Equal("s2-n3-m5-recall"))
	g.Expect(Anchor("S1", "CLI Binary")).To(Equal("s1-cli-binary"))
	g.Expect(Anchor("S2-N3-M5-P12", "Token Resolver")).To(Equal("s2-n3-m5-p12-token-resolver"))
}

func TestT8_ValidateElementID(t *testing.T) {
	t.Parallel()

	level1Focus := IDPath{} // empty = L1 context, accepted IDs are S<n>
	level2Focus, _ := ParseIDPath("S2")
	level3Focus, _ := ParseIDPath("S2-N3")
	level4Focus, _ := ParseIDPath("S2-N3-M5")

	tests := []struct {
		name   string
		level  int
		focus  IDPath
		id     string
		wantOK bool
	}{
		// L1: empty focus, S<n> accepted
		{"l1-s1-ok", 1, level1Focus, "S1", true},
		{"l1-s9-ok", 1, level1Focus, "S9", true},
		{"l1-n1-bad", 1, level1Focus, "S2-N1", false},
		// L2: focus=S2, accept S<n> ancestors, S2 itself, S2-N<m>
		{"l2-focus-ok", 2, level2Focus, "S2", true},
		{"l2-ancestor-ok", 2, level2Focus, "S1", true},
		{"l2-local-ok", 2, level2Focus, "S2-N3", true},
		{"l2-wrong-parent-bad", 2, level2Focus, "S3-N1", false},
		{"l2-too-deep-bad", 2, level2Focus, "S2-N3-M1", false},
		// L3: focus=S2-N3, accept S<n>, S2-N<m>, S2-N3, S2-N3-M<k>
		{"l3-focus-ok", 3, level3Focus, "S2-N3", true},
		{"l3-l1-ancestor-ok", 3, level3Focus, "S2", true},
		{"l3-l2-peer-ok", 3, level3Focus, "S2-N4", true},
		{"l3-local-ok", 3, level3Focus, "S2-N3-M5", true},
		{"l3-wrong-parent-bad", 3, level3Focus, "S2-N4-M5", false},
		// L4: focus=S2-N3-M5, accept S<n>, S2-N<m>, S2-N3-M<k>, S2-N3-M5-P<j>
		{"l4-focus-ok", 4, level4Focus, "S2-N3-M5", true},
		{"l4-l1-ancestor-ok", 4, level4Focus, "S2", true},
		{"l4-l2-ancestor-ok", 4, level4Focus, "S2-N3", true},
		{"l4-l3-peer-ok", 4, level4Focus, "S2-N3-M6", true},
		{"l4-local-ok", 4, level4Focus, "S2-N3-M5-P1", true},
		{"l4-wrong-parent-bad", 4, level4Focus, "S2-N3-M6-P1", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			err := ValidateElementID(test.level, test.focus, test.id)
			if test.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestT9_ValidateDiagramNodeID(t *testing.T) {
	t.Parallel()

	focus, _ := ParseIDPath("S2-N3-M5")

	tests := []struct {
		name   string
		id     string
		wantOK bool
	}{
		{"focus-ok", "S2-N3-M5", true},
		{"l1-ancestor-ok", "S2", true},
		{"l1-other-system-ok", "S5", true}, // shallower than focus — carry-over
		{"l2-ancestor-ok", "S2-N3", true},
		{"l2-other-container-ok", "S2-N1", true}, // shallower than focus — carry-over
		{"sibling-ok", "S2-N3-M3", true},
		{"sibling-ok-2", "S2-N3-M9", true},
		{"descendant-bad", "S2-N3-M5-P1", false},
		{"different-parent-bad", "S2-N4-M5", false},
		{"different-system-bad", "S3-N1-M1", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			err := ValidateDiagramNodeID(focus, test.id)
			if test.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}
