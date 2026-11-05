/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "io"
import "fmt"
import "math"
import "sync"
import "bytes"
import "reflect"
import "strings"
import "compress/gzip"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

const mosiPacketBytes = 64 * 1024

const mosiHTTPPath = "/cluster"
const mosiSubprotocol = "memcp.channels.v8"
const mosiMaxMessageBytes = 256 * 1024
const maxNodeRecordBytes = 32 * 1024 * 1024

// Every application frame contains only a channel and its opaque payload.
// Version negotiation belongs to the WebSocket subprotocol, not every message.
func encodeMosiMessage(m mosiMessage) ([]byte, error) {
	if len(m.Channel) == 0 || len(m.Channel) > 255 {
		return nil, fmt.Errorf("invalid node channel length")
	}
	if 1+len(m.Channel)+len(m.Payload) > mosiMaxMessageBytes {
		return nil, scm.ErrClusterRequestTooLarge
	}
	b := make([]byte, 1+len(m.Channel)+len(m.Payload))
	b[0] = byte(len(m.Channel))
	copy(b[1:], m.Channel)
	copy(b[1+len(m.Channel):], m.Payload)
	return b, nil
}
func decodeMosiMessage(frame []byte) (mosiMessage, error) {
	if len(frame) == 0 || len(frame) > mosiMaxMessageBytes || frame[0] == 0 || int(frame[0])+1 > len(frame) {
		return mosiMessage{}, fmt.Errorf("invalid node channel frame")
	}
	n := int(frame[0])
	return mosiMessage{Channel: string(frame[1 : 1+n]), Payload: frame[1+n:]}, nil
}

// MOSI's action selects its exact parameter layout. No map keys, generic
// metadata record or fields belonging to other actions cross the wire.
var mosiActions = [...]string{"", "read", "write", "recall", "invalidate", "ack", "failure", "grant", "begin", "object", "chunk", "object_end", "end", "reference"}

type mosiWriter struct{ data []byte }

