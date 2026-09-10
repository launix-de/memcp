//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package phpbridge

import "os"
import "time"
import "os/exec"
import "testing"

func TestIMAPTimeoutReapsProcess(t *testing.T) {
	binary := os.Getenv("MEMCP_TEST_IMAP_BINARY")
	if binary == "" {
		t.Skip("MEMCP_TEST_IMAP_BINARY is not configured")
	}
	// The helper deliberately never replies. No remote service is involved.
	cmd := exec.Command(binary, "-r", "usleep(5000000);")
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		t.Fatal(err)
	}
	worker := &imapWorker{cmd: cmd, input: input, output: output, done: make(chan error, 1)}
	go func() { worker.done <- cmd.Wait() }()
	defer worker.close()
	start := time.Now()
	if _, err = worker.call([]byte("{}"), 20*time.Millisecond); err == nil {
		t.Fatal("missing timeout")
	}
	if time.Since(start) > time.Second || cmd.ProcessState == nil || !worker.closed {
		t.Fatal("timed-out helper was not reaped promptly")
	}
	if _, err = worker.call([]byte("{}"), time.Second); err == nil {
		t.Fatal("reused failed helper")
	}
}
