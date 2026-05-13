package debuglog_test

import (
	"context"
	"os"
	"strings"
	"testing"

	g "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/debuglog"
)

func TestLog_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var nilLogger *debuglog.Logger

	// Nil-receiver methods must not panic.
	nilLogger.Log("stage", "msg=%s", "value")
	closer := nilLogger.Timed("stage", "arg=%s", "v")
	closer()
}

func TestLog_NoopWhenDisabled(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	logger, err := debuglog.New("", "")
	gomega.Expect(err).NotTo(g.HaveOccurred())

	// Must not panic on a disabled (no-op) logger.
	logger.Log("stage", "msg=%s", "value")

	// Must also no-op when nothing is in ctx.
	debuglog.Log(context.Background(), "stage", "msg=%s", "value")
}

func TestLog_WritesAndSyncs(t *testing.T) {
	t.Parallel()

	tmp, err := os.CreateTemp(t.TempDir(), "debuglog-*.log")
	if err != nil {
		t.Fatal(err)
	}

	path := tmp.Name()
	_ = tmp.Close()

	gomega := g.NewWithT(t)

	logger, newErr := debuglog.New(path, "test")
	gomega.Expect(newErr).NotTo(g.HaveOccurred())

	if newErr != nil {
		return
	}

	ctx := debuglog.WithLogger(context.Background(), logger)
	debuglog.Log(ctx, "some.stage", "key=%s val=%d", "hello", 42)

	contents, readErr := os.ReadFile(path)
	gomega.Expect(readErr).NotTo(g.HaveOccurred())

	if readErr != nil {
		return
	}

	line := string(contents)
	gomega.Expect(line).To(g.ContainSubstring("[test] some.stage: key=hello val=42"))
}

func TestTimed_LogsStartAndEnd(t *testing.T) {
	t.Parallel()

	tmp, err := os.CreateTemp(t.TempDir(), "debuglog-timed-*.log")
	if err != nil {
		t.Fatal(err)
	}

	path := tmp.Name()
	_ = tmp.Close()

	gomega := g.NewWithT(t)

	logger, newErr := debuglog.New(path, "timed")
	gomega.Expect(newErr).NotTo(g.HaveOccurred())

	if newErr != nil {
		return
	}

	ctx := debuglog.WithLogger(context.Background(), logger)
	closer := debuglog.Timed(ctx, "MyStage", "arg=%s", "val")
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

func TestTimed_NoLoggerInContext(t *testing.T) {
	t.Parallel()

	// Package-level Timed with no logger in ctx returns a no-op closer.
	closer := debuglog.Timed(context.Background(), "stage", "arg=%s", "v")
	closer()
}
