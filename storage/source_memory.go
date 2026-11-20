/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// These facts belong to one immutable native source definition. Public catalog
// ownership and retained transaction history share its payload, not its charge.
// Readers inspect only the database aggregate; no reader follows this mutex or
// a callable/parameter graph while holding schema/table/shard ownership.
type sourceDefinitionMemory struct {
	mu         sync.Mutex
	owner      *database
	references uint
	bytes      uint
}

// Validate the portable declaration DATA and count the actual owned graph in
// that same admission traversal. This never introspects a Proc, evaluates syntax
// or invokes a value's size callback. Frozen/decoded dictionaries are lists; the
// FastDict arm also preserves the existing accepted input data domain.
func sourceParameterMemory(value scm.Scmer) (uint, bool) {
	slot := uint(unsafe.Sizeof(scm.Scmer{}))
	bytes := slot
	if value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() {
		return bytes, true
	}
	if value.IsString() {
		return bytes + uint(len(value.String())), true
	}
	var children []scm.Scmer
	if value.IsSlice() {
		children = value.Slice()
	} else if value.IsFastDict() {
		if dict := value.FastDict(); dict != nil {
			children = dict.Pairs
			bytes += uint(unsafe.Sizeof(*dict)) + publicCatalogMapFootprint((cap(children)+1)/2)
		}
	} else {
		return 0, false
	}
	bytes += uint(cap(children)) * slot
	for _, child := range children {
		part, plain := sourceParameterMemory(child)
		if !plain {
			return 0, false
		}
		bytes += part - slot // the backing vector already owns the inline slot
	}
	return bytes, true
}

func (definition *ProcedureDescription) initializeSourceMemory() error {
	parameters, plain := sourceParameterMemory(definition.Parameters)
	if !plain {
		return fmt.Errorf("procedure parameters must contain only portable declaration data")
	}
	definition.memory.bytes = uint(unsafe.Sizeof(*definition)) +
		uint(len(definition.Name)+len(definition.Language)+len(definition.Source)) +
		parameters - uint(unsafe.Sizeof(scm.Scmer{}))
	return nil
}

// Called only by the actual outside-lock source compiler producer. This is an
// INITIAL callable estimate, like the finite FK descriptor; subsequent runtime
// specialization/JIT growth is not claimed by this source-metadata estimate.
func (definition *ProcedureDescription) publishInitialCallableMemory(callable scm.Scmer) {
	definition.memory.publishInitialCallableMemory(callable)
}

func (memory *sourceDefinitionMemory) publishInitialCallableMemory(callable scm.Scmer) {
	bytes := scm.ComputeSize(callable)
	memory.mu.Lock()
	defer memory.mu.Unlock()
	memory.bytes += bytes
	if owner := memory.owner; owner != nil {
		owner.adjustMetadataMemory(0, bytes)
	}
}

func (definition *ProcedureDescription) retainSourceMemory(db *database) {
	definition.memory.retain(db)
}

func (memory *sourceDefinitionMemory) retain(db *database) {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	if memory.references != 0 {
		if memory.owner != db {
			panic("source definition memory belongs to another database")
		}
		memory.references++
		return
	}
	memory.owner = db
	memory.references = 1
	db.adjustMetadataMemory(0, memory.bytes)
}

func (definition *ProcedureDescription) releaseSourceMemory(db *database) {
	definition.memory.release(db)
}

func (memory *sourceDefinitionMemory) release(db *database) {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	if memory.owner != db || memory.references == 0 {
		panic("source definition memory lost its database owner")
	}
	memory.references--
	if memory.references == 0 {
		db.adjustMetadataMemory(memory.bytes, 0)
		memory.owner = nil
	}
}

// Loading/transferring a decoded private schema keeps the same owned graph and
// numeric descriptor; it never sizes it again beneath the publication owner.
func (definition *ProcedureDescription) moveSourceMemoryOwner(db *database) {
	definition.memory.moveOwner(db)
}

func (memory *sourceDefinitionMemory) moveOwner(db *database) {
	memory.mu.Lock()
	defer memory.mu.Unlock()
	if memory.references != 1 {
		panic("source definition transfer requires one private catalog owner")
	}
	if memory.owner != db {
		if previous := memory.owner; previous != nil {
			previous.adjustMetadataMemory(memory.bytes, 0)
		}
		memory.owner = db
		db.adjustMetadataMemory(0, memory.bytes)
	}
}

// A database owns its source declarations and native catalog resource headers.
// Producers publish numeric deltas; diagnostics never walk a transaction graph.
func (db *database) adjustMetadataMemory(before, after uint) {
	if after >= before {
		db.metadataMemoryBytes.Add(uint64(after - before))
	} else {
		db.metadataMemoryBytes.Add(^uint64(before-after) + 1)
	}
}
