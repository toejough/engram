package main

import (
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// TestFetchHTTPClient_ConnectTimeoutAtMostThreeSeconds (design D6, M9): an
// unreachable parent costs at most 3s to connect, on top of the 30s total.
func TestFetchHTTPClient_ConnectTimeoutAtMostThreeSeconds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(fetchConnectTimeout).To(BeNumerically(">", 0))
	g.Expect(fetchConnectTimeout).To(BeNumerically("<=", 3*time.Second))
	g.Expect(fetchConnectDialer.Timeout).To(Equal(fetchConnectTimeout))
	g.Expect(fetchHTTPClient.Timeout).To(Equal(30 * time.Second))

	transport, isTransport := fetchHTTPClient.Transport.(*http.Transport)
	g.Expect(isTransport).To(BeTrue())

	if transport == nil {
		return
	}

	// Built from http.DefaultTransport's clone: the default TLS-handshake,
	// idle-connection and proxy settings are kept, only the dial changes.
	defaults, isDefaultTransport := http.DefaultTransport.(*http.Transport)
	g.Expect(isDefaultTransport).To(BeTrue())

	if defaults == nil {
		return
	}

	g.Expect(transport).NotTo(BeIdenticalTo(defaults))
	g.Expect(transport.DialContext).NotTo(BeNil())
	g.Expect(transport.Proxy).NotTo(BeNil())
	g.Expect(transport.TLSHandshakeTimeout).To(Equal(defaults.TLSHandshakeTimeout))
	g.Expect(transport.TLSHandshakeTimeout).To(BeNumerically(">", 0))
	g.Expect(transport.IdleConnTimeout).To(Equal(defaults.IdleConnTimeout))
	g.Expect(transport.IdleConnTimeout).To(BeNumerically(">", 0))
	g.Expect(transport.MaxIdleConns).To(Equal(defaults.MaxIdleConns))
}
