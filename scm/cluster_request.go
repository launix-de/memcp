/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "bytes"
import "context"
import "encoding/binary"

// ClusterRequestHandler is the sole channel receiving contract. Sending a result
// is a separate channel delivery chosen by the payload protocol.
type ClusterRequestHandler interface {
	HandleClusterRequest(context.Context, []byte)
}

type ClusterRequestHandlerFunc func(context.Context, []byte)

func (receive ClusterRequestHandlerFunc) HandleClusterRequest(ctx context.Context, payload []byte) {
	receive(ctx, payload)
}

// The object protocol owns these operation bytes; they are not transport fields.
// Append only: existing operation bytes retain their meaning. Catalog messages
// are native transaction-construction operations, interpreted by their owner.
var clusterOperations = [...]string{"", "view", "call", "batch", "enlist", "next", "ack", "finish", "prepare", "commit", "rollback", "savepoint", "rollback-savepoint", "receive",
	"catalog_create", "catalog_position", "catalog_retire", "catalog_key", "catalog_column"}

// EncodeClusterRequest writes fixed operation parameters in binary and at most
// one serialized value for dynamic arguments. Local native invocations do not
// use this codec or turn their borrowed Go input into serialized data.
func EncodeClusterRequest(request ClusterRequest, bindings ...*SerializationContext) (payload []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			payload = nil
			err = fmt.Errorf("cluster request encoding: %v", failure)
		}
	}()
	op := byte(0)
	for i, name := range clusterOperations {
		if name == request.Method {
			op = byte(i)
			break
		}
	}
	if op == 0 {
		return nil, fmt.Errorf("unsupported cluster operation %q", request.Method)
	}
	payload = []byte{op}
	args := request.Arguments
	switch request.Method {
	case "ack":
		if len(args) != 2 || !args[0].IsInt() || !args[1].IsInt() {
			return nil, fmt.Errorf("invalid ordered ACK")
		}
		payload = binary.BigEndian.AppendUint64(payload, uint64(args[0].Int()))
		payload = binary.BigEndian.AppendUint64(payload, uint64(args[1].Int()))
		args = nil
	case "enlist":
		if len(args) != 2 || !args[1].IsString() {
			return nil, fmt.Errorf("invalid enlist capability")
		}
		capability := args[1].String()
		if len(capability) > 65535 {
			return nil, fmt.Errorf("oversized enlist capability")
		}
		payload = binary.BigEndian.AppendUint16(payload, uint16(len(capability)))
		payload = append(payload, capability...)
		args = args[:1]
	case "prepare", "commit", "rollback", "savepoint", "rollback-savepoint":
		if len(args) == 0 || !args[0].IsString() {
			return nil, fmt.Errorf("missing transaction decision capability")
		}
		capability := args[0].String()
		if len(capability) > 65535 {
			return nil, fmt.Errorf("oversized decision capability")
		}
		payload = binary.BigEndian.AppendUint16(payload, uint16(len(capability)))
		payload = append(payload, capability...)
		args = args[1:]
	}
	if len(args) == 0 {
		return payload, nil
	}
	if len(bindings) == 0 || bindings[0] == nil {
		bindings = []*SerializationContext{new(SerializationContext)}
	}
	var tail bytes.Buffer
	Serialize(&tail, NewSlice([]Scmer{NewSymbol("quote"), NewSlice(args)}), &Globalenv, bindings...)
	return append(payload, tail.Bytes()...), nil
}

