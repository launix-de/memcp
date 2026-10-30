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
import "context"
import "compress/gzip"
import "encoding/json"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

const mosiPacketBytes = 64 * 1024

const mosiHTTPPath = "/cluster"
const mosiSubprotocol = "memcp.mosi.v2"
const mosiMaxMessageBytes = 256 * 1024

// One binary WebSocket message contains MOSI + version 2 + a uint32 metadata
// length, followed by JSON control metadata and raw compressed chunk bytes.
// Packet payloads are never base64-expanded. This envelope is shared by both
// dedicated replication listeners and the normal HTTP node-protocol route.
func encodeMosiMessage(message mosiMessage) ([]byte, error) {
	var payload []byte
	if message.Packet != nil {
		packet := *message.Packet
		payload, packet.Payload = packet.Payload, nil
		message.Packet = &packet
		if len(payload) != 0 && (packet.Kind != "chunk" || len(payload) > mosiPacketBytes+1024) {
			return nil, fmt.Errorf("invalid MOSI binary payload")
		}
	}
	metadata, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	if 9+len(metadata)+len(payload) > mosiMaxMessageBytes {
		return nil, scm.ErrClusterRequestTooLarge
	}
	frame := make([]byte, 9+len(metadata)+len(payload))
	copy(frame, "MOSI\x02")
	binary.BigEndian.PutUint32(frame[5:9], uint32(len(metadata)))
	copy(frame[9:], metadata)
	copy(frame[9+len(metadata):], payload)
	return frame, nil
}

func decodeMosiMessage(frame []byte) (mosiMessage, error) {
	var message mosiMessage
	if len(frame) < 9 || len(frame) > mosiMaxMessageBytes || string(frame[:5]) != "MOSI\x02" {
		return message, fmt.Errorf("invalid MOSI message version or size")
	}
	length := uint64(binary.BigEndian.Uint32(frame[5:9]))
	if length > uint64(len(frame)-9) {
		return message, fmt.Errorf("truncated MOSI metadata")
	}
	if err := json.Unmarshal(frame[9:9+int(length)], &message); err != nil {
		return mosiMessage{}, fmt.Errorf("invalid MOSI metadata")
	}
	payload := frame[9+int(length):]
	if message.Packet != nil && len(message.Packet.Payload) != 0 {
		return mosiMessage{}, fmt.Errorf("inline MOSI payload is not allowed")
	}
	if len(payload) != 0 {
		if message.Packet == nil || message.Packet.Kind != "chunk" || len(payload) > mosiPacketBytes+1024 {
			return mosiMessage{}, fmt.Errorf("invalid MOSI binary payload")
		}
		message.Packet.Payload = payload
	}
	return message, nil
}

type mosiResource struct {
	Database string `json:"database"`
	Table    string `json:"table,omitempty"`
	Shard    string `json:"shard,omitempty"`
}

func (resource mosiResource) key() string {
	if resource.Shard == "" {
		return resource.Database
	}
	return resource.Database + "/" + resource.Shard
}

func (resource mosiResource) valid() bool {
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
	From       string          `json:"from,omitempty"` // logical source URI, preserved across neighbour hops
	To         string          `json:"to,omitempty"`   // logical destination URI, independent of a connection
	Forwards   []shardForward  `json:"forwards,omitempty"`
	Kind       string          `json:"kind"`
	ID         string          `json:"id"`
	Resource   mosiResource    `json:"resource"`
	Epoch      uint64          `json:"epoch,omitempty"`
	State      string          `json:"state,omitempty"`
	Content    string          `json:"content,omitempty"`
	Object     string          `json:"object,omitempty"`
	ObjectKind string          `json:"object_kind,omitempty"`
	User       string          `json:"user,omitempty"`
	Hash       string          `json:"hash,omitempty"`
	Payload    []byte          `json:"payload,omitempty"`
	Error      string          `json:"error,omitempty"`
	Owner      string          `json:"owner,omitempty"`
	Readers    map[string]bool `json:"readers,omitempty"`
	Sequence   uint64          `json:"sequence,omitempty"`
	Boot       string          `json:"boot,omitempty"` // immutable originating cluster incarnation
}

func (peer *mosiPeer) post(packet mosiPacket) error {
	if peer.scanService != nil && peer.scanService.cluster != nil {
		if packet.From == "" {
			packet.From = peer.scanService.cluster.self.ID
		}
	}

	return peer.send(context.Background(), mosiMessage{Packet: &packet})
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
