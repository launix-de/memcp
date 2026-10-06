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
	shards[0].filterFeedback.observe(key, 100, 100, 100)
	shards[1].filterFeedback.observe(key, 900, 0, 900)
	tbl.publishFilterFeedback(key)
	value, source, known := tbl.filterSelectivity(key)
	if !known || source != "scan_feedback" || math.Abs(value-.1) > 1e-12 {
		t.Fatalf("weighted estimate %v %s %v", value, source, known)
	}
	shards[0].filterFeedback.observe(key, 100, 0, 100)
	entry := shards[0].filterFeedback[filterFeedbackSlot(key.key)].Load()
	if entry.value != .99 || entry.samples != 2 {
		t.Fatalf("EMA = %+v", entry)
	}
	// The immutable old entry remains unchanged after publishing its successor.
	shards[0].filterFeedback.observe(key, 100, 0, 100)
	if entry.value != .99 {
		t.Fatal("published observation was mutated")
	}
	tbl.plannerStatsToken.Store(2)
	if _, source, known := tbl.filterSelectivity(key); !known || source != "historical_scan_feedback" {
		t.Fatal("new generation must retain historical feedback with lower confidence")
	}
}

func TestFilterFeedbackUnknownShardAndInvalidSamples(t *testing.T) {
	tbl := feedbackTestTable(100, 900)
	key := &filterObservation{key: "predicate", value: .1, generation: tbl.plannerStatsToken.Load()}
	shard := tbl.topology.Load().shards[0]
	shard.filterFeedback.observe(key, 100, 50, 100)
	tbl.publishFilterFeedback(key)
	value, _, _ := tbl.filterSelectivity(key)
	if math.Abs(value-.14) > 1e-12 {
		t.Fatalf("missing shard prior = %v", value)
	}
	before := shard.filterFeedback[filterFeedbackSlot(key.key)].Load()
	shard.filterFeedback.observe(key, 0, 0, 0)
	shard.filterFeedback.observe(key, 100, 101, 100)
	shard.filterFeedback.observe(key, 100, -1, 100)
	if shard.filterFeedback[filterFeedbackSlot(key.key)].Load() != before {
		t.Fatal("invalid observation published")
	}
}

