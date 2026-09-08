/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.
You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"math"
	"sync"
	"testing"

	"encoding/json"

	"github.com/launix-de/memcp/scm"
)

func feedbackTestTable(populations ...uint32) *table {
	t := &table{}
	t.plannerStatsToken.Store(1)
	topology := &tableShardTopology{}
	for _, population := range populations {
		shard := &storageShard{t: t}
		shard.plannerMainRows.Store(population)
		topology.shards = append(topology.shards, shard)
	}
	t.topology.Store(topology)
	return t
}

func TestFilterFeedbackMixAndWeightedMerge(t *testing.T) {
	tbl := feedbackTestTable(100, 900)
	key := &filterObservation{key: "predicate", value: .1, generation: tbl.plannerStatsToken.Load()}
	shards := tbl.topology.Load().shards
	shards[0].filterFeedback.observe(key, 100, 100)
	shards[1].filterFeedback.observe(key, 900, 0)
	tbl.publishFilterFeedback(key)
	value, source, known := tbl.filterSelectivity(key)
	if !known || source != "scan_feedback" || math.Abs(value-.1) > 1e-12 {
		t.Fatalf("weighted estimate %v %s %v", value, source, known)
	}
	shards[0].filterFeedback.observe(key, 100, 0)
	entry := shards[0].filterFeedback[filterFeedbackSlot(key.key)].Load()
	if entry.value != .99 || entry.samples != 2 {
		t.Fatalf("EMA = %+v", entry)
	}
	// The immutable old entry remains unchanged after publishing its successor.
	shards[0].filterFeedback.observe(key, 100, 0)
	if entry.value != .99 {
		t.Fatal("published observation was mutated")
	}
	tbl.plannerStatsToken.Store(2)
	if _, _, known := tbl.filterSelectivity(key); known {
		t.Fatal("rebuild/DDL generation reused stale feedback")
	}
}

func TestFilterFeedbackUnknownShardAndInvalidSamples(t *testing.T) {
	tbl := feedbackTestTable(100, 900)
	key := &filterObservation{key: "predicate", value: .1, generation: tbl.plannerStatsToken.Load()}
	shard := tbl.topology.Load().shards[0]
	shard.filterFeedback.observe(key, 100, 50)
	tbl.publishFilterFeedback(key)
	value, _, _ := tbl.filterSelectivity(key)
	if math.Abs(value-.14) > 1e-12 {
		t.Fatalf("missing shard prior = %v", value)
	}
	before := shard.filterFeedback[filterFeedbackSlot(key.key)].Load()
	shard.filterFeedback.observe(key, 0, 0)
	shard.filterFeedback.observe(key, 100, 101)
	shard.filterFeedback.observe(key, 100, -1)
	if shard.filterFeedback[filterFeedbackSlot(key.key)].Load() != before {
		t.Fatal("invalid observation published")
	}
}

func TestFilterFeedbackHistogramInterpolation(t *testing.T) {
	tbl := feedbackTestTable(10000)
	add := func(key string, length int, matched int64) {
		k := &filterObservation{key: key, family: "label:contains", length: length, generation: tbl.plannerStatsToken.Load()}
		tbl.topology.Load().shards[0].filterFeedback.observe(k, 10000, matched)
		tbl.publishFilterFeedback(k)
	}
	add("short", 2, 5000)
	add("long", 4, 1000)
	for _, test := range []struct {
		length int
		want   float64
	}{{2, .5}, {3, .3}, {4, .1}, {5, .035}, {1, 1}} {
		value, source, known := tbl.filterSelectivity(&filterObservation{key: "unseen", family: "label:contains", length: test.length})
		if !known || source != "like_length_histogram" || math.Abs(value-test.want) > 1e-12 {
			t.Fatalf("length %d = %v %s %v", test.length, value, source, known)
		}
	}
	for _, key := range []*filterObservation{{key: "unseen", family: "other:contains", length: 3}, {key: "unseen", family: "label:contains", length: 20}, {key: "unseen"}} {
		if _, _, known := tbl.filterSelectivity(key); known {
			t.Fatal("unrelated histogram used")
		}
	}
}

func TestFilterFeedbackConcurrentPublication(t *testing.T) {
	tbl := feedbackTestTable(100)
	key := &filterObservation{key: "predicate", generation: tbl.plannerStatsToken.Load()}
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 100; i++ {
				tbl.topology.Load().shards[0].filterFeedback.observe(key, 100, int64(i))
				tbl.publishFilterFeedback(key)
				value, _, known := tbl.filterSelectivity(key)
				if known && (value < 0 || value > 1 || math.IsNaN(value)) {
					t.Errorf("invalid concurrent estimate %v", value)
				}
			}
		}()
	}
	workers.Wait()
}

func feedbackTestCompile(t testing.TB, body string) (scm.Scmer, []scm.Scmer, scm.Scmer) {
	t.Helper()
	filter := scm.Read("feedback-test", "(lambda (x) "+body+")")
	schema, values, _ := compileScanAccess(scm.NewSlice([]scm.Scmer{scm.NewString("c0")}), filter)
	if !scanFeedbackMetadata(schema.Slice()).IsNil() {
		for i := range values {
			values[i] = scm.Eval(values[i], &scm.Globalenv)
		}
	}
	return schema, values, filter
}

