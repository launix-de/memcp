/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "io"
import "bytes"
import "unsafe"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// Magic 3/version 3/encoding 3 stores two integer components. The first packs
// the 11-bit zone above biased signed 45-bit Unix seconds. A constant zone and
// nearby dates disappear into StorageInt's offset; bias preserves locality
// across 1970. The second stores nanoseconds divided by their exact common GCD.
// Each component uses the ordinary const/sequence/PFOR chooser, including NULLs,
// and has a uint64 byte length so nested legacy readers cannot consume trailers.
// Header: [magic,version,encoding] [count uint64 LE] [fraction scale uint32 LE],
// followed by two [length uint64 LE] [complete column frame] records.
// V2/encoding 2 remains a permanent three-StorageInt-component reader.
const nativeDateSecondsBias = int64(1 << 44)
const nativeDateSecondsMask = uint64(1<<45 - 1)

func nativeDateGCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

type nativeDateColumn struct {
	parts         []ColumnStorage
	count         uint32
	fractionScale int
	hasNull       bool
}

func newNativeDateColumn(scale int) *nativeDateColumn {
	if scale == 0 {
		scale = 1
	}
	return &nativeDateColumn{parts: []ColumnStorage{new(StorageSCMER), new(StorageSCMER)}, fractionScale: scale}
}
func (column *nativeDateColumn) fields(value scm.Scmer) [2]scm.Scmer {
	if value.IsNil() {
		return [2]scm.Scmer{}
	}
	if !value.IsDate() {
		panic("native date column requires dates or NULL")
	}
	seconds := value.Int()
	if seconds < -nativeDateSecondsBias || seconds >= nativeDateSecondsBias {
		panic("native date seconds out of range")
	}
	packed := uint64(value.DateZone())<<45 | uint64(seconds+nativeDateSecondsBias)
	return [2]scm.Scmer{scm.NewInt(int64(packed)), scm.NewInt(int64(value.DateNanosecond() / column.fractionScale))}
}
func (s *storageSCMERExact) prepare() {
	if s.nativeDates == nil {
		s.StorageSCMER.prepare()
		return
	}
	s.nativeDates.hasNull = false
	for _, part := range s.nativeDates.parts {
		part.prepare()
	}
}
func (s *storageSCMERExact) scan(row uint32, value scm.Scmer) {
	if s.nativeDates == nil {
		s.StorageSCMER.scan(row, value)
		return
	}
	s.nativeDates.hasNull = s.nativeDates.hasNull || value.IsNil()
	for i, field := range s.nativeDates.fields(value) {
		s.nativeDates.parts[i].scan(row, field)
	}
}
func (s *storageSCMERExact) proposeCompression(count uint32) ColumnStorage {
	if s.nativeDates == nil {
		return s.StorageSCMER.proposeCompression(count)
	}
	changed := false
	for i, part := range s.nativeDates.parts {
		if next := part.proposeCompression(count); next != nil {
			switch next.(type) {
			case *StorageConst, *StorageInt:
			case *StorageSeq:
				// Prefer the integer NULL bitmap for nullable date components.
				if s.nativeDates.hasNull {
					next = new(StorageInt)
				}
			default:
				// These components have an integer-or-NULL contract. Keep their
				// native integer codecs rather than JSON, enum or blob storage.
				next = new(StorageInt)
			}
			s.nativeDates.parts[i] = next
			changed = true
		}
	}
	if changed {
		return s
	}
	return nil
}
func (s *storageSCMERExact) init(count uint32) {
	if s.nativeDates == nil {
		s.StorageSCMER.init(count)
		return
	}
	s.nativeDates.count = count
	for _, part := range s.nativeDates.parts {
		part.init(count)
	}
}
func (s *storageSCMERExact) build(row uint32, value scm.Scmer) {
	if s.nativeDates == nil {
		s.StorageSCMER.build(row, value)
		return
	}
	for i, field := range s.nativeDates.fields(value) {
		s.nativeDates.parts[i].build(row, field)
	}
}
func (s *storageSCMERExact) finish() {
	if s.nativeDates != nil {
		for _, part := range s.nativeDates.parts {
			part.finish()
		}
	}
	s.storageJITFunctions.finish(s)
}

