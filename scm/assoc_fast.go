/*
Copyright (C) 2025-2026  Carl-Philip Hänsch

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

import (
	"encoding/binary"
	"hash/maphash"
	"math"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// Stable seed for hashing to ensure consistent indices across Set/Get calls.
var fastDictSeed maphash.Seed

func init() {
	fastDictSeed = maphash.MakeSeed()
}

// FastDict: shard-local assoc optimized for frequent set/merge operations.
// Implementation uses a flat pairs array plus a lightweight hash index
// to avoid O(N^2) behavior as it grows.
type FastDict struct {
	Pairs      []Scmer          // [k0, v0, k1, v1, ...]
	index      map[uint64]int   // hash -> first position in Pairs
	collisions map[uint64][]int // additional positions, allocated only on hash collision
}

func NewFastDictValue(capacityPairs int) *FastDict {
	if capacityPairs < 0 {
		capacityPairs = 0
	}
	return &FastDict{Pairs: make([]Scmer, 0, capacityPairs*2), index: make(map[uint64]int)}
}

func (d *FastDict) Iterate(fn func(k, v Scmer) bool) {
	for i := 0; i < len(d.Pairs); i += 2 {
		if !fn(d.Pairs[i], d.Pairs[i+1]) {
			return
		}
	}
}

// HashKey computes a stable hash for a Scheme value.
// It avoids allocating intermediate strings by inspecting types and
// feeding bytes directly to a streaming hasher. Lists are hashed by
// recursively hashing their elements with structural markers.
func HashKey(k Scmer) uint64 {
	var h maphash.Hash
	h.SetSeed(fastDictSeed)
	var writeScmer func(v Scmer)
	writeScmer = func(v Scmer) {
		switch v.GetTag() {
		case tagNil:
			h.WriteByte(0)
		case tagBool:
			h.WriteByte(1)
			if v.Bool() {
				h.WriteByte(1)
			} else {
				h.WriteByte(0)
			}
		case tagInt:
			h.WriteByte(2)
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(v.Int()))
			h.Write(b[:])
		case tagFloat:
			h.WriteByte(3)
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(v.Float()))
			h.Write(b[:])
		case tagString:
			h.WriteByte(4)
			h.WriteString(v.String())
		case tagSymbol:
			h.WriteByte(5)
			h.WriteString(v.String())
		case tagSlice:
			h.WriteByte(6)
			// write length to reduce collisions for different list sizes
			var b [8]byte
			slice := v.Slice()
			binary.LittleEndian.PutUint64(b[:], uint64(len(slice)))
			h.Write(b[:])
			for _, el := range slice {
				writeScmer(el)
			}
		case tagVector:
			h.WriteByte(7)
			vec := v.Vector()
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(len(vec)))
			h.Write(b[:])
			for _, el := range vec {
				var bb [8]byte
				binary.LittleEndian.PutUint64(bb[:], math.Float64bits(el))
				h.Write(bb[:])
			}
		case tagFunc, tagFuncEnv:
			h.WriteByte(8)
			// Hash native function by pointer to ensure stability
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(uintptr(unsafe.Pointer(v.ptr))))
			h.Write(b[:])
		case tagSourceInfo:
			writeScmer(v.SourceInfo().value)
		case tagFastDict:
			fd := v.FastDict()
			// Hash as list of pairs to match []Scmer assoc representation
			h.WriteByte(6)
			var b [8]byte
			if fd == nil {
				binary.LittleEndian.PutUint64(b[:], 0)
				h.Write(b[:])
				return
			}
			binary.LittleEndian.PutUint64(b[:], uint64(len(fd.Pairs)))
			h.Write(b[:])
			for i := 0; i < len(fd.Pairs); i += 2 {
				writeScmer(fd.Pairs[i])
				writeScmer(fd.Pairs[i+1])
			}
		case tagAny:
			if si, ok := v.Any().(SourceInfo); ok {
				writeScmer(si.value)
				return
			}
			fallback := reflect.TypeOf(v.Any()).String()
			h.WriteByte(255)
			h.WriteString(fallback)
		default:
			// Hash as list of pairs to match []Scmer assoc representation
			h.WriteByte(255)
			h.WriteString(v.String())
		}
	}
	writeScmer(k)
	return h.Sum64()
}

// hashStructuralScalar normalizes the scalar coercions needed by planner
// expression catalogs without charging every FastDict and optimizer
// fingerprint for date and numeric-string parsing.
func hashStructuralFloat(value float64) uint64 {
	if value == 0 {
		value = 0
	}
	return HashKey(NewFloat(value))
}

func hashStructuralScalar(value Scmer) uint64 {
	switch value.GetTag() {
	case tagNil:
		return hashStructuralFloat(0)
	case tagBool:
		if value.Bool() {
			return hashStructuralFloat(1)
		}
		return hashStructuralFloat(0)
	case tagInt, tagDate:
		return hashStructuralFloat(float64(value.Int()))
	case tagFloat:
		return hashStructuralFloat(value.Float())
	case tagString, tagSymbol, tagCString, tagBString:
		text := value.String()
		if text == "" || strings.EqualFold(text, "false") {
			return hashStructuralFloat(0)
		}
		if timestamp, ok := ParseDateString(text); ok {
			return hashStructuralFloat(float64(timestamp))
		}
		if numericPrefix(text) != "" {
			return hashStructuralFloat(value.Float())
		}
		return HashKey(NewString(text))
	default:
		return HashKey(value)
	}
}

func combineStructuralHash(hash, value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	value ^= value >> 31
	return (hash ^ value) * 0x100000001b3
}

// hashStructuralKey computes every immutable list node bottom-up once. The
// resulting memo is frozen before it is shared with lookup closures.
func hashStructuralKey(key Scmer, memo map[Scmer]uint64) uint64 {
	switch key.GetTag() {
	case tagSourceInfo:
		return hashStructuralKey(key.SourceInfo().value, memo)
	case tagAny:
		if source, ok := key.Any().(SourceInfo); ok {
			return hashStructuralKey(source.value, memo)
		}
		return HashKey(key)
	case tagSlice:
		if hash, ok := memo[key]; ok {
			return hash
		}
		items := key.Slice()
		hash := combineStructuralHash(0x6a09e667f3bcc909, uint64(len(items)))
		for _, item := range items {
			hash = combineStructuralHash(hash, hashStructuralKey(item, memo))
		}
		memo[key] = hash
		return hash
	case tagFastDict:
		panic("make_structural_index requires immutable list expressions")
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagSymbol, tagCString, tagBString:
		// Equal accepts numerically equal int/float/date values across tags.
		return hashStructuralScalar(key)
	default:
		return HashKey(key)
	}
}

// hashStructuralReadonly is the allocation-free counterpart used after
// publication for an equal expression that is not one of the original roots.
func hashStructuralReadonly(key Scmer) uint64 {
	switch key.GetTag() {
	case tagSourceInfo:
		return hashStructuralReadonly(key.SourceInfo().value)
	case tagAny:
		if source, ok := key.Any().(SourceInfo); ok {
			return hashStructuralReadonly(source.value)
		}
		return HashKey(key)
	case tagSlice:
		items := key.Slice()
		hash := combineStructuralHash(0x6a09e667f3bcc909, uint64(len(items)))
		for _, item := range items {
			hash = combineStructuralHash(hash, hashStructuralReadonly(item))
		}
		return hash
	case tagFastDict:
		panic("structural hashing rejects mutable FastDict expressions")
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagSymbol, tagCString, tagBString:
		return hashStructuralScalar(key)
	default:
		return HashKey(key)
	}
}

const structuralCatalogFastThreshold = 5

// structuralCatalog is mutable only during compile-local collection. Freeze
// copies its entries into an immutable lookup closure, so planner walkers need
// neither locks nor lazy memo writes.
type structuralCatalog struct {
	mu             sync.Mutex
	entries        []structuralIndexEntry
	buckets        map[uint64][]int
	forceCollision bool
}

func (catalog *structuralCatalog) hash(key Scmer, memo map[Scmer]uint64) uint64 {
	if catalog.forceCollision {
		return 0
	}
	return hashStructuralKey(key, memo)
}

func (catalog *structuralCatalog) set(key, value Scmer) {
	if catalog.buckets == nil {
		for i := range catalog.entries {
			if Equal(catalog.entries[i].key, key) {
				catalog.entries[i].value = value
				return
			}
		}
		catalog.entries = append(catalog.entries, structuralIndexEntry{key: key, value: value})
		if len(catalog.entries) < structuralCatalogFastThreshold {
			return
		}
		catalog.buckets = make(map[uint64][]int, len(catalog.entries))
		memo := make(map[Scmer]uint64)
		for i, entry := range catalog.entries {
			entryHash := catalog.hash(entry.key, memo)
			catalog.buckets[entryHash] = append(catalog.buckets[entryHash], i)
		}
		return
	}
	hash := uint64(0)
	if !catalog.forceCollision {
		hash = hashStructuralReadonly(key)
	}
	for _, i := range catalog.buckets[hash] {
		if Equal(catalog.entries[i].key, key) {
			catalog.entries[i].value = value
			return
		}
	}
	index := len(catalog.entries)
	catalog.entries = append(catalog.entries, structuralIndexEntry{key: key, value: value})
	catalog.buckets[hash] = append(catalog.buckets[hash], index)
}

func (catalog *structuralCatalog) insert(key, value Scmer) {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	catalog.set(key, value)
}

func (catalog *structuralCatalog) snapshot() []structuralIndexEntry {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	return append([]structuralIndexEntry(nil), catalog.entries...)
}

func frozenStructuralLookup(entries []structuralIndexEntry, forceCollision bool) Scmer {
	memo := make(map[Scmer]uint64)
	var buckets map[uint64][]structuralIndexEntry
	if len(entries) >= structuralCatalogFastThreshold {
		buckets = make(map[uint64][]structuralIndexEntry, len(entries))
		for _, entry := range entries {
			hash := uint64(0)
			if !forceCollision {
				hash = hashStructuralKey(entry.key, memo)
			}
			buckets[hash] = append(buckets[hash], entry)
		}
	}
	return NewFunc(func(args ...Scmer) Scmer {
		if len(args) != 1 {
			panic("frozen structural catalog lookup expects one key")
		}
		key := args[0]
		if key.GetTag() == tagFastDict {
			panic("frozen structural catalog rejects mutable FastDict keys")
		}
		if buckets == nil {
			for _, entry := range entries {
				if Equal(entry.key, key) {
					return entry.value
				}
			}
			return NewNil()
		}
		hash := uint64(0)
		if !forceCollision {
			if memoHash, ok := memo[key]; ok {
				hash = memoHash
			} else {
				hash = hashStructuralReadonly(key)
			}
		}
		for _, entry := range buckets[hash] {
			if Equal(entry.key, key) {
				return entry.value
			}
		}
		return NewNil()
	})
}

// NewStructuralCatalog returns an atomic compile-local collector. Calling it
// with (key value) inserts or replaces an equal key; calling it with no
// arguments publishes and returns a frozen read-only lookup closure. The
// optional constructor flag forces one bucket for collision tests.
func NewStructuralCatalog(a ...Scmer) Scmer {
	if len(a) > 1 {
		panic("make_structural_catalog expects an optional collision-test flag")
	}
	catalog := &structuralCatalog{forceCollision: len(a) == 1 && a[0].Bool()}
	return NewFunc(func(args ...Scmer) Scmer {
		switch len(args) {
		case 0:
			return frozenStructuralLookup(catalog.snapshot(), catalog.forceCollision)
		case 2:
			if args[0].GetTag() == tagFastDict {
				panic("structural catalog rejects mutable FastDict keys")
			}
			catalog.insert(args[0], args[1])
			return args[1]
		default:
			panic("structural catalog expects no arguments or key and value")
		}
	})
}

type structuralIndexEntry struct {
	key   Scmer
	value Scmer
}

// NewStructuralIndex eagerly indexes keys and every node below roots. The
// returned lookup closure performs read-only map access and is therefore safe
// for parallel planner walkers without a mutex.
func NewStructuralIndex(a ...Scmer) Scmer {
	if len(a) != 2 {
		panic("make_structural_index expects keys and roots")
	}
	keys := asSlice(a[0], "make_structural_index keys")
	roots := asSlice(a[1], "make_structural_index roots")
	memo := make(map[Scmer]uint64)
	entries := make(map[uint64][]structuralIndexEntry, len(keys))
	for i, key := range keys {
		hash := hashStructuralKey(key, memo)
		entries[hash] = append(entries[hash], structuralIndexEntry{key: key, value: NewInt(int64(i))})
	}
	for _, root := range roots {
		hashStructuralKey(root, memo)
	}
	return NewFunc(func(args ...Scmer) Scmer {
		if len(args) != 1 {
			panic("structural index lookup expects one expression")
		}
		expr := args[0]
		if expr.IsSourceInfo() {
			expr = expr.SourceInfo().value
		} else if expr.GetTag() == tagAny {
			if source, ok := expr.Any().(SourceInfo); ok {
				expr = source.value
			}
		}
		hash, ok := memo[expr]
		if !ok {
			if expr.GetTag() == tagSlice || expr.GetTag() == tagFastDict {
				panic("structural index lookup received an expression outside its roots")
			}
			hash = hashStructuralKey(expr, memo)
		}
		for _, entry := range entries[hash] {
			if Equal(entry.key, expr) {
				return entry.value
			}
		}
		return NewNil()
	})
}

func (d *FastDict) findPos(key Scmer, h uint64) (int, bool) {
	if d.index == nil {
		return -1, false
	}
	if pos, ok := d.index[h]; ok {
		if Equal(d.Pairs[pos], key) {
			return pos, true
		}
		for _, collisionPos := range d.collisions[h] {
			if Equal(d.Pairs[collisionPos], key) {
				return collisionPos, true
			}
		}
	}
	return -1, false
}

func (d *FastDict) Get(key Scmer) (Scmer, bool) {
	h := HashKey(key)
	if pos, ok := d.findPos(key, h); ok {
		return d.Pairs[pos+1], true
	}
	return NewNil(), false
}

// Set sets or merges a value for key. If merge is nil, it overwrites.
func (d *FastDict) Set(key, value Scmer, merge func(oldV, newV Scmer) Scmer) {
	if d.index == nil {
		d.index = make(map[uint64]int)
	}
	h := HashKey(key)
	if pos, ok := d.findPos(key, h); ok {
		if merge != nil {
			d.Pairs[pos+1] = merge(d.Pairs[pos+1], value)
		} else {
			d.Pairs[pos+1] = value
		}
		return
	}
	pos := len(d.Pairs)
	d.Pairs = append(d.Pairs, key, value)
	if _, exists := d.index[h]; exists {
		if d.collisions == nil {
			d.collisions = make(map[uint64][]int)
		}
		d.collisions[h] = append(d.collisions[h], pos)
	} else {
		d.index[h] = pos
	}
}

// Copy returns a deep copy of the FastDict (pairs and index).
func (d *FastDict) Copy() *FastDict {
	pairs := make([]Scmer, len(d.Pairs))
	copy(pairs, d.Pairs)
	idx := make(map[uint64]int, len(d.index))
	for h, pos := range d.index {
		idx[h] = pos
	}
	var collisions map[uint64][]int
	if len(d.collisions) > 0 {
		collisions = make(map[uint64][]int, len(d.collisions))
		for h, positions := range d.collisions {
			copied := make([]int, len(positions))
			copy(copied, positions)
			collisions[h] = copied
		}
	}
	return &FastDict{Pairs: pairs, index: idx, collisions: collisions}
}

func (d *FastDict) ToList() []Scmer { return d.Pairs }
