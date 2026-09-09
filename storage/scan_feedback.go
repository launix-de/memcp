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
	"fmt"
	"math"

	"strconv"
	"strings"

	"sync/atomic"

	"unicode/utf8"

	"github.com/launix-de/memcp/scm"
)

// Bounded direct-mapped caches: no map locks, LRU writes on reads, or unbounded
// retention of parameter values. Collisions evict observations, never alias them.
const filterFeedbackSlots = 64
const filterFeedbackMaxKey = 1024

type filterFeedbackCache [filterFeedbackSlots]atomic.Pointer[filterObservation]

type tableFilterFeedback struct {
	entries     [filterFeedbackSlots]*filterObservation
	fingerprint uint64
	schema      uint64
}

type filterObservation struct {
	key        string
	family     string // LIKE column, collation and wildcard topology; empty for scalar-only filters
	length     int
	value      float64
	population int64
	observed   int64
	// Pre-residual input work is independent of output selectivity. This is
	// generation-local costing telemetry, never a semantic membership bound.
	filterInput float64
	workKnown   bool
	samples     uint32
	generation  uint64
}

type filterFeedbackPart struct {
	text string
	slot int // -1 for a static token
}

// Compiled before residual pruning. Neither execution nor cache guards inspect
// callback ASTs. Runtime-bound values live in the existing access value vector.
type filterFeedbackSpec struct {
	parts       []filterFeedbackPart
	patternSlot int
	family      string
	prior       float64
}

func (s *filterFeedbackSpec) value() scm.Scmer {
	parts := make([]scm.Scmer, len(s.parts))
	for i, part := range s.parts {
		if part.slot < 0 {
			parts[i] = scm.NewString(part.text)
		} else {
			parts[i] = scm.NewInt(int64(part.slot))
		}
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSlice(parts), scm.NewInt(int64(s.patternSlot)), scm.NewString(s.family), scm.NewFloat(s.prior)})
}

func compileFilterFeedback(params, columns []scm.Scmer, body scm.Scmer, bindings *[]scm.Scmer) scm.Scmer {
	start := len(*bindings)
	spec := &filterFeedbackSpec{patternSlot: -1, prior: .1}
	nodes := 0
	var visit func(scm.Scmer) bool
	visit = func(expr scm.Scmer) bool {
		nodes++
		if nodes > 64 {
			return false
		}
		expr = expr.WithoutSourceInfo()
		if column, ok := scanParamColumn(expr, params, columns); ok {
			if isScanPseudoColName(column) || strings.HasPrefix(column, ".") || strings.HasPrefix(column, "$") {
				return false
			}
			spec.parts = append(spec.parts, filterFeedbackPart{text: "column:" + strconv.Quote(column), slot: -1})
			return true
		}
		if expr.IsNil() || expr.IsBool() || expr.IsInt() || expr.IsFloat() || expr.IsString() {
			spec.parts = append(spec.parts, filterFeedbackPart{slot: len(*bindings)})
			*bindings = append(*bindings, expr)
			return true
		}
		items, ok := scmerSlice(expr)
		if !ok || len(items) == 0 {
			return false
		}
		name, ok := scanSymbolName(items[0])
		if !ok {
			return false
		}
		if name == "optimize" && len(items) == 2 {
			return visit(items[1])
		}
		// Literal parameter binding emits (quote value), whereas scan callback
		// optimization may already have folded it to value. Both denote the same
		// scalar predicate. Do not recurse into quoted lists: those are data, not
		// executable filter expressions, and must never be interpreted as columns.
		if name == "quote" && len(items) == 2 {
			value := items[1]
			if value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() || value.IsString() {
				return visit(value)
			}
			return false
		}
		// Session parameters are constant over a scan; correlated outer-row values
		// deliberately do not enter this table-local model.
		if name == "session" && len(items) == 2 && items[1].IsString() {
			spec.parts = append(spec.parts, filterFeedbackPart{slot: len(*bindings)})
			*bindings = append(*bindings, expr)
			return true
		}
		switch name {
		case "equal?", "equal??", "=", "<", "<=", ">", ">=", "strlike", "nil?", "not", "sql_not", "and", "or", "mod", "+", "-", "*", "/", "concat", "strlen", "coalesceNil", "if", "floor", "ceil":
		default:
			return false
		}
		spec.parts = append(spec.parts, filterFeedbackPart{text: "(" + name, slot: -1})
		for _, arg := range items[1:] {
			if !visit(arg) {
				return false
			}
		}
		spec.parts = append(spec.parts, filterFeedbackPart{text: ")", slot: -1})
		return true
	}
	if !scanExprUsesParams(body, params) || !visit(body) {
		*bindings = (*bindings)[:start]
		return scm.NewNil()
	}
	if items, ok := scmerSlice(body); ok && len(items) >= 3 {
		name, _ := scanSymbolName(items[0])
		if name == "<" || name == "<=" || name == ">" || name == ">=" {
			spec.prior = 1.0 / 3
		}
		if name == "strlike" && len(items) <= 4 {
			if column, ok := scanParamColumn(items[1], params, columns); ok {
				collation := ""
				if len(items) == 4 {
					if !items[3].IsString() {
						return staticFilterFeedbackKey(spec.value(), *bindings)
					}
					collation = items[3].String()
				}
				// The first dynamic token is the pattern for a simple column LIKE value.
				for _, part := range spec.parts {
					if part.slot >= 0 {
						spec.patternSlot = part.slot
						break
					}
				}
				spec.family = strconv.Quote(column) + ":" + strconv.Quote(collation)
			}
		}
	}
	return staticFilterFeedbackKey(spec.value(), *bindings)
}

