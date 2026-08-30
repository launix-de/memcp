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
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSQLBodyUpdatesProcesslistInfo(t *testing.T) {
	const query = "SELECT SLEEP(1)"
	observed := ""
	server := &HttpServer{callback: NewFunc(func(a ...Scmer) Scmer {
		bodyFn := Apply(a[0], NewString("body"))
		if body := Apply(bodyFn).String(); body != query {
			t.Fatalf("expected request body %q, got %q", query, body)
		}
		ss := GetCurrentSessionState()
		if ss == nil {
			t.Fatal("expected HTTP session state")
		}
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

func TestHTTPRequestCarriesExecutionContextInSingleSessionFrame(t *testing.T) {
	server := &HttpServer{callback: NewFunc(func(a ...Scmer) Scmer {
		request := a[0]
		session := Apply(request, NewString("__session"))
		executionContext, ok := Apply(request, NewString("__execution_context")).Any().(*QueryExecutionContext)
		if !ok {
			t.Fatal("expected typed query execution context")
		}
		if executionContext.SessionState != GetCurrentSessionState() {
			t.Fatal("request and goroutine-local session state differ")
		}
		if executionContext.QuerySeq != CurrentQuerySeq() {
			t.Fatal("request and goroutine-local query sequence differ")
		}
		if session != executionContext.SessionState.GetOrCreateScmSession() {
			t.Fatal("request and process-list Scheme sessions differ")
		}
		return NewNil()
	})}
	req := httptest.NewRequest("POST", "/sql/database", strings.NewReader("SELECT 1"))
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)
}
