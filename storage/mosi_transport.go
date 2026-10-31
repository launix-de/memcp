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

import "io"
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
	receiveIO *ioSimulation // shared node ingress budget; receiver-only instrumentation

	operationPrefix string            // immutable authenticated connection binding for query operations
	recoveryDone    chan struct{}     // replication hello installed; nil for roles without recovery
	objectEndpoint  string            // immutable outbound URL, initialized before reader start
	objectPeer      clusterObjectPeer // immutable authenticated identity/incarnation, independent of connection
	scanService     *nodeScanService  // immutable before an outbound scan is dispatched
	conn            *websocket.Conn
	writeMu         sync.Mutex
	outgoing        chan mosiMessage
	writeDone       chan struct{}
	mu              sync.Mutex
	pending         map[uint64]chan mosiMessage
	packetOrder     map[string]chan struct{} // wire publication order per resource; payload and grant IDs may differ
	packetBytes     atomic.Int64             // bounded queued packet bytes, never shard admission state
	next            atomic.Uint64
	done            chan struct{}
	once            sync.Once
	handleValue     func(uint64, schemeValue) error
	handle          func(context.Context, string, json.RawMessage) (any, error)
	packet          func(*mosiPacket) error
	ctx             context.Context
	cancel          context.CancelFunc
	readDone        chan struct{}
	workers         sync.WaitGroup
	packets         sync.WaitGroup // ordered resource workers; may outlive a closed queue tail
}

func (p *mosiPeer) recovered() bool {
	if p == nil {
		return false
	}
	if p.recoveryDone == nil {
		return true
	}
	select {
	case <-p.recoveryDone:
		return true
	default:
		return false
	}
}

type mosiMessage struct {
	Slot    uint64          `json:"slot,omitempty"`
	Payload schemeValue     `json:"payload,omitempty"`
	encoded []byte          // immutable local frame; never part of wire metadata
	ID      uint64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Packet  *mosiPacket     `json:"packet,omitempty"`
}

type mosiRemoteError string

func (e mosiRemoteError) Error() string { return string(e) }

func newMosiPeer(conn *websocket.Conn, handle func(context.Context, string, json.RawMessage) (any, error), packet func(*mosiPacket) error, initialize ...func(*mosiPeer)) *mosiPeer {
	p := &mosiPeer{conn: conn, pending: make(map[uint64]chan mosiMessage), done: make(chan struct{}), handle: handle, packet: packet, outgoing: make(chan mosiMessage, 8), writeDone: make(chan struct{})}
	p.receiveIO = activeClusterReceiveIO()
	p.readDone = make(chan struct{})
	p.ctx, p.cancel = context.WithCancel(context.Background())
	conn.SetReadLimit(mosiMaxMessageBytes)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(15 * time.Second)) })
	// Publish immutable role/identity bindings before either network goroutine.
	for _, initialize := range initialize {
		initialize(p)
	}
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

func (p *mosiPeer) wait() {
	<-p.readDone
	<-p.writeDone
	p.packets.Wait()
	p.workers.Wait()
}

// Retirement can run inside an RPC worker on this peer. Join the reader and
// resource workers without waiting on that RPC itself. A cancelled queue tail
// can finish before an already executing predecessor, so packetOrder alone
// cannot join publication.
func (p *mosiPeer) waitPackets() {
	<-p.readDone
	p.packets.Wait()
}

