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
		Name: "stream_emit",

		Fn: func(a ...Scmer) Scmer {
			return Apply(a[0], a[1])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "invokes a streaming callback immediately; marks ordering-sensitive emission as an observable effect",
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "emit", Params: []*TypeDescriptor{{Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "value"},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "stream_window_reduce",

		Fn: func(a ...Scmer) (result Scmer) {
			offset := ToInt(a[0])
			limit := ToInt(a[1])
			if offset < 0 {
				panic("stream_window_reduce: offset must not be negative")
			}
			if limit < -1 {
				panic("stream_window_reduce: limit must be -1 or non-negative")
			}
			result = a[3]
			if limit == 0 {
				return result
			}
			reduceFn := OptimizeProcToSerialFunction(a[2])
			producerFn := OptimizeProcToSerialFunction(a[4])
			seen := 0
			emitted := 0
			emit := NewFunc(func(values ...Scmer) Scmer {
				if len(values) != 1 {
					panic("stream_window_reduce: emit expects exactly one complete value")
				}
				seen++
				if seen <= offset {
					return result
				}
				if limit >= 0 && emitted >= limit {
					return result
				}
				result = reduceFn(result, values[0])
				emitted++
				return result
			})
			producerFn(emit)
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "applies OFFSET/LIMIT and a serial reducer to complete values emitted by a nested streaming producer without collecting an intermediate relation",
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "offset", Description: "number of complete producer values to skip"},
				{Kind: "number", Label: "limit", Description: "maximum values to reduce, or -1 for no limit"},
				{Kind: "func", Label: "reduce", Description: "serial accumulator over complete values", Params: []*TypeDescriptor{{Kind: "any", Label: "acc"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "neutral", Description: "initial accumulator"},
				{Kind: "func", Label: "producer", Description: "nested streaming plan called with a one-value emit callback", Params: []*TypeDescriptor{{Kind: "func", Label: "emit", Description: "emits one complete value", Params: []*TypeDescriptor{{Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:  &TypeDescriptor{Kind: "any"},
			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "window_mut",

		Fn: func(a ...Scmer) Scmer {
			win := asSlice(a[0], "window_mut")
			emitFn := a[1]
			vals := asSlice(a[2], "window_mut vals")

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
				if i < len(vals) {
					slots[writePos+i] = vals[i]
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
		Type: &TypeDescriptor{Kind: "func", Description: "Ring buffer shift-insert for window functions. (window_mut window emit_fn vals) writes vals (a list of stride values) into the current slot, increments counter. If skip>0, decrements skip. Otherwise calls (emit_fn oldest_v0 oldest_v1 ... newest_v0 newest_v1) with all slot values ordered oldest-to-newest. Returns updated window.",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "list", Label: "window", Description: "ring buffer accumulator"}, &TypeDescriptor{Kind: "func", Label: "emit_fn", Description: "callback receiving all window values oldest-to-newest", Params: []*TypeDescriptor{{Kind: "any", Label: "values", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}, &TypeDescriptor{Kind: "list", Label: "vals", Description: "list of stride values to insert"}},
			Return: &TypeDescriptor{Kind: "list"},

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "window_flush",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "Flush remaining window buffer by shifting in nils. (window_flush window emit_fn count) shifts in count positions of nils, calling emit_fn for each displaced position. Returns nil.",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "list", Label: "window", Description: "ring buffer accumulator"}, &TypeDescriptor{Kind: "func", Label: "emit_fn", Description: "callback receiving all window values oldest-to-newest", Params: []*TypeDescriptor{{Kind: "any", Label: "values", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}, &TypeDescriptor{Kind: "number", Label: "count", Description: "number of nil positions to shift in"}},
			Return: &TypeDescriptor{Kind: "nil"},

			JITEmit: nil,
		},
	})
}
