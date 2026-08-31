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

import (
	"strconv"
)

type foldableCSEInfo struct {
	hash   uint64
	stable bool
}

type foldableCSECandidate struct {
	expression Scmer
	first      *Scmer
	firstTop   int
	symbol     Symbol
	shared     bool
}

type foldableCSEScanner struct {
	byHash      map[uint64][]*foldableCSECandidate
	candidates  []*foldableCSECandidate
	generations map[Symbol]uint64
	shadowed    map[Symbol]bool
	nextSymbol  uint64
}

func foldableCSETypeIsImmutable(td *TypeDescriptor) bool {
	if td == nil {
		return false
	}
	switch td.Kind {
	case "string", "number", "int", "bool", "nil", "symbol":
		return true
	default:
		return false
	}
}

func (scanner *foldableCSEScanner) scalarInfo(value Scmer) foldableCSEInfo {
	hash := combineStructuralHash(0x510e527fade682d1, uint64(value.GetTag()))
	hash = combineStructuralHash(hash, HashKey(value))
	if symbol, ok := scmerSymbol(value); ok {
		hash = combineStructuralHash(hash, scanner.generations[symbol])
	}
	return foldableCSEInfo{hash: hash, stable: true}
}

func (scanner *foldableCSEScanner) replacementSymbol() Symbol {
	scanner.nextSymbol++
	// NUL cannot occur in a parsed Scheme identifier. It keeps the temporary
	// unobservable even before the normal numbered-local lowering runs.
	return Symbol("\x00foldable_cse_" + strconv.FormatUint(scanner.nextSymbol, 10))
}

func (scanner *foldableCSEScanner) share(location *Scmer, expression Scmer, hash uint64, top int, guaranteed bool) {
	for _, candidate := range scanner.byHash[hash] {
		if !astStructuralEqual(candidate.expression, expression) {
			continue
		}
		if !candidate.shared {
			candidate.symbol = scanner.replacementSymbol()
			*candidate.first = NewSymbol(string(candidate.symbol))
			candidate.shared = true
		}
		*location = NewSymbol(string(candidate.symbol))
		return
	}
	if !guaranteed {
		return
	}
	candidate := &foldableCSECandidate{
		expression: expression,
		first:      location,
		firstTop:   top,
	}
	scanner.byHash[hash] = append(scanner.byHash[hash], candidate)
	scanner.candidates = append(scanner.candidates, candidate)
}

