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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPResponseProxyForwardsStreamingRequestAndResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != http.MethodPost || string(body) != `{"question":"test"}` {
			t.Fatalf("unexpected proxied request: %s %q", r.Method, body)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected content type to be forwarded, got %q", got)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "{\"answer\":\"ok\"}\n")
	}))
	defer upstream.Close()

	server := &HttpServer{callback: NewFunc(func(a ...Scmer) Scmer {
		proxy := Apply(a[1], NewString("proxy"))
		return Apply(proxy, NewString(upstream.URL+"/api/chat"))
	})}
	req := httptest.NewRequest(http.MethodPost, "/ollama-chat", strings.NewReader(`{"question":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected upstream status %d, got %d", http.StatusCreated, res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != "application/x-ndjson" {
		t.Fatalf("expected upstream content type, got %q", got)
	}
	if got := res.Body.String(); got != "{\"answer\":\"ok\"}\n" {
		t.Fatalf("unexpected upstream body %q", got)
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
