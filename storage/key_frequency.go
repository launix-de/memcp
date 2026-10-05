/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"math"
	"sort"

	"strconv"
	"strings"

	"sync/atomic"

	"encoding/json"

	"github.com/launix-de/memcp/scm"
)

// Misra-Gries retains bounded numeric heavy hitters during maintenance. Removed
// counts provide an explicit error bound; these hints never prove membership.
const keyFrequencySlots = 64
const keyFrequencyTop = 32
const keyFrequencyMinimumRows = 4096

type keyFrequencyCollector struct {
	counts  map[float64]uint64
	removed uint64
	rows    uint64
	nonnull uint64
	invalid bool
}
type keyFrequencyEntry struct {
	key   float64
	count float64
}
type keyFrequencyStatistics struct {
	historical bool
	top        []keyFrequencyEntry
	rows       uint64
	nonnull    uint64
	distinct   uint64 // exact only when no counters were removed
	removed    uint64
}

func numericFrequencyKey(v scm.Scmer) (float64, bool) {
	if v.IsInt() && v.Int() >= -(1<<53) && v.Int() <= 1<<53 {
		return float64(v.Int()), true
	}
	if v.IsFloat() && !math.IsNaN(v.Float()) && !math.IsInf(v.Float(), 0) {
		return v.Float(), true
	}
	return 0, false
}
func (c *keyFrequencyCollector) observe(v scm.Scmer) {
	c.rows++
	if v.IsNil() {
		return
	}
	c.nonnull++
	if c.invalid {
		return
	}
	key, ok := numericFrequencyKey(v)
	if !ok {
		c.invalid = true
		c.counts = nil
		return
	}
	if c.counts == nil {
		c.counts = make(map[float64]uint64, keyFrequencySlots)
	}
	if n, ok := c.counts[key]; ok {
		c.counts[key] = n + 1
		return
	}
	if len(c.counts) < keyFrequencySlots {
		c.counts[key] = 1
		return
	}
	for key, n := range c.counts {
		if n == 1 {
			delete(c.counts, key)
		} else {
			c.counts[key] = n - 1
		}
	}
	c.removed++
}
func (c *keyFrequencyCollector) finish() *keyFrequencyStatistics {
	if c.invalid || c.rows == 0 {
		return nil
	}
	s := &keyFrequencyStatistics{rows: c.rows, nonnull: c.nonnull, distinct: uint64(len(c.counts)), removed: c.removed}
	for key, n := range c.counts {
		// Residual counters below the cancellation error can be arbitrary late
		// arrivals, particularly for unique columns. Do not turn them into heavy
		// keys by adding the midpoint error; use the tail prior instead.
		if c.removed != 0 && n <= c.removed {
			continue
		}
		s.top = append(s.top, keyFrequencyEntry{key, float64(n) + float64(c.removed)/2})
	}
	sort.Slice(s.top, func(i, j int) bool {
		if s.top[i].count == s.top[j].count {
			return s.top[i].key < s.top[j].key
		}
		return s.top[i].count > s.top[j].count
	})
	if len(s.top) > keyFrequencyTop {
		s.top = append([]keyFrequencyEntry(nil), s.top[:keyFrequencyTop]...)
	}
	return s
}
func (s *keyFrequencyStatistics) frequency(v scm.Scmer, ndv uint64) (float64, bool) {
	if v.IsNil() {
		return float64(s.rows-s.nonnull) / float64(s.rows), true
	}
	key, ok := numericFrequencyKey(v)
	if !ok {
		return 0, false
	}
	var topRows float64
	for _, e := range s.top {
		if e.key == key {
			return math.Min(1, e.count/float64(s.rows)), true
		}
		topRows += e.count
	}
	return s.tailFrequency(ndv, topRows), true
}

// Tail NDV is an upper-bound prior unless the bounded collector saw the
// complete domain. NULL is separate from both the numeric top keys and tail.
func (s *keyFrequencyStatistics) tailFrequency(ndv uint64, topRows float64) float64 {
	if s.removed == 0 {
		ndv = s.distinct
	} else if s.nonnull < s.rows && ndv > 0 {
		ndv--
	}
	tailDistinct := math.Max(1, float64(ndv)-float64(len(s.top)))
	return math.Max(0, float64(s.nonnull)-topRows) / tailDistinct / float64(s.rows)
}

