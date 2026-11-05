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
import "bytes"
import "compress/gzip"
import "math"
import "sync"
import "reflect"
import "strings"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

const mosiPacketBytes = 64 * 1024

const mosiHTTPPath = "/cluster"
const mosiSubprotocol = "memcp.mosi.v5"
const mosiMaxMessageBytes = 256 * 1024
const maxNodeRecordBytes = 32 * 1024 * 1024

// A frame addresses a handle slot, a named endpoint, a reply slot or MOSI.
// Its body is the endpoint payload. No JSON envelope or embedded field names
// surround ordinary handle deliveries. Raw column chunks remain raw bytes.
func encodeMosiMessage(m mosiMessage) ([]byte, error) {
	b := append(make([]byte, 0, 256), "MOSI\x05"...)
	switch {
	case m.Packet != nil:
		packet := *m.Packet
		payload := packet.Payload
		packet.Payload = nil
		if len(payload) > 0 && (packet.Kind != "chunk" || len(payload) > mosiPacketBytes+1024) {
			return nil, fmt.Errorf("invalid MOSI binary payload")
		}
		metadata, err := marshalNode(packet)
		if err != nil {
			return nil, err
		}
		b = appendNodeBlob(append(b, 3), metadata)
		b = append(b, payload...)
	case m.Slot != 0:
		b = binary.AppendUvarint(append(b, 0), m.Slot)
		b = append(b, m.Payload...)
	case m.Method != "":
		endpoint := m.Method
		if endpoint == "rpc" {
			return nil, fmt.Errorf("reserved node endpoint")
		}
		if m.ID != 0 {
			endpoint = "rpc"
		}
		b = appendNodeBlob(append(b, 1), []byte(endpoint))
		if m.ID != 0 {
			// The RPC endpoint's payload supplies a return slot and method.
			// Ordinary one-way endpoints carry only their own payload.
			b = binary.AppendUvarint(b, m.ID)
			b = appendNodeBlob(b, []byte(m.Method))
		}
		b = append(b, m.Data...)
	default:
		b = binary.AppendUvarint(append(b, 2), m.ID)
		b = appendNodeBlob(b, []byte(m.Error))
		b = append(b, m.Data...)
	}
	if len(b) > mosiMaxMessageBytes {
		return nil, scm.ErrClusterRequestTooLarge
	}
	return b, nil
}
func decodeMosiMessage(frame []byte) (m mosiMessage, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			m = mosiMessage{}
			err = fmt.Errorf("invalid MOSI frame: %v", failure)
		}
	}()
	if len(frame) < 6 || len(frame) > mosiMaxMessageBytes || string(frame[:5]) != "MOSI\x05" {
		return m, fmt.Errorf("invalid MOSI message version or size")
	}
	r := nodeReader{frame[6:]}
	switch frame[5] {
	case 0:
		m.Slot = r.uint()
		if m.Slot == 0 {
			return m, fmt.Errorf("invalid handle slot")
		}
		m.Payload = r.data
	case 1:
		m.Method = string(r.blob())
		if m.Method == "rpc" {
			m.ID = r.uint()
			m.Method = string(r.blob())
			if m.ID == 0 || m.Method == "rpc" {
				return m, fmt.Errorf("invalid RPC endpoint payload")
			}
		}
		if len(r.data) > 0 {
			m.Data = r.data
		}
		if m.Method == "" {
			return m, fmt.Errorf("empty endpoint")
		}
	case 2:
		m.ID = r.uint()
		m.Error = string(r.blob())
		if len(r.data) > 0 {
			m.Data = r.data
		}
	case 3:
		m.Packet = new(mosiPacket)
		if err = unmarshalNode(r.blob(), m.Packet); err != nil {
			return m, err
		}
		if len(m.Packet.Payload) != 0 {
			return m, fmt.Errorf("inline payload")
		}
		if len(r.data) > 0 {
			if m.Packet.Kind != "chunk" || len(r.data) > mosiPacketBytes+1024 {
				return m, fmt.Errorf("invalid binary payload")
			}
			m.Packet.Payload = r.data
		}
	default:
		return m, fmt.Errorf("invalid recipient kind")
	}
	return m, nil
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
	if peer.scanService != nil && peer.scanService.cluster != nil {
		if packet.From == "" {
			packet.From = peer.scanService.cluster.self.ID
		}
	}

	message := mosiMessage{Packet: &packet}
	var err error
	message.encoded, err = encodeMosiMessage(message)
	if err != nil {
		return err
	}
	return peer.outgoing.publishPacket(peer.done, message)
}

func (peer *mosiPeer) sendObject(header mosiPacket, kind, name string, reader io.ReadCloser) error {
	defer reader.Close()
	header.Kind, header.ObjectKind, header.Object = "object", kind, name
	if err := peer.post(header); err != nil {
		return err
	}
	buffer := make([]byte, mosiPacketBytes)
	for {
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
