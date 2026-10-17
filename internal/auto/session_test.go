package auto_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4d1a_HeldText_TheBlocksDeliveredBesidePromptsAreWhatASessionHolds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		`{"type":"queue-operation","operation":"enqueue"}`,
		`{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}`,
		`{"type":"attachment","attachment":{"type":"skill_listing","content":"- dataviz: draws charts"}}`,
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
			`"content":["[engram memory] a delivered block","a second entry of the same hook"]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"1.2026-10-17.a-note.md"}]}}`,
		`not a record at all`,
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
			`"content":["a block of a later prompt"]}}`,
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("[engram memory] a delivered block"))
	g.Expect(held).To(ContainSubstring("a second entry of the same hook"))
	g.Expect(held).To(ContainSubstring("a block of a later prompt"))
	g.Expect(held).NotTo(ContainSubstring("Zorblax"))
	g.Expect(held).NotTo(ContainSubstring("dataviz"))
	g.Expect(held).NotTo(ContainSubstring("1.2026-10-17.a-note.md"))
}