// The projected keyset is already deduplicated and has a numeric lookup.
// Summation is bounded by the retained top keys, independent of grant count.
func (s *keyFrequencyStatistics) keySetFraction(keys recSetProjectKeys, ndv uint64) float64 {
	var topRows float64
	for _, e := range s.top {
		topRows += e.count
	}
	tail := s.tailFrequency(ndv, topRows)
	result := float64(keys.count()) * tail
	for _, e := range s.top {
		integer, ok := projectNumericInteger(scm.NewFloat(e.key))
		if !ok {
			continue
		}
		if _, found := keys.numericValues[integer]; found {
			result += e.count/float64(s.rows) - tail
		}
	}
	return math.Max(0, math.Min(1, result))
}

// Snapshot reads use immutable metadata only, including when storage is cold.
func (t *table) keyFrequency(column string) (*keyFrequencyStatistics, uint64) {
	snap := t.showColumnsSnapshot.Load()
	if snap == nil || snap.metadata == nil || snap.metadata.columns == nil {
		return nil, 0
	}
	meta := snap.metadata.columns
	for i, name := range meta.names {
		if name == column {
			s := meta.plannerStatistics[i]
			if s != nil {
				return s.KeyFrequency, meta.distinctEstimates[i]
			}
			break
		}
	}
	return nil, 0
}

// Accept only a complete scalar equality expression. Boundary coverage alone
// cannot justify treating a conjunct as the complete residual output rate.
func (t *table) keyFrequencySelectivity(schema, values []scm.Scmer) (float64, string, bool) {
	spec, ok := scmerSlice(scanFeedbackMetadata(schema))
	if !ok || len(spec) < 4 {
		return 0, "", false
	}
	parts, ok := scmerSlice(spec[0])
	if !ok || len(parts) != 4 || !parts[0].IsString() || !parts[3].IsString() || parts[3].String() != ")" {
		return 0, "", false
	}
	switch parts[0].String() {
	case "(equal?", "(equal??":
	default:
		return 0, "", false
	}
	columnPart, slot := parts[1], parts[2]
	if columnPart.IsInt() {
		columnPart, slot = slot, columnPart
	}
	if !columnPart.IsString() || !slot.IsInt() || slot.Int() < 0 || int(slot.Int()) >= len(values) {
		return 0, "", false
	}
	columnToken := columnPart.String()
	if len(columnToken) < 9 || columnToken[:7] != "column:" {
		return 0, "", false
	}
	// Tokens use Go quoting. Decode the column without interpreting predicate text.
	column, err := strconv.Unquote(columnToken[7:])
	if err != nil {
		return 0, "", false
	}
	s, ndv := t.keyFrequency(column)
	if s == nil {
		return 0, "", false
	}
	source := "numeric_key_frequency"
	if s.historical {
		source = "historical_numeric_key_frequency"
	}
	value := values[slot.Int()]
	if value.IsNil() {
		if parts[0].String() == "(equal??" {
			return 0, source, true // SQL equality rejects NULL on either side.
		}
		// Scheme equality matches NULL and falsey numeric zero. Only numeric
		// domains are summarized, so no other falsey primitive can be present.
		value = scm.NewInt(0)
	}
	rate, known := s.frequency(value, ndv)
	if key, numeric := numericFrequencyKey(value); known && numeric && key == 0 && parts[0].String() == "(equal?" {
		rate = math.Min(1, rate+float64(s.rows-s.nonnull)/float64(s.rows))
	}
	return rate, source, known
}

// Disposable, versioned hints checkpointed with schema metadata. Never persist
// shard identities or query bindings; missing/invalid hints mean unknown.
type persistedKeyFrequencies struct {
	Version int                        `json:"version"`
	Schema  uint64                     `json:"schema"`
	Columns []persistedColumnFrequency `json:"columns"`
}
type persistedColumnFrequency struct {
	Column        string       `json:"column"`
	Rows          uint64       `json:"rows"`
	Nonnull       uint64       `json:"nonnull"`
	Distinct      uint64       `json:"distinct"`
	ExactDistinct uint64       `json:"exact_distinct,omitempty"`
	Removed       uint64       `json:"removed,omitempty"`
	Top           [][2]float64 `json:"top"`
}