func TestFilterFeedbackHistogramInterpolation(t *testing.T) {
	tbl := feedbackTestTable(10000)
	add := func(key string, length int, matched int64) {
		k := &filterObservation{key: key, family: "label:contains", length: length, generation: tbl.plannerStatsToken.Load()}
		tbl.topology.Load().shards[0].filterFeedback.observe(k, 10000, matched, 10000)
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
				tbl.topology.Load().shards[0].filterFeedback.observe(key, 100, int64(i), 100)
				tbl.publishFilterFeedback(key)
				if _, err := json.Marshal(tbl.persistFilterFeedback()); err != nil {
					t.Error(err)
				}
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
	token, fingerprint := tbl.PlannerStatsToken(true), tbl.PlannerStatisticsFingerprint(true)
	for _, body := range []string{`(equal? x 1)`, `(equal? x 101)`} {
		schema, values, filter := feedbackTestCompile(t, body)
		tbl.scan(nil, schema, values, cols[:1], scm.Eval(filter, &scm.Globalenv), nil,
			scm.Globalenv.Vars[scm.Symbol("scan_count")], scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false)
		tbl.scanRecSet(nil, schema, values, cols[:1], scm.Eval(filter, &scm.Globalenv))
		if _, _, known := tbl.filterSelectivity(bindFilterFeedback(schema.Slice(), values)); known {
			t.Fatal("unique point probe trained redundant selectivity")
		}
	}
	if tbl.PlannerStatsToken(true) != token || tbl.PlannerStatisticsFingerprint(true) != fingerprint {
		t.Fatal("unique point probe invalidated cached planning statistics")
	}
}

func TestFilterFeedbackOnlyCompilationDoesNotEvaluateBoundaries(t *testing.T) {
	for _, body := range []string{`(equal? x (print "must not run"))`, `(equal? x (outer 1 x))`} {
		schema, bindings := feedbackTestAccess(scm.NewSlice([]scm.Scmer{scm.NewString("c0")}), scm.Read("feedback-test", "(lambda (x) "+body+")"))
		if len(schema.Slice()) != 0 || len(bindings) != 0 {
			t.Fatalf("unsafe statistical compilation: %s", body)
		}
	}
	filter := scm.Read("feedback-test", `(lambda (different_alias) (strlike different_alias "%abc%" "utf8mb4_general_ci"))`)
	cols := scm.NewSlice([]scm.Scmer{scm.NewString("c0")})
	schema, values := feedbackTestAccess(cols, filter)
	physical, bound, _ := feedbackTestCompile(t, `(strlike x "%abc%" "utf8mb4_general_ci")`)
	if bindFilterFeedback(schema.Slice(), values).key != bindFilterFeedback(physical.Slice(), bound).key {
		t.Fatal("logical and physical filter identities differ")
	}
}

func TestFilterFeedbackCheckpoint(t *testing.T) {
	tbl := feedbackTestTable(1000)
	tbl.Name = "documents"
	tbl.publishShowColumnsSnapshot()
	key := &filterObservation{key: "contains-word", family: "name:contains", length: 4, generation: tbl.plannerStatsToken.Load()}
	tbl.topology.Load().shards[0].filterFeedback.observe(key, 1000, 123, 1000)
	tbl.publishFilterFeedback(key)
	// Maintenance changes the runtime generation before schema checkpointing.
	tbl.plannerStatsToken.Add(1)
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	restored.plannerStatsToken.Store(9)
	restored.publishShowColumnsSnapshot()
	restored.restoreFilterFeedback()
	value, source, known := restored.filterSelectivity(key)
	if !known || value != .123 || source != "historical_scan_feedback" {
		t.Fatalf("restored: %v %s %v; %s", value, source, known, encoded)
	}
	value, source, known = restored.filterSelectivity(&filterObservation{key: "unseen", family: key.family, length: 5})
	if !known || math.Abs(value-.123*.35) > 1e-12 || source != "like_length_histogram" {
		t.Fatalf("restored histogram: %v %s %v", value, source, known)
	}
	if restored.RestoredFilterFeedback != nil {
		t.Fatal("retained decode buffer")
	}
	restored.Collation = "utf8mb4_bin"
	restored.publishShowColumnsSnapshot()
	if _, _, known := restored.filterSelectivity(key); known {
		t.Fatal("incompatible schema reused feedback")
	}
	tbl.PersistencyMode = Memory
	if tbl.persistFilterFeedback() != nil {
		t.Fatal("persisted memory hints")
	}
}

func TestFilterFeedbackInvalidCheckpoint(t *testing.T) {
	for _, data := range []string{`{"version":99}`, `{"version":1,"entries":"wrong"}`, `[]`, `{"version":1,"entries":[{"key":"x","length":-1,"value":0.2,"population":100,"samples":1}]}`} {
		tbl := feedbackTestTable(100)
		if err := json.Unmarshal([]byte(data), &tbl.RestoredFilterFeedback); err != nil {
			t.Fatal(err)
		}
		tbl.restoreFilterFeedback()
		if _, _, known := tbl.filterSelectivity(&filterObservation{key: "x"}); known {
			t.Fatal("invalid persisted hint accepted")
		}
	}
}

func TestCompleteRecSetFilterFeedback(t *testing.T) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(t, "recset_feedback_test", 1000)
	schema, values, filter := feedbackTestCompile(t, `(not (equal? (mod x 10) 0))`)
	condition := scm.Eval(scm.Optimize(filter, &scm.Globalenv, nil), &scm.Globalenv)
	result := tbl.scanRecSet(nil, schema, values, cols[:1], condition)
	key := bindFilterFeedback(schema.Slice(), values)
	value, source, known := tbl.filterSelectivity(key)
	if result.count != 900 || !known || value != .9 || source != "scan_feedback" {
		t.Fatalf("count=%d value=%v source=%s known=%v", result.count, value, source, known)
	}
}

func TestFilterFeedbackColdDatabaseRestart(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "feedback_restart", 1000)
	RebuildTable(tbl, true, false)
	filter := scm.Read("feedback-restart", `(lambda (x) (< x 124))`)
	schema, values, _ := compileScanAccess(scm.NewSlice([]scm.Scmer{scm.NewString("id")}), filter)
	for i := range values {
		values[i] = scm.Eval(values[i], &scm.Globalenv)
	}
	tbl.scanRecSet(nil, schema, values, []string{"id"}, scm.Eval(filter, &scm.Globalenv))
	key := bindFilterFeedback(schema.Slice(), values)
	if value, _, known := tbl.filterSelectivity(key); !known || value != .123 {
		t.Fatalf("training %v %v", value, known)
	}
	RebuildTable(tbl, true, false)
	db := newDatabase()
	db.Name = "feedback_restart"
	db.persistence = persistence
	db.srState = COLD
	db.ensureLoaded()
	restored := db.GetTable("items")
	if value, source, known := restored.filterSelectivity(key); !known || value != .123 || source != "historical_scan_feedback" {
		t.Fatalf("restart %v %s %v", value, source, known)
	}
	for _, shard := range restored.ActiveShards() {
		if shard.state() != COLD {
			t.Fatal("feedback lookup loaded shard")
		}
		if shard.filterFeedback[filterFeedbackSlot(key.key)].Load() != nil {
			t.Fatal("invented shard samples on restart")
		}
	}
}

