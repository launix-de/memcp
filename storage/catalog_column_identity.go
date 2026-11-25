/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "unsafe"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// This is the immutable configured origin of one generated-ID COLUMN. The
// table allocator's older Start/Issued provenance remains unchanged.
type columnIDGeneration struct {
	Version int   `json:"version"`
	Start   int64 `json:"start"`
}

func (generation *columnIDGeneration) validate() error {
	if generation == nil || generation.Version != 1 || generation.Start < 1 {
		return fmt.Errorf("invalid native generated-ID declaration")
	}
	return nil
}

type nativeColumnIDInitialization struct {
	token   string
	start   int64
	maximum int64
}

func decodeColumnIDInitialization(value scm.Scmer) *nativeColumnIDInitialization {
	parts := mustScmerSlice(value, "native table-ID initializer")
	if len(parts) != 5 || !parts[0].IsString() || parts[0].String() != "table-id" ||
		!parts[1].IsInt() || parts[1].Int() != 1 || !parts[2].IsString() || parts[2].String() == "" ||
		!parts[3].IsString() || !parts[4].IsString() {
		panic("table-ID initializer requires version, observation token, start and upper bound")
	}
	d := &nativeColumnIDInitialization{parts[2].String(), allocatorSignedDecimal(parts[3].String()), allocatorSignedDecimal(parts[4].String())}
	if d.start < 1 || d.maximum < d.start {
		panic("table-ID initializer requires a positive start within its declared bound")
	}
	return d
}

// Column and allocator selection are one history amendment. Existing explicit
// reseeds retain their original entry; no visitor interprets a dual-kind entry.
func (entry catalogDeclarationAmendment) allocatorBranch() *catalogAllocatorReset {
	if entry.allocator != nil {
		return entry.allocator
	}
	if entry.column != nil {
		return entry.column.allocator
	}
	return nil
}

func (initializer *nativeColumnInitializer) prepareIDBranch(columns []*column) {
	if initializer.ids == nil {
		return
	}
	for _, c := range columns {
		if c.AutoIncrement {
			panic("generated-ID append requires a table without an identity column")
		}
	}
	t, tx := initializer.target, initializer.operation.invocation
	state := t.currentTableAllocator(tx)
	if tableAllocatorToken(state) != initializer.ids.token || !state.IssuedKnown || state.Issued {
		panic("generated-ID append requires its observed proven-unused native allocator")
	}
	state.Version, state.Epoch = 2, uuid.NewString()
	state.Current, state.Next, state.HasNext = initializer.ids.start, initializer.ids.start, true
	if err := state.validate(); err != nil {
		panic(err)
	}
	branch := &catalogAllocatorReset{table: t, column: initializer.column.Name}
	branch.state.Store(&state)
	branch.bytes = uint(unsafe.Sizeof(*branch)) + uint(unsafe.Sizeof(state)) + uint(len(branch.column)+len(state.Table)+len(state.Epoch))
	initializer.allocator = branch
	initializer.column.generatedIDs = &columnIDGeneration{1, initializer.ids.start}
}

type nativeIdentityShardCount struct {
	shard *storageShard
	count uint64
}

type nativeIdentityShardRange struct {
	first int64
	count uint64
}

func identityCountParts(value scm.Scmer) []scm.Scmer {
	if value.IsNil() {
		return nil
	}
	if value.IsSlice() {
		return value.Slice()
	}
	if _, ok := value.Any().(*nativeIdentityShardCount); ok {
		return []scm.Scmer{value}
	}
	return mustScmerSlice(value, "native identity input counts")
}

// The same initializer admission pins actual visible input generations in the
// original journal. Only a bounded tuple per shard leaves this scan. A second
// scan uses its ordinary LOCAL reducer accumulator as the range offset; there
// is no shared row counter and no per-row allocator API/constructor.
func (initializer *nativeColumnInitializer) prepareIDRanges() {
	if initializer.ids == nil {
		return
	}
	t, tx := initializer.target, initializer.operation.invocation
	access := scanAccess{runtime: &scanAccessRuntime{initializer: initializer}}
	counts := t.scanWithBatchFrom(tx, nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, access, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"$record_ref"},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			ref := recordRefFromScmer(values[1])
			count := uint64(1)
			if !values[0].IsNil() {
				previous, ok := values[0].Any().(*nativeIdentityShardCount)
				if !ok || previous.shard != ref.shard {
					panic("native identity input lost its shard-local accumulator")
				}
				count += previous.count
			}
			return scm.NewAny(&nativeIdentityShardCount{ref.shard, count})
		}), scm.NewNil(), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewSlice(append(append([]scm.Scmer(nil), identityCountParts(values[0])...), identityCountParts(values[1])...))
		}), false, 0, nil)
	var inputs []*nativeIdentityShardCount
	var total uint64
	for _, value := range identityCountParts(counts) {
		input, ok := value.Any().(*nativeIdentityShardCount)
		if !ok || input.shard == nil || input.shard.t != t || input.count == 0 {
			panic("generated-ID append lost its native visible input")
		}
		if input.count > uint64(initializer.ids.maximum-initializer.ids.start)+1-total {
			panic("generated-ID historical range exceeds its declared bound")
		}
		total += input.count
		inputs = append(inputs, input)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].shard.uuid.String() < inputs[j].shard.uuid.String() })
	reservation := initializer.allocator.reserve(tableIDExplicit{}, total)
	initializer.idRanges = make(map[*storageShard]nativeIdentityShardRange, len(inputs))
	first := reservation.First
	for _, input := range inputs {
		if _, duplicate := initializer.idRanges[input.shard]; duplicate {
			panic("generated-ID append received a native generation twice")
		}
		initializer.idRanges[input.shard] = nativeIdentityShardRange{first, input.count}
		// An exhausted final range has no representable successor. Never add
		// one after the last shard or use unsigned predecessor arithmetic.
		if input != inputs[len(inputs)-1] {
			first += int64(input.count)
		}
	}
	initializer.idCount = total
}

func (initializer *nativeColumnInitializer) historicalID(ref recMapTarget, offset int64) scm.Scmer {
	rangeForShard, ok := initializer.idRanges[ref.shard]
	if !ok || offset < 0 || uint64(offset) >= rangeForShard.count {
		panic("generated-ID append input changed after its bounded reservation")
	}
	return scm.NewInt(rangeForShard.first + offset)
}

func (record *catalogDecisionRecord) hasIdentityAppends() bool {
	for _, d := range record.ColumnAppends {
		if d.GeneratedIDs != nil {
			return true
		}
	}
	return false
}

func (record *catalogDecisionRecord) validateIdentityAppendProfile() error {
	seen := make(map[string]bool)
	for _, d := range record.ColumnAppends {
		if d.GeneratedIDs == nil {
			continue
		}
		if record.Format != 31 || d.GeneratedIDs.validate() != nil || d.Type.Nullable || d.Initialization != nil || d.RowInitialization != nil || seen[d.Identity] {
			return fmt.Errorf("invalid required generated-ID append profile")
		}
		seen[d.Identity] = true
		found := false
		for _, reset := range record.AllocatorResets {
			found = found || reset.Table == d.Table && reset.Column == d.Name && reset.State.Table == d.Identity && reset.State.Version == 2
		}
		if !found {
			return fmt.Errorf("generated-ID append lost its selected native allocator epoch")
		}
	}
	return nil
}