func (w *mosiWriter) text(s string)   { w.data = appendNodeBlob(w.data, []byte(s)) }
func (w *mosiWriter) number(n uint64) { w.data = binary.AppendUvarint(w.data, n) }
func (w *mosiWriter) resource(r mosiResource) {
	switch {
	case r.Object != "":
		w.data = append(w.data, 2)
		w.text(r.Object)
	case r.Shard != "":
		w.data = append(w.data, 1)
		w.text(r.Database)
		w.text(r.Table)
		w.text(r.Shard)
	default:
		w.data = append(w.data, 0)
		w.text(r.Database)
	}
}
func (r *nodeReader) octet() byte {
	if len(r.data) == 0 {
		panic("truncated action")
	}
	v := r.data[0]
	r.data = r.data[1:]
	return v
}
func (r *nodeReader) resource() mosiResource {
	switch r.octet() {
	case 0:
		return mosiResource{Database: string(r.blob())}
	case 1:
		return mosiResource{Database: string(r.blob()), Table: string(r.blob()), Shard: string(r.blob())}
	case 2:
		return mosiResource{Object: string(r.blob())}
	default:
		panic("invalid resource kind")
	}
}
func (w *mosiWriter) routes(routes []shardForward) {
	w.number(uint64(len(routes)))
	for _, route := range routes {
		w.resource(route.From)
		w.resource(route.To)
	}
}
func (r *nodeReader) routes() []shardForward {
	n := r.uint()
	if n > uint64(len(r.data))/4 {
		panic("truncated routes")
	}
	if n == 0 {
		return nil
	}
	routes := make([]shardForward, int(n))
	for i := range routes {
		routes[i] = shardForward{From: r.resource(), To: r.resource()}
	}
	return routes
}
func (w *mosiWriter) readers(readers map[string]bool) {
	w.number(uint64(len(readers)))
	for peer, live := range readers {
		w.text(peer)
		if live {
			w.data = append(w.data, 1)
		} else {
			w.data = append(w.data, 0)
		}
	}
}
func (r *nodeReader) readers() map[string]bool {
	n := r.uint()
	if n > uint64(len(r.data))/2 {
		panic("truncated readers")
	}
	if n == 0 {
		return nil
	}
	readers := make(map[string]bool, int(n))
	for range n {
		peer := string(r.blob())
		live := r.octet()
		if live > 1 {
			panic("invalid reader")
		}
		readers[peer] = live == 1
	}
	return readers
}
func encodeMosiPacket(p mosiPacket) (data []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			data = nil
			err = fmt.Errorf("invalid MOSI payload: %v", failure)
		}
	}()
	if len(p.Payload) > 0 && (p.Kind != "chunk" || len(p.Payload) > mosiPacketBytes+1024) {
		return nil, fmt.Errorf("invalid MOSI binary payload")
	}
	action := 0
	for i, name := range mosiActions {
		if name == p.Kind {
			action = i
			break
		}
	}
	if action == 0 {
		return nil, fmt.Errorf("unknown MOSI action")
	}
	w := mosiWriter{data: make([]byte, 1, 256)}
	w.data[0] = byte(action)
	// These flags represent routing, not unrelated optional action parameters.
	flags := byte(0)
	if p.From != "" {
		flags |= 1
	}
	if p.To != "" {
		flags |= 2
	}
	w.data = append(w.data, flags)
	if flags&1 != 0 {
		w.text(p.From)
	}
	if flags&2 != 0 {
		w.text(p.To)
	}
	w.text(p.ID)
	w.resource(p.Resource)
	switch p.Kind {
	case "read", "write":
		w.number(p.Sequence)
		w.text(p.Boot)
		w.text(p.User)
		w.text(p.Hash)
		if p.TryRead {
			w.data = append(w.data, 1)
		} else {
			w.data = append(w.data, 0)
		}
		w.routes(p.Forwards)
	case "recall":
		w.number(p.Epoch)
		w.text(p.State)
		w.text(p.Owner)
		w.readers(p.Readers)
	case "invalidate":
		w.number(p.Epoch)
		w.routes(p.Forwards)
	case "ack":
		w.number(p.Epoch)
		w.text(p.Content)
	case "failure":
		w.number(p.Epoch)
		w.text(p.Content)
		w.text(p.State)
		w.text(p.Error)
	case "grant":
		w.number(p.Epoch)
		w.text(p.State)
		w.text(p.Content)
		w.text(p.Owner)
		w.readers(p.Readers)
	case "begin":
		w.number(p.Epoch)
		w.text(p.Content)
		w.text(p.User)
		w.text(p.Hash)
		w.routes(p.Forwards)
	case "object":
		w.number(p.Epoch)
		w.text(p.Content)
		w.text(p.ObjectKind)
		w.text(p.Object)
	case "chunk":
		w.number(p.Epoch)
		w.text(p.Content)
		w.text(p.ObjectKind)
		w.text(p.Object)
		w.data = append(w.data, p.Payload...)
	case "object_end":
		w.number(p.Epoch)
		w.text(p.Content)
		w.text(p.ObjectKind)
		w.text(p.Object)
	case "end":
		w.number(p.Epoch)
		w.text(p.Content)
	case "reference":
		w.number(p.Epoch)
		w.text(p.State)
		w.text(p.Content)
		w.text(p.Owner)
		w.readers(p.Readers)
		w.routes(p.Forwards)
		w.text(p.Object)
		if p.Capture == nil {
			w.data = append(w.data, 0)
		} else {
			w.data = appendReferences(w.data, []scm.ClusterReference{*p.Capture})
		}
	}
	return w.data, nil
}
func decodeMosiPacket(data []byte) (packet *mosiPacket, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			packet = nil
			err = fmt.Errorf("invalid MOSI payload: %v", failure)
		}
	}()
	r := nodeReader{data}
	action := int(r.octet())
	if action == 0 || action >= len(mosiActions) {
		return nil, fmt.Errorf("unknown MOSI action")
	}
	p := new(mosiPacket)
	p.Kind = mosiActions[action]
	flags := r.octet()
	if flags&^3 != 0 {
		return nil, fmt.Errorf("invalid MOSI route flags")
	}
	if flags&1 != 0 {
		p.From = string(r.blob())
	}
	if flags&2 != 0 {
		p.To = string(r.blob())
	}
	p.ID = string(r.blob())
	p.Resource = r.resource()
	switch p.Kind {
	case "read", "write":
		p.Sequence = r.uint()
		p.Boot = string(r.blob())
		p.User = string(r.blob())
		p.Hash = string(r.blob())
		flag := r.octet()
		if flag > 1 {
			panic("invalid read flag")
		}
		p.TryRead = flag == 1
		p.Forwards = r.routes()
	case "recall":
		p.Epoch = r.uint()
		p.State = string(r.blob())
		p.Owner = string(r.blob())
		p.Readers = r.readers()
	case "invalidate":
		p.Epoch = r.uint()
		p.Forwards = r.routes()
	case "ack":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
	case "failure":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
		p.State = string(r.blob())
		p.Error = string(r.blob())
	case "grant":
		p.Epoch = r.uint()
		p.State = string(r.blob())
		p.Content = string(r.blob())
		p.Owner = string(r.blob())
		p.Readers = r.readers()
	case "begin":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
		p.User = string(r.blob())
		p.Hash = string(r.blob())
		p.Forwards = r.routes()
	case "object":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
		p.ObjectKind = string(r.blob())
		p.Object = string(r.blob())
	case "chunk":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
		p.ObjectKind = string(r.blob())
		p.Object = string(r.blob())
		if len(r.data) > mosiPacketBytes+1024 {
			panic("oversized MOSI chunk")
		}
		if len(r.data) > 0 {
			p.Payload = r.data
		}
		r.data = nil
	case "object_end":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
		p.ObjectKind = string(r.blob())
		p.Object = string(r.blob())
	case "end":
		p.Epoch = r.uint()
		p.Content = string(r.blob())
	case "reference":
		p.Epoch = r.uint()
		p.State = string(r.blob())
		p.Content = string(r.blob())
		p.Owner = string(r.blob())
		p.Readers = r.readers()
		p.Forwards = r.routes()
		p.Object = string(r.blob())
		refs := r.references()
		if len(refs) > 1 {
			panic("invalid transfer capture")
		}
		if len(refs) == 1 {
			p.Capture = &refs[0]
		}
	}
	if len(r.data) != 0 {
		return nil, fmt.Errorf("trailing MOSI parameters")
	}
	return p, nil
}