func bindFilterFeedback(schema []scm.Scmer, values []scm.Scmer) *filterObservation {
	if len(schema) == 0 || !schema[0].IsSlice() {
		return nil
	}
	spec, ok := scmerSlice(scanFeedbackMetadata(schema))
	if !ok || (len(spec) != 4 && len(spec) != 5) {
		return nil
	}
	if len(spec) == 5 {
		key := spec[4].Slice()
		expected := key[4].Slice()
		matched, position := true, 0
		for _, part := range spec[0].Slice() {
			if !part.IsInt() {
				continue
			}
			slot := int(part.Int())
			if slot < 0 || slot >= len(values) || !scm.Equal(values[slot], expected[position]) {
				matched = false
				break
			}
			position++
		}
		if matched {
			return &filterObservation{key: key[0].String(), family: key[1].String(), length: int(key[2].Int()), value: key[3].Float()}
		}
	}
	var key strings.Builder
	for _, part := range spec[0].Slice() {
		key.WriteByte(' ')
		if part.IsString() {
			key.WriteString(part.String())
		} else {
			if part.Int() < 0 || int(part.Int()) >= len(values) {
				return nil
			}
			v := values[part.Int()]
			switch {
			case v.IsNil():
				key.WriteString("nil")
			case v.IsBool():
				key.WriteString(strconv.FormatBool(v.Bool()))
			case v.IsInt():
				key.WriteString("i:" + strconv.FormatInt(v.Int(), 10))
			case v.IsFloat():
				key.WriteString("f:" + strconv.FormatFloat(v.Float(), 'g', -1, 64))
			case v.IsString():
				if len(v.String()) > filterFeedbackMaxKey {
					return nil
				}
				key.WriteString("s:" + strconv.Quote(v.String()))
			default:
				return nil
			}
		}
		if key.Len() > filterFeedbackMaxKey {
			return nil
		}
	}
	result := &filterObservation{key: key.String(), value: spec[3].Float()}
	patternSlot := int(spec[1].Int())
	if patternSlot >= 0 && patternSlot < len(values) && values[patternSlot].IsString() {
		pattern := values[patternSlot].String()
		core := strings.ReplaceAll(strings.ReplaceAll(pattern, "%", ""), "_", "")
		result.length = utf8.RuneCountInString(core)
		result.value = filterTextPrior(result.length)
		// Only simple exact/prefix/suffix/contains patterns generalize by length.
		// Internal wildcards and escaped patterns retain exact scalar feedback only.
		inner := strings.Trim(pattern, "%")
		if !strings.ContainsAny(inner, "%_\\") && core != "" {
			result.family = spec[2].String() + fmt.Sprintf(":%t:%t", strings.HasPrefix(pattern, "%"), strings.HasSuffix(pattern, "%"))
		}
	}
	return result
}

func filterTextPrior(length int) float64 {
	if length == 0 {
		return 1
	}
	return math.Max(.01, .7*math.Pow(.35, float64(length-1)))
}

func filterFeedbackSlot(key string) uint64 {
	return plannerFingerprintString(1469598103934665603, key) % filterFeedbackSlots
}

// Called exactly once after successful completion, never from an element loop.
// One failed CAS drops a sample instead of spinning against concurrent queries.
func (cache *filterFeedbackCache) observe(key *filterObservation, population, matched, candidates int64) {
	if key == nil || population <= 0 || matched < 0 || matched > population {
		return
	}
	cell := &cache[filterFeedbackSlot(key.key)]
	old := cell.Load()
	measured := float64(matched) / float64(population)
	work := math.Min(1, math.Max(measured, float64(candidates)/float64(population)))
	// Repeating the identical population/rate cannot improve the estimate. Do
	// not allocate or dirty a shared cache line just to count redundant samples.
	if old != nil && old.key == key.key && old.generation == key.generation && old.population == population && old.value == measured && old.workKnown && old.filterInput == work {
		return
	}
	next := *key
	next.population = population
	next.observed = population
	next.samples = 1
	// A complete first observation replaces the cold prior. Subsequent complete
	// observations use the requested 99/1 EMA; no per-batch weighting bias.
	next.value = measured
	next.filterInput, next.workKnown = work, true
	if old != nil && old.key == key.key && old.generation == key.generation {
		next.value = .99*old.value + .01*measured
		// Access work can change abruptly when an autoindex becomes effective.
		// Do not smear that change over 100 queries using the result-rate EMA.
		next.samples = old.samples
		if next.samples < 1000000 {
			next.samples++
		}
	}
	cell.CompareAndSwap(old, &next)
}

