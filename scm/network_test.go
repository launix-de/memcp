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
