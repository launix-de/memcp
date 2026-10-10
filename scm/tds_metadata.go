/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
)

func tdsMetadataParameters(arguments []tdsRPCParameter) Scmer {
	parameters := make([]Scmer, 0, 2*len(arguments))
	seen := make(map[string]bool, len(arguments))
	for i, argument := range arguments {
		if argument.flags != 0 {
			panic("metadata RPC output parameters are unsupported")
		}
		name := strconv.Itoa(i)
		if argument.name != "" {
			name = argument.name
		}
		if seen[name] {
			panic("duplicate metadata RPC parameter")
		}
		seen[name] = true
		parameters = append(parameters, NewString(name), argument.value)
	}
	return NewSlice(parameters)
}

func (c *tdsConnection) metadataRPC(rpc tdsRPC, request *tdsRequest, response *tdsResponse) error {
	if c.server.metadata.IsNil() {
		panic("TDS metadata procedure callback is unavailable")
	}
	if c.server.resolveMetadata.IsNil() {
		panic("TDS metadata name callback is unavailable")
	}
	resolved := Apply(c.server.resolveMetadata, NewString(rpc.metadata), NewString(c.database))
	if !resolved.IsSlice() || len(resolved.Slice()) != 2 || !resolved.Slice()[0].IsString() || !resolved.Slice()[1].IsString() {
		panic("invalid TDS metadata procedure carrier")
	}
	database, procedure := resolved.Slice()[0].String(), resolved.Slice()[1].String()
	if database != c.database {
		panic("metadata callback resolved a different connection database")
	}
	if procedure == "" {
		panic("empty metadata procedure")
	}

	parameters := tdsMetadataParameters(rpc.arguments)
	atomic.AddInt64(&TotalHTTPRequests, 1)
	seq := c.ss.BeginQuery("Query", rpc.metadata)
	ctx, cancel := context.WithCancel(request.ctx)
	c.ss.SetCancel(seq, cancel)
	c.ss.SetQueryContext(seq, ctx)
	c.seq.Store(seq)
	defer cancel()
	defer c.seq.Store(0)
	defer c.ss.EndQuery(seq, "Sleep", "")
	sink := &tdsResult{w: response}
	Apply(c.server.metadata, NewString(database), NewString(procedure), parameters,
		NewFunc(sink.row), NewFunc(sink.fields), c.session, NewAny(c.ss), NewInt(int64(seq)))
	if err := request.ctx.Err(); err != nil {
		panic(err)
	}
	sink.publish()
	var tokens bytes.Buffer
	if request.finish() {
		tdsDone(&tokens, 0xfd, 0x20, 0)
	} else {
		tdsDone(&tokens, 0xff, 0x11, sink.count)
		tokens.WriteByte(0x79)
		tdsU32(&tokens, 0)
		tdsDone(&tokens, 0xfe, 0, 0)
	}
	if _, err := response.Write(tokens.Bytes()); err != nil {
		return fmt.Errorf("TDS metadata response: %w", err)
	}
	return response.flush(true)
}