func TestRestrictedRecSetDoesNotTrainTableFeedback(t *testing.T) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(t, "restricted_feedback", 1000)
	schema, values, filter := feedbackTestCompile(t, `(< x 100)`)
	source := tbl.scanRecSet(nil, schema, values, cols[:1], scm.Eval(filter, &scm.Globalenv))
	restrictedSchema, restrictedValues, restrictedFilter := feedbackTestCompile(t, `(< x 50)`)
	result := source.filterToRecSet(nil, cols[:1], scm.Eval(restrictedFilter, &scm.Globalenv), restrictedSchema, restrictedValues)
	if result.count != 50 {
		t.Fatalf("count %d", result.count)
	}
	key := bindFilterFeedback(restrictedSchema.Slice(), restrictedValues)
	if _, _, known := tbl.filterSelectivity(key); known {
		t.Fatal("conditional 50/100 trained global filter")
	}
}

func TestFilterFeedbackUnchangedMeasurementDoesNotWrite(t *testing.T) {
	tbl := feedbackTestTable(100)
	key := &filterObservation{key: "stable", generation: tbl.plannerStatsToken.Load()}
	shard := tbl.topology.Load().shards[0]
	shard.filterFeedback.observe(key, 100, 90, 100)
	tbl.publishFilterFeedback(key)
	beforeShard := shard.filterFeedback[filterFeedbackSlot(key.key)].Load()
	beforeTable := tbl.filterFeedback.Load()
	shard.filterFeedback.observe(key, 100, 90, 100)
	tbl.publishFilterFeedback(key)
	if beforeShard != shard.filterFeedback[filterFeedbackSlot(key.key)].Load() || beforeTable != tbl.filterFeedback.Load() {
		t.Fatal("identical observation dirtied a published cache cell")
	}
	shard.filterFeedback.observe(key, 200, 180, 200)
	if beforeShard == shard.filterFeedback[filterFeedbackSlot(key.key)].Load() {
		t.Fatal("changed population was ignored")
	}
}

func TestFilterFeedbackSameLengthWordsRetainExactRates(t *testing.T) {
	tbl := feedbackTestTable(1000)
	shard := tbl.topology.Load().shards[0]
	common := &filterObservation{key: "alpha", family: "label:contains", length: 5, generation: tbl.plannerStatsToken.Load()}
	rare := &filterObservation{key: "bravo", family: common.family, length: 5, generation: common.generation}
	shard.filterFeedback.observe(common, 1000, 900, 1000)
	tbl.publishFilterFeedback(common)
	shard.filterFeedback.observe(rare, 1000, 10, 1000)
	tbl.publishFilterFeedback(rare)
	for i := 0; i < 20; i++ {
		shard.filterFeedback.observe(common, 1000, 900, 1000)
		tbl.publishFilterFeedback(common)
	}
	for key, want := range map[*filterObservation]float64{common: .9, rare: .01} {
		if value, source, known := tbl.filterSelectivity(key); !known || value != want || source != "scan_feedback" {
			t.Fatalf("exact word %s: %v %s %v", key.key, value, source, known)
		}
	}
	unknown := &filterObservation{key: "cider", family: common.family, length: 5}
	if value, source, known := tbl.filterSelectivity(unknown); !known || math.Abs(value-.455) > 1e-12 || source != "like_length_histogram" {
		t.Fatalf("unseen word %v %s %v", value, source, known)
	}
	unknown.family = "different-column:contains"
	if _, _, known := tbl.filterSelectivity(unknown); known {
		t.Fatal("histogram crossed columns")
	}
}