func TestFilterFeedbackCompiledIdentityAndPersistence(t *testing.T) {
	Init(scm.Globalenv)
	schema, values, _ := feedbackTestCompile(t, `(strlike x "%abcd%" "utf8mb4_general_ci")`)
	key := bindFilterFeedback(schema.Slice(), values)
	if key == nil || key.length != 4 || key.family == "" {
		t.Fatalf("LIKE key: %+v", key)
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var decoded scm.Scmer
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	restored := bindFilterFeedback(decoded.Slice(), values)
	if restored == nil || restored.key != key.key {
		t.Fatalf("roundtrip key: %+v", restored)
	}
	changedValues := append([]scm.Scmer(nil), values...)
	for i, value := range changedValues {
		if value.IsString() && value.String() == "%abcd%" {
			changedValues[i] = scm.NewString("%different%")
		}
	}
	if changed := bindFilterFeedback(schema.Slice(), changedValues); changed == nil || changed.key == key.key {
		t.Fatal("cached literal identity ignored rebound values")
	}
	shifted := shiftCompiledScanAccessSlots(schema, 2)
	shiftedKey := bindFilterFeedback(shifted.Slice(), append([]scm.Scmer{scm.NewNil(), scm.NewNil()}, values...))
	if shiftedKey == nil || shiftedKey.key != key.key {
		t.Fatal("multi-scan slot shift changed identity")
	}
	for _, body := range []string{`(strlike x "abcd%" "utf8mb4_general_ci")`, `(strlike x "%abcd%" "utf8mb4_bin")`, `(strlike x "%ab_cd%" "utf8mb4_general_ci")`} {
		other, vals, _ := feedbackTestCompile(t, body)
		otherKey := bindFilterFeedback(other.Slice(), vals)
		if otherKey == nil || otherKey.key == key.key || otherKey.family == key.family {
			t.Fatalf("unrelated pattern aliases feedback: %s", body)
		}
	}
	for _, body := range []string{`(equal? x (random))`, `(equal? x (outer 1 x))`, `(begin (print x) true)`} {
		other, vals, _ := feedbackTestCompile(t, body)
		if bindFilterFeedback(other.Slice(), vals) != nil {
			t.Fatalf("unsafe filter admitted: %s", body)
		}
	}
}

func TestFilterFeedbackLearnsCompleteScan(t *testing.T) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(t, "feedback_complete", 100)
	schema, values, filter := feedbackTestCompile(t, `(equal? (mod x 10) 0)`)
	condition := scm.Eval(filter, &scm.Globalenv)
	mapper := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + 1) })
	combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) })
	result := tbl.scan(nil, schema, values, cols[:1], condition, nil, mapper, scm.NewInt(0), combine, false)
	if result.Int() != 10 {
		t.Fatalf("scan output: %v", result)
	}
	value, source, known := tbl.filterSelectivity(bindFilterFeedback(schema.Slice(), values))
	if !known || value != .1 || source != "scan_feedback" {
		t.Fatalf("scan feedback: %v %s %v", value, source, known)
	}
	// Early-exit scans must not train on their prefix.
	earlySchema, earlyValues, earlyFilter := feedbackTestCompile(t, `(equal? (mod x 7) 0)`)
	tbl.scanExists(nil, earlySchema, earlyValues, cols[:1], scm.Eval(earlyFilter, &scm.Globalenv))
	if _, _, known := tbl.filterSelectivity(bindFilterFeedback(earlySchema.Slice(), earlyValues)); known {
		t.Fatal("EXISTS prefix trained full-table selectivity")
	}
}

func TestFilterFeedbackUniquePointRetainsPlanStatistics(t *testing.T) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(t, "feedback_unique", 100)
	tbl.mu.Lock()
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: cols[:1]}}
	tbl.mu.Unlock()
	token, fingerprint := tbl.PlannerStatsToken(), tbl.PlannerStatisticsFingerprint()
	for _, body := range []string{`(equal? x 1)`, `(equal? x 101)`} {
		schema, values, filter := feedbackTestCompile(t, body)
		tbl.scan(nil, schema, values, cols[:1], scm.Eval(filter, &scm.Globalenv), nil,
			scm.Globalenv.Vars[scm.Symbol("scan_count")], scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false)
		if _, _, known := tbl.filterSelectivity(bindFilterFeedback(schema.Slice(), values)); known {
			t.Fatal("unique point probe trained redundant selectivity")
		}
	}
	if tbl.PlannerStatsToken() != token || tbl.PlannerStatisticsFingerprint() != fingerprint {
		t.Fatal("unique point probe invalidated cached planning statistics")
	}
}

func TestFilterFeedbackOnlyCompilationDoesNotEvaluateBoundaries(t *testing.T) {
	for _, body := range []string{`(equal? x (print "must not run"))`, `(equal? x (outer 1 x))`} {
		schema, bindings := compileFilterFeedbackAccess(scm.NewSlice([]scm.Scmer{scm.NewString("c0")}), scm.Read("feedback-test", "(lambda (x) "+body+")"))
		if len(schema.Slice()) != 0 || len(bindings) != 0 {
			t.Fatalf("unsafe statistical compilation: %s", body)
		}
	}
	filter := scm.Read("feedback-test", `(lambda (different_alias) (strlike different_alias "%abc%" "utf8mb4_general_ci"))`)
	cols := scm.NewSlice([]scm.Scmer{scm.NewString("c0")})
	schema, values := compileFilterFeedbackAccess(cols, filter)
	physical, bound, _ := feedbackTestCompile(t, `(strlike x "%abc%" "utf8mb4_general_ci")`)
	if bindFilterFeedback(schema.Slice(), values).key != bindFilterFeedback(physical.Slice(), bound).key {
		t.Fatal("logical and physical filter identities differ")
	}
}
