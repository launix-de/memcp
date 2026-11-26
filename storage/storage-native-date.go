/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "io"
import "unsafe"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// Exact scalar magic 3, version 2/encoding 2 stores native dates as three
// ordinary bit-packed integer columns: Unix seconds, nanoseconds and zone.
// Each component carries its complete magic-10/versioned storage frame and
// count. All three NULL masks must agree. No component reads past its frame.
// Mixed native values use the permanent private-JSON payload (encoding 1).
func (s *storageSCMERExact) serializeNativeDates(f io.Writer) bool {
	hasDate := false
	for _, value := range s.values {
		if value.IsDate() {
			hasDate = true
		} else if !value.IsNil() {
			return false
		}
	}
	if !hasDate {
		return false
	}
	if uint64(len(s.values)) > uint64(^uint32(0)) {
		panic("native date column count exceeds row identifier domain")
	}
	var components [3]StorageInt
	for i := range components {
		components[i].prepare()
	}
	parts := func(value scm.Scmer) [3]scm.Scmer {
		if value.IsNil() {
			return [3]scm.Scmer{}
		}
		return [3]scm.Scmer{scm.NewInt(value.Int()), scm.NewInt(int64(value.DateNanosecond())), scm.NewInt(int64(value.DateZone()))}
	}
	for row, value := range s.values {
		for part, field := range parts(value) {
			components[part].scan(uint32(row), field)
		}
	}
	for i := range components {
		components[i].init(uint32(len(s.values)))
	}
	for row, value := range s.values {
		for part, field := range parts(value) {
			components[part].build(uint32(row), field)
		}
	}
	(&nativeDateColumn{parts: components, count: uint32(len(s.values))}).serialize(f)
	return true
}

// nativeDateColumn owns the immutable packed streams. Decoding validates them
// once, but never materializes a second complete column of Scmer values.
type nativeDateColumn struct {
	parts [3]StorageInt
	count uint32
}

func (column *nativeDateColumn) serialize(f io.Writer) {
	if err := binary.Write(f, binary.LittleEndian, [3]byte{3, storageExactScalarVersion, 2}); err != nil {
		panic(err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint64(column.count)); err != nil {
		panic(err)
	}
	for part := range column.parts {
		column.parts[part].Serialize(f)
	}
}

// Component readers are permanent legacy codecs. Enforce complete reads at
// this new frame boundary without changing their historical behavior.
type nativeDateFrameReader struct{ io.Reader }

func (reader nativeDateFrameReader) Read(target []byte) (int, error) {
	count, err := io.ReadFull(reader.Reader, target)
	if err != nil {
		panic(err)
	}
	return count, nil
}

func (s *storageSCMERExact) deserializeNativeDates(f io.Reader) uint {
	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		panic(err)
	}
	if count > uint64(^uint32(0)) {
		panic("native date column count exceeds row identifier domain")
	}
	column := &nativeDateColumn{count: uint32(count)}
	f = nativeDateFrameReader{f}
	for part := range column.parts {
		var magic [1]byte
		readStorageBytes(f, magic[:])
		if magic[0] != 10 {
			panic("native date component must use integer storage")
		}
		if rows := column.parts[part].Deserialize(f); uint64(rows) != count {
			panic("native date component row counts disagree")
		}
	}
	// Validate before the immutable column can be published. The on-disk
	// contract is signed 45-bit seconds, 0..999999999 nanos and an 11-bit zone.
	for row := uint32(0); uint64(row) < count; row++ {
		seconds, nanos, zone := column.parts[0].GetValue(row), column.parts[1].GetValue(row), column.parts[2].GetValue(row)
		if seconds.IsNil() || nanos.IsNil() || zone.IsNil() {
			if !seconds.IsNil() || !nanos.IsNil() || !zone.IsNil() {
				panic("native date component NULL masks disagree")
			}
		} else if seconds.Int() < -(1<<44) || seconds.Int() >= 1<<44 || nanos.Int() < 0 || nanos.Int() >= 1000000000 || zone.Int() < 0 || zone.Int() > 2047 {
			panic("native date component out of range")
		}
	}
	s.nativeDates = column
	return uint(count)
}