// Publish a row-weighted merge, not an average of shard percentages. Missing
// shard observations retain the original prior. Reading these immutable cells
// does not access shard containers or acquire shard locks/scan rights.
func (t *table) publishFilterFeedback(key *filterObservation) {
	if key == nil || key.generation == 0 || t.plannerStatsToken.Load() != key.generation {
		return
	}
	topology := t.topology.Load()
	if topology == nil {
		return
	}
	next := *key
	next.value = 0
	next.population = 0
	next.observed = 0
	next.samples = 0
	next.filterInput, next.workKnown = 0, false
	for _, shard := range topology.shards {
		if shard == nil {
			continue
		}
		population := int64(shard.plannerMainRows.Load()) + int64(shard.plannerDeltaRows.Load())
		if population <= 0 {
			continue
		}
		value := key.value
		work := 1.0 // Missing shards receive no speculative index discount.
		entry := shard.filterFeedback[filterFeedbackSlot(key.key)].Load()
		if entry != nil && entry.key == key.key && entry.generation == key.generation {
			value = entry.value
			population = entry.population
			next.observed += population
			next.samples += entry.samples
			if entry.workKnown {
				work = entry.filterInput
				next.workKnown = true
			}
		}
		next.value += float64(population) * value
		next.filterInput += float64(population) * work
		next.population += population
	}
	// Cold shards may not yet expose their main count. Retain the prior for
	// the remaining table population instead of extrapolating warm shards alone.
	if missing := int64(t.CountEstimate()) - next.population; missing > 0 {
		next.value += float64(missing) * key.value
		next.filterInput += float64(missing)
		next.population += missing
	}
	if next.population <= 0 || next.samples == 0 {
		return
	}
	next.value /= float64(next.population)
	next.filterInput /= float64(next.population)
	old := t.filterFeedback.Load()
	if old != nil && old.schema == t.filterSchemaFingerprint() {
		previous := old.entries[filterFeedbackSlot(key.key)]
		if previous != nil && *previous == next {
			return
		}
	}
	snapshot := &tableFilterFeedback{}
	if old != nil && old.schema == t.filterSchemaFingerprint() {
		*snapshot = *old
	}
	snapshot.schema = t.filterSchemaFingerprint()
	snapshot.entries[filterFeedbackSlot(key.key)] = &next
	snapshot.updateFingerprint()
	if t.plannerStatsToken.Load() == key.generation {
		t.filterFeedback.CompareAndSwap(old, snapshot)
	}
}

// Exact scalar lookup is O(1). Unknown LIKE values use at most 64 immutable
// entries to form length buckets. Different patterns in one bucket have equal
// weight: repeatedly querying one word must not make it the whole histogram.
func (t *table) filterSelectivity(key *filterObservation) (float64, string, bool) {
	if key == nil {
		return 0, "", false
	}
	generation := t.plannerStatsToken.Load()
	snapshot := t.filterFeedback.Load()
	if snapshot == nil || snapshot.schema != t.filterSchemaFingerprint() {
		return 0, "", false
	}
	entry := snapshot.entries[filterFeedbackSlot(key.key)]
	if entry != nil && entry.key == key.key {
		if entry.generation != generation {
			return entry.value, "historical_scan_feedback", true
		}
		if entry.observed < entry.population {
			return entry.value, "partial_scan_feedback", true
		}
		return entry.value, "scan_feedback", true
	}
	if key.family == "" {
		return 0, "", false
	}
	var sums [65]float64
	var counts [65]int
	for _, entry := range snapshot.entries {
		if entry == nil || entry.family != key.family || entry.length > 64 {
			continue
		}
		sums[entry.length] += entry.value
		counts[entry.length]++
	}
	lower, upper := -1, -1
	for i := range counts {
		if counts[i] == 0 {
			continue
		}
		if i <= key.length {
			lower = i
		}
		if i >= key.length && upper < 0 {
			upper = i
		}
	}
	value := func(i int) float64 { return sums[i] / float64(counts[i]) }
	if lower < 0 && upper < 0 {
		return 0, "", false
	}
	estimate := 0.0
	if lower == upper {
		estimate = value(lower)
	} else if lower >= 0 && upper >= 0 {
		// Linear interpolation admits zero-hit buckets without log(0).
		fraction := float64(key.length-lower) / float64(upper-lower)
		estimate = value(lower)*(1-fraction) + value(upper)*fraction
	} else {
		nearest := lower
		if nearest < 0 {
			nearest = upper
		}
		// Outside the observed range use a calibrated prior slope. Limit the reach
		// to four characters; a far-away length is not statistical evidence.
		distance := key.length - nearest
		if distance < -4 || distance > 4 {
			return 0, "", false
		}
		estimate = value(nearest) * math.Pow(.35, float64(distance))
	}
	return math.Max(0, math.Min(1, estimate)), "like_length_histogram", true
}