// Test fixture construction for native feedback publication and binding.
func feedbackTestAccess(columnExpr, filterExpr scm.Scmer) (scm.Scmer, []scm.Scmer) {
	columns, columnsOK := scanStaticColumns(columnExpr)
	params, body, lambdaOK := scanLambdaParts(filterExpr)
	if !columnsOK || !lambdaOK || len(params) != len(columns) {
		return scm.NewSlice(nil), nil
	}
	var bindings []scm.Scmer
	spec := compileFilterFeedback(params, columns, body, &bindings)
	if spec.IsNil() {
		return scm.NewSlice(nil), nil
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), spec})}), bindings
}

func TestNumericKeyFrequencySkewAndTail(t *testing.T) {
	var collector keyFrequencyCollector
	for i := 0; i < 10000; i++ {
		key := int64(1)
		if i%10 >= 6 {
			key = 2 + int64((i/10)%99)
		}
		collector.observe(scm.NewInt(key))
		if len(collector.counts) > keyFrequencySlots {
			t.Fatal("unbounded counters")
		}
	}
	stats := collector.finish()
	hot, ok := stats.frequency(scm.NewInt(1), 100)
	if !ok || math.Abs(hot-.6) > .02 {
		t.Fatalf("hot estimate %g", hot)
	}
	tail, ok := stats.frequency(scm.NewInt(100), 100)
	if !ok || tail <= 0 || tail > .01 {
		t.Fatalf("tail estimate %g", tail)
	}
	if len(stats.top) > keyFrequencyTop {
		t.Fatal("unbounded retained keys")
	}
}

func TestNumericKeyFrequencyNullAndMixedDomain(t *testing.T) {
	var collector keyFrequencyCollector
	for _, v := range []scm.Scmer{scm.NewInt(1), scm.NewFloat(1), scm.NewInt(2), scm.NewNil()} {
		collector.observe(v)
	}
	stats := collector.finish()
	for _, tc := range []struct {
		key  scm.Scmer
		want float64
	}{{scm.NewInt(1), .5}, {scm.NewFloat(1), .5}, {scm.NewInt(2), .25}, {scm.NewNil(), .25}, {scm.NewInt(3), 0}} {
		got, ok := stats.frequency(tc.key, 100)
		if !ok || got != tc.want {
			t.Fatalf("%v: %g instead of %g", tc.key, got, tc.want)
		}
	}
	if _, ok := stats.frequency(scm.NewString("1"), 100); ok {
		t.Fatal("coerced string received numeric statistics")
	}
	collector.observe(scm.NewString("1"))
	if collector.finish() != nil {
		t.Fatal("mixed equality domain retained")
	}
}

func TestNumericKeyFrequencyCompletePredicateOnly(t *testing.T) {
	var collector keyFrequencyCollector
	collector.observe(scm.NewInt(7))
	collector.observe(scm.NewInt(7))
	collector.observe(scm.NewInt(9))
	tbl := &table{}
	tbl.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{metadata: &tableShowColumnsSnapshotMetadata{columns: &tableShowColumnsMetadata{
		names: []string{"owner"}, distinctEstimates: []uint64{2}, plannerStatistics: []*columnPlannerStatistics{{KeyFrequency: collector.finish()}},
	}}})
	// Build real access metadata, including expression identity and value slots.
	params := []scm.Scmer{scm.NewSymbol("owner")}
	columns := []scm.Scmer{scm.NewString("owner")}
	var values []scm.Scmer
	body := scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), params[0], scm.NewInt(7)})
	feedback := compileFilterFeedback(params, columns, body, &values)
	schema := []scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewInt(0), feedback})}

	got, _, ok := tbl.keyFrequencySelectivity(schema, values)
	if !ok || math.Abs(got-2.0/3) > 1e-12 {
		t.Fatalf("complete predicate = %g %v", got, ok)
	}
	values = nil
	compound := scm.NewSlice([]scm.Scmer{scm.NewSymbol("and"), body, scm.NewSlice([]scm.Scmer{scm.NewSymbol(">"), params[0], scm.NewInt(0)})})
	schema[0] = scm.NewSlice([]scm.Scmer{scm.NewInt(0), compileFilterFeedback(params, columns, compound, &values)})
	if _, _, ok := tbl.keyFrequencySelectivity(schema, values); ok {
		t.Fatal("conjunct frequency mistaken for full expression")
	}
}

