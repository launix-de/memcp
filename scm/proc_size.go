/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
package scm

import "unsafe"

// procSyntaxSize belongs only to private module-loader syntax, attested by
// the optimizer's lambda binding proof. Quoted data and callable/native payloads remain live values.
// This is an additive size decomposition, never an executable recipe.
type procSyntaxSize struct {
	params, body Scmer
	static       uint
	dynamic      []Scmer
}

func newProcSyntaxSize(params, body Scmer) *procSyntaxSize {
	result := &procSyntaxSize{params: params, body: body}
	var add func(Scmer)
	add = func(value Scmer) {
		switch value.GetTag() {
		case tagSlice:
			items := value.Slice()
			result.static += scmerStructOverhead
			if len(items) == 0 {
				return
			}
			result.static += goAllocOverhead
			if scmerIsSymbol(items[0], "quote") {
				add(items[0])
				result.dynamic = append(result.dynamic, items[1:]...)
				return
			}
			for _, item := range items {
				add(item)
			}
		case tagSourceInfo:
			info := value.SourceInfo()
			result.static += scmerStructOverhead + goAllocOverhead + 40
			if info.source != "" {
				result.static += align8(uint(len(info.source)))
			}
			add(info.value)
		case tagProc, tagJIT, tagAny, tagFastDict:
			result.dynamic = append(result.dynamic, value)
		default:
			result.static += ComputeSize(value)
		}
	}
	add(params)
	add(body)
	return result
}

func (s *procSyntaxSize) matches(p *Proc) bool {
	return s != nil && s.params == p.Params && s.body == p.Body
}

func (s *procSyntaxSize) ownSize() uint {
	result := goAllocOverhead + uint(unsafe.Sizeof(*s))
	if cap(s.dynamic) > 0 {
		result += goAllocOverhead + uint(cap(s.dynamic))*uint(unsafe.Sizeof(Scmer{}))
	}
	return result
}

func (s *procSyntaxSize) retainedSize() uint {
	result := s.static + s.ownSize()
	for _, value := range s.dynamic {
		result += ComputeSize(value)
	}
	return result
}
