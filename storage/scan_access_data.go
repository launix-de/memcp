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

import "github.com/launix-de/memcp/scm"

func initScanAccessData(en *scm.Env) {
	declare := func(name string, fn func(...scm.Scmer) scm.Scmer, description string) {
		scm.Declare(en, &scm.Declaration{Name: name, Fn: fn, Type: &scm.TypeDescriptor{
			Kind: "func", Description: description,
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "arguments", Variadic: true}},
			Return: &scm.TypeDescriptor{Kind: "any"},
		}})
	}
	declare("scan_access_schema", func(a ...scm.Scmer) scm.Scmer {
		boundaries, projections := a[0].Slice(), a[1].Slice()
		if len(boundaries) == 0 && len(projections) == 0 && a[2].IsNil() {
			return scm.NewSlice(nil)
		}
		consumer := scanAccessConsumerScan
		if scm.ToBool(a[3]) {
			consumer = scanAccessConsumerCoveredScan
		}
		header := newScanAccessHeader(len(boundaries), consumer, len(projections), -1)
		if !a[2].IsNil() {
			header = scm.NewSlice([]scm.Scmer{header, a[2]})
		}
		result := make([]scm.Scmer, 1, 1+len(boundaries)+len(projections))
		result[0] = header
		result = append(result, boundaries...)
		return scm.NewSlice(append(result, projections...))
	}, "packs explicit boundaries, projections, feedback metadata and coverage into an immutable operator argument")
	declare("scan_access_schema?", func(a ...scm.Scmer) scm.Scmer {
		if !a[0].IsSlice() || len(a[0].Slice()) == 0 {
			return scm.NewBool(false)
		}
		_, ok := decodeScanAccessHeader(a[0].Slice()[0])
		return scm.NewBool(ok)
	}, "reports whether a value contains a physical access header")
	declare("scan_access_shift", func(a ...scm.Scmer) scm.Scmer {
		return shiftCompiledScanAccessSlots(a[0], scm.ToInt(a[1]))
	}, "relocates value slots in an immutable physical access schema")
	declare("scan_access_cover", func(a ...scm.Scmer) scm.Scmer {
		if !a[0].IsSlice() || len(a[0].Slice()) == 0 {
			return a[0]
		}
		items := append([]scm.Scmer(nil), a[0].Slice()...)
		meta, ok := decodeScanAccessHeader(items[0])
		if !ok {
			panic("invalid scan access schema")
		}
		consumer := scanAccessConsumerScan
		if scm.ToBool(a[1]) {
			consumer = scanAccessConsumerCoveredScan
		}
		items[0] = preserveScanFeedbackHeader(newScanAccessHeader(meta.count, consumer, meta.projections, meta.mapperSlot), items[0])
		return scm.NewSlice(items)
	}, "records planner-proven residual coverage in a physical schema")
	declare("scan_feedback_key", func(a ...scm.Scmer) scm.Scmer {
		return staticFilterFeedbackKey(a[0], a[1].Slice())
	}, "prebinds explicit statistics identity tokens to literal values")
}