func TestNumericKeyFrequencySetSummation(t *testing.T) {
	var collector keyFrequencyCollector
	for i := 0; i < 10000; i++ {
		collector.observe(scm.NewInt(int64(i % 100)))
	}
	stats := collector.finish()
	keys := recSetProjectKeys{width: 1}
	for i := 0; i < 100; i++ {
		keys.values = append(keys.values, scm.NewInt(int64(i)))
	}
	if !keys.buildNumericLookup() {
		t.Fatal("numeric keys rejected")
	}
	got := stats.keySetFraction(keys, 100)
	var want float64
	for _, key := range keys.values {
		rate, _ := stats.frequency(key, 100)
		want += rate
	}
	if math.Abs(got-want) > 1e-12 || math.Abs(got-1) > 1e-12 {
		t.Fatalf("full domain %g expected %g", got, want)
	}
}

func TestNumericKeyFrequencyMetadataAndFeedbackPriority(t *testing.T) {
	Init(scm.Globalenv)
	var collector keyFrequencyCollector
	collector.observe(scm.NewInt(7))
	collector.observe(scm.NewInt(7))
	collector.observe(scm.NewInt(9))
	tbl := feedbackTestTable(3)
	tbl.PlannerRowEstimate.value.Store(3)
	tbl.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{metadata: &tableShowColumnsSnapshotMetadata{columns: &tableShowColumnsMetadata{
		names: []string{"c0"}, distinctEstimates: []uint64{2}, plannerStatistics: []*columnPlannerStatistics{{KeyFrequency: collector.finish()}},
	}}})
	schema, values, _ := feedbackTestCompile(t, `(equal? x 7)`)
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { t.Fatal("metadata read evaluated callback"); return scm.NewNil() })
	args := []scm.Scmer{scm.NewNil(), NewTableScmer(tbl), schema, scm.NewSlice(values), scm.NewSlice([]scm.Scmer{scm.NewString("c0")}), callback, scm.NewInt(0)}
	estimate := func() (float64, string) {
		result := scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_selectivity_estimate")], args...)
		value, source := math.NaN(), ""
		for _, item := range result.Slice() {
			pair := item.Slice()
			switch scm.String(pair[0]) {
			case "value":
				value = pair[1].Float()
			case "source":
				source = scm.String(pair[1])
			}
		}
		return value, source
	}
	value, source := estimate()
	if math.Abs(value-2.0/3) > 1e-12 || source != "numeric_key_frequency" {
		t.Fatalf("prior %g %s", value, source)
	}
	key := bindFilterFeedback(schema.Slice(), values)
	key.generation = tbl.plannerStatsToken.Load()
	tbl.topology.Load().shards[0].filterFeedback.observe(key, 3, 1, 3)
	tbl.publishFilterFeedback(key)
	value, source = estimate()
	if math.Abs(value-1.0/3) > 1e-12 || source != "scan_feedback" {
		t.Fatalf("feedback %g %s", value, source)
	}
}

func TestNumericKeyFrequencyColdPersistence(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "key_frequency_restart", 10000)
	RebuildTable(tbl, true, false)
	before, _ := tbl.keyFrequency("id")
	if before == nil {
		t.Fatal("rebuild did not publish frequency statistics")
	}
	baseline, ok := before.frequency(scm.NewInt(7), 10000)
	if !ok {
		t.Fatal("numeric estimate missing")
	}
	hints := tbl.persistKeyFrequencies()
	if hints == nil {
		t.Fatal("frequency statistics not persisted")
	}
	serialized, err := json.Marshal(hints)
	if err != nil {
		t.Fatal(err)
	}
	if len(serialized) > 10000*8/16 {
		t.Fatalf("hints too large: %d", len(serialized))
	}
	db := newDatabase()
	db.Name = "key_frequency_restart"
	db.persistence = persistence
	db.srState = COLD
	db.ensureLoaded()
	restored := db.GetTable("items")
	after, ndv := restored.keyFrequency("id")
	if after == nil {
		t.Fatal("frequency statistics not restored")
	}
	value, ok := after.frequency(scm.NewInt(7), ndv)
	if !ok || value != baseline {
		t.Fatalf("restored rate %g expected %g", value, baseline)
	}
	for _, shard := range restored.ActiveShards() {
		if shard.state() != COLD {
			t.Fatal("restoring/reading statistics loaded a shard")
		}
	}
	for _, bad := range []string{`{"version":99}`, `{"version":1,"columns":"invalid"}`} {
		var decoded persistedKeyFrequencies
		if err := json.Unmarshal([]byte(bad), &decoded); err != nil || decoded.Version != 0 {
			t.Fatalf("invalid optional hints rejected schema: %v %+v", err, decoded)
		}
	}
}

