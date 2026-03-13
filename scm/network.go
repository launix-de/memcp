/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
import "net"
import "time"
import "mime"
import "sync"
import "context"
import "strings"
import "strconv"
import "net/url"
import "net/http"
import "sync/atomic"
import "encoding/json"
import "path/filepath"
import "github.com/gorilla/websocket"

var httpServersMu sync.Mutex
var httpServers []*http.Server

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
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt64(&ActiveHTTPConnections, 1)
				atomic.AddInt64(&TotalHTTPRequests, 1)
			case http.StateClosed, http.StateHijacked:
				atomic.AddInt64(&ActiveHTTPConnections, -1)
			}
		},
	}
	httpServersMu.Lock()
	httpServers = append(httpServers, server)
	httpServersMu.Unlock()
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
		fs := http.FileServer(http.Dir(filepath.Join(wd, a[0].String())))
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
					panic("websocket closed")
				}
				return NewString("ok")
			})
		}),
	}
	var ss *SessionState
	if sid := req.Header.Get("X-Session-Id"); sid != "" {
		// Persistent HTTP session: reuse or create a long-lived SessionState.
		if v, ok := httpStates.Load(sid); ok {
			ss = v.(*SessionState)
		} else {
			ss = RegisterSession(user, req.RemoteAddr, "")
			httpStates.Store(sid, ss)
			if httpSessionAddHook != nil {
				httpSessionAddHook(sid, ss)
			}
		}
	} else {
		ss = RegisterSession(user, req.RemoteAddr, "")
		defer UnregisterSession(ss.ID)
	}
	defer ss.ReleaseAllLocks()
	// Reset killed flag in case this session was killed in a previous request
	ss.ResetKilled()
	ss.SetCommand("Query", req.Method+" "+req.URL.Path)
	// Watch for HTTP client disconnect and propagate to session kill
	reqDone := make(chan struct{})
	defer close(reqDone)
	go func() {
		select {
		case <-req.Context().Done():
			ss.Kill()
		case <-reqDone:
		}
	}()
	SetValues(map[string]any{"sessionStatePtr": ss}, func() {
		contextFn := func() {
			// catch panics and print out 500 Internal Server Error
			defer func() {
				if r := recover(); r != nil {
					if fmt.Sprint(r) != "websocket closed" {
						PrintError("error in http handler: " + fmt.Sprint(r))
					}
					// try to write error response; silently ignore if connection was hijacked (e.g. websocket)
					func() {
						defer func() { recover() }()
						res.Header().Set("Content-Type", "text/plain")
						res.WriteHeader(500)
						io.WriteString(res, "500 Internal Server Error: ")
						io.WriteString(res, fmt.Sprint(r))
					}()
				}
			}()
			Apply(s.callback, NewSlice(req_scm), NewSlice(res_scm))
		}
		// Persistent HTTP sessions reuse the same Scheme session so that
		// @variables set in one request are visible in subsequent requests.
		if req.Header.Get("X-Session-Id") != "" {
			NewContextWithSession(req.Context(), ss.GetOrCreateScmSession(), contextFn)
		} else {
			NewContext(req.Context(), contextFn)
		}
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

// hasActiveMySQLQueries returns true if any MySQL session is currently executing a query.
func hasActiveMySQLQueries() bool {
	active := false
	mysqlStates.Range(func(_, v any) bool {
		if strPtr(&v.(*SessionState).Command) == "Query" {
			active = true
			return false
		}
		return true
	})
	return active
}

// ShutdownServers stops all servers, drains in-flight requests, kills remaining sessions,
// waits for all kills to propagate, then returns so the caller can proceed with cleanup.
//
// Sequence:
//  1. Stop accepting new connections (MySQL listeners closed, HTTP listeners closed via Shutdown).
//  2. Drain: wait up to drainSeconds for in-flight requests to finish on their own.
//  3. Kill: send Kill() to every registered session.
//  4. Wait: poll until all MySQL queries have exited their panic-recovery paths (max 5 s).
//  5. HTTP: wait for HTTP handlers to finish (they exit quickly after Kill).
func ShutdownServers(drainSeconds int) {
	if drainSeconds <= 0 {
		drainSeconds = 10
	}
	drainDeadline := time.Now().Add(time.Duration(drainSeconds) * time.Second)

	// Phase 1: stop accepting new connections.
	mysqlListenersMu.Lock()
	listeners := mysqlListeners
	mysqlListeners = nil
	mysqlListenersMu.Unlock()
	for _, l := range listeners {
		func() { defer func() { recover() }(); l.Close() }()
	}
	httpServersMu.Lock()
	servers := httpServers
	httpServers = nil
	httpServersMu.Unlock()
	// HTTP Shutdown runs concurrently: stops new accepts and drains handlers.
	httpDone := make(chan struct{})
	go func() {
		defer close(httpDone)
		ctx, cancel := context.WithDeadline(context.Background(), drainDeadline)
		defer cancel()
		for _, srv := range servers {
			srv.Shutdown(ctx)
		}
	}()

	// Phase 2: drain — wait for in-flight MySQL queries to finish voluntarily.
	for time.Now().Before(drainDeadline) {
		if !hasActiveMySQLQueries() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Phase 3: kill all remaining sessions.
	for _, ss := range Snapshot() {
		ss.Kill()
	}

	// Phase 4: wait for kills to propagate through panic-recovery paths (max 5 s).
	killDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(killDeadline) {
		if !hasActiveMySQLQueries() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Phase 5: wait for HTTP handlers to exit (Kill already caused them to panic out).
	<-httpDone
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
