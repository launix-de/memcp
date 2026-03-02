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

// JITFactsUnknown returns fully unknown type facts.
func JITFactsUnknown() JITTypeFacts {
	return JITTypeFacts{Possible: JITTagMaskAny}
}

// JITFactsKnownTag returns facts for a value with a known single tag.
func JITFactsKnownTag(tag uint16) JITTypeFacts {
	mask := jitTagToMask(tag)
	if mask == 0 {
		return JITFactsUnknown()
	}
	return JITTypeFacts{Possible: mask}
}

// JITFactsConst returns exact facts for a compile-time constant Scmer.
func JITFactsConst(v Scmer) JITTypeFacts {
	return JITTypeFacts{
		Possible: jitTagToMask(v.GetTag()),
		HasConst: true,
		Const:    v,
	}
}

// IsSingleTag reports whether exactly one tag is possible.
func (f JITTypeFacts) IsSingleTag() bool {
	if f.Possible == 0 {
		return false
	}
	return (f.Possible & (f.Possible - 1)) == 0
}

// MayBeTag reports whether the given tag is still possible.
func (f JITTypeFacts) MayBeTag(tag uint16) bool {
	mask := jitTagToMask(tag)
	if mask == 0 {
		return false
	}
	return (f.Possible & mask) != 0
}

// RefineToTag intersects facts with a known tag branch.
func (f JITTypeFacts) RefineToTag(tag uint16) JITTypeFacts {
	mask := jitTagToMask(tag)
	if mask == 0 {
		return JITFactsUnknown()
	}
	refined := f
	refined.Possible &= mask
	if refined.Possible == 0 {
		refined.Possible = mask
	}
	if refined.HasConst && refined.Const.GetTag() != tag {
		refined.HasConst = false
		refined.Const = Scmer{}
	}
	return refined
}

// RefineNotTag intersects facts with the negative branch (tag != X).
func (f JITTypeFacts) RefineNotTag(tag uint16) JITTypeFacts {
	mask := jitTagToMask(tag)
	if mask == 0 {
		return f
	}
	refined := f
	refined.Possible &^= mask
	if refined.Possible == 0 {
		refined.Possible = JITTagMaskAny &^ mask
		if refined.Possible == 0 {
			refined.Possible = JITTagMaskAny
		}
	}
	if refined.HasConst && refined.Const.GetTag() == tag {
		refined.HasConst = false
		refined.Const = Scmer{}
	}
	return refined
}

// Join merges branch facts at a CFG join point.
func (f JITTypeFacts) Join(other JITTypeFacts) JITTypeFacts {
	out := JITTypeFacts{
		Possible: f.Possible | other.Possible,
	}
	if out.Possible == 0 {
		out.Possible = JITTagMaskAny
	}
	if f.HasConst && other.HasConst && f.Const == other.Const {
		out.HasConst = true
		out.Const = f.Const
	}
	return out
}

func jitTagToMask(tag uint16) JITTagMask {
	switch tag {
	case tagNil:
		return JITTagMaskNil
	case tagBool:
		return JITTagMaskBool
	case tagInt:
		return JITTagMaskInt
	case tagFloat:
		return JITTagMaskFloat
	case tagString:
		return JITTagMaskString
	case tagSlice:
		return JITTagMaskSlice
	case tagFastDict:
		return JITTagMaskFastDict
	default:
		return 0
	}
}
