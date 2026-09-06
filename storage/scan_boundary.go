/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "encoding/json"
import "fmt"
import "unsafe"

import "github.com/launix-de/memcp/scm"

// TagScanBoundary identifies immutable physical scan constraints. Boundary
// objects belong to the cached plan; invocation-specific data stays in the
// adjacent values array.
const TagScanBoundary = 103

// ScanBoundary is the immutable, Scheme-visible description of one physical
// access dimension. Slots address the scan's flat runtime values array. A
// negative slot means unbounded; slots <= -2 address a scan_batch input column.
// Implementations must be immutable because cached plans share them across
// concurrent executions.
type ScanBoundary interface {
	ColumnName() string
	Analyzer() IndexAnalyzer
	LowerSlot() int
	UpperSlot() int
	LowerInclusive() bool
	UpperInclusive() bool
	Collation() string
	NullSafe() bool
	MapperSlot() int
	MapColumns() []string
	Order() func(...scm.Scmer) scm.Scmer
	OrderMetadata() string
	Mandatory() bool
}

type scanBoundarySpec struct {
	column         string
	analyzer       IndexAnalyzer
	lowerSlot      int
	upperSlot      int
	lowerInclusive bool
	upperInclusive bool
	collation      string
	nullSafe       bool
	mapperSlot     int
	mapColumns     []string
	order          func(...scm.Scmer) scm.Scmer
	orderMetadata  string
	mandatory      bool
}

func (b *scanBoundarySpec) ColumnName() string                  { return b.column }
func (b *scanBoundarySpec) Analyzer() IndexAnalyzer             { return b.analyzer }
func (b *scanBoundarySpec) LowerSlot() int                      { return b.lowerSlot }
func (b *scanBoundarySpec) UpperSlot() int                      { return b.upperSlot }
func (b *scanBoundarySpec) LowerInclusive() bool                { return b.lowerInclusive }
func (b *scanBoundarySpec) UpperInclusive() bool                { return b.upperInclusive }
func (b *scanBoundarySpec) Collation() string                   { return b.collation }
func (b *scanBoundarySpec) NullSafe() bool                      { return b.nullSafe }
func (b *scanBoundarySpec) MapperSlot() int                     { return b.mapperSlot }
func (b *scanBoundarySpec) MapColumns() []string                { return b.mapColumns }
func (b *scanBoundarySpec) Order() func(...scm.Scmer) scm.Scmer { return b.order }
func (b *scanBoundarySpec) OrderMetadata() string               { return b.orderMetadata }
func (b *scanBoundarySpec) Mandatory() bool                     { return b.mandatory }

// scanBoundaryBox keeps the extensible interface behind one custom SCM tag.
// The box is allocated once while constructing a cached plan, never while a
// scan is executing.
type scanBoundaryBox struct {
	column         string
	analyzer       IndexAnalyzer
	lowerSlot      int
	upperSlot      int
	lowerInclusive bool
	upperInclusive bool
	collation      string
	nullSafe       bool
	mapperSlot     int
	mapColumns     []string
	order          func(...scm.Scmer) scm.Scmer
	orderMetadata  string
	mandatory      bool
}

func (b *scanBoundaryBox) ColumnName() string                  { return b.column }
func (b *scanBoundaryBox) Analyzer() IndexAnalyzer             { return b.analyzer }
func (b *scanBoundaryBox) LowerSlot() int                      { return b.lowerSlot }
func (b *scanBoundaryBox) UpperSlot() int                      { return b.upperSlot }
func (b *scanBoundaryBox) LowerInclusive() bool                { return b.lowerInclusive }
func (b *scanBoundaryBox) UpperInclusive() bool                { return b.upperInclusive }
func (b *scanBoundaryBox) Collation() string                   { return b.collation }
func (b *scanBoundaryBox) NullSafe() bool                      { return b.nullSafe }
func (b *scanBoundaryBox) MapperSlot() int                     { return b.mapperSlot }
func (b *scanBoundaryBox) MapColumns() []string                { return b.mapColumns }
func (b *scanBoundaryBox) Order() func(...scm.Scmer) scm.Scmer { return b.order }
func (b *scanBoundaryBox) OrderMetadata() string               { return b.orderMetadata }
func (b *scanBoundaryBox) Mandatory() bool                     { return b.mandatory }

func NewScanBoundaryScmer(boundary ScanBoundary) scm.Scmer {
	if boundary == nil || boundary.Analyzer() == nil {
		panic("scan boundary requires an analyzer")
	}
	box := &scanBoundaryBox{
		column: boundary.ColumnName(), analyzer: boundary.Analyzer(),
		lowerSlot: boundary.LowerSlot(), upperSlot: boundary.UpperSlot(),
		lowerInclusive: boundary.LowerInclusive(), upperInclusive: boundary.UpperInclusive(),
		collation: boundary.Collation(), nullSafe: boundary.NullSafe(), mapperSlot: boundary.MapperSlot(),
		mapColumns: boundary.MapColumns(), order: boundary.Order(), orderMetadata: boundary.OrderMetadata(),
		mandatory: boundary.Mandatory(),
	}
	return scm.NewCustom(TagScanBoundary, unsafe.Pointer(box))
}

