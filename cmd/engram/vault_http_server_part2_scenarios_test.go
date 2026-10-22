package main

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b35c17a2_FetchResponseReadLimit_OneByteOverAcceptedMaximum (792b35c17 The bounded fetch client): the edge
// reads one byte more than the largest response a child accepts (16 MiB, the requirement's figure), which is at
// least the request cap, so an oversized response arrives one byte over and is detected, not truncated.
func Test792b35c17a2_FetchResponseReadLimit_OneByteOverAcceptedMaximum(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.FetchResponseReadLimit).To(Equal(vhs2AcceptedMaximumBytes + 1))
	g.Expect(vhs2AcceptedMaximumBytes).To(BeNumerically(">=", cli.MaxServeRequestBytes))

	// The edge itself reads exactly that many bytes of a response that never ends.
	resp := &http.Response{StatusCode: http.StatusOK, Body: &vhs2EndlessBody{}}

	fetched, err := readNonNilResponse(resp)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(fetched.Status).To(Equal(http.StatusOK))
	g.Expect(fetched.Body).To(HaveLen(vhs2AcceptedMaximumBytes + 1))
}

// unexported constants.
const (
	// vhs2AcceptedMaximumBytes is the largest response a child accepts: 16 MiB (792b35c17 The bounded fetch client).
	vhs2AcceptedMaximumBytes = 16 << 20
)

// vhs2EndlessBody is a response body that yields bytes forever.
type vhs2EndlessBody struct{}

func (*vhs2EndlessBody) Close() error { return nil }

func (*vhs2EndlessBody) Read(buf []byte) (int, error) {
	for index := range buf {
		buf[index] = 'x'
	}

	return len(buf), nil
}
