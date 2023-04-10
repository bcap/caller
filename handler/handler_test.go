package handler

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ptype "github.com/bcap/caller/plan"
)

var plan1 = `
execution:
- call:
  http: GET {{addr}}/service1/listing 200 0 10240
  execution:
  - delay: 100ms to 200ms
  - parallel:
    concurrency: 2
    execution:
    - call:
      http: GET {{addr}}/service2/product?id=1 200 0 1024
      delay: 500ms
    - call:
      http: GET {{addr}}/service2/product?id=2 404 0 1024
      delay: 50ms
    - call:
      http: GET {{addr}}/service2/product?id=3 200 0 1024
      delay: 500ms
    - call:
      http: GET {{addr}}/service2/product?id=4 200 0 1024
      delay: 500ms
  - call:
    http: POST {{addr}}/service3/metrics 200 1024 10240
    delay: 100ms
  - delay: 100ms
`

func TestHandler(t *testing.T) {
	ctx, cancel, server, addr := launchServer(t)
	defer cancel()

	plan := preparePlan(t, plan1, addr)

	client := http.Client{}
	request, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr.AddrPort().String(), nil)
	require.NoError(t, err)
	err = WritePlanHeaders(request, plan, "")
	require.NoError(t, err)
	_, err = client.Do(request)
	require.NoError(t, err)

	// assert the access log
	accessLog := server.Handler.(*Handler).testAccessLog
	assertInLog(t, accessLog, "GET / 0 -> 200 0")
	assertInLog(t, accessLog, "GET /service1/listing 0 -> 200 10240")
	assertInLog(t, accessLog, "GET /service2/product?id=1 0 -> 200 1024")
	assertInLog(t, accessLog, "GET /service2/product?id=2 0 -> 404 1024")
	assertInLog(t, accessLog, "GET /service2/product?id=3 0 -> 200 1024")
	assertInLog(t, accessLog, "GET /service2/product?id=4 0 -> 200 1024")
	assertInLog(t, accessLog, "POST /service3/metrics 1024 -> 200 10240")
}

func assertInLog(t *testing.T, accessLog []string, msg string) {
	for _, entry := range accessLog {
		if strings.Contains(entry, msg) {
			return
		}
	}
	assert.Fail(t, "access log does not contain entry for %q", msg)
}

func preparePlan(t *testing.T, planStr string, addr *net.TCPAddr) ptype.Plan {
	planStr = strings.ReplaceAll(planStr, "{{addr}}", "http://"+addr.AddrPort().String())
	plan, err := ptype.FromYAML([]byte(planStr))
	require.NoError(t, err)
	return plan
}

func launchServer(t *testing.T) (context.Context, context.CancelFunc, *http.Server, *net.TCPAddr) {
	ctx, cancel := context.WithCancel(context.Background())
	server := http.Server{
		Handler:     &Handler{testCaptureAccessLog: true},
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", ":0")
	require.NoError(t, err)
	go func() {
		server.Serve(listener)
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return ctx, cancel, &server, listener.Addr().(*net.TCPAddr)
}