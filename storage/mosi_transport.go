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

// mosiPeer publishes channel deliveries in both directions on one connection.
// Payload handlers and storage I/O never run under its registry/order mutex.
// Consumers observe disconnect through done; no write is retried here.
type mosiPeer struct {
	viewMu            sync.Mutex
	declaredViews     map[string]int // sender knowledge, published only after enqueue
	declaredViewBytes int
	receivedViewBytes int
	receivedViewSlots int
	objectViews       sync.Map      // URI -> immutable constructor bytes received on this stream
	receiveIO         *ioSimulation // shared node ingress budget; receiver-only instrumentation

	operationPrefix string            // immutable authenticated connection binding for query operations
	recoveryDone    chan struct{}     // replication hello installed; nil for roles without recovery
	objectEndpoint  string            // immutable outbound URL, initialized before reader start
	objectPeer      clusterObjectPeer // immutable authenticated identity/incarnation, independent of connection
	scanSession     *scanRPCSession
	scanService     *nodeScanService // immutable before an outbound scan is dispatched
	conn            *websocket.Conn
	writeMu         sync.Mutex
	outgoing        *mosiMailbox
	writeDone       chan struct{}
	mu              sync.Mutex
	channels        *sync.Map                // channel name -> registered payload handler
	packetOrder     map[string]chan struct{} // wire publication order per resource; payload and grant IDs may differ
	packetBytes     atomic.Int64             // bounded queued packet bytes, never shard admission state
	next            atomic.Uint64
	done            chan struct{}
	once            sync.Once
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
	Channel string
	Payload []byte
	encoded []byte // immutable local frame, never wire metadata
}

// The socket writer is the only consumer. Producers publish immutable frames
// through an MPSC queue without taking a mutex or waiting for network progress.
// Exhaustion is reported immediately; bulk producers admit work before encoding.
type mosiMailItem struct {
	next    atomic.Pointer[mosiMailItem]
	message mosiMessage
}
type mosiMailbox struct {
	head  *mosiMailItem // writer-owned dummy node
	tail  atomic.Pointer[mosiMailItem]
	bytes atomic.Int64
	count atomic.Int64
	ready chan struct{}
	space chan struct{}
}

func newMosiMailbox() *mosiMailbox {
	dummy := new(mosiMailItem)
	q := &mosiMailbox{head: dummy, ready: make(chan struct{}, 1), space: make(chan struct{}, 1)}
	q.tail.Store(dummy)
	return q
}
func (q *mosiMailbox) len() int { return int(q.count.Load()) }
func (q *mosiMailbox) push(m mosiMessage) error {
	n := int64(len(m.encoded))
	for {
		used := q.bytes.Load()
		if used+n > 32<<20 {
			return fmt.Errorf("node send window exhausted")
		}
		if q.bytes.CompareAndSwap(used, used+n) {
			break
		}
	}
	if q.count.Add(1) > 65536 {
		q.count.Add(-1)
		q.bytes.Add(-n)
		return fmt.Errorf("node send window exhausted")
	}
	item := &mosiMailItem{message: m}
	previous := q.tail.Swap(item)
	previous.next.Store(item)
	select {
	case q.ready <- struct{}{}:
	default:
	}
	return nil
}
func (q *mosiMailbox) next(ctx context.Context) (mosiMessage, error) {
	for {
		if item := q.head.next.Load(); item != nil {
			m := item.message
			item.message = mosiMessage{}
			q.head = item
			q.bytes.Add(-int64(len(m.encoded)))
			q.count.Add(-1)
			select {
			case <-q.ready:
			default:
			}
			if q.head.next.Load() != nil {
				select {
				case q.ready <- struct{}{}:
				default:
				}
			}
			select {
			case q.space <- struct{}{}:
			default:
			}
			return m, nil
		}
		select {
		case <-q.ready:
		case <-ctx.Done():
			return mosiMessage{}, ctx.Err()
		}
	}
}

// Bulk producer admission is explicitly separate from Send. It leaves the
// control/return-channel reserve available while a shard stream is produced.
func (q *mosiMailbox) admitBulk(done <-chan struct{}) error {
	for q.bytes.Load() > 8<<20 || q.count.Load() >= 8192 {
		select {
		case <-q.space:
		case <-done:
			return fmt.Errorf("node peer disconnected")
		}
	}
	// Pass available capacity to other admitted producers too. A coalesced
	// pop notification alone could strand them after the writer drains fully.
	select {
	case q.space <- struct{}{}:
	default:
	}
	return nil
}

func newMosiPeer(conn *websocket.Conn, initialize ...func(*mosiPeer)) *mosiPeer {
	p := &mosiPeer{conn: conn, channels: &sync.Map{}, done: make(chan struct{}), outgoing: newMosiMailbox(), writeDone: make(chan struct{})}
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
	case <-p.done:
		return fmt.Errorf(mosi.NetworkError)
	default:
	}
	if m.encoded == nil {
		var err error
		m.encoded, err = encodeMosiMessage(m)
		if err != nil {
			return err
		}
	}
	return p.outgoing.push(m)
}

func (p *mosiPeer) write() {
	defer close(p.writeDone)
	defer p.close()
	for {
		m, err := p.outgoing.next(p.ctx)
		if err != nil {
			return
		}
		if err = p.writeMessage(m); err != nil {
			return
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
		if !p.dispatchMessage(m, len(frame), readyAt) {
			return
		}
	}
}

// The transport knows only registered channels. Each handler owns decoding,
// ordering, replies and whether admission runs inline or on a worker.
func (p *mosiPeer) dispatchMessage(m mosiMessage, size int, readyAt time.Time) bool {
	if p.channels == nil {
		return true
	}
	handler, ok := p.channels.Load(m.Channel)
	if !ok {
		// A role may register an undelivered-payload disposer under the local
		// empty key. The transport still knows nothing about payload formats.
		handler, ok = p.channels.Load("")
		if !ok {
			return true
		}
	}
	return handler.(nodeChannel)(p, m.Payload, size, readyAt)
}

// Delay delivery outside the reader so propagation latency overlaps independent
// messages and CPU work. Existing worker ownership joins timers on cancellation.
func (p *mosiPeer) waitDispatch(readyAt time.Time) bool {
	if p.receiveIO == nil || p.receiveIO.latency == 0 {
		return true
	}
	return p.receiveIO.wait(p.ctx, readyAt)
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
