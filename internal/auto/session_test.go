package auto_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

func Test793d4d1a_HeldText_TheNoteLinesOfTheBlocksDeliveredBesidePromptsAreWhatASessionHolds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		`{"type":"queue-operation","operation":"enqueue"}`,
		`{"type":"user","message":{"role":"user","content":"When is the Zorblax staging database reset?"}}`,
		`{"type":"attachment","attachment":{"type":"skill_listing","content":"- dataviz: draws charts"}}`,
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
			`"content":["[engram memory] The lines below are notes.\n` +
			`- 1.2026-10-17.first-note.md (when A): see 3.2026-10-17.third-note.md\n` +
			`- 2.2026-10-17.second-note.md (when B): body",` +
			`"another hook's entry, naming 4.2026-10-17.fourth-note.md"]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"5.2026-10-17.in-the-reply.md"}]}}`,
		`not a record at all`,
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
			`"content":["[engram memory] The lines below are notes.\n- 6.2026-10-17.later-note.md (when C): body"]}}`,
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("1.2026-10-17.first-note.md"))
	g.Expect(held).To(ContainSubstring("2.2026-10-17.second-note.md"))
	g.Expect(held).To(ContainSubstring("6.2026-10-17.later-note.md"))
	g.Expect(held).NotTo(ContainSubstring("3.2026-10-17.third-note.md"), "a note the body of another names")
	g.Expect(held).NotTo(ContainSubstring("when A"), "a situation")
	g.Expect(held).NotTo(ContainSubstring("4.2026-10-17.fourth-note.md"), "an entry of another hook")
	g.Expect(held).NotTo(ContainSubstring("Zorblax"), "the prompt")
	g.Expect(held).NotTo(ContainSubstring("dataviz"), "another attachment")
	g.Expect(held).NotTo(ContainSubstring("5.2026-10-17.in-the-reply.md"), "the reply")
}

func Test793d4d1d_HeldText_WhatTheAgentsOwnLookupPrintedIsHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		bashCall("t1", "engram query --phrase 1.2026-10-17.in-the-command.md"),
		toolResult("t1", `"path: 1.2026-10-17.a-string-result.md\nkind: fact"`),
		bashCall("t2", "cd /vault && engram query --phrase staging | head -50"),
		toolResult("t2", `[{"type":"text","text":"path: 2.2026-10-17.a-text-block-result.md"}]`),
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
			`"text":"the reply names 3.2026-10-17.in-the-reply.md"}]}}`,
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("1.2026-10-17.a-string-result.md"))
	g.Expect(held).To(ContainSubstring("2.2026-10-17.a-text-block-result.md"))
	g.Expect(held).To(ContainSubstring("1.2026-10-17.in-the-command.md"), "the command's text counts")
	g.Expect(held).NotTo(ContainSubstring("in-the-reply"))
}

func Test793d4d1e_HeldText_AListingOfTheVaultIsNotHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		bashCall("t1", "ls /home/u/.local/share/engram/vault"),
		toolResult("t1", `"1.2026-10-17.zorblax-staging-reset.md\n2.2026-10-17.format-python-before-commit.md"`),
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Read",` +
			`"input":{"file_path":"/vault/3.2026-10-17.read-note.md"}}]}}`,
		toolResult("t2", `"Information learned: 3.2026-10-17.read-note.md says so"`),
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).NotTo(ContainSubstring("1.2026-10-17.zorblax-staging-reset.md"))
	g.Expect(held).NotTo(ContainSubstring("2.2026-10-17.format-python-before-commit.md"))
	g.Expect(held).NotTo(ContainSubstring("3.2026-10-17.read-note.md"), "the result of another tool")
}

func Test793d4d1f_HeldText_ANoteFetchedWithEngramShowIsHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		bashCall("t1", "engram show 1.2026-10-17.zorblax-staging-reset"),
		toolResult("t1", `"---\ntype: fact\n---\n\nThe Zorblax staging database is reset every Sunday at 03:40 UTC.\n"`),
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("1.2026-10-17.zorblax-staging-reset"), "the command names the note")
}

// bashCall is the record of the agent running a Bash command.
func bashCall(id, command string) string {
	return `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"` + id +
		`","name":"Bash","input":{"command":"` + command + `"}}]}}`
}

// toolResult is the record of the result of a tool; content is the JSON of the result's content.
func toolResult(id, content string) string {
	return `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"` + id +
		`","content":` + content + `}]}}`
}
