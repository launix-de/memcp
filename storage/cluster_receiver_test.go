/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "time"
import "errors"
import "context"
import "testing"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

func deliverTestCompletion(t *testing.T, entry any, result scanCompletion) {
	t.Helper()
	entry.(*handleSlot).handler.Handle(scm.NewString(string(mustNodeMetadata(result))))
}
func receiveTestValue(receiver *clusterReceiver, ctx context.Context, value int64) error {
	_, err := receiver.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewInt(value)}})
	return err
}

func TestClusterReceiverReservationDoesNotWaitForCallback(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	receiver := newClusterReceiver(1, func(context.Context, scm.Scmer) (scm.Scmer, error) {
		close(entered)
		<-release
		return scm.NewNil(), nil
	})
	delivered := make(chan error, 1)
	go func() { delivered <- receiveTestValue(receiver, context.Background(), 1) }()
	<-entered
	defer func() { close(release); <-delivered; receiver.Close() }()
	reserved := make(chan bool, 1)
	go func() { reserved <- receiver.Expect(1) }()
	select {
	case ok := <-reserved:
		if !ok {
			t.Fatal("reservation failed while callback was active")
		}
	case <-time.After(time.Second):
		t.Fatal("callback blocked admission of independent work")
	}
}

func TestUniqueChannelCancellationJoinsActiveCommandBeforeUnlock(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_channel_cancel")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	guard := tbl.guardUniqueWrites([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan *clusterReceiver, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		ready <- value.Any().(*clusterReceiver)
		return scm.NewNil(), nil
	})
	receiver.Seal()
	defer receiver.Close()
	entered, left := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := serveUniqueChannel(ctx, &guard, receiver, func(commandCtx context.Context, _ []scm.Scmer) (scm.Scmer, error) {
			close(entered)
			<-commandCtx.Done()
			defer close(left)
			return scm.NewNil(), commandCtx.Err()
		})
		done <- err
	}()
	commands := <-ready
	called := make(chan error, 1)
	go func() {
		_, err := callUniqueChannel(context.Background(), commands, scm.NewString("apply"))
		called <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("reservation did not join its active command")
	}
	select {
	case <-left:
	default:
		t.Fatal("reservation returned before its mutation callback")
	}
	if err := <-called; !errors.Is(err, context.Canceled) {
		t.Fatalf("command cancellation: %v", err)
	}
	reacquired := tbl.guardUniqueWrites([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0)
	reacquired.lock()
	reacquired.unlock()
}

