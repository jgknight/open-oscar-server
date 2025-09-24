package toc

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/mk6i/retro-aim-server/state"
	"github.com/mk6i/retro-aim-server/wire"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockOSCARProxy is a mock of the OSCARProxy interface.
type mockOSCARProxy struct {
	mock.Mock
}

// newMockOSCARProxy creates a new mock instance.
func newMockOSCARProxy(t mock.TestingT) *mockOSCARProxy {
	mock := &mockOSCARProxy{}
	mock.Mock.Test(t)
	return mock
}

// Signon provides a mock function with given fields: ctx, args, tocVersion
func (_m *mockOSCARProxy) Signon(ctx context.Context, args []byte, tocVersion state.TOCVersion) (*state.Session, []string) {
	ret := _m.Called(ctx, args, tocVersion)

	var r0 *state.Session
	if rf, ok := ret.Get(0).(func(context.Context, []byte, state.TOCVersion) *state.Session); ok {
		r0 = rf(ctx, args, tocVersion)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*state.Session)
		}
	}

	return r0, ret.Get(1).([]string)
}

// NewServeMux provides a mock function with given fields:
func (_m *mockOSCARProxy) NewServeMux() http.Handler {
	ret := _m.Called()

	var r0 http.Handler
	if rf, ok := ret.Get(0).(func() http.Handler); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(http.Handler)
		}
	}

	return r0
}

// RecvBOS provides a mock function with given fields: ctx, sessBOS, chatRegistry, msgCh
func (_m *mockOSCARProxy) RecvBOS(ctx context.Context, sessBOS *state.Session, chatRegistry *ChatRegistry, msgCh chan []string) error {
	ret := _m.Called(ctx, sessBOS, chatRegistry, msgCh)
	return ret.Error(0)
}

// RecvClientCmd provides a mock function with given fields: ctx, sessBOS, chatRegistry, payload, toCh, doAsync
func (_m *mockOSCARProxy) RecvClientCmd(ctx context.Context, sessBOS *state.Session, chatRegistry *ChatRegistry, payload []byte, toCh chan<- []string, doAsync func(f func() error)) []string {
	ret := _m.Called(ctx, sessBOS, chatRegistry, payload, toCh, doAsync)

	return ret.Get(0).([]string)
}

// ensure correct behavior during global context cancellation (server shutdown)
func TestServer_handleTOCRequest_serverShutdown(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer wg.Done()
		sv := Server{
			bosProxy: testOSCARProxy(t),
			logger:   slog.Default(),
		}

		serverReader, _ := io.Pipe()

		fc := wire.NewFlapClient(0, serverReader, nil)
		closeConn := func() {
			_ = serverReader.Close()
		}
		sess := newTestSession("me")
		err := sv.handleTOCRequest(ctx, closeConn, sess, NewChatRegistry(), fc)
		assert.True(t, errors.Is(err, errTOCProcessing) || errors.Is(err, errServerWrite))
	}()

	// cancel context, simulating server shutdown
	cancel()

	// wait for handleTOCRequest to return
	wg.Wait()
}

func TestServer_login(t *testing.T) {
	roastedPass := wire.RoastTOCPassword([]byte("thepass"))
	cmdWithArgs := []byte(`me "xx` + hex.EncodeToString(roastedPass) + `"`) // Removed leading space

	cases := []struct {
		name           string
		cmd            []byte
		wantTocVersion state.TOCVersion
		wantErr        error
	}{
		{
			name:           "toc_signon",
			cmd:            append([]byte("toc_signon"), cmdWithArgs...),
			wantTocVersion: state.SupportsTOC,
		},
		{
			name:           "toc2_signon",
			cmd:            append([]byte("toc2_signon"), cmdWithArgs...),
			wantTocVersion: state.SupportsTOC2,
		},
		{
			name:           "toc2_login",
			cmd:            append([]byte("toc2_login"), cmdWithArgs...),
			wantTocVersion: state.SupportsTOC2 | state.SupportsTOC2Enhanced,
		},
		{
			name:    "unexpected cmd",
			cmd:     []byte("bad_cmd"),
			wantErr: errors.New("expected one of toc_signon, toc2_signon, toc2_login"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			clientConn, serverConn := net.Pipe()
			var wg sync.WaitGroup
			wg.Add(1)
			flapClient := wire.NewFlapClient(0, serverConn, serverConn)

			// this goroutine simulates the client program
			go func() {
				defer wg.Done()
				clientFlap := wire.NewFlapClient(0, clientConn, clientConn)
				err := clientFlap.SendDataFrame(tc.cmd)
				assert.NoError(t, err)
				// The server will send back a response, we need to read it to unblock the server.
				for {
					_, err := clientFlap.ReceiveFLAP()
					if err != nil {
						break
					}
				}
			}()
			defer func() {
				serverConn.Close() // close the server-side of the pipe to unblock the client
				wg.Wait()          // wait for the client goroutine to finish
				clientConn.Close() // now it's safe to close the client side
			}()

			proxy := newMockOSCARProxy(t)

			server := &Server{
				bosProxy: proxy,
				logger:   slog.Default(),
			}

			if tc.wantErr != nil {
				_, err := server.login(context.Background(), flapClient)
				assert.Equal(t, tc.wantErr, err)
				return
			}

			// Mocks for successful login cases
			testSess := newTestSession("me")
			proxy.On("Signon", mock.Anything, mock.Anything, mock.Anything).
				Return(testSess, []string{"some-reply"})

			sess, err := server.login(context.Background(), flapClient)
			assert.NoError(t, err)
			if err != nil {
				return // If there's an error, no further assertions on sess
			}
			assert.NotNil(t, sess)
			if sess == nil {
				return // If sess is nil, no further assertions on sess
			}
			assert.Equal(t, tc.wantTocVersion, sess.TocVersion())
		})
	}
}

