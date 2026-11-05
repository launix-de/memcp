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
