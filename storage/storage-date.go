/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "io"
import "unsafe"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// StorageDate retains the generic calendar scalar through compression. Seconds,
// subsecond nanoseconds and display zones use existing integer streams. Format
// magic 14, version 1; legacy integer and JSON column readers stay unchanged.
type StorageDate struct {
	storageJITFunctions
	seconds, nanos, zones StorageInt
	// A loaded historical integer codec already owns immutable whole seconds.
	// Retain that owner (including its reader/JIT state), never copy its struct.
	// Nanoseconds and zone are implicitly zero; a rebuild uses the normal three
	// streams above. This view introduces no persisted format.
	legacySeconds ColumnStorage
	legacyCount   uint32
}

// restoreCalendarStorage canonicalizes historical native calendar carriers
// while their decoded column is still private. It does not change old bytes,
// evaluate SQL, materialize an SCM row bag, or rerun assignment side effects.
func restoreCalendarStorage(source ColumnStorage, count uint32, restore func(scm.Scmer) scm.Scmer) ColumnStorage {
	if count == 0 {
		return source
	}
	if _, current := source.(*StorageDate); current {
		return source
	}
	if constant, ok := source.(*StorageConst); ok {
		if constant.value.IsNil() || constant.value.IsDate() {
			return source
		}
		return &StorageConst{value: restore(constant.value), count: constant.count}
	}
	switch source.(type) {
	case *StorageInt, *StorageSeq:
		// These codecs return only integer seconds or NULL. The native calendar
		// recovery contract maps each integer to NewDate, without SQL policy.
		// The source is still private to this load; the view becomes its sole
		// published owner and needs no traversal or compilation under the lock.
		return &StorageDate{legacySeconds: source, legacyCount: count}
	}
	return buildCalendarStorage(source, count, restore)
}

// buildCalendarStorage is the ordinary native three-stream builder. General
// historical carriers require it on load; an integer view needs it only when
// explicitly serialized, since format 14 requires three StorageInt components
// with matching per-row NULLs (it cannot embed a StorageSeq or constant sidecar).
func buildCalendarStorage(source ColumnStorage, count uint32, restore func(scm.Scmer) scm.Scmer) *StorageDate {
	result := new(StorageDate)
	result.prepare()
	// Native shard columns are bounded; pass twice through one bounded decode
	// buffer, following the ordinary compressor's prepare/scan/build lifecycle.
	buffer := make([]scm.Scmer, 1024)
	reader := source.GetCachedReader(ReadCompressed)
	defer ReleaseColumnReader(reader)
	visit := func(build bool) {
		for offset := uint32(0); offset < count; {
			amount := min(uint32(len(buffer)), count-offset)
			reader.GetValueRange(offset, amount, buffer, 1)
			for index, value := range buffer[:amount] {
				if restore != nil && !value.IsNil() {
					value = restore(value)
				}
				if build {
					result.build(offset+uint32(index), value)
				} else {
					result.scan(offset+uint32(index), value)
				}
			}
			offset += amount
		}
	}
	visit(false)
	result.init(count)
	visit(true)
	// The destination stays private to the load or writer. Readers use the
	// native fallback; a load caller may hold its shard lock, so do not compile.
	return result
}

