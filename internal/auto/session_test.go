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
		`{"type":"assistant","message":{"role":"assistant","content":[` +
			`{"type":"text","text":"5.2026-10-17.in-the-reply.md"}]}}`,
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

func Test855d4e1a_HeldText_TheChunkIDsOfThePassagesDeliveredAreHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"prompt.submit",` +
		`"content":["[engram memory] The lines below are notes and passages.\n` +
		`- /chunks/an-earlier-session.jsonl#c3 (passage): the freeze, see /chunks/another-session.jsonl#c9\n` +
		`- 1.2026-10-17.first-note.md (when A): body"]}}`

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("/chunks/an-earlier-session.jsonl#c3"), "the chunk ID of the passage line")
	g.Expect(held).To(ContainSubstring("1.2026-10-17.first-note.md"), "the file name of the note line")
	g.Expect(held).NotTo(ContainSubstring("the freeze"), "the text of the passage")
	g.Expect(held).NotTo(ContainSubstring("another-session"), "an ID the text of the passage names")
}

func Test855d4e1b_HeldText_APassageFetchedWithEngramShowChunkAndANoteTheAgentSavedAreHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		bashCall("t1", "engram show-chunk /chunks/an-earlier-session.jsonl#c3"),
		toolResult("t1", `"the user said the cutover is frozen"`),
		bashCall("t2", "engram learn fact --slug zorblax-freeze --situation s"),
		toolResult("t2", `"/vault/2.2026-10-17.zorblax-freeze.md"`),
		bashCall("t3", "engram amend 3.2026-10-17.a-note.md --situation s"),
		toolResult("t3", `"amended 3.2026-10-17.a-note.md"`),
		bashCall("t4", "git status"),
		toolResult("t4", `"4.2026-10-17.a-file-of-git.md"`),
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("engram show-chunk /chunks/an-earlier-session.jsonl#c3"), "the command")
	g.Expect(held).To(ContainSubstring("the cutover is frozen"), "the result of the command")
	g.Expect(held).To(ContainSubstring("engram learn fact --slug zorblax-freeze"), "the command that saved")
	g.Expect(held).To(ContainSubstring("2.2026-10-17.zorblax-freeze.md"), "the note the result names")
	g.Expect(held).To(ContainSubstring("3.2026-10-17.a-note.md"), "the note the amend names")
	g.Expect(held).NotTo(ContainSubstring("4.2026-10-17.a-file-of-git.md"), "the result of another command")
}

func Test855d4e1c_HeldText_ABlockDeliveredWithinATurnIsHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"Tag v1.4.0."}}`,
		bashCall("t1", "touch tag-v1.4.0"),
		toolResult("t1", `"done"`),
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"tool.call",` +
			`"hookEvent":"PostToolUse","content":["[engram memory] The lines below are notes.\n` +
			`- 1.2026-10-18.changelog-note.md (when A): body"]}}`,
		bashCall("t2", "make ci"),
		toolResult("t2", `"failed"`),
		`{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"classic.PostToolBatch",` +
			`"hookEvent":"PostToolBatch","content":["[engram memory] The lines below are notes and passages.\n` +
			`- 2.2026-10-18.make-ci-note.md (when B): body\n- /chunks/an-earlier-session.jsonl#c3 (passage): text"]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done."}]}}`,
	}, "\n")

	held := auto.HeldText([]byte(record))

	g.Expect(held).To(ContainSubstring("1.2026-10-18.changelog-note.md"), "the note of a block put at a tool call")
	g.Expect(held).To(ContainSubstring("2.2026-10-18.make-ci-note.md"), "the note of a block put after a batch")
	g.Expect(held).To(ContainSubstring("/chunks/an-earlier-session.jsonl#c3"), "the passage of the block of a batch")
	g.Expect(held).NotTo(ContainSubstring("body"), "the text of a note")
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

func Test855d2c1o_IsMemoryCommand_ACommandThatRunsOneOfTheMemoryCommandsIsOne(t *testing.T) {
	t.Parallel()

	for command, want := range map[string]bool{
		"engram query --text=x":              true,
		"cd /vault && engram query | head":   true,
		"engram show 1.2026-10-17.a-note.md": true,
		"engram show-chunk /c.jsonl#c3":      true,
		"engram recall glance":               true,
		"engram learn fact --slug s":         true,
		"engram amend 3.a-note.md":           true,
		"engram update":                      false,
		"git status":                         false,
		"":                                   false,
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			NewWithT(t).Expect(auto.IsMemoryCommand(command)).To(Equal(want))
		})
	}
}
