/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "time"
import "weak"
import "context"
import "strconv"
import "runtime"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

// A channel receives opaque bytes. Its adapter owns decoding and scheduling;
// the transport has no request, response, Scheme or MOSI message types.
// channelDelivery contains peer-local scheduling information, never wire fields.
type channelDelivery struct {
	context.Context
	peer     *mosiPeer
	size     int
	readyAt  time.Time
	accepted bool
}

// channelHandler adapts scheduling to the sole raw receive interface.
func channelHandler(receive func(*mosiPeer, []byte, int, time.Time) bool) scm.ClusterRequestHandler {
	return scm.ClusterRequestHandlerFunc(func(ctx context.Context, payload []byte) {
		delivery := ctx.(*channelDelivery)
		delivery.accepted = receive(delivery.peer, payload, delivery.size, delivery.readyAt)
	})
}

func (p *mosiPeer) ctxOrBackground() context.Context {
	if p.ctx != nil {
		return p.ctx
	}
	return context.Background()
}
func (p *mosiPeer) registerChannel(name string, handler scm.ClusterRequestHandler) func() {
	if len(name) == 0 || len(name) > 255 {
		panic("invalid node channel")
	}
	p.mu.Lock()
	if p.channels == nil {
		p.channels = &sync.Map{}
	}
	channels := p.channels
	p.mu.Unlock()
	if _, exists := channels.LoadOrStore(name, handler); exists {
		panic("duplicate node channel: " + name)
	}
	return func() { channels.Delete(name) }
}

// Worker scheduling is an adapter, shared by binary and Scheme handlers.
func workerChannel(receive func(*mosiPeer, []byte) error, finished ...func()) scm.ClusterRequestHandler {
	return channelHandler(func(p *mosiPeer, payload []byte, size int, readyAt time.Time) bool {
		if p.packetBytes.Add(int64(size)) > 32<<20 {
			p.packetBytes.Add(-int64(size))
			return false
		}
		p.workers.Add(1)
		go func() {
			for _, finish := range finished {
				defer finish()
			}
			defer p.workers.Done()
			defer p.packetBytes.Add(-int64(size))
			defer func() {
				if recover() != nil {
					p.close()
				}
			}()
			if p.waitDispatch(readyAt) {
				if err := receive(p, payload); err != nil {
					p.close()
				}
			}
		}()
		return true
	})
}
func registerMosiChannel(p *mosiPeer, receive func(*mosiPacket) error) {
	if receive == nil {
		return
	}
	p.packet = receive
	p.channels.LoadOrStore("mosi", channelHandler(func(peer *mosiPeer, payload []byte, size int, readyAt time.Time) bool {
		packet, err := decodeMosiPacket(payload)
		return err == nil && peer.dispatchPacket(packet, size, readyAt)
	}))
}

// These are the replication command protocol's payloads, not transport headers.
// The caller chooses the return URI. The reply is an ordinary channel delivery.
type mosiRemoteError string

func (e mosiRemoteError) Error() string { return string(e) }

func registerNodeCommand(p *mosiPeer, name string, execute func(context.Context, []byte) (any, error)) {
	p.registerChannel(name, workerChannel(func(peer *mosiPeer, payload []byte) error {
		if len(payload) < 1 || payload[0] == 0 || int(payload[0])+1 > len(payload) {
			return fmt.Errorf("invalid return channel")
		}
		returnChannel := string(payload[1 : 1+int(payload[0])])
		parameters := payload[1+int(payload[0]):]
		if len(returnChannel) == 0 || len(returnChannel) > 255 {
			return fmt.Errorf("invalid return channel")
		}
		var result []byte
		var failure string
		func() {
			defer func() {
				if caught := recover(); caught != nil {
					failure = fmt.Sprint(caught)
				}
			}()
			value, err := execute(peer.ctxOrBackground(), parameters)
			if err == nil {
				result, err = marshalNode(value)
			}
			if err != nil {
				failure = err.Error()
			}
		}()
		data := append([]byte{0}, result...)
		if failure != "" {
			data = append([]byte{1}, []byte(failure)...)
		}
		return peer.send(peer.ctxOrBackground(), mosiMessage{Channel: returnChannel, Payload: data})
	}))
}

