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

package scm

/*
 Window ring buffer helpers for LEAD/LAG window functions.

 Accumulator layout (flat list):
   (skip_count counter stride slot_0_v0 slot_0_v1 ... slot_N_vM)

 - skip_count: rows to skip before first emit (LEAD offset, 0 for LAG)
 - counter: monotonic write position
 - stride: number of values per slot
 - slots: window_size * stride values

 window_mut writes vals into the current slot, increments counter,
 and either decrements skip or calls emit_fn with all slot values
 ordered oldest-to-newest.

 window_flush shifts in count positions of nils, emitting each time.
*/

func init_window() {
	DeclareTitle("Window Functions")

	Declare(&Globalenv, &Declaration{
		"window_mut", "Ring buffer shift-insert for window functions. (window_mut window emit_fn vals...) writes vals into the current slot, increments counter. If skip>0, decrements skip. Otherwise calls (emit_fn oldest_v0 oldest_v1 ... newest_v0 newest_v1) with all slot values ordered oldest-to-newest. Returns updated window.",
		2, 1000,
		[]DeclarationParameter{
			{"window", "list", "ring buffer accumulator", nil},
			{"emit_fn", "func", "callback receiving all window values oldest-to-newest", nil},
			{"vals...", "any", "values to insert (stride count, default nil)", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			win := asSlice(a[0], "window_mut")
			emitFn := a[1]
			// vals are a[2], a[3], ... (stride values)

			if len(win) < 3 {
				panic("window_mut: window must have at least 3 elements (skip, counter, stride)")
			}

			skip := int(win[0].Int())
			counter := int(win[1].Int())
			stride := int(win[2].Int())
			slots := win[3:] // flat: window_size * stride values
			windowSize := len(slots) / stride

			if windowSize == 0 || stride == 0 {
				panic("window_mut: invalid window dimensions")
			}

			// write vals into current slot
			writePos := (counter % windowSize) * stride
			for i := 0; i < stride; i++ {
				if 2+i < len(a) { // a[2+i] = vals[i]
					slots[writePos+i] = a[2+i]
				} else {
					slots[writePos+i] = NewNil()
				}
			}
			counter++

			// build result window
			result := make([]Scmer, len(win))
			if skip > 0 {
				result[0] = NewInt(int64(skip - 1))
			} else {
				result[0] = NewInt(0)
			}
			result[1] = NewInt(int64(counter))
			result[2] = NewInt(int64(stride))
			copy(result[3:], slots)

			// emit if not skipping
			if skip <= 0 {
				// build args: all values oldest-to-newest
				args := make([]Scmer, len(slots))
				for i := 0; i < windowSize; i++ {
					srcPos := ((counter + i) % windowSize) * stride
					dstPos := i * stride
					for j := 0; j < stride; j++ {
						args[dstPos+j] = slots[srcPos+j]
					}
				}
				Apply(emitFn, args...)
			}

			return NewSlice(result)
		},
		false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"window_flush", "Flush remaining window buffer by shifting in nils. (window_flush window emit_fn count) shifts in count positions of nils, calling emit_fn for each displaced position. Returns nil.",
		3, 3,
		[]DeclarationParameter{
			{"window", "list", "ring buffer accumulator", nil},
			{"emit_fn", "func", "callback receiving all window values oldest-to-newest", nil},
			{"count", "number", "number of nil positions to shift in", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			win := asSlice(a[0], "window_flush")
			emitFn := a[1]
			count := int(a[2].Int())

			if len(win) < 3 {
				panic("window_flush: window must have at least 3 elements")
			}

			counter := int(win[1].Int())
			stride := int(win[2].Int())
			slots := make([]Scmer, len(win)-3)
			copy(slots, win[3:])
			windowSize := len(slots) / stride

			for n := 0; n < count; n++ {
				// write nils into current slot
				writePos := (counter % windowSize) * stride
				for i := 0; i < stride; i++ {
					slots[writePos+i] = NewNil()
				}
				counter++

				// build args: all values oldest-to-newest
				args := make([]Scmer, len(slots))
				for i := 0; i < windowSize; i++ {
					srcPos := ((counter + i) % windowSize) * stride
					dstPos := i * stride
					for j := 0; j < stride; j++ {
						args[dstPos+j] = slots[srcPos+j]
					}
				}
				Apply(emitFn, args...)
			}

			return NewNil()
		},
		false, false, nil,
	})
}
