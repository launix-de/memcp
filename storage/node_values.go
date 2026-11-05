/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Each peer owns what it has declared and what it has received. Declarations
// precede references on the existing FIFO; GC only sends retirement messages.
func (peer *mosiPeer) valuePeer() *scm.SerializationPeer {
	peer.valueOnce.Do(func() {
		values := &scm.SerializationPeer{
			Declare: func(id string, source []byte, lambda bool) error {
				value, err := uuid.Parse(id)
				if err != nil {
					return err
				}
				channel := "value.declare"
				if lambda {
					channel = "lambda.declare"
				}
				return peer.send(peer.ctxOrBackground(), mosiMessage{Channel: channel, Payload: append(value[:], source...)})
			},
			Send: func(id string, payload []byte) error {
				return peer.send(peer.ctxOrBackground(), mosiMessage{Channel: "value/" + id, Payload: payload})
			},
			Register: func(id string, receive func([]byte) error) func() {
				return peer.registerChannel("value/"+id, channelHandler(func(connection *mosiPeer, payload []byte, _ int, _ time.Time) bool {
					if connection.objectPeer != peer.objectPeer {
						return false
					}
					return receive(payload) == nil
				}))
			},
			Prepare:       prepareScanProcedure(scm.NewSymbol(scm.LambdaScopeSymbol())),
			AfterReceived: peer.afterReceivedValues,
		}
		peer.values.Store(values)
	})
	return peer.values.Load()
}
func registerValueChannels(peer *mosiPeer) {
	for _, name := range []string{"value.declare", "lambda.declare"} {
		name := name
		peer.registerChannel(name, channelHandler(func(connection *mosiPeer, payload []byte, _ int, _ time.Time) bool {
			if len(payload) < 16 {
				return false
			}
			id, err := uuid.FromBytes(payload[:16])
			if err != nil {
				return false
			}
			return connection.valuePeer().Receive(id.String(), payload[16:], name == "lambda.declare") == nil
		}))
	}
}

// Retirement acknowledgement is a receive-side publication barrier. It must
// not delete a value while a preceding task is still queued for decoding. The
// counters cover messages, never shard rows, and add no waiting to Send.
type valueReceiveEpoch struct {
	activity atomic.Uint64
	done     chan struct{}
}

const valueEpochStopped uint64 = 1 << 63

func (epoch *valueReceiveEpoch) hold() bool {
	for {
		old := epoch.activity.Load()
		if old&valueEpochStopped != 0 {
			return false
		}
		if epoch.activity.CompareAndSwap(old, old+1) {
			return true
		}
	}
}
func (epoch *valueReceiveEpoch) leave() {
	if epoch.activity.Add(^uint64(0)) == valueEpochStopped {
		close(epoch.done)
	}
}
func (peer *mosiPeer) holdReceivedValues() *valueReceiveEpoch {
	for {
		epoch := peer.valueEpoch.Load()
		if epoch == nil {
			next := &valueReceiveEpoch{done: make(chan struct{})}
			peer.valueEpoch.CompareAndSwap(nil, next)
			continue
		}
		if epoch.hold() {
			return epoch
		}
	}
}
func (peer *mosiPeer) afterReceivedValues(release func()) {
	next := &valueReceiveEpoch{done: make(chan struct{})}
	old := peer.valueEpoch.Swap(next)
	if old == nil {
		release()
		return
	}
	for {
		count := old.activity.Load()
		if old.activity.CompareAndSwap(count, count|valueEpochStopped) {
			if count == 0 {
				close(old.done)
			}
			break
		}
	}
	peer.workers.Add(1)
	go func() { defer peer.workers.Done(); <-old.done; release() }()
}

func queuedValuePublications(refs []scm.ClusterReference) {
	for _, ref := range refs {
		ref.Publication.Queued()
	}
}
