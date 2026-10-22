package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b7c6a2_IDAndDateFromNoteFilename_RefusesNoLeadingID: a filename whose first segment is not a Luhmann ID has
// no ID and no date, although it has the three dot-separated segments of a note filename. It is tested directly
// because the rename path only reaches the helper after termFromDefinitionSlug has confirmed the filename.
func Test792b7c6a2_IDAndDateFromNoteFilename_RefusesNoLeadingID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	gotID, gotDate, gotOK := cli.ExportIDAndDateFromNoteFilename("not-an-id.2026-07-10.vocab-x-definition.md")

	g.Expect(gotOK).To(BeFalse())
	g.Expect(gotID).To(BeEmpty())
	g.Expect(gotDate).To(BeEmpty())
}

// Test792b7c6a3_IDAndDateFromNoteFilename_RefusesTooFewSegments: a filename with fewer than three dot-separated
// segments is refused; the second case has a valid ID and date but no slug, so only the segment count refuses it
// (the first, 211.md, is refused earlier by the ID guard). It is tested directly because the rename path only reaches
// the helper after termFromDefinitionSlug has confirmed the filename.
func Test792b7c6a3_IDAndDateFromNoteFilename_RefusesTooFewSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
	}{
		{name: "id only", filename: "211.md"},
		{name: "id and date without a slug", filename: "211.2026-07-10.md"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			gotID, gotDate, gotOK := cli.ExportIDAndDateFromNoteFilename(tc.filename)

			g.Expect(gotOK).To(BeFalse())
			g.Expect(gotID).To(BeEmpty())
			g.Expect(gotDate).To(BeEmpty())
		})
	}
}