func DecodeClusterRequest(payload []byte, decode ProcedureDecode) (request ClusterRequest, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("cluster request decoding: %v", failure)
		}
	}()
	if len(payload) == 0 || int(payload[0]) >= len(clusterOperations) || payload[0] == 0 {
		return request, fmt.Errorf("invalid cluster operation")
	}
	request.Method, payload = clusterOperations[payload[0]], payload[1:]
	switch request.Method {
	case "ack":
		if len(payload) != 16 {
			return request, fmt.Errorf("invalid ordered ACK size")
		}
		request.Arguments = []Scmer{NewInt(int64(binary.BigEndian.Uint64(payload))), NewInt(int64(binary.BigEndian.Uint64(payload[8:])))}
		return request, nil
	case "enlist":
		if len(payload) < 2 {
			return request, fmt.Errorf("truncated enlist capability")
		}
		n := int(binary.BigEndian.Uint16(payload))
		payload = payload[2:]
		if n > len(payload) {
			return request, fmt.Errorf("truncated enlist capability")
		}
		request.Arguments = []Scmer{NewString(string(payload[:n]))}
		payload = payload[n:]
	case "prepare", "commit", "rollback", "savepoint", "rollback-savepoint":
		if len(payload) < 2 {
			return request, fmt.Errorf("truncated decision capability")
		}
		n := int(binary.BigEndian.Uint16(payload))
		payload = payload[2:]
		if n > len(payload) {
			return request, fmt.Errorf("truncated decision capability")
		}
		request.Arguments = []Scmer{NewString(string(payload[:n]))}
		payload = payload[n:]
	}
	if len(payload) != 0 {
		tokens := tokenize("cluster arguments", string(payload), false)
		expression := readFrom(&tokens)
		if len(tokens) != 0 {
			return request, fmt.Errorf("cluster arguments contain trailing data")
		}
		if decode.Prepare != nil {
			expression, err = decode.Prepare(expression)
			if err != nil {
				return request, err
			}
		}
		expression, err = ResolveSerializedObjects(expression, decode.Resolve, decode.ResolveHandle)
		if err != nil {
			return request, err
		}
		value := Eval(expression, &Globalenv)
		if !value.IsSlice() {
			return request, fmt.Errorf("cluster arguments must be a list")
		}
		for _, argument := range value.Slice() {
			// The former per-argument codec JITted a decoded procedure. Preserve
			// that behavior when all dynamic arguments share one serialized tail.
			if argument.IsProc() {
				argument = CompileJIT(argument, true)
			}
			request.Arguments = append(request.Arguments, argument)
		}
	}
	if request.Method == "enlist" {
		if len(request.Arguments) != 2 {
			return request, fmt.Errorf("enlist requires one participant")
		}
		request.Arguments[0], request.Arguments[1] = request.Arguments[1], request.Arguments[0]
	}
	return request, nil
}

type clusterDeliveryKey struct{}
type clusterValueDelivery struct {
	decode ProcedureDecode
	reply  func(Scmer, error)
}

// WithClusterValueDelivery binds the native invocation's reconstruction helpers
// and its payload-defined result sink. It neither registers nor waits for RPCs.
func WithClusterValueDelivery(ctx context.Context, decode ProcedureDecode, reply func(Scmer, error)) context.Context {
	return context.WithValue(ctx, clusterDeliveryKey{}, clusterValueDelivery{decode, reply})
}

// ReceiveClusterValue is a codec helper for the object-value protocol. Binary
// channels need neither this helper nor a serializable runtime object.
func ReceiveClusterValue(ctx context.Context, payload []byte, execute func(ClusterRequest) (any, error)) {
	delivery, ok := ctx.Value(clusterDeliveryKey{}).(clusterValueDelivery)
	if !ok {
		panic("cluster value delivery has no result sink")
	}
	request, err := DecodeClusterRequest(payload, delivery.decode)
	if err != nil {
		delivery.reply(NewNil(), err)
		return
	}
	result, err := execute(request)
	if err != nil {
		delivery.reply(NewNil(), err)
		return
	}
	if result == nil {
		delivery.reply(NewNil(), nil)
		return
	}
	value, ok := result.(Scmer)
	if !ok {
		delivery.reply(NewNil(), fmt.Errorf("cluster response is not a runtime value"))
		return
	}
	delivery.reply(value, nil)
}
