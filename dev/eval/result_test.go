//go:build targ

package eval_test

import (
	"os"
	"testing"

	"github.com/toejough/engram/dev/eval"
)

func TestParseResult_BadJSON_Errors(t *testing.T) {
	t.Parallel()

	if _, err := eval.ParseResult([]byte("not json")); err == nil {
		t.Fatal("expected error on bad json")
	}
}

func TestParseResult_ExtractsLayer1(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/result.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	rs, err := eval.ParseResult(raw)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}

	if rs.SessionID != "6c024b14-40b7-405b-bab4-f04153abe8c2" {
		t.Fatalf("session id: got %q", rs.SessionID)
	}

	m := rs.Layer1()
	if m.Turns != 3 || m.DurationMS != 6137 {
		t.Fatalf("turns/duration: got %+v", m)
	}
	if m.TotalTokens != 51 {
		t.Fatalf("total tokens: got %d", m.TotalTokens)
	}
	if m.CostUSD < 0.07 || m.CostUSD > 0.08 {
		t.Fatalf("cost: got %v", m.CostUSD)
	}
}
