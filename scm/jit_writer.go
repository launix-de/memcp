/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
package scm

import "unsafe"

// JITPage represents one page of mmap'd executable memory.
type JITPage struct {
	RwBase unsafe.Pointer // writable mapping
	RxBase unsafe.Pointer // executable mapping
	Next   *JITPage
}

// JITWriter is the platform-independent code emitter scaffold.
// Architecture-specific emit methods are defined in jit_<arch>.go files.
type JITWriter struct {
	Ptr     unsafe.Pointer // current write pointer (into mmap memory)
	End     unsafe.Pointer // page end minus reserve
	Start   unsafe.Pointer // page start for position calculation
	Pages   []*JITPage
	Current *JITPage

	Labels    [64]int32
	LabelNext uint8

	Fixups    [128]JITFixup
	FixupNext uint8
}

// DefineLabel allocates a new label at the current write position.
func (w *JITWriter) DefineLabel() uint8 {
	id := w.LabelNext
	w.LabelNext++
	w.Labels[id] = int32(uintptr(w.Ptr) - uintptr(w.Start))
	return id
}

// ReserveLabel allocates a label ID for later placement via MarkLabel.
func (w *JITWriter) ReserveLabel() uint8 {
	id := w.LabelNext
	w.LabelNext++
	w.Labels[id] = -1 // undefined until MarkLabel
	return id
}

// MarkLabel sets the position of a previously reserved label.
func (w *JITWriter) MarkLabel(id uint8) {
	w.Labels[id] = int32(uintptr(w.Ptr) - uintptr(w.Start))
}

// AddFixup records a forward reference to be patched by ResolveFixups.
func (w *JITWriter) AddFixup(labelID uint8, size uint8, relative bool) {
	w.Fixups[w.FixupNext] = JITFixup{
		CodePos:  int32(uintptr(w.Ptr) - uintptr(w.Start)),
		LabelID:  labelID,
		Size:     size,
		Relative: relative,
	}
	w.FixupNext++
}

// ResolveFixups patches all recorded forward references after code generation.
func (w *JITWriter) ResolveFixups() {
	for i := uint8(0); i < w.FixupNext; i++ {
		f := &w.Fixups[i]
		targetPos := w.Labels[f.LabelID]
		if targetPos < 0 {
			panic("jit: undefined label")
		}
		patchAddr := unsafe.Add(w.Start, int(f.CodePos))
		if f.Relative {
			offset := targetPos - (f.CodePos + int32(f.Size))
			*(*int32)(patchAddr) = offset
		} else {
			*(*int32)(patchAddr) = targetPos
		}
	}
}
