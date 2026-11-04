/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "strings"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

var clusterEncodingSink []byte

// Exercise the actual completion/transport metadata path. Scmer fixtures are
// frozen before timing, as in shard completion; decoding validates the result.
func BenchmarkClusterCompletionEncoding(b *testing.B) {
	dict := scm.NewFastDictValue(512)
	for i := 0; i < 512; i++ {
		dict.Set(scm.NewInt(int64(i)), scm.NewString("dictionary payload"), nil)
	}
	rows := make([]scm.Scmer, 128)
	for i := range rows {
		rows[i] = scm.NewSlice([]scm.Scmer{scm.NewInt(int64(i)), scm.NewString("ordered page")})
	}
	// The size check receives a response whose Scheme payload is already
	// serialized. Measure that actual boundary, not a second Scmer traversal.
	result := func(value scm.Scmer) remoteScanResult {
		source, err := freezeScanValue(value)
		if err != nil {
			b.Fatal(err)
		}
		return remoteScanResult{Version: scanWireVersion, Shard: remoteScanShard{Schema: 7, Resource: mosiResource{Database: "bench", Table: "items", Shard: "00000000-0000-0000-0000-000000000001"}}, Value: []byte(source), OutCount: 1}
	}
	input := &storageShard{uuid: uuid.MustParse("00000000-0000-4000-8000-000000000001"), t: &table{Name: "items", schema: &database{Name: "bench"}}}
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		encoded, err := freezeScanValue(view)
		return scm.ClusterReference{URI: "http://node/cluster/00000000-0000-4000-8000-000000000001", Owner: "00000000-0000-4000-8000-000000000002", Incarnation: "00000000-0000-4000-8000-000000000003", Token: "00000000-0000-4000-8000-000000000004", Type: object.ClusterObjectType(), View: []byte(encoded)}, err
	})
	encodedReference, err := freezeScanValue(scm.NewAny(input), bindings)
	if err != nil {
		b.Fatal(err)
	}
	reference := result(scm.NewNil())
	reference.Value = []byte(encodedReference)
	operationID := strings.Repeat("0", 64) + "/00000000-0000-7000-8000-000000000001"
	for name, value := range map[string]any{
		"scalar":       result(scm.NewInt(42)),
		"list":         result(scm.NewSlice(rows[:8])),
		"dictionary":   result(scm.NewFastDict(dict)),
		"reference":    reference,
		"ordered-page": result(scm.NewSlice(rows)),
	} {
		b.Run(name, func(b *testing.B) {
			offers := value.(remoteScanResult).ClusterObjectOffers()
			encode := func() []byte {
				data, err := marshalNode(value)
				if err != nil {
					b.Fatal(err)
				}
				envelope, err := marshalNode(scanCompletion{Return: operationID, Data: data, Offers: offers})
				if err != nil {
					b.Fatal(err)
				}
				frame, err := encodeMosiMessage(mosiMessage{Method: "scan.result", Data: envelope})
				if err != nil {
					b.Fatal(err)
				}
				return frame
			}
			frame := encode()
			b.Run("encode", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					clusterEncodingSink = encode()
				}
				b.ReportMetric(float64(len(frame)), "wire-B")
			})
			b.Run("decode", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					message, err := decodeMosiMessage(frame)
					if err != nil {
						b.Fatal(err)
					}
					var completion scanCompletion
					if err := unmarshalNode(message.Data, &completion); err != nil {
						b.Fatal(err)
					}
					var result remoteScanResult
					if err := unmarshalNode(completion.Data, &result); err != nil {
						b.Fatal(err)
					}
					if result.Version != scanWireVersion || result.OutCount != 1 {
						b.Fatal("completion changed")
					}
				}
			})
		})
	}
}