// Node control metadata uses a bounded binary value stream. Scheme values are
// deliberately still reconstruction text; this is not a binary Scmer format.
// Structs use their existing field contract, so partial control DTOs remain valid.
const (
	nodeNil byte = iota
	nodeFalse
	nodeTrue
	nodeInt
	nodeUint
	nodeFloat
	nodeString
	nodeBytes
	nodeList
	nodeMap
	nodeScheme
	nodePairs
)

type nodeField struct {
	name      string
	index     []int
	omitEmpty bool
}
type nodeLayout struct {
	fields []nodeField
	byName map[string][]int
}

var nodeLayouts sync.Map
var nodeScmerType = reflect.TypeOf(scm.Scmer{})

func metadataLayout(t reflect.Type) *nodeLayout {
	if v, ok := nodeLayouts.Load(t); ok {
		return v.(*nodeLayout)
	}
	layout := &nodeLayout{byName: make(map[string][]int)}
	var add func(reflect.Type, []int)
	add = func(current reflect.Type, prefix []int) {
		for i := 0; i < current.NumField(); i++ {
			f := current.Field(i)
			index := append(append([]int(nil), prefix...), i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
				add(f.Type, index)
				continue
			}
			if !f.IsExported() {
				continue
			}
			if name == "" {
				name = f.Name
			}
			layout.fields = append(layout.fields, nodeField{name, index, strings.Contains(f.Tag.Get("json"), ",omitempty")})
			layout.byName[name] = index
			layout.byName[strings.ToLower(name)] = index
		}
	}
	add(t, nil)
	actual, _ := nodeLayouts.LoadOrStore(t, layout)
	return actual.(*nodeLayout)
}
func marshalNode(value any) (data []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			data = nil
			err = fmt.Errorf("invalid typed payload: %v", failure)
		}
		if err == nil && len(data) > maxNodeRecordBytes {
			data = nil
			err = scm.ErrClusterRequestTooLarge
		}
	}()
	switch v := value.(type) {
	case scm.ClusterReference:
		return appendReferences(nil, []scm.ClusterReference{v}), nil
	case []scm.ClusterReference:
		return appendReferences(nil, v), nil
	case replicaAdminResult:
		return freezeScanValue(scm.NewSlice([]scm.Scmer{v.Value, scm.NewSlice(v.Rows), scm.NewSlice(v.Fields)}))

	case replicaRecord:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.record(v)
		return w.data, nil
	case []replicaRecord:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.records(v)
		return w.data, nil
	case clusterSchemaLocations:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.locations(v)
		return w.data, nil
	case replicaRecovery:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.text(v.Boot)
		w.records(v.Resources)
		return w.data, nil
	case replicaAuth:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.text(v.Hash)
		if v.Admin {
			w.data = append(w.data, 1)
		} else {
			w.data = append(w.data, 0)
		}
		w.readers(v.Databases)
		return w.data, nil
	case scanChallenge:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.number(uint64(v.Version))
		flags := byte(0)
		if v.Boot != "" {
			flags |= 1
		}
		if v.Node != "" {
			flags |= 2
		}
		if v.Nonce != "" {
			flags |= 4
		}
		if v.Proof != "" {
			flags |= 8
		}
		if v.Return != "" {
			flags |= 16
		}
		w.data = append(w.data, flags)
		if flags&1 != 0 {
			w.text(v.Boot)
		}
		if flags&2 != 0 {
			w.text(v.Node)
		}
		if flags&4 != 0 {
			w.text(v.Nonce)
		}
		if flags&8 != 0 {
			w.text(v.Proof)
		}
		if flags&16 != 0 {
			w.text(v.Return)
		}
		return w.data, nil

	case remoteScanDescription:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.description(v)
		return w.data, nil
	case remoteInsertTask:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.insert(v)
		return w.data, nil
	case remoteInsertResult:
		w := mosiWriter{data: make([]byte, 0, 128)}
		flags := byte(0)
		if v.Rejected {
			flags |= 1
		}
		if v.FirstID != nil {
			flags |= 2
		}
		w.data = append(w.data, flags)
		w.integer(int64(v.Count))
		if v.FirstID != nil {
			w.integer(*v.FirstID)
		}
		return w.data, nil
	case mosiResource:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.resource(v)
		return w.data, nil
	case clusterHello:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.hello(v)
		return w.data, nil
	case clusterIDRequest:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.text(v.Database)
		w.text(v.Table)
		w.number(v.Explicit)
		w.number(v.Count)
		return w.data, nil
	case clusterIDReply:
		return binary.AppendUvarint(nil, v.Base), nil
	case objectViewDeclaration:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.text(v.URI)
		w.data = append(w.data, v.View...)
		return w.data, nil

	case remoteScanTask:
		return encodeScanTask(v), nil
	case remoteScanResult:
		return encodeScanResult(v), nil
	case remoteScanDescribe:
		w := mosiWriter{data: make([]byte, 0, 128)}
		w.number(uint64(v.Version))
		w.text(v.Database)
		w.text(v.Table)
		return w.data, nil
	case clusterObjectReply:
		w := mosiWriter{data: make([]byte, 0, 128)}
		if v.Raw != nil {
			w.data = append(w.data, 1)
		} else {
			w.data = append(w.data, 0)
		}
		if v.Raw != nil {
			w.data = append(w.data, v.Raw...)
		} else {
			w.data = append(w.data, v.Value...)
		}
		return w.data, nil
	}

	if completion, ok := value.(scanCompletion); ok {
		return encodeScanCompletion(completion)
	}
	if packet, ok := value.(mosiPacket); ok {
		return encodeMosiPacket(packet)
	}
	defer func() {
		if failure := recover(); failure != nil {
			data = nil
			err = fmt.Errorf("node metadata encoding: %v", failure)
		}
	}()
	if encoded, ok := value.(encodedReply); ok {
		return encoded.data, nil
	}
	data = appendNode(make([]byte, 0, 256), reflect.ValueOf(value), 0)
	if len(data) > maxNodeRecordBytes {
		return nil, scm.ErrClusterRequestTooLarge
	}
	return data, nil
}
func appendNode(b []byte, v reflect.Value, depth int) []byte {
	if depth > 128 {
		panic("metadata nesting limit")
	}
	if !v.IsValid() {
		return append(b, nodeNil)
	}
	if v.Type() == nodeScmerType {
		source, err := freezeScanValue(v.Interface().(scm.Scmer))
		if err != nil {
			panic(err)
		}
		return appendNodeBlob(append(b, nodeScheme), source)
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return append(b, nodeNil)
		}
		return appendNode(b, v.Elem(), depth+1)
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return append(b, nodeTrue)
		}
		return append(b, nodeFalse)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return binary.AppendVarint(append(b, nodeInt), v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return binary.AppendUvarint(append(b, nodeUint), v.Uint())
	case reflect.Float32, reflect.Float64:
		return binary.LittleEndian.AppendUint64(append(b, nodeFloat), math.Float64bits(v.Float()))
	case reflect.String:
		return appendNodeBlob(append(b, nodeString), []byte(v.String()))
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return append(b, nodeNil)
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return appendNodeBlob(append(b, nodeBytes), v.Bytes())
		}
		b = binary.AppendUvarint(append(b, nodeList), uint64(v.Len()))
		for i := 0; i < v.Len(); i++ {
			b = appendNode(b, v.Index(i), depth+1)
		}
		return b
	case reflect.Struct:
		layout := metadataLayout(v.Type())
		count := 0
		for _, field := range layout.fields {
			if !field.omitEmpty || !v.FieldByIndex(field.index).IsZero() {
				count++
			}
		}
		b = binary.AppendUvarint(append(b, nodeMap), uint64(count))
		for _, field := range layout.fields {
			if field.omitEmpty && v.FieldByIndex(field.index).IsZero() {
				continue
			}
			b = appendNodeBlob(b, []byte(field.name))
			b = appendNode(b, v.FieldByIndex(field.index), depth+1)
		}
		return b
	case reflect.Map:
		if v.IsNil() {
			return append(b, nodeNil)
		}
		if v.Type().Key().Kind() != reflect.String {
			b = binary.AppendUvarint(append(b, nodePairs), uint64(v.Len()))
			iter := v.MapRange()
			for iter.Next() {
				b = appendNode(b, iter.Key(), depth+1)
				b = appendNode(b, iter.Value(), depth+1)
			}
			return b
		}
		b = binary.AppendUvarint(append(b, nodeMap), uint64(v.Len()))
		iter := v.MapRange()
		for iter.Next() {
			b = appendNodeBlob(b, []byte(iter.Key().String()))
			b = appendNode(b, iter.Value(), depth+1)
		}
		return b
	}
	panic("unsupported node metadata type " + v.Type().String())
}
func appendNodeBlob(b, data []byte) []byte {
	b = binary.AppendUvarint(b, uint64(len(data)))
	return append(b, data...)
}

