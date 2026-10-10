package cli_test

// Refresh and resituate convert a CRLF note to LF in the write they already
// make, like adopt and rename.

import (
	"testing"

	. "github.com/onsi/gomega"
)

// Test792b14c7a3_RunResituate_ConvertsCRLFNote: a CRLF fact or feedback note is
// resituated (not refused) and written as LF, keeping its other keys.
func Test792b14c7a3_RunResituate_ConvertsCRLFNote(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lfNote := resituateUnknownKeysNote(noteType, "luhmann_old: \"12\"\n")

			after, writes, err := runExchangeSurvivalResituate(t.Context(), crlf(lfNote))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(writes).To(Equal(1))
			g.Expect(after).NotTo(ContainSubstring("\r"))

			fields := frontmatterOf(after)
			g.Expect(fields["situation"]).To(Equal("resituated context"))
			g.Expect(fields["luhmann_old"]).To(Equal("12"))
		})
	}
}

// Test792b31c7a11_RefreshSkill_ConvertsCRLFNote: a CRLF skill note is refreshed (not
// refused) and written as LF, keeping its authored fields.
func Test792b31c7a11_RefreshSkill_ConvertsCRLFNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	after, err := runSkillAcceptMode(t.Context(), "refresh", crlf(curatePromotedNoteFixture()))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(after).NotTo(ContainSubstring("\r"))

	fields := frontmatterOf(after)
	g.Expect(fields["situation"]).To(Equal(frontmatterOf(curatePromotedNoteFixture())["situation"]))
	g.Expect(fields["pending"]).To(BeTrue())
}