func ScanBoundaryFromScmer(value scm.Scmer) ScanBoundary {
	box := (*scanBoundaryBox)(value.Custom(TagScanBoundary))
	if box == nil || box.analyzer == nil {
		panic("invalid scan boundary")
	}
	return box
}

func scanBoundaryAnalyzer(kind string) IndexAnalyzer {
	switch kind {
	case "equal":
		return EqualMatcher
	case "range":
		return RangeMatcher
	case "like":
		return LikeMatcher
	case "recset":
		return RecSetMatcher
	default:
		panic("unknown scan boundary kind " + kind)
	}
}

func scanBoundaryString(pointer unsafe.Pointer) string {
	box := (*scanBoundaryBox)(pointer)
	return fmt.Sprintf("(scan_boundary %q %q %d %d %t %t %q %t)",
		box.analyzer.Kind(), box.column, box.lowerSlot, box.upperSlot,
		box.lowerInclusive, box.upperInclusive, box.collation, box.nullSafe)
}

func scanBoundaryJSONEncode(pointer unsafe.Pointer) any {
	box := (*scanBoundaryBox)(pointer)
	if box.order != nil || box.orderMetadata != "" {
		panic("ordered runtime scan boundaries cannot be persisted")
	}
	mapColumns := box.mapColumns
	if mapColumns == nil {
		mapColumns = []string{}
	}
	return []any{
		box.analyzer.Kind(), box.column, box.lowerSlot, box.upperSlot,
		box.lowerInclusive, box.upperInclusive, box.collation, box.nullSafe,
		box.mapperSlot, mapColumns, box.mandatory,
	}
}

func scanBoundaryJSONInt(value any) int {
	switch number := value.(type) {
	case json.Number:
		result, err := number.Int64()
		if err == nil {
			return int(result)
		}
	case float64:
		return int(number)
	case int:
		return number
	}
	panic("invalid persisted scan boundary slot")
}

func scanBoundaryJSONDecode(value any) unsafe.Pointer {
	items, ok := value.([]any)
	if !ok || len(items) != 11 {
		panic("invalid persisted scan boundary")
	}
	kind, kindOK := items[0].(string)
	column, columnOK := items[1].(string)
	lowerInclusive, lowerOK := items[4].(bool)
	upperInclusive, upperOK := items[5].(bool)
	collation, collationOK := items[6].(string)
	nullSafe, nullSafeOK := items[7].(bool)
	mandatory, mandatoryOK := items[10].(bool)
	mapItems, mapOK := items[9].([]any)
	if !kindOK || !columnOK || !lowerOK || !upperOK || !collationOK || !nullSafeOK || !mandatoryOK || !mapOK {
		panic("invalid persisted scan boundary fields")
	}
	mapColumns := make([]string, len(mapItems))
	for i, item := range mapItems {
		column, ok := item.(string)
		if !ok {
			panic("invalid persisted scan boundary map column")
		}
		mapColumns[i] = column
	}
	return unsafe.Pointer(&scanBoundaryBox{
		column: column, analyzer: scanBoundaryAnalyzer(kind),
		lowerSlot: scanBoundaryJSONInt(items[2]), upperSlot: scanBoundaryJSONInt(items[3]),
		lowerInclusive: lowerInclusive, upperInclusive: upperInclusive,
		collation: collation, nullSafe: nullSafe,
		mapperSlot: scanBoundaryJSONInt(items[8]), mapColumns: mapColumns,
		mandatory: mandatory,
	})
}

func registerScanBoundaryFormats() {
	scm.CustomStringer[TagScanBoundary] = scanBoundaryString
	scm.CustomJSONCodecs[TagScanBoundary] = scm.CustomJSONCodec{
		Encode: scanBoundaryJSONEncode,
		Decode: scanBoundaryJSONDecode,
	}
}

func newScanBoundarySpec(column string, analyzer IndexAnalyzer, lowerSlot, upperSlot int,
	lowerInclusive, upperInclusive bool, collation string, nullSafe bool, mapperSlot int,
	mapColumns []string, order func(...scm.Scmer) scm.Scmer, orderMetadata string, mandatory bool,
) scm.Scmer {
	return NewScanBoundaryScmer(&scanBoundarySpec{
		column: column, analyzer: analyzer, lowerSlot: lowerSlot, upperSlot: upperSlot,
		lowerInclusive: lowerInclusive, upperInclusive: upperInclusive,
		collation: collation, nullSafe: nullSafe, mapperSlot: mapperSlot,
		mapColumns: mapColumns, order: order, orderMetadata: orderMetadata, mandatory: mandatory,
	})
}

