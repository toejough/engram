//go:build targ

package dev

import (
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

func TestT4_IDPath_Append(t *testing.T) {
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