func TestUniqueChannelCommandPanicReleasesReservation(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_channel_panic")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	guard := tbl.guardUniqueWrites([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0)
	ready := make(chan *clusterReceiver, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		ready <- value.Any().(*clusterReceiver)
		return scm.NewNil(), nil
	})
	receiver.Seal()
	defer receiver.Close()
	done := make(chan bool, 1)
	go func() {
		committed, _ := serveUniqueChannel(context.Background(), &guard, receiver, func(context.Context, []scm.Scmer) (scm.Scmer, error) {
			panic("unique-command-panic")
		})
		done <- committed
	}()
	commands := <-ready
	var failure any
	func() {
		defer func() { failure = recover() }()
		_, _ = callUniqueChannel(context.Background(), commands, scm.NewString("apply"))
	}()
	if failure != "unique-command-panic" {
		t.Fatalf("lost command panic: %v", failure)
	}
	select {
	case committed := <-done:
		if committed {
			t.Fatal("failed command committed its reservation")
		}
	case <-time.After(time.Second):
		t.Fatal("failed command retained its reservation")
	}
	guard.lock()
	guard.unlock()
}
func TestClusterReceiverProductionAndParallelLimits(t *testing.T) {
	var calls atomic.Int64
	receiver := newClusterReceiver(1, func(context.Context, scm.Scmer) (scm.Scmer, error) { calls.Add(1); return scm.NewNil(), nil })
	defer receiver.Close()
	if err := receiveTestValue(receiver, context.Background(), 1); err != nil {
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
		go func() { defer workers.Done(); _ = receiveTestValue(receiver, context.Background(), 1) }()
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
	if receiver.Expect(1) || receiveTestValue(receiver, context.Background(), 1) == nil {
		t.Fatal("late work admitted")
	}
}
func TestClusterReceiverSingleUnlimitedAndCleanup(t *testing.T) {
	for _, limit := range []int64{1, -1} {
		calls := 0
		receiver := newClusterReceiver(limit, func(context.Context, scm.Scmer) (scm.Scmer, error) { calls++; return scm.NewNil(), nil })
		receiver.Seal()
		if err := receiveTestValue(receiver, context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		if limit == -1 {
			for i := 0; i < 100; i++ {
				if err := receiveTestValue(receiver, context.Background(), 1); err != nil {
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
		if receiveTestValue(receiver, context.Background(), 1) == nil {
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
	receiver := newClusterReceiver(-1, func(context.Context, scm.Scmer) (scm.Scmer, error) {
		close(entered)
		<-release
		return scm.NewNil(), nil
	})
	delivered := make(chan error, 1)
	go func() { delivered <- receiveTestValue(receiver, context.Background(), 1) }()
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
	if receiveTestValue(receiver, context.Background(), 1) == nil {
		t.Fatal("late callback admitted")
	}
}
func TestClusterReceiverCancelledBeforeCallback(t *testing.T) {
	receiver := newClusterReceiver(-1, func(context.Context, scm.Scmer) (scm.Scmer, error) {
		t.Fatal("cancelled callback ran")
		return scm.NewNil(), nil
	})
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
	receiver := newClusterReceiver(-1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		if !value.IsSlice() || len(value.Slice()) != 2 {
			t.Error("lost session call parameters", value)
		}
		calls++
		return scm.NewNil(), nil
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
	receiver := newClusterReceiver(64, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		index := int(value.Int())
		if seen[index] {
			t.Errorf("duplicate callback for %d", index)
		}
		seen[index] = true
		return scm.NewNil(), nil
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

func waitReceiverBudget(t *testing.T, receiver *clusterReceiver, remaining int64) {
	t.Helper()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for receiver.remaining.Load() != remaining {
		select {
		case <-timeout.C:
			t.Fatal("delivery did not reserve its budget", remaining)
		case <-tick.C:
		}
	}
}

func TestClusterReceiverFIFOAndQueuedCancellation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var order []int64
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		if value.Int() == 0 {
			close(entered)
			<-release
		}
		order = append(order, value.Int())
		return scm.NewNil(), nil
	})
	results := make([]chan error, 5)
	for i := range results {
		results[i] = make(chan error, 1)
	}
	go func() { results[0] <- receiveTestValue(receiver, context.Background(), 0) }()
	<-entered
	defer receiver.Close()
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	if !receiver.Expect(4) {
		t.Fatal("active callback prevented further delegation")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	defer cancel()
	for value := 1; value <= 4; value++ {
		ctx := context.Background()
		if value == 2 {
			ctx = cancelled
		}
		go func() { results[value] <- receiveTestValue(receiver, ctx, int64(value)) }()
		waitReceiverBudget(t, receiver, int64(4-value))
	}
	receiver.Seal()
	if receiver.Expect(1) || receiveTestValue(receiver, context.Background(), 99) == nil {
		t.Fatal("sealed or exhausted receiver admitted excess work")
	}
	cancel()
	select {
	case err := <-results[2]:
		if err != context.Canceled {
			t.Fatal("queued cancellation lost", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled waiter remained behind the blocked callback")
	}
	select {
	case <-receiver.done:
		t.Fatal("budget exhaustion completed before accepted callbacks")
	default:
	}
	unblock()
	for _, value := range []int{0, 1, 3, 4} {
		if err := <-results[value]; err != nil {
			t.Fatal(err)
		}
	}
	<-receiver.done
	want := []int64{0, 1, 3, 4}
	if len(order) != len(want) {
		t.Fatal("wrong callback count", order)
	}
	for i, value := range want {
		if order[i] != value {
			t.Fatal("callback order changed", order)
		}
	}
}

func TestClusterReceiverCloseCancelsAndJoinsDeliveries(t *testing.T) {
	entered, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	receiver := newClusterReceiver(3, func(ctx context.Context, _ scm.Scmer) (scm.Scmer, error) {
		calls.Add(1)
		close(entered)
		<-ctx.Done()
		close(cancelled)
		<-release
		return scm.NewNil(), ctx.Err()
	})
	unblock := sync.OnceFunc(func() { close(release) })
	defer receiver.Close()
	defer unblock()
	results := make(chan error, 3)
	go func() { results <- receiveTestValue(receiver, context.Background(), 0) }()
	<-entered
	for value := 1; value <= 2; value++ {
		go func() { results <- receiveTestValue(receiver, context.Background(), int64(value)) }()
	}
	waitReceiverBudget(t, receiver, 0)
	closed := make(chan struct{}, 2)
	for range 2 {
		go func() { receiver.Close(); closed <- struct{}{} }()
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		unblock()
		t.Fatal("Close did not cancel its active callback")
	}
	// Queued callers return without waiting for the active callback to exit.
	for range 2 {
		if err := <-results; err != context.Canceled {
			t.Fatal("queued delivery survived Close", err)
		}
	}
	if receiveTestValue(receiver, context.Background(), 99) == nil || receiver.Expect(1) {
		t.Fatal("Close left admission open")
	}
	select {
	case <-closed:
		t.Fatal("Close returned before callback state was released")
	case <-receiver.done:
		t.Fatal("completion preceded callback exit")
	default:
	}
	unblock()
	if err := <-results; err != context.Canceled {
		t.Fatal(err)
	}
	<-closed
	<-closed
	if calls.Load() != 1 {
		t.Fatal("queued callbacks executed during Close", calls.Load())
	}
}

func TestClusterReceiverFailureAdvancesFIFO(t *testing.T) {
	for _, failure := range []string{"error", "panic"} {
		t.Run(failure, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			want := errors.New("callback failure")
			var calls atomic.Int64
			receiver := newClusterReceiver(2, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
				calls.Add(1)
				if value.Int() == 0 {
					close(entered)
					<-release
					if failure == "panic" {
						panic(want)
					}
					return scm.NewNil(), want
				}
				return scm.NewNil(), nil
			})
			defer receiver.Close()
			defer unblock()
			receiver.Seal()
			first := make(chan any, 1)
			go func() {
				var result any
				defer func() {
					if raised := recover(); raised != nil {
						result = raised
					}
					first <- result
				}()
				result = receiveTestValue(receiver, context.Background(), 0)
			}()
			<-entered
			second := make(chan error, 1)
			go func() { second <- receiveTestValue(receiver, context.Background(), 1) }()
			waitReceiverBudget(t, receiver, 0)
			unblock()
			if actual := <-first; actual != want {
				t.Fatal("callback failure was replaced", actual)
			}
			if err := <-second; err != nil {
				t.Fatal("failure stranded the successor", err)
			}
			receiver.Close()
			if calls.Load() != 2 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestClusterReceiverImportedCloseCancelsPendingReply(t *testing.T) {
	service := &nodeScanService{}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	entered := make(chan struct{})
	value, err := scm.ReconstructClusterObject(scm.ClusterHandle{
		Reference: scm.ClusterReference{Type: "receiver"},
		Call: func(ctx context.Context, _ scm.ClusterRequest) (scm.Scmer, error) {
			close(entered)
			// Imported receive forwards CancelWithoutJoin: Close cancels the
			// reply wait even when an idle socket writer has not drained publication.
			return scm.NewNil(), service.control(ctx, peer, "object.call", nil, nil, false)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	receiver := value.Any().(*clusterReceiver)
	delivered := make(chan error, 1)
	go func() { delivered <- receiveTestValue(receiver, context.Background(), 1) }()
	<-entered
	closed := make(chan struct{})
	go func() { receiver.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		close(peer.done)
		<-closed
		t.Fatal("Close remained blocked behind a full transport queue")
	}
	if err := <-delivered; err != context.Canceled {
		t.Fatal("blocked send lost cancellation", err)
	}
	service.handles.Range(func(key, value any) bool {
		entry := value.(*handleSlot)
		if _, cleanup := entry.handler.(*abandonedControlHandle); !cleanup || !entry.owner.IsNil() {
			t.Error("cancelled send retained a query collector", key)
			return true
		}
		// A published request may still return. Only its state-free disposer
		// survives cancellation, and that late completion must retire it.
		payload, err := freezeScanValue(scm.NewString(string(mustNodeMetadata(scanCompletion{}))))
		if err != nil {
			t.Fatal(err)
		}
		if err := (&scanRPCSession{service: service, peer: peer}).handleValue(key.(uint64), payload); err != nil {
			t.Fatal(err)
		}
		return true
	})
	service.handles.Range(func(key, _ any) bool {
		t.Error("late completion retained its cleanup slot", key)
		return true
	})
}

func TestClusterReceiverRemoteCancellationAndIndependentDelivery(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	entered, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	blocked := newClusterReceiver(-1, func(ctx context.Context, _ scm.Scmer) (scm.Scmer, error) {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		<-release
		return scm.NewNil(), ctx.Err()
	})
	defer blocked.Close()
	defer unblock()
	importReceiver := func(receiver *clusterReceiver) *clusterReceiver {
		t.Helper()
		ref, err := owner.exportObject(source, receiver, scm.NewNil())
		if err != nil {
			t.Fatal(err)
		}
		value, err := client.importObject(context.Background(), peer, ref)
		if err != nil {
			t.Fatal(err)
		}
		return value.Any().(*clusterReceiver)
	}
	forwarded := importReceiver(blocked)
	defer forwarded.Close()
	delivered := make(chan error, 1)
	go func() { delivered <- receiveTestValue(forwarded, context.Background(), 1) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("remote callback did not start")
	}
	var otherCalls atomic.Int64
	other := newClusterReceiver(1, func(context.Context, scm.Scmer) (scm.Scmer, error) { otherCalls.Add(1); return scm.NewNil(), nil })
	other.Seal()
	defer other.Close()
	independent := importReceiver(other)
	defer independent.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := receiveTestValue(independent, ctx, 2); err != nil || otherCalls.Load() != 1 {
		t.Fatal("blocked return channel serialized independent work", err)
	}
	closed := make(chan struct{})
	go func() { forwarded.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("imported receiver did not cancel and join its call")
	}
	if err := <-delivered; err != context.Canceled {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("remote scope did not receive cancellation")
	}
	unblock()
	blocked.Close()
	if receiveTestValue(forwarded, context.Background(), 3) == nil {
		t.Fatal("closed imported receiver accepted a late delivery")
	}
}

func TestClusterReceiverRemoteDisconnectJoinsCallback(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	entered, cancelled := make(chan struct{}), make(chan struct{})
	receiver := newClusterReceiver(-1, func(ctx context.Context, _ scm.Scmer) (scm.Scmer, error) {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		return scm.NewNil(), ctx.Err()
	})
	defer receiver.Close()
	ref, err := owner.exportObject(source, receiver, scm.NewNil())
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	forwarded := value.Any().(*clusterReceiver)
	defer forwarded.Close()
	result := make(chan error, 1)
	go func() { result <- receiveTestValue(forwarded, context.Background(), 1) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("remote callback did not start")
	}
	peer.close()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("disconnect returned success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("disconnect stranded the forwarding callback")
	}
	forwarded.Close()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("disconnect did not cancel remote callback state")
	}
	receiver.Close()
	client.handles.Range(func(key, _ any) bool {
		t.Error("disconnect retained collector", key)
		return true
	})
}
