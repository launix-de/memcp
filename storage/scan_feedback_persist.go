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

	"strings"

	"encoding/json"
)

// Optional, bounded planning hints. They are checkpointed with schema metadata,
// never written by a scan. Runtime generations and shard EMA state are not durable.
type persistedFilterFeedback struct {
	Version int                          `json:"version"`
	Schema  uint64                       `json:"schema"`
	Entries []persistedFilterObservation `json:"entries"`
}
type persistedFilterObservation struct {
	Key        string  `json:"key"`
	Family     string  `json:"family,omitempty"`
	Length     int     `json:"length,omitempty"`
	Value      float64 `json:"value"`
	Population int64   `json:"population"`
	Observed   int64   `json:"observed"`
	Samples    uint32  `json:"samples"`
}

// Invalid optional hints must not prevent loading an otherwise readable database.
func (p *persistedFilterFeedback) UnmarshalJSON(data []byte) error {
	*p = persistedFilterFeedback{}
	if len(data) > 256*1024 {
		return nil
	}
	type plain persistedFilterFeedback
	var decoded plain
	if json.Unmarshal(data, &decoded) != nil || decoded.Version != 1 || len(decoded.Entries) > filterFeedbackSlots {
		return nil
	}
	*p = persistedFilterFeedback(decoded)
	return nil
}

func (t *table) filterSchemaFingerprint() uint64 {
	if snapshot := t.showColumnsSnapshot.Load(); snapshot != nil {
		return snapshot.filterSchema
	}
	return 0
}

func (t *table) persistFilterFeedback() *persistedFilterFeedback {
	if t.PersistencyMode == Memory || t.PersistencyMode == Cache || strings.HasPrefix(t.Name, ".") {
		return nil
	}
	snapshot := t.filterFeedback.Load()
	if snapshot == nil || snapshot.schema != t.filterSchemaFingerprint() {
		return nil
	}
	result := &persistedFilterFeedback{Version: 1, Schema: snapshot.schema}
	for _, e := range snapshot.entries {
		if e != nil {
			result.Entries = append(result.Entries, persistedFilterObservation{e.key, e.family, e.length, e.value, e.population, e.observed, e.samples})
		}
	}
	if len(result.Entries) == 0 {
		return nil
	}
	return result
}

// Called during schema loading before publishing the table to readers. Only the
// aggregate is restored: copying a global average into each shard would invent
// local measurements. Historical values deliberately carry lower confidence.
func (t *table) restoreFilterFeedback() {
	p := t.RestoredFilterFeedback
	t.RestoredFilterFeedback = nil
	if p == nil || p.Version != 1 || p.Schema != t.filterSchemaFingerprint() || len(p.Entries) > filterFeedbackSlots || t.PersistencyMode == Memory || t.PersistencyMode == Cache || strings.HasPrefix(t.Name, ".") {
		return
	}
	snapshot := &tableFilterFeedback{schema: p.Schema}
	for _, e := range p.Entries {
		if len(e.Key) == 0 || len(e.Key) > filterFeedbackMaxKey || len(e.Family) > filterFeedbackMaxKey || e.Length < 0 || e.Length > 64 || math.IsNaN(e.Value) || math.IsInf(e.Value, 0) || e.Value < 0 || e.Value > 1 || e.Population <= 0 || e.Observed < 0 || e.Observed > e.Population || e.Samples == 0 {
			continue
		}
		snapshot.entries[filterFeedbackSlot(e.Key)] = &filterObservation{key: e.Key, family: e.Family, length: e.Length, value: e.Value, population: e.Population, observed: e.Observed, samples: e.Samples}
	}
	snapshot.updateFingerprint()
	t.filterFeedback.Store(snapshot)
}