// Used only when rewriting a legacy in-memory representation. Ordinary rebuilds
// enter the same prepare/scan/propose/init/build protocol directly and publish
// compressed RAM, including MEMORY-engine columns that are never serialized.
func buildNativeDateColumn(count uint32, scale int, value func(uint32) scm.Scmer) *storageSCMERExact {
	s := &storageSCMERExact{nativeDates: newNativeDateColumn(scale)}
	if count == 0 {
		s.nativeDates.parts = []ColumnStorage{new(StorageConst), new(StorageConst)}
		s.finish()
		return s
	}
	for {
		s.prepare()
		for row := uint32(0); row < count; row++ {
			s.scan(row, value(row))
		}
		if s.proposeCompression(count) == nil {
			break
		}
	}
	s.init(count)
	for row := uint32(0); row < count; row++ {
		s.build(row, value(row))
	}
	s.finish()
	return s
}
func (s *storageSCMERExact) serializeNativeDates(f io.Writer) bool {
	hasDate, scale := false, 0
	for _, value := range s.values {
		if value.IsDate() {
			hasDate = true
			scale = nativeDateGCD(scale, value.DateNanosecond())
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
	buildNativeDateColumn(uint32(len(s.values)), scale, s.GetValue).nativeDates.serialize(f)
	return true
}

// A sequence point read uses its existing invocation-local bulk decoder,
// never the shared atomic pivot used by StorageSeq.GetValue. The one-row tuple
// stays on the Go stack; no per-read cursor object or heap buffer is retained.
func nativeDateIntegerValue(part ColumnStorage, row uint32) scm.Scmer {
	if sequence, ok := part.(*StorageSeq); ok {
		var result [1]scm.Scmer
		sequence.GetValueRange(row, 1, result[:], 1)
		return result[0]
	}
	return part.GetValue(row)
}

func (column *nativeDateColumn) value(row uint32) scm.Scmer {
	seconds := nativeDateIntegerValue(column.parts[0], row)
	if seconds.IsNil() {
		return seconds
	}
	if len(column.parts) == 3 {
		return scm.NewDateNanoWithZone(seconds.Int(), int(nativeDateIntegerValue(column.parts[1], row).Int()), int(nativeDateIntegerValue(column.parts[2], row).Int()))
	}
	packed := uint64(seconds.Int())
	return scm.NewDateNanoWithZone(int64(packed&nativeDateSecondsMask)-nativeDateSecondsBias,
		int(nativeDateIntegerValue(column.parts[1], row).Int())*column.fractionScale, int(packed>>45))
}
func (column *nativeDateColumn) serialize(f io.Writer) {
	if len(column.parts) == 3 {
		scale := 0
		for row := uint32(0); row < column.count; row++ {
			scale = nativeDateGCD(scale, column.value(row).DateNanosecond())
		}
		buildNativeDateColumn(column.count, scale, column.value).nativeDates.serialize(f)
		return
	}
	for _, value := range []any{[3]byte{3, storageExactScalarVersion, 3}, uint64(column.count), uint32(column.fractionScale)} {
		if err := binary.Write(f, binary.LittleEndian, value); err != nil {
			panic(err)
		}
	}
	for _, part := range column.parts {
		var frame bytes.Buffer
		part.Serialize(&frame)
		if err := binary.Write(f, binary.LittleEndian, uint64(frame.Len())); err != nil {
			panic(err)
		}
		if _, err := frame.WriteTo(f); err != nil {
			panic(err)
		}
	}
}

// ReadFull makes the historical integer/sequence readers fail on truncation.
// New composite components are individually bounded; no reader crosses frames.
type nativeDateFrameReader struct{ io.Reader }

func (reader nativeDateFrameReader) Read(target []byte) (int, error) {
	count, err := io.ReadFull(reader.Reader, target)
	if err != nil {
		panic(err)
	}
	return count, nil
}
func readNativeDateCount(f io.Reader) uint32 {
	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		panic(err)
	}
	if count > uint64(^uint32(0)) {
		panic("native date column count exceeds row identifier domain")
	}
	return uint32(count)
}
func (s *storageSCMERExact) deserializeNativeDates(f io.Reader) uint {
	column := &nativeDateColumn{count: readNativeDateCount(f), parts: make([]ColumnStorage, 3), fractionScale: 1}
	f = nativeDateFrameReader{f}
	for i := range column.parts {
		var magic [1]byte
		readStorageBytes(f, magic[:])
		if magic[0] != 10 {
			panic("native date component must use integer storage")
		}
		part := new(StorageInt)
		if rows := part.Deserialize(f); rows != uint(column.count) {
			panic("native date component row counts disagree")
		}
		column.parts[i] = part
	}
	column.validate()
	s.nativeDates = column
	return uint(column.count)
}
func (s *storageSCMERExact) deserializePackedNativeDates(f io.Reader) uint {
	column := &nativeDateColumn{count: readNativeDateCount(f), parts: make([]ColumnStorage, 2)}
	var scale uint32
	if err := binary.Read(f, binary.LittleEndian, &scale); err != nil {
		panic(err)
	}
	if scale == 0 || scale >= 1000000000 {
		panic("invalid native date fraction scale")
	}
	column.fractionScale = int(scale)
	for i := range column.parts {
		var length uint64
		if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
			panic(err)
		}
		if length > uint64(int(^uint(0)>>1)) || length == 0 {
			panic("invalid native date component length")
		}
		payload := make([]byte, int(length))
		readStorageBytes(f, payload)
		var part ColumnStorage
		switch payload[0] {
		case 10:
			part = new(StorageInt)
		case 11:
			part = new(StorageSeq)
		case 41:
			part = new(StorageConst)
		default:
			panic("invalid native date integer component")
		}
		reader := bytes.NewReader(payload[1:])
		if rows := part.Deserialize(nativeDateFrameReader{reader}); rows != uint(column.count) || reader.Len() != 0 {
			panic("native date component extent mismatch")
		}
		column.parts[i] = part
	}
	column.validate()
	s.nativeDates = column
	return uint(column.count)
}
func (column *nativeDateColumn) validate() {
	for row := uint32(0); row < column.count; row++ {
		first := column.parts[0].GetValue(row)
		for _, part := range column.parts[1:] {
			value := part.GetValue(row)
			if first.IsNil() != value.IsNil() || !value.IsNil() && !value.IsInt() {
				panic("native date integer/NULL components disagree")
			}
		}
		if first.IsNil() {
			column.hasNull = true
			continue
		}
		if !first.IsInt() {
			panic("native date clock is not an integer")
		}
		if len(column.parts) == 2 {
			if first.Int() < 0 || uint64(first.Int()) >= 1<<56 {
				panic("native date clock out of range")
			}
			fraction := column.parts[1].GetValue(row).Int()
			if fraction < 0 || fraction > 999999999/int64(column.fractionScale) {
				panic("native date fraction out of range")
			}
		} else {
			seconds, nanos, zone := first.Int(), column.parts[1].GetValue(row).Int(), column.parts[2].GetValue(row).Int()
			if seconds < -nativeDateSecondsBias || seconds >= nativeDateSecondsBias || nanos < 0 || nanos >= 1000000000 || zone < 0 || zone > 2047 {
				panic("native date component out of range")
			}
		}
	}
}
func (s *storageSCMERExact) GetValue(index uint32) scm.Scmer {
	if s.nativeDates == nil {
		return s.StorageSCMER.GetValue(index)
	}
	return s.nativeDates.value(index)
}

