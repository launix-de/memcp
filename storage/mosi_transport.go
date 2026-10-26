/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "fmt"
import "sync"
import "time"
import "context"
import "net/http"
import "sync/atomic"
import "encoding/json"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/storage/mosi"

// Roles share one HTTP endpoint and codec, while each adapter owns its own
// authentication and authorization. Choosing a role in a header grants no rights.
var mosiHTTP struct {
	mu    sync.Mutex
	roles atomic.Pointer[map[string]http.Handler]
}

func serveMosiHTTP(w http.ResponseWriter, req *http.Request) {
	if roles := mosiHTTP.roles.Load(); roles != nil {
		if handler := (*roles)[req.Header.Get("X-MemCP-Role")]; handler != nil {
			handler.ServeHTTP(w, req)
			return
		}
	}
	http.Error(w, "node role is not enabled", http.StatusForbidden)
}

func registerMosiRole(role string, handler http.Handler) (func(), error) {
	mosiHTTP.mu.Lock()
	defer mosiHTTP.mu.Unlock()
	next := make(map[string]http.Handler)
	if old := mosiHTTP.roles.Load(); old != nil {
		for key, value := range *old {
			next[key] = value
		}
	}
	if role == "" || handler == nil || next[role] != nil {
		return nil, fmt.Errorf("invalid or duplicate node role")
	}
	next[role] = handler
	mosiHTTP.roles.Store(&next)
	var once sync.Once
	return func() {
		once.Do(func() {
			mosiHTTP.mu.Lock()
			defer mosiHTTP.mu.Unlock()
			next := make(map[string]http.Handler)
			for key, value := range *mosiHTTP.roles.Load() {
				if key != role {
					next[key] = value
				}
			}
			mosiHTTP.roles.Store(&next)
		})
	}, nil
}

// mosiPeer multiplexes requests in both directions over one peer connection.
// Neither RPC handlers nor storage I/O run under its mutex.
// A disconnected peer fails every outstanding call; no write is retried here.
type mosiPeer struct {
	scanService *nodeScanService // immutable before an outbound scan is dispatched
	conn        *websocket.Conn
	writeMu     sync.Mutex
	outgoing    chan mosiMessage
	writeDone   chan struct{}
	mu          sync.Mutex
	pending     map[uint64]chan mosiMessage
	next        atomic.Uint64
	done        chan struct{}
	once        sync.Once
	handle      func(context.Context, string, json.RawMessage) (any, error)
	packet      func(*mosiPacket) error
	ctx         context.Context
	cancel      context.CancelFunc
	readDone    chan struct{}
	workers     sync.WaitGroup
}

type mosiMessage struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Packet *mosiPacket     `json:"packet,omitempty"`
}

type mosiRemoteError string

func (e mosiRemoteError) Error() string { return string(e) }

func newMosiPeer(conn *websocket.Conn, handle func(context.Context, string, json.RawMessage) (any, error), packet func(*mosiPacket) error) *mosiPeer {
	p := &mosiPeer{conn: conn, pending: make(map[uint64]chan mosiMessage), done: make(chan struct{}), handle: handle, packet: packet, outgoing: make(chan mosiMessage, 8), writeDone: make(chan struct{})}
	p.readDone = make(chan struct{})
	p.ctx, p.cancel = context.WithCancel(context.Background())
	conn.SetReadLimit(mosiMaxMessageBytes)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(15 * time.Second)) })
	go p.read()
	go p.write()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-p.done:
				return
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
					p.close()
					return
				}
			}
		}
	}()
	return p
}

func (p *mosiPeer) close() {
	p.once.Do(func() { close(p.done); p.cancel(); p.conn.Close() })
}

func (p *mosiPeer) wait() { <-p.readDone; <-p.writeDone; p.workers.Wait() }

func (p *mosiPeer) send(m mosiMessage) error {
	select {
	case <-p.done:
		return fmt.Errorf(mosi.NetworkError)
	case p.outgoing <- m:
		return nil
	}
}

// The bounded queue owns each message after send returns. Producers never
// wait for a protocol reply here; the connection reader completes acquisitions.
func (p *mosiPeer) write() {
	defer close(p.writeDone)
	defer p.close()
	for {
		select {
		case <-p.done:
			return
		case m := <-p.outgoing:
			if err := p.writeMessage(m); err != nil {
				return
			}
		}
	}
}

func (p *mosiPeer) writeMessage(m mosiMessage) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	select {
	case <-p.done:
		return fmt.Errorf(mosi.NetworkError)
	default:
	}
	if err := p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	frame, err := encodeMosiMessage(m)
	if err != nil {
		return err
	}
	return p.conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (p *mosiPeer) call(ctx context.Context, method string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	id := p.next.Add(1)
	ch := make(chan mosiMessage, 1)
	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.pending, id); p.mu.Unlock() }()
	if err := p.send(mosiMessage{ID: id, Method: method, Data: data}); err != nil {
		p.close()
		return err
	}
	select {
	case <-p.done:
		return fmt.Errorf("replication connection closed")
	case <-ctx.Done():
		return ctx.Err()
	case reply := <-ch:
		if reply.Error != "" {
			return mosiRemoteError(reply.Error)
		}
		if output == nil {
			return nil
		}
		return json.Unmarshal(reply.Data, output)
	}
}

func (p *mosiPeer) read() {
	defer close(p.readDone)
	defer p.close()
	for {
		kind, frame, err := p.conn.ReadMessage()
		if err != nil || kind != websocket.BinaryMessage {
			return
		}
		m, err := decodeMosiMessage(frame)
		if err != nil {
			return
		}
		if m.Packet != nil {
			if p.packet == nil {
				return
			}
			if err := p.packet(m.Packet); err != nil {
				fmt.Println("replication packet rejected:", err)
				return
			}
			continue
		}
		if m.Method == "" {
			p.mu.Lock()
			ch := p.pending[m.ID]
			p.mu.Unlock()
			if ch != nil {
				select {
				case ch <- m:
				default:
				}
			}
			continue
		}
		p.workers.Add(1)
		go func(m mosiMessage) {
			defer p.workers.Done()
			reply := mosiMessage{ID: m.ID}
			func() {
				defer func() {
					if r := recover(); r != nil {
						reply.Error = fmt.Sprint(r)
					}
				}()
				value, err := p.handle(p.ctx, m.Method, m.Data)
				if err != nil {
					reply.Error = err.Error()
					return
				}
				reply.Data, err = json.Marshal(value)
				if err != nil {
					reply.Error = err.Error()
				}
			}()
			if m.ID == 0 {
				if reply.Error != "" {
					p.close()
				}
				return
			} // fire-and-forget messages complete via their registered channel
			if err := p.send(reply); err != nil {
				p.close()
			}
		}(m)
	}
}
