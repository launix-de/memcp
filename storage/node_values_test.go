/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "context"
import "testing"

func TestValueRetirementJoinsAdmittedDecoders(t *testing.T) {
	peer := &mosiPeer{ctx: context.Background(), done: make(chan struct{})}
	entered, finish, retired := make(chan struct{}), make(chan struct{}), make(chan struct{})
	handler := workerChannel(func(*mosiPeer, []byte) error { close(entered); <-finish; return nil })
	delivery := &channelDelivery{Context: peer.ctx, peer: peer, accepted: true}
	handler.HandleClusterRequest(delivery, nil)
	<-entered
	peer.afterReceivedValues(func() { close(retired) })
	select {
	case <-retired:
		t.Fatal("retired value while an earlier decoder was admitted")
	default:
	}
	close(finish)
	select {
	case <-retired:
	case <-time.After(time.Second):
		t.Fatal("retirement did not finish after decoder")
	}
	peer.workers.Wait()
}

func TestValueRetirementRetriesQueueRefusal(t *testing.T) {
	peer := &mosiPeer{ctx: context.Background(), done: make(chan struct{}), outgoing: newMosiMailbox()}
	service := &nodeScanService{peers: map[*mosiPeer]bool{peer: true}}
	peer.scanService = service
	defer service.stopObjects()
	peer.outgoing.bytes.Store(32 << 20)
	if err := peer.sendValueRelease("test-value", []byte{0}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		registry := service.objectRegistry()
		registry.mu.Lock()
		pending := len(registry.releases)
		registry.mu.Unlock()
		if peer.outgoing.rejected.Load() > 1 && pending == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("release retry did not retain rejected notification")
		}
		time.Sleep(time.Millisecond)
	}
	peer.outgoing.bytes.Store(0)
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if message.Channel != "value/test-value" || len(message.Payload) != 1 || message.Payload[0] != 0 {
			t.Fatal("wrong retirement notification", message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("release did not retry independently")
	}
	close(peer.done)
	if !(&clusterValueRelease{peer, "test-value", 0}).deliver() {
		t.Fatal("disconnected edge retained a declaration")
	}
}