type nodeReader struct{ data []byte }

func (r *nodeReader) uint() uint64 {
	v, n := binary.Uvarint(r.data)
	if n <= 0 {
		panic("truncated node integer")
	}
	r.data = r.data[n:]
	return v
}
func (r *nodeReader) blob() []byte {
	n := r.uint()
	if n > uint64(len(r.data)) {
		panic("truncated node bytes")
	}
	v := r.data[:int(n)]
	r.data = r.data[int(n):]
	return v
}
func unmarshalNode(data []byte, target any) (err error) {
	if len(data) > maxNodeRecordBytes {
		return scm.ErrClusterRequestTooLarge
	}
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("invalid typed payload: %v", failure)
		}
	}()
	raw := nodeReader{data}
	typed := true
	switch p := target.(type) {
	case *scm.ClusterReference:
		refs := raw.references()
		if len(refs) != 1 {
			panic("expected one native reference")
		}
		*p = refs[0]
	case *[]scm.ClusterReference:
		*p = raw.references()
	case *replicaAdminResult:
		value, err := schemeValue(data).reconstruct()
		if err != nil {
			return err
		}
		if !value.IsSlice() || len(value.Slice()) != 3 {
			return fmt.Errorf("invalid admin result")
		}
		items := value.Slice()
		if !items[1].IsSlice() || !items[2].IsSlice() {
			return fmt.Errorf("invalid admin rows")
		}
		*p = replicaAdminResult{Value: items[0], Rows: items[1].Slice(), Fields: items[2].Slice()}
		raw.data = nil

	case *replicaRecord:
		*p = raw.record()
	case *[]replicaRecord:
		*p = raw.records()
	case *clusterSchemaLocations:
		*p = raw.locations()
	case *replicaRecovery:
		*p = replicaRecovery{Boot: string(raw.blob()), Resources: raw.records()}
	case *replicaAuth:
		p.Hash = string(raw.blob())
		flag := raw.octet()
		if flag > 1 {
			panic("invalid administrator flag")
		}
		p.Admin = flag == 1
		p.Databases = raw.readers()
	case *scanChallenge:
		*p = scanChallenge{Version: int(raw.uint())}
		flags := raw.octet()
		if flags&^31 != 0 {
			panic("invalid authentication flags")
		}
		if flags&1 != 0 {
			p.Boot = string(raw.blob())
		}
		if flags&2 != 0 {
			p.Node = string(raw.blob())
		}
		if flags&4 != 0 {
			p.Nonce = string(raw.blob())
		}
		if flags&8 != 0 {
			p.Proof = string(raw.blob())
		}
		if flags&16 != 0 {
			p.Return = string(raw.blob())
		}

	case *remoteScanDescription:
		*p = raw.description()
	case *remoteInsertTask:
		*p = raw.insert()
	case *remoteInsertResult:
		flags := raw.octet()
		if flags&^3 != 0 {
			panic("invalid insert result")
		}
		*p = remoteInsertResult{Rejected: flags&1 != 0, Count: int(raw.integer())}
		if flags&2 != 0 {
			n := raw.integer()
			p.FirstID = &n
		}
	case *mosiResource:
		*p = raw.resource()
	case *clusterHello:
		*p = raw.hello()
	case *clusterIDRequest:
		*p = clusterIDRequest{Database: string(raw.blob()), Table: string(raw.blob()), Explicit: raw.uint(), Count: raw.uint()}
	case *clusterIDReply:
		*p = clusterIDReply{Base: raw.uint()}
	case *objectViewDeclaration:
		*p = objectViewDeclaration{URI: string(raw.blob()), View: raw.data}
		raw.data = nil

	case *remoteScanTask:
		*p = raw.scanTask()
	case *remoteScanResult:
		*p = raw.scanResult()
	case *remoteScanDescribe:
		*p = remoteScanDescribe{Version: int(raw.uint()), Database: string(raw.blob()), Table: string(raw.blob())}
	case *clusterObjectReply:
		kind := raw.octet()
		if kind > 1 {
			panic("invalid object result")
		}
		*p = clusterObjectReply{}
		if kind == 1 {
			p.Raw = raw.data
		} else {
			p.Value = raw.data
			if err := scm.RetainClusterValues(p.Value); err != nil {
				return err
			}
		}
		raw.data = nil
	default:
		typed = false
	}
	if typed {
		if len(raw.data) != 0 {
			return fmt.Errorf("trailing typed parameters")
		}
		return nil
	}

	if completion, ok := target.(*scanCompletion); ok {
		return decodeScanCompletion(data, completion)
	}
	if packet, ok := target.(*mosiPacket); ok {
		decoded, err := decodeMosiPacket(data)
		if err == nil {
			*packet = *decoded
		}
		return err
	}
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("node metadata decoding: %v", failure)
		}
	}()
	if len(data) > maxNodeRecordBytes {
		return scm.ErrClusterRequestTooLarge
	}
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("node decoder requires a destination")
	}
	r := nodeReader{data}
	r.value(v.Elem(), 0)
	if len(r.data) != 0 {
		return fmt.Errorf("trailing node metadata")
	}
	return nil
}
func (r *nodeReader) value(v reflect.Value, depth int) {
	if depth > 128 || len(r.data) == 0 {
		panic("truncated/excessive node metadata")
	}
	tag := r.data[0]
	r.data = r.data[1:]
	if tag == nodeNil {
		if v.IsValid() {
			v.SetZero()
		}
		return
	}
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Interface {
		var decoded any
		switch tag {
		case nodeFalse, nodeTrue:
			decoded = tag == nodeTrue
		case nodeInt:
			n, k := binary.Varint(r.data)
			if k <= 0 {
				panic("truncated integer")
			}
			r.data = r.data[k:]
			decoded = n
		case nodeUint:
			decoded = r.uint()
		case nodeString:
			decoded = string(r.blob())
		case nodeBytes:
			decoded = r.blob()
		case nodeFloat:
			if len(r.data) < 8 {
				panic("truncated float")
			}
			decoded = math.Float64frombits(binary.LittleEndian.Uint64(r.data))
			r.data = r.data[8:]
		case nodeList:
			n := r.uint()
			if n > uint64(len(r.data)) {
				panic("excessive node list")
			}
			items := make([]any, int(n))
			for i := range items {
				r.value(reflect.ValueOf(&items[i]).Elem(), depth+1)
			}
			decoded = items
		case nodeMap:
			n := r.uint()
			if n > uint64(len(r.data)/2) {
				panic("excessive node map")
			}
			items := make(map[string]any, int(n))
			for i := uint64(0); i < n; i++ {
				key := string(r.blob())
				var item any
				r.value(reflect.ValueOf(&item).Elem(), depth+1)
				items[key] = item
			}
			decoded = items

		default:
			panic("typed node destination required")
		}
		v.Set(reflect.ValueOf(decoded))
		return
	}
	switch tag {
	case nodeFalse, nodeTrue:
		if v.IsValid() {
			v.SetBool(tag == nodeTrue)
		}
	case nodeInt:
		n, k := binary.Varint(r.data)
		if k <= 0 {
			panic("truncated integer")
		}
		r.data = r.data[k:]
		if v.IsValid() {
			if v.OverflowInt(n) {
				panic("integer overflow")
			}
			v.SetInt(n)
		}
	case nodeUint:
		n := r.uint()
		if v.IsValid() {
			if v.OverflowUint(n) {
				panic("integer overflow")
			}
			v.SetUint(n)
		}
	case nodeFloat:
		if len(r.data) < 8 {
			panic("truncated float")
		}
		n := math.Float64frombits(binary.LittleEndian.Uint64(r.data))
		r.data = r.data[8:]
		if v.IsValid() {
			v.SetFloat(n)
		}
	case nodeString:
		b := r.blob()
		if v.IsValid() {
			v.SetString(string(b))
		}
	case nodeBytes:
		b := r.blob()
		if v.IsValid() {
			if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
				v.SetBytes(b)
			} else {
				nested := nodeReader{b}
				nested.value(v, depth+1)
				if len(nested.data) != 0 {
					panic("trailing nested metadata")
				}
			}
		}
	case nodeScheme:
		b := r.blob()
		if v.IsValid() {
			if v.Type() != nodeScmerType {
				panic("invalid Scheme metadata destination")
			}
			value, err := schemeValue(b).reconstruct()
			if err != nil {
				panic(err)
			}
			v.Set(reflect.ValueOf(value))
		}
	case nodeList:
		n := r.uint()
		if n > uint64(len(r.data)) {
			panic("excessive node list")
		}
		if v.IsValid() {
			v.Set(reflect.MakeSlice(v.Type(), int(n), int(n)))
		}
		for i := 0; i < int(n); i++ {
			var item reflect.Value
			if v.IsValid() {
				item = v.Index(i)
			}
			r.value(item, depth+1)
		}
	case nodePairs:
		n := r.uint()
		if n > uint64(len(r.data)/2) {
			panic("excessive node map")
		}
		if v.IsValid() {
			if v.Kind() != reflect.Map {
				panic("invalid node map")
			}
			v.Set(reflect.MakeMapWithSize(v.Type(), int(n)))
		}
		for i := uint64(0); i < n; i++ {
			var key, value reflect.Value
			if v.IsValid() {
				key = reflect.New(v.Type().Key()).Elem()
				value = reflect.New(v.Type().Elem()).Elem()
			}
			r.value(key, depth+1)
			r.value(value, depth+1)
			if v.IsValid() {
				v.SetMapIndex(key, value)
			}
		}
	case nodeMap:
		n := r.uint()
		if n > uint64(len(r.data)/2) {
			panic("excessive node map")
		}
		var layout *nodeLayout
		if v.IsValid() {
			if v.Kind() == reflect.Struct {
				layout = metadataLayout(v.Type())
			} else if v.Kind() == reflect.Map {
				v.Set(reflect.MakeMapWithSize(v.Type(), int(n)))
			} else {
				panic("invalid node map destination")
			}
		}
		for i := uint64(0); i < n; i++ {
			name := r.blob()
			var field reflect.Value
			if v.IsValid() {
				if layout != nil {
					// The common struct path looks up bytes without copying each
					// field name. Only map keys need a retained string allocation.
					index, ok := layout.byName[string(name)]
					if !ok {
						index, ok = layout.byName[strings.ToLower(string(name))]
					}
					if ok {
						field = v.FieldByIndex(index)
					}
				} else {
					field = reflect.New(v.Type().Elem()).Elem()
				}
			}
			r.value(field, depth+1)
			if v.IsValid() && layout == nil {
				key := reflect.New(v.Type().Key()).Elem()
				key.SetString(string(name))
				v.SetMapIndex(key, field)
			}
		}

	default:
		panic("unknown node metadata tag")
	}
}

