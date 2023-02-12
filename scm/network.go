/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "io"
import "fmt"
import "time"
import "strconv"
import "net/http"
import "runtime/debug"

// build this function into your SCM environment to offer http server capabilities
func HTTPServe(a ...Scmer) Scmer {
	// HTTP endpoint; params: (port, handler)
	port := String(a[0])
	handler := new(HttpServer)
	handler.callback = a[1] // lambda(req, res)
	server := &http.Server {
		Addr: fmt.Sprintf(":%v", port),
		Handler: handler,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go server.ListenAndServe()
	// TODO: ListenAndServeTLS
	return "ok"
}

// HTTP handler with a scheme script underneath
type HttpServer struct {
	callback Scmer
}

func (s *HttpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	query_scm := make([]Scmer, 0)
	for k, v := range req.URL.Query() {
		for _, v2 := range v {
			query_scm = append(query_scm, k, v2)
		}
	}
	header_scm := make([]Scmer, 0)
	for k, v := range req.Header {
		for _, v2 := range v {
			header_scm = append(header_scm, k, v2)
		}
	}
	// helper
	pwtostring := func (s string, isset bool) Scmer {
		if isset {
			return s
		} else {
			return nil
		}
	}
	req_scm := []Scmer {
		"method", req.Method,
		"host", req.Host,
		"path", req.URL.Path,
		"query", query_scm,
		"header", header_scm,
		"username", req.URL.User.Username(),
		"password", pwtostring(req.URL.User.Password()),
		"ip", req.RemoteAddr,
		// TODO: req.Body io.ReadCloser
	}
	res_scm := []Scmer {
		"header", func (a ...Scmer) Scmer {
			res.Header().Set(String(a[0]), String(a[1]))
			return "ok"
		},
		"status", func (a ...Scmer) Scmer {
			// status after header!
			status, _ := strconv.Atoi(String(a[0]))
			res.WriteHeader(status)
			return "ok"
		},
		"println", func (a ...Scmer) Scmer {
			// result-print-function (TODO: better interface with headers, JSON support etc.)
			// TODO: if a[0] is []Scmer -> build JSON object
			io.WriteString(res, String(a[0]) + "\n")
			return "ok"
		},
	}
	// catch panics and print out 500 Internal Server Error
	defer func () {
		if r := recover(); r != nil {
			fmt.Println("request failed:", req_scm, r, string(debug.Stack()))
			res.Header().Set("Content-Type", "text/plain")
			res.WriteHeader(500)
			io.WriteString(res, "500 Internal Server Error.")
		}
	}()
	Apply(s.callback, []Scmer{req_scm, res_scm})
	// TODO: req.Body io.ReadCloser
}
