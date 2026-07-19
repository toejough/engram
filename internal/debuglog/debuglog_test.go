package debuglog_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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

	// A nil writer means "logging disabled": New returns a nil *Logger whose
	// methods are all no-ops.
	logger := debuglog.New(nil, "", fixedNow)
	gomega.Expect(logger).To(g.BeNil())

	// Must not panic on a disabled (no-op) logger.
	logger.Log("stage", "msg=%s", "value")

	// Must also no-op when nothing is in ctx.
	debuglog.Log(context.Background(), "stage", "msg=%s", "value")
}

func TestLog_WritesTimestampedLine(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "test", fixedNow)

	ctx := debuglog.WithLogger(context.Background(), logger)
	debuglog.Log(ctx, "some.stage", "key=%s val=%d", "hello", 42)

	gomega.Expect(out.String()).To(g.Equal(
		"2026-07-19T12:00:00Z [test] some.stage: key=hello val=42\n"))
}

func TestTimed_LogsStartAndEndWithDuration(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "timed", steppingNow())

	ctx := debuglog.WithLogger(context.Background(), logger)
	closer := debuglog.Timed(ctx, "MyStage", "arg=%s", "val")
	closer()

	text := out.String()
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.start: arg=val"))
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.end: took=1s"))

	lines := strings.Split(strings.TrimSpace(text), "\n")
	gomega.Expect(lines).To(g.HaveLen(2))
}

func TestTimed_NoLoggerInContext(t *testing.T) {
	t.Parallel()

	// Package-level Timed with no logger in ctx returns a no-op closer.
	closer := debuglog.Timed(context.Background(), "stage", "arg=%s", "v")
	closer()
}

// fixedNow returns a constant instant so timestamp output is exact.
func fixedNow() time.Time {
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

// steppingNow returns a clock that advances one second per call, making
// Timed's took= duration deterministic. Call sequence inside Timed:
// start-line timestamp, start capture, took argument, end-line timestamp —
// so took = 1s exactly.
func steppingNow() func() time.Time {
	current := fixedNow()

	return func() time.Time {
		now := current
		current = current.Add(time.Second)

		return now
	}
}