type mosiResource struct {
	Object   string `json:"object,omitempty"`
	Database string `json:"database"`
	Table    string `json:"table,omitempty"`
	Shard    string `json:"shard,omitempty"`
}

func (resource mosiResource) key() string {
	if resource.Object != "" {
		return "@" + resource.Object
	}
	if resource.Shard == "" {
		return resource.Database
	}
	return resource.Database + "/" + resource.Shard
}

func (resource mosiResource) valid() bool {
	if resource.Object != "" {
		return resource == globalResource
	}
	if !validReplicaDatabase(resource.Database) {
		return false
	}
	if resource.Shard == "" {
		return resource.Table == ""
	}
	_, err := uuid.Parse(resource.Shard)
	return err == nil && resource.Table != ""
}

// Packet kinds select their header fields: read/write carry user proof;
// recall/invalidate carry an epoch; begin/object/chunk/end install one resource;
// grant/failure/ack complete a state transition. A chunk is independently gzip
// compressed and bounded, so queues never retain a whole shard or column.
type mosiPacket struct {
	Capture    *scm.ClusterReference `json:"capture,omitempty"`
	From       string                `json:"from,omitempty"` // logical source URI, preserved across neighbour hops
	To         string                `json:"to,omitempty"`   // logical destination URI, independent of a connection
	Forwards   []shardForward        `json:"forwards,omitempty"`
	Kind       string                `json:"kind"`
	ID         string                `json:"id"`
	Resource   mosiResource          `json:"resource"`
	Epoch      uint64                `json:"epoch,omitempty"`
	State      string                `json:"state,omitempty"`
	Content    string                `json:"content,omitempty"`
	Object     string                `json:"object,omitempty"`
	ObjectKind string                `json:"object_kind,omitempty"`
	User       string                `json:"user,omitempty"`
	Hash       string                `json:"hash,omitempty"`
	Payload    []byte                `json:"payload,omitempty"`
	Error      string                `json:"error,omitempty"`
	Owner      string                `json:"owner,omitempty"`
	Readers    map[string]bool       `json:"readers,omitempty"`
	Sequence   uint64                `json:"sequence,omitempty"`
	Boot       string                `json:"boot,omitempty"`     // immutable originating cluster incarnation
	TryRead    bool                  `json:"try_read,omitempty"` // decline read acquisition instead of draining an active writer
}

