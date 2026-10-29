/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "bytes"
import "testing"
import "strings"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

var clusterEncodingSink []byte

// Measure the existing size check + collector + transport against retaining the
// first encoding. Fixtures are prepared outside timing, as in shard completion.
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
		return remoteScanResult{Version: scanWireVersion, Shard: remoteScanShard{Schema: 7, Resource: mosiResource{Database: "bench", Table: "items", Shard: "00000000-0000-0000-0000-000000000001"}}, Value: json.RawMessage(source), OutCount: 1}
	}
	input := &storageShard{uuid: uuid.MustParse("00000000-0000-4000-8000-000000000001"), t: &table{Name: "items", schema: &database{Name: "bench"}}}
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		encoded, err := freezeScanValue(view)
		return scm.ClusterReference{URI: "http://node/cluster/00000000-0000-4000-8000-000000000001", Owner: "00000000-0000-4000-8000-000000000002", Incarnation: "00000000-0000-4000-8000-000000000003", Token: "00000000-0000-4000-8000-000000000004", Type: object.ClusterObjectType(), View: json.RawMessage(encoded)}, err
	})
	encodedReference, err := freezeScanValue(scm.NewAny(input), bindings)
	if err != nil {
		b.Fatal(err)
	}
	reference := result(scm.NewNil())
	reference.Value = json.RawMessage(encodedReference)
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
			for _, mode := range []string{"existing", "reuse", "buffer"} {
				b.Run(mode, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						var data []byte
						var err error
						if mode == "buffer" {
							var buf bytes.Buffer
							err = json.NewEncoder(&buf).Encode(value)
							data = buf.Bytes()
						} else {
							data, err = json.Marshal(value)
						}
						if err != nil {
							b.Fatal(err)
						}
						if len(data) > 192*1024 {
							b.Fatal("fixture exceeds limit")
						}
						if mode == "existing" {
							data, err = json.Marshal(value)
							if err != nil {
								b.Fatal(err)
							}
						}
						completion := scanCompletion{Return: operationID, Data: data, Offers: offers}
						envelope, err := json.Marshal(completion)
						if err != nil {
							b.Fatal(err)
						}
						clusterEncodingSink, err = encodeMosiMessage(mosiMessage{Method: "scan.result", Data: envelope})
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