func (p *mosiPeer) send(ctx context.Context, m mosiMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
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
	frame := m.encoded
	if frame == nil {
		var err error
		frame, err = encodeMosiMessage(m)
		if err != nil {
			return err
		}
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
	if err := p.send(ctx, mosiMessage{ID: id, Method: method, Data: data}); err != nil {
		if ctx.Err() == nil {
			p.close()
		}
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
	var previousStart time.Time
	var previousBytes int
	for {
		// Finish the prior message's byte budget before consuming the next header.
		// Decode/dispatch time already elapsed is credited; idle time grants no burst.
		if p.receiveIO != nil && !previousStart.IsZero() && !p.receiveIO.pace(p.ctx, previousBytes, previousStart) {
			return
		}
		kind, reader, err := p.conn.NextReader()
		if err != nil || kind != websocket.BinaryMessage {
			return
		}
		// Waiting for a new message is idle time, not transmission work.
		// Start crediting elapsed work only after its header has arrived.
		if p.receiveIO != nil {
			previousStart = time.Now()
		}
		frame, err := io.ReadAll(reader)
		if err != nil {
			return
		}
		previousBytes = len(frame)
		var readyAt time.Time
		if p.receiveIO != nil {
			p.receiveIO.stats.operations.Add(1)
			p.receiveIO.stats.bytes.Add(uint64(len(frame)))
			readyAt = time.Now().Add(p.receiveIO.latency)
		}
		m, err := decodeMosiMessage(frame)
		if err != nil {
			return
		}
		if m.Packet != nil {
			if p.packet == nil {
				return
			}
			if !p.dispatchPacket(m.Packet, len(frame), readyAt) {
				return
			}
			continue
		}
		if m.Slot == 0 && m.Method == "" && (p.receiveIO == nil || p.receiveIO.latency == 0) {
			p.deliverReply(m)
			continue
		}
		if p.packetBytes.Add(int64(len(frame))) > 32<<20 {
			p.packetBytes.Add(-int64(len(frame)))
			return
		}
		p.workers.Add(1)
		go func(m mosiMessage, size int, readyAt time.Time) {
			defer p.workers.Done()
			defer p.packetBytes.Add(-int64(size))
			if !p.waitDispatch(readyAt) {
				return
			}
			if m.Slot != 0 {
				defer func() {
					if recover() != nil {
						p.close()
					}
				}()
				if p.handleValue == nil || p.handleValue(m.Slot, m.Payload) != nil {
					p.close()
				}
				return
			}
			if m.Method == "" {
				p.deliverReply(m)
				return
			}
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
					fmt.Println("node message rejected:", m.Method, reply.Error)
					p.close()
				}
				return
			} // fire-and-forget messages complete via their registered channel
			if err := p.send(p.ctx, reply); err != nil {
				p.close()
			}
		}(m, len(frame), readyAt)
	}
}

// Delay delivery outside the reader so propagation latency overlaps independent
// messages and CPU work. Existing worker ownership joins timers on cancellation.
func (p *mosiPeer) waitDispatch(readyAt time.Time) bool {
	if p.receiveIO == nil || p.receiveIO.latency == 0 {
		return true
	}
	return p.receiveIO.wait(p.ctx, readyAt)
}

func (p *mosiPeer) deliverReply(m mosiMessage) {
	p.mu.Lock()
	ch := p.pending[m.ID]
	p.mu.Unlock()
	if ch != nil {
		select {
		case ch <- m:
		default:
		}
	}
}

// dispatchPacket never executes an object's storage work in the network reader.
// Different resources run independently. Packets of the SAME resource retain
// socket order, including a grant AFTER a transfer end whose ID differs. Merely
// starting a goroutine per packet could publish rights before data installation.
// The byte ceiling bounds private queues under a stalled backend without
// retaining an unbounded shard dump.
func (p *mosiPeer) dispatchPacket(packet *mosiPacket, size int, readyAt time.Time) bool {
	const maxQueuedPacketBytes = 32 << 20
	if p.packetBytes.Add(int64(size)) > maxQueuedPacketBytes {
		p.packetBytes.Add(-int64(size))
		return false
	}
	key := packet.ID
	if packet.Resource.valid() {
		key = packet.Resource.key()
	}
	done := make(chan struct{})
	p.mu.Lock()
	if p.packetOrder == nil {
		p.packetOrder = make(map[string]chan struct{})
	}
	previous := p.packetOrder[key]
	var namespaceReady <-chan struct{}
	if packet.Resource.Shard != "" {
		namespaceReady = p.packetOrder[packet.Resource.Database]
	}
	p.packetOrder[key] = done
	p.mu.Unlock()
	p.packets.Add(1)
	go func() {
		defer p.packets.Done()
		defer p.packetBytes.Add(-int64(size))
		defer func() {
			p.mu.Lock()
			if p.packetOrder[key] == done {
				delete(p.packetOrder, key)
			}
			close(done)
			p.mu.Unlock()
		}()
		if !p.waitDispatch(readyAt) {
			return
		}
		if previous != nil {
			select {
			case <-previous:
			case <-p.done:
				return
			}
		}
		// A preceding schema publication introduces this data object's identity.
		// Independent shard queues must not overtake that namespace barrier.
		if namespaceReady != nil {
			select {
			case <-namespaceReady:
			case <-p.done:
				return
			}
		}
		select {
		case <-p.done:
			return
		default:
		}
		if err := p.packet(packet); err != nil {
			fmt.Println("replication packet rejected:", err)
			p.close()
		}
	}()
	return true
}