func newExactScanAccessSchema(columns []string) []scm.Scmer {
	schema := make([]scm.Scmer, scanAccessSchemaHeaderSize+len(columns))
	schema[0] = newScanAccessHeader(len(columns), scanAccessConsumerScan, 0, -1)
	for i, column := range columns {
		schema[scanAccessSchemaHeaderSize+i] = newScanBoundarySpec(
			column, EqualMatcher, i, i, true, true, "", false, -1, nil, nil, "", false)
	}
	return schema
}

func exactScanAccess(schema []scm.Scmer, values []scm.Scmer) scanAccess {
	count := len(schema) - scanAccessSchemaHeaderSize
	access := scanAccess{schema: schema, values: values, compiledCount: count, exactAdjacent: true}
	if count > 0 {
		access.firstBoundary = (*scanBoundaryBox)(schema[scanAccessSchemaHeaderSize].Custom(TagScanBoundary))
	}
	return access
}

// scanAccessSegmentFromAnalyzed is the phase boundary for the few storage
// constraints that are still synthesized by Go (ORDER drivers, RecSets, and
// maintenance probes). The analyzer-only structs never enter scanAccess:
// execution receives the same Scheme boundary objects and adjacent values as a
// planner-compiled SQL scan.
func scanAccessSegmentFromAnalyzed(analyzed analyzedBoundaries) scanAccessSegment {
	if len(analyzed) == 0 {
		return scanAccessSegment{}
	}
	boxes := make([]scanBoundaryBox, len(analyzed))
	items := make([]scm.Scmer, len(analyzed))
	values := make([]scm.Scmer, 0, len(analyzed)*2)
	for i, boundary := range analyzed {
		lowerSlot := -1
		if boundary.lowerBatch {
			lowerSlot = -2 - boundary.lowerBatchSubidx
		} else if boundary.matcher != RangeMatcher || !boundary.lower.IsNil() {
			lowerSlot = len(values)
			values = append(values, boundary.lower)
		}
		upperSlot := -1
		if boundary.upperBatch {
			upperSlot = -2 - boundary.upperBatchSubidx
		} else if boundary.matcher != RangeMatcher || !boundary.upper.IsNil() {
			if lowerSlot >= 0 && boundaryValueEqual(boundary.lower, boundary.upper) {
				upperSlot = lowerSlot
			} else {
				upperSlot = len(values)
				values = append(values, boundary.upper)
			}
		}
		mapperSlot := -1
		if !boundary.mapFn.IsNil() {
			mapperSlot = len(values)
			values = append(values, scm.NewSlice([]scm.Scmer{scm.NewString(boundary.col), boundary.mapFn}))
		}
		boxes[i] = scanBoundaryBox{
			column: boundary.col, analyzer: boundary.matcher,
			lowerSlot: lowerSlot, upperSlot: upperSlot,
			lowerInclusive: boundary.lowerInclusive, upperInclusive: boundary.upperInclusive,
			collation: boundary.collation, nullSafe: boundary.nullSafe,
			mapperSlot: mapperSlot, mapColumns: boundary.mapCols,
			order: boundary.order, orderMetadata: boundary.orderMeta,
			mandatory: boundary.mandatory,
		}
		items[i] = scm.NewCustom(TagScanBoundary, unsafe.Pointer(&boxes[i]))
	}
	return scanAccessSegment{items: items, values: values}
}

func scanAccessFromAnalyzed(analyzed analyzedBoundaries) scanAccess {
	segment := scanAccessSegmentFromAnalyzed(analyzed)
	if len(segment.items) == 0 {
		return scanAccess{}
	}
	schema := make([]scm.Scmer, scanAccessSchemaHeaderSize+len(segment.items))
	schema[0] = newScanAccessHeader(len(segment.items), scanAccessConsumerScan, 0, -1)
	copy(schema[scanAccessSchemaHeaderSize:], segment.items)
	return scanAccess{
		schema: schema, values: segment.values, compiledCount: len(segment.items),
		firstBoundary: (*scanBoundaryBox)(segment.items[0].Custom(TagScanBoundary)),
	}
}

func scanAccessAsSegment(access scanAccess) scanAccessSegment {
	if access.runtime != nil {
		panic("nested runtime scan access cannot be embedded")
	}
	if access.compiledCount == 0 {
		return scanAccessSegment{}
	}
	return scanAccessSegment{
		items:  access.schema[scanAccessSchemaHeaderSize : scanAccessSchemaHeaderSize+access.compiledCount],
		values: access.values,
	}
}