// nodeResultReceiver buffers an immediate response before Wait. Only the
// consumer waits; publication and delivery never rendezvous with that waiter.
type nodeResultReceiver struct {
	done    chan struct{}
	once    sync.Once
	payload []byte
	release func()
	cleanup runtime.Cleanup
}

func (p *mosiPeer) createResultChannel() (string, *nodeResultReceiver) {
	name := "return/" + strconv.FormatUint(p.next.Add(1), 10)
	r := &nodeResultReceiver{done: make(chan struct{})}
	weakReceiver := weak.Make(r)
	r.release = p.registerChannel(name, workerChannel(func(_ *mosiPeer, payload []byte) error {
		if receiver := weakReceiver.Value(); receiver != nil {
			receiver.once.Do(func() {
				receiver.payload = payload
				receiver.release()
				receiver.cleanup.Stop()
				close(receiver.done)
			})
		}
		return nil
	}))
	r.cleanup = runtime.AddCleanup(r, func(release func()) { release() }, r.release)
	return name, r
}
func (r *nodeResultReceiver) wait(ctx context.Context, peer *mosiPeer) ([]byte, error) {
	defer runtime.KeepAlive(r)
	defer r.cleanup.Stop()
	defer r.release()
	select {
	case <-r.done:
		return r.payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-peer.done:
		return nil, fmt.Errorf("node connection closed")
	}
}

// startCommand registers before sending and returns without waiting for a reply.
// Independent commands can all be started before their consumers wait.
func (p *mosiPeer) startCommand(ctx context.Context, method string, input any) (*nodeResultReceiver, error) {
	data, err := encodeCommandParameters(method, input)
	if err != nil {
		return nil, err
	}
	channel, receiver := p.createResultChannel()
	payload := append(append([]byte{byte(len(channel))}, channel...), data...)
	err = p.send(ctx, mosiMessage{Channel: method, Payload: payload})
	if err != nil {
		receiver.release()
		receiver.cleanup.Stop()
		return nil, err
	}
	return receiver, nil
}
func (p *mosiPeer) request(ctx context.Context, method string, input, output any) error {
	receiver, err := p.startCommand(ctx, method, input)
	if err != nil {
		return err
	}
	payload, err := receiver.wait(ctx, p)
	if err != nil {
		return err
	}
	if len(payload) == 0 || payload[0] > 1 {
		return fmt.Errorf("invalid command result")
	}
	if payload[0] == 1 {
		return mosiRemoteError(string(payload[1:]))
	}
	if output != nil {
		return unmarshalNode(payload[1:], output)
	}
	return nil
}

func registerScanChannels(p *mosiPeer, session *scanRPCSession) {
	p.channels = &session.service.channels
	p.scanSession = session
	session.service.channelOnce.Do(func() {
		// No per-retired-channel tombstones: a state-free disposer returns
		// unused native grants carried by late deliveries after deregistration.
		p.channels.Store("", workerChannel(func(peer *mosiPeer, payload []byte) error {
			return peer.scanService.discardHandlePayload(peer, payload)
		}))
		for _, name := range []string{"object.declare", "object.forget"} {
			name := name
			p.registerChannel(name, channelHandler(func(peer *mosiPeer, payload []byte, _ int, _ time.Time) bool {
				// These immutable declarations must precede later tasks on this
				// stream. They never evaluate Scheme or execute storage work.
				return peer.publishObjectView(name, payload) == nil
			}))
		}
		for _, name := range []string{"scan.cancel", "object.discard"} {
			name := name
			p.registerChannel(name, workerChannel(func(peer *mosiPeer, payload []byte) error {
				_, err := peer.scanSession.handle(peer.ctxOrBackground(), name, payload)
				return err
			}))
		}
		for _, name := range []string{"scan.describe", "scan.execute", "insert.execute", "cluster.hello", "cluster.try_read", "cluster.report", "cluster.reserve_ids", "cluster.location", "cluster.location_update", "cluster.schema_locations"} {
			name := name
			p.registerChannel(name, workerChannel(func(peer *mosiPeer, payload []byte) error {
				_, err := peer.scanSession.receiveInvocation(peer.ctxOrBackground(), name, payload)
				return err
			}))
		}
	})
}

