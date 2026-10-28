/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "context"
import "testing"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

func deliverTestCompletion(t *testing.T, entry any, result scanCompletion) {
	t.Helper()
	_, err := entry.(*receiverRoute).receiver.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewAny(result)}})
	if err != nil {
		t.Error(err)
	}
}
func receiveTestValue(receiver *clusterReceiver) error {
	_, err := receiver.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewInt(1)}})
	return err
}
func TestClusterReceiverProductionAndParallelLimits(t *testing.T) {
	var calls atomic.Int64
	receiver := newClusterReceiver(1, func(context.Context, scm.Scmer) error { calls.Add(1); return nil })
	defer receiver.Close()
	if err := receiveTestValue(receiver); err != nil {
		t.Fatal(err)
	}
	select {
	case <-receiver.done:
		t.Fatal("completed during production")
	default:
	}
	if !receiver.Expect(64) {
		t.Fatal("failed to reserve more work")
	}
	var workers sync.WaitGroup
	for i := 0; i < 128; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); _ = receiveTestValue(receiver) }()
	}
	workers.Wait()
	if calls.Load() != 65 {
		t.Fatal("limit consumed more than once", calls.Load())
	}
	select {
	case <-receiver.done:
		t.Fatal("completed before seal")
	default:
	}
	receiver.Seal()
	<-receiver.done
	if receiver.Expect(1) || receiveTestValue(receiver) == nil {
		t.Fatal("late work admitted")
	}
}
func TestClusterReceiverSingleUnlimitedAndCleanup(t *testing.T) {
	for _, limit := range []int64{1, -1} {
		calls := 0
		receiver := newClusterReceiver(limit, func(context.Context, scm.Scmer) error { calls++; return nil })
		receiver.Seal()
		if err := receiveTestValue(receiver); err != nil {
			t.Fatal(err)
		}
		if limit == -1 {
			for i := 0; i < 100; i++ {
				if err := receiveTestValue(receiver); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case <-receiver.done:
				t.Fatal("unlimited receiver lost ownership")
			default:
			}
		}
		receiver.Close()
		if receiveTestValue(receiver) == nil {
			t.Fatal("callback after cleanup")
		}
		want := 1
		if limit == -1 {
			want = 101
		}
		if calls != want {
			t.Fatal(calls)
		}
	}
}
func TestClusterReceiverCloseJoinsCallback(t *testing.T) {
	entered, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	receiver := newClusterReceiver(-1, func(context.Context, scm.Scmer) error { close(entered); <-release; return nil })
	delivered := make(chan error, 1)
	go func() { delivered <- receiveTestValue(receiver) }()
	<-entered
	go func() { receiver.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("cleanup returned before callback")
	default:
	}
	close(release)
	if err := <-delivered; err != nil {
		t.Fatal(err)
	}
	<-closed
	if receiveTestValue(receiver) == nil {
		t.Fatal("late callback admitted")
	}
}
func TestClusterReceiverCancelledBeforeCallback(t *testing.T) {
	receiver := newClusterReceiver(-1, func(context.Context, scm.Scmer) error { t.Fatal("cancelled callback ran"); return nil })
	defer receiver.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := receiver.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewNil()}})
	if err != context.Canceled {
		t.Fatal(err)
	}
}

func TestClusterReceiverRemoteUnlimitedUsesObjectLifetime(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	calls := 0
	receiver := newClusterReceiver(-1, func(_ context.Context, value scm.Scmer) error {
		if !value.IsSlice() || len(value.Slice()) != 2 {
			t.Error("lost session call parameters", value)
		}
		calls++
		return nil
	})
	defer receiver.Close()
	receiver.Seal()
	ref, err := owner.exportObject(source, receiver, scm.NewNil())
	if err != nil {
		t.Fatal(err)
	}
	imported, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	object, _ := scm.ClusterObjectOf(imported)
	for _, name := range []string{"get", "set"} {
		_, err := object.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewString(name), scm.NewInt(7)})}})
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("wrong callback count", calls)
	}
	receiver.Close()
	if _, err := object.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewNil()}}); err == nil {
		t.Fatal("late remote callback ran")
	}
}

func TestClusterReceiverEachParallelValueIsProcessedOnce(t *testing.T) {
	seen := make([]bool, 64)
	receiver := newClusterReceiver(64, func(_ context.Context, value scm.Scmer) error {
		index := int(value.Int())
		if seen[index] {
			t.Errorf("duplicate callback for %d", index)
		}
		seen[index] = true
		return nil
	})
	receiver.Seal()
	defer receiver.Close()
	var workers sync.WaitGroup
	for i := range seen {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := receiver.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewInt(int64(i))}})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	<-receiver.done
	for i, processed := range seen {
		if !processed {
			t.Error("missing callback", i)
		}
	}
}