func (s *StorageDate) ComputeSize() uint {
	if s.legacySeconds != nil {
		return uint(unsafe.Sizeof(*s)) + s.legacySeconds.ComputeSize()
	}
	return uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.seconds)-unsafe.Sizeof(s.nanos)-unsafe.Sizeof(s.zones)) + s.seconds.ComputeSize() + s.nanos.ComputeSize() + s.zones.ComputeSize()
}
func (s *StorageDate) String() string      { return "date[ns]" }
func (s *StorageDate) JITValueType() uint8 { return scm.JITTypeUnknown }
func (s *StorageDate) DistinctCount() uint {
	if s.legacySeconds != nil {
		return s.legacySeconds.DistinctCount()
	}
	return max(s.seconds.DistinctCount(), s.nanos.DistinctCount())
}
func (s *StorageDate) GetCachedReader(hints ColumnReadHints) ColumnReader {
	return s.storageJITFunctions.reader(s)
}
func (s *StorageDate) GetValue(index uint32) scm.Scmer {
	if s.legacySeconds != nil {
		second := s.legacySeconds.GetValue(index)
		if second.IsNil() {
			return second
		}
		return scm.NewDate(second.Int())
	}
	second := s.seconds.GetValue(index)
	if second.IsNil() {
		return second
	}
	return scm.NewDateNanoWithZone(second.Int(), int(s.nanos.GetValue(index).Int()), int(s.zones.GetValue(index).Int()))
}
func dateFromStreams(target, nanos, zones []scm.Scmer, stride int) {
	for index := range nanos {
		position := index * stride
		if !target[position].IsNil() {
			target[position] = scm.NewDateNanoWithZone(target[position].Int(), int(nanos[index].Int()), int(zones[index].Int()))
		}
	}
}
func (s *StorageDate) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if s.legacySeconds != nil {
		s.legacySeconds.GetValueMulti(recids, target, stride)
		calendarSecondsToDates(target, len(recids), stride)
		return
	}
	parts := make([]scm.Scmer, 2*len(recids))
	s.seconds.GetValueMulti(recids, target, stride)
	s.nanos.GetValueMulti(recids, parts[:len(recids)], 1)
	s.zones.GetValueMulti(recids, parts[len(recids):], 1)
	dateFromStreams(target, parts[:len(recids)], parts[len(recids):], stride)
}
func (s *StorageDate) GetValueRange(recid, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if s.legacySeconds != nil {
		s.legacySeconds.GetValueRange(recid, count, target, stride)
		calendarSecondsToDates(target, int(count), stride)
		return
	}
	parts := make([]scm.Scmer, 2*int(count))
	s.seconds.GetValueRange(recid, count, target, stride)
	s.nanos.GetValueRange(recid, count, parts[:count], 1)
	s.zones.GetValueRange(recid, count, parts[count:], 1)
	dateFromStreams(target, parts[:count], parts[count:], stride)
}
func calendarSecondsToDates(target []scm.Scmer, count, stride int) {
	for index := 0; index < count; index++ {
		position := index * stride
		if !target[position].IsNil() {
			target[position] = scm.NewDate(target[position].Int())
		}
	}
}
func (s *StorageDate) prepare() {
	s.legacySeconds = nil
	s.legacyCount = 0
	s.seconds.prepare()
	s.nanos.prepare()
	s.zones.prepare()
}
func dateStorageParts(value scm.Scmer) (scm.Scmer, scm.Scmer, scm.Scmer) {
	if value.IsNil() {
		return value, value, value
	}
	if !value.IsDate() {
		panic("date storage requires calendar values")
	}
	return scm.NewInt(value.Int()), scm.NewInt(int64(value.DateNanosecond())), scm.NewInt(int64(value.DateZone()))
}
func (s *StorageDate) scan(index uint32, value scm.Scmer) {
	seconds, nanos, zones := dateStorageParts(value)
	s.seconds.scan(index, seconds)
	s.nanos.scan(index, nanos)
	s.zones.scan(index, zones)
}
func (s *StorageDate) proposeCompression(count uint32) ColumnStorage { return nil }
func (s *StorageDate) init(count uint32) {
	s.seconds.init(count)
	s.nanos.init(count)
	s.zones.init(count)
}
func (s *StorageDate) build(index uint32, value scm.Scmer) {
	seconds, nanos, zones := dateStorageParts(value)
	s.seconds.build(index, seconds)
	s.nanos.build(index, nanos)
	s.zones.build(index, zones)
}
func (s *StorageDate) finish() {
	if s.legacySeconds == nil {
		s.seconds.finish()
		s.nanos.finish()
		s.zones.finish()
	}
	s.storageJITFunctions.finish(s)
}
func (s *StorageDate) Serialize(writer io.Writer) {
	if s.legacySeconds != nil {
		// A read leaves the old blob untouched. An actual write must use the
		// current format; build privately without mutating this shared view or
		// compiling code. Normal rebuilds already produce a fresh native date.
		buildCalendarStorage(s, s.legacyCount, nil).Serialize(writer)
		return
	}
	if err := binary.Write(writer, binary.LittleEndian, [2]byte{14, 1}); err != nil {
		panic(err)
	}
	s.seconds.Serialize(writer)
	s.nanos.Serialize(writer)
	s.zones.Serialize(writer)
}

// exactCalendarReader gives the new format strict component framing without
// changing the permanent legacy StorageInt reader's behavior.
type exactCalendarReader struct{ io.Reader }

func (reader exactCalendarReader) Read(value []byte) (int, error) {
	count, err := io.ReadFull(reader.Reader, value)
	if err != nil {
		panic(err)
	}
	return count, nil
}

func (s *StorageDate) Deserialize(reader io.Reader) uint {
	s.legacySeconds = nil
	s.legacyCount = 0
	reader = exactCalendarReader{reader}
	var version [1]byte
	readStorageBytes(reader, version[:])
	if version[0] != 1 {
		panic("unknown date column version")
	}
	count := s.seconds.DeserializeEx(reader, true)
	if s.nanos.DeserializeEx(reader, true) != count || s.zones.DeserializeEx(reader, true) != count {
		panic("date column component lengths differ")
	}
	// Validate privately before the generation can be published.
	for index := uint32(0); index < uint32(count); index++ {
		second, nano, zone := s.seconds.GetValue(index), s.nanos.GetValue(index), s.zones.GetValue(index)
		if second.IsNil() {
			if !nano.IsNil() || !zone.IsNil() {
				panic("invalid null date components")
			}
			continue
		}
		if nano.IsNil() || zone.IsNil() {
			panic("missing date components")
		}
		scm.NewDateNanoWithZone(second.Int(), int(nano.Int()), int(zone.Int()))
	}
	return count
}

func storageDateValue(s *StorageDate, index scm.Scmer) scm.Scmer {
	return s.GetValue(uint32(index.Int()))
}
func (s *StorageDate) JITEmit(ctx *scm.JITContext, index, result scm.JITValueDesc) scm.JITValueDesc {
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	index = scm.JITPrepareScmerGoArg(ctx, index)
	value := ctx.EmitGoCallScalar(scm.GoFuncAddr(storageDateValue), []scm.JITValueDesc{owner, index}, 2)
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
func (s *StorageDate) JITEmitGetValueMulti(ctx *scm.JITContext, recids, target, stride, result scm.JITValueDesc) scm.JITValueDesc {
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	ctx.EmitGoCallVoid(scm.GoFuncAddr((*StorageDate).GetValueMulti), []scm.JITValueDesc{owner, recids, target, stride})
	return result
}
func (s *StorageDate) JITEmitGetValueRange(ctx *scm.JITContext, recid, count, target, stride, result scm.JITValueDesc) scm.JITValueDesc {
	ctx.TrackPointer(unsafe.Pointer(s))
	owner := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	ctx.EmitGoCallVoid(scm.GoFuncAddr((*StorageDate).GetValueRange), []scm.JITValueDesc{owner, recid, count, target, stride})
	return result
}
