package jsonlutil_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/jsonlutil"
)

func TestParseLines_ValidJSON(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	type item struct {
		Name string `json:"name"`
	}

	data := []byte("{\"name\":\"a\"}\n{\"name\":\"b\"}\n")
	result := jsonlutil.ParseLines[item](data)
	g.Expect(result).To(gomega.HaveLen(2))
	g.Expect(result[0].Name).To(gomega.Equal("a"))
	g.Expect(result[1].Name).To(gomega.Equal("b"))
}

func TestParseLines_SkipsEmptyAndMalformed(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	type item struct {
		Name string `json:"name"`
	}

	data := []byte("{\"name\":\"a\"}\n\nbad json\n{\"name\":\"b\"}\n")
	result := jsonlutil.ParseLines[item](data)
	g.Expect(result).To(gomega.HaveLen(2))
}

func TestParseLines_EmptyInput(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	type item struct{}

	result := jsonlutil.ParseLines[item](nil)
	g.Expect(result).To(gomega.BeEmpty())
}