func registerReplicationChannels(p *mosiPeer, names []string, execute func(context.Context, string, []byte) (any, error), packet func(*mosiPacket) error) {
	for _, name := range names {
		name := name
		registerNodeCommand(p, name, func(ctx context.Context, payload []byte) (any, error) { return execute(ctx, name, payload) })
	}
	registerMosiChannel(p, packet)
}

// readAuth is a bootstrap consumer wait. Its scoped channel uses the same raw
// dispatcher as application handlers; only the socket writer sends messages.
// The permanent reader starts after authentication publishes immutable identity.
func (p *mosiPeer) readAuth(channel string, output any) error {
	received := false
	var decodeErr error
	release := p.registerChannel(channel, channelHandler(func(_ *mosiPeer, payload []byte, _ int, _ time.Time) bool {
		decodeErr = unmarshalNode(payload, output)
		received = true
		return decodeErr == nil
	}))
	defer release()
	started := time.Now()
	kind, frame, err := p.conn.ReadMessage()
	if err != nil {
		return err
	}
	if kind != websocket.BinaryMessage {
		return fmt.Errorf("binary node authentication required")
	}
	if profile := p.receiveIO; profile != nil {
		profile.stats.operations.Add(1)
		ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
		defer cancel()
		ready := time.Now().Add(profile.latency)
		if !profile.transfer(ctx, len(frame), started) || !profile.wait(ctx, ready) {
			return fmt.Errorf("node authentication throttle deadline")
		}
	}
	message, err := decodeMosiMessage(frame)
	if err != nil || !p.dispatchMessage(message, len(frame), time.Time{}) || !received {
		return fmt.Errorf("invalid node authentication")
	}
	return decodeErr
}

func (w *mosiWriter) member(p clusterMember) { w.text(p.ID); w.text(p.Boot); w.text(p.Address) }
func (r *nodeReader) member() clusterMember {
	return clusterMember{ID: string(r.blob()), Boot: string(r.blob()), Address: string(r.blob())}
}
func (w *mosiWriter) hello(p clusterHello) {
	w.number(uint64(p.Version))
	w.member(p.Self)
	w.number(uint64(len(p.Members)))
	for _, p := range p.Members {
		w.member(p)
	}
}
func (r *nodeReader) hello() clusterHello {
	p := clusterHello{Version: int(r.uint()), Self: r.member()}
	n := r.uint()
	if n > uint64(len(r.data))/3 {
		panic("truncated cluster members")
	}
	if n > 0 {
		p.Members = make([]clusterMember, int(n))
	}
	for i := range p.Members {
		p.Members[i] = r.member()
	}
	return p
}

