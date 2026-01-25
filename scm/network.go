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
	port := a[0].String()
	handler := &HttpServer{a[1]}
	server := &http.Server{
		Addr:           fmt.Sprintf(":%v", port),
		Handler:        handler,
		ReadTimeout:    300 * time.Second,
		WriteTimeout:   300 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go server.ListenAndServe()
	// TODO: ListenAndServeTLS
	return NewBool(true)
}

// build this function into your file-local SCM environment to serve static files
func HTTPStaticGetter(wd string) func(...Scmer) Scmer {
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".html", "text/html")
	mime.AddExtensionType(".svg", "image/svg+xml")

	return func(a ...Scmer) Scmer {
		fs := http.FileServer(http.Dir(wd + "/" + a[0].String()))
		return NewFunc(func(a ...Scmer) Scmer { // req res
			resList := mustSliceNet("static response", a[1])
			reqList := mustSliceNet("static request", a[0])
			Apply(Apply(a[1], NewString("header")), NewString("Content-Type"), NewString(""))
			fs.ServeHTTP(resList[1].Any().(http.ResponseWriter), reqList[1].Any().(*http.Request))
			return NewNil()
		})
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
			query_scm = append(query_scm, NewString(k), NewString(v2))
		}
	}
	header_scm := make([]Scmer, 0)
	for k, v := range req.Header {
		for _, v2 := range v {
			header_scm = append(header_scm, NewString(k), NewString(v2))
		}
	}
	// read user/pass from basicauth
	user, pass, upok := req.BasicAuth()
	if !upok {
		// if no basicauth is provided, read from URL
		user = req.URL.User.Username()
		pass, upok = req.URL.User.Password()
	}
	req_scm := []Scmer{
		NewString("req"), NewAny(req),
		NewString("method"), NewString(req.Method),
		NewString("host"), NewString(req.Host),
		NewString("path"), NewString(req.URL.Path),
		NewString("query"), NewSlice(query_scm),
		NewString("header"), NewSlice(header_scm),
		NewString("username"), NewString(user),
		NewString("password"), NewString(pass),
		NewString("ip"), NewString(req.RemoteAddr),
		NewString("body"), NewFunc(func(a ...Scmer) Scmer {
			var b strings.Builder
			io.Copy(&b, req.Body)
			req.Body.Close()
			return NewString(b.String())
		}),
		NewString("bodyParts"), NewFunc(func(a ...Scmer) Scmer {
			result := []Scmer{}
			var b strings.Builder
			io.Copy(&b, req.Body)
			req.Body.Close()
			s := b.String()
			if s == "" {
				return NewSlice(result)
			}
			state := 0 // 0 -> await =, 1 = await &
			for i := 0; i < len(s); i++ {
				if state == 0 && s[i] == '=' {
					s2, err := url.QueryUnescape(s[:i])
					if err != nil {
						panic(err)
					}
					result = append(result, NewString(s2))
					state = 1
					s = s[i+1:]
					i = 0
				} else if state == 1 && s[i] == '&' {
					s2, err := url.QueryUnescape(s[:i])
					if err != nil {
						panic(err)
					}
					result = append(result, NewString(s2))
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
			result = append(result, NewString(s2))
			return NewSlice(result)
		}),
	}
	var res_lock sync.Mutex
	res_scm := []Scmer{
		NewString("res"), NewAny(res),
		NewString("header"), NewFunc(func(a ...Scmer) Scmer {
			res_lock.Lock()
			res.Header().Set(a[0].String(), a[1].String())
			res_lock.Unlock()
			return NewString("ok")
		}),
		NewString("status"), NewFunc(func(a ...Scmer) Scmer {
			// status after header!
			res_lock.Lock()
			status, _ := strconv.Atoi(a[0].String())
			res.WriteHeader(status)
			res_lock.Unlock()
			return NewString("ok")
		}),
		NewString("print"), NewFunc(func(a ...Scmer) Scmer {
			// naive output
			res_lock.Lock()
			for _, s := range a {
				io.WriteString(res, s.String())
			}
			res_lock.Unlock()
			return NewString("ok")
		}),
		NewString("println"), NewFunc(func(a ...Scmer) Scmer {
			// naive output
			res_lock.Lock()
			io.WriteString(res, a[0].String()+"\n")
			res_lock.Unlock()
			return NewString("ok")
		}),
		NewString("jsonl"), NewFunc(func(a ...Scmer) Scmer {
			// print json line (only assoc)
			res_lock.Lock()
			io.WriteString(res, "{")
			dict := mustSliceNet("jsonl", a[0])
			for i, v := range dict {
				if i%2 == 0 {
					// key
					bytes, _ := json.Marshal(v.String())
					res.Write(bytes)
					io.WriteString(res, ": ")
				} else {
					bytes, err := json.Marshal(scmerToGo(v))
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
			res_lock.Unlock()
			return NewString("ok")
		}),
		NewString("websocket"), NewFunc(func(a ...Scmer) Scmer {
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
				defer func() {
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
					if messageType == 1 {
						Apply(a[0], NewString(string(msg)))
					}
				}
			}()
			// return send callback
			var sendmutex sync.Mutex
			return NewFunc(func(a ...Scmer) Scmer {
				sendmutex.Lock()
				defer sendmutex.Unlock()
				err := ws.WriteMessage(int(a[0].Int()), []byte(a[1].String()))
				if err != nil {
					panic(err)
				}
				return NewString("ok")
			})
		}),
	}
	NewContext(req.Context(), func() {
		// catch panics and print out 500 Internal Server Error
		defer func() {
			if r := recover(); r != nil {
				PrintError("error in http handler: " + fmt.Sprint(r))
				res.Header().Set("Content-Type", "text/plain")
				res.WriteHeader(500)
				io.WriteString(res, "500 Internal Server Error: ")
				io.WriteString(res, fmt.Sprint(r))
			}
		}()
		Apply(s.callback, NewSlice(req_scm), NewSlice(res_scm))
	})
}

func scmerToGo(v Scmer) any {
	switch v.GetTag() {
	case tagNil:
		return nil
	case tagBool:
		return v.Bool()
	case tagInt:
		return v.Int()
	case tagFloat:
		return v.Float()
	case tagString, tagSymbol:
		return v.String()
	case tagSlice:
		list := v.Slice()
		out := make([]any, len(list))
		for i, item := range list {
			out[i] = scmerToGo(item)
		}
		return out
	case tagFastDict:
		fd := v.FastDict()
		if fd == nil {
			return []any{}
		}
		out := make([]any, len(fd.Pairs))
		for i, item := range fd.Pairs {
			out[i] = scmerToGo(item)
		}
		return out
	case tagAny:
		return v.Any()
	default:
		return v.String()
	}
}

func mustSliceNet(ctx string, v Scmer) []Scmer {
	if v.IsSlice() {
		return v.Slice()
	}
	if v.IsFastDict() {
		fd := v.FastDict()
		if fd == nil {
			return []Scmer{}
		}
		return fd.Pairs
	}
	panic(fmt.Sprintf("%s expects list", ctx))
}
