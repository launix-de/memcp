/*
 * go-mysqlstack
 * xelabs.org
 *
 * Copyright (c) 2021 XeLabs
 * Copyright (c) 2023-2026 Carl-Philip Hänsch
 * GPL License
 *
 */

package driver

import (
	"net"
	"testing"
	"time"

	"github.com/launix-de/go-mysqlstack/xlog"
)

func TestSessionDoneClosesOnClose(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	s := newSession(xlog.NewStdLog(xlog.Level(xlog.ERROR)), 1, "MemCP", server)
	defer s.Close()

	s.Close()

	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatalf("expected session Done channel to close")
	}
}

func TestSessionCloseCancelsCurrentQuery(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	s := newSession(xlog.NewStdLog(xlog.Level(xlog.ERROR)), 1, "MemCP", server)
	cancelled := make(chan struct{}, 1)
	s.SetQueryCancel(func() { cancelled <- struct{}{} })
	s.Close()
	s.Close()

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("expected current query to be cancelled")
	}
	select {
	case <-cancelled:
		t.Fatal("query cancellation ran more than once")
	default:
	}
}

func TestSessionClearQueryCancelPreservesCompletedQuery(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	s := newSession(xlog.NewStdLog(xlog.Level(xlog.ERROR)), 1, "MemCP", server)
	cancelled := false
	id := s.SetQueryCancel(func() { cancelled = true })
	s.ClearQueryCancel(id)
	s.Close()
	if cancelled {
		t.Fatal("completed query was cancelled during connection close")
	}
}