// Internal companion to the single public statistics reader. No shard access,
// and no inference of physical work from another predicate's LIKE histogram.
func (t *table) filterInputSelectivity(key *filterObservation) scm.Scmer {
	if key == nil {
		return scm.NewNil()
	}
	snapshot := t.filterFeedback.Load()
	if snapshot == nil || snapshot.schema != t.filterSchemaFingerprint() {
		return scm.NewNil()
	}
	entry := snapshot.entries[filterFeedbackSlot(key.key)]
	if entry == nil || entry.key != key.key || !entry.workKnown || entry.generation != t.plannerStatsToken.Load() {
		return scm.NewNil()
	}
	return scm.NewFloat(entry.filterInput)
}

// Geometric classes retain discrimination for very small selectivities. A
// changed class invalidates cached costing, while small EMA changes do not.
func filterFeedbackClass(value float64) uint64 {
	if value <= 0 {
		return 0
	}
	_, exponent := math.Frexp(value)
	return uint64(int64(exponent) + 2048)
}

func (t *table) filterFeedbackFingerprint() uint64 {
	if snapshot := t.filterFeedback.Load(); snapshot != nil && snapshot.schema == t.filterSchemaFingerprint() {
		return snapshot.fingerprint
	}
	return 0
}

// A wrapped header carries optional feedback and filter-readset metadata. The original
// integer header and boundary layout remain readable, including persisted plans.
func scanFeedbackMetadata(schema []scm.Scmer) scm.Scmer {
	if len(schema) > 0 && schema[0].IsSlice() {
		header := schema[0].Slice()
		if len(header) == 2 || len(header) == 3 {
			return header[1]
		}
	}
	return scm.NewNil()
}
func preserveScanAccessHeaderMetadata(header, old scm.Scmer) scm.Scmer {
	if old.IsSlice() && (len(old.Slice()) == 2 || len(old.Slice()) == 3) {
		fields := append([]scm.Scmer(nil), old.Slice()...)
		fields[0] = header
		return scm.NewSlice(fields)
	}
	return header
}

// Logical costing only needs a statistical identity. It must not compile/eval
// physical boundaries, which can contain correlated or effectful expressions.
// Literal-only scans reuse a serialized, compile-time identity. Dynamic session
// parameters still bind once per invocation. Neither path does key work in a
// shard loop, and the common literal path avoids building/hashing AST strings.
func staticFilterFeedbackKey(spec scm.Scmer, bindings []scm.Scmer) scm.Scmer {
	var expected []scm.Scmer
	for _, part := range spec.Slice()[0].Slice() {
		if !part.IsInt() {
			continue
		}
		value := bindings[part.Int()]
		expected = append(expected, value)
		if !value.IsNil() && !value.IsBool() && !value.IsInt() && !value.IsFloat() && !value.IsString() {
			return spec
		}
	}
	schema := []scm.Scmer{scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), spec})}
	key := bindFilterFeedback(schema, bindings)
	if key == nil {
		return spec
	}
	result := append([]scm.Scmer(nil), spec.Slice()...)
	result = append(result, scm.NewSlice([]scm.Scmer{scm.NewString(key.key), scm.NewString(key.family), scm.NewInt(int64(key.length)), scm.NewFloat(key.value), scm.NewSlice(expected)}))
	return scm.NewSlice(result)
}

func (s *tableFilterFeedback) updateFingerprint() {
	s.fingerprint = 0
	for _, entry := range s.entries {
		if entry != nil {
			s.fingerprint = plannerFingerprintString(s.fingerprint, entry.key)
			s.fingerprint = plannerFingerprintMix(s.fingerprint, filterFeedbackClass(entry.value))
			if entry.workKnown {
				s.fingerprint = plannerFingerprintMix(s.fingerprint, filterFeedbackClass(entry.filterInput)+1)
			}
			s.fingerprint = plannerFingerprintMix(s.fingerprint, plannerFractionBucket(float64(entry.observed)/float64(entry.population)))
		}
	}
}
