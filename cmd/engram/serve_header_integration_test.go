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
// scenario test below cannot pass on a bound that drifted together with its own tolerance.
func Test792b35c3a4_ReadHeaderTimeout_IsTenSeconds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(readHeaderTimeout).To(Equal(vhsiHeaderBound))
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

	addr, server, status := vhsiServeAndPing(t)

	defer server.stop(t)

	g.Expect(status).To(Equal(http.StatusOK))

	conn := vhsiDialWithRetry(t, addr)
	if conn == nil {
		return
	}

	defer func() { _ = conn.Close() }()

	started := time.Now()

	_, writeErr := conn.Write([]byte(vhsiPartialRequest))
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	stopDrip := vhsiDripHeaderBytes(conn)

	defer stopDrip()

	readErr := conn.SetReadDeadline(started.Add(vhsiHeaderBound + vhsiSlackAfterBound))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	closeErr := vhsiReadUntilClosed(conn)
	elapsed := time.Since(started)

	g.Expect(closeErr).To(Or(MatchError(io.EOF), MatchError(syscall.ECONNRESET), MatchError(syscall.EPIPE)))
	g.Expect(elapsed).To(BeNumerically(">=", vhsiHeaderBound-vhsiClockTolerance))
	g.Expect(elapsed).To(BeNumerically("<", vhsiHeaderBound+vhsiSlackAfterBound))
}

// unexported constants.
const (
	// vhsiClockTolerance absorbs the gap between the client taking its start time and the server accepting.
	vhsiClockTolerance = time.Second
	// vhsiDialAttempts bounds how often the client retries until the server listens.
	vhsiDialAttempts = 100
	// vhsiDialRetryWait is the wait between dial attempts.
	vhsiDialRetryWait = 50 * time.Millisecond
	// vhsiDripInterval is how often the slow client sends one more byte of a header value.
	vhsiDripInterval = 500 * time.Millisecond
	// vhsiHeaderBound is the 10 seconds of 792b35c3 Request bodies.
	vhsiHeaderBound = 10 * time.Second
	// vhsiPartialRequest is a request whose headers never end.
	vhsiPartialRequest = "GET /vhsi-ping HTTP/1.1\r\nHost: localhost\r\nX-Slow: a"
	// vhsiPingPattern is the control route.
	vhsiPingPattern = "/vhsi-ping"
	// vhsiPromptTimeout bounds the control request.
	vhsiPromptTimeout = 5 * time.Second
	// vhsiServeAttempts bounds how often the server is started again on a new port that another process took.
	vhsiServeAttempts = 5
	// vhsiSlackAfterBound is how long after the bound the server may take to close the connection.
	vhsiSlackAfterBound = 5 * time.Second
	// vhsiStopTimeout bounds how long the server may take to stop after cancellation.
	vhsiStopTimeout = 5 * time.Second
)

// vhsiOKHandler answers every request 200.
type vhsiOKHandler struct{}

func (vhsiOKHandler) Serve(_ context.Context, _ cli.ServeRequest) cli.ServeResponse {
	return cli.ServeResponse{Status: http.StatusOK}
}

// vhsiServer is a test server started on a loopback address. failed is closed when it exited without being stopped,
// which is how a bind error shows when another process took the port between the probe and the bind.
type vhsiServer struct {
	cancel context.CancelFunc
	failed chan struct{}
	served chan error
}

// discard stops a server that never served, without checking how it ended.
func (server vhsiServer) discard() {
	server.cancel()
	<-server.served
}

// stop stops the server and checks it stopped as the expected shutdown.
func (server vhsiServer) stop(t *testing.T) {
	t.Helper()

	server.cancel()

	g := NewWithT(t)

	select {
	case err := <-server.served:
		g.Expect(errors.Is(err, http.ErrServerClosed)).To(BeTrue())
	case <-time.After(vhsiStopTimeout):
		g.Expect(false).To(BeTrue(), "the server did not stop after cancellation")
	}
}

// vhsiDialWithRetry connects to addr, retrying until the server listens.
func vhsiDialWithRetry(t *testing.T, addr string) net.Conn {
	t.Helper()

	g := NewWithT(t)

	var dialer net.Dialer

	var dialErr error

	for range vhsiDialAttempts {
		conn, err := dialer.DialContext(t.Context(), "tcp", addr)
		if err == nil {
			return conn
		}

		dialErr = err

		time.Sleep(vhsiDialRetryWait)
	}

	g.Expect(dialErr).NotTo(HaveOccurred())

	return nil
}

// vhsiDripHeaderBytes keeps adding one byte to the open header value until stopped, so the client is slow, not idle.
func vhsiDripHeaderBytes(conn net.Conn) func() {
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		ticker := time.NewTicker(vhsiDripInterval)

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

// vhsiFreeLoopbackAddr returns a loopback address that was free a moment ago.
func vhsiFreeLoopbackAddr(t *testing.T) string {
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

// vhsiPromptStatus sends one complete request at once, retrying until the server listens, and returns its status.
// It returns false when the server exited without being stopped, so the address is not its.
func vhsiPromptStatus(t *testing.T, addr string, failed <-chan struct{}) (int, bool) {
	t.Helper()

	g := NewWithT(t)

	ctx, cancel := context.WithTimeout(t.Context(), vhsiPromptTimeout)

	defer cancel()

	var status int

	var getErr error

	for range vhsiDialAttempts {
		status, getErr = vhsiTryGet(ctx, "http://"+addr+vhsiPingPattern)
		if getErr == nil {
			return status, true
		}

		select {
		case <-failed:
			return 0, false
		default:
		}

		time.Sleep(vhsiDialRetryWait)
	}

	select {
	case <-failed:
		return 0, false
	default:
	}

	g.Expect(getErr).NotTo(HaveOccurred())

	return status, true
}

// vhsiReadUntilClosed reads and discards until the connection fails, and returns that error.
func vhsiReadUntilClosed(conn net.Conn) error {
	buf := make([]byte, 1024)

	for {
		_, err := conn.Read(buf)
		if err != nil {
			return err
		}
	}
}

// vhsiServeAndPing starts the server on a free loopback port and sends a prompt request. On a loaded machine another
// process can take the port between the probe and the bind, and the server then exits at once, so it is started
// again on a new port. It returns the address, the running server and the status of the prompt request.
func vhsiServeAndPing(t *testing.T) (string, vhsiServer, int) {
	t.Helper()

	for range vhsiServeAttempts {
		addr := vhsiFreeLoopbackAddr(t)
		server := vhsiStartServer(t, addr)

		status, served := vhsiPromptStatus(t, addr, server.failed)
		if served {
			return addr, server, status
		}

		server.discard()
	}

	t.Fatalf("no server listened in %d attempts", vhsiServeAttempts)

	return "", vhsiServer{}, 0
}

// vhsiStartServer serves one ping route through the production primitives on addr.
func vhsiStartServer(t *testing.T, addr string) vhsiServer {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	mux := realNewServeMux()
	realRegisterRoute(mux, http.MethodGet, vhsiPingPattern, vhsiOKHandler{})

	server := vhsiServer{cancel: cancel, failed: make(chan struct{}), served: make(chan error, 1)}

	go func() {
		err := realListenAndServe(ctx, mux, addr)
		if ctx.Err() == nil {
			close(server.failed)
		}

		server.served <- err
	}()

	return server
}

// vhsiTryGet issues one GET and returns its status, closing the response body.
func vhsiTryGet(ctx context.Context, url string) (int, error) {
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