func (scanner *foldableCSEScanner) walk(location *Scmer, top int, guaranteed bool) foldableCSEInfo {
	value := *location
	if value.IsSourceInfo() {
		return scanner.walk(&value.SourceInfo().value, top, guaranteed)
	}
	if !value.IsSlice() {
		switch value.GetTag() {
		case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagSymbol, tagNthLocalVar, tagCString, tagBString:
			return scanner.scalarInfo(value)
		default:
			return foldableCSEInfo{}
		}
	}

	items := value.Slice()
	if len(items) == 0 {
		return foldableCSEInfo{hash: combineStructuralHash(0x1f83d9abfb41bd6b, 0), stable: true}
	}
	head, headOK := scmerSymbol(items[0])
	if headOK {
		switch head {
		case "quote":
			return foldableCSEInfo{hash: hashASTReadonly(value), stable: true}
		case "lambda", "eval", "parser", "outer", "begin", "begin_mut", "!begin":
			return foldableCSEInfo{}
		case "define", "set", "setN":
			if len(items) >= 3 {
				scanner.walk(&items[2], top, guaranteed)
			}
			return foldableCSEInfo{}
		case "match", "match_mut":
			if len(items) >= 2 {
				scanner.walk(&items[1], top, guaranteed)
			}
			for i := 3; i < len(items); i += 2 {
				scanner.walk(&items[i], top, false)
			}
			if len(items)%2 == 1 {
				scanner.walk(&items[len(items)-1], top, false)
			}
			return foldableCSEInfo{}
		case "if", "and", "or", "coalesce", "coalesceNil":
			if len(items) >= 2 {
				scanner.walk(&items[1], top, guaranteed)
			}
			for i := 2; i < len(items); i++ {
				scanner.walk(&items[i], top, false)
			}
			return foldableCSEInfo{}
		}
	}

	declaration := DeclarationForValue(items[0])
	if headOK && (scanner.shadowed[head] || scanner.generations[head] > 0) {
		declaration = nil
	}
	if declaration == nil || declaration.IsSpecialForm {
		return foldableCSEInfo{}
	}
	hash := combineStructuralHash(0x5be0cd19137e2179, uint64(len(items)))
	headInfo := scanner.scalarInfo(items[0])
	hash = combineStructuralHash(hash, headInfo.hash)
	stable := declaration.IsFoldable() && !declaration.Type.HasSideEffects
	for i := 1; i < len(items); i++ {
		// A nested candidate in a later argument cannot be lifted in front of
		// arguments that Scheme evaluates before it. The complete outer call may
		// still become a candidate at its own location below.
		info := scanner.walk(&items[i], top, guaranteed && i == 1)
		hash = combineStructuralHash(hash, info.hash)
		parameterIndex := i - 1
		if parameterIndex >= len(declaration.Type.Params) {
			parameterIndex = len(declaration.Type.Params) - 1
		}
		var parameter *TypeDescriptor
		if parameterIndex >= 0 {
			parameter = declaration.Type.Params[parameterIndex]
		}
		stable = stable && info.stable && foldableCSETypeIsImmutable(parameter)
	}
	if stable && foldableCSETypeIsImmutable(declaration.Type.Return) {
		scanner.share(location, value, hash, top, guaranteed)
		return foldableCSEInfo{hash: hash, stable: true}
	}
	return foldableCSEInfo{}
}

// optimizeBeginFoldableCSE shares deterministic immutable calls within one
// lexical begin. Discovery, dependency hashing and rewriting happen in the
// same tree walk; structural comparisons are limited to equal hash buckets.
func optimizeBeginFoldableCSE(body []Scmer, bodyStart int, ome *optimizerMetainfo) ([]Scmer, int) {
	scanner := foldableCSEScanner{
		byHash:      make(map[uint64][]*foldableCSECandidate),
		generations: make(map[Symbol]uint64),
		shadowed:    make(map[Symbol]bool),
	}
	for symbol := range ome.variableReplacement {
		scanner.shadowed[symbol] = true
	}
	for symbol := range ome.variableTypes {
		scanner.shadowed[symbol] = true
	}
	for top := bodyStart; top < len(body); top++ {
		scanner.walk(&body[top], top, true)
		expression := body[top]
		if stripped, ok := scmerStripSourceInfo(expression); ok {
			expression = stripped
		}
		if items, ok := scmerSlice(expression); ok && len(items) >= 3 &&
			(scmerIsSymbol(items[0], "define") || scmerIsSymbol(items[0], "set")) {
			if symbol, ok := scmerSymbol(items[1]); ok {
				scanner.generations[symbol]++
			}
		}
	}

	insertions := make(map[int][]Scmer)
	count := 0
	for _, candidate := range scanner.candidates {
		if !candidate.shared {
			continue
		}
		definition := NewSlice([]Scmer{
			NewSymbol("define"),
			NewSymbol(string(candidate.symbol)),
			candidate.expression,
		})
		insertions[candidate.firstTop] = append(insertions[candidate.firstTop], definition)
		count++
	}
	if count == 0 {
		return body, 0
	}
	rewritten := make([]Scmer, 0, len(body)+count)
	rewritten = append(rewritten, body[:bodyStart]...)
	for top := bodyStart; top < len(body); top++ {
		rewritten = append(rewritten, insertions[top]...)
		rewritten = append(rewritten, body[top])
	}
	return rewritten, count
}