func (p *persistedKeyFrequencies) UnmarshalJSON(data []byte) error {
	*p = persistedKeyFrequencies{}
	if len(data) > 1024*1024 {
		return nil
	}
	type plain persistedKeyFrequencies
	var decoded plain
	if json.Unmarshal(data, &decoded) != nil || decoded.Version != 1 || len(decoded.Columns) > 1024 {
		return nil
	}
	*p = persistedKeyFrequencies(decoded)
	return nil
}
func (t *table) persistKeyFrequencies() *persistedKeyFrequencies {
	if t.PersistencyMode == Memory || t.PersistencyMode == Cache || strings.HasPrefix(t.Name, ".") {
		return nil
	}
	snapshot := t.showColumnsSnapshot.Load()
	if snapshot == nil || snapshot.metadata == nil || snapshot.metadata.columns == nil {
		return nil
	}
	result := &persistedKeyFrequencies{Version: 1, Schema: snapshot.filterSchema}
	columns := snapshot.metadata.columns
	for i, name := range columns.names {
		stats := columns.plannerStatistics[i]
		if stats == nil || stats.KeyFrequency == nil {
			continue
		}
		s := stats.KeyFrequency
		if s.rows < keyFrequencyMinimumRows {
			continue
		}
		e := persistedColumnFrequency{Column: name, Rows: s.rows, Nonnull: s.nonnull, Distinct: columns.distinctEstimates[i], Removed: s.removed}
		if s.removed == 0 {
			e.ExactDistinct = s.distinct
		}
		for _, entry := range s.top {
			e.Top = append(e.Top, [2]float64{entry.key, entry.count})
		}
		result.Columns = append(result.Columns, e)
	}
	if len(result.Columns) == 0 || len(result.Columns) > 1024 {
		return nil
	}
	return result
}

// Startup only, before publishing the database. Restored hints remain historical;
// restoring them must not cause any column/shard load or synthesize observations.
func (t *table) restoreKeyFrequencies() {
	p := t.RestoredKeyFrequencies
	t.RestoredKeyFrequencies = nil
	if p == nil || p.Version != 1 || p.Schema != t.filterSchemaFingerprint() || t.PersistencyMode == Memory || t.PersistencyMode == Cache || strings.HasPrefix(t.Name, ".") {
		return
	}
	for _, e := range p.Columns {
		if e.Rows < keyFrequencyMinimumRows || e.Nonnull > e.Rows || e.Distinct > e.Rows || e.Removed > e.Nonnull/(keyFrequencySlots+1) || len(e.Top) > keyFrequencyTop || e.ExactDistinct > keyFrequencySlots || e.ExactDistinct > e.Nonnull {
			continue
		}
		if e.Removed == 0 && (e.ExactDistinct < uint64(len(e.Top)) || (e.Nonnull > 0 && e.ExactDistinct == 0)) {
			continue
		}
		s := &keyFrequencyStatistics{rows: e.Rows, nonnull: e.Nonnull, removed: e.Removed, historical: true}
		s.distinct = e.ExactDistinct
		valid := true
		var total float64
		for i, entry := range e.Top {
			key, count := entry[0], entry[1]
			if math.IsNaN(key) || math.IsInf(key, 0) || math.IsNaN(count) || math.IsInf(count, 0) || count <= 0 || count > float64(e.Nonnull) {
				valid = false
				break
			}
			for j := 0; j < i; j++ {
				if e.Top[j][0] == key {
					valid = false
					break
				}
			}
			total += count
			s.top = append(s.top, keyFrequencyEntry{key, count})
		}
		if !valid || total > float64(e.Nonnull) {
			continue
		}
		for _, column := range t.Columns {
			if column.Name != e.Column || column.IsTemp || !column.Computor.IsNil() || len(column.OrcSortCols) > 0 {
				continue
			}
			stats := &columnPlannerStatistics{KeyFrequency: s, Confidence: .35, Source: "historical_key_frequency", NullCount: e.Rows - e.Nonnull, NullFraction: float64(e.Rows-e.Nonnull) / float64(e.Rows), MinEstimate: scm.NewNil(), MaxEstimate: scm.NewNil()}
			atomic.StoreUint64(&column.DistinctEstimate, e.Distinct)
			column.PlannerStats.Store(stats)
			break
		}
	}
	t.publishShowColumnsSnapshot()
}