// Directory entries have their own compact layout. These are native resource
// parameters, not transport metadata; presence bits omit absent optional state.
func (w *mosiWriter) record(p replicaRecord) {
	w.resource(p.Resource)
	w.number(p.Epoch)
	flags := byte(0)
	if p.State != "" {
		flags |= 1
	}
	if p.Owner != "" {
		flags |= 2
	}
	if p.Content != "" {
		flags |= 4
	}
	if len(p.Readers) > 0 {
		flags |= 8
	}
	if len(p.Forwards) > 0 {
		flags |= 16
	}
	if p.Handoff != "" {
		flags |= 32
	}
	if p.InitialClaimed {
		flags |= 64
	}
	if len(p.Requests) > 0 {
		flags |= 128
	}
	w.data = append(w.data, flags)
	if flags&1 != 0 {
		w.text(p.State)
	}
	if flags&2 != 0 {
		w.text(p.Owner)
	}
	if flags&4 != 0 {
		w.text(p.Content)
	}
	if flags&8 != 0 {
		w.readers(p.Readers)
	}
	if flags&16 != 0 {
		w.routes(p.Forwards)
	}
	if flags&32 != 0 {
		w.text(p.Handoff)
	}
	if flags&128 != 0 {
		w.number(uint64(len(p.Requests)))
		for key, mark := range p.Requests {
			w.text(key)
			w.text(mark.Boot)
			w.number(mark.Sequence)
			w.text(mark.ID)
			bits := byte(0)
			if mark.Write {
				bits |= 1
			}
			if mark.Completed {
				bits |= 2
			}
			w.data = append(w.data, bits)
		}
	}
}
func (r *nodeReader) record() replicaRecord {
	p := replicaRecord{Resource: r.resource(), Epoch: r.uint()}
	flags := r.octet()
	p.InitialClaimed = flags&64 != 0
	if flags&1 != 0 {
		p.State = string(r.blob())
	}
	if flags&2 != 0 {
		p.Owner = string(r.blob())
	}
	if flags&4 != 0 {
		p.Content = string(r.blob())
	}
	if flags&8 != 0 {
		p.Readers = r.readers()
	}
	if flags&16 != 0 {
		p.Forwards = r.routes()
	}
	if flags&32 != 0 {
		p.Handoff = string(r.blob())
	}
	if flags&128 != 0 {
		n := r.uint()
		if n > uint64(len(r.data))/5 {
			panic("truncated request history")
		}
		p.Requests = make(map[string]replicaRequestMark, int(n))
		for range n {
			key := string(r.blob())
			m := replicaRequestMark{Boot: string(r.blob()), Sequence: r.uint(), ID: string(r.blob())}
			bits := r.octet()
			if bits&^3 != 0 {
				panic("invalid request flags")
			}
			m.Write = bits&1 != 0
			m.Completed = bits&2 != 0
			p.Requests[key] = m
		}
	}
	return p
}
func (w *mosiWriter) records(records []replicaRecord) {
	w.number(uint64(len(records)))
	for _, p := range records {
		w.record(p)
	}
}
func (r *nodeReader) records() []replicaRecord {
	n := r.uint()
	if n > uint64(len(r.data))/4 {
		panic("truncated directory")
	}
	if n == 0 {
		return nil
	}
	records := make([]replicaRecord, int(n))
	for i := range records {
		records[i] = r.record()
	}
	return records
}
func (w *mosiWriter) locations(p clusterSchemaLocations) {
	w.resource(p.Header.Resource)
	w.text(p.Header.ID)
	w.number(p.Header.Epoch)
	w.text(p.Header.Content)
	w.text(p.After)
	w.records(p.Entries)
}
func (r *nodeReader) locations() clusterSchemaLocations {
	return clusterSchemaLocations{Header: mosiPacket{Resource: r.resource(), ID: string(r.blob()), Epoch: r.uint(), Content: string(r.blob())}, After: string(r.blob()), Entries: r.records()}
}

// Replication methods have independent parameter contracts. Login never carries
// SQL/history/boot fields; recovery never carries SQL; retirement has no body.
func encodeCommandParameters(method string, input any) (data []byte, err error) {
	p, ok := input.(replicaRequest)
	if !ok {
		return marshalNode(input)
	}
	w := mosiWriter{data: make([]byte, 0, 128)}
	w.text(p.Database)
	w.text(p.User)
	w.text(p.Hash)
	w.blob(p.Salt)
	w.blob(p.Scramble)
	switch method {
	case "login":
	case "hello":
		w.text(p.Boot)
		w.records(p.Resources)
	case "hello_done":
		w.text(p.Boot)
	case "admin":
		if p.PostgreSQL {
			w.data = append(w.data, 1)
		} else {
			w.data = append(w.data, 0)
		}
		w.data = append(w.data, p.SQL...)
	default:
		return nil, fmt.Errorf("unknown replication parameters: %s", method)
	}
	return w.data, nil
}
func decodeReplicaParameters(method string, data []byte, p *replicaRequest) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("invalid %s parameters: %v", method, failure)
		}
	}()
	r := nodeReader{data}
	*p = replicaRequest{Database: string(r.blob()), User: string(r.blob()), Hash: string(r.blob()), Salt: r.blob(), Scramble: r.blob()}
	switch method {
	case "login":
	case "hello":
		p.Boot = string(r.blob())
		p.Resources = r.records()
	case "hello_done":
		p.Boot = string(r.blob())
	case "admin":
		flag := r.octet()
		if flag > 1 {
			panic("invalid SQL dialect")
		}
		p.PostgreSQL = flag == 1
		p.SQL = string(r.data)
		r.data = nil
	default:
		return fmt.Errorf("unknown replication parameters: %s", method)
	}
	if len(r.data) != 0 {
		return fmt.Errorf("trailing replication parameters")
	}
	return nil
}
