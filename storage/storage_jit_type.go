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

import "github.com/launix-de/memcp/scm"

// JITValueType reports an exact physical Scmer tag only when every valid main
// row has that representation. In particular, a nullable storage is not given
// the type of its non-NULL values: an inlined consumer is then still required
// to inspect the runtime tag. Delta rows are not represented by ColumnStorage
// and must never use these answers.

func (s *StorageInt) JITValueType() uint8 {
	if s.hasNull {
		return scm.JITTypeUnknown
	}
	return scm.TagInt
}

func (s *StorageFloat) JITValueType() uint8 {
	if s.hasNull {
		return scm.JITTypeUnknown
	}
	return scm.TagFloat
}

func (s *StorageDecimal) JITValueType() uint8 {
	if s.inner.hasNull {
		return scm.JITTypeUnknown
	}
	if s.scaleExp > 0 {
		return scm.TagInt
	}
	return scm.TagFloat
}

func (s *StorageConst) JITValueType() uint8 {
	return s.value.GetTag()
}

func (s *StorageSCMER) JITValueType() uint8 {
	// StorageSCMER is also the mutable backing store used by compute proxies.
	// Its analysis-time enum sample would become stale after SetValue, so it
	// cannot provide a durable finished-main-column type guarantee.
	return scm.JITTypeUnknown
}

func (s *StorageSeq) JITValueType() uint8 {
	if s.start.hasNull {
		return scm.JITTypeUnknown
	}
	return scm.TagFloat
}

func (s *StorageEnum) JITValueType() uint8 {
	if s.k == 0 {
		return scm.JITTypeUnknown
	}
	tag := s.values[0].GetTag()
	for i := uint8(1); i < s.k; i++ {
		if s.values[i].GetTag() != tag {
			return scm.JITTypeUnknown
		}
	}
	return tag
}

func (s *StorageString) JITValueType() uint8 {
	if s.stringHasNull() {
		return scm.JITTypeUnknown
	}
	switch s.format {
	case FormatRaw:
		return scm.TagString
	case FormatUUIDLower, FormatUUIDUpper:
		return scm.TagCString
	case FormatHexLower, FormatHexUpper,
		FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
		// decodeAt represents an empty value as TagString because no payload
		// pointer exists. A positive minimum length proves uniform CString.
		if s.lens.offset > 0 {
			return scm.TagCString
		}
	case FormatBase64Upper, FormatBase64Lower:
		if s.lens.offset > 0 {
			return scm.TagBString
		}
	}
	return scm.JITTypeUnknown
}

func (s *StorageString) stringHasNull() bool {
	if s.nodict {
		return s.starts.hasNull
	}
	return s.values.hasNull
}

func (s *StoragePrefix) JITValueType() uint8 {
	if s.values.stringHasNull() {
		return scm.JITTypeUnknown
	}
	// Prefix concatenation always materializes a plain Go string, regardless
	// of the suffix storage's physical string representation.
	return scm.TagString
}

func (s *StorageSparse) JITValueType() uint8 {
	if s.count > 0 && s.i == 0 {
		return scm.TagNil
	}
	return scm.JITTypeUnknown
}

func (s *StorageComputeProxy) JITValueType() uint8 {
	// A proxy may repair an invalid main entry by evaluating its computor. Its
	// backing storage therefore cannot prove the tag returned by the proxy.
	return scm.JITTypeUnknown
}

func (s *OverlayBlob) JITValueType() uint8 {
	if s.Base == nil {
		return scm.JITTypeUnknown
	}
	tag := s.Base.JITValueType()
	switch tag {
	case scm.TagCString, scm.TagBString:
		// Resolving a blob marker materializes TagString, whereas an ordinary
		// compressed value passes through unchanged.
		return scm.JITTypeUnknown
	default:
		return tag
	}
}