func TestNumericKeyFrequencySnapshotPublication(t *testing.T) {
	var first, second keyFrequencyCollector
	for i := 0; i < 100; i++ {
		first.observe(scm.NewInt(1))
		second.observe(scm.NewInt(int64(i % 2)))
	}
	snapshots := []*tableShowColumnsSnapshot{
		{metadata: &tableShowColumnsSnapshotMetadata{columns: &tableShowColumnsMetadata{
			names: []string{"owner"}, distinctEstimates: []uint64{1},
			plannerStatistics: []*columnPlannerStatistics{{KeyFrequency: first.finish()}},
		}}},
		{metadata: &tableShowColumnsSnapshotMetadata{columns: &tableShowColumnsMetadata{
			names: []string{"owner"}, distinctEstimates: []uint64{2},
			plannerStatistics: []*columnPlannerStatistics{{KeyFrequency: second.finish()}},
		}}},
	}
	tbl := &table{}
	tbl.showColumnsSnapshot.Store(snapshots[0])
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for j := 0; j < 1000; j++ {
				stats, ndv := tbl.keyFrequency("owner")
				rate, known := stats.frequency(scm.NewInt(1), ndv)
				if !known || (rate != 1 && rate != .5) {
					t.Errorf("torn frequency snapshot: %g %v", rate, known)
					return
				}
			}
		}()
	}
	for i := 0; i < 1000; i++ {
		tbl.showColumnsSnapshot.Store(snapshots[i%2])
	}
	readers.Wait()
}

func TestNumericKeyFrequencyUniqueTail(t *testing.T) {
	var collector keyFrequencyCollector
	for i := 0; i < 10000; i++ {
		collector.observe(scm.NewInt(int64(i)))
	}
	stats := collector.finish()
	if len(stats.top) != 0 {
		t.Fatal("unique late-arriving values became false heavy keys")
	}
	for _, key := range []int64{0, 7, 9900, 9999} {
		rate, known := stats.frequency(scm.NewInt(key), 10000)
		if !known || math.Abs(rate-.0001) > 1e-12 {
			t.Fatalf("unique key %d received a heavy-key prior: %g", key, rate)
		}
	}
}

func TestNumericKeyFrequencyPredicateNullSemantics(t *testing.T) {
	var collector keyFrequencyCollector
	for _, value := range []scm.Scmer{scm.NewInt(0), scm.NewInt(1), scm.NewInt(2), scm.NewNil()} {
		collector.observe(value)
	}
	tbl := &table{}
	tbl.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{metadata: &tableShowColumnsSnapshotMetadata{columns: &tableShowColumnsMetadata{
		names: []string{"owner"}, distinctEstimates: []uint64{4},
		plannerStatistics: []*columnPlannerStatistics{{KeyFrequency: collector.finish()}},
	}}})
	for _, tc := range []struct {
		operator string
		value    scm.Scmer
		want     float64
	}{{"equal?", scm.NewNil(), .5}, {"equal?", scm.NewInt(0), .5},
		{"equal??", scm.NewNil(), 0}, {"equal??", scm.NewInt(0), .25}} {
		params := []scm.Scmer{scm.NewSymbol("x")}
		var values []scm.Scmer
		body := scm.NewSlice([]scm.Scmer{scm.NewSymbol(tc.operator), params[0], tc.value})
		metadata := compileFilterFeedback(params, []scm.Scmer{scm.NewString("owner")}, body, &values)
		schema := []scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewInt(0), metadata})}
		rate, _, known := tbl.keyFrequencySelectivity(schema, values)
		if !known || rate != tc.want {
			t.Fatalf("%s %v: %g instead of %g (known %v)", tc.operator, tc.value, rate, tc.want, known)
		}
	}
}

func TestNumericKeyFrequencyRetainedMemory(t *testing.T) {
	c := &column{Name: "owner"}
	tbl := &table{Columns: []*column{c}}
	before := tbl.metadataMemory()
	var collector keyFrequencyCollector
	for i := 0; i < 10000; i++ {
		collector.observe(scm.NewInt(int64(i % 64)))
	}
	stats := collector.finish()
	if cap(stats.top) > keyFrequencyTop {
		t.Fatal("truncated top keys retain discarded backing storage")
	}
	c.PlannerStats.Store(&columnPlannerStatistics{KeyFrequency: stats})
	after := tbl.metadataMemory()
	if after <= before || after-before > 10000*8/16 {
		t.Fatalf("retained frequency memory not charged or not compact: %d bytes", after-before)
	}
}
