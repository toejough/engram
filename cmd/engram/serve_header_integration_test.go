//go:build integration

package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b35c3a4_ReadHeaderTimeout_IsTenSeconds pins the 10 seconds the requirement names, so the long-running
// scenario test above cannot pass on a bound that drifted together with its own tolerance.
func Test792b35c3a4_ReadHeaderTimeout_IsTenSeconds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(readHeaderTimeout).To(Equal(vhs1HeaderBound))
}

// Test792b35c3a4_RealListenAndServe_ASlowHeaderReadIsCutOff verifies the scenario "A slow header read is cut off":
// a client that keeps sending request headers slowly, never finishing them, is disconnected by the server once the
// wait exceeds 10 seconds. It serves through the production primitives (realNewServeMux, realRegisterRoute,
// realListenAndServe) on a loopback port and waits out the real clock, because the bound is an operating-system
// effect of net/http's connection handling; a prompt, complete request on the same server is the control that the
// closure comes from the slow header and not from the server refusing every connection.
func Test792b35c3a4_RealListenAndServe_ASlowHeaderReadIsCutOff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	addr := vhs1FreeLoopbackAddr(t)
	stopServer := vhs1StartServer(t, addr)

	defer stopServer()

	g.Expect(vhs1PromptStatus(t, addr)).To(Equal(http.StatusOK))

	conn := vhs1DialWithRetry(t, addr)
	if conn == nil {
		return
	}

	defer func() { _ = conn.Close() }()

	started := time.Now()

	_, writeErr := conn.Write([]byte(vhs1PartialRequest))
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	stopDrip := vhs1DripHeaderBytes(conn)

	defer stopDrip()

	readErr := conn.SetReadDeadline(started.Add(vhs1HeaderBound + vhs1SlackAfterBound))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	closeErr := vhs1ReadUntilClosed(conn)
	elapsed := time.Since(started)

	g.Expect(closeErr).To(Or(MatchError(io.EOF), MatchError(syscall.ECONNRESET), MatchError(syscall.EPIPE)))
	g.Expect(elapsed).To(BeNumerically(">=", vhs1HeaderBound-vhs1ClockTolerance))
	g.Expect(elapsed).To(BeNumerically("<", vhs1HeaderBound+vhs1SlackAfterBound))
}

// unexported constants.
const (
	// vhs1ClockTolerance absorbs the gap between the client taking its start time and the server accepting.
	vhs1ClockTolerance = time.Second
	// vhs1DialAttempts bounds how often the client retries until the server listens.
	vhs1DialAttempts = 100
	// vhs1DialRetryWait is the wait between dial attempts.
	vhs1DialRetryWait = 50 * time.Millisecond
	// vhs1DripInterval is how often the slow client sends one more byte of a header value.
	vhs1DripInterval = 500 * time.Millisecond
	// vhs1HeaderBound is the 10 seconds of 792b35c3 Request bodies.
	vhs1HeaderBound = 10 * time.Second
	// vhs1PartialRequest is a request whose headers never end.
	vhs1PartialRequest = "GET /vhs1-ping HTTP/1.1\r\nHost: localhost\r\nX-Slow: a"
	// vhs1PingPattern is the control route.
	vhs1PingPattern = "/vhs1-ping"
	// vhs1PromptTimeout bounds the control request.
	vhs1PromptTimeout = 5 * time.Second
	// vhs1SlackAfterBound is how long after the bound the server may take to close the connection.
	vhs1SlackAfterBound = 5 * time.Second
	// vhs1StopTimeout bounds how long the server may take to stop after cancellation.
	vhs1StopTimeout = 5 * time.Second
)

// vhs1OKHandler answers every request 200.
type vhs1OKHandler struct{}

func (vhs1OKHandler) Serve(_ context.Context, _ cli.ServeRequest) cli.ServeResponse {
	return cli.ServeResponse{Status: http.StatusOK}
}

// vhs1DialWithRetry connects to addr, retrying until the server listens.
func vhs1DialWithRetry(t *testing.T, addr string) net.Conn {
	t.Helper()

	g := NewWithT(t)

	var dialer net.Dialer

	var dialErr error

	for range vhs1DialAttempts {
		conn, err := dialer.DialContext(t.Context(), "tcp", addr)
		if err == nil {
			return conn
		}

		dialErr = err

		time.Sleep(vhs1DialRetryWait)
	}

	g.Expect(dialErr).NotTo(HaveOccurred())

	return nil
}

// vhs1DripHeaderBytes keeps adding one byte to the open header value until stopped, so the client is slow, not idle.
func vhs1DripHeaderBytes(conn net.Conn) func() {
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		ticker := time.NewTicker(vhs1DripInterval)

		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, err := conn.Write([]byte("a"))
				if err != nil {
					return
				}
			}
		}
	}()

	return func() {
		close(done)
		<-finished
	}
}

// vhs1FreeLoopbackAddr returns a loopback address that was free a moment ago.
func vhs1FreeLoopbackAddr(t *testing.T) string {
	t.Helper()

	g := NewWithT(t)

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil || listener == nil {
		return ""
	}

	addr := listener.Addr().String()

	g.Expect(listener.Close()).To(Succeed())

	return addr
}

// vhs1PromptStatus sends one complete request at once, retrying until the server listens, and returns its status.
func vhs1PromptStatus(t *testing.T, addr string) int {
	t.Helper()

	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(t.Context(), vhs1PromptTimeout)

	defer cancel()

	var status int

	var getErr error

	for range vhs1DialAttempts {
		status, getErr = vhs1TryGet(ctx, "http://"+addr+vhs1PingPattern)
		if getErr == nil {
			break
		}

		time.Sleep(vhs1DialRetryWait)
	}

	g.Expect(getErr).NotTo(HaveOccurred())

	return status
}

// vhs1ReadUntilClosed reads and discards until the connection fails, and returns that error.
func vhs1ReadUntilClosed(conn net.Conn) error {
	buf := make([]byte, 1024)

	for {
		_, err := conn.Read(buf)
		if err != nil {
			return err
		}
	}
}

// vhs1StartServer serves one ping route through the production primitives on addr; the returned func stops it
// and checks it stopped as the expected shutdown.
func vhs1StartServer(t *testing.T, addr string) func() {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	mux := realNewServeMux()
	realRegisterRoute(mux, http.MethodGet, vhs1PingPattern, vhs1OKHandler{})

	served := make(chan error, 1)

	go func() { served <- realListenAndServe(ctx, mux, addr) }()

	return func() {
		cancel()

		g := NewWithT(t)

		select {
		case err := <-served:
			g.Expect(errors.Is(err, http.ErrServerClosed)).To(BeTrue())
		case <-time.After(vhs1StopTimeout):
			g.Expect(false).To(BeTrue(), "the server did not stop after cancellation")
		}
	}
}

// vhs1TryGet issues one GET and returns its status, closing the response body.
func vhs1TryGet(ctx context.Context, url string) (int, error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return 0, reqErr
	}

	resp, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		return 0, doErr
	}

	status := resp.StatusCode

	return status, resp.Body.Close()
}
