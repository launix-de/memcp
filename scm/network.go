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
import "mime"
import "sync"
import "strings"
import "strconv"
import "net/url"
import "net/http"
import "encoding/json"
import "github.com/gorilla/websocket"

// build this function into your SCM environment to offer http server capabilities
func HTTPServe(a ...Scmer) Scmer {
	// HTTP endpoint; params: (port, handler)
	port := String(a[0])
	handler := &HttpServer{a[1]}
	server := &http.Server {
		Addr: fmt.Sprintf(":%v", port),
		Handler: handler,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go server.ListenAndServe()
	// TODO: ListenAndServeTLS
	return true
}

// build this function into your file-local SCM environment to serve static files
func HTTPStaticGetter(wd string) func (...Scmer) Scmer {
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".html", "text/html")
	return func (a ...Scmer) Scmer {
		fs := http.FileServer(http.Dir(wd + "/" + String(a[0])))
		return func (a ...Scmer) Scmer { // req res
			fs.ServeHTTP(a[1].([]Scmer)[1].(http.ResponseWriter), a[0].([]Scmer)[1].(*http.Request))
			return nil
		}
	}
}

// TODO: implement NewServeMux.Handle(route, http.StripPrefix(pfx, handler))

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
	// read user/pass from basicauth
	user, pass, upok := req.BasicAuth()
	if !upok {
		// if no basicauth is provided, read from URL
		user = req.URL.User.Username()
		pass, upok = req.URL.User.Password()
	}
	req_scm := []Scmer {
		"req", req, // must be first item to be found by us
		"method", req.Method,
		"host", req.Host,
		"path", req.URL.Path,
		"query", query_scm,
		"header", header_scm,
		"username", user,
		"password", pass,
		"ip", req.RemoteAddr,
		"body", func(a ...Scmer) Scmer {
			var b strings.Builder
			io.Copy(&b, req.Body)
			req.Body.Close()
			return b.String()
		},
		"bodyParts", func(a ...Scmer) Scmer {
			result := []Scmer{}
			var b strings.Builder
			io.Copy(&b, req.Body)
			req.Body.Close()
			s := b.String()
			if s == "" {
				return result
			}
			state := 0 // 0 -> await =, 1 = await &
			for i := 0; i < len(s); i++ {
				if state == 0 && s[i] == '=' {
					s2, err := url.QueryUnescape(s[:i])
					if err != nil {
						panic(err)
					}
					result = append(result, s2)
					state = 1
					s = s[i+1:]
					i = 0
				} else if state == 1 && s[i] == '&' {
					s2, err := url.QueryUnescape(s[:i])
					if err != nil {
						panic(err)
					}
					result = append(result, s2)
					state = 0
					s = s[i+1:]
					i = 0
				}
			}
			if state != 1 {
				panic("invalid post data")
			}
			s2, err := url.QueryUnescape(s)
			if err != nil {
				panic(err)
			}
			result = append(result, s2)
			return result
		},
	}
	var res_lock sync.Mutex
	res_scm := []Scmer {
		"res", res,
		"header", func (a ...Scmer) Scmer {
			res_lock.Lock()
			res.Header().Set(String(a[0]), String(a[1]))
			res_lock.Unlock();
			return "ok"
		},
		"status", func (a ...Scmer) Scmer {
			// status after header!
			res_lock.Lock()
			status, _ := strconv.Atoi(String(a[0]))
			res.WriteHeader(status)
			res_lock.Unlock();
			return "ok"
		},
		"print", func (a ...Scmer) Scmer {
			// naive output
			res_lock.Lock()
			for _, s := range a {
				io.WriteString(res, String(s))
			}
			res_lock.Unlock();
			return "ok"
		},
		"println", func (a ...Scmer) Scmer {
			// naive output
			res_lock.Lock()
			io.WriteString(res, String(a[0]) + "\n")
			res_lock.Unlock();
			return "ok"
		},
		"jsonl", func (a ...Scmer) Scmer {
			// print json line (only assoc)
			res_lock.Lock()
			io.WriteString(res, "{")
			dict := a[0].([]Scmer)
			for i, v := range dict {
				if i % 2 == 0 {
					// key
					bytes, _ := json.Marshal(String(v))
					res.Write(bytes)
					io.WriteString(res, ": ")
				} else {
					bytes, err := json.Marshal(v)
					if err != nil {
						panic(err)
					}
					res.Write(bytes)
					if i < len(dict)-1 {
						io.WriteString(res, ", ")
					}
				}
			}
			io.WriteString(res, "}\n")
			res_lock.Unlock();
			return "ok"
		},
		"websocket", func (a ...Scmer) Scmer {
			// upgrade to a websocket, params: onMessage, onClose
			var upgrader = websocket.Upgrader{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
			}
			upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			ws, err := upgrader.Upgrade(res, req, nil)
			if err != nil {
				// TODO: better error handling
				panic(err)
			}
			go func() {
				defer func () {
					if r := recover(); r != nil {
						PrintError("error in websocket receive: " + fmt.Sprint(r))
					}
				}()
				for {
					// websocket read loop
					messageType, msg, err := ws.ReadMessage()
					if err != nil {
						if _, ok := err.(*websocket.CloseError); ok {
							// closed connection
							if len(a) > 1 {
								Apply(a[1]) // close callback
							}
							return // exit endless loop
						} else {
							// TODO: better error handling
							panic(err)
						}
					}
					// TODO: messageType 1 = text, 2 = binary?
					if messageType == 1 {
						Apply(a[0], string(msg)) // 1st parameter is callback to receive message
					}
				}
			}()
			// return send callback
			var sendmutex sync.Mutex
			return func(a ...Scmer) Scmer {
				sendmutex.Lock()
				defer sendmutex.Unlock()
				err := ws.WriteMessage(ToInt(a[0]), []byte(String(a[1])))
				if err != nil {
					panic(err)
				}
				return "ok"
			}
		},
	}
	NewContext(req.Context(), func() {
		// catch panics and print out 500 Internal Server Error
		defer func () {
			if r := recover(); r != nil {
				PrintError("error in http handler: " + fmt.Sprint(r))
				res.Header().Set("Content-Type", "text/plain")
				res.WriteHeader(500)
				io.WriteString(res, "500 Internal Server Error: ")
				io.WriteString(res, fmt.Sprint(r))
			}
		}()
		Apply(s.callback, req_scm, res_scm)
	})
}