// Concrete dispatch keeps the bounded decode buffers on the stack. In
// particular, sequence columns amortize their local segment cursor per batch.
func nativeDateIntegerRange(part ColumnStorage, row, count uint32, target []scm.Scmer) {
	switch part := part.(type) {
	case *StorageInt:
		part.GetValueRange(row, count, target, 1)
	case *StorageSeq:
		part.GetValueRange(row, count, target, 1)
	case *StorageConst:
		part.GetValueRange(row, count, target, 1)
	default:
		panic("unexpected native date component")
	}
}
func nativeDateIntegerMulti(part ColumnStorage, rows []uint32, target []scm.Scmer) {
	switch part := part.(type) {
	case *StorageInt:
		part.GetValueMulti(rows, target, 1)
	case *StorageSeq:
		part.GetValueMulti(rows, target, 1)
	case *StorageConst:
		part.GetValueMulti(rows, target, 1)
	default:
		panic("unexpected native date component")
	}
}
func (column *nativeDateColumn) decoded(clock, fraction scm.Scmer) scm.Scmer {
	if clock.IsNil() {
		return clock
	}
	packed := uint64(clock.Int())
	return scm.NewDateNanoWithZone(int64(packed&nativeDateSecondsMask)-nativeDateSecondsBias,
		int(fraction.Int())*column.fractionScale, int(packed>>45))
}
func (s *storageSCMERExact) GetValueRange(recid, count uint32, target []scm.Scmer, stride int) {
	if s.nativeDates == nil {
		s.StorageSCMER.GetValueRange(recid, count, target, stride)
		return
	}
	if stride <= 0 {
		stride = 1
	}
	if len(s.nativeDates.parts) == 3 {
		// The permanent v2 reader consists only of stateless integer columns.
		for i := uint32(0); i < count; i++ {
			target[int(i)*stride] = s.GetValue(recid + i)
		}
		return
	}
	var clock, fraction [128]scm.Scmer
	for offset := uint32(0); offset < count; {
		n := min(uint32(len(clock)), count-offset)
		nativeDateIntegerRange(s.nativeDates.parts[0], recid+offset, n, clock[:])
		nativeDateIntegerRange(s.nativeDates.parts[1], recid+offset, n, fraction[:])
		for i := uint32(0); i < n; i++ {
			target[int(offset+i)*stride] = s.nativeDates.decoded(clock[i], fraction[i])
		}
		offset += n
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
	if len(s.nativeDates.parts) == 3 {
		for i, row := range recids {
			target[i*stride] = s.GetValue(row)
		}
		return
	}
	var clock, fraction [128]scm.Scmer
	for offset := 0; offset < len(recids); {
		n := min(len(clock), len(recids)-offset)
		rows := recids[offset : offset+n]
		nativeDateIntegerMulti(s.nativeDates.parts[0], rows, clock[:])
		nativeDateIntegerMulti(s.nativeDates.parts[1], rows, fraction[:])
		for i := 0; i < n; i++ {
			target[(offset+i)*stride] = s.nativeDates.decoded(clock[i], fraction[i])
		}
		offset += n
	}
}
func (s *storageSCMERExact) GetCachedReader(hints ColumnReadHints) ColumnReader {
	return s.storageJITFunctions.reader(s)
}
func (s *storageSCMERExact) JITValueType() uint8 {
	if s.nativeDates != nil && !s.nativeDates.hasNull {
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
		result += uint(unsafe.Sizeof(*s.nativeDates)) + uint(cap(s.nativeDates.parts))*uint(unsafe.Sizeof(s.nativeDates.parts[0]))
		for _, part := range s.nativeDates.parts {
			result += part.ComputeSize()
		}
	}
	return result
}

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
