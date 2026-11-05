//go:build linux

/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "os"
import "time"
import "context"
import "golang.org/x/sys/unix"

// Linux's Go netpoll timeout rounds idle sub-millisecond timers up to one
// millisecond. A kernel timer wakes the poller at the requested deadline without
// keeping a CPU busy. Failure falls back to the ordinary, conservative timer.
func waitSubMillisecondIO(ctx context.Context, deadline time.Time) (ready, supported bool) {
	if ctx.Err() != nil {
		return false, true
	}
	fd, err := unix.TimerfdCreate(unix.CLOCK_MONOTONIC, unix.TFD_NONBLOCK|unix.TFD_CLOEXEC)
	if err != nil {
		return false, false
	}
	file := os.NewFile(uintptr(fd), "memcp-io-deadline")
	defer file.Close()
	delay := time.Until(deadline)
	if delay <= 0 {
		return ctx.Err() == nil, true
	}
	if err := unix.TimerfdSettime(fd, 0, &unix.ItimerSpec{Value: unix.NsecToTimespec(delay.Nanoseconds())}, nil); err != nil {
		return false, false
	}
	// Closing through os.File also wakes its parked Go reader. An integer-fd
	// close in the callback could accidentally close a subsequently reused fd.
	stop := context.AfterFunc(ctx, func() { file.Close() })
	defer stop()
	var expired [8]byte
	_, err = file.Read(expired[:])
	if err != nil && ctx.Err() == nil {
		return false, false
	}
	return err == nil && ctx.Err() == nil, true
}
