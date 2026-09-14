/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPRequestUsesOnlyExplicitServerBuiltPayload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != http.MethodPost || string(body) != `{"model":"server-choice","input":"checked"}` {
			t.Fatalf("unexpected request: %s %q", r.Method, body)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"output":"filtered"}`)
	}))
	defer upstream.Close()

	result := HTTPRequest(
		NewString(http.MethodPost),
		NewString(upstream.URL+"/model"),
		NewSlice([]Scmer{NewString("Content-Type"), NewString("application/json")}),
		NewString(`{"model":"server-choice","input":"checked"}`),
	)
	if got := Apply(result, NewString("status")).Int(); got != http.StatusCreated {
		t.Fatalf("unexpected status %d", got)
	}
	if got := Apply(result, NewString("body")).String(); got != `{"output":"filtered"}` {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestHTTPSQLBodyUpdatesProcesslistInfo(t *testing.T) {
	const query = "SELECT SLEEP(1)"
	observed := ""
	server := &HttpServer{callback: NewFunc(func(a ...Scmer) Scmer {
		bodyFn := Apply(a[0], NewString("body"))
		if body := Apply(bodyFn).String(); body != query {
			t.Fatalf("expected request body %q, got %q", query, body)
		}
		ss := Apply(a[0], NewString("__session_state")).Any().(*SessionState)
		observed = strPtr(&ss.Info)
		return NewNil()
	})}
	req := httptest.NewRequest("POST", "/sql/database", strings.NewReader(query))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if observed != query {
		t.Fatalf("expected processlist info %q, got %q", query, observed)
	}
}

func TestHTTPRequestCarriesSessionAndQueryIdentity(t *testing.T) {
	server := &HttpServer{callback: NewFunc(func(a ...Scmer) Scmer {
		request := a[0]
		session := Apply(request, NewString("__session"))
		ss, ok := Apply(request, NewString("__session_state")).Any().(*SessionState)
		if !ok {
			t.Fatal("expected typed session state")
		}
		querySeq := Apply(request, NewString("__query_seq")).Int()
		if querySeq == 0 {
			t.Fatal("request is missing explicit query state")
		}
		if session != ss.GetOrCreateScmSession() {
			t.Fatal("request and process-list Scheme sessions differ")
		}
		return NewNil()
	})}
	req := httptest.NewRequest("POST", "/sql/database", strings.NewReader("SELECT 1"))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)
}

func TestHTTPProxyHandlerRoutingAndForwarding(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.URL.EscapedPath() != "/base/api/a%2Fb" || r.URL.RawQuery != "fixed=1&q=two" || string(body) != "payload" {
			t.Errorf("unexpected upstream request: %s %s %q", r.Method, r.URL, body)
		}
		if r.Header.Get("X-Private") != "" || r.Header.Get("X-Forwarded-Host") != "browser.example" {
			t.Errorf("unexpected forwarding headers: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Connection", "X-Upstream-Private")
		w.Header().Set("X-Upstream-Private", "drop")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	proxy := HTTPProxy(NewString(upstream.URL + "/base?fixed=1"))
	if calls.Load() != 0 {
		t.Fatal("constructing the handler contacted the upstream")
	}
	// Use the ordinary Scheme request/response objects and route before invoking
	// the proxy. Other paths delegate to the existing fallback handler.
	fallback := NewFunc(func(a ...Scmer) Scmer {
		Apply(Apply(a[1], NewString("status")), NewInt(404))
		return Apply(Apply(a[1], NewString("println")), NewString("fallback"))
	})
	router := NewFunc(func(a ...Scmer) Scmer {
		if strings.HasPrefix(Apply(a[0], NewString("path")).String(), "/api/") {
			return Apply(proxy, a...)
		}
		return Apply(fallback, a...)
	})
	server := &HttpServer{callback: router}
	req := httptest.NewRequest("POST", "http://browser.example/api/a%2Fb?q=two", strings.NewReader("payload"))
	req.Header.Set("Connection", "X-Private")
	req.Header.Set("X-Private", "drop")
	req.Header.Set("X-Forwarded-Host", "spoofed")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != 201 || res.Body.String() != `{"ok":true}` || res.Header().Get("Content-Type") != "application/json" || len(res.Header().Values("Content-Type")) != 1 {
		t.Fatalf("unexpected proxy response: %d %v %s", res.Code, res.Header(), res.Body)
	}
	if res.Header().Get("X-Upstream-Private") != "" || res.Header().Get("Connection") != "" {
		t.Fatalf("hop-by-hop response headers leaked: %v", res.Header())
	}
	res = httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest("GET", "http://browser.example/other", nil))
	if res.Code != 404 || !strings.Contains(res.Body.String(), "fallback") || calls.Load() != 1 {
		t.Fatalf("fallback did not handle unmatched path: %d %s, upstream calls=%d", res.Code, res.Body, calls.Load())
	}
}

func TestHTTPProxyStreamsBeforeUpstreamFinishes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, "first")
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "last")
	}))
	defer upstream.Close()
	proxy := httptest.NewServer(&HttpServer{callback: HTTPProxy(NewString(upstream.URL))})
	defer proxy.Close()
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", proxy.URL, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var chunk [5]byte
	if _, err := io.ReadFull(res.Body, chunk[:]); err != nil || string(chunk[:]) != "first" {
		t.Fatalf("first chunk unavailable before completion: %q, %v", chunk, err)
	}
}

func TestHTTPProxyPropagatesClientCancellation(t *testing.T) {
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer upstream.Close()
	proxy := httptest.NewServer(&HttpServer{callback: HTTPProxy(NewString(upstream.URL))})
	defer proxy.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", proxy.URL, nil)
	done := make(chan error, 1)
	go func() {
		res, err := http.DefaultClient.Do(req)
		if res != nil {
			res.Body.Close()
		}
		done <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("upstream not reached")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream request was not cancelled")
	}
	if err := <-done; err == nil {
		t.Fatal("client request succeeded after cancellation")
	}
}

func TestHTTPProxyRejectsInvalidTargets(t *testing.T) {
	for _, target := range []string{"", "/relative", "ftp://example.com", "http://example.com/#fragment", ":"} {
		t.Run(target, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid target accepted")
				}
			}()
			HTTPProxy(NewString(target))
		})
	}
}

func TestHTTPProxyUnavailableUpstreamReturnsBadGateway(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	upstream.Close()
	server := &HttpServer{callback: HTTPProxy(NewString(upstream.URL))}
	res := httptest.NewRecorder()
	server.ServeHTTP(res, httptest.NewRequest("GET", "http://local/", nil))
	if res.Code != http.StatusBadGateway {
		t.Fatalf("unavailable upstream returned %d instead of 502", res.Code)
	}
}

func TestHTTPProxyInterruptedStreamDoesNotAppendErrorResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "prefix")
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	defer upstream.Close()
	proxy := httptest.NewServer(&HttpServer{callback: HTTPProxy(NewString(upstream.URL))})
	defer proxy.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err == nil || string(body) != "prefix" {
		t.Fatalf("interrupted upstream stream was altered: %q, %v", body, err)
	}
}
