/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "io"
import "sync"
import "time"
import "testing"
import "sync/atomic"
import "github.com/chzyer/readline"

type replPipeRead struct {
	reader  *io.PipeReader
	entered chan struct{}
	left    chan struct{}
}

func (r *replPipeRead) Read(b []byte) (int, error) {
	close(r.entered)
	n, err := r.reader.Read(b)
	close(r.left)
	return n, err
}

type replCloseObservation struct {
	io.ReadCloser
	calls   atomic.Int32
	entered chan struct{}
	proceed chan struct{}
}

func (r *replCloseObservation) Close() error {
	if r.calls.Add(1) == 1 {
		close(r.entered)
	}
	<-r.proceed
	return r.ReadCloser.Close()
}

type replPipeFixture struct {
	owner    *replOwner
	input    *replPipeRead
	writer   *io.PipeWriter
	observed *replCloseObservation
	release  func()
}

func newReplPipeFixture(t *testing.T) *replPipeFixture {
	t.Helper()
	reader, writer := io.Pipe()
	input := &replPipeRead{reader: reader, entered: make(chan struct{}), left: make(chan struct{})}
	cancel := readline.NewCancelableStdin(input)
	cfg := &readline.Config{
		Stdin: cancel, Stdout: io.Discard, Stderr: io.Discard,
		HistoryLimit: -1, DisableAutoSaveHistory: true,
		FuncGetWidth:       func() int { return 80 },
		FuncIsTerminal:     func() bool { return false },
		FuncMakeRaw:        func() error { return nil },
		FuncExitRaw:        func() error { return nil },
		FuncOnWidthChanged: func(func()) {},
	}
	if err := cfg.Init(); err != nil {
		t.Fatal(err)
	}
	observed := &replCloseObservation{ReadCloser: cfg.Stdin, entered: make(chan struct{}), proceed: make(chan struct{})}
	// Install before the terminal starts; no published config is mutated.
	cfg.Stdin = observed
	instance, err := readline.NewEx(cfg)
	if err != nil {
		cancel.Close()
		reader.Close()
		writer.Close()
		t.Fatal(err)
	}
	owner := &replOwner{instance: instance, input: cancel}
	var release sync.Once
	f := &replPipeFixture{owner: owner, input: input, writer: writer, observed: observed,
		release: func() { release.Do(func() { close(observed.proceed) }) }}
	t.Cleanup(func() {
		// Rescue blocked reads even if a future close-order regression fails.
		f.release()
		cancel.Close()
		writer.Close()
		reader.Close()
		owner.close()
		activeRepl.CompareAndSwap(owner, nil)
	})
	if previous := activeRepl.Swap(owner); previous != nil {
		activeRepl.Store(previous)
		t.Fatal("test found an already published REPL")
	}
	return f
}

func awaitReplEvent[T any](t *testing.T, event <-chan T, name string) T {
	t.Helper()
	select {
	case result := <-event:
		return result
	case <-time.After(5 * time.Second):
		t.Fatalf("did not observe %s", name)
		var zero T
		return zero
	}
}

func TestReplOwnerConcurrentCloseWithOpenPipe(t *testing.T) {
	for _, deferredFirst := range []bool{false, true} {
		name := "external_close_first"
		if deferredFirst {
			name = "deferred_close_first"
		}
		t.Run(name, func(t *testing.T) {
			f := newReplPipeFixture(t)
			readDone := make(chan error, 1)
			go func() {
				_, err := f.owner.instance.Readline()
				readDone <- err
			}()
			awaitReplEvent(t, f.input.entered, "underlying pipe read")
			firstDone := make(chan bool, 1)
			go func() {
				if deferredFirst {
					f.owner.close()
					firstDone <- true
				} else {
					firstDone <- CloseRepl()
				}
			}()
			awaitReplEvent(t, f.observed.entered, "instance close after cancellation")
			// Instance.Close is paused before it can send Operation.Close EOF:
			// EOF here must come from cancelling our own input reader first.
			if err := awaitReplEvent(t, readDone, "EOF before terminal close"); err != io.EOF {
				t.Fatalf("readline error = %v; want EOF", err)
			}
			secondDone := make(chan bool, 1)
			deferDone := make(chan struct{})
			started := make(chan struct{})
			go func() { started <- struct{}{}; secondDone <- CloseRepl() }()
			go func() { started <- struct{}{}; f.owner.close(); close(deferDone) }()
			awaitReplEvent(t, started, "concurrent public close invocation")
			awaitReplEvent(t, started, "deferred close invocation")
			f.release()
			if !awaitReplEvent(t, firstDone, "first close") || !awaitReplEvent(t, secondDone, "concurrent public close") {
				t.Fatal("active owner was absent during close")
			}
			awaitReplEvent(t, deferDone, "deferred close")
			// One actual Instance.Close closes Config.Stdin and Terminal.Stdin.
			if calls := f.observed.calls.Load(); calls != 2 {
				t.Fatalf("underlying close calls = %d; want exactly one instance close", calls)
			}
			assertReplPipeWriterStillOpen(t, f)
		})
	}
}

func assertReplPipeWriterStillOpen(t *testing.T, f *replPipeFixture) {
	t.Helper()
	written := make(chan error, 1)
	go func() { _, err := f.writer.Write([]byte("x")); written <- err }()
	if err := awaitReplEvent(t, written, "write to still-open pipe"); err != nil {
		t.Fatalf("REPL close closed the caller's pipe: %v", err)
	}
	awaitReplEvent(t, f.input.left, "underlying read released by caller data")
}

func TestReplOwnerNaturalEOFAndNoOwner(t *testing.T) {
	if CloseRepl() {
		t.Fatal("unexpected active REPL before publication")
	}
	f := newReplPipeFixture(t)
	f.release()
	done := make(chan error, 1)
	go func() {
		_, err := f.owner.instance.Readline()
		f.owner.close()
		activeRepl.CompareAndSwap(f.owner, nil)
		done <- err
	}()
	awaitReplEvent(t, f.input.entered, "underlying pipe read")
	f.writer.Close()
	if err := awaitReplEvent(t, done, "natural EOF and owner cleanup"); err != io.EOF {
		t.Fatalf("readline error = %v; want EOF", err)
	}
	awaitReplEvent(t, f.input.left, "underlying read EOF")
	if activeRepl.Load() != nil || CloseRepl() {
		t.Fatal("completed REPL still published an owner")
	}
}
