//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package phpbridge

// #include "bridge.h"
import "C"

import "io"
import "fmt"
import "time"
import "bytes"
import _ "embed"
import "unsafe"
import "context"
import "os/exec"
import "strconv"
import "strings"
import "runtime/cgo"
import "encoding/json"
import "encoding/binary"
import "github.com/dunglas/frankenphp"

//go:embed imap_worker.php
var imapWorkerSource string

//go:embed imap_frontend.php
var imapFrontendSource string

var imapExecutable string
var imapMemoryLimit int64

const imapProtocolLimit = 64 << 20
const imapCallTimeout = 60 * time.Second

func registerIMAP(binaryPath string, memoryLimit int64) error {
	if binaryPath == "" {
		return nil
	}
	resolved, err := exec.LookPath(binaryPath)
	if err != nil {
		return fmt.Errorf("PHPIMAPBinary: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	probe := exec.CommandContext(ctx, resolved, "-d", "display_errors=stderr", "-r", `echo json_encode(["zts"=>PHP_ZTS,"functions"=>get_extension_funcs("imap"),"constants"=>get_defined_constants(true)["imap"]??[]]);`)
	data, err := probe.Output()
	if err != nil {
		return fmt.Errorf("PHPIMAPBinary probe: %w", err)
	}
	var info struct {
		ZTS       int              `json:"zts"`
		Functions []string         `json:"functions"`
		Constants map[string]int64 `json:"constants"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return fmt.Errorf("PHPIMAPBinary probe: %w", err)
	}
	if info.ZTS != 0 || len(info.Functions) == 0 {
		return fmt.Errorf("PHPIMAPBinary must be an NTS PHP CLI with the native IMAP extension")
	}
	for _, name := range info.Functions {
		if !strings.HasPrefix(name, "imap_") || strings.ContainsAny(name, "\n\x00") {
			return fmt.Errorf("invalid IMAP function name")
		}
	}
	constants, _ := json.Marshal(info.Constants)
	// This is our adapter source, not application code. It is installed in each
	// request so PHP's normal shutdown also destroys all frontend handles.
	source := strings.TrimPrefix(imapFrontendSource, "<?php") + "\nnamespace { foreach (json_decode(" + strconv.Quote(string(constants)) + ", true) as $name=>$value) { if (!defined($name)) define($name,$value); } unset($name,$value); }"
	csource, names := C.CString(source), C.CString(strings.Join(info.Functions, "\n"))
	defer C.free(unsafe.Pointer(csource))
	defer C.free(unsafe.Pointer(names))
	C.memcp_imap_configure(csource, names)
	imapExecutable, imapMemoryLimit = resolved, memoryLimit
	frankenphp.RegisterExtension(C.memcp_imap_module())
	return nil
}

type imapWorker struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output io.ReadCloser
	done   chan error
	closed bool
}

func startIMAPWorker() (*imapWorker, error) {
	if imapExecutable == "" {
		return nil, fmt.Errorf("isolated IMAP is not configured")
	}
	cmd := exec.Command(imapExecutable, "-d", "display_errors=stderr", "-d", "log_errors=0", "-d", "memory_limit="+strconv.FormatInt(imapMemoryLimit, 10), "-r", strings.TrimPrefix(imapWorkerSource, "<?php"))
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, err
	}
	worker := &imapWorker{cmd: cmd, input: input, output: output, done: make(chan error, 1)}
	go func() { worker.done <- cmd.Wait() }()
	return worker, nil
}
func (w *imapWorker) close() {
	if w.closed {
		return
	}
	w.closed = true
	w.input.Close()
	select {
	case <-w.done:
	case <-time.After(200 * time.Millisecond):
		w.cmd.Process.Kill()
		<-w.done
	}
	w.output.Close()
}
func (w *imapWorker) call(request []byte, timeout time.Duration) ([]byte, error) {
	if w.closed {
		return nil, fmt.Errorf("IMAP helper has stopped")
	}
	if len(request) > imapProtocolLimit {
		return nil, fmt.Errorf("IMAP request exceeds 64 MiB")
	}
	type reply struct {
		data []byte
		err  error
	}
	finished := make(chan reply, 1)
	go func() {
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], uint32(len(request)))
		_, err := io.Copy(w.input, io.MultiReader(bytes.NewReader(header[:]), bytes.NewReader(request)))
		if err != nil {
			finished <- reply{err: err}
			return
		}
		if _, err = io.ReadFull(w.output, header[:]); err != nil {
			finished <- reply{err: err}
			return
		}
		n := binary.BigEndian.Uint32(header[:])
		if n > imapProtocolLimit {
			finished <- reply{err: fmt.Errorf("IMAP response exceeds 64 MiB")}
			return
		}
		data := make([]byte, n)
		_, err = io.ReadFull(w.output, data)
		finished <- reply{data, err}
	}()
	select {
	case result := <-finished:
		if result.err != nil {
			w.close()
		}
		return result.data, result.err
	case <-time.After(timeout):
		w.close()
		<-finished
		return nil, fmt.Errorf("IMAP helper call exceeded %s", timeout)
	}
}

//export memcp_imap_call
func memcp_imap_call(handle C.uintptr_t, data *C.char, length C.size_t) (result C.memcp_text) {
	defer func() {
		if err := recover(); err != nil {
			result.error = C.CString(fmt.Sprint(err))
		}
	}()
	if length > imapProtocolLimit {
		result.error = C.CString("IMAP request exceeds 64 MiB")
		return
	}
	r := cgo.Handle(handle).Value().(*textRequest)
	var err error
	if r.imap == nil {
		r.imap, err = startIMAPWorker()
		if err != nil {
			result.error = C.CString(err.Error())
			return
		}
	}
	out, err := r.imap.call(C.GoBytes(unsafe.Pointer(data), C.int(length)), imapCallTimeout)
	if err != nil {
		result.error = C.CString(err.Error())
		return
	}
	result.data = (*C.char)(C.CBytes(out))
	result.length = C.size_t(len(out))
	return
}
