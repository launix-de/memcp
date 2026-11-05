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

type nodeCommand struct {
	Return string
	Data   []byte
}
type nodeCommandResult struct {
	Data  []byte
	Error string
}

func registerNodeCommand(p *mosiPeer, name string, execute func(context.Context, []byte) (any, error)) {
	p.registerChannel(name, workerChannel(func(peer *mosiPeer, payload []byte) error {
		var command nodeCommand
		if err := unmarshalNode(payload, &command); err != nil {
			return err
		}
		if len(command.Return) == 0 || len(command.Return) > 255 {
			return fmt.Errorf("invalid return channel")
		}
		result := nodeCommandResult{}
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					result.Error = fmt.Sprint(failure)
				}
			}()
			value, err := execute(peer.ctxOrBackground(), command.Data)
			if err == nil {
				result.Data, err = marshalNode(value)
			}
			if err != nil {
				result.Error = err.Error()
			}
		}()
		data, err := marshalNode(result)
		if err != nil {
			return err
		}
		return peer.send(peer.ctxOrBackground(), mosiMessage{Channel: command.Return, Payload: data})
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
	data, err := marshalNode(input)
	if err != nil {
		return nil, err
	}
	channel, receiver := p.createResultChannel()
	payload, err := marshalNode(nodeCommand{Return: channel, Data: data})
	if err == nil {
		err = p.send(ctx, mosiMessage{Channel: method, Payload: payload})
	}
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
	var result nodeCommandResult
	if err = unmarshalNode(payload, &result); err != nil {
		return err
	}
	if result.Error != "" {
		return mosiRemoteError(result.Error)
	}
	if output == nil {
		return nil
	}
	return unmarshalNode(result.Data, output)
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
		for _, name := range []string{"node.control", "scan.cancel", "object.discard"} {
			name := name
			p.registerChannel(name, workerChannel(func(peer *mosiPeer, payload []byte) error {
				_, err := peer.scanSession.handle(peer.ctxOrBackground(), name, payload)
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
