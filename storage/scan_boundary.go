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
	boundary       ScanBoundary
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

func NewScanBoundaryScmer(boundary ScanBoundary) scm.Scmer {
	if boundary == nil || boundary.Analyzer() == nil {
		panic("scan boundary requires an analyzer")
	}
	box := &scanBoundaryBox{
		boundary: boundary, column: boundary.ColumnName(), analyzer: boundary.Analyzer(),
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
	if box == nil || box.boundary == nil {
		panic("invalid scan boundary")
	}
	return box.boundary
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
	return scanAccess{schema: schema, values: values, compiledCount: count}
}
