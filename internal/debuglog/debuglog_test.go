package debuglog_test

import (
	"os"
	"strings"
	"testing"

	"engram/internal/debuglog"

	g "github.com/onsi/gomega"
)

//nolint:paralleltest // package-level state requires serial tests
func TestLog_NoopWhenDisabled(t *testing.T) {
	gomega := g.NewWithT(t)

	initErr := debuglog.Init("", "")
	gomega.Expect(initErr).NotTo(g.HaveOccurred())

	// Must not panic.
	debuglog.Log("stage", "msg=%s", "value")
}

// debuglog uses package-level state; tests cannot run in parallel.
//
//nolint:paralleltest // package-level state requires serial tests
func TestLog_WritesAndSyncs(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "debuglog-*.log")
	if err != nil {
		t.Fatal(err)
	}

	path := tmp.Name()
	_ = tmp.Close()

	gomega := g.NewWithT(t)

	initErr := debuglog.Init(path, "test")
	gomega.Expect(initErr).NotTo(g.HaveOccurred())

	if initErr != nil {
		return
	}

	debuglog.Log("some.stage", "key=%s val=%d", "hello", 42)

	contents, readErr := os.ReadFile(path)
	gomega.Expect(readErr).NotTo(g.HaveOccurred())

	if readErr != nil {
		return
	}

	line := string(contents)
	gomega.Expect(line).To(g.ContainSubstring("[test] some.stage: key=hello val=42"))
}

//nolint:paralleltest // package-level state requires serial tests
func TestTimed_LogsStartAndEnd(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "debuglog-timed-*.log")
	if err != nil {
		t.Fatal(err)
	}

	path := tmp.Name()
	_ = tmp.Close()

	gomega := g.NewWithT(t)

	initErr := debuglog.Init(path, "timed")
	gomega.Expect(initErr).NotTo(g.HaveOccurred())

	if initErr != nil {
		return
	}

	closer := debuglog.Timed("MyStage", "arg=%s", "val")
	closer()

	contents, readErr := os.ReadFile(path)
	gomega.Expect(readErr).NotTo(g.HaveOccurred())

	if readErr != nil {
		return
	}

	text := string(contents)
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.start: arg=val"))
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.end: took="))

	lines := strings.Split(strings.TrimSpace(text), "\n")
	gomega.Expect(lines).To(g.HaveLen(2))
}