func (peer *mosiPeer) post(packet mosiPacket) error {
	if packet.Capture != nil {
		defer packet.Capture.Publication.Queued()
	}
	if peer.scanService != nil && peer.scanService.cluster != nil {
		if packet.From == "" {
			packet.From = peer.scanService.cluster.self.ID
		}
	}

	payload, err := encodeMosiPacket(packet)
	if err != nil {
		return err
	}
	return peer.send(peer.ctxOrBackground(), mosiMessage{Channel: "mosi", Payload: payload})
}

func (peer *mosiPeer) sendObject(header mosiPacket, kind, name string, reader io.ReadCloser) error {
	defer reader.Close()
	header.Kind, header.ObjectKind, header.Object = "object", kind, name
	if err := peer.post(header); err != nil {
		return err
	}
	buffer := make([]byte, mosiPacketBytes)
	for {
		if err := peer.outgoing.admitBulk(peer.done); err != nil {
			return err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			var packed bytes.Buffer
			compressor := gzip.NewWriter(&packed)
			if _, writeErr := compressor.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if closeErr := compressor.Close(); closeErr != nil {
				return closeErr
			}
			header.Kind, header.Payload = "chunk", packed.Bytes()
			if sendErr := peer.post(header); sendErr != nil {
				return sendErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	header.Kind, header.Payload = "object_end", nil
	return peer.post(header)
}

func decodeMosiChunk(packet mosiPacket) ([]byte, error) {
	if len(packet.Payload) > mosiPacketBytes+1024 {
		return nil, fmt.Errorf("oversized replication packet")
	}
	reader, err := gzip.NewReader(bytes.NewReader(packet.Payload))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, mosiPacketBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > mosiPacketBytes {
		return nil, fmt.Errorf("oversized replication payload")
	}
	return data, nil
}

// Capture manifests are part of the protocols which actually transfer native
// references, not an envelope imposed on every channel. Views were already
// declared on the stream; transferManifest removes their repeated bytes.
func appendReferences(b []byte, refs []scm.ClusterReference) []byte {
	b = binary.AppendUvarint(b, uint64(len(refs)))
	for _, ref := range refs {
		for _, value := range []string{ref.URI, ref.Owner, ref.Incarnation, ref.Token, ref.Type} {
			b = appendNodeBlob(b, []byte(value))
		}
		b = appendNodeBlob(b, ref.View)
	}
	return b
}
func (r *nodeReader) references() []scm.ClusterReference {
	n := r.uint()
	if n > uint64(len(r.data))/6 {
		panic("truncated capture manifest")
	}
	if n == 0 {
		return nil
	}
	refs := make([]scm.ClusterReference, int(n))
	for i := range refs {
		refs[i] = scm.ClusterReference{URI: string(r.blob()), Owner: string(r.blob()), Incarnation: string(r.blob()), Token: string(r.blob()), Type: string(r.blob()), View: r.blob()}
		if err := scm.RetainClusterValues(refs[i].View); err != nil {
			panic(err)
		}
		if len(refs[i].View) == 0 {
			refs[i].View = nil
		}
	}
	return refs
}
