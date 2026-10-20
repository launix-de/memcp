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
import "sync/atomic"
import "encoding/json"
import "github.com/gorilla/websocket"

// replicationPeer multiplexes requests in both directions over the slave's
// outbound connection. Neither RPC handlers nor storage I/O run under its mutex.
// A disconnected peer fails every outstanding call; no write is retried here.
type replicationPeer struct {
	conn     *websocket.Conn
	writeMu  sync.Mutex
	mu       sync.Mutex
	pending  map[uint64]chan replicationMessage
	next     atomic.Uint64
	done     chan struct{}
	once     sync.Once
	handle   func(context.Context, string, json.RawMessage) (any, error)
	ctx      context.Context
	cancel   context.CancelFunc
	readDone chan struct{}
	workers  sync.WaitGroup
}

type replicationMessage struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type replicationRemoteError string

func (e replicationRemoteError) Error() string { return string(e) }

func newReplicationPeer(conn *websocket.Conn, handle func(context.Context, string, json.RawMessage) (any, error)) *replicationPeer {
	p := &replicationPeer{conn: conn, pending: make(map[uint64]chan replicationMessage), done: make(chan struct{}), handle: handle}
	p.readDone = make(chan struct{})
	p.ctx, p.cancel = context.WithCancel(context.Background())
	conn.SetReadLimit(1 << 30)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(15 * time.Second)) })
	go p.read()
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

func (p *replicationPeer) close() {
	p.once.Do(func() { close(p.done); p.cancel(); p.conn.Close() })
}

func (p *replicationPeer) wait() { <-p.readDone; p.workers.Wait() }

func (p *replicationPeer) send(m replicationMessage) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	select {
	case <-p.done:
		return fmt.Errorf(replicationNetworkError)
	default:
	}
	if err := p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	return p.conn.WriteJSON(m)
}

func (p *replicationPeer) call(ctx context.Context, method string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	id := p.next.Add(1)
	ch := make(chan replicationMessage, 1)
	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.pending, id); p.mu.Unlock() }()
	if err := p.send(replicationMessage{ID: id, Method: method, Data: data}); err != nil {
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
			return replicationRemoteError(reply.Error)
		}
		if output == nil {
			return nil
		}
		return json.Unmarshal(reply.Data, output)
	}
}

func (p *replicationPeer) read() {
	defer close(p.readDone)
	defer p.close()
	for {
		var m replicationMessage
		if err := p.conn.ReadJSON(&m); err != nil {
			return
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
		go func(m replicationMessage) {
			defer p.workers.Done()
			reply := replicationMessage{ID: m.ID}
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
			if err := p.send(reply); err != nil {
				p.close()
			}
		}(m)
	}
}
