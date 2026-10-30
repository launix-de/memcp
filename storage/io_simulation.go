/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
This program is free software under the GNU General Public License, version 3
or later, WITHOUT ANY WARRANTY. See <https://www.gnu.org/licenses/>.
*/
package storage

import "fmt"
import "math"
import "sync"
import "time"
import "context"
import "sync/atomic"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type ioCounters struct {
	operations atomic.Uint64
	bytes      atomic.Uint64
	waitNS     atomic.Uint64
}

func (s *ioCounters) snapshot() map[string]uint64 {
	return map[string]uint64{"operations": s.operations.Load(), "bytes": s.bytes.Load(), "wait_ns": s.waitNS.Load()}
}

var clusterReceiveCounters, fakeRemoteCounters ioCounters

type ioSimulation struct {
	latency        time.Duration
	bytesPerSecond float64
	stats          *ioCounters
	mu             sync.Mutex
	next           time.Time
}

func newIOSimulation(mbps, latencyMS float64, stats *ioCounters) (*ioSimulation, error) {
	if math.IsNaN(mbps) || math.IsInf(mbps, 0) || mbps < 0 || math.IsNaN(latencyMS) || math.IsInf(latencyMS, 0) || latencyMS < 0 || latencyMS > 3600000 {
		return nil, fmt.Errorf("I/O simulation requires finite nonnegative Mbps and latency_ms (at most one hour)")
	}
	if mbps > 0 && float64(mosiMaxMessageBytes)/(mbps*125000) > 3600 {
		return nil, fmt.Errorf("I/O simulation bandwidth would delay one frame by over an hour")
	}
	return &ioSimulation{latency: time.Duration(latencyMS * float64(time.Millisecond)), bytesPerSecond: mbps * 125000, stats: stats}, nil
}

// Timers belong to transport/backend operations, never to row execution.
func (s *ioSimulation) wait(ctx context.Context, deadline time.Time) bool {
	delay := time.Until(deadline)
	if delay <= 0 {
		return ctx.Err() == nil
	}
	started := time.Now()
	timer := time.NewTimer(delay)
	defer timer.Stop()
	defer func() { s.stats.waitNS.Add(uint64(time.Since(started))) }()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Each backend instance shares an aggregate bandwidth budget across its streams.
// Actual filesystem/encoding time counts toward the reserved transmission time.
func (s *ioSimulation) transfer(ctx context.Context, size int, started time.Time) bool {
	s.stats.bytes.Add(uint64(size))
	return s.pace(ctx, size, started)
}

func (s *ioSimulation) pace(ctx context.Context, size int, started time.Time) bool {
	if size == 0 || s.bytesPerSecond == 0 {
		return true
	}
	s.mu.Lock()
	if s.next.After(started) {
		started = s.next
	}
	s.next = started.Add(time.Duration(float64(size) / s.bytesPerSecond * float64(time.Second)))
	deadline := s.next
	s.mu.Unlock()
	return s.wait(ctx, deadline)
}

func (s *ioSimulation) request() {
	s.stats.operations.Add(1)
	if s.latency > 0 {
		s.wait(context.Background(), time.Now().Add(s.latency))
	}
}

var clusterReceiveProfile atomic.Pointer[ioSimulation]

func activeClusterReceiveIO() *ioSimulation {
	return clusterReceiveProfile.Load()
}

// ConfigureClusterIO is startup-only. Connections borrow immutable limits and
// share this node's ingress budget. Zero limits leave transport at full speed.
func ConfigureClusterIO(mbps, latencyMS float64) error {
	profile, err := newIOSimulation(mbps, latencyMS, &clusterReceiveCounters)
	if err == nil {
		clusterReceiveProfile.Store(profile)
	}
	return err
}

func initIOInstrumentation(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "io_stats", Fn: func(...scm.Scmer) scm.Scmer {
		data, _ := json.Marshal(map[string]any{"cluster_receive": clusterReceiveCounters.snapshot(), "fake_remote": fakeRemoteCounters.snapshot()})
		return scm.NewString(string(data))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "Diagnostic process-local transport/backend operations, actual bytes and injected wait nanoseconds; counters are sampled outside row execution", Params: []*scm.TypeDescriptor{}, Return: &scm.TypeDescriptor{Kind: "string"}}})
}
