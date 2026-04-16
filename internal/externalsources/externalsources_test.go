// Package externalsources_test verifies the public types of the externalsources package.
package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestExternalFileFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	file := externalsources.ExternalFile{
		Kind: externalsources.KindClaudeMd,
		Path: "/some/abs/path/CLAUDE.md",
	}

	g.Expect(file.Kind).To(Equal(externalsources.KindClaudeMd))
	g.Expect(file.Path).To(Equal("/some/abs/path/CLAUDE.md"))
}

func TestKindString(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.KindClaudeMd.String()).To(Equal("claude_md"))
	g.Expect(externalsources.KindRules.String()).To(Equal("rules"))
	g.Expect(externalsources.KindAutoMemory.String()).To(Equal("auto_memory"))
	g.Expect(externalsources.KindSkill.String()).To(Equal("skill"))
	g.Expect(externalsources.KindUnknown.String()).To(Equal("unknown"))
	g.Expect(externalsources.Kind(unrecognizedKindValue).String()).To(Equal("invalid"))
}

// unexported constants.
const (
	unrecognizedKindValue = 99
)
