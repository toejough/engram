//go:build probe

package embed_test

import (
	"testing"

	"github.com/knights-analytics/hugot"
)

// TestProbe_HugotLinks asserts the Hugot pure-Go backend compiles and
// links on this machine before any real spike work begins. Run with:
//
//	go test -tags=probe -run=TestProbe_HugotLinks ./internal/embed/...
func TestProbe_HugotLinks(t *testing.T) {
	t.Parallel()
	_ = hugot.NewGoSession
	t.Log("Hugot+GoMLX link OK")
}