func (s *storageSCMERExact) GetValue(index uint32) scm.Scmer {
	if s.nativeDates == nil {
		return s.StorageSCMER.GetValue(index)
	}
	parts := &s.nativeDates.parts
	seconds := parts[0].GetValue(index)
	if seconds.IsNil() {
		return seconds
	}
	return scm.NewDateNanoWithZone(seconds.Int(), int(parts[1].GetValue(index).Int()), int(parts[2].GetValue(index).Int()))
}
func (s *storageSCMERExact) GetValueRange(recid, count uint32, target []scm.Scmer, stride int) {
	if s.nativeDates == nil {
		s.StorageSCMER.GetValueRange(recid, count, target, stride)
		return
	}
	if stride <= 0 {
		stride = 1
	}
	for index := uint32(0); index < count; index++ {
		target[int(index)*stride] = s.GetValue(recid + index)
	}
}
func (s *storageSCMERExact) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if s.nativeDates == nil {
		s.StorageSCMER.GetValueMulti(recids, target, stride)
		return
	}
	if stride <= 0 {
		stride = 1
	}
	for index, recid := range recids {
		target[index*stride] = s.GetValue(recid)
	}
}
func (s *storageSCMERExact) GetCachedReader(hints ColumnReadHints) ColumnReader {
	return s.storageJITFunctions.reader(s)
}
func (s *storageSCMERExact) JITValueType() uint8 {
	if s.nativeDates != nil && !s.nativeDates.parts[0].hasNull {
		return scm.TagDate
	}
	return scm.JITTypeUnknown
}
func (s *storageSCMERExact) DistinctCount() uint {
	if s.nativeDates != nil {
		return uint(s.nativeDates.count)
	}
	return s.StorageSCMER.DistinctCount()
}
func (s *storageSCMERExact) ComputeSize() uint {
	result := uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.StorageSCMER)) + s.StorageSCMER.ComputeSize()
	if s.nativeDates != nil {
		result += uint(unsafe.Sizeof(*s.nativeDates) - unsafe.Sizeof(s.nativeDates.parts))
		for part := range s.nativeDates.parts {
			result += s.nativeDates.parts[part].ComputeSize()
		}
	}
	return result
}
func (s *storageSCMERExact) finish() { s.storageJITFunctions.finish(s) }

func exactScalarDateValue(s *storageSCMERExact, index scm.Scmer) scm.Scmer {
	return s.GetValue(uint32(index.Int()))
}
func (s *storageSCMERExact) JITEmit(ctx *scm.JITContext, index, result scm.JITValueDesc) scm.JITValueDesc {
	if s.nativeDates == nil {
		return s.StorageSCMER.JITEmit(ctx, index, result)
	}
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	index = scm.JITPrepareScmerGoArg(ctx, index)
	value := ctx.EmitGoCallScalar(scm.GoFuncAddr(exactScalarDateValue), []scm.JITValueDesc{owner, index}, 2)
	value.NoHeapPointer = false
	ctx.BindReg(value.Reg, &value)
	ctx.BindReg(value.Reg2, &value)
	if result.Loc == scm.LocAny {
		return value
	}
	ctx.EmitMovPairToResult(&value, &result)
	result.Type = scm.JITTypeUnknown
	result.NoHeapPointer = false
	return result
}
func (s *storageSCMERExact) JITEmitGetValueMulti(ctx *scm.JITContext, recids, target, stride, result scm.JITValueDesc) scm.JITValueDesc {
	if s.nativeDates == nil {
		return s.StorageSCMER.JITEmitGetValueMulti(ctx, recids, target, stride, result)
	}
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	ctx.EmitGoCallVoid(scm.GoFuncAddr((*storageSCMERExact).GetValueMulti), []scm.JITValueDesc{owner, recids, target, stride})
	return result
}
func (s *storageSCMERExact) JITEmitGetValueRange(ctx *scm.JITContext, recid, count, target, stride, result scm.JITValueDesc) scm.JITValueDesc {
	if s.nativeDates == nil {
		return s.StorageSCMER.JITEmitGetValueRange(ctx, recid, count, target, stride, result)
	}
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	ctx.EmitGoCallVoid(scm.GoFuncAddr((*storageSCMERExact).GetValueRange), []scm.JITValueDesc{owner, recid, count, target, stride})
	return result
}
