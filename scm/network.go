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
import "time"
import "net/http"
import "fmt"

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
	Apply(s.callback, []Scmer{
		req.URL.Path, // TODO: change to associative array with all possible request info
		func (a ...Scmer) Scmer {
			// result-print-function (TODO: better interface with headers, JSON support etc.)
			io.WriteString(res, String(a[0]))
			return "ok"
		},
	})
	/* TODO: put the following into a data structure
	io.WriteString(res, "Method = ")
	io.WriteString(res, req.Method)
	io.WriteString(res, "\n")
	io.WriteString(res, "Path = ")
	io.WriteString(res, req.URL.Path) // or: RawPath
	io.WriteString(res, "\n")
	io.WriteString(res, "Params = ")
	io.WriteString(res, fmt.Sprint(req.URL.Query())) // map[string][]string
	io.WriteString(res, "\n")
	io.WriteString(res, "Header = ")
	io.WriteString(res, fmt.Sprint(req.Header))
	io.WriteString(res, "\n")
	*/
	// TODO: req.Body io.ReadCloser
	// req.URL.User.Username()
	// req.URL.User.Password()
	// req.ContentLength == -1 or >= 0
	// req.Host -> multiple hostnames according to DNS system
	// req.RemoteAddr -> IP
}