// ensure correct behavior when client TCP connection disconnects
func TestServer_handleTOCRequest_clientReadDisconnect(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	serverReader, _ := io.Pipe()

	go func() {
		defer wg.Done()
		closeConn := func() {
			_ = serverReader.Close()
		}
		sess := newTestSession("me")
		fc := wire.NewFlapClient(0, serverReader, nil)

		sv := Server{
			bosProxy: testOSCARProxy(t),
			logger:   slog.Default(),
		}
		err := sv.handleTOCRequest(context.Background(), closeConn, sess, NewChatRegistry(), fc)
		assert.ErrorIs(t, err, errClientReq)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	}()

	// simulate a client TCP disconnect
	_ = serverReader.Close()

	// wait for handleTOCRequest to return
	wg.Wait()
}

// ensure correct behavior when session gets closed by another login
func TestServer_handleTOCRequest_sessClose(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	sess := newTestSession("me")

	go func() {
		defer wg.Done()

		serverReader, _ := io.Pipe()
		fc := wire.NewFlapClient(0, serverReader, nil)

		closeConn := func() {
			_ = serverReader.Close()
		}

		sv := Server{
			bosProxy: testOSCARProxy(t),
			logger:   slog.Default(),
		}
		err := sv.handleTOCRequest(context.Background(), closeConn, sess, NewChatRegistry(), fc)
		assert.ErrorIs(t, err, errTOCProcessing)
		assert.ErrorIs(t, err, errDisconnect)
	}()

	// close the session, simulating another client login kicking this session
	sess.Close()

	// wait for handleTOCRequest to return
	wg.Wait()
}

// ensure correct behavior when writing server response fails
func TestServer_handleTOCRequest_replyFailure(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	go func() {
		defer wg.Done()
		closeConn := func() {
			_ = serverReader.Close()
		}
		sess := newTestSession("me")
		fc := wire.NewFlapClient(0, serverReader, serverWriter)

		sv := Server{
			bosProxy: testOSCARProxy(t),
			logger:   slog.Default(),
		}
		err := sv.handleTOCRequest(context.Background(), closeConn, sess, NewChatRegistry(), fc)
		assert.ErrorIs(t, err, errServerWrite)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	}()

	// simulate a failed TCP write
	_ = serverWriter.Close()

	// set up a TOC client
	fc := wire.NewFlapClient(0, clientReader, clientWriter)

	// send a TOC command
	err := fc.SendDataFrame([]byte(`toc_get_status`))
	assert.NoError(t, err)

	// wait for handleTOCRequest to return
	wg.Wait()
}

// ensure correct behavior when writing server response fails
func TestServer_handleTOCRequest_happyPath(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	go func() {
		defer wg.Done()
		closeConn := func() {
			_ = serverReader.Close()
		}
		fc := wire.NewFlapClient(0, serverReader, serverWriter)
		sv := Server{
			bosProxy: testOSCARProxy(t),
			logger:   slog.Default(),
		}
		err := sv.handleTOCRequest(context.Background(), closeConn, newTestSession("me"), NewChatRegistry(), fc)
		assert.ErrorIs(t, err, errClientReq)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	}()

	// set up a TOC client
	fc := wire.NewFlapClient(0, clientReader, clientWriter)

	// send a malformed TOC command to the server
	err := fc.SendDataFrame([]byte(`toc_get_status`))
	assert.NoError(t, err)

	// wait for the TOC response from the server
	frame, err := fc.ReceiveFLAP()
	assert.NoError(t, err)

	// expecting an error from TOC because the command is malformed. this
	// demonstrates that a command was processed by the TOC handler.
	assert.Contains(t, string(frame.Payload), "internal server error")

	// cleanly disconnect
	_ = serverReader.Close()

	// wait for handleTOCRequest to return
	wg.Wait()
}
