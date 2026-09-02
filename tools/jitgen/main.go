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

// jitgen reads Go source files, finds Declare() calls, builds SSA for the
// operator function bodies, and generates JIT emitter closures.
//
// Usage:
//
//	go run ./tools/jitgen/ scm/alu.go                    # list operators
//	go run ./tools/jitgen/ -dump=+ scm/alu.go             # SSA dump for +
//	go run ./tools/jitgen/ -patch scm/alu.go              # patch source
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var dumpOp string
var onlyOp string
var doPatch bool
var doWipe bool
var verbose bool
var missingOnly bool
var policyOnly bool
var jobs = runtime.GOMAXPROCS(0)

const generatedBanner = "/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */"
const phiSlotBytes = 16
const phiStoreChunkSize = 3
const inlineInstructionBudget = 256
const emitterInlineInstructionBudget = 2048

func main() {
	var files []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-dump=") {
			dumpOp = arg[len("-dump="):]
		} else if strings.HasPrefix(arg, "-only=") {
			onlyOp = arg[len("-only="):]
		} else if arg == "-patch" {
			doPatch = true
		} else if arg == "-wipe" {
			doWipe = true
		} else if arg == "-missing-only" {
			missingOnly = true
		} else if arg == "-policy-only" {
			policyOnly = true
		} else if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else if strings.HasPrefix(arg, "-jobs=") {
			parsed, err := strconv.Atoi(strings.TrimPrefix(arg, "-jobs="))
			if err != nil || parsed < 1 {
				fmt.Fprintf(os.Stderr, "invalid worker count %q\n", arg)
				os.Exit(1)
			}
			jobs = parsed
		} else {
			files = append(files, arg)
		}
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "usage: jitgen [-dump=OP] [-patch] [-missing-only] [-policy-only] [-wipe] [-jobs=N] <file.go> ...\n")
		os.Exit(1)
	}

	if doWipe {
		wipeFiles(files)
		return
	}

	// Determine package from file paths
	pkgDir := "./" + filepath.Dir(files[0])
	overlay, preservedEmitterOffsets, err := generatedEmitterOverlay(pkgDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare source overlay: %v\n", err)
		os.Exit(1)
	}

	// Load package with full type info for SSA
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports | packages.NeedName,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, pkgDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load package: %v\n", err)
		os.Exit(1)
	}
	if len(pkgs) == 0 {
		fmt.Fprintf(os.Stderr, "no packages found\n")
		os.Exit(1)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		hardErr := false
		for _, e := range pkg.Errors {
			if doPatch {
				// Patch mode must tolerate temporarily broken generated sections.
				// We still proceed and let per-function generation decide what can
				// be rewritten in this run.
				continue
			}
			msg := e.Error()
			if strings.Contains(msg, "declared and not used") {
				// Regenerating from a temporarily inconsistent generated file is
				// allowed; the patch pass will rewrite these sections.
				continue
			}
			if strings.Contains(msg, "imported and not used") {
				// The analysis overlay replaces generated emitters with stubs; an
				// import referenced only by generated code can therefore look unused.
				continue
			}
			if strings.Contains(msg, "missing return") {
				// Transitional state while generated emitters are being rewritten.
				// Patch mode will replace these sections in the same run.
				continue
			}
			hardErr = true
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		if hardErr {
			os.Exit(1)
		}
	}
	fset := pkg.Fset

	// Build SSA
	prog, _ := ssautil.AllPackages(pkgs, 0)
	prog.Build()

	// Index all SSA functions by source position.
	// Prefer non-synthetic functions when multiple share the same position
	// (e.g. method vs thunk).
	ssaFuncs := map[token.Pos]*ssa.Function{}
	ssaFuncsByName := map[string]*ssa.Function{}
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Pkg == nil || fn.Pkg.Pkg == nil || fn.Pkg.Pkg.Path() != pkg.PkgPath || fn.Synthetic != "" {
			continue
		}
		// A bare Go identifier in Declaration.Fn can only denote a package-level
		// function. Do not let an unrelated method with the same name win this
		// index nondeterministically.
		if fn.Signature == nil || fn.Signature.Recv() == nil {
			ssaFuncsByName[fn.Name()] = fn
		}
		if fn.Pos().IsValid() {
			if existing, ok := ssaFuncs[fn.Pos()]; ok {
				// Keep the non-synthetic one (real function, not thunk)
				if existing.Synthetic != "" && fn.Synthetic == "" {
					ssaFuncs[fn.Pos()] = fn
				}
			} else {
				ssaFuncs[fn.Pos()] = fn
			}
		}
	}

	// Which files to process
	absFiles := map[string]bool{}
	for _, f := range files {
		abs, _ := filepath.Abs(f)
		absFiles[abs] = true
	}

	// Collect operators from AST (for patching byte offsets)
	var ops []operatorInfo
	var stInfos []storageInfo
	for _, astFile := range pkg.Syntax {
		fname := fset.Position(astFile.Pos()).Filename
		abs, _ := filepath.Abs(fname)
		if !absFiles[abs] {
			continue
		}
		ops = append(ops, collectOperators(fset, astFile, fname)...)
		stInfos = append(stInfos, collectStorageMethods(fset, astFile, fname)...)
	}
	for i := range ops {
		if ops[i].jitExpr == nil {
			continue
		}
		path, _ := filepath.Abs(ops[i].path)
		offset := fset.Position(ops[i].jitExpr.Pos()).Offset
		ops[i].preservedEnd = preservedEmitterOffsets[path][offset]
	}

	// Generate operator emitters in parallel after the package and SSA graph have
	// been built once. Patches are still assembled and applied serially below so
	// source rewriting and diagnostics remain deterministic.
	generated := generateOperators(ops, ssaFuncs, ssaFuncsByName)

	// Process each operator (pattern 1: Declare)
	patches := map[string][]patchEntry{}
	for opIndex, op := range ops {
		if onlyOp != "" && op.name != onlyOp {
			continue
		}
		if operatorHasCustomJITEmit(op) {
			continue
		}
		generation := generated[opIndex]
		ssaFn := generation.ssaFn
		if ssaFn == nil {
			fmt.Fprintf(os.Stderr, "  %s: %s — SSA function not found\n", op.path, op.name)
			continue
		}

		if dumpOp == op.name {
			dumpSSA(ssaFn)
		}

		newText, genErr := generation.newText, generation.genErr
		if genErr != "" && verbose {
			fmt.Fprintln(os.Stderr, newText)
		}
		usesFallback := false
		usesNativeCall := false
		inlineCallbacks := hasDynamicSSACall(ssaFn)
		if reason := interfaceAssertionBoundaryReason(ssaFn); reason != "" {
			usesNativeCall = true
			inlineCallbacks = false
			fmt.Printf("  %s: %s CALL: %s\n", op.path, op.name, reason)
			newText = generateNativeCallClosure(op.name, reason)
		} else if genErr == "" {
			fmt.Printf("  %s: %s OK\n", op.path, op.name)
		} else if reason := nativeCallBoundaryReason(ssaFn); reason != "" {
			usesNativeCall = true
			inlineCallbacks = false
			fmt.Printf("  %s: %s CALL: %s (%s)\n", op.path, op.name, reason, genErr)
			newText = generateNativeCallClosure(op.name, reason)
		} else {
			usesFallback = true
			fmt.Printf("  %s: %s FALLBACK: %s\n", op.path, op.name, genErr)
			if verbose {
				dumpSSA(ssaFn)
			}
			newText = generateFallbackClosure(op.name)
		}
		inlineCost := generation.inlineCost
		if usesFallback || usesNativeCall {
			inlineCost = math.MaxUint16
		}
		// Declaration emitters are expressions nested one indentation level below
		// their TypeDescriptor field. Keep generated output gofmt-stable both when
		// inserting a new JITEmit field and when replacing an existing closure.
		newText = strings.ReplaceAll(newText, "\n", "\n\t")

		if doPatch {
			if missingOnly && !operatorJITEmitMissing(op) {
				continue
			}
			if policyOnly {
				costText := strconv.FormatUint(uint64(inlineCost), 10)
				if op.jitInlineCostExpr != nil {
					pos := fset.Position(op.jitInlineCostExpr.Pos())
					end := fset.Position(op.jitInlineCostExpr.End())
					patches[op.path] = append(patches[op.path], patchEntry{startOff: pos.Offset, endOff: end.Offset, newText: costText, opName: op.name + ".JITInlineCost"})
				} else if op.typeInsertPos.IsValid() {
					insertPos := fset.Position(op.typeInsertPos)
					patches[op.path] = append(patches[op.path], patchEntry{startOff: insertPos.Offset, endOff: insertPos.Offset, newText: "\tJITInlineCost: " + costText + ",\n\t\t", opName: op.name + ".JITInlineCost"})
				}
				continue
			}
			var pos, end token.Position
			if op.jitExpr != nil {
				pos = fset.Position(op.jitExpr.Pos())
				end = fset.Position(op.jitExpr.End())
				if op.preservedEnd > 0 {
					end.Offset = op.preservedEnd
				}
			} else {
				pos = fset.Position(op.jitInsertPos)
				end = pos
				newText = "\tJITEmit: " + newText + ",\n\t\t"
				if usesFallback && op.jitVirtualExpr == nil {
					newText += "JITVirtualArgs: true,\n\t\t"
				}
			}
			patches[op.path] = append(patches[op.path], patchEntry{
				startOff: pos.Offset,
				endOff:   end.Offset,
				newText:  newText,
				opName:   op.name,
			})
			if (usesFallback || usesNativeCall) && op.jitExpr != nil {
				if op.jitVirtualExpr != nil {
					virtualPos := fset.Position(op.jitVirtualExpr.Pos())
					virtualEnd := fset.Position(op.jitVirtualExpr.End())
					patches[op.path] = append(patches[op.path], patchEntry{
						startOff: virtualPos.Offset,
						endOff:   virtualEnd.Offset,
						newText:  "true",
						opName:   op.name + ".JITVirtualArgs",
					})
				} else if op.typeInsertPos.IsValid() {
					insertPos := fset.Position(op.typeInsertPos)
					patches[op.path] = append(patches[op.path], patchEntry{
						startOff: insertPos.Offset,
						endOff:   insertPos.Offset,
						newText:  "\tJITVirtualArgs: true,\n\t\t",
						opName:   op.name + ".JITVirtualArgs",
					})
				}
			}
			if hasDynamicSSACall(ssaFn) {
				value := "false"
				if inlineCallbacks && !usesFallback {
					value = "true"
				}
				if op.jitInlineCallbacksExpr != nil {
					pos := fset.Position(op.jitInlineCallbacksExpr.Pos())
					end := fset.Position(op.jitInlineCallbacksExpr.End())
					patches[op.path] = append(patches[op.path], patchEntry{startOff: pos.Offset, endOff: end.Offset, newText: value, opName: op.name + ".JITInlineCallbacks"})
				} else if op.typeInsertPos.IsValid() {
					insertPos := fset.Position(op.typeInsertPos)
					patches[op.path] = append(patches[op.path], patchEntry{startOff: insertPos.Offset, endOff: insertPos.Offset, newText: "\tJITInlineCallbacks: " + value + ",\n\t\t", opName: op.name + ".JITInlineCallbacks"})
				}
			}
			costText := strconv.FormatUint(uint64(inlineCost), 10)
			if op.jitInlineCostExpr != nil {
				pos := fset.Position(op.jitInlineCostExpr.Pos())
				end := fset.Position(op.jitInlineCostExpr.End())
				patches[op.path] = append(patches[op.path], patchEntry{startOff: pos.Offset, endOff: end.Offset, newText: costText, opName: op.name + ".JITInlineCost"})
			} else if op.typeInsertPos.IsValid() {
				insertPos := fset.Position(op.typeInsertPos)
				patches[op.path] = append(patches[op.path], patchEntry{startOff: insertPos.Offset, endOff: insertPos.Offset, newText: "\tJITInlineCost: " + costText + ",\n\t\t", opName: op.name + ".JITInlineCost"})
			}
		}
	}

	// Process each storage type (pattern 2: ColumnStorage.GetValue → JITEmit)
	for _, si := range stInfos {
		if policyOnly {
			continue
		}
		if onlyOp != "" && si.typeName != onlyOp && si.typeName+".GetValue" != onlyOp {
			continue
		}
		ssaFn := ssaFuncs[si.getValuePos]
		if ssaFn == nil {
			fmt.Fprintf(os.Stderr, "  %s: %s.GetValue — SSA function not found\n", si.path, si.typeName)
			continue
		}

		if dumpOp == si.typeName || dumpOp == si.typeName+".GetValue" {
			dumpSSA(ssaFn)
		}

		newText, genErr := generateStorageBody(si.typeName, ssaFn, nil)
		if genErr == "" {
			fmt.Printf("  %s: %s.GetValue OK\n", si.path, si.typeName)
		} else {
			fmt.Printf("  %s: %s.GetValue FALLBACK: %s\n", si.path, si.typeName, genErr)
			if verbose {
				dumpSSA(ssaFn)
			}
			// Fallback: emit a Go call to GetValue (unbound method, receiver as first arg)
			newText = "\n\t/* TODO: " + genErr + " */\n" +
				"\treturn ctx.EmitGoCallScalar(scm.GoFuncAddr((*" + si.typeName + ").GetValue), []scm.JITValueDesc{thisptr, idx}, 2)\n"
		}

		if doPatch {
			// Patch body of JITEmit method (between { and })
			bodyStart := fset.Position(si.jitEmitBody.Lbrace).Offset + 1
			bodyEnd := fset.Position(si.jitEmitBody.Rbrace).Offset
			patches[si.path] = append(patches[si.path], patchEntry{
				startOff: bodyStart,
				endOff:   bodyEnd,
				newText:  "\n" + newText,
				opName:   si.typeName + ".JITEmit",
			})
		}
	}

	if doPatch {
		for path, plist := range patches {
			applyPatches(path, plist)
		}
	}
}

type operatorGeneration struct {
	ssaFn      *ssa.Function
	newText    string
	genErr     string
	inlineCost uint16
}

func generateOperators(ops []operatorInfo, ssaFuncs map[token.Pos]*ssa.Function, ssaFuncsByName map[string]*ssa.Function) []operatorGeneration {
	results := make([]operatorGeneration, len(ops))
	work := make(chan int)
	workerCount := jobs
	if workerCount > len(ops) {
		workerCount = len(ops)
	}
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for index := range work {
				op := ops[index]
				if onlyOp != "" && op.name != onlyOp {
					continue
				}
				if operatorHasCustomJITEmit(op) {
					continue
				}
				var fn *ssa.Function
				if op.funcLit != nil {
					fn = ssaFuncs[op.funcLit.Pos()]
				} else {
					fn = ssaFuncsByName[op.funcName]
				}
				result := operatorGeneration{ssaFn: fn}
				if fn != nil {
					result.newText, result.genErr, result.inlineCost = generateClosureCost(op.name, fn, nil, op.path)
					if result.genErr == "" {
						if _, err := parser.ParseExpr(result.newText); err != nil {
							result.genErr = "generated invalid Go expression: " + err.Error()
						}
					}
				}
				results[index] = result
			}
		}()
	}
	for index := range ops {
		work <- index
	}
	close(work)
	workers.Wait()
	return results
}

// generatedEmitterOverlay replaces Declaration emitter expressions only for
// SSA package loading. Their source spans and line breaks stay intact, so patch
// offsets remain valid and x/tools only analyzes builtin Fn bodies.
func generatedEmitterOverlay(pkgDir string) (map[string][]byte, map[string]map[int]int, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil, nil, err
	}
	overlay := map[string][]byte{}
	preserved := map[string]map[int]int{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path, err := filepath.Abs(filepath.Join(pkgDir, entry.Name()))
		if err != nil {
			return nil, nil, err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			return nil, nil, err
		}
		stub := "func(*JITContext,[]Scmer,[]JITValueDesc,JITValueDesc)JITValueDesc{panic(0)},"
		for _, imported := range file.Imports {
			if imported.Path.Value == `"unsafe"` {
				stub = "func(*JITContext,[]Scmer,[]JITValueDesc,JITValueDesc)JITValueDesc{_=unsafe.Pointer(nil);panic(0)},"
				break
			}
		}
		patched := append([]byte(nil), source...)
		changed := false
		ast.Inspect(file, func(node ast.Node) bool {
			kv, ok := node.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "JITEmit" {
				return true
			}
			start := fset.Position(kv.Value.Pos()).Offset
			end := fset.Position(kv.Value.End()).Offset
			if start < 0 || end > len(source) || start >= end {
				return true
			}
			if _, ok := kv.Value.(*ast.FuncLit); !ok {
				return true
			}
			replaceEnd := end
			if replaceEnd < len(source) && source[replaceEnd] == ',' {
				replaceEnd++
			}
			span := patched[start:replaceEnd]
			firstNewline := bytes.IndexByte(span, '\n')
			if firstNewline >= 0 && firstNewline < len(stub) {
				return true
			}
			for i := range span {
				if span[i] != '\n' && span[i] != '\r' {
					span[i] = ' '
				}
			}
			copy(span, stub)
			if preserved[path] == nil {
				preserved[path] = map[int]int{}
			}
			preserved[path][start] = end
			changed = true
			return false
		})
		if changed {
			overlay[path] = patched
		}
	}
	return overlay, preserved, nil
}

func operatorJITEmitMissing(op operatorInfo) bool {
	if op.preservedEnd > 0 {
		return false
	}
	if op.jitExpr == nil {
		return true
	}
	ident, ok := op.jitExpr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func operatorHasCustomJITEmit(op operatorInfo) bool {
	if op.jitExpr == nil {
		return false
	}
	if _, generated := op.jitExpr.(*ast.FuncLit); generated {
		return false
	}
	ident, ok := op.jitExpr.(*ast.Ident)
	return !ok || ident.Name != "nil"
}

func dynamicSSACalls(fn *ssa.Function) []*ssa.Call {
	var calls []*ssa.Call
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok || call.Call.StaticCallee() != nil {
				continue
			}
			if _, builtin := call.Call.Value.(*ssa.Builtin); builtin {
				continue
			}
			calls = append(calls, call)
		}
	}
	return calls
}

func hasDynamicSSACall(fn *ssa.Function) bool {
	return len(dynamicSSACalls(fn)) != 0
}

// interfaceAssertionBoundaryReason keeps interface type assertions in Go.
// Their itab/data pair and comma-ok result must remain one GC-visible native
// operation; splitting the tuple across generated CFG paths can expose a
// partially live interface at a safepoint. The surrounding Scheme expression
// is still JIT compiled and calls this declaration through its compact native
// boundary.
func interfaceAssertionBoundaryReason(fn *ssa.Function) string {
	seen := map[*ssa.Function]bool{}
	var inspect func(*ssa.Function) bool
	inspect = func(current *ssa.Function) bool {
		if current == nil || seen[current] {
			return false
		}
		seen[current] = true
		for _, block := range current.Blocks {
			for _, instr := range block.Instrs {
				assertion, ok := instr.(*ssa.TypeAssert)
				if !ok {
					continue
				}
				if _, isInterface := assertion.AssertedType.Underlying().(*types.Interface); isInterface {
					return true
				}
			}
		}
		for _, nested := range current.AnonFuncs {
			if inspect(nested) {
				return true
			}
		}
		return false
	}
	if inspect(fn) {
		return "interface type assertion"
	}
	return ""
}

// nativeCallBoundaryReason identifies Go semantics that cannot safely execute
// inside generated machine code. These operations retain a deliberate native
// call boundary instead of being reported as an accidental JITGen fallback.
func nativeCallBoundaryReason(fn *ssa.Function) string {
	if fn != nil && fn.Signature != nil {
		params := fn.Signature.Params()
		results := fn.Signature.Results()
		variadicScmer := fn.Signature.Variadic() && params.Len() == 1 && func() bool {
			slice, ok := params.At(0).Type().Underlying().(*types.Slice)
			return ok && isScmerType(slice.Elem())
		}()
		if !variadicScmer || results.Len() != 1 || !isScmerType(results.At(0).Type()) {
			return "non-Scheme native declaration signature"
		}
	}
	seen := map[*ssa.Function]bool{}
	var inspect func(*ssa.Function) string
	inspect = func(current *ssa.Function) string {
		if current == nil || seen[current] {
			return ""
		}
		seen[current] = true
		for _, block := range current.Blocks {
			for _, instr := range block.Instrs {
				switch value := instr.(type) {
				case *ssa.MakeChan:
					return "channel construction"
				case *ssa.Go:
					return "goroutine launch"
				case *ssa.Defer:
					return "deferred call"
				case *ssa.Select:
					return "channel select"
				case *ssa.Range:
					return "native range iterator"
				case *ssa.MakeClosure:
					return "escaping or recursive Go closure"
				case *ssa.Call:
					for _, arg := range value.Call.Args {
						if _, ok := arg.(*ssa.Function); ok {
							return "static Go callback value"
						}
					}
				}
			}
		}
		for _, nested := range current.AnonFuncs {
			if reason := inspect(nested); reason != "" {
				return reason
			}
		}
		return ""
	}
	return inspect(fn)
}

func generateFallbackClosure(opName string) string {
	return fmt.Sprintf(`func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	return jitEmitGoVariadicCallFromDescs(ctx, declarations[%q].Fn, args, result)
}`, opName)
}

func generateNativeCallClosure(opName, reason string) string {
	return fmt.Sprintf(`func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	// JITGen native call boundary: %s.
	return jitEmitGoVariadicCallFromDescs(ctx, declarations[%q].Fn, args, result)
}`, reason, opName)
}

// --- AST operator collection (for patching byte offsets) ---

type operatorInfo struct {
	name                   string
	path                   string
	line                   int
	funcLit                *ast.FuncLit
	funcName               string
	jitExpr                ast.Expr
	jitVirtualExpr         ast.Expr
	jitInlineCallbacksExpr ast.Expr
	jitInlineCostExpr      ast.Expr
	jitInsertPos           token.Pos
	typeInsertPos          token.Pos
	preservedEnd           int
}

func keyedValue(comp *ast.CompositeLit, name string) ast.Expr {
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if ok && key.Name == name {
			return kv.Value
		}
	}
	return nil
}

func collectOperators(fset *token.FileSet, f *ast.File, path string) []operatorInfo {
	var ops []operatorInfo
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "Declare" || len(call.Args) < 2 {
			return true
		}
		unary, ok := call.Args[1].(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			return true
		}
		comp, ok := unary.X.(*ast.CompositeLit)
		if !ok {
			return true
		}

		var nameExpr, fnExpr, jitExpr, jitVirtualExpr, jitInlineCallbacksExpr, jitInlineCostExpr ast.Expr
		var jitInsertPos token.Pos
		var typeInsertPos token.Pos
		if len(comp.Elts) > 0 {
			if _, keyed := comp.Elts[0].(*ast.KeyValueExpr); keyed {
				nameExpr = keyedValue(comp, "Name")
				fnExpr = keyedValue(comp, "Fn")
				typeExpr := keyedValue(comp, "Type")
				if unaryType, ok := typeExpr.(*ast.UnaryExpr); ok && unaryType.Op == token.AND {
					if typeComp, ok := unaryType.X.(*ast.CompositeLit); ok {
						kind, ok := stringLiteral(keyedValue(typeComp, "Kind"))
						if !ok || kind != "func" {
							return true
						}
						jitExpr = keyedValue(typeComp, "JITEmit")
						jitVirtualExpr = keyedValue(typeComp, "JITVirtualArgs")
						jitInlineCallbacksExpr = keyedValue(typeComp, "JITInlineCallbacks")
						jitInlineCostExpr = keyedValue(typeComp, "JITInlineCost")
						typeInsertPos = typeComp.Rbrace
						if jitExpr == nil {
							jitInsertPos = typeComp.Rbrace
						}
					}
				}
			} else if len(comp.Elts) >= 11 {
				nameExpr = comp.Elts[0]
				fnExpr = comp.Elts[6]
				jitExpr = comp.Elts[10]
			}
		}
		nameLit, ok := nameExpr.(*ast.BasicLit)
		if !ok || nameLit.Kind != token.STRING || (jitExpr == nil && !jitInsertPos.IsValid()) {
			return true
		}
		funcLit, isLiteral := fnExpr.(*ast.FuncLit)
		funcName := ""
		if ident, ok := fnExpr.(*ast.Ident); ok {
			funcName = ident.Name
		}
		if !isLiteral && funcName == "" {
			return true
		}
		ops = append(ops, operatorInfo{
			name:                   strings.Trim(nameLit.Value, "\""),
			path:                   path,
			line:                   fset.Position(nameLit.Pos()).Line,
			funcLit:                funcLit,
			funcName:               funcName,
			jitExpr:                jitExpr,
			jitVirtualExpr:         jitVirtualExpr,
			jitInlineCallbacksExpr: jitInlineCallbacksExpr,
			jitInlineCostExpr:      jitInlineCostExpr,
			jitInsertPos:           jitInsertPos,
			typeInsertPos:          typeInsertPos,
		})
		return true
	})
	return ops
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

// --- Storage method collection (pattern 2: ColumnStorage.GetValue → JITEmit) ---

type storageInfo struct {
	typeName    string         // e.g. "StorageInt"
	path        string         // source file path
	recvName    string         // receiver variable name (e.g. "s", "p")
	getValuePos token.Pos      // position of GetValue func keyword (for SSA lookup)
	jitEmitBody *ast.BlockStmt // body of JITEmit method (for patching)
}

// collectStorageMethods finds types in f that have both GetValue and JITEmit methods.
func collectStorageMethods(fset *token.FileSet, f *ast.File, path string) []storageInfo {
	// First pass: collect all methods by receiver type name
	type methodInfo struct {
		funcPos  token.Pos // position of func name (for SSA lookup)
		body     *ast.BlockStmt
		recvName string // receiver variable name
	}
	getValues := map[string]methodInfo{}
	jitEmits := map[string]methodInfo{}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		// Extract receiver type name (handle *T)
		recvType := fn.Recv.List[0].Type
		if star, ok := recvType.(*ast.StarExpr); ok {
			recvType = star.X
		}
		ident, ok := recvType.(*ast.Ident)
		if !ok {
			continue
		}
		typeName := ident.Name
		recvName := ""
		if len(fn.Recv.List[0].Names) > 0 {
			recvName = fn.Recv.List[0].Names[0].Name
		}

		switch fn.Name.Name {
		case "GetValue":
			getValues[typeName] = methodInfo{funcPos: fn.Name.Pos(), body: fn.Body, recvName: recvName}
		case "JITEmit":
			jitEmits[typeName] = methodInfo{funcPos: fn.Name.Pos(), body: fn.Body, recvName: recvName}
		}
	}

	// Second pass: pair them up
	var result []storageInfo
	for typeName, gv := range getValues {
		je, ok := jitEmits[typeName]
		if !ok {
			continue
		}
		result = append(result, storageInfo{
			typeName:    typeName,
			path:        path,
			recvName:    je.recvName,
			getValuePos: gv.funcPos,
			jitEmitBody: je.body,
		})
	}
	return result
}

// --- SSA dump ---

func dumpSSA(fn *ssa.Function) {
	fmt.Printf("\n  SSA for %s (%d blocks):\n", fn.Name(), len(fn.Blocks))
	for _, block := range fn.Blocks {
		fmt.Printf("    BB%d:", block.Index)
		if len(block.Preds) > 0 {
			preds := make([]string, len(block.Preds))
			for i, p := range block.Preds {
				preds[i] = fmt.Sprintf("BB%d", p.Index)
			}
			fmt.Printf(" <- %s", strings.Join(preds, ", "))
		}
		fmt.Println()
		for _, instr := range block.Instrs {
			fmt.Printf("      %-60s %T\n", instr, instr)
		}
		succs := block.Succs
		if len(succs) > 0 {
			ss := make([]string, len(succs))
			for i, s := range succs {
				ss[i] = fmt.Sprintf("BB%d", s.Index)
			}
			fmt.Printf("      -> %s\n", strings.Join(ss, ", "))
		}
		fmt.Println()
	}
}

// --- codegen ---

// genVal tracks how an SSA value is represented in the generated Go code.
// goVar is a Go variable name: either a JITValueDesc (isDesc=true) or a Reg.
//
// Scmer runtime layout contract (important for emitter generation):
//   - Scmer is split into two machine words: ptr + aux.
//   - ptr always carries pointer-typed data (or type-sentinel pointers for int/float).
//   - aux always carries non-pointer payload/tag bits (int payload, float bits, string/slice len, etc.).
//   - JITValueDesc{Loc: LocRegPair} means both halves are live and must be preserved.
//   - JITValueDesc{Loc: LocReg/LocImm} means scalar payload only; ptr half is not materialized yet.
//   - If Type is known (not JITTypeUnknown), tag information is compile-time known and must not consume
//     an extra runtime register.
type genVal struct {
	goVar            string
	isDesc           bool   // true = JITValueDesc (Scmer ptr+aux pair or scalar payload descriptor), false = Reg (raw scalar)
	argIdx           int    // >= 0: deferred arg reference from IndexAddr (constant index), not yet loaded
	argIdxVar        string // non-empty: deferred arg reference with variable index (goVar of index desc)
	argBase          int    // compile-time offset into the emitter's variadic args
	marker           string // "_newbool"/"_newint"/"_newfloat" for deferred constructors
	resultTargetVar  string // generated bool: scalar payload was emitted directly into result.Reg2
	deferredIndexSSA string // SSA name of index operand (for deferred IndexAddr on slices)
	deferredBaseSSA  string // SSA name of base operand for deferred local FieldAddr deref
	offsetExpr       string // Go expression for byte offset from thisptr (for _fieldaddr/_fieldconst markers)
	stackBase        string
	stackLen         int
	variadicOffset   int
	variadicLen      int
	variadicLenKnown bool
	sourceInput      int
	hasSourceInput   bool
	sliceInput       int
	hasSliceInput    bool
	lengthInput      int
	hasLengthInput   bool
	pinAcrossBlock   bool // mutable aggregate whose register identity is embedded in another block
	tuple            []genVal
	closureFn        *ssa.Function
	closureBindings  []closureBinding
	cellName         string
	cellScope        uint32
	fieldBaseType    types.Type
	fieldType        types.Type
	fieldName        string
	aggregateType    types.Type
}

type closureBinding struct {
	outerName string
	value     genVal
	scope     uint32
}

// ssaValueRewriter can replace SSA values while traversing instructions.
// Returning nil keeps the original node.
type ssaValueRewriter func(in ssa.Value) ssa.Value

type codeGen struct {
	w             strings.Builder
	wDecl         strings.Builder
	vals          map[string]genVal
	paramName     string
	nextDesc      int
	nextReg       int
	nextLabel     int
	fn            *ssa.Function
	bbLabels      map[uint64]string // scoped BB id → label var name
	bbPosVars     map[uint64]string // scoped BB id → int32 machine code position var
	bbDone        map[uint64]bool   // scoped BB id → already generated
	bbQueued      map[uint64]bool   // scoped BB id → queued for future generation
	bbQueue       []int             // queue of BB indices to generate
	bbScope       uint32            // current BB namespace id
	nextBBScope   uint32            // monotonically increasing fallback namespace id
	inlineCallSeq map[uint64]uint32 // caller scoped-BB id -> inline call ordinal
	phiRegs       map[string]string // SSA phi name → stack offset string (e.g. "0", "8", "16")
	phiPair       map[string]bool   // SSA phi name → true if value occupies 2 words (16 bytes)
	phiTriple     map[string]bool   // SSA phi name → true if value occupies 3 words (24 bytes)
	phiTypeTag    map[string]string // SSA phi name → static JIT tag constant (or JITTypeUnknown)
	bbPhiBase     map[int]int       // BB index → phi base stack offset (bytes)
	bbPhiCount    map[int]int       // BB index → number of phi slots
	phiStackSize  int               // total bytes reserved on stack for phi nodes (local to current function/inline)
	phiFrameFixup string            // Go var name for the current function's local phi-frame base
	curBlock      int               // current BB index being generated
	multiBlock    bool              // true if function has >1 block
	endLabel      string            // label for shared epilogue (multi-block)
	storageMode   bool              // true for ColumnStorage.GetValue pattern (vs Declare pattern)
	typeName      string            // struct type name for FieldAddr (e.g. "StorageInt")

	// Inline call state (non-empty when processing an inlined function)
	inlineReturnRegVar  string   // Go variable naming the result register (multi-block inline)
	inlineReturnReg2Var string   // Go variable naming the second Scmer result register
	inlineReturnsScm    bool     // true when current inline callee returns Scmer
	inlineReturnTuple   []genVal // stack-backed destinations for multi-result inline returns
	inlineEndLabel      string   // label after inlined blocks
	// Top-level multi-block storage returns are merged through a register-based
	// virtual phi (instead of writing result directly in each return block).
	returnPhiReg  string
	returnPhiReg2 string

	// Field deduplication: cache FieldAddr+UnOp deref results by field name
	fieldCache map[string]genVal

	// Reference counting for SSA values (remaining uses)
	refCounts map[string]int

	// Scalar values consumed by the constructor returned from the same block.
	// Arithmetic may produce these directly in result.Reg2.
	directResultPayloads map[string]string

	// SSA name aliases (e.g. Convert no-ops redirect to source)
	ssaAliases map[string]string

	// Top-level package path (the output package, not the inlined callee's package)
	topLevelPkgPath  string
	importedPkgAlias map[string]string
	opName           string
	// True for storage GetValue emitters that materialize idxInt/idxPinned vars.
	hasStorageIdx bool

	// Phi register protection: tracks registers protected during phi loads
	// at a block header. Cleared when the first non-Phi instruction is emitted.
	phiProtectedRegVars []string

	// When true, emitters are generated as recursive BBDescriptor.RenderPS(ps)
	// closures and branch lowering must recurse via bbs[i].RenderPS.
	bbClosureMode bool
	// forceLegacyCFG disables closure-recursive If/Jump lowering while keeping
	// descriptor predeclaration/assignment mode active.
	forceLegacyCFG bool
	// Descriptor predeclarations used by recursive BB closure mode, so
	// descriptors can flow across closure boundaries without scope breakage.
	closureDescDecl map[string]bool
	// Register predeclarations for closure-mode fixup handles (EmitSubRSP32Fixup).
	closureRegDecl map[string]bool
	// Optional callback-based SSA node rewrite hook.
	valueRewriter      ssaValueRewriter
	inlineInstructions int
}

func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	if src == nil {
		return nil
	}
	dst := make(map[K]V, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// clone creates an isolated code-generation transaction. Inlining is allowed
// to discover unsupported SSA by panicking; no partially emitted instructions,
// register names, phi state, or descriptor ownership may leak into the local
// Go-call fallback that follows.
func (g *codeGen) clone() *codeGen {
	clone := *g
	clone.w = strings.Builder{}
	clone.w.WriteString(g.w.String())
	clone.wDecl = strings.Builder{}
	clone.wDecl.WriteString(g.wDecl.String())
	clone.vals = cloneMap(g.vals)
	clone.bbLabels = cloneMap(g.bbLabels)
	clone.bbPosVars = cloneMap(g.bbPosVars)
	clone.bbDone = cloneMap(g.bbDone)
	clone.bbQueued = cloneMap(g.bbQueued)
	clone.bbQueue = append([]int(nil), g.bbQueue...)
	clone.inlineCallSeq = cloneMap(g.inlineCallSeq)
	clone.phiRegs = cloneMap(g.phiRegs)
	clone.phiPair = cloneMap(g.phiPair)
	clone.phiTriple = cloneMap(g.phiTriple)
	clone.phiTypeTag = cloneMap(g.phiTypeTag)
	clone.bbPhiBase = cloneMap(g.bbPhiBase)
	clone.bbPhiCount = cloneMap(g.bbPhiCount)
	clone.fieldCache = cloneMap(g.fieldCache)
	clone.refCounts = cloneMap(g.refCounts)
	clone.directResultPayloads = cloneMap(g.directResultPayloads)
	clone.ssaAliases = cloneMap(g.ssaAliases)
	clone.importedPkgAlias = cloneMap(g.importedPkgAlias)
	clone.phiProtectedRegVars = append([]string(nil), g.phiProtectedRegVars...)
	clone.closureDescDecl = cloneMap(g.closureDescDecl)
	clone.closureRegDecl = cloneMap(g.closureRegDecl)
	return &clone
}

type descSnapshot struct {
	desc string
	snap string
}

// overlayDescVar returns the currently active descriptor variable for an SSA value.
// Deferred emit patterns must not keep stale Go variable snapshots across BB
// boundaries; they should rebind to the latest SSA descriptor mapping.
func (g *codeGen) overlayDescVar(fallback, ssaName string) string {
	if ssaName == "" {
		return fallback
	}
	if cur, ok := g.vals[ssaName]; ok && cur.isDesc && cur.goVar != "" {
		return cur.goVar
	}
	return fallback
}

func (g *codeGen) rewriteSSAValue(v ssa.Value) ssa.Value {
	if v == nil || g.valueRewriter == nil {
		return v
	}
	if rv := g.valueRewriter(v); rv != nil {
		return rv
	}
	return v
}

func (g *codeGen) allocDesc() string {
	name := fmt.Sprintf("d%d", g.nextDesc)
	g.nextDesc++
	if g.bbClosureMode {
		if g.closureDescDecl == nil {
			g.closureDescDecl = map[string]bool{}
		}
		if !g.closureDescDecl[name] {
			g.closureDescDecl[name] = true
			fmt.Fprintf(&g.wDecl, "\t\t\tvar %s JITValueDesc\n", name)
			fmt.Fprintf(&g.wDecl, "\t\t\t_ = %s\n", name)
		}
	}
	return name
}

func (g *codeGen) allocReg() string {
	name := fmt.Sprintf("r%d", g.nextReg)
	g.nextReg++
	return name
}

func (g *codeGen) allocLabel() string {
	name := fmt.Sprintf("lbl%d", g.nextLabel)
	g.nextLabel++
	return name
}

func (g *codeGen) allocTemp(prefix string) string {
	name := fmt.Sprintf("%s%d", prefix, g.nextDesc)
	g.nextDesc++
	return name
}

func (g *codeGen) allClosureDescVars() []string {
	if len(g.closureDescDecl) == 0 && len(g.vals) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(g.closureDescDecl)+len(g.vals))
	stableCells := make(map[string]struct{})
	for _, gv := range g.vals {
		if gv.cellName != "" && gv.marker == "_slice" && gv.goVar != "" {
			stableCells[gv.goVar] = struct{}{}
		}
	}
	names := make([]string, 0, len(g.closureDescDecl)+len(g.vals))
	for name := range g.closureDescDecl {
		if _, stable := stableCells[name]; stable {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, gv := range g.vals {
		if !gv.isDesc {
			continue
		}
		name := gv.goVar
		if _, stable := stableCells[name]; stable {
			continue
		}
		if len(name) < 2 || name[0] != 'd' {
			continue
		}
		if _, err := parseDescNum(name); err != nil {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	sortDescNames(names)
	return names
}

func sortDescNames(names []string) {
	sort.Slice(names, func(i, j int) bool {
		ni, ei := parseDescNum(names[i])
		nj, ej := parseDescNum(names[j])
		if ei == nil && ej == nil {
			if ni == nj {
				return names[i] < names[j]
			}
			return ni < nj
		}
		if ei == nil {
			return true
		}
		if ej == nil {
			return false
		}
		return names[i] < names[j]
	})
}

func parseDescNum(name string) (int, error) {
	if len(name) < 2 || name[0] != 'd' {
		return 0, fmt.Errorf("not a descriptor: %s", name)
	}
	return strconv.Atoi(name[1:])
}

func (g *codeGen) normalizeDescVarList(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n == "" || !strings.HasPrefix(n, "d") {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sortDescNames(out)
	return out
}

func (g *codeGen) neededDescVarsForBlock(bbIdx int) []string {
	if bbIdx < 0 || bbIdx >= len(g.fn.Blocks) {
		return nil
	}
	block := g.fn.Blocks[bbIdx]
	var names []string
	for _, instr := range block.Instrs {
		for _, op := range instr.Operands(nil) {
			if *op == nil {
				continue
			}
			if _, isConst := (*op).(*ssa.Const); isConst {
				continue
			}
			name := (*op).Name()
			if alias, ok := g.ssaAliases[name]; ok {
				name = alias
			}
			gv, ok := g.vals[name]
			if !ok || !gv.isDesc || gv.goVar == "" {
				continue
			}
			names = append(names, gv.goVar)
		}
	}
	// Include successor-phi edge inputs as well: these values are consumed by
	// emitEdgePhiMoves when rendering this block's outgoing branches, but they
	// are often not direct operands of this block's own instructions.
	for succPos, succ := range block.Succs {
		edgeIdx, ok := g.phiEdgeIndexForSucc(succ.Index, succPos)
		if !ok {
			continue
		}
		for _, instr := range succ.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				break
			}
			if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
				continue
			}
			edge := phi.Edges[edgeIdx]
			if edge == nil {
				continue
			}
			if _, isConst := edge.(*ssa.Const); isConst {
				continue
			}
			name := edge.Name()
			if alias, ok := g.ssaAliases[name]; ok {
				name = alias
			}
			gv, ok := g.vals[name]
			if !ok || !gv.isDesc || gv.goVar == "" {
				continue
			}
			names = append(names, gv.goVar)
		}
	}
	return g.normalizeDescVarList(names)
}

func (g *codeGen) emitSaveClosureDescState(names []string) []descSnapshot {
	if !g.bbClosureMode {
		return nil
	}
	names = g.normalizeDescVarList(names)
	if len(names) == 0 {
		return nil
	}
	out := make([]descSnapshot, 0, len(names))
	for _, name := range names {
		snap := g.allocTemp("snap")
		g.emit("%s := %s", snap, name)
		out = append(out, descSnapshot{desc: name, snap: snap})
	}
	return out
}

func (g *codeGen) emitRestoreClosureDescState(snaps []descSnapshot) {
	for _, s := range snaps {
		g.emit("%s = %s", s.desc, s.snap)
	}
}

func (g *codeGen) emitProtectIncomingArgRegs() string {
	if g.storageMode {
		return ""
	}
	// A generated callee may use every register in the shared allocator. Give
	// incoming register values stable stack homes instead of reserving their
	// registers for the complete builtin body; nested emitters can then spill
	// and reuse the full bank without invalidating caller values.
	g.emit("for i := range args {")
	g.emit("\tctx.StabilizeDescForControlFlow(&args[i])")
	g.emit("}")
	return ""
}

func (g *codeGen) emitUnprotectIncomingArgRegs(pinned string) {
	// Incoming registers are released by the deferred emitter epilogue. Returns
	// are emitted directly from SSA blocks, so an appended cleanup statement
	// would be unreachable for single-block functions.
	_ = pinned
}

func (g *codeGen) scopedBBID(bbIdx int) uint64 {
	return (uint64(g.bbScope) << 32) | uint64(uint32(bbIdx))
}

// emitAllocRegExcept emits a ctx.AllocRegExcept(gv.Reg) when guard is true and
// gv is a register-located descriptor, otherwise emits ctx.AllocReg().
//
// This prevents the eviction-alias bug: without the guard, AllocReg() might
// evict gv.Reg and return it as the new register, making any subsequent
// EmitMovRegReg(dst, gv.Reg) a no-op self-copy (and letting the following
// ALU op destroy the original value).
//
// The generated one-liner is architecture-agnostic and hides the
// protect/unprotect implementation detail from the caller.
func (g *codeGen) emitAllocRegExcept(dstVar, indent string, guard bool, gv genVal) {
	if guard && gv.isDesc {
		g.emit("%s%s := ctx.AllocRegExcept(%s.Reg)", indent, dstVar, gv.goVar)
	} else {
		g.emit("%s%s := ctx.AllocReg()", indent, dstVar)
	}
}

// emitNormalizeUnsignedNarrow canonicalizes an integer descriptor to unsigned
// N-bit semantics (N < 64). Arithmetic executes in 64-bit registers, so this
// keeps uint8/uint16/uint32 wrap-around behavior correct.
func (g *codeGen) emitNormalizeUnsignedNarrow(descVar string, bits int) {
	if bits <= 0 || bits >= 64 {
		return
	}
	mask := (uint64(1) << uint(bits)) - 1
	shift := 64 - bits
	g.emit("if %s.Loc == LocImm {", descVar)
	g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: %s.Type, Imm: NewInt(int64(uint64(%s.Imm.Int()) & 0x%x))}", descVar, descVar, descVar, mask)
	g.emit("} else {")
	g.emit("\tctx.EmitShlRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("\tctx.EmitShrRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("}")
}

// emitNormalizeSignedNarrow canonicalizes an integer descriptor to signed
// N-bit semantics (N < 64) by sign-extending from bit N-1.
func (g *codeGen) emitNormalizeSignedNarrow(descVar string, bits int) {
	if bits <= 0 || bits >= 64 {
		return
	}
	shift := 64 - bits
	g.emit("if %s.Loc == LocImm {", descVar)
	switch bits {
	case 8:
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: %s.Type, Imm: NewInt(int64(int8(%s.Imm.Int())))}", descVar, descVar, descVar)
	case 16:
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: %s.Type, Imm: NewInt(int64(int16(%s.Imm.Int())))}", descVar, descVar, descVar)
	case 32:
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: %s.Type, Imm: NewInt(int64(int32(%s.Imm.Int())))}", descVar, descVar, descVar)
	default:
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: %s.Type, Imm: NewInt(%s.Imm.Int())}", descVar, descVar, descVar)
	}
	g.emit("} else {")
	g.emit("\tctx.EmitShlRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("\tctx.EmitSarRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("}")
}

func (g *codeGen) emit(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	if g.bbClosureMode && generatedDescDeclaration(line) {
		if i := strings.Index(line, " := "); i > 1 {
			name := line[:i]
			if g.closureDescDecl == nil {
				g.closureDescDecl = map[string]bool{}
			}
			if !g.closureDescDecl[name] {
				g.closureDescDecl[name] = true
				fmt.Fprintf(&g.wDecl, "\t\t\tvar %s JITValueDesc\n", name)
				fmt.Fprintf(&g.wDecl, "\t\t\t_ = %s\n", name)
			}
			line = name + " = " + line[i+4:]
		}
	} else if g.bbClosureMode && strings.Contains(line, " := ctx.AllocStack(") {
		if i := strings.Index(line, " := "); i > 0 {
			name := line[:i]
			if g.closureRegDecl == nil {
				g.closureRegDecl = map[string]bool{}
			}
			if !g.closureRegDecl[name] {
				g.closureRegDecl[name] = true
				fmt.Fprintf(&g.wDecl, "\t\t\tvar %s int32\n", name)
			}
			line = name + " = " + line[i+4:]
		}
	}
	fmt.Fprintf(&g.w, "\t\t\t%s\n", line)
}

func generatedDescDeclaration(line string) bool {
	i := strings.Index(line, " := ")
	if i < 2 || line[0] != 'd' {
		return false
	}
	for _, char := range line[1:i] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func goCallWordCount(t types.Type) int {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Kind() == types.String {
			return 2
		}
		return 1
	case *types.Pointer, *types.Signature, *types.Map, *types.Chan:
		return 1
	case *types.Interface:
		return 2
	case *types.Slice:
		return 3
	case *types.Struct:
		sz := types.SizesFor("gc", "amd64").Sizeof(t)
		if sz > 0 && sz <= 24 && sz%8 == 0 {
			return int(sz / 8)
		}
		return 0
	case *types.Array:
		sz := types.SizesFor("gc", "amd64").Sizeof(t)
		if sz > 0 && sz <= 24 {
			return int((sz + 7) / 8)
		}
		return 0
	default:
		return 0
	}
}

func goCallPointerMask(t types.Type) uint8 {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Kind() == types.String {
			return 1
		}
		return 0
	case *types.Pointer, *types.Signature, *types.Map, *types.Chan:
		return 1
	case *types.Interface:
		return 3
	case *types.Slice:
		return 1
	case *types.Struct:
		sizes := types.SizesFor("gc", "amd64")
		offsets := sizes.Offsetsof(fieldVarsOf(u))
		var mask uint8
		for i := 0; i < u.NumFields(); i++ {
			mask |= goCallPointerMask(u.Field(i).Type()) << (offsets[i] / 8)
		}
		return mask
	case *types.Array:
		elemWords := goCallWordCount(u.Elem())
		if elemWords < 1 {
			return 0
		}
		elemMask := goCallPointerMask(u.Elem())
		var mask uint8
		for i := int64(0); i < u.Len(); i++ {
			mask |= elemMask << uint(int(i)*elemWords)
		}
		return mask
	default:
		return 0
	}
}

func (g *codeGen) staticFuncExpr(callee *ssa.Function) (string, bool) {
	if callee == nil || callee.Signature == nil {
		return "", false
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return "", false
	}
	calleePkgPath := callee.Pkg.Pkg.Path()
	alias, imported := g.importedPkgAlias[calleePkgPath]
	if calleePkgPath != g.topLevelPkgPath && (!imported || !token.IsExported(callee.Name())) {
		return "", false
	}
	if recv := callee.Signature.Recv(); recv != nil {
		qualifier := func(pkg *types.Package) string {
			if pkg == nil || pkg.Path() == g.topLevelPkgPath {
				return ""
			}
			if importedAlias, ok := g.importedPkgAlias[pkg.Path()]; ok {
				return importedAlias
			}
			return pkg.Name()
		}
		return "(" + types.TypeString(recv.Type(), qualifier) + ")." + callee.Name(), true
	}
	if callee.Pkg.Pkg.Path() == g.topLevelPkgPath {
		return callee.Name(), true
	}
	return alias + "." + callee.Name(), true
}

func (g *codeGen) staticConstExpr(value *ssa.Const, target types.Type) (string, bool) {
	if value == nil || value.Value == nil {
		return "", false
	}
	base := ""
	switch value.Value.Kind() {
	case constant.String:
		base = strconv.Quote(constant.StringVal(value.Value))
	case constant.Int:
		base = value.Value.ExactString()
	case constant.Bool:
		base = value.Value.ExactString()
	default:
		return "", false
	}
	if _, named := target.(*types.Named); named {
		qualifier := func(pkg *types.Package) string {
			if pkg == nil || pkg.Path() == g.topLevelPkgPath {
				return ""
			}
			return pkg.Name()
		}
		return types.TypeString(target, qualifier) + "(" + base + ")", true
	}
	return base, true
}

func (g *codeGen) sourceTypeExpr(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg == nil || pkg.Path() == g.topLevelPkgPath {
			return ""
		}
		if alias, ok := g.importedPkgAlias[pkg.Path()]; ok {
			return alias
		}
		panic(fmt.Sprintf("source type %s requires missing import %q", t, pkg.Path()))
	})
}

func (g *codeGen) globalSourceExpr(global *ssa.Global) (string, bool) {
	if global == nil || global.Pkg == nil || global.Pkg.Pkg == nil {
		return "", false
	}
	expr := global.Name()
	if global.Pkg.Pkg.Path() != g.topLevelPkgPath {
		alias, ok := g.importedPkgAlias[global.Pkg.Pkg.Path()]
		if !ok || !token.IsExported(global.Name()) {
			return "", false
		}
		expr = alias + "." + expr
	}
	return expr, true
}

func allocIsClosureCell(v *ssa.Alloc) bool {
	if v.Referrers() == nil {
		return false
	}
	for _, ref := range *v.Referrers() {
		closure, ok := ref.(*ssa.MakeClosure)
		if !ok {
			continue
		}
		for _, binding := range closure.Bindings {
			if binding == v {
				return true
			}
		}
	}
	return false
}

func closureHasStaticCall(fn *ssa.Function, name string) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			if callee := call.Call.StaticCallee(); callee != nil && callee.Name() == name {
				return true
			}
		}
	}
	return false
}

func isForwardingMergeClosure(fn *ssa.Function) bool {
	if fn == nil || len(fn.FreeVars) != 1 || fn.Signature.Params().Len() != 2 || fn.Signature.Results().Len() != 1 || !isScmerType(fn.Signature.Results().At(0).Type()) {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if ok && call.Call.StaticCallee() == nil && len(call.Call.Args) == 1 {
				return true
			}
		}
	}
	return false
}

// emitGenericStaticCall lowers a static non-method Go call using signature-driven
// ABI word mapping. Returns true if it emitted code, false if caller should fall back.
func (g *codeGen) emitGenericStaticCall(name string, callee *ssa.Function, args []ssa.Value) bool {
	funcExpr, ok := g.staticFuncExpr(callee)
	if !ok {
		return false
	}
	sig := callee.Signature
	params := sig.Params()
	argOffset := 0
	if sig.Recv() != nil {
		argOffset = 1
	}
	if params.Len()+argOffset != len(args) {
		return false
	}
	results := sig.Results()
	retWords := 0
	indirectResult := false
	if g.storageMode && results.Len() > 1 {
		return false
	}
	if results.Len() == 1 && name == "" {
		return false
	}
	resultWords := make([]int, results.Len())
	resultPointerMasks := make([]uint8, results.Len())
	for i := 0; i < results.Len(); i++ {
		words := goCallWordCount(results.At(i).Type())
		if results.Len() == 1 && words == 0 {
			switch results.At(i).Type().Underlying().(type) {
			case *types.Struct, *types.Array:
				indirectResult = true
				words = 1
			}
		}
		if words < 1 || words > 3 || retWords+words > 16 {
			return false
		}
		resultWords[i] = words
		if indirectResult {
			resultPointerMasks[i] = 1
		} else {
			resultPointerMasks[i] = goCallPointerMask(results.At(i).Type())
		}
		retWords += words
	}
	resolved := make([]genVal, len(args))
	argVars := make([]string, 0, len(args))
	constantArgs := make([]bool, len(args))
	indirectArgs := make([]bool, len(args))
	for i, a := range args {
		_, constantArgs[i] = a.(*ssa.Const)
		_, globalArg := a.(*ssa.Global)
		if !constantArgs[i] && !globalArg {
			candidate := g.lookup(g.rewriteSSAValue(a))
			if candidate.goVar == "" {
				return false
			}
		}
		resolved[i] = g.resolveValue(a)
		paramType := func() types.Type {
			if i == 0 && argOffset == 1 {
				return sig.Recv().Type()
			}
			return params.At(i - argOffset).Type()
		}()
		wordCount := goCallWordCount(paramType)
		if _, isAggregate := paramType.Underlying().(*types.Struct); isAggregate && resolved[i].marker == "_aggregate_ptr" && wordCount == 0 {
			indirectArgs[i] = true
			argVars = append(argVars, resolved[i].goVar)
			continue
		}
		switch wordCount {
		case 0:
			// Zero-sized values have no Go internal-ABI words. Keep them in the
			// source-level signature, but do not invent a machine operand.
			if types.SizesFor("gc", "amd64").Sizeof(paramType) != 0 {
				return false
			}
			continue
		case 1:
			// If the value is currently an aggregate, this call shape is not representable.
			g.emit("if %s.Loc == LocRegPair || %s.Loc == LocStackPair || %s.Loc == LocRegTriple || %s.Loc == LocStackTriple {", resolved[i].goVar, resolved[i].goVar, resolved[i].goVar, resolved[i].goVar)
			g.emit("\tpanic(\"jit: generic call arg expects 1-word value\")")
			g.emit("}")
		case 2:
			// Scmer values may be folded to one-word scalars by the JIT type
			// system, but every native Go boundary still requires ptr+aux.
			if isScmerType(paramType) {
				prepare := "JITPrepareScmerGoArg"
				if g.storageMode {
					prepare = "scm." + prepare
				}
				g.emit("%s = %s(ctx, %s)", resolved[i].goVar, prepare, resolved[i].goVar)
				break
			}
			g.emit("ctx.EnsureDesc(&%s)", resolved[i].goVar)
			g.emit("if %s.Loc == LocImm {", resolved[i].goVar)
			g.emit("\ttmpPair := JITValueDesc{Loc: LocRegPair, Type: %s.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}", resolved[i].goVar)
			if basic, ok := paramType.Underlying().(*types.Basic); ok && basic.Kind() == types.String {
				g.emit("\tctx.TrackImm(%s.Imm)", resolved[i].goVar)
				g.emit("\tptrWord, _ := %s.Imm.RawWords()", resolved[i].goVar)
				g.emit("\tctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))")
				g.emit("\tctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(%s.Imm.String())))", resolved[i].goVar)
			} else {
				g.emit("\tif %s.Imm.GetTag() == tagBool {", resolved[i].goVar)
				g.emit("\t\tctx.EmitMakeBool(tmpPair, %s)", resolved[i].goVar)
				g.emit("\t} else if %s.Imm.GetTag() == tagInt {", resolved[i].goVar)
				g.emit("\t\tctx.EmitMakeInt(tmpPair, %s)", resolved[i].goVar)
				g.emit("\t} else if %s.Imm.GetTag() == tagFloat {", resolved[i].goVar)
				g.emit("\t\tctx.EmitMakeFloat(tmpPair, %s)", resolved[i].goVar)
				g.emit("\t} else if %s.Imm.GetTag() == tagNil {", resolved[i].goVar)
				g.emit("\t\tctx.EmitMakeNil(tmpPair)")
				g.emit("\t} else {")
				g.emit("\t\tptrWord, auxWord := %s.Imm.RawWords()", resolved[i].goVar)
				g.emit("\t\tctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))")
				g.emit("\t\tctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)")
				g.emit("\t}")
			}
			g.emit("\t%s = tmpPair", resolved[i].goVar)
			g.emit("} else if %s.Loc == LocReg {", resolved[i].goVar)
			g.emit("\ttmpPair := JITValueDesc{Loc: LocRegPair, Type: %s.Type, Reg: ctx.AllocRegExcept(%s.Reg), Reg2: ctx.AllocRegExcept(%s.Reg)}", resolved[i].goVar, resolved[i].goVar, resolved[i].goVar)
			g.emit("\tswitch %s.Type {", resolved[i].goVar)
			g.emit("\tcase tagBool:")
			g.emit("\t\tctx.EmitMakeBool(tmpPair, %s)", resolved[i].goVar)
			g.emit("\tcase tagInt:")
			g.emit("\t\tctx.EmitMakeInt(tmpPair, %s)", resolved[i].goVar)
			g.emit("\tcase tagFloat:")
			g.emit("\t\tctx.EmitMakeFloat(tmpPair, %s)", resolved[i].goVar)
			g.emit("\tdefault:")
			g.emit("\t\tpanic(\"jit: generic call arg scalar type unknown for 2-word value\")")
			g.emit("\t}")
			g.emit("\tctx.FreeDesc(&%s)", resolved[i].goVar)
			g.emit("\t%s = tmpPair", resolved[i].goVar)
			g.emit("}")
			g.emit("if %s.Loc != LocRegPair && %s.Loc != LocStackPair && %s.Loc != LocInputPair {", resolved[i].goVar, resolved[i].goVar, resolved[i].goVar)
			g.emit("\tpanic(\"jit: generic call arg expects 2-word value (%s arg%d)\")", funcExpr, i)
			g.emit("}")
		case 3:
			g.emit("ctx.EnsureDesc(&%s)", resolved[i].goVar)
			g.emit("if %s.Loc != LocRegTriple && %s.Loc != LocStackTriple {", resolved[i].goVar, resolved[i].goVar)
			g.emit("\tpanic(\"jit: generic call arg expects 3-word Go slice (%s arg%d)\")", funcExpr, i)
			g.emit("}")
		default:
			if _, isAggregate := paramType.Underlying().(*types.Struct); isAggregate && resolved[i].marker == "_aggregate_ptr" {
				indirectArgs[i] = true
				break
			}
			return false
		}
		argVars = append(argVars, resolved[i].goVar)
	}
	needsWrapper := indirectResult
	for _, indirect := range indirectArgs {
		needsWrapper = needsWrapper || indirect
	}
	if needsWrapper {
		if results.Len() > 1 {
			return false
		}
		paramsExpr := make([]string, len(args))
		callExpr := make([]string, len(args))
		for i, arg := range args {
			typeExpr := g.sourceTypeExpr(arg.Type())
			if indirectArgs[i] {
				paramsExpr[i] = fmt.Sprintf("arg%d *%s", i, typeExpr)
				callExpr[i] = fmt.Sprintf("*arg%d", i)
			} else {
				paramsExpr[i] = fmt.Sprintf("arg%d %s", i, typeExpr)
				callExpr[i] = fmt.Sprintf("arg%d", i)
			}
		}
		if sig.Variadic() && len(callExpr) > 0 {
			callExpr[len(callExpr)-1] += "..."
		}
		originalCall := fmt.Sprintf("%s(%s)", funcExpr, strings.Join(callExpr, ", "))
		if results.Len() == 0 {
			funcExpr = fmt.Sprintf("(func(%s) { %s })", strings.Join(paramsExpr, ", "), originalCall)
		} else if indirectResult {
			resultType := g.sourceTypeExpr(results.At(0).Type())
			funcExpr = fmt.Sprintf("(func(%s) *%s { value := %s; return &value })", strings.Join(paramsExpr, ", "), resultType, originalCall)
		} else {
			resultType := g.sourceTypeExpr(results.At(0).Type())
			funcExpr = fmt.Sprintf("(func(%s) %s { return %s })", strings.Join(paramsExpr, ", "), resultType, originalCall)
		}
	}
	// Materializing a later argument may spill an earlier one. Refresh every
	// descriptor only after all arguments have been prepared so flattenArgs
	// observes the final register/stack placement for the complete call.
	for _, arg := range resolved {
		if arg.isDesc && arg.goVar != "" {
			g.emit("ctx.SyncDesc(&%s)", arg.goVar)
		}
	}
	argList := strings.Join(argVars, ", ")
	if results.Len() == 0 {
		g.emit("ctx.EmitGoCallVoid(GoFuncAddr(%s), []JITValueDesc{%s})", funcExpr, argList)
		for i, isConstant := range constantArgs {
			if isConstant {
				g.emit("ctx.FreeDesc(&%s)", resolved[i].goVar)
			}
		}
		return true
	}
	if results.Len() > 1 {
		dv := g.allocTemp("callResults")
		wordLiterals := make([]string, len(resultWords))
		maskLiterals := make([]string, len(resultPointerMasks))
		for i := range resultWords {
			wordLiterals[i] = strconv.Itoa(resultWords[i])
			maskLiterals[i] = strconv.Itoa(int(resultPointerMasks[i]))
		}
		g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(%s), []JITValueDesc{%s}, []uint8{%s}, []uint8{%s})", dv, funcExpr, argList, strings.Join(wordLiterals, ", "), strings.Join(maskLiterals, ", "))
		for i, isConstant := range constantArgs {
			if isConstant {
				g.emit("ctx.FreeDesc(&%s)", resolved[i].goVar)
			}
		}
		tuple := make([]genVal, results.Len())
		for i := 0; i < results.Len(); i++ {
			resultDesc := g.allocDesc()
			g.emit("%s := %s[%d]", resultDesc, dv, i)
			g.emit("_ = %s", resultDesc)
			marker := ""
			if _, ok := results.At(i).Type().Underlying().(*types.Slice); ok {
				marker = "_slice"
			}
			if basic, ok := results.At(i).Type().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
				marker = "_gostring"
			}
			if _, ok := results.At(i).Type().Underlying().(*types.Array); ok {
				marker = "_goarrayvalue"
			}
			if _, ok := results.At(i).Type().Underlying().(*types.Signature); ok {
				marker = "_gofunc_variadic"
			}
			tuple[i] = genVal{goVar: resultDesc, isDesc: true, marker: marker}
		}
		g.vals[name] = genVal{tuple: tuple}
		return true
	}
	retType := results.At(0).Type()
	dv := g.allocDesc()
	g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(%s), []JITValueDesc{%s}, %d)", dv, funcExpr, argList, retWords)
	g.emit("%s.NoHeapPointer = %t", dv, resultPointerMasks[0] == 0)
	if basic, ok := retType.Underlying().(*types.Basic); ok && basic.Kind() == types.Bool {
		// Go's internal ABI only defines the low byte of a bool result. Clear
		// unspecified upper bits before generated CFG conditions consume it as
		// a full register value.
		g.emit("ctx.EmitAndRegImm32(%s.Reg, 1)", dv)
		g.emit("%s.Type = tagBool", dv)
	}
	// Bind and protect GoCall result registers. The nil-ownership from
	// EmitGoCallScalar prevents spilling, but BindReg makes it trackable.
	// We protect until freeDeadOperands releases this value.
	if retWords == 1 {
		g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
	} else if retWords == 2 {
		g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
		g.emit("ctx.BindReg(%s.Reg2, &%s)", dv, dv)
	} else if retWords == 3 {
		g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
		g.emit("ctx.BindReg(%s.Reg2, &%s)", dv, dv)
		g.emit("ctx.BindReg(%s.Reg3, &%s)", dv, dv)
	}
	for i, isConstant := range constantArgs {
		if isConstant {
			g.emit("ctx.FreeDesc(&%s)", resolved[i].goVar)
		}
	}
	marker := ""
	if bt, ok := retType.Underlying().(*types.Basic); ok && bt.Kind() == types.String {
		marker = "_gostring"
	}
	if _, ok := retType.Underlying().(*types.Slice); ok {
		marker = "_slice"
	}
	if _, ok := retType.Underlying().(*types.Array); ok {
		marker = "_goarrayvalue"
	}
	if _, ok := retType.Underlying().(*types.Signature); ok {
		marker = "_gofunc_variadic"
	}
	g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker}
	if indirectResult {
		g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_aggregate_ptr", pinAcrossBlock: true}
	}
	return true
}

func (g *codeGen) recordSliceResult(name string, producer ssa.Value, desc string) {
	if phiTarget, shape, direct := g.directPhiTarget(producer); direct && shape == phiTargetTriple {
		phiDesc := g.allocDesc()
		g.emit("%s := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: %s}", phiDesc, phiTarget)
		g.emit("ctx.EmitCopyDescWords(&%s, &%s, 3)", phiDesc, desc)
		g.emit("ctx.FreeDesc(&%s)", desc)
		g.emit("%s = %s", desc, phiDesc)
	}
	g.vals[name] = genVal{goVar: desc, isDesc: true, marker: "_slice", pinAcrossBlock: true}
}

// emitSerialCallableCall lowers a call through a callback prepared by
// PrepareSerialProc or OptimizeProcToSerialFunction. SSA represents these calls
// either as an indirect function call or as a statically resolved
// (*SerialProc).Call method. Keeping both representations on this path lets a
// known lambda recursively invoke its JIT emitter at the actual callback site.
func (g *codeGen) emitSerialCallableCall(name string, producer ssa.Value, callable, callArgs genVal) {
	if callArgs.marker == "_slice" || callArgs.marker == "_variadic_args" {
		argsDesc := callArgs
		if callArgs.marker == "_variadic_args" {
			end := ":"
			if callArgs.variadicLenKnown {
				end = fmt.Sprintf(":%d", callArgs.variadicOffset+callArgs.variadicLen)
			}
			materialized := g.allocDesc()
			g.emit("%s := jitMaterializeVirtualGoSlice(ctx, args[%d%s])", materialized, callArgs.variadicOffset, end)
			argsDesc = genVal{goVar: materialized, isDesc: true, marker: "_slice"}
		}
		callbackCallable := g.allocDesc()
		g.emit("%s := jitCopyScmerToPair(ctx, %s)", callbackCallable, callable.goVar)
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeCallbackSlice), []JITValueDesc{%s, %s}, 2)", dv, callbackCallable, argsDesc.goVar)
		g.vals[name] = genVal{goVar: dv, isDesc: true}
		return
	}
	if callArgs.stackBase == "" {
		panic("serial callback args are not a local Scmer array")
	}
	dv := g.allocDesc()
	argsVar := g.allocTemp("callbackArgs")
	g.emit("%s := make([]JITValueDesc, %d)", argsVar, callArgs.stackLen)
	for i := 0; i < callArgs.stackLen; i++ {
		g.emit("%s[%d] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s)+%d}", argsVar, i, callArgs.stackBase, i*16)
	}
	g.emit("var %s JITValueDesc", dv)
	callbackTargetOff := ""
	phiTarget, phiShape, directPhiTarget := g.directPhiTarget(producer)
	directPhiTarget = directPhiTarget && phiShape == phiTargetPair
	if directPhiTarget {
		callbackTargetOff = phiTarget
	} else {
		callbackResultOff := g.allocTemp("callbackResultOff")
		g.emit("%s := ctx.AllocStack(16)", callbackResultOff)
		callbackTargetOff = "int32(" + callbackResultOff + ")"
	}
	callbackTarget := fmt.Sprintf("JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: %s, ID: 0}", callbackTargetOff)
	g.emit("ctx.FreeDesc(&%s)", callArgs.goVar)
	g.emit("if %s.Loc == LocLambdaTemplate && %s.Lambda != nil {", callable.goVar, callable.goVar)
	stableArgs := g.allocTemp("stableCallbackArgs")
	g.emit("\t%s := ctx.StabilizeCallbackArgs(%s)", stableArgs, argsVar)
	preservedVar := g.allocTemp("outerRegs")
	g.emit("\tctx.ReclaimUntrackedRegs()")
	g.emit("\t%s := ctx.PreserveOuterRegs()", preservedVar)
	g.emit("\t%s = JITEmitProcInlineWithOuter(ctx, &%s.Lambda.Proc, %s.Lambda.Outer, %s, ctx.SliceBase, %s)", dv, callable.goVar, callable.goVar, stableArgs, callbackTarget)
	g.emit("\tctx.RestoreOuterRegs(%s)", preservedVar)
	g.emit("\tctx.ReclaimUntrackedRegs()")
	g.emit("} else {")
	knownResult := g.allocDesc()
	knownFlag := g.allocTemp("knownBuiltin")
	g.emit("\t%s, %s := jitEmitKnownDeclaration(ctx, %s, %s, %s)", knownResult, knownFlag, callable.goVar, argsVar, callbackTarget)
	g.emit("\tif %s {", knownFlag)
	g.emit("\t\t%s = %s", dv, knownResult)
	g.emit("\t} else {")
	callbackCallable := g.allocDesc()
	g.emit("\t\t%s := jitCopyScmerToPair(ctx, %s)", callbackCallable, callable.goVar)
	callbackHelper := ""
	switch callArgs.stackLen {
	case 1:
		callbackHelper = "jitInvokeCallback1"
	case 2:
		callbackHelper = "jitInvokeCallback2"
	case 3:
		callbackHelper = "jitInvokeCallback3"
	case 4:
		callbackHelper = "jitInvokeCallback4"
	default:
		panic(fmt.Sprintf("dynamic callback with unsupported arity: %d", callArgs.stackLen))
	}
	g.emit("\t\tcallbackCallArgs := make([]JITValueDesc, 0, %d)", callArgs.stackLen+1)
	g.emit("\t\tcallbackCallArgs = append(callbackCallArgs, %s)", callbackCallable)
	g.emit("\t\tcallbackCallArgs = append(callbackCallArgs, %s...)", argsVar)
	g.emit("\t\t%s = ctx.EmitGoCallScalarInto(GoFuncAddr(%s), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})", dv, callbackHelper)
	g.emit("\t\tctx.EmitStoreScmerToStack(%s, %s)", dv, callbackTargetOff)
	g.emit("\t\tctx.FreeDesc(&%s)", dv)
	g.emit("\t\t%s = %s", dv, callbackTarget)
	g.emit("\t}")
	g.emit("}")
	g.vals[name] = genVal{goVar: dv, isDesc: true}
}

func isSerialProcCall(callee *ssa.Function) bool {
	if callee == nil || callee.Name() != "Call" || callee.Signature.Recv() == nil {
		return false
	}
	receiver := callee.Signature.Recv().Type().String()
	return strings.HasSuffix(receiver, ".SerialProc") || strings.HasSuffix(receiver, "*SerialProc")
}

// emitInterfaceInvoke lowers an SSA invoke through a small typed adapter. The
// interface dispatch remains a native Go call boundary, while all surrounding
// argument/result flow stays visible to the JIT register allocator.
func (g *codeGen) emitInterfaceInvoke(name string, call *ssa.Call) bool {
	if call == nil || !call.Call.IsInvoke() || call.Call.Method == nil {
		return false
	}
	sig, ok := call.Call.Method.Type().(*types.Signature)
	if !ok || sig.Variadic() || sig.Results().Len() > 1 || sig.Params().Len() != len(call.Call.Args) {
		return false
	}
	receiver := g.resolveValue(call.Call.Value)
	if receiver.goVar == "" || !receiver.isDesc {
		return false
	}
	receiverType := g.sourceTypeExpr(call.Call.Value.Type())
	params := []string{"receiver " + receiverType}
	callArgs := make([]string, len(call.Call.Args))
	descVars := []string{receiver.goVar}
	for i, arg := range call.Call.Args {
		words := goCallWordCount(arg.Type())
		if words < 1 || words > 3 {
			return false
		}
		resolved := g.resolveValue(arg)
		if resolved.goVar == "" || !resolved.isDesc {
			return false
		}
		paramName := fmt.Sprintf("arg%d", i)
		params = append(params, paramName+" "+g.sourceTypeExpr(arg.Type()))
		callArgs[i] = paramName
		descVars = append(descVars, resolved.goVar)
	}
	invoke := fmt.Sprintf("receiver.%s(%s)", call.Call.Method.Name(), strings.Join(callArgs, ", "))
	if sig.Results().Len() == 0 {
		adapter := fmt.Sprintf("func(%s) { %s }", strings.Join(params, ", "), invoke)
		g.emit("ctx.EmitGoCallVoid(GoFuncAddr(%s), []JITValueDesc{%s})", adapter, strings.Join(descVars, ", "))
		return true
	}
	resultType := sig.Results().At(0).Type()
	words := goCallWordCount(resultType)
	if words < 1 || words > 3 || name == "" {
		return false
	}
	resultTypeExpr := g.sourceTypeExpr(resultType)
	adapter := fmt.Sprintf("func(%s) %s { return %s }", strings.Join(params, ", "), resultTypeExpr, invoke)
	dv := g.allocDesc()
	g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(%s), []JITValueDesc{%s}, %d)", dv, adapter, strings.Join(descVars, ", "), words)
	marker := ""
	switch resultType.Underlying().(type) {
	case *types.Interface:
		marker = "_goiface"
	case *types.Slice:
		marker = "_slice"
	case *types.Map:
		marker = "_gomap"
	case *types.Signature:
		marker = "_gofunc_variadic"
	}
	g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: words > 1}
	return true
}

// bothImmCond returns the Go condition for "both x and y are LocImm".
// When x == y (self-comparison, e.g. NaN check), emits only one check to avoid vet warning.
func bothImmCond(x, y string) string {
	if x == y {
		return x + ".Loc == LocImm"
	}
	return x + ".Loc == LocImm && " + y + ".Loc == LocImm"
}

func fitsInt32(v int64) bool {
	return v >= -2147483648 && v <= 2147483647
}

func (g *codeGen) emitMulConstOnReg(regExpr string, k int64, indent string) {
	switch k {
	case 0:
		g.emit("%sctx.EmitMovRegImm64(%s, 0)", indent, regExpr)
	case 1:
		// no-op
	case 2:
		g.emit("%sctx.EmitAddInt64(%s, %s)", indent, regExpr, regExpr)
	default:
		if fitsInt32(k) {
			g.emit("%sctx.EmitImulRegImm32(%s, int32(%d))", indent, regExpr, k)
		} else {
			g.emit("%sctx.EmitMovRegImm64(RegR11, uint64(%d))", indent, k)
			g.emit("%sctx.EmitImulInt64(%s, RegR11)", indent, regExpr)
		}
	}
}

// isFieldCachedDesc reports whether goVar is one of the cached field descriptors.
// Cached field values are semantically read-only sources and must not be
// destructively modified in-place by ALU emission.
func (g *codeGen) isFieldCachedDesc(goVar string) bool {
	for _, cached := range g.fieldCache {
		if cached.goVar == goVar && cached.isDesc {
			return true
		}
	}
	return false
}

// ensureBBLabel returns the label var name for a BB, reserving it if needed.
func (g *codeGen) ensureBBLabel(bbIdx int) string {
	bbID := g.scopedBBID(bbIdx)
	if lbl, ok := g.bbLabels[bbID]; ok {
		return lbl
	}
	lbl := g.allocLabel()
	g.bbLabels[bbID] = lbl
	g.emit("%s := ctx.ReserveLabel()", lbl)
	g.emit("_ = %s", lbl)
	return lbl
}

func (g *codeGen) ensureBBPosVar(bbIdx int) string {
	bbID := g.scopedBBID(bbIdx)
	if v, ok := g.bbPosVars[bbID]; ok {
		return v
	}
	v := fmt.Sprintf("bbpos_%d_%d", g.bbScope, bbIdx)
	g.bbPosVars[bbID] = v
	g.emit("%s := int32(-1)", v)
	g.emit("_ = %s", v)
	return v
}

// isGeneralBB reports BBs that should always get a label because they may be
// branch targets outside pure linear one-pass fallthrough.
func (g *codeGen) isGeneralBB(bbIdx int) bool {
	if bbIdx < 0 || bbIdx >= len(g.fn.Blocks) {
		return false
	}
	bb := g.fn.Blocks[bbIdx]
	if len(bb.Preds) > 1 {
		return true
	}
	for _, p := range bb.Preds {
		if p.Index >= bbIdx {
			return true
		}
	}
	return false
}

// enqueueBB adds a BB to the processing queue if not already done/queued.
func (g *codeGen) enqueueBB(bbIdx int) {
	bbID := g.scopedBBID(bbIdx)
	if g.bbDone[bbID] || g.bbQueued[bbID] {
		return
	}
	g.bbQueue = append(g.bbQueue, bbIdx)
	g.bbQueued[bbID] = true
}

// enqueueBBFront adds a BB to the front of the processing queue if not already done/queued.
func (g *codeGen) enqueueBBFront(bbIdx int) {
	bbID := g.scopedBBID(bbIdx)
	if g.bbDone[bbID] {
		return
	}
	if g.bbQueued[bbID] {
		// Keep fallthrough semantics: if we intentionally avoid emitting a jump,
		// the target block must be emitted next. Move existing queued target to front.
		for i, q := range g.bbQueue {
			if q == bbIdx {
				if i == 0 {
					return
				}
				copy(g.bbQueue[1:i+1], g.bbQueue[0:i])
				g.bbQueue[0] = bbIdx
				return
			}
		}
		// Safety: if queued map is stale, fall through and requeue at front.
	}
	g.bbQueue = append([]int{bbIdx}, g.bbQueue...)
	g.bbQueued[bbID] = true
}

func (g *codeGen) blockPhis(bbIdx int) []*ssa.Phi {
	if bbIdx < 0 || bbIdx >= len(g.fn.Blocks) {
		return nil
	}
	var out []*ssa.Phi
	for _, instr := range g.fn.Blocks[bbIdx].Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		out = append(out, phi)
	}
	return out
}

func (g *codeGen) phiSlotOffExpr(bbIdx int, phiIdx int) string {
	phis := g.blockPhis(bbIdx)
	if phiIdx >= 0 && phiIdx < len(phis) {
		if off, ok := g.phiRegs[phis[phiIdx].Name()]; ok {
			if g.forceLegacyCFG && g.phiFrameFixup != "" {
				return fmt.Sprintf("int32(%s)+int32(%s)", g.phiFrameFixup, off)
			}
			offset, err := strconv.Atoi(off)
			if err == nil {
				return fmt.Sprintf("int32(bbs[%d].PhiBase)+int32(%d)", bbIdx, offset-g.bbPhiBase[bbIdx])
			}
		}
	}
	return fmt.Sprintf("int32(bbs[%d].PhiBase)+int32(%d)", bbIdx, phiIdx*phiSlotBytes)
}

// directPhiTarget returns the stack slot and shape for a producer feeding one
// phi node on its block's sole outgoing edge. A producer before a branch must
// not write early: another edge may retain the phi's previous value.
func (g *codeGen) directPhiTarget(value ssa.Value) (string, JITTargetShape, bool) {
	refs := value.Referrers()
	if refs == nil {
		return "", phiTargetScalar, false
	}
	var phi *ssa.Phi
	for _, ref := range *refs {
		candidate, ok := ref.(*ssa.Phi)
		if !ok {
			continue
		}
		if phi != nil && phi != candidate {
			return "", phiTargetScalar, false
		}
		phi = candidate
	}
	if phi == nil {
		return "", phiTargetScalar, false
	}
	producer, ok := value.(ssa.Instruction)
	if !ok {
		return "", phiTargetScalar, false
	}
	producerBlock := producer.Block()
	if producerBlock == nil || len(producerBlock.Succs) != 1 || producerBlock.Succs[0] != phi.Block() {
		return "", phiTargetScalar, false
	}
	directEdge := false
	for edgeIndex, edge := range phi.Edges {
		if edge != value {
			continue
		}
		if edgeIndex >= len(phi.Block().Preds) || phi.Block().Preds[edgeIndex] != producerBlock {
			return "", phiTargetScalar, false
		}
		directEdge = true
	}
	if !directEdge {
		return "", phiTargetScalar, false
	}
	shape := phiTargetScalar
	if isPhiTripleType(phi.Type()) {
		shape = phiTargetTriple
	} else if isPhiPairType(phi.Type()) {
		shape = phiTargetPair
	}
	for phiIdx, candidate := range g.blockPhis(phi.Block().Index) {
		if candidate == phi {
			return g.phiSlotOffExpr(phi.Block().Index, phiIdx), shape, true
		}
	}
	return "", phiTargetScalar, false
}

type JITTargetShape uint8

const (
	phiTargetScalar JITTargetShape = iota
	phiTargetPair
	phiTargetTriple
)

func (g *codeGen) phiValueAlreadyStored(value ssa.Value, targetBBIdx, phiIdx int) bool {
	phis := g.blockPhis(targetBBIdx)
	if phiIdx < 0 || phiIdx >= len(phis) {
		return false
	}
	target := phis[phiIdx]
	if value == target {
		return true
	}
	if !jitgenProducerSupportsPhiTarget(value) {
		return false
	}
	off, _, ok := g.directPhiTarget(value)
	return ok && off == g.phiSlotOffExpr(targetBBIdx, phiIdx)
}

func jitgenProducerSupportsPhiTarget(value ssa.Value) bool {
	switch v := value.(type) {
	case *ssa.MakeSlice:
		return true
	case *ssa.Call:
		if builtin, ok := v.Call.Value.(*ssa.Builtin); ok {
			return builtin.Name() == "append"
		}
		callee := v.Call.StaticCallee()
		if callee == nil {
			return true
		}
		switch callee.Name() {
		case "NewBool", "NewInt", "NewFloat", "NewNil":
			return true
		default:
			return false
		}
	case *ssa.BinOp:
		return true
	default:
		return false
	}
}

func (g *codeGen) emitDirectPhiStore(value ssa.Value) {
	if _, ok := value.(*ssa.BinOp); !ok {
		return
	}
	target, shape, ok := g.directPhiTarget(value)
	if !ok || shape != phiTargetScalar {
		return
	}
	gv, ok := g.vals[value.Name()]
	if !ok || !gv.isDesc || gv.goVar == "" {
		return
	}
	g.emit("ctx.EnsureDesc(&%s)", gv.goVar)
	g.emit("ctx.EmitStoreToStack(%s, %s)", gv.goVar, target)
}

func (g *codeGen) emitBBPhiLayout() {
	for bbIdx := range g.fn.Blocks {
		base, ok := g.bbPhiBase[bbIdx]
		if !ok {
			continue
		}
		count := g.bbPhiCount[bbIdx]
		if g.phiFrameFixup != "" {
			g.emit("bbs[%d].PhiBase = int32(%s) + int32(%d)", bbIdx, g.phiFrameFixup, base)
		} else {
			g.emit("bbs[%d].PhiBase = int32(%d)", bbIdx, base)
		}
		g.emit("bbs[%d].PhiCount = uint16(%d)", bbIdx, count)
	}
}

func (g *codeGen) emitConstDescForSSAConst(c *ssa.Const) genVal {
	dv := g.allocDesc()
	if c.Value == nil {
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}", dv)
		return genVal{goVar: dv, isDesc: true}
	}
	switch c.Value.Kind() {
	case constant.Bool:
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%t)}", dv, constant.BoolVal(c.Value))
	case constant.Int:
		ival, _ := constant.Int64Val(c.Value)
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%d)}", dv, ival)
	case constant.Float:
		fval, _ := constant.Float64Val(c.Value)
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%v)}", dv, fval)
	case constant.String:
		sval := constant.StringVal(c.Value)
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(%q)}", dv, sval)
	default:
		panic(fmt.Sprintf("unsupported phi const kind: %s", c))
	}
	return genVal{goVar: dv, isDesc: true}
}

func (g *codeGen) emitBuildPhiStateForEdge(psVar string, targetBBIdx int, succPos int, generalExpr string) {
	g.emit("%s := PhiState{General: %s}", psVar, generalExpr)
	if overlayVars := g.allClosureDescVars(); len(overlayVars) > 0 {
		maxIdx := -1
		for _, ov := range overlayVars {
			if idx, err := parseDescNum(ov); err == nil && idx > maxIdx {
				maxIdx = idx
			}
		}
		if maxIdx >= 0 {
			g.emit("%s.OverlayValues = make([]JITValueDesc, %d)", psVar, maxIdx+1)
		}
		for _, ov := range overlayVars {
			idx, err := parseDescNum(ov)
			if err != nil {
				continue
			}
			g.emit("%s.OverlayValues[%d] = %s", psVar, idx, ov)
		}
	}
	phis := g.blockPhis(targetBBIdx)
	if len(phis) == 0 {
		return
	}
	g.emit("%s.PhiValues = make([]JITValueDesc, %d)", psVar, len(phis))
	edgeIdx, ok := g.phiEdgeIndexForSucc(targetBBIdx, succPos)
	if !ok {
		g.emit("%s.General = true", psVar)
		return
	}
	for phiIdx, phi := range phis {
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			continue
		}
		edge := phi.Edges[edgeIdx]
		if g.phiValueAlreadyStored(edge, targetBBIdx, phiIdx) {
			continue
		}
		if c, ok := edge.(*ssa.Const); ok {
			cv := g.emitConstDescForSSAConst(c)
			g.emit("%s.PhiValues[%d] = %s", psVar, phiIdx, cv.goVar)
			continue
		}
		name := edge.Name()
		if name == "" {
			continue
		}
		gv, ok := g.vals[name]
		if !ok || !gv.isDesc {
			g.emit("%s.General = true", psVar)
			continue
		}
		tmp := g.allocDesc()
		g.emit("%s := %s", tmp, gv.goVar)
		g.emit("%s.PhiValues[%d] = %s", psVar, phiIdx, tmp)
	}
}

// phiEdgeIndexForSucc resolves the phi edge index in targetBB for the
// outgoing edge at succPos of the current block. This handles duplicated
// successor blocks (then/else targeting the same BB).
func (g *codeGen) phiEdgeIndexForSucc(targetBBIdx int, succPos int) (int, bool) {
	if g.curBlock < 0 || g.curBlock >= len(g.fn.Blocks) {
		return 0, false
	}
	cur := g.fn.Blocks[g.curBlock]
	if succPos < 0 || succPos >= len(cur.Succs) {
		return 0, false
	}
	if cur.Succs[succPos].Index != targetBBIdx {
		return 0, false
	}
	dupOrd := 0
	for i := 0; i <= succPos; i++ {
		if cur.Succs[i].Index == targetBBIdx {
			dupOrd++
		}
	}
	target := g.fn.Blocks[targetBBIdx]
	seen := 0
	for i, pred := range target.Preds {
		if pred.Index == g.curBlock {
			seen++
			if seen == dupOrd {
				return i, true
			}
		}
	}
	return 0, false
}

func isScmerType(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Name() == "Scmer"
}

func isByteType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Uint8
}

func isByteSliceType(t types.Type) bool {
	slice, ok := t.Underlying().(*types.Slice)
	return ok && isByteType(slice.Elem())
}

func phiStartsWithBoundedEmptySlice(phi *ssa.Phi) bool {
	for _, edge := range phi.Edges {
		makeSlice, ok := edge.(*ssa.MakeSlice)
		if !ok || makeSlice.Cap == nil {
			continue
		}
		length, ok := makeSlice.Len.(*ssa.Const)
		if !ok || length.Value == nil || length.Value.Kind() != constant.Int {
			continue
		}
		lengthValue, exact := constant.Int64Val(length.Value)
		if exact && lengthValue == 0 && makeSlice.Cap != makeSlice.Len {
			return true
		}
	}
	return false
}

func jitTagForSSAType(t types.Type) string {
	if t == nil {
		return "JITTypeUnknown"
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsBoolean != 0:
			return "tagBool"
		case u.Info()&types.IsInteger != 0:
			return "tagInt"
		case u.Info()&types.IsFloat != 0:
			return "tagFloat"
		case u.Kind() == types.String:
			return "tagString"
		}
	}
	return "JITTypeUnknown"
}

// phiEdgeSpecializationScore estimates how specific an incoming edge is for
// target BB phis. Higher scores are preferred for immediate fallthrough.
func (g *codeGen) phiEdgeSpecializationScore(targetBBIdx int, succPos int) int {
	targetBlock := g.fn.Blocks[targetBBIdx]
	edgeIdx, ok := g.phiEdgeIndexForSucc(targetBBIdx, succPos)
	if !ok {
		return 0
	}
	score := 0
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			continue
		}
		edge := phi.Edges[edgeIdx]
		if _, ok := edge.(*ssa.Const); ok {
			score += 4
		}
		if edge.Type() != nil {
			if isScmerType(edge.Type()) {
				score++
			} else {
				score += 2
			}
		}
	}
	return score
}

func (g *codeGen) preferredIfFallthrough(thenBB, elseBB int) int {
	thenScore := g.phiEdgeSpecializationScore(thenBB, 0)
	elseScore := g.phiEdgeSpecializationScore(elseBB, 1)
	if thenScore > elseScore {
		return thenBB
	}
	if elseScore > thenScore {
		return elseBB
	}
	return elseBB
}

func (g *codeGen) emitProtectDescVars(descVars []string) {
	for _, dv := range descVars {
		g.emit("ctx.SyncDesc(&%s)", dv)
		g.emit("if %s.Loc == LocReg {", dv)
		g.emit("\tctx.ProtectReg(%s.Reg)", dv)
		g.emit("} else if %s.Loc == LocRegPair {", dv)
		g.emit("\tctx.ProtectReg(%s.Reg)", dv)
		g.emit("\tctx.ProtectReg(%s.Reg2)", dv)
		g.emit("}")
	}
}

func (g *codeGen) emitUnprotectDescVars(descVars []string) {
	for _, dv := range descVars {
		g.emit("if %s.Loc == LocReg {", dv)
		g.emit("\tctx.UnprotectReg(%s.Reg)", dv)
		g.emit("} else if %s.Loc == LocRegPair {", dv)
		g.emit("\tctx.UnprotectReg(%s.Reg)", dv)
		g.emit("\tctx.UnprotectReg(%s.Reg2)", dv)
		g.emit("}")
	}
}

func (g *codeGen) externalDescVars(block *ssa.BasicBlock) []string {
	seen := make(map[string]struct{})
	vars := make([]string, 0)
	for _, instr := range block.Instrs {
		for _, operand := range instr.Operands(nil) {
			if operand == nil || *operand == nil {
				continue
			}
			value := *operand
			if definition, ok := value.(ssa.Instruction); ok && definition.Block() == block {
				continue
			}
			generated, ok := g.vals[value.Name()]
			if !ok || !generated.isDesc || generated.goVar == "" || !generated.pinAcrossBlock {
				continue
			}
			if _, exists := seen[generated.goVar]; exists {
				continue
			}
			seen[generated.goVar] = struct{}{}
			vars = append(vars, generated.goVar)
		}
	}
	sort.Strings(vars)
	return vars
}

func (g *codeGen) emitPinDescVars(descVars []string) string {
	for _, variable := range descVars {
		g.emit("ctx.StabilizeDescForControlFlow(&%s)", variable)
	}
	return ""
}

func (g *codeGen) emitIfClosure(v *ssa.If) {
	thenBB := v.Block().Succs[0].Index
	elseBB := v.Block().Succs[1].Index
	if constantCond, ok := v.Cond.(*ssa.Const); ok && constantCond.Value != nil && constantCond.Value.Kind() == constant.Bool {
		targetBB := elseBB
		succPos := 1
		if constant.BoolVal(constantCond.Value) {
			targetBB = thenBB
			succPos = 0
		}
		g.emitEdgePhiMoves(targetBB, succPos)
		ps := g.allocTemp("ps")
		g.emitBuildPhiStateForEdge(ps, targetBB, succPos, "ps.General")
		g.emit("return bbs[%d].RenderPS(%s)", targetBB, ps)
		return
	}
	cond := g.vals[v.Cond.Name()]
	if !cond.isDesc {
		panic(fmt.Sprintf("If: %s unimplemented for %s.Loc (descriptor missing: isDesc=false, goVar=%s, marker=%q; expected LocImm|LocReg)",
			v, v.Cond.Name(), cond.goVar, cond.marker))
	}

	condVar := g.allocDesc()
	g.emit("%s := %s", condVar, cond.goVar)
	g.emit("ctx.EnsureDesc(&%s)", condVar)
	g.emit("if %s.Loc != LocImm && %s.Loc != LocReg {", condVar, condVar)
	g.emit("\tpanic(\"jit: If condition is neither LocImm nor LocReg\")")
	g.emit("}")

	// Constant-pruned branch: recurse into exactly one successor.
	g.emit("if %s.Loc == LocImm {", condVar)
	g.emit("\tif %s.Imm.Bool() {", condVar)
	g.emit("\t\tif ps.General {")
	g.emitEdgePhiMoves(thenBB, 0)
	g.emit("\t\t}")
	thenPS := g.allocTemp("ps")
	g.emitBuildPhiStateForEdge(thenPS, thenBB, 0, "ps.General")
	g.emit("\t\treturn bbs[%d].RenderPS(%s)", thenBB, thenPS)
	g.emit("\t}")
	g.emit("\tif ps.General {")
	g.emitEdgePhiMoves(elseBB, 1)
	g.emit("\t}")
	elsePS := g.allocTemp("ps")
	g.emitBuildPhiStateForEdge(elsePS, elseBB, 1, "ps.General")
	g.emit("\treturn bbs[%d].RenderPS(%s)", elseBB, elsePS)
	g.emit("}")

	// Dynamic condition in a specialized BB can otherwise poison successor
	// general renderers with edge-local specialized overlays. Canonicalize by
	// switching this BB to general first; successor rendering then sees stable
	// locations from the generalized predecessor state.
	g.emit("if !ps.General {")
	g.emitSpecializedPhiStackWrites(g.curBlock, "ps", "\t")
	g.emit("\tps.General = true")
	g.emit("\treturn bbs[%d].RenderPS(ps)", g.curBlock)
	g.emit("}")

	// Dynamic branch: emit edge helpers with runtime condition and render both
	// successors in general mode.
	thenLbl := g.ensureBBLabel(thenBB)
	elseLbl := g.ensureBBLabel(elseBB)
	thenEdgeLbl := g.allocLabel()
	elseEdgeLbl := g.allocLabel()
	g.emit("%s := ctx.ReserveLabel()", thenEdgeLbl)
	g.emit("%s := ctx.ReserveLabel()", elseEdgeLbl)
	g.emit("ctx.EmitCmpRegImm32(%s.Reg, 0)", condVar)
	g.emit("ctx.EmitJump(CondNotEqual, %s)", thenEdgeLbl)
	g.emit("ctx.EmitJmp(%s)", elseEdgeLbl)
	g.emit("ctx.MarkLabel(%s)", thenEdgeLbl)
	g.emitEdgePhiMoves(thenBB, 0)
	g.emit("ctx.EmitJmp(%s)", thenLbl)
	g.emit("ctx.MarkLabel(%s)", elseEdgeLbl)
	g.emitEdgePhiMoves(elseBB, 1)
	g.emit("ctx.EmitJmp(%s)", elseLbl)

	thenPSGeneral := g.allocTemp("ps")
	elsePSGeneral := g.allocTemp("ps")
	// Dynamic branches still need edge state for successor live-ins.
	// Render successor labels in general mode, but include edge overlays/phis.
	g.emitBuildPhiStateForEdge(thenPSGeneral, thenBB, 0, "true")
	g.emitBuildPhiStateForEdge(elsePSGeneral, elseBB, 1, "true")

	if g.preferredIfFallthrough(thenBB, elseBB) == thenBB {
		snaps := g.emitSaveClosureDescState(g.allClosureDescVars())
		allocSnap := g.allocTemp("alloc")
		g.emit("%s := ctx.SnapshotAllocState()", allocSnap)
		g.emit("if !bbs[%d].Rendered {", thenBB)
		g.emit("\tbbs[%d].RenderPS(%s)", thenBB, thenPSGeneral)
		g.emit("}")
		g.emit("ctx.RestoreAllocState(%s)", allocSnap)
		g.emitRestoreClosureDescState(snaps)
		g.emit("if !bbs[%d].Rendered {", elseBB)
		g.emit("\treturn bbs[%d].RenderPS(%s)", elseBB, elsePSGeneral)
		g.emit("}")
	} else {
		snaps := g.emitSaveClosureDescState(g.allClosureDescVars())
		allocSnap := g.allocTemp("alloc")
		g.emit("%s := ctx.SnapshotAllocState()", allocSnap)
		g.emit("if !bbs[%d].Rendered {", elseBB)
		g.emit("\tbbs[%d].RenderPS(%s)", elseBB, elsePSGeneral)
		g.emit("}")
		g.emit("ctx.RestoreAllocState(%s)", allocSnap)
		g.emitRestoreClosureDescState(snaps)
		g.emit("if !bbs[%d].Rendered {", thenBB)
		g.emit("\treturn bbs[%d].RenderPS(%s)", thenBB, thenPSGeneral)
		g.emit("}")
	}
	g.emit("return result")
}

func (g *codeGen) emitJumpClosure(v *ssa.Jump) {
	targetBB := v.Block().Succs[0].Index
	g.emit("if ps.General {")
	g.emitEdgePhiMoves(targetBB, 0)
	g.emit("}")
	nextPS := g.allocTemp("ps")
	g.emitBuildPhiStateForEdge(nextPS, targetBB, 0, "ps.General")
	g.emit("if %s.General && bbs[%d].Rendered {", nextPS, targetBB)
	if lbl, ok := g.bbLabels[g.scopedBBID(targetBB)]; ok {
		g.emit("\tctx.EmitJmp(%s)", lbl)
	} else {
		panic(fmt.Sprintf("jitgen: recursive mode missing label for BB%d", targetBB))
	}
	g.emit("\treturn result")
	g.emit("}")
	g.emit("return bbs[%d].RenderPS(%s)", targetBB, nextPS)
}

// emitEdgePhiMoves emits machine-code-level MOVs for phi edges to targetBB from
// the successor edge succPos of the current block.
func (g *codeGen) emitEdgePhiMoves(targetBBIdx int, succPos int) {
	moves := g.collectEdgePhiMoves(targetBBIdx, succPos)
	if len(moves) == 0 {
		return
	}
	if g.phiMovesRequireSingleChunk(moves) {
		deps := g.phiMoveDepsForRange(moves, 0, len(moves))
		g.emitProtectDescVars(deps)
		for i := 0; i < len(moves); i++ {
			m := moves[i]
			g.emitPhiMov(m.phiOff, m.edge, m.phiType)
		}
		g.emitUnprotectDescVars(deps)
		return
	}
	for start := 0; start < len(moves); start += phiStoreChunkSize {
		end := start + phiStoreChunkSize
		if end > len(moves) {
			end = len(moves)
		}
		deps := g.phiMoveDepsForRange(moves, start, end)
		g.emitProtectDescVars(deps)
		for i := start; i < end; i++ {
			m := moves[i]
			g.emitPhiMov(m.phiOff, m.edge, m.phiType)
		}
		g.emitUnprotectDescVars(deps)
	}
}

type phiEdgeMove struct {
	phiOff  string
	edge    ssa.Value
	phiType types.Type
}

func (g *codeGen) collectEdgePhiMoves(targetBBIdx int, succPos int) []phiEdgeMove {
	targetBlock := g.fn.Blocks[targetBBIdx]
	edgeIdx, ok := g.phiEdgeIndexForSucc(targetBBIdx, succPos)
	if !ok {
		return nil
	}
	out := make([]phiEdgeMove, 0, len(targetBlock.Instrs))
	phiIdx := 0
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			panic(fmt.Sprintf("phi edge index out of range for %s: edge=%d len=%d", phi.Name(), edgeIdx, len(phi.Edges)))
		}
		edge := phi.Edges[edgeIdx]
		if g.phiValueAlreadyStored(edge, targetBBIdx, phiIdx) {
			phiIdx++
			continue
		}
		out = append(out, phiEdgeMove{
			phiOff:  g.phiSlotOffExpr(targetBBIdx, phiIdx),
			edge:    edge,
			phiType: phi.Type(),
		})
		phiIdx++
	}
	return out
}

func (g *codeGen) phiMoveDepsForRange(moves []phiEdgeMove, start, end int) []string {
	var out []string
	for i := start; i < end; i++ {
		edge := moves[i].edge
		if edge == nil {
			continue
		}
		if _, isConst := edge.(*ssa.Const); isConst {
			continue
		}
		name := edge.Name()
		if alias, ok := g.ssaAliases[name]; ok {
			name = alias
		}
		gv, ok := g.vals[name]
		if !ok || !gv.isDesc || gv.goVar == "" {
			continue
		}
		out = append(out, gv.goVar)
	}
	return g.normalizeDescVarList(out)
}

// phiMovesRequireSingleChunk returns true if phi edge stores for this successor
// depend on phi values that are themselves stored in one of the destination
// slots on the same edge. In that case, chunking can clobber stack-backed
// sources before they are materialized.
func (g *codeGen) phiMovesRequireSingleChunk(moves []phiEdgeMove) bool {
	if len(moves) < 2 {
		return false
	}
	writes := make(map[string]struct{}, len(moves))
	for _, m := range moves {
		writes[m.phiOff] = struct{}{}
	}
	for _, m := range moves {
		edge := m.edge
		if edge == nil {
			continue
		}
		if _, isConst := edge.(*ssa.Const); isConst {
			continue
		}
		name := edge.Name()
		if alias, ok := g.ssaAliases[name]; ok {
			name = alias
		}
		// If a source is a phi value whose slot is also written on this edge,
		// emit all moves in one protected block to preserve simultaneous semantics.
		if srcOff, ok := g.phiRegs[name]; ok {
			if _, exists := writes[srcOff]; exists {
				return true
			}
		}
	}
	return false
}

// emitPhiMov emits a machine-code store from an SSA value to a phi stack slot.
// phiOff is the stack offset string (e.g. "0", "8", "16").
func (g *codeGen) emitPhiMov(phiOff string, v ssa.Value, phiType types.Type) {
	phiTriple := isPhiTripleType(phiType)
	phiPair := isPhiPairType(phiType)
	phiOffHi := "(" + phiOff + ")+8"
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, %s)", phiOff)
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else if c.Value.Kind() == constant.String {
			sval := constant.StringVal(c.Value)
			if phiPair {
				g.emit("ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(%q)}, %s)", sval, phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(%q)}, %s)", sval, phiOff)
			}
		} else if c.Value.Kind() == constant.Bool {
			bval := constant.BoolVal(c.Value)
			if bval {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, %s)", phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, %s)", phiOff)
			}
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			if signed, bits, ok := intTypeInfo(phiType); ok {
				ival = normalizeIntConstForType(ival, signed, bits)
			}
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%d)}, %s)", ival, phiOff)
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%v)}, %s)", fval, phiOff)
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.marker == "_variadic_args" {
			if !phiTriple {
				panic(fmt.Sprintf("variadic args require a slice phi: %s", v))
			}
			end := ":"
			if src.variadicLenKnown {
				end = fmt.Sprintf(":%d", src.variadicOffset+src.variadicLen)
			}
			materialized := g.allocDesc()
			g.emit("%s := jitMaterializeVirtualGoSlice(ctx, args[%d%s])", materialized, src.variadicOffset, end)
			g.emit("ctx.EmitStoreRegMem(%s.Reg, RegRSP, %s)", materialized, phiOff)
			g.emit("ctx.EmitStoreRegMem(%s.Reg2, RegRSP, %s+8)", materialized, phiOff)
			g.emit("ctx.EmitStoreRegMem(%s.Reg3, RegRSP, %s+16)", materialized, phiOff)
			g.emit("ctx.FreeDesc(&%s)", materialized)
			return
		}
		if src.isDesc {
			edgeSrc := g.allocDesc()
			g.emit("%s := %s", edgeSrc, src.goVar)
			g.emit("if %s.Loc == LocNone { panic(\"jit: phi source has no location\") }", edgeSrc)
			if phiTriple {
				g.emit("ctx.SyncDesc(&%s)", edgeSrc)
				g.emit("if %s.Loc == LocStackTriple {", edgeSrc)
				g.emit("\tctx.EmitCopyStackWords(%s, %s, 3)", edgeSrc, phiOff)
				g.emit("} else {")
				g.emit("\tif %s.Loc != LocRegTriple { panic(\"jit: slice phi source is not a triple\") }", edgeSrc)
				g.emit("\tctx.EmitStoreRegMem(%s.Reg, RegRSP, %s)", edgeSrc, phiOff)
				g.emit("\tctx.EmitStoreRegMem(%s.Reg2, RegRSP, %s+8)", edgeSrc, phiOff)
				g.emit("\tctx.EmitStoreRegMem(%s.Reg3, RegRSP, %s+16)", edgeSrc, phiOff)
				g.emit("}")
				return
			}
			if phiPair {
				g.emit("ctx.SyncDesc(&%s)", edgeSrc)
				g.emit("if %s.Loc == LocStackPair {", edgeSrc)
				g.emit("\tctx.EmitCopyStackWords(%s, %s, 2)", edgeSrc, phiOff)
				g.emit("} else if %s.Loc == LocInputPair {", edgeSrc)
				g.emit("\tctx.EnsureDesc(&%s)", edgeSrc)
				g.emit("\tctx.EmitStoreScmerToStack(%s, %s)", edgeSrc, phiOff)
				g.emit("} else if %s.Loc == LocRegPair || %s.Loc == LocImm {", edgeSrc, edgeSrc)
				g.emit("\tctx.EmitStoreScmerToStack(%s, %s)", edgeSrc, phiOff)
				g.emit("} else {")
				g.emit("\tctx.EnsureDesc(&%s)", edgeSrc)
				g.emit("\tctx.EmitStoreToStack(%s, %s)", edgeSrc, phiOff)
				g.emit("\tctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
				g.emit("}")
				return
			}
			g.emit("ctx.EnsureDesc(&%s)", edgeSrc)
			if signed, bits, ok := intTypeInfo(phiType); ok && bits > 0 && bits < 64 {
				tmp := g.allocDesc()
				g.emit("%s := %s", tmp, edgeSrc)
				if signed {
					g.emitNormalizeSignedNarrow(tmp, bits)
				} else {
					g.emitNormalizeUnsignedNarrow(tmp, bits)
				}
				g.emit("ctx.EmitStoreToStack(%s, %s)", tmp, phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(%s, %s)", edgeSrc, phiOff)
			}
			// Note: we do NOT call useOperand here. Phi edge references keep the
			// value alive (inflated refcount) but are not consumed. This prevents
			// over-decrement when the same value appears on mutually exclusive
			// conditional paths (each path's emitPhiMov runs at codegen time).
		} else {
			panic(fmt.Sprintf("phi edge references unknown value in BB%d: %s", g.curBlock, v))
		}
	}
}

// emitEdgePhiMovesIndent is like emitEdgePhiMoves but with a given indent prefix.
func (g *codeGen) emitEdgePhiMovesIndent(targetBBIdx int, succPos int, indent string) {
	targetBlock := g.fn.Blocks[targetBBIdx]
	edgeIdx, ok := g.phiEdgeIndexForSucc(targetBBIdx, succPos)
	if !ok {
		return
	}
	phiIdx := 0
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		_ = indent
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			panic(fmt.Sprintf("phi edge index out of range for %s: edge=%d len=%d", phi.Name(), edgeIdx, len(phi.Edges)))
		}
		edge := phi.Edges[edgeIdx]
		g.emitPhiMov(g.phiSlotOffExpr(targetBBIdx, phiIdx), edge, phi.Type())
		phiIdx++
	}
}

// emitPhiMovIndent emits a phi stack store with a given indent prefix.
func (g *codeGen) emitPhiMovIndent(phiOff string, v ssa.Value, indent string) {
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)\n", indent, phiOff)
		} else if c.Value.Kind() == constant.String {
			sval := constant.StringVal(c.Value)
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(%q)}, %s)\n", indent, sval, phiOff)
		} else if c.Value.Kind() == constant.Bool {
			bval := constant.BoolVal(c.Value)
			var ival int
			if bval {
				ival = 1
			}
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(%d)}, %s)\n", indent, ival, phiOff)
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(%d)}, %s)\n", indent, ival, phiOff)
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewFloat(%v)}, %s)\n", indent, fval, phiOff)
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.isDesc {
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(%s, %s)\n", indent, src.goVar, phiOff)
			// Note: no useOperand — same reasoning as emitPhiMov.
		} else {
			panic(fmt.Sprintf("phi edge references unknown value: %s", v))
		}
	}
}

// allocPhiRegs pre-scans the function for phis, counts all phi nodes across BBs,
// and assigns fixed 16-byte stack slots.
// Phi values live on the stack at [RSP + offset] to avoid register pressure.
// A temp register is allocated on each read and freed after use.
//
// Offsets are local to this function's phi frame. Every inlined helper reserves
// its own frame through AllocStack, so carrying the caller's absolute offset into
// the callee would add the frame base twice.
func (g *codeGen) allocPhiRegs() {
	offset := 0
	for _, block := range g.fn.Blocks {
		phis := g.blockPhis(block.Index)
		if len(phis) == 0 {
			continue
		}
		g.bbPhiBase[block.Index] = offset
		g.bbPhiCount[block.Index] = len(phis)
		for _, phi := range phis {
			phiName := phi.Name()
			triple := isPhiTripleType(phi.Type())
			pair := isPhiPairType(phi.Type())
			g.phiRegs[phiName] = fmt.Sprintf("%d", offset)
			g.phiPair[phiName] = pair
			g.phiTriple[phiName] = triple
			g.phiTypeTag[phiName] = jitTagForSSAType(phi.Type())
			if triple {
				offset += 24
			} else {
				offset += phiSlotBytes
			}
		}
	}
	g.phiStackSize = offset

	// Allocate phi space from the unified frame. Generated CFGs may reserve
	// additional callback and spill homes after these slots, so the bump
	// allocator cannot release only this prefix at the emitter epilogue.
	// No SUB RSP here — the outer compiler owns the frame.
	if g.phiStackSize > 0 {
		phiBaseVar := g.allocTemp("phiBase")
		if g.bbClosureMode {
			if g.closureRegDecl == nil {
				g.closureRegDecl = map[string]bool{}
			}
			if !g.closureRegDecl[phiBaseVar] {
				g.closureRegDecl[phiBaseVar] = true
				fmt.Fprintf(&g.wDecl, "\t\t\tvar %s int32\n", phiBaseVar)
				fmt.Fprintf(&g.wDecl, "\t\t\t_ = %s\n", phiBaseVar)
			}
			g.emit("%s = ctx.AllocStack(int32(%d))", phiBaseVar, g.phiStackSize)
		} else {
			g.emit("%s := ctx.AllocStack(int32(%d))", phiBaseVar, g.phiStackSize)
		}
		g.phiFrameFixup = phiBaseVar
	}
}

// initAllPhiDescs materializes descriptors for all phi values so resolveValue
// works independently from BB declaration order while emitting recursive
// renderers.
func (g *codeGen) initAllPhiDescs() {
	for _, bb := range g.fn.Blocks {
		for _, instr := range bb.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				break
			}
			name := phi.Name()
			phiOff, ok := g.phiRegs[name]
			if !ok {
				continue
			}
			phiTag := g.phiTypeTag[name]
			if phiTag == "" {
				phiTag = "JITTypeUnknown"
			}
			dv := g.allocDesc()
			phiBaseExpr := ""
			if g.phiFrameFixup != "" {
				phiBaseExpr = "int32(" + g.phiFrameFixup + ")+"
			}
			if g.phiTriple[name] {
				g.emit("%s := JITValueDesc{Loc: LocStackTriple, Type: %s, StackOff: %sint32(%s)}", dv, phiTag, phiBaseExpr, phiOff)
			} else if g.phiPair[name] {
				g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: %s, StackOff: %sint32(%s)}", dv, phiTag, phiBaseExpr, phiOff)
			} else {
				g.emit("%s := JITValueDesc{Loc: LocStack, Type: %s, StackOff: %sint32(%s)}", dv, phiTag, phiBaseExpr, phiOff)
			}
			g.emit("_ = %s", dv)
			generated := genVal{goVar: dv, isDesc: true}
			if g.phiTriple[name] {
				generated.marker = "_slice"
				generated.pinAcrossBlock = true
			} else if _, ok := phi.Type().Underlying().(*types.Signature); ok {
				generated.marker = "_gofunc"
				generated.pinAcrossBlock = true
			}
			g.vals[name] = generated
		}
	}
}

// inlineCall inlines a callee's SSA into the current code generation.
// The callee's params are mapped to the caller's args, and the callee's
// return value is captured. Returns the genVal representing the result.
func (g *codeGen) inlineCall(callee *ssa.Function, callArgs []ssa.Value) genVal {
	return g.inlineCallCaptured(callee, callArgs, nil)
}

func (g *codeGen) inlineCallCaptured(callee *ssa.Function, callArgs []ssa.Value, captures []closureBinding) genVal {
	// Resolve caller's arguments BEFORE switching state
	resolvedArgs := make([]genVal, len(callArgs))
	for i, arg := range callArgs {
		resolvedArgs[i] = g.resolveValue(arg)
	}

	// Save caller state
	savedFn := g.fn
	savedBBQueue := g.bbQueue
	savedBBDone := g.bbDone
	savedBBQueued := g.bbQueued
	savedBBLabels := g.bbLabels
	savedBBPosVars := g.bbPosVars
	savedBBScope := g.bbScope
	savedCurBlock := g.curBlock
	savedPhiRegs := g.phiRegs
	savedPhiPair := g.phiPair
	savedPhiTriple := g.phiTriple
	savedPhiTypeTag := g.phiTypeTag
	savedBBPhiBase := g.bbPhiBase
	savedBBPhiCount := g.bbPhiCount
	savedPhiStackSize := g.phiStackSize
	savedPhiFrameFixup := g.phiFrameFixup
	savedVals := g.vals
	savedMultiBlock := g.multiBlock
	savedEndLabel := g.endLabel
	savedInlineReturnReg := g.inlineReturnRegVar
	savedInlineReturnReg2 := g.inlineReturnReg2Var
	savedInlineReturnsScm := g.inlineReturnsScm
	savedInlineReturnTuple := g.inlineReturnTuple
	savedInlineEndLabel := g.inlineEndLabel
	savedReturnPhiReg := g.returnPhiReg
	savedReturnPhiReg2 := g.returnPhiReg2
	savedRefCounts := g.refCounts
	savedAliases := g.ssaAliases
	savedFieldCache := g.fieldCache
	savedPhiProtected := g.phiProtectedRegVars
	savedTypeName := g.typeName
	savedForceLegacyCFG := g.forceLegacyCFG

	// Set up callee state
	g.fn = callee
	if recv := callee.Signature.Recv(); recv != nil {
		switch rt := recv.Type().(type) {
		case *types.Pointer:
			if n, ok := rt.Elem().(*types.Named); ok && n.Obj() != nil {
				g.typeName = n.Obj().Name()
			}
		case *types.Named:
			if rt.Obj() != nil {
				g.typeName = rt.Obj().Name()
			}
		}
	}
	g.bbQueue = nil
	g.bbDone = map[uint64]bool{}
	g.bbQueued = map[uint64]bool{}
	g.bbLabels = map[uint64]string{}
	g.bbPosVars = map[uint64]string{}
	// Allocate a globally unique namespace for each inline call.
	g.nextBBScope++
	g.bbScope = g.nextBBScope
	g.phiRegs = map[string]string{}
	g.phiPair = map[string]bool{}
	g.phiTriple = map[string]bool{}
	g.phiTypeTag = map[string]string{}
	g.bbPhiBase = map[int]int{}
	g.bbPhiCount = map[int]int{}
	g.phiFrameFixup = ""
	g.vals = map[string]genVal{}
	g.refCounts = computeRefCounts(callee)
	g.ssaAliases = map[string]string{}
	// Do not share cached field loads across inline boundaries:
	// the callee receiver may be a different sub-struct (e.g. multiple inlined
	// StorageInt receivers inside StorageString/StorageSeq).
	g.fieldCache = map[string]genVal{}
	g.forceLegacyCFG = true

	// Map callee params -> resolved caller args.
	// Always use per-inline descriptor copies so callee-side FreeDesc/Loc
	// rewrites cannot mutate caller descriptor variables by alias.
	for i, param := range callee.Params {
		arg := resolvedArgs[i]
		isReceiverParam := (callee.Signature.Recv() != nil && i == 0) || arg.goVar == "thisptr" || arg.marker == "_storage_recv"
		if arg.isDesc && !isReceiverParam && g.refCounts[param.Name()] > 0 {
			pv := g.allocDesc()
			g.emit("%s := %s", pv, arg.goVar)
			g.emit("_ = %s", pv)
			g.emit("ctx.StabilizeDescForControlFlow(&%s)", pv)
			copied := arg
			copied.goVar = pv
			g.vals[param.Name()] = copied
		} else {
			g.vals[param.Name()] = arg
		}
	}
	if len(captures) != len(callee.FreeVars) {
		panic(fmt.Sprintf("closure capture count %d does not match %d free variables", len(captures), len(callee.FreeVars)))
	}
	for i, freeVar := range callee.FreeVars {
		g.vals[freeVar.Name()] = captures[i].value
	}

	// Give caller values that remain live after the inline call a stable stack
	// home. Recursive/multi-block emitters may then use the complete register
	// bank without accumulating register protections across their CFG.
	for i, arg := range callArgs {
		if _, isConst := arg.(*ssa.Const); isConst {
			continue
		}
		argName := arg.Name()
		if alias, ok := savedAliases[argName]; ok {
			argName = alias
		}
		// Conservative correctness-first policy:
		// Every non-constant argument may still be needed by the caller after
		// this inline site (especially across phi edges / nested inlines).
		// Prevent destructive parameter reuse in the callee and prevent spills
		// of caller-live argument registers while the inline body emits.
		_ = argName
		g.refCounts[callee.Params[i].Name()]++
		if savedRefCounts[argName] <= 1 {
			continue
		}
		resolved := resolvedArgs[i]
		if resolved.isDesc {
			g.emit("ctx.StabilizeDescForControlFlow(&%s)", resolved.goVar)
		}
	}

	// Pre-allocate phi regs for callee
	g.allocPhiRegs()
	g.initAllPhiDescs()

	isMultiBlock := len(callee.Blocks) > 1
	g.multiBlock = isMultiBlock

	// Detect if callee returns Scmer (2-word pair) or scalar (1 word).
	returnsScmer := false
	results := callee.Signature.Results()
	if results.Len() == 1 {
		if named, ok := results.At(0).Type().(*types.Named); ok && named.Obj().Name() == "Scmer" {
			returnsScmer = true
		}
	}

	// For multi-block, reserve only an end label.
	// Return registers are allocated lazily on first encountered Return.
	if isMultiBlock {
		g.inlineReturnRegVar = ""
		g.inlineReturnReg2Var = ""
		g.inlineReturnsScm = returnsScmer
		g.inlineReturnTuple = nil
		stackBackedSingleResult := results.Len() == 1 && !returnsScmer && goCallWordCount(results.At(0).Type()) > 1
		if results.Len() > 1 || stackBackedSingleResult {
			g.inlineReturnTuple = make([]genVal, results.Len())
			for i := 0; i < results.Len(); i++ {
				words := goCallWordCount(results.At(i).Type())
				if words < 1 || words > 3 {
					panic(fmt.Sprintf("unsupported inline result type %s", results.At(i).Type()))
				}
				off := g.allocTemp("inlineResultOff")
				dv := g.allocDesc()
				g.emit("%s := ctx.AllocStack(int32(%d))", off, words*8)
				loc := "LocStack"
				marker := ""
				if words == 2 {
					loc = "LocStackPair"
				} else if words == 3 {
					loc = "LocStackTriple"
					marker = "_slice"
				}
				if basic, ok := results.At(i).Type().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
					marker = "_gostring"
				}
				g.emit("%s := JITValueDesc{Loc: %s, Type: %s, StackOff: %s}", dv, loc, jitTagForSSAType(results.At(i).Type()), off)
				g.inlineReturnTuple[i] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: true}
			}
		}

		inlineEnd := g.allocLabel()
		g.emit("%s := ctx.ReserveLabel()", inlineEnd)
		g.inlineEndLabel = inlineEnd
		g.endLabel = "" // don't use outer endLabel
	} else {
		g.inlineReturnRegVar = ""
		g.inlineReturnReg2Var = ""
		g.inlineReturnsScm = false
		g.inlineReturnTuple = nil
		g.inlineEndLabel = ""
		g.endLabel = ""
	}

	// Process callee blocks
	var singleBlockResult genVal
	for i := range callee.Blocks {
		g.ensureBBPosVar(i)
		// A branch may discover an edge to a block that has already been
		// rendered in the generator's work-list order. Reserving every block
		// label up front guarantees that such cross- and back-edges target the
		// label marked at block entry instead of creating an orphaned label.
		g.ensureBBLabel(i)
	}
	g.bbQueue = []int{0}
	g.bbQueued[g.scopedBBID(0)] = true
	for len(g.bbQueue) > 0 {
		bbIdx := g.bbQueue[0]
		g.bbQueue = g.bbQueue[1:]
		bbID := g.scopedBBID(bbIdx)
		delete(g.bbQueued, bbID)
		if g.bbDone[bbID] {
			continue
		}
		g.bbDone[bbID] = true
		g.curBlock = bbIdx

		if posVar, ok := g.bbPosVars[bbID]; ok {
			g.emit("%s = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))", posVar)
		}
		if lbl, ok := g.bbLabels[bbID]; ok {
			g.emit("ctx.MarkLabel(%s)", lbl)
			g.emit("ctx.ResolveFixups()")
		}
		g.resetAllPhiDescsToStack()
		g.emit("ctx.ReclaimUntrackedRegs()")

		block := callee.Blocks[bbIdx]
		if blockEndsInPanic(block) {
			// Error-only blocks stay native but do not participate in register
			// allocation for successful paths. Scheme treats unsupported inputs
			// as panics; constructing the helper's formatted Go error here would
			// otherwise retain an unnecessary Go-call boundary in every emitter.
			g.emit("ctx.EmitGoPanic(%q)", "jit: invalid arguments for inlined Go helper")
			continue
		}
		for _, instr := range block.Instrs {
			g.emit("ctx.ReclaimUntrackedRegs()")
			if ret, ok := instr.(*ssa.Return); ok && !isMultiBlock {
				// Single-block: capture return value directly, no code emitted
				if len(ret.Results) > 0 {
					singleBlockResult = g.resolveValue(ret.Results[0])
				}
				break
			} else {
				g.emitInstr(instr)
				g.stabilizeCrossBlockValue(instr)
				g.freeDeadOperands(instr)
				if _, isRet := instr.(*ssa.Return); isRet {
					break
				}
			}
		}
	}

	if isMultiBlock {
		g.emit("ctx.MarkLabel(%s)", g.inlineEndLabel)
	}
	// Resolve fixups only once at top-level end. Inline bodies may run while
	// outer-function labels are still pending.
	// Note: no ADD RSP for inlined callee's phis — the unified phi frame
	// is managed by the outer function (allocated via fixup, freed at end).

	// Determine result
	var result genVal
	if isMultiBlock {
		if results.Len() == 0 {
			result = genVal{}
		} else if len(g.inlineReturnTuple) > 0 {
			if results.Len() == 1 {
				result = g.inlineReturnTuple[0]
			} else {
				result = genVal{tuple: g.inlineReturnTuple}
			}
		} else if g.inlineReturnRegVar == "" {
			panic(fmt.Sprintf("inline callee has no return register: %s", callee))
		} else if g.inlineReturnsScm {
			dv := g.allocDesc()
			// Wrap the register pair in a JITValueDesc (Scmer = 2 words)
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, g.inlineReturnRegVar, g.inlineReturnReg2Var)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnRegVar, dv)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnReg2Var, dv)
			result = genVal{goVar: dv, isDesc: true}
		} else {
			dv := g.allocDesc()
			// Wrap the bare register in a JITValueDesc for type safety
			g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, g.inlineReturnRegVar)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnRegVar, dv)
			result = genVal{goVar: dv, isDesc: true}
		}
	} else {
		result = singleBlockResult
	}

	updatedCaptures := make([]genVal, len(captures))
	for i, freeVar := range callee.FreeVars {
		updatedCaptures[i] = g.vals[freeVar.Name()]
	}
	updatedCells := make(map[string]genVal)
	for _, value := range g.vals {
		if value.cellName == "" || value.cellScope != g.bbScope {
			continue
		}
		// Store updates the allocation's SSA key. Use that canonical cell entry;
		// selecting an arbitrary alias from the map can return an older value or
		// even a differently shaped intermediate carrying the same cellName.
		if authoritative, ok := g.vals[value.cellName]; ok {
			updatedCells[value.cellName] = authoritative
		}
	}

	// Restore caller state
	g.fn = savedFn
	g.bbQueue = savedBBQueue
	g.bbDone = savedBBDone
	g.bbQueued = savedBBQueued
	g.bbLabels = savedBBLabels
	g.bbPosVars = savedBBPosVars
	g.bbScope = savedBBScope
	g.curBlock = savedCurBlock
	g.phiRegs = savedPhiRegs
	g.phiPair = savedPhiPair
	g.phiTriple = savedPhiTriple
	g.phiTypeTag = savedPhiTypeTag
	g.bbPhiBase = savedBBPhiBase
	g.bbPhiCount = savedBBPhiCount
	g.phiStackSize = savedPhiStackSize
	g.phiFrameFixup = savedPhiFrameFixup
	g.vals = savedVals
	g.multiBlock = savedMultiBlock
	g.endLabel = savedEndLabel
	g.inlineReturnRegVar = savedInlineReturnReg
	g.inlineReturnReg2Var = savedInlineReturnReg2
	g.inlineReturnsScm = savedInlineReturnsScm
	g.inlineReturnTuple = savedInlineReturnTuple
	g.inlineEndLabel = savedInlineEndLabel
	g.returnPhiReg = savedReturnPhiReg
	g.returnPhiReg2 = savedReturnPhiReg2
	g.refCounts = savedRefCounts
	g.ssaAliases = savedAliases
	g.fieldCache = savedFieldCache
	g.phiProtectedRegVars = savedPhiProtected
	g.typeName = savedTypeName
	g.forceLegacyCFG = savedForceLegacyCFG
	for cellName, updated := range updatedCells {
		if updated.cellScope == savedBBScope {
			g.vals[cellName] = updated
		}
	}
	for i, capture := range captures {
		// A closure can flow through another inlined function before it is
		// invoked. SSA temporary names are only unique within one function;
		// never let such a transitive capture overwrite an unrelated value
		// (notably a phi) in the intermediate caller's namespace.
		if capture.scope == savedBBScope {
			g.vals[capture.outerName] = updatedCaptures[i]
		}
	}

	return result
}

func inlineInstructionCount(fn *ssa.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		count += len(block.Instrs)
	}
	return count
}

// functionBuildsScmerStruct reports helpers which assemble Scmer through a
// local aggregate and FieldAddr stores. Those stores need a descriptor owned
// by the caller's register namespace; keeping the helper as a native Go call
// is both smaller and safer than expanding its allocation graph inline.
func functionBuildsScmerStruct(fn *ssa.Function) bool {
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			allocation, ok := instruction.(*ssa.Alloc)
			if !ok {
				continue
			}
			pointer, ok := allocation.Type().Underlying().(*types.Pointer)
			if ok && isScmerType(pointer.Elem()) {
				return true
			}
		}
	}
	return false
}

func blockEndsInPanic(block *ssa.BasicBlock) bool {
	return block != nil && len(block.Instrs) > 0 && func() bool {
		_, ok := block.Instrs[len(block.Instrs)-1].(*ssa.Panic)
		return ok
	}()
}

// tryInlineCall prefers SSA visibility for small helpers, but treats an
// unsupported instruction as a local boundary instead of rejecting the whole
// builtin. The cloned generator makes this rollback exact.
func (g *codeGen) tryInlineCall(callee *ssa.Function, callArgs []ssa.Value) (result genVal, ok bool) {
	if callee == nil || callee.Blocks == nil || callee.Signature.Results().Len() > 1 {
		return genVal{}, false
	}
	// Inline package-local helpers because they expose the builtin's own type and
	// callback flow to JITGen. Larger calls into other packages are already
	// optimized Go entry points; expanding their implementation duplicates
	// library machinery and crosses an implementation boundary which the emitter
	// does not own. Tiny foreign helpers remain eligible when a native call cannot
	// represent their receiver efficiently.
	if callee.Pkg != nil && callee.Pkg.Pkg != nil && callee.Pkg.Pkg.Path() != g.topLevelPkgPath && inlineInstructionCount(callee) > 32 {
		return genVal{}, false
	}
	// Pointer-receiver methods preserve object identity and commonly combine
	// field mutation with maps, slices, or write barriers. Keep that compact Go
	// call boundary while still inlining the surrounding builtin loop and its
	// Scheme callbacks. Value-receiver helpers remain normal inline candidates.
	if receiver := callee.Signature.Recv(); receiver != nil {
		if _, pointerReceiver := receiver.Type().Underlying().(*types.Pointer); pointerReceiver {
			return genVal{}, false
		}
	}
	if callee.Signature.Results().Len() == 1 && goCallWordCount(callee.Signature.Results().At(0).Type()) == 0 {
		return genVal{}, false
	}
	if functionBuildsScmerStruct(callee) {
		return genVal{}, false
	}
	if cost := inlineInstructionCount(callee); cost > inlineInstructionBudget {
		if verbose {
			fmt.Fprintf(os.Stderr, "jitgen: not inlining %s: cost %d exceeds budget %d\n", callee, cost, inlineInstructionBudget)
		}
		return genVal{}, false
	}
	cost := inlineInstructionCount(callee)
	if g.inlineInstructions+cost > emitterInlineInstructionBudget {
		return genVal{}, false
	}
	trial := g.clone()
	trial.inlineInstructions += cost
	defer func() {
		if recovered := recover(); recovered != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "jitgen: cannot inline %s locally: %v\n", callee, recovered)
			}
			result = genVal{}
			ok = false
		}
	}()
	result = trial.inlineCall(callee, callArgs)
	generated := trial.w.String()
	declarations := trial.wDecl.String()
	trial.w = strings.Builder{}
	trial.wDecl = strings.Builder{}
	*g = *trial
	g.w.WriteString(generated)
	g.wDecl.WriteString(declarations)
	return result, true
}

func (g *codeGen) tryInlineClosure(closure genVal, callArgs []ssa.Value) (result genVal, ok bool) {
	if closure.closureFn == nil || closure.closureFn.Signature.Results().Len() > 1 || inlineInstructionCount(closure.closureFn) > inlineInstructionBudget {
		return genVal{}, false
	}
	cost := inlineInstructionCount(closure.closureFn)
	if g.inlineInstructions+cost > emitterInlineInstructionBudget {
		return genVal{}, false
	}
	trial := g.clone()
	trial.inlineInstructions += cost
	defer func() {
		if recovered := recover(); recovered != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "jitgen: cannot inline closure %s locally: %v\n", closure.closureFn, recovered)
			}
			result = genVal{}
			ok = false
		}
	}()
	result = trial.inlineCallCaptured(closure.closureFn, callArgs, closure.closureBindings)
	generated := trial.w.String()
	declarations := trial.wDecl.String()
	trial.w = strings.Builder{}
	trial.wDecl = strings.Builder{}
	*g = *trial
	g.w.WriteString(generated)
	g.wDecl.WriteString(declarations)
	return result, true
}

func (g *codeGen) emitSpecializedPhiStackWrites(bbIdx int, psVar string, indent string) {
	phis := g.blockPhis(bbIdx)
	for phiIdx, phi := range phis {
		tmp := g.allocDesc()
		phiOff := g.phiSlotOffExpr(bbIdx, phiIdx)
		g.emit("%sif len(%s.PhiValues) > %d && %s.PhiValues[%d].Loc != LocNone {", indent, psVar, phiIdx, psVar, phiIdx)
		g.emit("%s\t%s := %s.PhiValues[%d]", indent, tmp, psVar, phiIdx)
		g.emit("%s\tctx.EnsureDesc(&%s)", indent, tmp)
		if isPhiTripleType(phi.Type()) {
			g.emit("%s\tctx.EmitStoreRegMem(%s.Reg, RegRSP, %s)", indent, tmp, phiOff)
			g.emit("%s\tctx.EmitStoreRegMem(%s.Reg2, RegRSP, %s+8)", indent, tmp, phiOff)
			g.emit("%s\tctx.EmitStoreRegMem(%s.Reg3, RegRSP, %s+16)", indent, tmp, phiOff)
		} else if isPhiPairType(phi.Type()) {
			g.emit("%s\tctx.EmitStoreScmerToStack(%s, %s)", indent, tmp, phiOff)
		} else {
			g.emit("%s\tctx.EmitStoreToStack(%s, %s)", indent, tmp, phiOff)
		}
		g.emit("%s}", indent)
	}
}

func (g *codeGen) emitRecursiveBBRenderers() {
	prevMode := g.bbClosureMode
	g.bbClosureMode = true
	defer func() { g.bbClosureMode = prevMode }()

	for i := range g.fn.Blocks {
		g.ensureBBPosVar(i)
		g.ensureBBLabel(i)
	}

	for bbIdx, block := range g.fn.Blocks {
		bbID := g.scopedBBID(bbIdx)
		lbl := g.bbLabels[bbID]
		posVar := g.bbPosVars[bbID]

		g.emit("bbs[%d].RenderPS = func(ps PhiState) JITValueDesc {", bbIdx)
		g.emit("if !ps.General {")
		g.emitSpecializedPhiStackWrites(bbIdx, "ps", "\t")
		// TODO: specialization/unrolling disabled until phi constant propagation
		// is properly implemented. Factors to consider: loop iteration savings,
		// fetch elimination, register pressure trade-offs.
		g.emit("\tif bbs[%d].VisitCount >= 0 {", bbIdx)
		g.emit("\t\tps.General = true")
		g.emit("\t\treturn bbs[%d].RenderPS(ps)", bbIdx)
		g.emit("\t}")
		g.emit("}")
		g.emit("bbs[%d].VisitCount++", bbIdx)
		g.emit("if ps.General {")
		g.emit("\tif bbs[%d].Rendered {", bbIdx)
		g.emit("\t\tctx.EmitJmp(%s)", lbl)
		g.emit("\t\treturn result")
		g.emit("\t}")
		g.emit("\tbbs[%d].Rendered = true", bbIdx)
		g.emit("\tbbs[%d].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))", bbIdx)
		g.emit("\t%s = bbs[%d].Address", posVar, bbIdx)
		g.emit("\tctx.MarkLabel(%s)", lbl)
		g.emit("\tctx.ResolveFixups()")
		g.emit("}")

		g.curBlock = bbIdx
		g.resetAllPhiDescsToStack()
		g.applyPhiStateOverlay(bbIdx)
		g.emit("ctx.ReclaimUntrackedRegs()")
		g.emitPinDescVars(g.externalDescVars(block))
		if blockEndsInPanic(block) && !g.storageMode && g.opName != "" {
			g.emit("_ = jitEmitGoVariadicCallFromDescs(ctx, declarations[%q].Fn, args, result)", g.opName)
			g.emit("ctx.EmitGoPanic(%q)", "jit: builtin panic boundary unexpectedly returned")
			g.emit("return result")
			g.emit("}")
			continue
		}

		for _, instr := range block.Instrs {
			g.emitInstr(instr)
			g.stabilizeCrossBlockValue(instr)
			g.freeDeadOperands(instr)
			if _, isRet := instr.(*ssa.Return); isRet {
				break
			}
		}
		g.emit("return result")
		g.emit("}")
	}
}

// generateClosure tries to generate a JIT emitter closure for the given SSA function.
// Returns (closureCode, "") on success, or ("", errorDescription) on failure.
func newCodeGen(fn *ssa.Function, rewrite ssaValueRewriter, sourceAliases ...map[string]string) *codeGen {
	imports := make(map[string]string)
	if len(sourceAliases) > 0 {
		for path, alias := range sourceAliases[0] {
			imports[path] = alias
		}
	} else {
		for _, imported := range fn.Pkg.Pkg.Imports() {
			imports[imported.Path()] = imported.Name()
		}
	}
	return &codeGen{
		vals:                 map[string]genVal{},
		fn:                   fn,
		bbLabels:             map[uint64]string{},
		bbPosVars:            map[uint64]string{},
		bbDone:               map[uint64]bool{},
		bbQueued:             map[uint64]bool{},
		inlineCallSeq:        map[uint64]uint32{},
		phiRegs:              map[string]string{},
		phiPair:              map[string]bool{},
		phiTriple:            map[string]bool{},
		phiTypeTag:           map[string]string{},
		bbPhiBase:            map[int]int{},
		bbPhiCount:           map[int]int{},
		fieldCache:           map[string]genVal{},
		refCounts:            computeRefCounts(fn),
		directResultPayloads: computeDirectResultPayloads(fn),
		ssaAliases:           map[string]string{},
		topLevelPkgPath:      fn.Pkg.Pkg.Path(),
		importedPkgAlias:     imports,
		valueRewriter:        rewrite,
	}
}

// emitBodyConfig parametrizes the divergent parts of the shared emitter body.
type emitBodyConfig struct {
	entryGeneral     bool   // PhiState{General: ...} at entry (closure: true, storage: false)
	useReturnPhiRegs bool   // allocate returnPhiReg/Reg2 for multi-block merge (storage only)
	bbsDeclPrefix    string // "scm." for storage package, "" for scm package
}

// emitBody generates the shared core: phi allocation, BB renderers, entry call, epilogue.
// For single-block functions, it skips the BB closure infrastructure entirely.
func (g *codeGen) emitBody(cfg emitBodyConfig) {
	g.allocPhiRegs()
	g.initAllPhiDescs()

	// Single-block fast path: no BB closures, no phi state, just emit instructions inline.
	if len(g.fn.Blocks) == 1 {
		pinnedArgRegs := g.emitProtectIncomingArgRegs()
		g.curBlock = 0
		for _, instr := range g.fn.Blocks[0].Instrs {
			g.emitInstr(instr)
			g.freeDeadOperands(instr)
			if _, isRet := instr.(*ssa.Return); isRet {
				break
			}
		}
		g.emitUnprotectIncomingArgRegs(pinnedArgRegs)
		if g.hasStorageIdx {
			g.emit("if idxPinned { ctx.UnprotectReg(idxPinnedReg) }")
		}
		g.emit("return result")
		return
	}

	// Multi-block path: full BB closure infrastructure.
	g.emit("var bbs [%d]%sBBDescriptor", len(g.fn.Blocks), cfg.bbsDeclPrefix)
	g.emitBBPhiLayout()

	if g.multiBlock {
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tctx.BindReg(result.Reg, &result)")
		g.emit("\tctx.BindReg(result.Reg2, &result)")
		g.emit("}")
		// A multi-block emitter writes every return arm into the caller-selected
		// pair. Reserve that pair for the complete CFG render: path-local register
		// reclamation must never recycle an output register before the arm which
		// actually produces the runtime result has been emitted.
		g.emit("resultRegsProtected := result.Loc == LocRegPair")
		g.emit("if resultRegsProtected {")
		g.emit("\tctx.ProtectReg(result.Reg)")
		g.emit("\tctx.ProtectReg(result.Reg2)")
		g.emit("}")
		if cfg.useReturnPhiRegs {
			g.returnPhiReg = g.allocReg()
			g.returnPhiReg2 = g.allocReg()
			g.emit("%s := ctx.AllocReg()", g.returnPhiReg)
			g.emit("%s := ctx.AllocRegExcept(%s)", g.returnPhiReg2, g.returnPhiReg)
		}
		g.endLabel = g.allocLabel()
		g.emit("%s := ctx.ReserveLabel()", g.endLabel)
	}

	g.emitRecursiveBBRenderers()
	pinnedArgRegs := g.emitProtectIncomingArgRegs()
	entryPS := g.allocTemp("ps")
	g.emit("%s := %sPhiState{General: %v}", entryPS, cfg.bbsDeclPrefix, cfg.entryGeneral)
	g.emit("_ = bbs[0].RenderPS(%s)", entryPS)

	// Epilogue
	if g.multiBlock {
		g.emit("ctx.MarkLabel(%s)", g.endLabel)
		if cfg.useReturnPhiRegs && g.returnPhiReg != "" && g.returnPhiReg2 != "" {
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, g.returnPhiReg, g.returnPhiReg2)
			g.emit("ctx.EmitMovPairToResult(&%s, &result)", dv)
			g.emit("ctx.FreeReg(%s)", g.returnPhiReg)
			g.emit("ctx.FreeReg(%s)", g.returnPhiReg2)
		}
		g.emit("ctx.ResolveFixups()")
	}
	if g.hasStorageIdx {
		g.emit("if idxPinned { ctx.UnprotectReg(idxPinnedReg) }")
	}
	if g.multiBlock {
		g.emit("if resultRegsProtected {")
		g.emit("\tctx.UnprotectReg(result.Reg2)")
		g.emit("\tctx.UnprotectReg(result.Reg)")
		g.emit("}")
	}
	g.emitUnprotectIncomingArgRegs(pinnedArgRegs)
	g.emit("return result")
}

func generateClosure(opName string, fn *ssa.Function, rewrite ssaValueRewriter, sourcePath ...string) (string, string) {
	code, errMsg, _ := generateClosureCost(opName, fn, rewrite, sourcePath...)
	return code, errMsg
}

func generateClosureCost(opName string, fn *ssa.Function, rewrite ssaValueRewriter, sourcePath ...string) (code string, errMsg string, inlineCost uint16) {
	defer func() {
		if r := recover(); r != nil {
			if os.Getenv("JITGEN_DEBUG_PANIC") == "1" && (dumpOp == "" || dumpOp == opName) {
				panic(r)
			}
			code = ""
			errMsg = fmt.Sprintf("%v", r)
			inlineCost = 0
		}
	}()

	aliases := map[string]string{}
	if len(sourcePath) > 0 {
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), sourcePath[0], nil, parser.ImportsOnly)
		if parseErr != nil {
			panic(parseErr)
		}
		for _, imported := range parsed.Imports {
			path, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil {
				panic(unquoteErr)
			}
			alias := filepath.Base(path)
			if imported.Name != nil && imported.Name.Name != "." && imported.Name.Name != "_" {
				alias = imported.Name.Name
			}
			aliases[path] = alias
		}
	}
	g := newCodeGen(fn, rewrite, aliases)
	g.opName = opName
	fmt.Fprintf(&g.w, "\t\t\t%s\n", generatedBanner)
	if len(fn.Params) > 0 {
		g.paramName = fn.Params[0].Name()
		if _, ok := fn.Params[0].Type().Underlying().(*types.Slice); ok {
			g.vals[fn.Params[0].Name()] = genVal{marker: "_variadic_args"}
		}
	}

	g.multiBlock = len(fn.Blocks) > 1

	g.emitBody(emitBodyConfig{
		entryGeneral:  false,
		bbsDeclPrefix: "",
	})

	// Keep generated native emitters out of vanilla binaries. jitEnabled is a
	// build-tag-selected constant, so the Go compiler eliminates either this
	// fallback or the (potentially very large) native emitter body completely.
	// This prevents adding JIT coverage from perturbing non-JIT instruction
	// layout and performance while retaining one generated source of truth.
	guard := fmt.Sprintf("\tif !jitEnabled {\n\t\treturn jitEmitGoVariadicCallFromDescs(ctx, declarations[%q].Fn, args, result)\n\t}\n", opName)
	result := fmt.Sprintf("func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {\n%s%s%s\t\t}",
		guard, g.wDecl.String(), injectBindRegCalls(g.w.String()))
	cost := inlineInstructionCount(fn) + g.inlineInstructions
	if cost >= math.MaxUint16 {
		cost = math.MaxUint16 - 1
	}
	return result, "", uint16(cost)
}

// generateStorageBody generates the body of a JITEmit method from GetValue SSA.
// The generated code lives inside:
//
//	func (s *StorageXxx) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc { ... }
func generateStorageBody(typeName string, fn *ssa.Function, rewrite ssaValueRewriter) (code string, errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			code = ""
			errMsg = fmt.Sprintf("%v", r)
		}
	}()
	g := newCodeGen(fn, rewrite)
	g.storageMode = true
	g.typeName = typeName
	fmt.Fprintf(&g.w, "\t%s\n", generatedBanner)
	g.multiBlock = len(fn.Blocks) > 1
	g.returnPhiReg = ""
	g.returnPhiReg2 = ""

	// GetValue has 2 params: receiver (s *StorageXxx) and index (i uint32)
	// Map receiver to thisptr (LocImm at JIT compile time)
	if len(fn.Params) >= 1 {
		g.vals[fn.Params[0].Name()] = genVal{goVar: "thisptr", isDesc: true, marker: "_storage_recv"}
	}
	// Map index: idx is a Scmer (JITValueDesc), but GetValue's i is uint32.
	// Extract the integer value from the Scmer.
	// Dead code elimination: if the index parameter is never used in the body,
	// just free it and skip the conversion boilerplate.
	if len(fn.Params) >= 2 && g.refCounts[fn.Params[1].Name()] == 0 {
		g.emit("ctx.FreeDesc(&idx)")
	} else if len(fn.Params) >= 2 {
		g.emit("var idxInt JITValueDesc")
		g.emit("if idx.Loc == LocImm {")
		g.emit("\tidxInt = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(idx.Imm.Int())}")
		g.emit("} else if idx.Loc == LocRegPair {")
		g.emit("\tctx.FreeReg(idx.Reg)") // free ptr, keep aux (integer value)
		g.emit("\tidxInt = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: idx.Reg2}")
		g.emit("} else {")
		g.emit("\tidxInt = idx")
		g.emit("}")
		// GetValue's index parameter is uint32: normalize once at entry.
		g.emit("if idxInt.Loc == LocImm {")
		g.emit("\tidxInt = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}")
		g.emit("} else {")
		g.emit("\tctx.EnsureDesc(&idxInt)")
		g.emit("\tif idxInt.Loc != LocReg { panic(\"jit: idxInt not in register\") }")
		g.emit("\tctx.EmitShlRegImm8(idxInt.Reg, 32)")
		g.emit("\tctx.EmitShrRegImm8(idxInt.Reg, 32)")
		g.emit("\tctx.BindReg(idxInt.Reg, &idxInt)")
		g.emit("}")
		if g.multiBlock {
			g.emit("idxPinned := idxInt.Loc == LocReg")
			g.emit("idxPinnedReg := idxInt.Reg")
			g.emit("if idxPinned { ctx.ProtectReg(idxPinnedReg) }")
			g.hasStorageIdx = true
		}
		g.vals[fn.Params[1].Name()] = genVal{goVar: "idxInt", isDesc: true}
	}

	g.emitBody(emitBodyConfig{
		entryGeneral:     false,
		useReturnPhiRegs: true,
		bbsDeclPrefix:    "scm.",
	})

	code = g.wDecl.String() + g.w.String()
	// In storage mode, generated code goes in the storage package and needs scm. prefix
	if g.storageMode {
		code = addScmPrefix(code)
	}
	code = injectBindRegCalls(code)
	return code, ""
}

// addScmPrefix adds "scm." prefix to scm package identifiers in generated code.
// This is needed when the generated code goes into the storage package.
func addScmPrefix(code string) string {
	// Words that need the scm. prefix — these are exported identifiers from the scm package
	scmIdents := map[string]bool{
		"JITValueDesc": true, "JITTypeUnknown": true, "JITContext": true,
		"BBDescriptor": true, "PhiState": true,
		"LocNone": true, "LocReg": true, "LocRegPair": true, "LocRegTriple": true,
		"LocStack": true, "LocStackPair": true, "LocStackTriple": true, "LocInputPair": true, "LocMem": true, "LocImm": true, "LocAny": true,
		"NewInt": true, "NewFloat": true, "NewBool": true, "NewNil": true, "NewString": true,
		"NewFastDict": true, "NewFastDictValue": true,
		"Scmer": true, "GoFuncAddr": true, "JITBuildMergeClosure": true,
		"JITIntDiv": true, "JITEmitGoCallResults": true, "JITCloneScmerSlice": true, "JITAppendScmerSlice": true, "JITAppendScmerSliceCopy": true, "JITNewSliceCopy": true,
		"JITPanic":                     true,
		"EnsureDesc":                   true,
		"ConcatStrings":                true,
		"OptimizeProcToSerialFunction": true,
		"CondEqual":                    true, "CondNotEqual": true, "CondSignedLess": true, "CondSignedGreater": true, "CondSignedLessOrEqual": true, "CondSignedGreaterOrEqual": true,
		"CondUnsignedBelow": true, "CondUnsignedAboveOrEqual": true, "CondUnsignedBelowOrEqual": true, "CondUnsignedAbove": true,
		"RegRAX": true, "RegRBX": true, "RegRCX": true, "RegRDX": true,
		"RegRSI": true, "RegRDI": true, "RegRSP": true, "RegRBP": true,
		"RegR8": true, "RegR9": true, "RegR10": true, "RegR11": true,
		"RegR12": true, "RegR13": true, "RegR14": true, "RegR15": true,
		"RegX0": true, "RegX1": true, "RegX2": true, "RegX3": true, "RegX4": true, "RegX5": true,
	}
	// Map unexported tag constants to their exported equivalents
	scmTagMap := map[string]string{
		"tagNil": "scm.TagNil", "tagBool": "scm.TagBool", "tagInt": "scm.TagInt",
		"tagFloat": "scm.TagFloat", "tagString": "scm.TagString", "tagSymbol": "scm.TagSymbol",
		"tagSlice": "scm.TagSlice", "tagFastDict": "scm.TagFastDict", "tagDate": "scm.TagDate",
	}

	var result strings.Builder
	i := 0
	for i < len(code) {
		// Try to match an identifier starting at position i
		if isIdentStart(code[i]) {
			j := i + 1
			for j < len(code) && isIdentCont(code[j]) {
				j++
			}
			word := code[i:j]
			// Only prefix if not already preceded by a dot (e.g., not part of x.NewInt)
			preceded := i > 0 && code[i-1] == '.'
			if !preceded {
				if mapped, ok := scmTagMap[word]; ok {
					result.WriteString(mapped)
					i = j
					continue
				}
				if scmIdents[word] {
					result.WriteString("scm.")
				}
			}
			result.WriteString(word)
			i = j
		} else {
			result.WriteByte(code[i])
			i++
		}
	}
	return result.String()
}

func isIdentStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// computeRefCounts counts how many times each SSA value is referenced as an
// operand across all blocks of the function. Constants are excluded.
func computeRefCounts(fn *ssa.Function) map[string]int {
	counts := map[string]int{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, op := range instr.Operands(nil) {
				if *op == nil {
					continue
				}
				if _, isConst := (*op).(*ssa.Const); isConst {
					continue
				}
				counts[(*op).Name()]++
			}
		}
	}
	return counts
}

// computeDirectResultPayloads finds scalar SSA values whose NewBool/NewInt/
// NewFloat wrapper is returned directly. Those payloads can use the
// caller-selected Scmer aux register as their ALU target.
func computeDirectResultPayloads(fn *ssa.Function) map[string]string {
	result := map[string]string{}
	if fn == nil {
		return result
	}
	for _, instr := range fn.Blocks[0].Instrs {
		ret, ok := instr.(*ssa.Return)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		call, ok := ret.Results[0].(*ssa.Call)
		if !ok || len(call.Call.Args) != 1 {
			continue
		}
		callee := call.Call.StaticCallee()
		if callee == nil {
			continue
		}
		marker := ""
		switch callee.Name() {
		case "NewBool":
			marker = "_newbool"
		case "NewInt":
			marker = "_newint"
		case "NewFloat":
			marker = "_newfloat"
		}
		if marker != "" {
			producer, ok := call.Call.Args[0].(ssa.Instruction)
			if ok && producer.Block() == ret.Block() {
				result[call.Call.Args[0].Name()] = marker
			}
		}
	}
	return result
}

// emitAllocResultAwareReg chooses the caller's requested scalar payload
// register when it cannot alias a still-live operand. The final EmitMake*
// remains unconditional; its architecture-specific register move is a no-op
// for this placement and materializes other destination forms normally.
func (g *codeGen) emitAllocResultAwareReg(dstVar, targetVar, indent string, direct bool, excludes ...string) {
	if !direct {
		g.emit("%s%s := ctx.AllocRegExcept(%s)", indent, dstVar, strings.Join(excludes, ", "))
		return
	}
	condition := "result.Loc == LocRegPair"
	for _, exclude := range excludes {
		condition += " && result.Reg2 != " + exclude
	}
	g.emit("%svar %s Reg", indent, dstVar)
	g.emit("%sif %s {", indent, condition)
	g.emit("%s\t%s = result.Reg2", indent, dstVar)
	g.emit("%s\t%s = true", indent, targetVar)
	g.emit("%s} else {", indent)
	g.emit("%s\t%s = ctx.AllocRegExcept(%s)", indent, dstVar, strings.Join(excludes, ", "))
	g.emit("%s}", indent)
}

// useOperand decrements the refcount of an SSA value and emits FreeDesc when it reaches zero.
func (g *codeGen) useOperand(name string) {
	// Resolve aliases (from Convert no-ops): redirect to canonical SSA name
	if alias, ok := g.ssaAliases[name]; ok {
		name = alias
	}
	count, ok := g.refCounts[name]
	if !ok {
		return
	}
	count--
	g.refCounts[name] = count
	if count > 0 {
		return
	}
	gv, ok := g.vals[name]
	if !ok || !gv.isDesc || gv.goVar == "" {
		return
	}
	// Don't free markers or special values
	if gv.marker != "" {
		return
	}
	// Don't free field-cached values — their register is shared across
	// multiple SSA values and must stay alive for the duration.
	for _, cached := range g.fieldCache {
		if cached.goVar == gv.goVar {
			return
		}
	}
	g.emit("ctx.FreeDesc(&%s)", gv.goVar)
}

// keepAliveForMarker bumps the refcount of a marker argument (NewInt, NewFloat,
// NewBool) so that freeDeadOperands at the Call site doesn't free the argument's
// register. The register is later freed by the Return handler.
func (g *codeGen) keepAliveForMarker(arg ssa.Value) {
	if _, isConst := arg.(*ssa.Const); isConst {
		return
	}
	argName := arg.Name()
	if alias, ok := g.ssaAliases[argName]; ok {
		argName = alias
	}
	g.refCounts[argName]++
}

// freeDeadOperands decrements refcounts for all operands of an instruction
// and emits FreeDesc for any that reached zero.
func (g *codeGen) freeDeadOperands(instr ssa.Instruction) {
	// Skip IndexAddr: it doesn't emit code (just creates a marker).
	// The actual code is deferred to the UnOp handler; freeing here would
	// release registers before the code that uses them is emitted.
	if _, isIdx := instr.(*ssa.IndexAddr); isIdx {
		return
	}
	// Skip FieldAddr: same pattern — just creates a marker, code emitted in UnOp.
	if _, isFA := instr.(*ssa.FieldAddr); isFA {
		return
	}
	// Skip Phi: phi edge operands are consumed by emitPhiMov (which calls
	// useOperand), not here. Decrementing here would prematurely reduce
	// refcounts for back-edge values before they are produced, causing
	// destructive ALU ops to consume values still needed by phi stores.
	if _, isPhi := instr.(*ssa.Phi); isPhi {
		return
	}
	// Skip Return: any cleanup emitted after a return statement would be
	// unreachable in generated Go code and may break compilation.
	if _, isRet := instr.(*ssa.Return); isRet {
		return
	}
	for _, op := range instr.Operands(nil) {
		if *op == nil {
			continue
		}
		if _, isConst := (*op).(*ssa.Const); isConst {
			continue
		}
		g.useOperand((*op).Name())
	}
}

// resetAllPhiDescsToStack restores phi descriptors to their canonical
// stack-backed locations at BB entry. This prevents stale descriptor state
// from one emitted BB affecting compile-time lowering decisions in another BB.
func (g *codeGen) resetAllPhiDescsToStack() {
	phiNames := make([]string, 0, len(g.phiRegs))
	for phiName := range g.phiRegs {
		phiNames = append(phiNames, phiName)
	}
	sort.Slice(phiNames, func(i, j int) bool {
		left, leftErr := strconv.Atoi(g.phiRegs[phiNames[i]])
		right, rightErr := strconv.Atoi(g.phiRegs[phiNames[j]])
		if leftErr == nil && rightErr == nil && left != right {
			return left < right
		}
		return phiNames[i] < phiNames[j]
	})
	for _, phiName := range phiNames {
		phiOff := g.phiRegs[phiName]
		gv, ok := g.vals[phiName]
		if !ok || !gv.isDesc {
			// Phi descriptors are declared lazily when lowering the phi
			// instruction itself. Skip unseen phis here to avoid generating
			// unused temporary declarations in functions where some phi values
			// are not materialized on a given path.
			continue
		}
		phiTag := g.phiTypeTag[phiName]
		if phiTag == "" {
			phiTag = "JITTypeUnknown"
		}
		stackOff := "int32(" + phiOff + ")"
		if g.phiFrameFixup != "" && !g.storageMode {
			stackOff = "int32(" + g.phiFrameFixup + ")+" + stackOff
		}
		if g.phiTriple[phiName] {
			g.emit("%s = JITValueDesc{Loc: LocStackTriple, Type: %s, StackOff: %s}", gv.goVar, phiTag, stackOff)
		} else if g.phiPair[phiName] {
			g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: %s, StackOff: %s}", gv.goVar, phiTag, stackOff)
		} else {
			g.emit("%s = JITValueDesc{Loc: LocStack, Type: %s, StackOff: %s}", gv.goVar, phiTag, stackOff)
		}
	}
}

// applyPhiStateOverlay sets block-local phi descriptors from ps.PhiValues when
// a specialized renderer call provides overlays for this block.
func (g *codeGen) applyPhiStateOverlay(bbIdx int) {
	phiDescVars := map[string]bool{}
	for phiName := range g.phiRegs {
		gv, ok := g.vals[phiName]
		if !ok || !gv.isDesc || gv.goVar == "" {
			continue
		}
		phiDescVars[gv.goVar] = true
	}

	for _, ov := range g.allClosureDescVars() {
		idx, err := parseDescNum(ov)
		if err != nil {
			continue
		}
		if phiDescVars[ov] {
			g.emit("if !ps.General && len(ps.OverlayValues) > %d && ps.OverlayValues[%d].Loc != LocNone {", idx, idx)
		} else {
			g.emit("if len(ps.OverlayValues) > %d && ps.OverlayValues[%d].Loc != LocNone {", idx, idx)
		}
		g.emit("\t%s = ps.OverlayValues[%d]", ov, idx)
		g.emit("}")
	}

	phis := g.blockPhis(bbIdx)
	for phiIdx, phi := range phis {
		phiOff := g.phiSlotOffExpr(bbIdx, phiIdx)
		stackOff := "int32(" + phiOff + ")"
		if g.phiFrameFixup != "" && !g.storageMode {
			stackOff = "int32(" + g.phiFrameFixup + ")+" + stackOff
		}
		gv, ok := g.vals[phi.Name()]
		if !ok || !gv.isDesc {
			dv := g.allocDesc()
			phiTag := g.phiTypeTag[phi.Name()]
			if phiTag == "" {
				phiTag = "JITTypeUnknown"
			}
			if g.phiTriple[phi.Name()] {
				g.emit("%s := JITValueDesc{Loc: LocStackTriple, Type: %s, StackOff: %s}", dv, phiTag, stackOff)
			} else if g.phiPair[phi.Name()] {
				g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: %s, StackOff: %s}", dv, phiTag, stackOff)
			} else {
				g.emit("%s := JITValueDesc{Loc: LocStack, Type: %s, StackOff: %s}", dv, phiTag, stackOff)
			}
			gv = genVal{goVar: dv, isDesc: true}
			if g.phiTriple[phi.Name()] {
				gv.marker = "_slice"
				gv.pinAcrossBlock = true
			}
			g.vals[phi.Name()] = gv
		}
		g.emit("if !ps.General && len(ps.PhiValues) > %d && ps.PhiValues[%d].Loc != LocNone {", phiIdx, phiIdx)
		g.emit("\t%s = ps.PhiValues[%d]", gv.goVar, phiIdx)
		g.emit("}")
	}
}

// ssaValueUsesRemaining returns how many remaining uses an SSA value has.
func (g *codeGen) ssaValueUsesRemaining(name string) int {
	if alias, ok := g.ssaAliases[name]; ok {
		name = alias
	}
	if count, ok := g.refCounts[name]; ok {
		return count
	}
	return 0
}

func (g *codeGen) stabilizeCrossBlockValue(instr ssa.Instruction) {
	value, ok := instr.(ssa.Value)
	if !ok || value.Referrers() == nil || instr.Block() == nil {
		return
	}
	crossesBlock := false
	for _, ref := range *value.Referrers() {
		if ref.Block() != nil && ref.Block() != instr.Block() {
			crossesBlock = true
			break
		}
	}
	if !crossesBlock {
		return
	}
	generated, ok := g.vals[value.Name()]
	if !ok || !generated.isDesc || generated.goVar == "" {
		return
	}
	if generated.marker == "_dynamic_variadic_element" {
		panic(fmt.Sprintf("dynamic variadic element crosses a control-flow edge: %s", value))
	}
	g.emit("ctx.StabilizeDescForControlFlow(&%s)", generated.goVar)
}

func ssaValueCrossesControlFlow(value ssa.Value) bool {
	instr, ok := value.(ssa.Instruction)
	if !ok || instr.Block() == nil || value.Referrers() == nil {
		return false
	}
	for _, ref := range *value.Referrers() {
		if ref.Block() != nil && ref.Block() != instr.Block() {
			return true
		}
	}
	return false
}

// usedByOutgoingPhi reports whether SSA value `name` appears as a phi edge
// operand on any outgoing edge of the current basic block.
func (g *codeGen) usedByOutgoingPhi(name string) bool {
	if g.fn == nil || g.curBlock < 0 || g.curBlock >= len(g.fn.Blocks) {
		return false
	}
	if alias, ok := g.ssaAliases[name]; ok {
		name = alias
	}
	cur := g.fn.Blocks[g.curBlock]
	for _, succ := range cur.Succs {
		for _, instr := range succ.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				break
			}
			for i, pred := range succ.Preds {
				if pred.Index != g.curBlock {
					continue
				}
				edge := phi.Edges[i]
				if edge == nil {
					continue
				}
				if c, isConst := edge.(*ssa.Const); isConst && c != nil {
					continue
				}
				edgeName := edge.Name()
				if alias, ok := g.ssaAliases[edgeName]; ok {
					edgeName = alias
				}
				if edgeName == name {
					return true
				}
			}
		}
	}
	return false
}

// flushPhiProtections emits UnprotectReg for all phi-loaded registers
// collected during the current block's Phi instructions.
// Phi registers are protected during loading to prevent mutual eviction and
// to keep them live until the block body starts.
func (g *codeGen) flushPhiProtections() {
	for _, rv := range g.phiProtectedRegVars {
		g.emit("ctx.UnprotectReg(%s)", rv)
	}
	g.phiProtectedRegVars = nil
}

// emitGoCallScmer1 emits a 1-arg Go call returning Scmer and copies the
// returned pair into fresh registers so operand cleanup cannot free it via
// register aliasing with call inputs.
func (g *codeGen) emitInstrLegacy(instr ssa.Instruction) {
	// When we encounter the first non-Phi instruction in a block,
	// unprotect all phi-loaded registers. The protection was only needed
	// to prevent mutual spilling during the phi load sequence.
	if _, isPhi := instr.(*ssa.Phi); !isPhi && len(g.phiProtectedRegVars) > 0 {
		g.flushPhiProtections()
	}

	val, isVal := instr.(ssa.Value)
	name := ""
	if isVal {
		name = val.Name()
	}

	switch v := instr.(type) {
	case *ssa.Extract:
		tuple := g.lookup(v.Tuple)
		if v.Index < 0 || v.Index >= len(tuple.tuple) {
			panic(fmt.Sprintf("Extract index %d outside tuple of %d values", v.Index, len(tuple.tuple)))
		}
		g.vals[name] = tuple.tuple[v.Index]
	case *ssa.Lookup:
		container := g.lookup(v.X)
		fieldExpr := strings.TrimPrefix(container.marker, "_globalvalue:")
		mapType, isMap := v.X.Type().Underlying().(*types.Map)
		key, isConst := v.Index.(*ssa.Const)
		keyExpr, keyOK := g.staticConstExpr(key, func() types.Type {
			if isMap {
				return mapType.Key()
			}
			return nil
		}())
		if fieldExpr == container.marker || !isMap || !isConst || !keyOK || !isScmerType(mapType.Elem()) {
			if isMap && strings.HasPrefix(container.marker, "_globalvalue:") {
				globalMap := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func() %s { return %s }), nil, 1)", globalMap, g.sourceTypeExpr(v.X.Type()), strings.TrimPrefix(container.marker, "_globalvalue:"))
				container = genVal{goVar: globalMap, isDesc: true, marker: "_gomap"}
			}
			if !isMap || !container.isDesc || container.goVar == "" {
				panic(fmt.Sprintf("Lookup: %s (map=%t marker=%q desc=%t var=%q)", v, isMap, container.marker, container.isDesc, container.goVar))
			}
			keyValue := g.resolveValue(v.Index)
			valueWords := goCallWordCount(mapType.Elem())
			if valueWords < 1 || valueWords > 3 {
				panic(fmt.Sprintf("Lookup result has unsupported ABI shape: %s", v))
			}
			mapExpr := g.sourceTypeExpr(v.X.Type())
			keyExpr := g.sourceTypeExpr(mapType.Key())
			valueExpr := g.sourceTypeExpr(mapType.Elem())
			valueMarker := ""
			if isScmerType(mapType.Elem()) {
				valueMarker = "_scmer_struct"
			} else if _, ok := mapType.Elem().Underlying().(*types.Slice); ok {
				valueMarker = "_slice"
			} else if basic, ok := mapType.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
				valueMarker = "_gostring"
			}
			if v.CommaOk {
				results := g.allocTemp("lookupResults")
				valueDesc := g.allocDesc()
				okDesc := g.allocDesc()
				g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(func(m %s, k %s) (%s, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{%s, %s}, []uint8{%d, 1}, []uint8{%d, 0})", results, mapExpr, keyExpr, valueExpr, container.goVar, keyValue.goVar, valueWords, goCallPointerMask(mapType.Elem()))
				g.emit("%s := %s[0]", valueDesc, results)
				g.emit("%s := %s[1]", okDesc, results)
				g.emit("ctx.EmitAndRegImm32(%s.Reg, 1)", okDesc)
				g.emit("%s.Type = tagBool", okDesc)
				g.vals[name] = genVal{tuple: []genVal{{goVar: valueDesc, isDesc: true, marker: valueMarker}, {goVar: okDesc, isDesc: true}}}
			} else {
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(m %s, k %s) %s { return m[k] }), []JITValueDesc{%s, %s}, %d)", dv, mapExpr, keyExpr, valueExpr, container.goVar, keyValue.goVar, valueWords)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: valueMarker}
			}
			break
		}
		valueVar := g.allocTemp("globalLookup")
		valueDesc := g.allocDesc()
		if v.CommaOk {
			okVar := g.allocTemp("globalLookupOK")
			okDesc := g.allocDesc()
			g.emit("%s, %s := %s[%s]", valueVar, okVar, fieldExpr, keyExpr)
			g.emit("ctx.TrackImm(%s)", valueVar)
			g.emit("%s := JITValueDesc{Loc: LocImm, Type: %s.GetTag(), Imm: %s, Rooted: true}", valueDesc, valueVar, valueVar)
			g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s), NoHeapPointer: true}", okDesc, okVar)
			g.vals[name] = genVal{tuple: []genVal{{goVar: valueDesc, isDesc: true}, {goVar: okDesc, isDesc: true}}}
		} else {
			g.emit("%s := %s[%s]", valueVar, fieldExpr, keyExpr)
			g.emit("ctx.TrackImm(%s)", valueVar)
			g.emit("%s := JITValueDesc{Loc: LocImm, Type: %s.GetTag(), Imm: %s, Rooted: true}", valueDesc, valueVar, valueVar)
			g.vals[name] = genVal{goVar: valueDesc, isDesc: true, marker: "_knownimm"}
		}

	case *ssa.Index:
		src := g.resolveValue(v.X)
		_, sourceIsSlice := v.X.Type().Underlying().(*types.Slice)
		sourceBasic, sourceIsBasic := v.X.Type().Underlying().(*types.Basic)
		sourceIsString := sourceIsBasic && sourceBasic.Kind() == types.String
		if src.goVar == "" || !src.isDesc || (!sourceIsSlice && !sourceIsString) {
			panic(fmt.Sprintf("Index: %s (source=%s marker=%q desc=%t goVar=%q)", v, v.X.Name(), src.marker, src.isDesc, src.goVar))
		}
		elem := func() types.Type {
			switch aggregate := v.X.Type().Underlying().(type) {
			case *types.Slice:
				return aggregate.Elem()
			case *types.Array:
				return aggregate.Elem()
			case *types.Basic:
				if aggregate.Kind() == types.String {
					return types.Typ[types.Uint8]
				}
			}
			return nil
		}()
		if elem == nil || !isByteType(elem) {
			panic(fmt.Sprintf("Index of unsupported element type: %s", v))
		}
		idx := g.resolveValue(v.Index)
		address := g.allocDesc()
		value := g.allocDesc()
		valueReg := g.allocReg()
		if sourceIsString {
			g.emit("ctx.EnsureGoStringHeader(&%s)", src.goVar)
		}
		g.emit("%s := ctx.EmitSliceElementAddress(&%s, &%s, 1)", address, src.goVar, idx.goVar)
		g.emit("ctx.EnsureDesc(&%s)", address)
		g.emit("%s := ctx.AllocRegExcept(%s.Reg)", valueReg, address)
		g.emit("ctx.EmitMovRegMemB(%s, %s.Reg, 0)", valueReg, address)
		g.emit("ctx.FreeDesc(&%s)", address)
		g.emit("%s := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s, NoHeapPointer: true}", value, valueReg)
		g.emit("ctx.BindReg(%s, &%s)", valueReg, value)
		g.vals[name] = genVal{goVar: value, isDesc: true}

	case *ssa.IndexAddr:
		if v.X.Name() == g.paramName {
			if c, ok := v.Index.(*ssa.Const); ok {
				idx, ok := constInt64Value(c.Value)
				if !ok {
					panic(fmt.Sprintf("IndexAddr expects int constant index: %s", c))
				}
				g.vals[name] = genVal{argIdx: int(idx)}
			} else {
				// Variable index (e.g. phi loop counter)
				idxVal := g.resolveValue(v.Index)
				g.vals[name] = genVal{argIdx: -1, argIdxVar: idxVal.goVar}
			}
		} else if src := g.vals[v.X.Name()]; src.marker == "_variadic_args" {
			if c, ok := v.Index.(*ssa.Const); ok {
				idx, ok := constInt64Value(c.Value)
				if !ok {
					panic(fmt.Sprintf("IndexAddr expects int constant index: %s", c))
				}
				g.vals[name] = genVal{argIdx: src.variadicOffset + int(idx)}
			} else {
				idxVal := g.resolveValue(v.Index)
				g.vals[name] = genVal{argIdx: -1, argIdxVar: idxVal.goVar, argBase: src.variadicOffset}
			}
		} else if src := g.vals[v.X.Name()]; src.marker == "_goarrayptr" {
			idx := g.resolveValue(v.Index)
			arrayType := v.X.Type().Underlying().(*types.Pointer).Elem()
			elemType := v.Type().Underlying().(*types.Pointer).Elem()
			g.vals[name] = genVal{goVar: src.goVar, argIdxVar: idx.goVar, marker: "_goarrayelem", aggregateType: arrayType, fieldType: elemType}
		} else if _, isGlobal := v.X.(*ssa.Global); isGlobal {
			// IndexAddr on a global array/slice (e.g. &pow10f[idx])
			globalName, resolved := g.globalSourceExpr(v.X.(*ssa.Global))
			if !resolved {
				panic(fmt.Sprintf("IndexAddr on inaccessible global: %s", v))
			}
			idxVal := g.resolveValue(v.Index)
			elemType := v.Type().Underlying().(*types.Pointer).Elem().Underlying()
			elemSize := elemSizeOf(elemType)
			idxSSAName := ""
			if _, isConst := v.Index.(*ssa.Const); !isConst {
				idxSSAName = v.Index.Name()
			}
			g.vals[name] = genVal{argIdx: -1, argIdxVar: idxVal.goVar,
				marker: fmt.Sprintf("_globaladdr:%d:%s", elemSize, globalName), deferredIndexSSA: idxSSAName}
		} else {
			// IndexAddr on a local slice (e.g. from Slice() or FieldAddr)
			src := g.vals[v.X.Name()]
			if src.marker == "_stackslice" {
				idx, ok := v.Index.(*ssa.Const)
				if !ok {
					panic(fmt.Sprintf("dynamic IndexAddr on local stack slice: %s", v))
				}
				idxValue, ok := constInt64Value(idx.Value)
				if !ok || idxValue < 0 || idxValue >= int64(src.stackLen) {
					panic(fmt.Sprintf("invalid IndexAddr on local stack slice: %s", v))
				}
				elemType := v.Type().Underlying().(*types.Pointer).Elem()
				elemSize := types.SizesFor("gc", "amd64").Sizeof(elemType)
				g.vals[name] = genVal{
					marker:     fmt.Sprintf("_stackaddr:%d", elemSize),
					offsetExpr: fmt.Sprintf("int32(%s)+int32(%d)", src.stackBase, idxValue*elemSize),
				}
			} else if strings.HasPrefix(src.marker, "_stackarray:") {
				parts := strings.Split(src.marker, ":")
				if len(parts) != 3 {
					panic(fmt.Sprintf("malformed stack array marker: %q", src.marker))
				}
				elemSize, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					panic(fmt.Sprintf("malformed stack array element size: %q", src.marker))
				}
				idx, ok := v.Index.(*ssa.Const)
				if !ok {
					panic(fmt.Sprintf("dynamic IndexAddr on local stack array: %s", v))
				}
				idxValue, ok := constInt64Value(idx.Value)
				if !ok {
					panic(fmt.Sprintf("non-integer IndexAddr on local stack array: %s", v))
				}
				g.vals[name] = genVal{
					marker:     fmt.Sprintf("_stackaddr:%d", elemSize),
					offsetExpr: fmt.Sprintf("int32(%s)+int32(%d)", src.goVar, idxValue*elemSize),
				}
			} else if strings.HasPrefix(src.marker, "_fieldaddr:array:") {
				// Direct indexing on receiver array field address, e.g. &s.thresholds[i].
				idxVal := g.resolveValue(v.Index)
				elemType := v.Type().Underlying().(*types.Pointer).Elem().Underlying()
				elemSize := elemSizeOf(elemType)
				baseDesc := g.allocDesc()
				baseReg := g.allocReg()
				g.emit("var %s JITValueDesc", baseDesc)
				g.emit("%s := ctx.AllocReg()", baseReg)
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tctx.EmitMovRegImm64(%s, uint64(uintptr(thisptr.Imm.Int()) + %s))", baseReg, src.offsetExpr)
				g.emit("} else {")
				g.emit("\tctx.EmitMovRegReg(%s, thisptr.Reg)", baseReg)
				g.emit("\tctx.EmitAddRegImm32(%s, int32(%s))", baseReg, src.offsetExpr)
				g.emit("}")
				g.emit("%s = JITValueDesc{Loc: LocReg, Reg: %s}", baseDesc, baseReg)
				idxSSAName := ""
				if _, isConst := v.Index.(*ssa.Const); !isConst {
					idxSSAName = v.Index.Name()
				}
				g.vals[name] = genVal{argIdx: -1, argIdxVar: idxVal.goVar,
					marker: fmt.Sprintf("_sliceaddr:%d:%s", elemSize, baseDesc), deferredIndexSSA: idxSSAName}
			} else if src.marker == "_slice" {
				// src.goVar is a JITValueDesc with Reg=data_ptr
				idxVal := g.resolveValue(v.Index)
				// Determine element size from pointed-to type
				elemType := v.Type().Underlying().(*types.Pointer).Elem().Underlying()
				elemSize := elemSizeOf(elemType)
				idxSSAName := ""
				if _, isConst := v.Index.(*ssa.Const); !isConst {
					idxSSAName = v.Index.Name()
				}
				g.vals[name] = genVal{argIdx: -1, argIdxVar: idxVal.goVar,
					marker: fmt.Sprintf("_sliceaddr:%d:%s", elemSize, src.goVar), deferredIndexSSA: idxSSAName}
			} else {
				panic(fmt.Sprintf("IndexAddr on non-parameter: %s (x=%s marker=%q isDesc=%v goVar=%s)", v, v.X.Name(), src.marker, src.isDesc, src.goVar))
			}
		}

	case *ssa.FieldAddr:
		// &s.field — struct field address (direct or nested)
		src := g.vals[v.X.Name()]
		globalExpr := ""
		if global, ok := v.X.(*ssa.Global); ok {
			var resolved bool
			globalExpr, resolved = g.globalSourceExpr(global)
			if !resolved {
				panic(fmt.Sprintf("FieldAddr on unresolved global: %s", v))
			}
		}

		// Extract field info from SSA types
		ptrType := v.X.Type().Underlying().(*types.Pointer)
		structType := ptrType.Elem().Underlying().(*types.Struct)
		field := structType.Field(v.Field)
		if field.Pkg() != nil && field.Pkg().Path() != g.topLevelPkgPath && !field.Exported() {
			panic(fmt.Sprintf("FieldAddr cannot name unexported external field: %s", v))
		}
		fieldName := field.Name()
		fieldType := field.Type().Underlying()

		// Determine the offset expression and struct type name for this field
		var offsetExpr string
		var cacheKey string
		var isImmutable bool
		localFieldAddr := false

		if globalExpr != "" {
			// Package globals are resolved by the generated emitter itself. This
			// preserves the symbolic field expression for constant map lookups.
		} else if (src.argIdx >= 0 || src.argIdxVar != "") && (fieldName == "ptr" || fieldName == "aux") {
			if src.argIdx < 0 {
				panic(fmt.Sprintf("dynamic variadic Scmer field address: %s", v))
			}
			base := g.allocDesc()
			g.emit("%s := args[%d]", base, src.argIdx)
			g.emit("%s.ID = 0", base)
			g.vals[name] = genVal{marker: "_descfield:" + fieldName + ":" + base}
			break
		} else if src.marker == "_storage_recv" {
			// Direct field of receiver
			tag := structType.Tag(v.Field)
			isImmutable = strings.Contains(tag, `jit:"immutable-after-finish"`)
			offsetExpr = fmt.Sprintf("unsafe.Offsetof((*%s)(nil).%s)", g.typeName, fieldName)
			cacheKey = fieldName
		} else if strings.HasPrefix(src.marker, "_fieldaddr:") || strings.HasPrefix(src.marker, "_fieldconst:") {
			// Nested field access: src is a pointer to a sub-struct within the top-level struct.
			// Cascade the offset: parent offset + inner field offset.
			// Compute inner field offset at jitgen time (handles unexported fields from external packages).
			sizes := types.SizesFor("gc", "amd64")
			offsets := sizes.Offsetsof(fieldVarsOf(structType))
			innerOffset := offsets[v.Field]
			tag := structType.Tag(v.Field)
			isImmutable = strings.Contains(tag, `jit:"immutable-after-finish"`)
			offsetExpr = src.offsetExpr + fmt.Sprintf(" + %d", innerOffset)
			// Compound cache key from parent marker's field name
			parts := strings.SplitN(src.marker, ":", 3)
			parentField := parts[2]
			cacheKey = parentField + "." + fieldName
		} else if src.isDesc && (fieldName == "ptr" || fieldName == "aux") {
			// Descriptor-backed Scmer receiver (e.g. inlined methods with signature
			// like func (s Scmer) ...). Scmer is already split in JITValueDesc as
			// ptr+aux, so FieldAddr must reference descriptor halves, not thisptr.
			g.vals[name] = genVal{marker: "_descfield:" + fieldName + ":" + src.goVar}
			break
		} else if src.isDesc {
			// FieldAddr on a local pointer descriptor (non-receiver), e.g.
			// fd := a.FastDict(); &fd.Pairs
			sizes := types.SizesFor("gc", "amd64")
			offsets := sizes.Offsetsof(fieldVarsOf(structType))
			innerOffset := offsets[v.Field]
			offsetExpr = fmt.Sprintf("%d", innerOffset)
			localFieldAddr = true
		} else {
			panic(fmt.Sprintf("FieldAddr on non-receiver: %s", v))
		}

		// Determine field size for the load instruction
		var sizeStr string
		var goTypeName string
		// Scmer is a two-word struct (ptr + aux), must be loaded as a pair.
		if isScmerType(field.Type()) {
			sizeStr = "scmer"
		} else {
			switch t := fieldType.(type) {
			case *types.Basic:
				goTypeName = t.Name()
				switch t.Kind() {
				case types.String:
					// Go strings are two words: data pointer + length.
					sizeStr = "slice"
				case types.Bool, types.Uint8, types.Int8:
					sizeStr = "1"
				case types.Uint16, types.Int16:
					sizeStr = "2"
				case types.Uint32, types.Int32:
					sizeStr = "4"
				default:
					sizeStr = "8"
				}
			case *types.Slice:
				sizeStr = "slice"
			case *types.Array:
				// Keep array as addressable aggregate; indexed loads are lowered via IndexAddr.
				sizeStr = "array"
			default:
				sizeStr = "8"
			}
		}

		// Create marker with offsetExpr
		if globalExpr != "" {
			g.vals[name] = genVal{marker: "_globalfield:" + globalExpr + "." + fieldName}
		} else if localFieldAddr {
			g.vals[name] = genVal{
				goVar:           src.goVar,
				marker:          "_fieldaddrlocal:" + sizeStr,
				offsetExpr:      offsetExpr,
				deferredBaseSSA: v.X.Name(),
				fieldBaseType:   ptrType.Elem(),
				fieldType:       field.Type(),
				fieldName:       fieldName,
			}
		} else if isImmutable && sizeStr == "scmer" {
			g.vals[name] = genVal{marker: "_fieldconst:scmer:" + cacheKey, offsetExpr: offsetExpr}
		} else if isImmutable && sizeStr == "slice" {
			g.vals[name] = genVal{marker: "_fieldconst:slice:" + cacheKey, offsetExpr: offsetExpr}
		} else if isImmutable && goTypeName != "" {
			g.vals[name] = genVal{marker: "_fieldconst:" + goTypeName + ":" + cacheKey, offsetExpr: offsetExpr}
		} else {
			g.vals[name] = genVal{marker: "_fieldaddr:" + sizeStr + ":" + cacheKey, offsetExpr: offsetExpr}
		}

	case *ssa.UnOp:
		if v.Op == token.SUB {
			src := g.resolveValue(v.X)
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\tif %s.Type == tagFloat {", src.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-%s.Imm.Float())}", dv, src.goVar)
			g.emit("\t} else {")
			g.emit("\t\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-%s.Imm.Int())}", dv, src.goVar)
			g.emit("\t}")
			g.emit("} else {")
			g.emit("\tif %s.Type == tagFloat {", src.goVar)
			negReg := g.allocReg()
			g.emit("\t\t%s := ctx.AllocRegExcept(%s.Reg)", negReg, src.goVar)
			g.emit("\t\tctx.EmitMovRegImm64(%s, 0)", negReg)
			g.emit("\t\tctx.EmitSubFloat64(%s, %s.Reg)", negReg, src.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s}", dv, negReg)
			g.emit("\t} else {")
			negIntReg := g.allocReg()
			g.emit("\t\t%s := ctx.AllocRegExcept(%s.Reg)", negIntReg, src.goVar)
			g.emit("\t\tctx.EmitMovRegImm64(%s, 0)", negIntReg)
			g.emit("\t\tctx.EmitSubInt64(%s, %s.Reg)", negIntReg, src.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, negIntReg)
			g.emit("\t}")
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.NOT {
			src := g.resolveValue(v.X)
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!%s.Imm.Bool())}", dv, src.goVar)
			g.emit("} else {")
			g.emit("\tnegReg := ctx.AllocReg()")
			g.emit("\tif %s.Loc == LocRegPair {", src.goVar)
			g.emit("\t\tctx.EmitMovRegReg(negReg, %s.Reg2)", src.goVar)
			g.emit("\t\tctx.EmitAndRegImm32(negReg, 1)")
			g.emit("\t\tctx.EmitCmpRegImm32(negReg, 0)")
			g.emit("\t\tctx.EmitSetcc(negReg, CondEqual)")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}", dv)
			g.emit("\t} else if %s.Loc == LocReg {", src.goVar)
			g.emit("\t\tctx.EmitMovRegReg(negReg, %s.Reg)", src.goVar)
			g.emit("\t\tctx.EmitAndRegImm32(negReg, 1)")
			g.emit("\t\tctx.EmitCmpRegImm32(negReg, 0)")
			g.emit("\t\tctx.EmitSetcc(negReg, CondEqual)")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}", dv)
			g.emit("\t} else {")
			g.emit("\t\tpanic(\"UnOp ! unsupported source location\")")
			g.emit("\t}")
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.MUL {
			src := g.vals[v.X.Name()]
			if global, ok := v.X.(*ssa.Global); ok {
				expr, resolved := g.globalSourceExpr(global)
				if !resolved {
					panic(fmt.Sprintf("unresolved global dereference: %s", v))
				}
				elemType := v.Type()
				words := goCallWordCount(elemType)
				if words == 0 && types.SizesFor("gc", "amd64").Sizeof(elemType) == 0 {
					g.vals[name] = genVal{marker: "_gozero", aggregateType: elemType}
					break
				}
				if words < 1 || words > 3 {
					panic(fmt.Sprintf("global dereference has unsupported ABI shape: %s", v))
				}
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func() %s { return %s }), nil, %d)", dv, g.sourceTypeExpr(elemType), expr, words)
				marker := ""
				switch elemType.Underlying().(type) {
				case *types.Slice:
					marker = "_slice"
				case *types.Map:
					marker = "_gomap"
				case *types.Signature:
					marker = "_gofunc_variadic"
				}
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: words > 1}
			} else if src.cellName != "" && src.marker != "_alloc" {
				if src.isDesc {
					copyDesc := g.allocDesc()
					g.emit("%s := %s", copyDesc, src.goVar)
					g.emit("_ = %s", copyDesc)
					src.goVar = copyDesc
				}
				src.cellName = ""
				src.cellScope = 0
				g.vals[name] = src
			} else if src.marker == "_goptr" {
				pointerType, ok := v.X.Type().Underlying().(*types.Pointer)
				if !ok {
					panic(fmt.Sprintf("aggregate pointer has non-pointer SSA type: %s", v))
				}
				switch pointerType.Elem().Underlying().(type) {
				case *types.Struct, *types.Array:
					g.vals[name] = genVal{goVar: src.goVar, isDesc: true, marker: "_aggregate_ptr", aggregateType: pointerType.Elem(), pinAcrossBlock: true}
				default:
					words := goCallWordCount(pointerType.Elem())
					if words < 1 || words > 3 {
						panic(fmt.Sprintf("unsupported pointer dereference ABI shape: %s", v))
					}
					dv := g.allocDesc()
					typeExpr := g.sourceTypeExpr(pointerType.Elem())
					g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(value *%s) %s { return *value }), []JITValueDesc{%s}, %d)", dv, typeExpr, typeExpr, src.goVar, words)
					marker := ""
					switch pointerType.Elem().Underlying().(type) {
					case *types.Slice:
						marker = "_slice"
					case *types.Map:
						marker = "_gomap"
					case *types.Signature:
						marker = "_gofunc_variadic"
					}
					g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: words > 1}
				}
			} else if strings.HasPrefix(src.marker, "_globalfield:") {
				g.vals[name] = genVal{marker: "_globalvalue:" + strings.TrimPrefix(src.marker, "_globalfield:")}
			} else if strings.HasPrefix(src.marker, "_descfield:") {
				// Deref of descriptor-backed Scmer field address.
				// marker format: "_descfield:<ptr|aux>:<descVar>"
				parts := strings.SplitN(src.marker, ":", 3)
				fieldName := parts[1]
				base := parts[2]
				dv := g.allocDesc()
				g.emit("var %s JITValueDesc", dv)
				g.emit("ctx.EnsureDesc(&%s)", base)
				g.emit("if %s.Loc == LocImm {", base)
				if fieldName == "ptr" {
					g.emit("\tptrWord, _ := %s.Imm.RawWords()", base)
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}", dv)
				} else {
					g.emit("\t_, auxWord := %s.Imm.RawWords()", base)
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}", dv)
				}
				g.emit("} else {")
				g.emit("\tif %s.Loc != LocRegPair { panic(\"jitgen: desc field base is not LocRegPair\") }", base)
				rv := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv)
				if fieldName == "ptr" {
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", rv, base)
				} else {
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg2)", rv, base)
				}
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, rv)
				g.emit("}")
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else if strings.HasPrefix(src.marker, "_fieldaddrlocal:") {
				// Deref of FieldAddr on a local pointer descriptor (non-receiver).
				parts := strings.SplitN(src.marker, ":", 2) // "_fieldaddrlocal", size
				sizeStr := parts[1]
				base := src.goVar
				dv := g.allocDesc()
				g.emit("var %s JITValueDesc", dv)
				g.emit("ctx.EnsureDesc(&%s)", base)
				g.emit("if %s.Loc == LocImm {", base)
				g.emit("\tfieldAddr := uintptr(%s.Imm.Int()) + %s", base, src.offsetExpr)
				switch sizeStr {
				case "scmer":
					ptrReg := g.allocReg()
					auxReg := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s)", auxReg, ptrReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", ptrReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr+8)", auxReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
				case "1":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem8(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem16(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem32(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					capReg := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", ptrReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr+8)", lenReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr+16)", capReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv, ptrReg, lenReg, capReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\tctx.EmitMovRegImm64(%s, uint64(fieldAddr))", ptrReg)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg)
				}
				g.emit("} else {")
				g.emit("\toff := int32(%s)", src.offsetExpr)
				g.emit("\tbaseReg := %s.Reg", base)
				switch sizeStr {
				case "scmer":
					ptrReg := g.allocReg()
					auxReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(baseReg, %s)", auxReg, ptrReg)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off)", ptrReg)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off+8)", auxReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
				case "1":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.EmitMovRegMemB(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.EmitMovRegMemW(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.EmitMovRegMemL(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					capReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(baseReg, %s)", lenReg, ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(baseReg, %s, %s)", capReg, ptrReg, lenReg)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off)", ptrReg)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off+8)", lenReg)
					g.emit("\tctx.EmitMovRegMem(%s, baseReg, off+16)", capReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv, ptrReg, lenReg, capReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", ptrReg)
					g.emit("\tctx.EmitMovRegReg(%s, baseReg)", ptrReg)
					g.emit("\tctx.EmitAddRegImm32(%s, off)", ptrReg)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg)
				}
				g.emit("}")
				if sizeStr == "slice" || sizeStr == "array" {
					g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice"}
				} else {
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				}
				if src.deferredBaseSSA != "" {
					g.useOperand(src.deferredBaseSSA)
				}
			} else if strings.HasPrefix(src.marker, "_fieldconst:") {
				// Deref of immutable FieldAddr → constant-fold (LocImm thisptr) or runtime load (LocReg thisptr).
				parts := strings.SplitN(src.marker, ":", 3) // "_fieldconst", goType, fieldName
				goType := parts[1]
				fieldName := parts[2]

				if goType == "slice" {
					// Immutable slice/string header: keep data pointer in a register.
					// Do NOT encode raw pointers as NewInt immediates; they are plain
					// addresses, not tagged integers.
					cacheKey := fieldName
					if cached, ok := g.fieldCache[cacheKey]; ok {
						g.vals[name] = cached
						break
					}
					dv := g.allocDesc()
					ptrReg2 := g.allocReg()
					g.emit("var %s JITValueDesc", dv)
					g.emit("%s := ctx.AllocReg()", ptrReg2)
					g.emit("if thisptr.Loc == LocImm {")
					// Constant receiver: fold load address, but still materialize pointer in a GPR.
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\tdataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))")
					g.emit("\tsliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))")
					g.emit("\tctx.EmitMovRegImm64(%s, uint64(dataPtr))", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s, StackOff: int32(sliceLen)}", dv, ptrReg2)
					g.emit("} else {")
					// Register receiver: load data pointer from field.
					g.emit("\toff := int32(%s)", src.offsetExpr)
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg2)
					g.emit("}")
					g.emit("ctx.BindReg(%s, &%s)", ptrReg2, dv)
					gv := genVal{goVar: dv, isDesc: true, marker: "_slice"}
					g.vals[name] = gv
					g.fieldCache[cacheKey] = gv
					break
				}

				if goType == "scmer" {
					// Immutable Scmer field: at JIT-compile-time, read the two-word
					// value from the struct and return it as LocImm (zero code emitted).
					// This enables maximum constant folding downstream.
					cacheKey := fieldName
					if cached, ok := g.fieldCache[cacheKey]; ok {
						g.vals[name] = cached
						break
					}
					dv := g.allocDesc()
					g.emit("var %s JITValueDesc", dv)
					g.emit("if thisptr.Loc == LocImm {")
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\tval := *(*Scmer)(unsafe.Pointer(fieldAddr))")
					g.emit("\tctx.TrackImm(val)")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: val.GetTag(), Imm: val}", dv)
					g.emit("} else {")
					// Register receiver: load both words at runtime
					ptrReg := g.allocReg()
					auxReg := g.allocReg()
					g.emit("\toff := int32(%s)", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s)", auxReg, ptrReg)
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg)
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off+8)", auxReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
					g.emit("\tctx.BindReg(%s, &%s)", ptrReg, dv)
					g.emit("\tctx.BindReg(%s, &%s)", auxReg, dv)
					g.emit("}")
					gv := genVal{goVar: dv, isDesc: true}
					g.vals[name] = gv
					g.fieldCache[cacheKey] = gv
					break
				}

				// Scalar immutable field: no field deduplication (LocImm re-reads are free,
				// LocReg reloads use fresh short-lived registers to avoid pressure).

				// Determine register-relative load emit helper for LocReg thisptr path
				var emitLoadRel string
				switch goType {
				case "bool", "uint8", "int8":
					emitLoadRel = "EmitMovRegMemB"
				case "uint16", "int16":
					emitLoadRel = "EmitMovRegMemW"
				case "uint32", "int32":
					emitLoadRel = "EmitMovRegMemL"
				default: // int64, uint64
					emitLoadRel = "EmitMovRegMem"
				}

				dv := g.allocDesc()
				rv := g.allocReg()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if thisptr.Loc == LocImm {")
				// thisptr is compile-time constant → read immutable field at JIT compile time
				g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
				switch goType {
				case "bool":
					g.emit("\tval := *(*bool)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(val)}", dv)
				case "uint8":
					g.emit("\tval := *(*uint8)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "int8":
					g.emit("\tval := *(*int8)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "uint16":
					g.emit("\tval := *(*uint16)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "int16":
					g.emit("\tval := *(*int16)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "uint32":
					g.emit("\tval := *(*uint32)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "int32":
					g.emit("\tval := *(*int32)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				case "int64":
					g.emit("\tval := *(*int64)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(val)}", dv)
				case "uint64":
					g.emit("\tval := *(*uint64)(unsafe.Pointer(fieldAddr))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(val))}", dv)
				default:
					panic(fmt.Sprintf("unsupported immutable field type %s for %s", goType, fieldName))
				}
				g.emit("} else {")
				// thisptr is in a register → emit register-relative load at runtime
				g.emit("\toff := int32(%s)", src.offsetExpr)
				g.emit("\t%s := ctx.AllocReg()", rv)
				g.emit("\tctx.%s(%s, thisptr.Reg, off)", emitLoadRel, rv)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				g.emit("}")
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else if strings.HasPrefix(src.marker, "_fieldaddr:") {
				// Deref of FieldAddr → load from struct field at compile-time address
				parts := strings.SplitN(src.marker, ":", 3) // "_fieldaddr", size, fieldName
				sizeStr := parts[1]
				fieldName := parts[2]

				// Field deduplication: reuse cached load if available
				cacheKey := fieldName
				if cached, ok := g.fieldCache[cacheKey]; ok {
					g.vals[name] = cached
					break
				}

				dv := g.allocDesc()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", "thisptr")
				// Compile-time: compute address and emit load from fixed memory
				switch sizeStr {
				case "1":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem8(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem16(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem32(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					// Load slice header: ptr (8 bytes), len (8 bytes), cap (8 bytes)
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					capReg := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", ptrReg)    // data ptr
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr+8)", lenReg)  // length
					g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr+16)", capReg) // capacity
					g.emit("\t%s = JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv, ptrReg, lenReg, capReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\tctx.EmitMovRegImm64(%s, uint64(fieldAddr))", ptrReg)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg)
				}
				g.emit("} else {")
				// thisptr is in a register → emit register-relative loads
				g.emit("\toff := int32(%s)", src.offsetExpr)
				switch sizeStr {
				case "1":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.EmitMovRegMemB(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "2":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.EmitMovRegMemW(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "4":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.EmitMovRegMemL(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "8":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "slice":
					ptrReg2 := g.allocReg()
					lenReg2 := g.allocReg()
					capReg2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg2)
					g.emit("\t%s := ctx.AllocRegExcept(%s)", lenReg2, ptrReg2)
					g.emit("\t%s := ctx.AllocRegExcept(%s, %s)", capReg2, ptrReg2, lenReg2)
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg2)    // data ptr
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off+8)", lenReg2)  // length
					g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off+16)", capReg2) // capacity
					g.emit("\t%s = JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv, ptrReg2, lenReg2, capReg2)
				case "array":
					ptrReg2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg2)
					g.emit("\tctx.EmitMovRegReg(%s, thisptr.Reg)", ptrReg2)
					g.emit("\tctx.EmitAddRegImm32(%s, off)", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg2)
				}
				g.emit("}")
				if sizeStr == "slice" || sizeStr == "array" {
					gv := genVal{goVar: dv, isDesc: true, marker: "_slice"}
					g.vals[name] = gv
					g.fieldCache[cacheKey] = gv
				} else {
					gv := genVal{goVar: dv, isDesc: true}
					g.vals[name] = gv
					g.fieldCache[cacheKey] = gv
				}
			} else if strings.HasPrefix(src.marker, "_sliceaddr:") {
				// IndexAddr+Deref on a local slice (from FieldAddr or Slice())
				// marker: "_sliceaddr:elemSize:descVar"
				parts := strings.SplitN(src.marker, ":", 3)
				elemSize := parts[1]
				sliceDescVar := parts[2]
				sliceDescVar = g.overlayDescVar(sliceDescVar, src.deferredBaseSSA)
				idxDescVar := g.overlayDescVar(src.argIdxVar, src.deferredIndexSSA)
				dv := g.allocDesc()
				addressDesc := g.allocDesc()
				g.emit("%s := ctx.EmitSliceElementAddress(&%s, &%s, %s)", addressDesc, sliceDescVar, idxDescVar, elemSize)
				switch elemSize {
				case "8":
					g.emit("ctx.EnsureDesc(&%s)", addressDesc)
					// Load through the address in place. Architecture emitters handle
					// destination==base, avoiding a second live register.
					g.emit("ctx.EmitMovRegMem(%s.Reg, %s.Reg, 0)", addressDesc, addressDesc)
					g.emit("%s := %s", dv, addressDesc)
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				default:
					// Load aux first, then reuse the address register for ptr. A Scmer
					// element therefore needs only two live registers even when its slice
					// header is pinned across recursive CFG blocks.
					if phiTarget, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
						g.emit("ctx.EmitLoadScmerToStack(&%s, %s)", addressDesc, phiTarget)
						g.emit("ctx.FreeDesc(&%s)", addressDesc)
						g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: %s}", dv, phiTarget)
					} else {
						auxReg := g.allocReg()
						g.emit("ctx.EnsureDesc(&%s)", addressDesc)
						g.emit("%s := ctx.AllocRegExcept(%s.Reg)", auxReg, addressDesc)
						g.emit("ctx.EmitMovRegMem(%s, %s.Reg, 8)", auxReg, addressDesc)
						g.emit("ctx.EmitMovRegMem(%s.Reg, %s.Reg, 0)", addressDesc, addressDesc)
						g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s.Reg, Reg2: %s}", dv, addressDesc, auxReg)
					}
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				}
				// Free the deferred index operand from IndexAddr now that we've used it
				if src.deferredIndexSSA != "" {
					g.useOperand(src.deferredIndexSSA)
				}
			} else if strings.HasPrefix(src.marker, "_globaladdr:") {
				// IndexAddr+Deref on a global array (e.g. &pow10f[idx])
				// marker: "_globaladdr:elemSize:globalName"
				parts := strings.SplitN(src.marker, ":", 3)
				elemSize := parts[1]
				globalName := parts[2]
				dv := g.allocDesc()
				scratch := g.allocReg()
				idxDescVar := g.overlayDescVar(src.argIdxVar, src.deferredIndexSSA)
				g.emit("%s := ctx.AllocReg()", scratch)
				// Load base address of global array at compile time
				g.emit("ctx.EmitMovRegImm64(%s, uint64(uintptr(unsafe.Pointer(&%s[0]))))", scratch, globalName)
				// Compute byte offset: idx * elemSize, add to base
				idxReg := g.allocReg()
				g.emit("%s := ctx.AllocReg()", idxReg)
				g.emit("if %s.Loc == LocImm {", idxDescVar)
				g.emit("\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()) * %s)", idxReg, idxDescVar, elemSize)
				g.emit("} else {")
				g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", idxReg, idxDescVar)
				switch elemSize {
				case "8":
					g.emit("\tctx.EmitShlRegImm8(%s, 3)", idxReg) // *8
				default:
					g.emit("\tscratch2 := ctx.AllocReg()")
					g.emit("\tctx.EmitMovRegImm64(scratch2, %s)", elemSize)
					g.emit("\tctx.EmitImulInt64(%s, scratch2)", idxReg)
					g.emit("\tctx.FreeReg(scratch2)")
				}
				g.emit("}")
				// Add base pointer
				g.emit("ctx.EmitAddInt64(%s, %s)", scratch, idxReg)
				g.emit("ctx.FreeReg(%s)", idxReg)
				// Load value
				// Protect scratch so AllocReg cannot spill it and alias rv==scratch.
				rv := g.allocReg()
				g.emit("%s := ctx.AllocRegExcept(%s)", rv, scratch)
				g.emit("ctx.EmitMovRegMem(%s, %s, 0)", rv, scratch)
				g.emit("ctx.FreeReg(%s)", scratch)
				g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				// Free the deferred index operand
				if src.deferredIndexSSA != "" {
					g.useOperand(src.deferredIndexSSA)
				}
			} else if src.argIdx >= 0 {
				// Fused IndexAddr+Deref → args[i] already describes this argument
				dv := g.allocDesc()
				g.emit("%s := args[%d]", dv, src.argIdx)
				// Borrowed descriptor from caller: never own/free caller placements.
				g.emit("%s.ID = 0", dv)
				g.vals[name] = genVal{goVar: dv, isDesc: true, sourceInput: src.argIdx, hasSourceInput: true}
			} else if src.argIdxVar != "" {
				// Variable-index IndexAddr+Deref on emitter args.
				// If the index is known at emit-time, reuse args[idx] directly.
				// Otherwise, emit runtime selection across the fixed args list.
				dv := g.allocDesc()
				idxDescVar := g.overlayDescVar(src.argIdxVar, src.deferredIndexSSA)
				selectionOff := g.allocTemp("dynamicArgOff")
				doneLbl := g.allocLabel()
				oobLbl := g.allocLabel()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", idxDescVar)
				g.emit("\tidx := int(%s.Imm.Int()) + %d", idxDescVar, src.argBase)
				g.emit("\tif idx < 0 || idx >= len(args) {")
				g.emit("\t\tpanic(\"jitgen: dynamic args index out of range\")")
				g.emit("\t}")
				g.emit("\t%s = args[idx]", dv)
				g.emit("\t%s.ID = 0", dv)
				g.emit("} else {")
				g.emit("\tctx.EnsureDesc(&%s)", idxDescVar)
				g.emit("\t%s := ctx.AllocStack(16)", selectionOff)
				g.emit("\tctx.ProtectReg(%s.Reg)", idxDescVar)
				g.emit("\t%s := ctx.ReserveLabel()", doneLbl)
				g.emit("\t%s := ctx.ReserveLabel()", oobLbl)
				g.emit("\tctx.EmitCmpRegImm32(%s.Reg, int32(len(args)-%d))", idxDescVar, src.argBase)
				g.emit("\tctx.EmitJump(CondUnsignedAboveOrEqual, %s)", oobLbl)
				g.emit("\tfor i := %d; i < len(args); i++ {", src.argBase)
				g.emit("\t\tnextLbl := ctx.ReserveLabel()")
				g.emit("\t\tctx.EmitCmpRegImm32(%s.Reg, int32(i-%d))", idxDescVar, src.argBase)
				g.emit("\t\tctx.EmitJump(CondNotEqual, nextLbl)")
				g.emit("\t\tai := args[i]")
				g.emit("\t\tai.ID = 0")
				g.emit("\t\tctx.EmitStoreScmerToStack(ai, int32(%s))", selectionOff)
				g.emit("\t\tctx.EmitJmp(%s)", doneLbl)
				g.emit("\t\tctx.MarkLabel(nextLbl)")
				g.emit("\t}")
				g.emit("\tctx.MarkLabel(%s)", oobLbl)
				g.emit("\tctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(%s))", selectionOff)
				g.emit("\tctx.MarkLabel(%s)", doneLbl)
				g.emit("\tctx.UnprotectReg(%s.Reg)", idxDescVar)
				g.emit("\t%s = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s), Rooted: true}", dv, selectionOff)
				g.emit("}")
				if phiTarget, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
					g.emit("ctx.EmitStoreScmerToStack(%s, %s)", dv, phiTarget)
					g.emit("ctx.FreeDesc(&%s)", dv)
					g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: %s}", dv, phiTarget)
					g.vals[name] = genVal{goVar: dv, isDesc: true, pinAcrossBlock: true}
				} else if ssaValueCrossesControlFlow(v) {
					stableOff := g.allocTemp("dynamicArgOff")
					g.emit("%s := ctx.AllocStack(16)", stableOff)
					g.emit("ctx.EmitStoreScmerToStack(%s, int32(%s))", dv, stableOff)
					g.emit("ctx.FreeDesc(&%s)", dv)
					g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s), Rooted: true}", dv, stableOff)
					g.vals[name] = genVal{goVar: dv, isDesc: true, pinAcrossBlock: true}
				} else {
					g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_dynamic_variadic_element"}
				}
			} else {
				panic(fmt.Sprintf("deref of non-arg pointer: %s", v))
			}
		} else {
			panic(fmt.Sprintf("unsupported UnOp %s", v.Op))
		}

	case *ssa.Call:
		// Check for builtins first (len, cap, etc.)
		if builtin, ok := v.Call.Value.(*ssa.Builtin); ok {
			switch builtin.Name() {
			case "append":
				if len(v.Call.Args) != 2 {
					panic(fmt.Sprintf("append with unsupported argument count: %s", v))
				}
				sliceType := v.Call.Args[0].Type().Underlying().(*types.Slice)
				slice := g.vals[v.Call.Args[0].Name()]
				elements := g.vals[v.Call.Args[1].Name()]
				if !isScmerType(sliceType.Elem()) {
					if !slice.isDesc || !elements.isDesc || slice.marker != "_slice" || elements.marker != "_slice" {
						panic(fmt.Sprintf("append of non-Scmer slices requires descriptors: %s", v))
					}
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					typeExpr := g.sourceTypeExpr(v.Call.Args[0].Type())
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src %s) %s { return append(dst, src...) }), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, typeExpr, typeExpr, slice.goVar, elements.goVar)
					g.emit("%s := %s[0]", dv, callResults)
					g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
					break
				}
				if elements.marker == "_variadic_args" {
					end := ":"
					if elements.variadicLenKnown {
						end = fmt.Sprintf(":%d", elements.variadicOffset+elements.variadicLen)
					}
					materialized := g.allocDesc()
					g.emit("%s := jitMaterializeVirtualGoSlice(ctx, args[%d%s])", materialized, elements.variadicOffset, end)
					if constantSlice, ok := v.Call.Args[0].(*ssa.Const); ok && constantSlice.Value == nil {
						g.recordSliceResult(name, v, materialized)
						break
					}
					if slice.marker != "_slice" {
						panic(fmt.Sprintf("append virtual args to unsupported slice: %s (slice=%q)", v, slice.marker))
					}
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, slice.goVar, materialized)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				cloneWholeSlice := elements.marker == "_slice" && slice.marker == "_stackslice" && slice.stackLen == 0
				if constantSlice, ok := v.Call.Args[0].(*ssa.Const); ok && constantSlice.Value == nil && elements.marker == "_slice" {
					cloneWholeSlice = true
				}
				if cloneWholeSlice {
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{%s}, []uint8{3}, []uint8{1})", callResults, elements.goVar)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				if slice.marker == "_slice" && elements.marker == "_slice" {
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, slice.goVar, elements.goVar)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				if slice.marker == "_stackslice" && slice.stackBase != "" && elements.marker == "_slice" {
					stackSlice := g.allocDesc()
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					capReg := g.allocReg()
					g.emit("%s := ctx.AllocReg()", ptrReg)
					g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
					g.emit("%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
					g.emit("%s := JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: %s, Reg2: %s, Reg3: %s, NoHeapPointer: false}", stackSlice, ptrReg, lenReg, capReg)
					g.emit("ctx.EmitLeaRegMem(%s.Reg, ctx.StackReg, int32(%s))", stackSlice, slice.stackBase)
					g.emit("ctx.EmitMovRegImm64(%s.Reg2, uint64(%d))", stackSlice, slice.stackLen)
					g.emit("ctx.EmitMovRegImm64(%s.Reg3, uint64(%d))", stackSlice, slice.stackLen)
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSliceCopy), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, stackSlice, elements.goVar)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				if slice.marker == "_stackslice" && slice.stackBase != "" && elements.marker == "_stackslice" && elements.stackBase != "" {
					makeStackHeader := func(value genVal) string {
						header := g.allocDesc()
						ptrReg := g.allocReg()
						lenReg := g.allocReg()
						capReg := g.allocReg()
						g.emit("%s := ctx.AllocReg()", ptrReg)
						g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
						g.emit("%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
						g.emit("%s := JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: %s, Reg2: %s, Reg3: %s}", header, ptrReg, lenReg, capReg)
						g.emit("ctx.EmitLeaRegMem(%s.Reg, ctx.StackReg, int32(%s))", header, value.stackBase)
						g.emit("ctx.EmitMovRegImm64(%s.Reg2, uint64(%d))", header, value.stackLen)
						g.emit("ctx.EmitMovRegImm64(%s.Reg3, uint64(%d))", header, value.stackLen)
						return header
					}
					left := makeStackHeader(slice)
					right := makeStackHeader(elements)
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSliceCopy), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, left, right)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				if slice.marker != "_slice" || elements.stackBase == "" {
					panic(fmt.Sprintf("append requires a descriptor slice and local Scmer elements: %s (slice=%q elements=%q/%d)", v, slice.marker, elements.marker, elements.stackLen))
				}
				// A statically bounded initial capacity does not prove that a loop's
				// phi reaches this append at most that many times. Keep Go's growth
				// semantics unless a future range proof covers the complete loop.
				boundedSingleAppend := false
				if !boundedSingleAppend {
					added := g.allocDesc()
					addedPtr := g.allocReg()
					addedLen := g.allocReg()
					addedCap := g.allocReg()
					g.emit("%s := ctx.AllocReg()", addedPtr)
					g.emit("%s := ctx.AllocRegExcept(%s)", addedLen, addedPtr)
					g.emit("%s := ctx.AllocRegExcept(%s, %s)", addedCap, addedPtr, addedLen)
					g.emit("%s := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: %s, Reg2: %s, Reg3: %s}", added, addedPtr, addedLen, addedCap)
					g.emit("ctx.BindReg(%s, &%s)", addedPtr, added)
					g.emit("ctx.BindReg(%s, &%s)", addedLen, added)
					g.emit("ctx.BindReg(%s, &%s)", addedCap, added)
					g.emit("ctx.EmitLeaRegMem(%s.Reg, ctx.StackReg, int32(%s))", added, elements.stackBase)
					g.emit("ctx.EmitMovRegImm64(%s.Reg2, uint64(%d))", added, elements.stackLen)
					g.emit("ctx.EmitMovRegImm64(%s.Reg3, uint64(%d))", added, elements.stackLen)
					callResults := g.allocTemp("callResults")
					dv := g.allocDesc()
					g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, slice.goVar, added)
					g.emit("%s := %s[0]", dv, callResults)
					g.recordSliceResult(name, v, dv)
					break
				}
				g.emit("ctx.EnsureDesc(&%s)", slice.goVar)
				g.emit("if %s.Loc != LocRegTriple { panic(\"jit: append requires a Go slice header\") }", slice.goVar)
				g.emit("ctx.ProtectReg(%s.Reg)", slice.goVar)
				g.emit("ctx.ProtectReg(%s.Reg2)", slice.goVar)
				g.emit("ctx.ProtectReg(%s.Reg3)", slice.goVar)
				capacityOK := g.allocLabel()
				g.emit("%s := ctx.ReserveLabel()", capacityOK)
				g.emit("ctx.EmitCmpInt64(%s.Reg2, %s.Reg3)", slice.goVar, slice.goVar)
				g.emit("ctx.EmitJump(CondUnsignedBelow, %s)", capacityOK)
				g.emit("ctx.EmitGoPanic(\"jit: generated append exceeded its fixed capacity\")")
				g.emit("ctx.MarkLabel(%s)", capacityOK)
				index := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2, NoHeapPointer: true}", index, slice.goVar)
				address := g.allocDesc()
				g.emit("%s := ctx.EmitSliceElementAddress(&%s, &%s, int32(16))", address, slice.goVar, index)
				value := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s)}", value, elements.stackBase)
				g.emit("ctx.EmitStoreScmerAt(&%s, &%s)", address, value)
				g.emit("ctx.FreeDesc(&%s)", address)
				g.emit("ctx.EmitAddRegImm32(%s.Reg2, 1)", slice.goVar)
				g.emit("ctx.UnprotectReg(%s.Reg3)", slice.goVar)
				g.emit("ctx.UnprotectReg(%s.Reg2)", slice.goVar)
				g.emit("ctx.UnprotectReg(%s.Reg)", slice.goVar)
				dv := g.allocDesc()
				if phiTarget, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetTriple {
					g.emit("ctx.EmitStoreRegMem(%s.Reg, RegRSP, %s)", slice.goVar, phiTarget)
					g.emit("ctx.EmitStoreRegMem(%s.Reg2, RegRSP, %s+8)", slice.goVar, phiTarget)
					g.emit("ctx.EmitStoreRegMem(%s.Reg3, RegRSP, %s+16)", slice.goVar, phiTarget)
					g.emit("ctx.FreeDesc(&%s)", slice.goVar)
					g.emit("%s := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: %s}", dv, phiTarget)
				} else {
					g.emit("%s := %s", dv, slice.goVar)
					g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
					g.emit("ctx.BindReg(%s.Reg2, &%s)", dv, dv)
					g.emit("ctx.BindReg(%s.Reg3, &%s)", dv, dv)
				}
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
			case "len":
				arg := v.Call.Args[0]
				if arg.Name() == g.paramName {
					// len(args) — known at emit time
					dv := g.allocDesc()
					g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}", dv)
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				} else {
					// len of a local descriptor-backed value (slice/string intermediates)
					src := g.vals[arg.Name()]
					if src.marker == "_variadic_args" {
						dv := g.allocDesc()
						if src.variadicLenKnown {
							g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%d)}", dv, src.variadicLen)
						} else {
							g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)-%d))}", dv, src.variadicOffset)
						}
						g.vals[name] = genVal{goVar: dv, isDesc: true}
					} else if src.marker == "_slice" || src.marker == "_gostring" || src.isDesc {
						dv := g.allocDesc()
						g.emit("var %s JITValueDesc", dv)
						g.emit("if %s.SliceSizeKnown {", src.goVar)
						g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.KnownSliceLen))}", dv, src.goVar)
						g.emit("} else if %s.Loc == LocImm {", src.goVar)
						if src.marker == "_gostring" {
							// LocImm Scmer string constant: derive Go-string length.
							g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(%s.Imm.String())))}", dv, src.goVar)
						} else {
							// Legacy LocImm slice path stores length in StackOff.
							g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.StackOff))}", dv, src.goVar)
						}
						g.emit("} else if %s.Loc == LocStackTriple {", src.goVar)
						g.emit("\t%s = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: %s.StackOff + 8, NoHeapPointer: true}", dv, src.goVar)
						if src.marker == "_gostring" {
							g.emit("} else if %s.Loc == LocStackPair {", src.goVar)
							g.emit("\t%s = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: %s.StackOff + 8, NoHeapPointer: true}", dv, src.goVar)
						}
						g.emit("} else {")
						g.emit("\tctx.EnsureDesc(&%s)", src.goVar)
						g.emit("\tif %s.Loc == LocRegPair || %s.Loc == LocRegTriple {", src.goVar, src.goVar)
						g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2, ID: 0}", dv, src.goVar)
						g.emit("\t} else if %s.Loc == LocReg {", src.goVar)
						g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg, ID: 0}", dv, src.goVar)
						g.emit("\t} else {")
						g.emit("\t\tpanic(\"len on unsupported descriptor location\")")
						g.emit("\t}")
						g.emit("}")
						g.vals[name] = genVal{goVar: dv, isDesc: true}
						if src.hasSliceInput {
							withProvenance := g.vals[name]
							withProvenance.lengthInput = src.sliceInput
							withProvenance.hasLengthInput = true
							g.vals[name] = withProvenance
						}
					} else {
						panic(fmt.Sprintf("len on non-parameter: %s", v))
					}
				}
			case "cap":
				arg := v.Call.Args[0]
				src := g.vals[arg.Name()]
				if arg.Name() == g.paramName || src.marker == "_variadic_args" {
					dv := g.allocDesc()
					offset := src.variadicOffset
					if src.variadicLenKnown {
						g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%d)}", dv, src.variadicLen)
					} else {
						g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)-%d))}", dv, offset)
					}
					g.vals[name] = genVal{goVar: dv, isDesc: true}
					break
				}
				if src.goVar == "" || !src.isDesc || src.marker != "_slice" {
					panic(fmt.Sprintf("cap on unsupported value: %s", v))
				}
				dv := g.allocDesc()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.SliceSizeKnown {", src.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.KnownSliceCap))}", dv, src.goVar)
				g.emit("} else if %s.Loc == LocStackTriple {", src.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: %s.StackOff + 16, NoHeapPointer: true}", dv, src.goVar)
				g.emit("} else {")
				g.emit("\tctx.EnsureDesc(&%s)", src.goVar)
				g.emit("\tif %s.Loc != LocRegTriple { panic(\"cap requires a slice triple\") }", src.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg3, ID: 0}", dv, src.goVar)
				g.emit("}")
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			case "copy":
				if len(v.Call.Args) != 2 {
					panic(fmt.Sprintf("copy with unsupported arity: %s", v))
				}
				dstType, dstOK := v.Call.Args[0].Type().Underlying().(*types.Slice)
				srcType, srcOK := v.Call.Args[1].Type().Underlying().(*types.Slice)
				if !dstOK || !srcOK || !types.Identical(dstType.Elem(), srcType.Elem()) {
					panic(fmt.Sprintf("copy of unsupported types: %s", v))
				}
				helper := ""
				switch {
				case isScmerType(dstType.Elem()):
					helper = "jitCopyScmerSlice"
				case isByteType(dstType.Elem()):
					helper = "jitCopyByteSlice"
				default:
					panic(fmt.Sprintf("copy of unsupported element type: %s", dstType.Elem()))
				}
				dst := g.resolveValue(v.Call.Args[0])
				src := g.resolveValue(v.Call.Args[1])
				g.emit("ctx.EnsureDesc(&%s)", dst.goVar)
				g.emit("ctx.EnsureDesc(&%s)", src.goVar)
				callResults := g.allocTemp("callResults")
				dv := g.allocDesc()
				g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(%s), []JITValueDesc{%s, %s}, []uint8{1}, []uint8{0})", callResults, helper, dst.goVar, src.goVar)
				g.emit("%s := %s[0]", dv, callResults)
				g.emit("%s.Type = tagInt", dv)
				if name == "" {
					g.emit("_ = %s", dv)
				} else {
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				}
			default:
				panic(fmt.Sprintf("unsupported builtin: %s", builtin.Name()))
			}
			break
		}
		callee := v.Call.StaticCallee()
		if callee == nil {
			if g.emitInterfaceInvoke(name, v) {
				break
			}
			callable := g.vals[v.Call.Value.Name()]
			if callable.marker == "_go_closure" {
				result, ok := g.tryInlineClosure(callable, v.Call.Args)
				if !ok {
					panic(fmt.Sprintf("unsupported known closure call: %s", v))
				}
				if name != "" {
					g.vals[name] = result
				}
				break
			}
			if callable.marker == "_gofunc_variadic" && len(v.Call.Args) == 1 {
				callArgs := g.vals[v.Call.Args[0].Name()]
				var argsDesc genVal
				switch callArgs.marker {
				case "_slice":
					argsDesc = callArgs
				case "_variadic_args":
					end := ":"
					if callArgs.variadicLenKnown {
						end = fmt.Sprintf(":%d", callArgs.variadicOffset+callArgs.variadicLen)
					}
					materialized := g.allocDesc()
					g.emit("%s := jitMaterializeVirtualGoSlice(ctx, args[%d%s])", materialized, callArgs.variadicOffset, end)
					argsDesc = genVal{goVar: materialized, isDesc: true, marker: "_slice"}
				case "_stackslice":
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					capReg := g.allocReg()
					header := g.allocDesc()
					g.emit("%s := ctx.AllocReg()", ptrReg)
					g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
					g.emit("%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
					g.emit("%s := JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: %s, Reg2: %s, Reg3: %s}", header, ptrReg, lenReg, capReg)
					g.emit("ctx.EmitLeaRegMem(%s.Reg, ctx.StackReg, int32(%s))", header, callArgs.stackBase)
					g.emit("ctx.EmitMovRegImm64(%s.Reg2, uint64(%d))", header, callArgs.stackLen)
					g.emit("ctx.EmitMovRegImm64(%s.Reg3, uint64(%d))", header, callArgs.stackLen)
					argsDesc = genVal{goVar: header, isDesc: true, marker: "_slice"}
				default:
					panic(fmt.Sprintf("dynamic Go call args are not a Scmer slice: %s", v))
				}
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeGoFunctionSlice), []JITValueDesc{%s, %s}, 2)", dv, callable.goVar, argsDesc.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			if callable.marker == "_gofunc" {
				sig, ok := v.Call.Value.Type().Underlying().(*types.Signature)
				if !ok || sig.Params().Len() != 2 || sig.Results().Len() != 1 || !isScmerType(sig.Params().At(0).Type()) || !isScmerType(sig.Params().At(1).Type()) || !isScmerType(sig.Results().At(0).Type()) {
					panic(fmt.Sprintf("dynamic Go function has unsupported signature: %s", v))
				}
				left := g.resolveValue(v.Call.Args[0])
				right := g.resolveValue(v.Call.Args[1])
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeMergeCallback), []JITValueDesc{%s, %s, %s}, 2)", dv, callable.goVar, left.goVar, right.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			if callable.marker != "_serial_callable" || len(v.Call.Args) != 1 {
				panic(fmt.Sprintf("dynamic call: %s", v))
			}
			g.emitSerialCallableCall(name, v, callable, g.vals[v.Call.Args[0].Name()])
			break
		}
		if isSerialProcCall(callee) {
			if len(v.Call.Args) != 2 {
				panic(fmt.Sprintf("SerialProc.Call with unsupported arity: %s", v))
			}
			callable := g.vals[v.Call.Args[0].Name()]
			if callable.marker != "_serial_callable" {
				panic(fmt.Sprintf("SerialProc.Call receiver is not a prepared callback: %s", v))
			}
			g.emitSerialCallableCall(name, v, callable, g.vals[v.Call.Args[1].Name()])
			break
		}
		atomicPkg := callee.Pkg != nil && callee.Pkg.Pkg != nil && callee.Pkg.Pkg.Path() == "sync/atomic"
		atomicLoad := callee.Name() == "LoadInt64" || (atomicPkg && callee.Name() == "Load")
		atomicStore := callee.Name() == "StoreInt64" || (atomicPkg && callee.Name() == "Store")
		if atomicLoad {
			// sync/atomic.LoadInt64(ptr) / (*sync/atomic.Int64).Load() → int64
			// SSA method form passes receiver pointer as first argument.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			if strings.HasPrefix(arg.marker, "_fieldaddr:") || strings.HasPrefix(arg.marker, "_fieldconst:") {
				rv := g.allocReg()
				g.emit("%s := ctx.AllocReg()", rv)
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", arg.offsetExpr)
				g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", rv)
				g.emit("} else {")
				g.emit("\toff := int32(%s)", arg.offsetExpr)
				g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", rv)
				g.emit("}")
				g.emit("%s := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, rv)
			} else {
				panic(fmt.Sprintf("LoadInt64 arg is not a field address: marker=%q", arg.marker))
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
			break
		}
		if atomicStore {
			// sync/atomic.StoreInt64(ptr, val) / (*sync/atomic.Int64).Store(val)
			var dst genVal
			var val genVal
			if callee.Name() == "StoreInt64" {
				dst = g.vals[v.Call.Args[0].Name()]
				val = g.resolveValue(v.Call.Args[1])
			} else {
				dst = g.vals[v.Call.Args[0].Name()]
				val = g.resolveValue(v.Call.Args[1])
			}
			if strings.HasPrefix(dst.marker, "_fieldaddr:") || strings.HasPrefix(dst.marker, "_fieldconst:") {
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tbaseReg := ctx.AllocReg()")
				g.emit("\tif %s.Loc == LocReg {", val.goVar)
				g.emit("\t\tctx.FreeReg(baseReg)")
				g.emit("\t\tbaseReg = ctx.AllocRegExcept(%s.Reg)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + %s))", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.EmitStoreRegMem(RegR11, baseReg, 0)")
				g.emit("\t} else {")
				g.emit("\t\tctx.EmitStoreRegMem(%s.Reg, baseReg, 0)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.FreeReg(baseReg)")
				g.emit("} else {")
				g.emit("\toff := int32(%s)", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.EmitStoreRegMem(RegR11, thisptr.Reg, off)")
				g.emit("\t} else {")
				g.emit("\t\tctx.EmitStoreRegMem(%s.Reg, thisptr.Reg, off)", val.goVar)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("StoreInt64 dst is not a field address: marker=%q", dst.marker))
			}
			break
		}
		if callee.Name() == "String" && len(v.Call.Args) > 0 && !isScmerType(v.Call.Args[0].Type()) {
			if !g.emitGenericStaticCall(name, callee, v.Call.Args) {
				panic(fmt.Sprintf("unsupported non-Scmer String method: %s", v))
			}
			break
		}
		switch callee.Name() {
		case "asSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Type == tagSlice {", arg.goVar)
			g.emit("\t%s = jitKnownSliceHeader(ctx, &%s)", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{%s}, 3)", dv, arg.goVar)
			g.emit("}")
			g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("ctx.BindReg(%s.Reg2, &%s)", dv, dv)
			g.emit("ctx.BindReg(%s.Reg3, &%s)", dv, dv)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", sliceInput: arg.sourceInput, hasSliceInput: arg.hasSourceInput}
		case "GetTag":
			arg := g.vals[v.Call.Args[0].Name()]
			if !arg.isDesc {
				panic("GetTag expects Scmer descriptor")
			}
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGetTagDesc(&%s, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			// EmitGetTagDesc already sets Type: tagInt on LocReg results
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsNil":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagNil, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsInt":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagInt, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFloat":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagFloat, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsBool":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagBool, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsString":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagString, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagSlice, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFastDict":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitTagEqualsBorrowed(&%s, tagFastDict, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Bool":
			// (Scmer).Bool() — extract bool from Scmer.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s.ID = 0", tmp)
			g.emit("%s := ctx.EmitBoolDesc(&%s, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Int":
			// (Scmer).Int() — extract int64 from Scmer.
			// Fast-path only when type is statically known int; otherwise call helper
			// for full runtime semantics (float/string/bool/date conversions).
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int())}", dv, arg.goVar)
			g.emit("} else if %s.Type == tagInt && %s.Loc == LocRegPair {", arg.goVar, arg.goVar)
			g.emit("\tctx.FreeReg(%s.Reg)", arg.goVar) // free ptr, keep aux
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, dv)
			g.emit("} else if %s.Type == tagInt && %s.Loc == LocReg {", arg.goVar, arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", arg.goVar, dv)
			g.emit("} else {")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.emit("\t%s.Type = tagInt", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Float":
			// (Scmer).Float() — extract float64 from Scmer.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%s.Imm.Float())}", dv, arg.goVar)
			g.emit("} else if %s.Type == tagFloat && %s.Loc == LocReg {", arg.goVar, arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", arg.goVar, dv)
			g.emit("} else if %s.Type == tagFloat && %s.Loc == LocRegPair {", arg.goVar, arg.goVar)
			g.emit("\tctx.FreeReg(%s.Reg)", arg.goVar) // free ptr, keep aux (float bits)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, dv)
			g.emit("} else {")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.emit("\t%s.Type = tagFloat", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "String":
			// (Scmer).String() string — extract Go string from Scmer
			// arg: Scmer (2 words), result: Go string (2 words: ptr+len)
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			pair := g.allocDesc()
			g.emit("%s := %s", pair, arg.goVar)
			g.emit("ctx.SyncDesc(&%s)", pair)
			g.emit("if %s.Loc == LocMem {", pair)
			g.emit("\ttmpScalar := JITValueDesc{Loc: LocReg, Type: %s.Type, Reg: ctx.AllocReg()}", pair)
			g.emit("\tscratch := ctx.AllocRegExcept(tmpScalar.Reg)")
			g.emit("\tctx.EmitMovRegImm64(scratch, uint64(%s.MemPtr))", pair)
			g.emit("\tctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)")
			g.emit("\tctx.FreeReg(scratch)")
			g.emit("\tctx.BindReg(tmpScalar.Reg, &tmpScalar)")
			g.emit("\t%s = tmpScalar", pair)
			g.emit("}")
			g.emit("%s = JITPrepareScmerGoArg(ctx, %s)", pair, pair)
			g.emit("if %s.Loc != LocRegPair && %s.Loc != LocStackPair && %s.Loc != LocInputPair {", pair, pair, pair)
			g.emit("\tpanic(\"jit: Scmer.String receiver not materialized as pair\")")
			g.emit("}")
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{%s}, 2)", dv, pair)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
		case "NewBool":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			if target, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
				dv := g.allocDesc()
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagBool, %s)", src.goVar, target)
				g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: tagBool, StackOff: %s}", dv, target)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else {
				g.vals[name] = genVal{goVar: src.goVar, isDesc: src.isDesc, marker: "_newbool", resultTargetVar: src.resultTargetVar}
			}
		case "NewInt":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			if target, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
				dv := g.allocDesc()
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagInt, %s)", src.goVar, target)
				g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: tagInt, StackOff: %s}", dv, target)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else {
				g.vals[name] = genVal{goVar: src.goVar, isDesc: src.isDesc, marker: "_newint", resultTargetVar: src.resultTargetVar}
			}
		case "NewFloat":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			if target, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
				dv := g.allocDesc()
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagFloat, %s)", src.goVar, target)
				g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: tagFloat, StackOff: %s}", dv, target)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else {
				g.vals[name] = genVal{goVar: src.goVar, isDesc: src.isDesc, marker: "_newfloat", resultTargetVar: src.resultTargetVar}
			}
		case "NewNil":
			dv := g.allocDesc()
			if target, shape, direct := g.directPhiTarget(v); direct && shape == phiTargetPair {
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}", dv)
				g.emit("ctx.EmitStoreScmerToStack(%s, %s)", dv, target)
				g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: tagNil, StackOff: %s}", dv, target)
			} else {
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}", dv)
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "NewString":
			// NewString(s string) Scmer — arg is a Go string (2 words: ptr+len), result is Scmer (2 words)
			arg := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: arg.goVar, isDesc: arg.isDesc, marker: "_newstring"}
		case "NewFastDict":
			// NewFastDict(fd *FastDict) Scmer — construct Scmer from *FastDict ptr
			// arg: 1 word (raw pointer), result: 2 words (Scmer)
			src := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\tpanic(\"NewFastDict: LocImm not expected at JIT compile time\")")
			g.emit("} else {")
			auxReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", auxReg)
			g.emit("\tctx.EmitMovRegImm64(%s, makeAux(tagFastDict, 0))", auxReg)
			g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: %s.Reg, Reg2: %s}", dv, src.goVar, auxReg)
			g.emit("\tctx.TransferReg(%s.Reg)", src.goVar)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", src.goVar, dv)
			g.emit("\tctx.BindReg(%s, &%s)", auxReg, dv)
			g.emit("\t%s.Loc = LocNone", src.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "NewFastDictValue":
			// NewFastDictValue(cap int) *FastDict — Go call, returns 1 word
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "NewSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			if constantSlice, ok := v.Call.Args[0].(*ssa.Const); ok && constantSlice.Value == nil {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}", dv)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_newargslice"}
				break
			}
			if arg.marker == "_slice" {
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitNewSliceFromGoSlice(&%s)", dv, arg.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			if arg.marker == "_stackslice" && arg.stackBase != "" {
				goSlice := g.allocDesc()
				goSlicePtr := g.allocReg()
				goSliceLen := g.allocReg()
				goSliceCap := g.allocReg()
				g.emit("%s := ctx.AllocReg()", goSlicePtr)
				g.emit("%s := ctx.AllocRegExcept(%s)", goSliceLen, goSlicePtr)
				g.emit("%s := ctx.AllocRegExcept(%s, %s)", goSliceCap, goSlicePtr, goSliceLen)
				g.emit("%s := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: %s, Reg2: %s, Reg3: %s}", goSlice, goSlicePtr, goSliceLen, goSliceCap)
				g.emit("ctx.BindReg(%s, &%s)", goSlicePtr, goSlice)
				g.emit("ctx.BindReg(%s, &%s)", goSliceLen, goSlice)
				g.emit("ctx.BindReg(%s, &%s)", goSliceCap, goSlice)
				g.emit("ctx.EmitLeaRegMem(%s.Reg, ctx.StackReg, int32(%s))", goSlice, arg.stackBase)
				g.emit("ctx.EmitMovRegImm64(%s.Reg2, uint64(%d))", goSlice, arg.stackLen)
				g.emit("ctx.EmitMovRegImm64(%s.Reg3, uint64(%d))", goSlice, arg.stackLen)
				callResults := g.allocTemp("callResults")
				dv := g.allocDesc()
				g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{%s}, []uint8{2}, []uint8{1})", callResults, goSlice)
				g.emit("%s := %s[0]", dv, callResults)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			if arg.marker != "_variadic_args" {
				panic(fmt.Sprintf("NewSlice on non-variadic parameter: %s", v))
			}
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: append([]JITValueDesc(nil), args...)}", dv)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_newargslice"}
		case "OptimizeProcToSerialFunction", "PrepareSerialProc":
			// Both callback preparation helpers are compiler-only when the callback
			// shape is known. Preserve the lambda template so the generated Call
			// instruction below recursively invokes its emitter at the loop site.
			// Known lambda templates remain compile-time values and can be inlined by
			// the generated caller. A dynamic callback may be hoisted into a hidden
			// entry argument only when it is an actual input of the enclosing JIT Proc.
			// Nested builtin arguments otherwise have no valid entry-point provenance;
			// preserve their callable Scmer so callback dispatch can select Proc.JIT or
			// the interpreter at the loop site.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			if arg.marker == "_knownimm" {
				optimizedVar := g.allocTemp("optimizedCallback")
				g.emit("%s := NewFunc(OptimizeProcToSerialFunction(%s.Imm))", optimizedVar, arg.goVar)
				g.emit("ctx.TrackImm(%s)", optimizedVar)
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: %s, Rooted: true}", dv, optimizedVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_serial_callable"}
				break
			}
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocLambdaTemplate {", arg.goVar)
			g.emit("\t%s = %s", dv, arg.goVar)
			g.emit("} else if %s.Loc == LocImm {", arg.goVar)
			optimizedVar := g.allocTemp("optimizedCallback")
			g.emit("\t%s := NewFunc(OptimizeProcToSerialFunction(%s.Imm))", optimizedVar, arg.goVar)
			g.emit("\tctx.TrackImm(%s)", optimizedVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: %s, Rooted: true}", dv, optimizedVar)
			g.emit("} else {")
			g.emit("\tif %s.Loc == LocInputPair && int(%s.StackOff) < ctx.InputArgCount {", arg.goVar, arg.goVar)
			g.emit("\t\t%s = ctx.RequestOptimizedCallback(int(%s.StackOff))", dv, arg.goVar)
			g.emit("\t} else {")
			g.emit("\t\t%s = jitCopyScmerToPair(ctx, %s)", dv, arg.goVar)
			g.emit("\t}")
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_serial_callable"}
		case "FastDict":
			// (Scmer).FastDict() *FastDict — extract ptr field, free aux
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("ctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\tpanic(\"FastDict: LocImm not expected at JIT compile time\")")
			g.emit("} else if %s.Loc != LocRegPair {", arg.goVar)
			g.emit("\tpanic(\"FastDict: expected Scmer register pair\")")
			g.emit("} else {")
			g.emit("\tctx.FreeReg(%s.Reg2)", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s.Reg}", dv, arg.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", arg.goVar, dv)
			g.emit("\t%s.Loc = LocNone", arg.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Set":
			// (*FastDict).Set(fd, key, value, mergeFn) — void Go call
			recv := g.vals[v.Call.Args[0].Name()]     // *FastDict (1 word)
			key := g.vals[v.Call.Args[1].Name()]      // Scmer (2 words)
			val := g.vals[v.Call.Args[2].Name()]      // Scmer (2 words)
			mergeFn := g.resolveValue(v.Call.Args[3]) // func (1 word)
			g.emit("%s = JITPrepareScmerGoArg(ctx, %s)", key.goVar, key.goVar)
			g.emit("%s = JITPrepareScmerGoArg(ctx, %s)", val.goVar, val.goVar)
			g.emit("ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{%s, %s, %s, %s})", recv.goVar, key.goVar, val.goVar, mergeFn.goVar)
		case "Sqrt":
			// math.Sqrt(float64) float64 via bit-helper (Go ABI float args are not marshaled directly).
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			src := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(%s.Imm.Float()))}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("\tvar %s JITValueDesc", src)
			g.emit("\tif %s.Loc == LocRegPair {", arg.goVar)
			g.emit("\t\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, src)
			g.emit("\t} else {")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg, &%s)", arg.goVar, src)
			g.emit("\t}")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{%s}, 1)", dv, src)
			g.emit("\t%s.Type = tagFloat", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Floor":
			fallthrough
		case "archFloor":
			// math arch helper for floor(float64) float64
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			src := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Floor(%s.Imm.Float()))}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("\tvar %s JITValueDesc", src)
			g.emit("\tif %s.Loc == LocRegPair {", arg.goVar)
			g.emit("\t\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, src)
			g.emit("\t} else {")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg, &%s)", arg.goVar, src)
			g.emit("\t}")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITFloorBits), []JITValueDesc{%s}, 1)", dv, src)
			g.emit("\t%s.Type = tagFloat", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Ceil":
			fallthrough
		case "archCeil":
			// math arch helper for ceil(float64) float64
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			src := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Ceil(%s.Imm.Float()))}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("\tvar %s JITValueDesc", src)
			g.emit("\tif %s.Loc == LocRegPair {", arg.goVar)
			g.emit("\t\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, src)
			g.emit("\t} else {")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg, &%s)", arg.goVar, src)
			g.emit("\t}")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITCeilBits), []JITValueDesc{%s}, 1)", dv, src)
			g.emit("\t%s.Type = tagFloat", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "archTrunc":
			fallthrough
		case "Trunc":
			// trunc(float64) float64 without Go-call ABI float args.
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(%s.Imm.Float()))}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("\tvar truncSrc Reg")
			g.emit("\tif %s.Loc == LocRegPair {", arg.goVar)
			g.emit("\t\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t\ttruncSrc = %s.Reg2", arg.goVar)
			g.emit("\t} else {")
			g.emit("\t\ttruncSrc = %s.Reg", arg.goVar)
			g.emit("\t}")
			g.emit("\ttruncInt := ctx.AllocRegExcept(truncSrc)")
			g.emit("\tctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)")
			g.emit("\tctx.EmitCvtInt64ToFloat64(RegX0, truncInt)")
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}", dv)
			g.emit("\tctx.BindReg(truncInt, &%s)", dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Abs":
			// math.Abs(float64) float64 via bit-helper.
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			src := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Abs(%s.Imm.Float()))}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.EnsureDesc(&%s)", arg.goVar)
			g.emit("\tvar %s JITValueDesc", src)
			g.emit("\tif %s.Loc == LocRegPair {", arg.goVar)
			g.emit("\t\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, src)
			g.emit("\t} else {")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", src, arg.goVar)
			g.emit("\t\tctx.BindReg(%s.Reg, &%s)", arg.goVar, src)
			g.emit("\t}")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITAbsBits), []JITValueDesc{%s}, 1)", dv, src)
			g.emit("\t%s.Type = tagFloat", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Slice":
			// (Scmer).Slice() []Scmer — decode the complete ptr/len/cap header.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := jitKnownSliceHeader(ctx, &%s)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", sliceInput: arg.sourceInput, hasSliceInput: arg.hasSourceInput}
		case "JITBuildMergeClosure":
			// JITBuildMergeClosure(func(...Scmer) Scmer) func(Scmer, Scmer) Scmer
			// arg: 1 word, result: 1 word
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(JITBuildMergeClosure), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "LoadInt64":
			// sync/atomic.LoadInt64(ptr) int64 — atomic load from field address
			// ptr is a FieldAddr-based descriptor; on x86 aligned MOV is atomic
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			if strings.HasPrefix(arg.marker, "_fieldaddr:") || strings.HasPrefix(arg.marker, "_fieldconst:") {
				rv := g.allocReg()
				g.emit("%s := ctx.AllocReg()", rv)
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", arg.offsetExpr)
				g.emit("\tctx.EmitMovRegMem64(%s, fieldAddr)", rv)
				g.emit("} else {")
				g.emit("\toff := int32(%s)", arg.offsetExpr)
				g.emit("\tctx.EmitMovRegMem(%s, thisptr.Reg, off)", rv)
				g.emit("}")
				g.emit("%s := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, rv)
			} else {
				panic(fmt.Sprintf("LoadInt64 arg is not a field address: marker=%q", arg.marker))
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "StoreInt64":
			// sync/atomic.StoreInt64(ptr, val) — atomic store to field address
			// On x86, aligned MOV is atomic for 64-bit values
			dst := g.vals[v.Call.Args[0].Name()]
			val := g.resolveValue(v.Call.Args[1])
			if strings.HasPrefix(dst.marker, "_fieldaddr:") || strings.HasPrefix(dst.marker, "_fieldconst:") {
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tbaseReg := ctx.AllocReg()")
				g.emit("\tif %s.Loc == LocReg {", val.goVar)
				g.emit("\t\tctx.FreeReg(baseReg)")
				g.emit("\t\tbaseReg = ctx.AllocRegExcept(%s.Reg)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + %s))", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.EmitStoreRegMem(RegR11, baseReg, 0)")
				g.emit("\t} else {")
				g.emit("\t\tctx.EmitStoreRegMem(%s.Reg, baseReg, 0)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.FreeReg(baseReg)")
				g.emit("} else {")
				g.emit("\toff := int32(%s)", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.EmitStoreRegMem(RegR11, thisptr.Reg, off)")
				g.emit("\t} else {")
				g.emit("\t\tctx.EmitStoreRegMem(%s.Reg, thisptr.Reg, off)", val.goVar)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("StoreInt64 dst is not a field address: marker=%q", dst.marker))
			}
		default:
			if g.storageMode {
				if g.emitGenericStaticCall(name, callee, v.Call.Args) {
					break
				}
				if callee.Blocks == nil {
					panic(fmt.Sprintf("unsupported call: %s", v))
				}
				result := g.inlineCall(callee, v.Call.Args)
				if name != "" {
					g.vals[name] = result
				}
				break
			}
			if result, ok := g.tryInlineCall(callee, v.Call.Args); ok {
				if name != "" {
					if _, signature := v.Type().Underlying().(*types.Signature); signature {
						result.marker = "_gofunc_variadic"
					}
					g.vals[name] = result
				}
				break
			}
			if !g.emitGenericStaticCall(name, callee, v.Call.Args) {
				panic(fmt.Sprintf("unsupported call: %s", v))
			}
		}

	case *ssa.BinOp:
		// Check for string concatenation before integer path
		if basic, ok := v.X.Type().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
			if v.Op == token.ADD {
				// String concatenation: call runtime concat function
				xVal := g.resolveValue(v.X)
				yVal := g.resolveValue(v.Y)
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(ConcatStrings), []JITValueDesc{%s, %s}, 2)", dv, xVal.goVar, yVal.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
				break
			}
			if v.Op == token.EQL || v.Op == token.NEQ {
				materializeString := func(value ssa.Value) genVal {
					resolved := g.resolveValue(value)
					if resolved.marker == "_gostring" {
						return resolved
					}
					dv := g.allocDesc()
					g.emit("var %s JITValueDesc", dv)
					g.emit("if %s.Loc == LocImm {", resolved.goVar)
					g.emit("\tctx.TrackImm(%s.Imm)", resolved.goVar)
					g.emit("\tptrWord, _ := %s.Imm.RawWords()", resolved.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}", dv)
					g.emit("\tctx.EmitMovRegImm64(%s.Reg, uint64(ptrWord))", dv)
					g.emit("\tctx.EmitMovRegImm64(%s.Reg2, uint64(len(%s.Imm.String())))", dv, resolved.goVar)
					g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
					g.emit("\tctx.BindReg(%s.Reg2, &%s)", dv, dv)
					g.emit("} else {")
					g.emit("\t%s = %s", dv, resolved.goVar)
					g.emit("}")
					return genVal{goVar: dv, isDesc: true, marker: "_gostring"}
				}
				xVal := materializeString(v.X)
				yVal := materializeString(v.Y)
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{%s, %s}, 1)", dv, xVal.goVar, yVal.goVar)
				g.emit("ctx.EmitAndRegImm32(%s.Reg, 1)", dv)
				if v.Op == token.NEQ {
					g.emit("ctx.EmitCmpRegImm32(%s.Reg, 0)", dv)
					g.emit("ctx.EmitSetcc(%s.Reg, CondEqual)", dv)
				}
				g.emit("%s.Type = tagBool", dv)
				g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
		}
		xVal := g.resolveValue(v.X)
		directResultMarker := g.directResultPayloads[name]
		directResult := directResultMarker == "_newint" || directResultMarker == "_newfloat"
		resultTargetVar := ""
		if directResult {
			resultTargetVar = g.allocTemp("resultTarget")
			g.emit("%s := false", resultTargetVar)
		}
		// Check if v.X has more remaining uses (excluding this one).
		// If so, destructive operations must copy before modifying.
		xMultiUse := false
		if _, isConst := v.X.(*ssa.Const); !isConst {
			xMultiUse = g.ssaValueUsesRemaining(v.X.Name()) > 1
		}
		if v.Op == token.SUB || v.Op == token.ADD {
			// Conservative for + and -: avoid destructive updates on x to prevent
			// alias/overwrite corner cases when SSA values are reused later.
			xMultiUse = true
		}
		if g.usedByOutgoingPhi(v.X.Name()) {
			xMultiUse = true
		}
		if g.storageMode {
			// Conservative in storage emitters: SSA value reuse across phi edges
			// and inlined blocks is subtle; prefer non-destructive BinOps.
			xMultiUse = true
		}
		if g.isFieldCachedDesc(xVal.goVar) {
			xMultiUse = true
		}
		if directResult {
			// Keep the input intact until the result placement is selected. The
			// selected register can then be the caller's result.Reg2.
			xMultiUse = true
		}
		if floatAluOp := floatAluEmitFunc(v.Op); floatAluOp != "" && isFloat64Type(v.Type()) && isFloat64Type(v.X.Type()) && isFloat64Type(v.Y.Type()) {
			dv := g.allocDesc()
			goOp := goOpStr(v.Op)
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal, ok := constFloat64Value(c.Value)
				if !ok {
					panic(fmt.Sprintf("unsupported float arithmetic const kind: %s", c))
				}
				bits := math.Float64bits(cmpVal)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%s.Imm.Float() %s %g)}", dv, xVal.goVar, goOp, cmpVal)
				g.emit("} else {")
				if xMultiUse {
					g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directResult, xVal.goVar+".Reg")
					g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
					g.emit("\tctx.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.%s(scratch, RegR11)", floatAluOp)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				} else {
					g.emit("\tctx.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.%s(%s.Reg, RegR11)", floatAluOp, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("ctx.EnsureDescsTogether(&%s, &%s)", xVal.goVar, yVal.goVar)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%s.Imm.Float() %s %s.Imm.Float())}", dv, xVal.goVar, goOp, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directResult, yVal.goVar+".Reg")
				g.emit("\t_, xBits := %s.Imm.RawWords()", xVal.goVar)
				g.emit("\tctx.EmitMovRegImm64(scratch, xBits)")
				g.emit("\tctx.%s(scratch, %s.Reg)", floatAluOp, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directResult, xVal.goVar+".Reg")
					g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.%s(scratch, RegR11)", floatAluOp)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				} else {
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.%s(%s.Reg, RegR11)", floatAluOp, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocResultAwareReg(copyReg, resultTargetVar, "\t", directResult, xVal.goVar+".Reg", yVal.goVar+".Reg")
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.%s(%s, %s.Reg)", floatAluOp, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.%s(%s.Reg, %s.Reg)", floatAluOp, xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			if directResult {
				g.emit("if %s && %s.Loc == LocReg { ctx.BindReg(result.Reg2, &result) }", resultTargetVar, dv)
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true, resultTargetVar: resultTargetVar}
			break
		}
		xSigned, _, xIsInt := intTypeInfo(v.X.Type())
		ySigned, _, yIsInt := intTypeInfo(v.Y.Type())
		resSigned, resBits, resIsInt := intTypeInfo(v.Type())
		narrowUnsigned := resIsInt && !resSigned && resBits > 0 && resBits < 64
		cc := opToCC(v.Op)
		unsignedCompare := cc != "" && xIsInt && yIsInt && !xSigned && !ySigned
		if unsignedCompare {
			cc = opToCCUnsigned(v.Op)
		}
		goOp := goOpStr(v.Op)
		if cc != "" {
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok && c.Value == nil && (v.Op == token.EQL || v.Op == token.NEQ) {
				nilComparable := false
				switch v.X.Type().Underlying().(type) {
				case *types.Pointer, *types.Interface, *types.Slice, *types.Map, *types.Chan, *types.Signature:
					nilComparable = true
				}
				if nilComparable {
					g.emit("var %s JITValueDesc", dv)
					g.emit("if %s.Loc == LocImm {", xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.IsNil() %s true)}", dv, xVal.goVar, goOp)
					g.emit("} else {")
					g.emit("\tctx.EnsureDesc(&%s)", xVal.goVar)
					g.emit("\tif %s.Loc != LocReg && %s.Loc != LocRegPair && %s.Loc != LocRegTriple { panic(\"jit: nil comparison requires a register value\") }", xVal.goVar, xVal.goVar, xVal.goVar)
					rv := g.allocReg()
					g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
					g.emit("\tctx.EmitCmpRegImm32(%s.Reg, 0)", xVal.goVar)
					g.emit("\tctx.EmitSetcc(%s, %s)", rv, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
					g.emit("}")
					g.vals[name] = genVal{goVar: dv, isDesc: true}
					break
				}
			}
			if sbx, okx := v.X.Type().Underlying().(*types.Basic); okx && sbx.Kind() == types.String {
				if sby, oky := v.Y.Type().Underlying().(*types.Basic); oky && sby.Kind() == types.String {
					if c, ok := v.Y.(*ssa.Const); ok {
						s := constant.StringVal(c.Value)
						if s == "" && (v.Op == token.EQL || v.Op == token.NEQ) {
							g.emit("var %s JITValueDesc", dv)
							g.emit("if %s.Loc == LocImm {", xVal.goVar)
							g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.String() %s \"\")}", dv, xVal.goVar, goOp)
							g.emit("} else if %s.Loc == LocRegPair {", xVal.goVar)
							rv := g.allocReg()
							g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
							g.emit("\tctx.EmitCmpRegImm32(%s.Reg2, 0)", xVal.goVar)
							g.emit("\tctx.EmitSetcc(%s, %s)", rv, cc)
							g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
							g.emit("} else {")
							g.emit("\tpanic(\"jit: string compare expects LocRegPair or LocImm\")")
							g.emit("}")
							g.vals[name] = genVal{goVar: dv, isDesc: true}
							break
						}
					}
					panic(fmt.Sprintf("unsupported compare const kind: %s", v.Y))
				}
			}
			if isFloat64Type(v.X.Type()) && isFloat64Type(v.Y.Type()) {
				if c, ok := v.Y.(*ssa.Const); ok {
					cmpVal, ok := constFloat64Value(c.Value)
					if !ok {
						panic(fmt.Sprintf("unsupported compare const kind: %s", c))
					}
					bits := math.Float64bits(cmpVal)
					g.emit("var %s JITValueDesc", dv)
					g.emit("if %s.Loc == LocImm {", xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Float() %s %g)}", dv, xVal.goVar, goOp, cmpVal)
					g.emit("} else {")
					rv := g.allocReg()
					g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
					g.emit("\tctx.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.EmitCmpFloat64Setcc(%s, %s.Reg, RegR11, %s)", rv, xVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
					g.emit("}")
				} else {
					yVal := g.resolveValue(v.Y)
					g.emit("ctx.EnsureDescsTogether(&%s, &%s)", xVal.goVar, yVal.goVar)
					g.emit("var %s JITValueDesc", dv)
					g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Float() %s %s.Imm.Float())}", dv, xVal.goVar, goOp, yVal.goVar)
					g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
					rv := g.allocReg()
					g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.EmitCmpFloat64Setcc(%s, %s.Reg, RegR11, %s)", rv, xVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
					g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg)", rv2, yVal.goVar)
					g.emit("\t_, xBits := %s.Imm.RawWords()", xVal.goVar)
					g.emit("\tctx.EmitMovRegImm64(RegR11, xBits)")
					g.emit("\tctx.EmitCmpFloat64Setcc(%s, RegR11, %s.Reg, %s)", rv2, yVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv2)
					g.emit("} else {")
					rv3 := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", rv3, xVal.goVar, yVal.goVar)
					g.emit("\tctx.EmitCmpFloat64Setcc(%s, %s.Reg, %s.Reg, %s)", rv3, xVal.goVar, yVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv3)
					g.emit("}")
				}
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal, ok := constInt64Value(c.Value)
				if !ok {
					panic(fmt.Sprintf("unsupported compare const kind: %s", c))
				}
				// Constant-fold if x is LocImm
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				if unsignedCompare {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(%s.Imm.Int()) %s uint64(0x%x))}", dv, xVal.goVar, goOp, uint64(cmpVal))
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Int() %s %d)}", dv, xVal.goVar, goOp, cmpVal)
				}
				g.emit("} else {")
				// Fresh register for result — CMP is non-destructive, SetCC writes only the target.
				// Protect xVal.Reg when multi-use: AllocReg must not return xVal.Reg (SetCC would clobber it).
				rv := g.allocReg()
				g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
				if fitsInt32(cmpVal) {
					g.emit("\tctx.EmitCmpRegImm32(%s.Reg, %d)", xVal.goVar, cmpVal)
				} else {
					g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
					g.emit("\tctx.EmitCmpInt64(%s.Reg, RegR11)", xVal.goVar)
				}
				g.emit("\tctx.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("ctx.EnsureDescsTogether(&%s, &%s)", xVal.goVar, yVal.goVar)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				if unsignedCompare {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(%s.Imm.Int()) %s uint64(%s.Imm.Int()))}", dv, xVal.goVar, goOp, yVal.goVar)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOp, yVal.goVar)
				}
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				// y is imm, x is reg → CmpRegImm32. Protect xVal.Reg when multi-use.
				rv := g.allocReg()
				g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
				g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
				g.emit("\t\tctx.EmitCmpRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
				g.emit("\t} else {")
				g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
				g.emit("\t\tctx.EmitCmpInt64(%s.Reg, RegR11)", xVal.goVar)
				g.emit("\t}")
				g.emit("\tctx.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				// x is imm, y is reg → materialize x, CMP
				rv2 := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv2)
				g.emit("\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.EmitCmpInt64(RegR11, %s.Reg)", yVal.goVar)
				g.emit("\tctx.EmitSetcc(%s, %s)", rv2, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv2)
				g.emit("} else {")
				// Both regs: protect xVal.Reg when multi-use (SetCC would clobber if rv3==xVal.Reg).
				rv3 := g.allocReg()
				g.emitAllocRegExcept(rv3, "\t", xMultiUse, xVal)
				g.emit("\tctx.EmitCmpInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
				g.emit("\tctx.EmitSetcc(%s, %s)", rv3, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv3)
				g.emit("}")
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if aluOp := aluEmitFunc(v.Op); aluOp != "" {
			// Arithmetic BinOp: ADD, SUB, MUL
			dv := g.allocDesc()
			directIntResult := directResultMarker == "_newint"
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal, ok := constInt64Value(c.Value)
				if !ok {
					panic(fmt.Sprintf("unsupported arithmetic const kind: %s", c))
				}
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %s %d)}", dv, xVal.goVar, goOpStr(v.Op), cmpVal)
				g.emit("} else {")
				if xMultiUse {
					// x is needed again → result must go into a fresh register
					if v.Op == token.SUB {
						// SUB is non-commutative: copy x, then subtract const
						g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg")
						g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						if fitsInt32(cmpVal) {
							g.emit("\tctx.EmitSubRegImm32(scratch, int32(%d))", cmpVal)
						} else {
							g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
							g.emit("\tctx.EmitSubInt64(scratch, RegR11)")
						}
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					} else {
						// ADD/MUL: commutative, order doesn't matter
						g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg")
						g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						if v.Op == token.MUL {
							g.emitMulConstOnReg("scratch", cmpVal, "\t")
						} else if fitsInt32(cmpVal) {
							g.emit("\tctx.EmitAddRegImm32(scratch, int32(%d))", cmpVal)
						} else {
							g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
							g.emit("\tctx.%s(scratch, RegR11)", aluOp)
						}
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					}
				} else {
					// x is consumed; prefer immediate-form ALU to avoid materializing constants in a temp register.
					if v.Op == token.MUL {
						g.emitMulConstOnReg(fmt.Sprintf("%s.Reg", xVal.goVar), cmpVal, "\t")
					} else if fitsInt32(cmpVal) {
						switch v.Op {
						case token.ADD:
							g.emit("\tctx.EmitAddRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						case token.SUB:
							g.emit("\tctx.EmitSubRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						default:
							g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
							g.emit("\tctx.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
						}
					} else {
						g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
						g.emit("\tctx.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("ctx.EnsureDescsTogether(&%s, &%s)", xVal.goVar, yVal.goVar)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOpStr(v.Op), yVal.goVar)
				// Identity optimizations: ADD/SUB 0 is no-op
				if v.Op == token.ADD || v.Op == token.SUB {
					// y is LocImm 0 → x + 0 = x, x - 0 = x
					g.emit("} else if %s.Loc == LocImm && %s.Imm.Int() == 0 {", yVal.goVar, yVal.goVar)
					if xMultiUse {
						copyReg := g.allocReg()
						g.emitAllocResultAwareReg(copyReg, resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg")
						g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
					} else {
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
					}
				}
				if v.Op == token.ADD {
					// x is LocImm 0 → 0 + y = y (commutative)
					g.emit("} else if %s.Loc == LocImm && %s.Imm.Int() == 0 {", xVal.goVar, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, yVal.goVar)
				}
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				// x is const, y is reg → materialize x into scratch, ALU (result in scratch)
				g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directIntResult, yVal.goVar+".Reg")
				g.emit("\tctx.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.%s(scratch, %s.Reg)", aluOp, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				// y is const, x is reg → use R11 for constant (result in x.Reg or scratch)
				if xMultiUse {
					if v.Op == token.SUB {
						// SUB is non-commutative: copy x, then subtract y
						g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg")
						g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
						g.emit("\t\tctx.EmitSubRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t} else {")
						g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.EmitSubInt64(scratch, RegR11)")
						g.emit("\t}")
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					} else {
						// ADD/MUL: commutative, order doesn't matter
						g.emitAllocResultAwareReg("scratch", resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg")
						g.emit("\tctx.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
						if v.Op == token.ADD {
							g.emit("\t\tctx.EmitAddRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						} else if v.Op == token.MUL {
							g.emit("\t\tctx.EmitImulRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						} else {
							g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
							g.emit("\t\tctx.%s(scratch, RegR11)", aluOp)
						}
						g.emit("\t} else {")
						g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.%s(scratch, RegR11)", aluOp)
						g.emit("\t}")
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					}
				} else {
					// x consumed, y constant: immediate-form ALU when possible.
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					if v.Op == token.ADD {
						g.emit("\t\tctx.EmitAddRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else if v.Op == token.SUB {
						g.emit("\t\tctx.EmitSubRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else if v.Op == token.MUL {
						g.emit("\t\tctx.EmitImulRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else {
						g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					}
					g.emit("\t} else {")
					g.emit("\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\tctx.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocResultAwareReg(copyReg, resultTargetVar, "\t", directIntResult, xVal.goVar+".Reg", yVal.goVar+".Reg")
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.%s(%s, %s.Reg)", aluOp, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.%s(%s.Reg, %s.Reg)", aluOp, xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			}
			// Neutralize xVal if its register was transferred to the result (destructive ALU)
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			if directIntResult {
				g.emit("if %s && %s.Loc == LocReg { ctx.BindReg(result.Reg2, &result) }", resultTargetVar, dv)
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true, resultTargetVar: resultTargetVar}
		} else if v.Op == token.QUO {
			// Integer division: uses SHR for power-of-2, IDIV otherwise
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok {
				divisor := c.Int64()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() / %d)}", dv, xVal.goVar, divisor)
				g.emit("} else {")
				if xMultiUse {
					// Copy to fresh register (xVal is needed again)
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						shift := 0
						for d := divisor; d > 1; d >>= 1 {
							shift++
						}
						g.emit("\tctx.EmitShrRegImm8(%s, %d)", copyReg, shift)
					} else {
						g.emit("\tctx.EmitIdivRegImm(%s, %d)", copyReg, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						shift := 0
						for d := divisor; d > 1; d >>= 1 {
							shift++
						}
						g.emit("\tctx.EmitShrRegImm8(%s.Reg, %d)", xVal.goVar, shift)
					} else {
						g.emit("\tctx.EmitIdivRegImm(%s.Reg, %d)", xVal.goVar, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() / %s.Imm.Int())}", dv, xVal.goVar, yVal.goVar)
				g.emit("} else {")
				g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITIntDiv), []JITValueDesc{%s, %s}, 1)", dv, xVal.goVar, yVal.goVar)
				g.emit("}")
			}
			// Neutralize xVal if its register was transferred to the result
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.REM {
			// Integer modulo
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok {
				divisor := c.Int64()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %% %d)}", dv, xVal.goVar, divisor)
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						g.emit("\tctx.EmitAndRegImm32(%s, %d)", copyReg, divisor-1)
					} else {
						g.emit("\tctx.EmitIremRegImm(%s, %d)", copyReg, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						g.emit("\tctx.EmitAndRegImm32(%s.Reg, %d)", xVal.goVar, divisor-1)
					} else {
						g.emit("\tctx.EmitIremRegImm(%s.Reg, %d)", xVal.goVar, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %% %s.Imm.Int())}", dv, xVal.goVar, yVal.goVar)
				g.emit("} else {")
				g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{%s, %s}, 1)", dv, xVal.goVar, yVal.goVar)
				g.emit("}")
			}
			// Neutralize xVal if its register was transferred to the result
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.SHL || v.Op == token.SHR {
			// Shift operations
			dv := g.allocDesc()
			emitFn := "EmitShlRegCl"
			immFn := "EmitShlRegImm8"
			goShOp := "<<"
			if v.Op == token.SHR {
				emitFn = "EmitShrRegCl"
				immFn = "EmitShrRegImm8"
				goShOp = ">>"
			}
			if c, ok := v.Y.(*ssa.Const); ok {
				shiftAmt := c.Int64()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(%s.Imm.Int()) %s %d))}", dv, xVal.goVar, goShOp, shiftAmt)
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.%s(%s, %d)", immFn, copyReg, shiftAmt)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.%s(%s.Reg, %d)", immFn, xVal.goVar, shiftAmt)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(%s.Imm.Int()) %s uint64(%s.Imm.Int())))}", dv, xVal.goVar, goShOp, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				// y (shift amount) is const
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.%s(%s, uint8(%s.Imm.Int()))", immFn, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.%s(%s.Reg, uint8(%s.Imm.Int()))", immFn, xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				// Variable shift: must use CL register.
				// RCX may be allocated for another value (e.g. phi register);
				// save/restore it around the CL usage.
				g.emit("\t{")
				g.emit("\t\tshiftSrc := %s.Reg", xVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t\t", true, xVal)
					g.emit("\t\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t\tshiftSrc = %s", copyReg)
				} else {
					g.emit("\t\tif shiftSrc == RegRCX {")
					g.emit("\t\t\tnewReg := ctx.AllocReg()")
					g.emit("\t\t\tctx.EmitMovRegReg(newReg, RegRCX)")
					g.emit("\t\t\tshiftSrc = newReg")
					g.emit("\t\t}")
				}
				g.emit("\t\trcxUsed := ctx.FreeRegs & (1 << uint(RegRCX)) == 0 && %s.Reg != RegRCX", yVal.goVar)
				g.emit("\t\tif rcxUsed {")
				g.emit("\t\t\tctx.EmitMovRegReg(RegR11, RegRCX)") // save RCX in scratch R11
				g.emit("\t\t}")
				g.emit("\t\tif %s.Reg != RegRCX {", yVal.goVar)
				g.emit("\t\t\tctx.EmitMovRegReg(RegRCX, %s.Reg)", yVal.goVar)
				g.emit("\t\t}")
				g.emit("\t\tctx.%s(shiftSrc)", emitFn)
				g.emit("\t\tif rcxUsed {")
				g.emit("\t\t\tctx.EmitMovRegReg(RegRCX, RegR11)") // restore RCX from R11
				g.emit("\t\t}")
				g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: shiftSrc}", dv)
				g.emit("\t}")
				g.emit("}")
			}
			// Neutralize xVal if its register was transferred to the result
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.AND {
			// Bitwise AND
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal := c.Int64()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() & %d)}", dv, xVal.goVar, cmpVal)
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if fitsInt32(cmpVal) {
						g.emit("\tctx.EmitAndRegImm32(%s, int32(%d))", copyReg, cmpVal)
					} else {
						g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
						g.emit("\tctx.EmitAndInt64(%s, RegR11)", copyReg)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if fitsInt32(cmpVal) {
						g.emit("\tctx.EmitAndRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
					} else {
						g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
						g.emit("\tctx.EmitAndInt64(%s.Reg, RegR11)", xVal.goVar)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() & %s.Imm.Int())}", dv, xVal.goVar, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				g.emitAllocRegExcept("scratch", "\t", true, yVal)
				g.emit("\tctx.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.EmitAndInt64(scratch, %s.Reg)", yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.EmitAndRegImm32(%s, int32(%s.Imm.Int()))", copyReg, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.EmitAndInt64(%s, RegR11)", copyReg)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.EmitAndRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.EmitAndInt64(%s.Reg, RegR11)", xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.EmitAndInt64(%s, %s.Reg)", copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.EmitAndInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			}
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.OR {
			// Bitwise OR
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal := c.Int64()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() | %d)}", dv, xVal.goVar, cmpVal)
				g.emit("} else if %d == 0 {", cmpVal)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if fitsInt32(cmpVal) {
						g.emit("\tctx.EmitOrRegImm32(%s, int32(%d))", copyReg, cmpVal)
					} else {
						g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
						g.emit("\tctx.EmitOrInt64(%s, RegR11)", copyReg)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if fitsInt32(cmpVal) {
						g.emit("\tctx.EmitOrRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
					} else {
						g.emit("\tctx.EmitMovRegImm64(RegR11, 0x%x)", uint64(cmpVal))
						g.emit("\tctx.EmitOrInt64(%s.Reg, RegR11)", xVal.goVar)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() | %s.Imm.Int())}", dv, xVal.goVar, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm && %s.Imm.Int() == 0 {", xVal.goVar, xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm && %s.Imm.Int() == 0 {", yVal.goVar, yVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				g.emitAllocRegExcept("scratch", "\t", true, yVal)
				g.emit("\tctx.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.EmitOrInt64(scratch, %s.Reg)", yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.EmitOrRegImm32(%s, int32(%s.Imm.Int()))", copyReg, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.EmitOrInt64(%s, RegR11)", copyReg)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.EmitOrRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.EmitOrInt64(%s.Reg, RegR11)", xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.EmitOrInt64(%s, %s.Reg)", copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.EmitOrInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			}
			// Neutralize xVal if its register was transferred to the result
			if narrowUnsigned {
				g.emitNormalizeUnsignedNarrow(dv, resBits)
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("unsupported BinOp %s", v.Op))
		}

	case *ssa.Return:
		if g.inlineEndLabel != "" {
			// Inlined multi-block function: MOV result to designated register, JMP to end
			g.emitInlineReturn(v)
		} else if g.multiBlock {
			g.emitReturnMultiBlock(v)
		} else {
			g.emitReturnSingleBlock(v)
		}

	case *ssa.Phi:
		// Phi output locations are fixed stack slots. Keep descriptors on stack
		// and materialize into registers only at use sites.
		if g.bbClosureMode {
			// In recursive BB-closure mode, phi descriptors are initialized at BB
			// entry via resetAllPhiDescsToStack()+applyPhiStateOverlay.
			if gv, ok := g.vals[name]; ok && gv.isDesc {
				break
			}
		}
		if phiOff, ok := g.phiRegs[name]; ok {
			stackOff := "int32(" + phiOff + ")"
			if g.phiFrameFixup != "" && !g.storageMode {
				stackOff = "int32(" + g.phiFrameFixup + ")+" + stackOff
			}
			if g.phiTriple[name] {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: %s}", dv, stackOff)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
			} else if g.phiPair[name] {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: %s}", dv, stackOff)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: %s}", dv, stackOff)
				marker := ""
				if _, ok := v.Type().Underlying().(*types.Signature); ok {
					marker = "_gofunc"
				}
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: marker != ""}
			}
		} else {
			panic(fmt.Sprintf("phi %s has no allocated stack slot", name))
		}

	case *ssa.If:
		if g.bbClosureMode && !g.forceLegacyCFG {
			g.emitIfClosure(v)
			break
		}
		thenBB := v.Block().Succs[0].Index
		elseBB := v.Block().Succs[1].Index
		// SSA-constant condition: emit only taken edge and enqueue exactly one BB.
		if c, ok := v.Cond.(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.Bool {
			takenBB := elseBB
			takenSuccPos := 1
			if constant.BoolVal(c.Value) {
				takenBB = thenBB
				takenSuccPos = 0
			}
			g.emitEdgePhiMoves(takenBB, takenSuccPos)
			// Phase 2 pruning: only render the reachable branch.
			// If the target BB is not rendered yet, enqueue it next and fall through
			// without emitting an unconditional jump.
			// For already-rendered targets (backedge/cross-edge), emit a direct jump.
			if g.bbDone[g.scopedBBID(takenBB)] {
				if lbl, ok := g.bbLabels[g.scopedBBID(takenBB)]; ok {
					g.emit("ctx.EmitJmp(%s)", lbl)
				} else if posVar, ok := g.bbPosVars[g.scopedBBID(takenBB)]; ok {
					g.emit("ctx.EmitJmpToPos(%s)", posVar)
				} else {
					panic(fmt.Sprintf("jitgen: rendered BB%d requires jump label", takenBB))
				}
			} else {
				g.enqueueBBFront(takenBB)
			}
			break
		}
		cond := g.vals[v.Cond.Name()]
		if !cond.isDesc {
			panic(fmt.Sprintf("If: %s unimplemented for %s.Loc (descriptor missing: isDesc=false, goVar=%s, marker=%q; expected LocImm|LocReg)",
				v, v.Cond.Name(), cond.goVar, cond.marker))
		}
		// Materialize branch conditions before emitting cmp/jcc.
		// Phi-backed conditions may be LocStack at BB entry.
		condVar := g.allocDesc()
		g.emit("%s := %s", condVar, cond.goVar)
		g.emit("ctx.EnsureDesc(&%s)", condVar)
		g.emit("if %s.Loc != LocImm && %s.Loc != LocReg {", condVar, condVar)
		g.emit("\tpanic(\"jit: If condition is neither LocImm nor LocReg\")")
		g.emit("}")
		// Ensure labels for both targets
		thenLbl := g.ensureBBLabel(thenBB)
		elseLbl := g.ensureBBLabel(elseBB)
		// Reserve edge-helper labels (both edges become explicit helper blocks)
		thenEdgeLbl := g.allocLabel()
		elseEdgeLbl := g.allocLabel()
		g.emit("%s := ctx.ReserveLabel()", thenEdgeLbl)
		g.emit("%s := ctx.ReserveLabel()", elseEdgeLbl)

		// Phase 3 step: JIT-time constant If pruning.
		// When condVar is LocImm during emitter execution, emit only the taken
		// edge helper and enqueue only one successor BB.
		g.emit("if %s.Loc == LocImm {", condVar)
		g.emit("\tif %s.Imm.Bool() {", condVar)
		g.emit("\t\tctx.MarkLabel(%s)", thenEdgeLbl)
		g.emitEdgePhiMoves(thenBB, 0)
		g.emit("\t\tctx.EmitJmp(%s)", thenLbl)
		g.emit("\t} else {")
		g.emit("\t\tctx.MarkLabel(%s)", elseEdgeLbl)
		g.emitEdgePhiMoves(elseBB, 1)
		g.emit("\t\tctx.EmitJmp(%s)", elseLbl)
		g.emit("\t}")
		g.emit("} else {")
		// Runtime: CMP + JNE to then-edge helper, otherwise else-edge helper.
		g.emit("\tctx.EmitCmpRegImm32(%s.Reg, 0)", condVar)
		g.emit("\tctx.EmitJump(CondNotEqual, %s)", thenEdgeLbl)
		g.emit("\tctx.EmitJmp(%s)", elseEdgeLbl)
		// Dynamic condition: both helper edges are reachable.
		g.emit("\tctx.MarkLabel(%s)", thenEdgeLbl)
		g.emitEdgePhiMoves(thenBB, 0)
		g.emit("\tctx.EmitJmp(%s)", thenLbl)
		g.emit("\tctx.MarkLabel(%s)", elseEdgeLbl)
		g.emitEdgePhiMoves(elseBB, 1)
		g.emit("\tctx.EmitJmp(%s)", elseLbl)
		g.emit("}")
		// Generator scheduling mirrors the emitted pruning above.
		// For LocImm-only execution paths we enqueue a single successor; for
		// dynamic conditions we must keep both successors reachable while
		// preferring the more specialized successor as immediate fallthrough.
		if immCond, ok := v.Cond.(*ssa.Const); ok && immCond.Value != nil && immCond.Value.Kind() == constant.Bool {
			if constant.BoolVal(immCond.Value) {
				g.enqueueBBFront(thenBB)
			} else {
				g.enqueueBBFront(elseBB)
			}
		} else {
			if g.preferredIfFallthrough(thenBB, elseBB) == thenBB {
				g.enqueueBB(elseBB)
				g.enqueueBBFront(thenBB)
			} else {
				g.enqueueBB(thenBB)
				g.enqueueBBFront(elseBB)
			}
		}

	case *ssa.Jump:
		if g.bbClosureMode && !g.forceLegacyCFG {
			g.emitJumpClosure(v)
			break
		}
		targetBB := v.Block().Succs[0].Index
		g.emitEdgePhiMoves(targetBB, 0)
		// Phase 2 pruning: for forward/unrendered targets, render target next and
		// fall through without emitting an unconditional jump.
		// If target is already rendered (backedge/cross-edge), emit a direct jump.
		if g.bbDone[g.scopedBBID(targetBB)] {
			if lbl, ok := g.bbLabels[g.scopedBBID(targetBB)]; ok {
				g.emit("ctx.EmitJmp(%s)", lbl)
			} else if posVar, ok := g.bbPosVars[g.scopedBBID(targetBB)]; ok {
				g.emit("ctx.EmitJmpToPos(%s)", posVar)
			} else {
				panic(fmt.Sprintf("jitgen: rendered BB%d requires jump label", targetBB))
			}
		} else {
			g.enqueueBBFront(targetBB)
		}

	case *ssa.Convert:
		src := g.resolveValue(v.X)
		if src.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
		}
		dv := g.allocDesc()
		srcType := v.X.Type().Underlying()
		dstType := v.Type().Underlying()
		srcBasic, srcOk := srcType.(*types.Basic)
		dstBasic, dstOk := dstType.(*types.Basic)
		if isNoopPointerConvert(v.X.Type(), v.Type()) {
			srcName := v.X.Name()
			if _, isConst := v.X.(*ssa.Const); !isConst {
				g.ssaAliases[name] = srcName
				// Merge convert result's uses into source's refcount
				g.refCounts[srcName] += g.refCounts[name]
				delete(g.refCounts, name)
			}
			g.vals[name] = src
			break
		}
		if srcOk && dstOk && isIntegerKind(srcBasic.Kind()) && isIntegerKind(dstBasic.Kind()) {
			srcSigned, srcBits, srcInfoOK := intTypeInfo(v.X.Type())
			dstSigned, dstBits, dstInfoOK := intTypeInfo(v.Type())
			if !srcInfoOK || !dstInfoOK {
				panic(fmt.Sprintf("unsupported integer Convert %s → %s", v.X.Type(), v.Type()))
			}

			// Exact same integer representation: alias source.
			if srcSigned == dstSigned && srcBits == dstBits {
				srcName := v.X.Name()
				if _, isConst := v.X.(*ssa.Const); !isConst {
					g.ssaAliases[name] = srcName
					// Merge convert result's uses into source's refcount
					g.refCounts[srcName] += g.refCounts[name]
					delete(g.refCounts, name)
				}
				g.vals[name] = src
				if !src.isDesc {
					// Bare register → wrap in JITValueDesc
					g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, src.goVar)
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				}
				break
			}

			srcTy := intTypeName(srcSigned, srcBits)
			dstTy := intTypeName(dstSigned, dstBits)
			if srcTy == "" || dstTy == "" {
				panic(fmt.Sprintf("unsupported integer Convert %s → %s", v.X.Type(), v.Type()))
			}

			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			// Materialize with explicit source+destination casts to preserve wrap/sign semantics.
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s(%s(%s.Imm.Int()))))}", dv, dstTy, srcTy, src.goVar)
			g.emit("} else {")
			tmpReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", tmpReg)
			g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", tmpReg, src.goVar)
			// Normalize source width/sign first.
			if srcBits > 0 && srcBits < 64 {
				shift := 64 - srcBits
				g.emit("\tctx.EmitShlRegImm8(%s, %d)", tmpReg, shift)
				if srcSigned {
					g.emit("\tctx.EmitSarRegImm8(%s, %d)", tmpReg, shift)
				} else {
					g.emit("\tctx.EmitShrRegImm8(%s, %d)", tmpReg, shift)
				}
			}
			// Then normalize destination width/sign for actual conversion target.
			if dstBits > 0 && dstBits < 64 {
				shift := 64 - dstBits
				g.emit("\tctx.EmitShlRegImm8(%s, %d)", tmpReg, shift)
				if dstSigned {
					g.emit("\tctx.EmitSarRegImm8(%s, %d)", tmpReg, shift)
				} else {
					g.emit("\tctx.EmitShrRegImm8(%s, %d)", tmpReg, shift)
				}
			}
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, tmpReg)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if srcOk && dstOk && isIntegerKind(srcBasic.Kind()) && dstBasic.Kind() == types.Float64 {
			// int → float64: emit CVTSI2SD
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(%s.Imm.Int()))}", dv, src.goVar)
			g.emit("} else {")
			g.emit("\tctx.EmitCvtInt64ToFloat64(RegX0, %s.Reg)", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, src.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if srcOk && dstOk && srcBasic.Kind() == types.Float64 && isIntegerKind(dstBasic.Kind()) {
			// float64 → int: truncate toward zero (Go conversion semantics)
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.Imm.Float()))}", dv, src.goVar)
			g.emit("} else {")
			tmpReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", tmpReg)
			g.emit("\tctx.EmitCvtFloatBitsToInt64(%s, %s.Reg)", tmpReg, src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, tmpReg)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if srcBasic != nil && srcBasic.Kind() == types.String && isByteSliceType(dstType) {
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
			callResults := g.allocTemp("callResults")
			g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{%s}, []uint8{3}, []uint8{1})", callResults, src.goVar)
			g.emit("%s := %s[0]", dv, callResults)
			g.emit("%s.Type = tagSlice", dv)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
		} else if isByteSliceType(srcType) && dstBasic != nil && dstBasic.Kind() == types.String {
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
			callResults := g.allocTemp("callResults")
			g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{%s}, []uint8{2}, []uint8{1})", callResults, src.goVar)
			g.emit("%s := %s[0]", dv, callResults)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
		} else {
			panic(fmt.Sprintf("unsupported Convert %s → %s", v.X.Type(), v.Type()))
		}

	case *ssa.ChangeType:
		// ChangeType preserves the machine representation (for example
		// time.Month -> int); only the Go static type changes.
		g.vals[name] = g.resolveValue(v.X)

	case *ssa.Alloc:
		if ptr, ok := v.Type().Underlying().(*types.Pointer); ok {
			if isScmerType(ptr.Elem()) {
				dataReg := g.allocReg()
				auxReg := g.allocReg()
				dv := g.allocDesc()
				g.emit("%s := ctx.AllocReg()", dataReg)
				g.emit("%s := ctx.AllocRegExcept(%s)", auxReg, dataReg)
				g.emit("ctx.EmitMovRegImm64(%s, 0)", dataReg)
				g.emit("ctx.EmitMovRegImm64(%s, 0)", auxReg)
				g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, dataReg, auxReg)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_scmer_struct", cellName: name, cellScope: g.bbScope}
				break
			}
			if array, ok := ptr.Elem().Underlying().(*types.Array); ok && isScmerType(array.Elem()) {
				elemSize := types.SizesFor("gc", "amd64").Sizeof(array.Elem())
				stackBase := g.allocTemp("stackArray")
				g.emit("%s := ctx.AllocStack(int32(%d))", stackBase, elemSize*array.Len())
				g.emit("_ = %s", stackBase)
				g.vals[name] = genVal{
					goVar:  stackBase,
					marker: fmt.Sprintf("_stackarray:%d:%d", elemSize, array.Len()),
				}
				break
			}
			if _, ok := ptr.Elem().Underlying().(*types.Array); ok {
				dv := g.allocDesc()
				typeExpr := g.sourceTypeExpr(ptr.Elem())
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func() *%s { return new(%s) }), nil, 1)", dv, typeExpr, typeExpr)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_goarrayptr", aggregateType: ptr.Elem(), pinAcrossBlock: true}
				break
			}
		}
		if ptr, ok := v.Type().Underlying().(*types.Pointer); ok && !allocIsClosureCell(v) {
			dv := g.allocDesc()
			typeExpr := g.sourceTypeExpr(ptr.Elem())
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func() *%s { return new(%s) }), nil, 1)", dv, typeExpr, typeExpr)
			g.emit("ctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_goptr", pinAcrossBlock: true}
			break
		}
		// Non-array allocations are currently closure cells. Their value is
		// forwarded by Store and consumed by MakeClosure without runtime storage.
		g.vals[name] = genVal{marker: "_alloc", cellName: name, cellScope: g.bbScope}

	case *ssa.Store:
		dst := g.vals[v.Addr.Name()]
		if global, ok := v.Addr.(*ssa.Global); ok {
			expr, resolved := g.globalSourceExpr(global)
			if !resolved {
				panic(fmt.Sprintf("unresolved global Store: %s", v))
			}
			src := g.resolveValue(v.Val)
			valueType := g.sourceTypeExpr(v.Val.Type())
			g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(value %s) { %s = value }), []JITValueDesc{%s})", valueType, expr, src.goVar)
		} else if dst.marker == "_goarrayelem" {
			src := g.resolveValue(v.Val)
			arrayExpr := g.sourceTypeExpr(dst.aggregateType)
			elemExpr := g.sourceTypeExpr(dst.fieldType)
			if src.marker == "_aggregate_ptr" {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst *%s, index int, src *%s) { dst[index] = *src }), []JITValueDesc{%s, %s, %s})", arrayExpr, elemExpr, dst.goVar, dst.argIdxVar, src.goVar)
			} else {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst *%s, index int, value %s) { dst[index] = value }), []JITValueDesc{%s, %s, %s})", arrayExpr, elemExpr, dst.goVar, dst.argIdxVar, src.goVar)
			}
		} else if dst.marker == "_goarrayptr" {
			src := g.resolveValue(v.Val)
			valueType := g.sourceTypeExpr(v.Val.Type())
			if src.marker == "_aggregate_ptr" {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *%s) { *dst = *src }), []JITValueDesc{%s, %s})", valueType, dst.goVar, src.goVar)
			} else if src.marker == "_goarrayvalue" {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst *%s, src %s) { *dst = src }), []JITValueDesc{%s, %s})", valueType, valueType, dst.goVar, src.goVar)
			} else {
				panic(fmt.Sprintf("unsupported Go array Store: %s", v))
			}
		} else if dst.marker == "_goptr" {
			src := g.resolveValue(v.Val)
			valueType := g.sourceTypeExpr(v.Val.Type())
			if src.marker == "_aggregate_ptr" {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *%s) { *dst = *src }), []JITValueDesc{%s, %s})", valueType, dst.goVar, src.goVar)
			} else {
				g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(dst *%s, value %s) { *dst = value }), []JITValueDesc{%s, %s})", valueType, valueType, dst.goVar, src.goVar)
			}
		} else if strings.HasPrefix(dst.marker, "_fieldaddrlocal:") && dst.fieldBaseType != nil && dst.fieldType != nil {
			src := g.resolveValue(v.Val)
			baseType := g.sourceTypeExpr(dst.fieldBaseType)
			fieldType := g.sourceTypeExpr(dst.fieldType)
			g.emit("ctx.EnsureDesc(&%s)", dst.goVar)
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
			g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(base *%s, value %s) { base.%s = value }), []JITValueDesc{%s, %s})", baseType, fieldType, dst.fieldName, dst.goVar, src.goVar)
		} else if strings.HasPrefix(dst.marker, "_descfield:") {
			parts := strings.SplitN(dst.marker, ":", 3)
			fieldName := parts[1]
			base := parts[2]
			src := g.resolveValue(v.Val)
			if src.goVar == "" || !src.isDesc {
				panic(fmt.Sprintf("descriptor field Store has unresolved source %s (marker=%q desc=%t)", v.Val, src.marker, src.isDesc))
			}
			g.emit("ctx.EnsureDesc(&%s)", src.goVar)
			if fieldName == "ptr" {
				g.emit("ctx.EmitMovToReg(%s.Reg, %s)", base, src.goVar)
			} else {
				g.emit("ctx.EmitMovToReg(%s.Reg2, %s)", base, src.goVar)
			}
		} else if strings.HasPrefix(dst.marker, "_stackaddr:") {
			rewritten := g.rewriteSSAValue(v.Val)
			src, ok := g.vals[rewritten.Name()]
			if !ok {
				src = g.resolveValue(rewritten)
			} else if src.isDesc {
				// A full Scmer stack store accepts resident and stack-backed pairs.
				// Keep cross-block values in their stable homes instead of loading
				// them into registers that an inlined callback may subsequently use.
				g.emit("ctx.SyncDesc(&%s)", src.goVar)
			}
			if !isScmerType(v.Val.Type()) {
				panic(fmt.Sprintf("unsupported non-Scmer stack array Store: %s", v))
			}
			switch src.marker {
			case "_newbool":
				g.emit("ctx.EnsureDesc(&%s)", src.goVar)
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagBool, %s)", src.goVar, dst.offsetExpr)
			case "_newint":
				g.emit("ctx.EnsureDesc(&%s)", src.goVar)
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagInt, %s)", src.goVar, dst.offsetExpr)
			case "_newfloat":
				g.emit("ctx.EnsureDesc(&%s)", src.goVar)
				g.emit("ctx.EmitStoreTypedScmerToStack(%s, tagFloat, %s)", src.goVar, dst.offsetExpr)
			default:
				if src.marker == "_newargslice" {
					materialized := g.allocDesc()
					g.emit("%s := jitMaterializeVirtualSlice(ctx, %s, JITValueDesc{Loc: LocAny})", materialized, src.goVar)
					g.emit("ctx.EmitStoreScmerToStack(%s, %s)", materialized, dst.offsetExpr)
					g.emit("ctx.FreeDesc(&%s)", materialized)
				} else {
					g.emit("ctx.EmitStoreScmerToStack(%s, %s)", src.goVar, dst.offsetExpr)
				}
			}
		} else if dst.cellName != "" {
			// Storing to an allocation: just remember the stored value
			rewritten := g.rewriteSSAValue(v.Val)
			src, ok := g.vals[rewritten.Name()]
			if !ok {
				src = g.resolveValue(rewritten)
			} else if src.isDesc {
				// Aggregate producers such as append already have a stable stack
				// result. Keep that producer-selected location when it flows into a
				// captured cell; materializing it here would pin one loop iteration's
				// registers and disconnect subsequent writes to the same stack home.
				g.emit("ctx.SyncDesc(&%s)", src.goVar)
			}
			if isScmerType(v.Val.Type()) && (dst.marker == "_alloc" || dst.marker == "_scmer_cell") {
				stored := src
				materializedVar := ""
				if stored.marker == "_newargslice" {
					materialized := g.allocDesc()
					g.emit("%s := jitMaterializeVirtualSlice(ctx, %s, JITValueDesc{Loc: LocAny})", materialized, stored.goVar)
					stored = genVal{goVar: materialized, isDesc: true}
					materializedVar = materialized
				}
				if dst.marker == "_alloc" {
					pair := g.allocDesc()
					g.emit("%s := jitCopyScmerToPair(ctx, %s)", pair, stored.goVar)
					if materializedVar != "" {
						g.emit("ctx.FreeDesc(&%s)", materializedVar)
					}
					cellOff := g.allocTemp("scmerCellOff")
					cellDesc := g.allocDesc()
					g.emit("%s := ctx.AllocStack(16)", cellOff)
					g.emit("ctx.EmitStoreScmerToStack(%s, int32(%s))", pair, cellOff)
					g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: %s.Type, StackOff: int32(%s)}", cellDesc, pair, cellOff)
					src = genVal{goVar: cellDesc, isDesc: true, marker: "_scmer_cell", cellName: dst.cellName, cellScope: dst.cellScope, pinAcrossBlock: true}
					g.emit("ctx.FreeDesc(&%s)", pair)
				} else {
					g.emit("ctx.EmitCopyScmerToDesc(&%s, &%s)", dst.goVar, stored.goVar)
					if materializedVar != "" {
						g.emit("ctx.FreeDesc(&%s)", materializedVar)
					}
					src = dst
				}
			} else if dst.isDesc && src.isDesc && dst.marker == "_slice" && src.marker == "_slice" {
				g.emit("ctx.SyncDesc(&%s)", dst.goVar)
				g.emit("if %s.Loc != LocStackTriple { panic(\"jit: captured slice requires a stable stack home\") }", dst.goVar)
				g.emit("ctx.EmitCopyDescWords(&%s, &%s, 3)", dst.goVar, src.goVar)
				src = dst
			}
			if src.isDesc && src.marker == "_slice" {
				g.emit("ctx.StabilizeDescForControlFlow(&%s)", src.goVar)
			}
			src.cellName = dst.cellName
			src.cellScope = dst.cellScope
			g.vals[v.Addr.Name()] = src
		} else if strings.HasPrefix(dst.marker, "_sliceaddr:") {
			parts := strings.SplitN(dst.marker, ":", 3)
			elemSize := parts[1]
			sliceDescVar := g.overlayDescVar(parts[2], dst.deferredBaseSSA)
			idxDescVar := g.overlayDescVar(dst.argIdxVar, dst.deferredIndexSSA)
			rewritten := g.rewriteSSAValue(v.Val)
			src, ok := g.vals[rewritten.Name()]
			if !ok {
				src = g.resolveValue(rewritten)
			} else if src.isDesc {
				// EmitStoreScmerAt protects the computed address before it
				// materializes the value. Preserve a cross-block value's stable
				// stack home here so address generation cannot spill a stale copy.
				g.emit("ctx.SyncDesc(&%s)", src.goVar)
			}
			address := g.allocDesc()
			g.emit("%s := ctx.EmitSliceElementAddress(&%s, &%s, int32(%s))", address, sliceDescVar, idxDescVar, elemSize)
			g.emit("ctx.EmitStoreScmerAt(&%s, &%s)", address, src.goVar)
			g.emit("ctx.FreeDesc(&%s)", address)
			if dst.deferredIndexSSA != "" {
				g.useOperand(dst.deferredIndexSSA)
			}
		} else {
			panic(fmt.Sprintf("unsupported Store: %s", v))
		}

	case *ssa.MakeClosure:
		closureFn, ok := v.Fn.(*ssa.Function)
		if !ok {
			panic(fmt.Sprintf("MakeClosure function is not static: %s", v))
		}
		bindings := make([]closureBinding, len(v.Bindings))
		for i, binding := range v.Bindings {
			captured, ok := g.vals[binding.Name()]
			if !ok {
				panic(fmt.Sprintf("MakeClosure unresolved binding %s", binding.Name()))
			}
			bindings[i] = closureBinding{outerName: binding.Name(), value: captured, scope: g.bbScope}
		}
		if len(bindings) == 1 && closureFn.Signature.Params().Len() == 1 && closureFn.Signature.Results().Len() == 0 && closureHasStaticCall(closureFn, "Apply") && bindings[0].value.isDesc {
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(JITBuildScmerCallback), []JITValueDesc{%s}, 1)", dv, bindings[0].value.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gofunc", pinAcrossBlock: true}
			break
		}
		g.vals[name] = genVal{marker: "_go_closure", closureFn: closureFn, closureBindings: bindings}

	case *ssa.MakeInterface:
		inner := g.resolveValue(v.X)
		panicOnly := v.Referrers() != nil && len(*v.Referrers()) == 1
		if panicOnly {
			_, panicOnly = (*v.Referrers())[0].(*ssa.Panic)
		}
		if panicOnly {
			// Panic lowering keeps Scheme's existing panic conversion behavior.
			g.vals[name] = inner
			break
		}
		sourceType := g.sourceTypeExpr(v.X.Type())
		targetType := g.sourceTypeExpr(v.Type())
		dv := g.allocDesc()
		if inner.marker == "_aggregate_ptr" {
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(value *%s) %s { return *value }), []JITValueDesc{%s}, 2)", dv, sourceType, targetType, inner.goVar)
		} else {
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(value %s) %s { return value }), []JITValueDesc{%s}, 2)", dv, sourceType, targetType, inner.goVar)
		}
		g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_goiface", pinAcrossBlock: true}

	case *ssa.TypeAssert:
		src := g.resolveValue(v.X)
		if !v.CommaOk {
			dv := g.allocDesc()
			if isScmerType(v.AssertedType) {
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(jitAssertScmer), []JITValueDesc{%s}, 2)", dv, src.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
				break
			}
			words := goCallWordCount(v.AssertedType)
			if words < 1 || words > 3 {
				panic(fmt.Sprintf("non-comma-ok TypeAssert has unsupported ABI shape: %s", v))
			}
			typeExpr := g.sourceTypeExpr(v.AssertedType)
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(value any) %s { return value.(%s) }), []JITValueDesc{%s}, %d)", dv, typeExpr, typeExpr, src.goVar, words)
			marker := ""
			switch v.AssertedType.Underlying().(type) {
			case *types.Interface:
				marker = "_goiface"
			case *types.Slice:
				marker = "_slice"
			case *types.Map:
				marker = "_gomap"
			case *types.Signature:
				marker = "_gofunc_variadic"
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker, pinAcrossBlock: words > 1}
			break
		}
		helper := ""
		resultWords := 0
		resultPointerMask := 0
		switch asserted := v.AssertedType.Underlying().(type) {
		case *types.Basic:
			if asserted.Kind() == types.String {
				helper = "jitAssertString"
				resultWords = 2
				resultPointerMask = 1
			}
		case *types.Interface:
			if types.TypeString(v.AssertedType, nil) == "io.Reader" {
				helper = "jitAssertReader"
				resultWords = 2
				resultPointerMask = 3
			}
		case *types.Signature:
			if g.sourceTypeExpr(v.AssertedType) == "func(...Scmer) Scmer" {
				helper = "jitAssertScmerFunction"
				resultWords = 1
				resultPointerMask = 1
			}
		}
		if helper == "" {
			panic(fmt.Sprintf("unsupported TypeAssert: %s", v))
		}
		callResults := g.allocTemp("callResults")
		valueDesc := g.allocDesc()
		okDesc := g.allocDesc()
		g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(%s), []JITValueDesc{%s}, []uint8{%d, 1}, []uint8{%d, 0})", callResults, helper, src.goVar, resultWords, resultPointerMask)
		g.emit("%s := %s[0]", valueDesc, callResults)
		g.emit("%s := %s[1]", okDesc, callResults)
		g.emit("_ = %s", valueDesc)
		g.emit("_ = %s", okDesc)
		g.emit("ctx.EmitAndRegImm32(%s.Reg, 1)", okDesc)
		g.emit("%s.Type = tagBool", okDesc)
		valueMarker := ""
		if helper == "jitAssertString" {
			valueMarker = "_gostring"
		}
		g.vals[name] = genVal{tuple: []genVal{{goVar: valueDesc, isDesc: true, marker: valueMarker}, {goVar: okDesc, isDesc: true}}}

	case *ssa.ChangeInterface:
		src := g.resolveValue(v.X)
		if g.sourceTypeExpr(v.X.Type()) == "io.Reader" && g.sourceTypeExpr(v.Type()) == "any" {
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(jitReaderToAny), []JITValueDesc{%s}, 2)", dv, src.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_goiface"}
			break
		}
		targetType := g.sourceTypeExpr(v.Type())
		sourceType := g.sourceTypeExpr(v.X.Type())
		words := goCallWordCount(v.Type())
		if words < 1 || words > 3 {
			panic(fmt.Sprintf("unsupported ChangeInterface ABI shape: %s", v))
		}
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(value %s) %s { return value }), []JITValueDesc{%s}, %d)", dv, sourceType, targetType, src.goVar, words)
		g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_goiface", pinAcrossBlock: words > 1}

	case *ssa.MakeMap:
		mapType := v.Type().Underlying().(*types.Map)
		reserve := "JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}"
		if v.Reserve != nil {
			reserve = g.resolveValue(v.Reserve).goVar
		}
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) %s { return make(%s, size) }), []JITValueDesc{%s}, 1)", dv, g.sourceTypeExpr(v.Type()), g.sourceTypeExpr(v.Type()), reserve)
		g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gomap", pinAcrossBlock: true}
		_ = mapType

	case *ssa.MapUpdate:
		mapType := v.Map.Type().Underlying().(*types.Map)
		container := g.resolveValue(v.Map)
		key := g.resolveValue(v.Key)
		value := g.resolveValue(v.Value)
		g.emit("ctx.EmitGoCallVoid(GoFuncAddr(func(m %s, key %s, value %s) { m[key] = value }), []JITValueDesc{%s, %s, %s})", g.sourceTypeExpr(v.Map.Type()), g.sourceTypeExpr(mapType.Key()), g.sourceTypeExpr(mapType.Elem()), container.goVar, key.goVar, value.goVar)

	case *ssa.Panic:
		panicVal := g.resolveValue(v.X)
		if !panicVal.isDesc {
			panic(fmt.Sprintf("unsupported Panic payload: %s", v))
		}
		g.emit("ctx.EnsureDesc(&%s)", panicVal.goVar)
		g.emit("if %s.Loc == LocImm {", panicVal.goVar)
		g.emit("\ttmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tif %s.Imm.GetTag() == tagBool {", panicVal.goVar)
		g.emit("\t\tctx.EmitMakeBool(tmpPair, %s)", panicVal.goVar)
		g.emit("\t} else if %s.Imm.GetTag() == tagInt {", panicVal.goVar)
		g.emit("\t\tctx.EmitMakeInt(tmpPair, %s)", panicVal.goVar)
		g.emit("\t} else if %s.Imm.GetTag() == tagFloat {", panicVal.goVar)
		g.emit("\t\tctx.EmitMakeFloat(tmpPair, %s)", panicVal.goVar)
		g.emit("\t} else if %s.Imm.GetTag() == tagNil {", panicVal.goVar)
		g.emit("\t\tctx.EmitMakeNil(tmpPair)")
		g.emit("\t} else {")
		g.emit("\t\tptrWord, auxWord := %s.Imm.RawWords()", panicVal.goVar)
		g.emit("\t\tctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))")
		g.emit("\t\tctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)")
		g.emit("\t}")
		g.emit("\t%s = tmpPair", panicVal.goVar)
		g.emit("} else if %s.Loc == LocReg {", panicVal.goVar)
		g.emit("\ttmpPair := JITValueDesc{Loc: LocRegPair, Type: %s.Type, Reg: ctx.AllocRegExcept(%s.Reg), Reg2: ctx.AllocRegExcept(%s.Reg)}", panicVal.goVar, panicVal.goVar, panicVal.goVar)
		g.emit("\tswitch %s.Type {", panicVal.goVar)
		g.emit("\tcase tagBool:")
		g.emit("\t\tctx.EmitMakeBool(tmpPair, %s)", panicVal.goVar)
		g.emit("\tcase tagInt:")
		g.emit("\t\tctx.EmitMakeInt(tmpPair, %s)", panicVal.goVar)
		g.emit("\tcase tagFloat:")
		g.emit("\t\tctx.EmitMakeFloat(tmpPair, %s)", panicVal.goVar)
		g.emit("\tdefault:")
		g.emit("\t\tpanic(\"jit: panic arg scalar type unknown for Scmer pair\")")
		g.emit("\t}")
		g.emit("\tctx.FreeDesc(&%s)", panicVal.goVar)
		g.emit("\t%s = tmpPair", panicVal.goVar)
		g.emit("}")
		g.emit("if %s.Loc != LocRegPair && %s.Loc != LocStackPair && %s.Loc != LocInputPair {", panicVal.goVar, panicVal.goVar, panicVal.goVar)
		g.emit("\tpanic(\"jit: panic arg expects Scmer pair\")")
		g.emit("}")
		if g.storageMode {
			g.emit("ctx.EmitGoCallVoid(GoFuncAddr(JITPanic), []JITValueDesc{%s})", panicVal.goVar)
		} else {
			g.emit("ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{%s})", panicVal.goVar)
		}

	case *ssa.Slice:
		// A variadic Go parameter is already represented by the emitter's args
		// descriptors. Keep constant subslices virtual so loops can index those
		// descriptors directly instead of allocating a temporary []Scmer header.
		variadic := g.vals[v.X.Name()]
		if variadic.marker == "_variadic_args" {
			low := int64(0)
			if v.Low != nil {
				low = constInt(v.Low)
			}
			if low < 0 {
				panic(fmt.Sprintf("negative variadic Slice low bound: %s", v))
			}
			out := genVal{marker: "_variadic_args", variadicOffset: variadic.variadicOffset + int(low)}
			if v.High != nil {
				high := constInt(v.High)
				if high < low {
					panic(fmt.Sprintf("invalid variadic Slice bounds: %s", v))
				}
				out.variadicLen = int(high - low)
				out.variadicLenKnown = true
			}
			if v.Max != nil {
				_ = constInt(v.Max)
			}
			g.vals[name] = out
			break
		}
		if variadic.marker == "_goarrayptr" {
			array := v.X.Type().Underlying().(*types.Pointer).Elem().Underlying().(*types.Array)
			bound := func(value ssa.Value, fallback int64) string {
				if value == nil {
					return strconv.FormatInt(fallback, 10)
				}
				constantValue, ok := value.(*ssa.Const)
				if !ok {
					panic(fmt.Sprintf("dynamic Slice bound on Go array: %s", v))
				}
				return strconv.FormatInt(constInt(constantValue), 10)
			}
			low := bound(v.Low, 0)
			high := bound(v.High, array.Len())
			max := bound(v.Max, array.Len())
			result := g.allocTemp("sliceResults")
			dv := g.allocDesc()
			arrayExpr := g.sourceTypeExpr(v.X.Type().Underlying().(*types.Pointer).Elem())
			elemExpr := g.sourceTypeExpr(array.Elem())
			g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *%s) []%s { return value[%s:%s:%s] }), []JITValueDesc{%s}, []uint8{3}, []uint8{1})", result, arrayExpr, elemExpr, low, high, max, variadic.goVar)
			g.emit("%s := %s[0]", dv, result)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
			break
		}
		if g.storageMode {
			// Storage fast path: materialize a proper Go string/slice header
			// as LocRegPair{ptr,len}. Never collapse to LocImm because Go calls
			// expect 2 ABI words for string/slice values.
			x := g.vals[v.X.Name()]
			if !x.isDesc {
				panic(fmt.Sprintf("Slice on non-desc: %s", v))
			}
			low := g.resolveValue(v.Low)
			high := g.resolveValue(v.High)
			g.emit("ctx.EnsureDesc(&%s)", x.goVar)
			if low.isDesc {
				g.emit("ctx.EnsureDesc(&%s)", low.goVar)
			}
			if high.isDesc {
				g.emit("ctx.EnsureDesc(&%s)", high.goVar)
			}
			ptrReg := g.allocReg()
			lenReg := g.allocReg()
			dv := g.allocDesc()
			g.emit("%s := ctx.AllocReg()", ptrReg)
			g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
			g.emit("ctx.EnsureDesc(&%s)", x.goVar)
			if low.isDesc {
				g.emit("ctx.EnsureDesc(&%s)", low.goVar)
			}
			if high.isDesc {
				g.emit("ctx.EnsureDesc(&%s)", high.goVar)
			}
			g.emit("if %s.Loc == LocImm {", x.goVar)
			g.emit("\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", ptrReg, x.goVar)
			g.emit("} else if %s.Loc == LocRegPair {", x.goVar)
			g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", ptrReg, x.goVar)
			g.emit("} else {")
			g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", ptrReg, x.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", low.goVar)
			g.emit("\tif %s.Imm.Int() != 0 {", low.goVar)
			g.emit("\t\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", low.goVar, low.goVar)
			g.emit("\t\t\tctx.EmitAddRegImm32(%s, int32(%s.Imm.Int()))", ptrReg, low.goVar)
			g.emit("\t\t} else {")
			g.emit("\t\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\t\tctx.EmitAddInt64(%s, RegR11)", ptrReg)
			g.emit("\t\t}")
			g.emit("\t}")
			g.emit("} else {")
			g.emit("\tctx.EmitAddInt64(%s, %s.Reg)", ptrReg, low.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", high.goVar)
			g.emit("\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", lenReg, high.goVar)
			g.emit("} else {")
			g.emit("\tctx.EmitMovRegReg(%s, %s.Reg)", lenReg, high.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", low.goVar)
			g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", low.goVar, low.goVar)
			g.emit("\t\tctx.EmitSubRegImm32(%s, int32(%s.Imm.Int()))", lenReg, low.goVar)
			g.emit("\t} else {")
			g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\tctx.EmitSubInt64(%s, RegR11)", lenReg)
			g.emit("\t}")
			g.emit("} else {")
			g.emit("\tctx.EmitSubInt64(%s, %s.Reg)", lenReg, low.goVar)
			g.emit("}")
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, lenReg)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
			break
		}
		// Sub-slice: strings use ptr+len, Go slices use the complete ptr+len+cap ABI header.
		x := g.vals[v.X.Name()]
		if strings.HasPrefix(x.marker, "_stackarray:") {
			parts := strings.Split(x.marker, ":")
			if len(parts) != 3 {
				panic(fmt.Sprintf("malformed stack array marker: %q", x.marker))
			}
			arrayLen, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				panic(fmt.Sprintf("malformed stack array length: %q", x.marker))
			}
			elemSize, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				panic(fmt.Sprintf("malformed stack array element size: %q", x.marker))
			}
			low := int64(0)
			high := arrayLen
			max := arrayLen
			if v.Low != nil {
				low = constInt(v.Low)
			}
			if v.High != nil {
				high = constInt(v.High)
			}
			if v.Max != nil {
				max = constInt(v.Max)
			}
			if low < 0 || low > high || high > max || max > arrayLen {
				panic(fmt.Sprintf("invalid local stack array Slice bounds: %s", v))
			}
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(%d), KnownSliceCap: int32(%d), SliceSizeKnown: true}", dv, high-low, max-low)
			g.emit("_ = %s", dv)
			stackBase := x.goVar
			if low != 0 {
				stackBase = fmt.Sprintf("int32(%s)+int32(%d)", x.goVar, low*elemSize)
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_stackslice", stackBase: stackBase, stackLen: int(high - low)}
			break
		}
		if x.marker == "_alloc" {
			ptr, ok := v.X.Type().Underlying().(*types.Pointer)
			if !ok {
				panic(fmt.Sprintf("Slice on unsupported allocation: %s", v))
			}
			array, ok := ptr.Elem().Underlying().(*types.Array)
			if !ok || array.Len() != 0 {
				panic(fmt.Sprintf("Slice on non-empty local allocation needs rooted stack storage: %s", v))
			}
			ptrReg := g.allocReg()
			lenReg := g.allocReg()
			capReg := g.allocReg()
			dv := g.allocDesc()
			g.emit("%s := ctx.AllocReg()", ptrReg)
			g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
			g.emit("%s := ctx.AllocRegExcept(%s, %s)", capReg, ptrReg, lenReg)
			g.emit("ctx.EmitMovRegImm64(%s, 0)", ptrReg)
			g.emit("ctx.EmitMovRegImm64(%s, 0)", lenReg)
			g.emit("ctx.EmitMovRegImm64(%s, 0)", capReg)
			g.emit("%s := JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv, ptrReg, lenReg, capReg)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice", pinAcrossBlock: true}
			break
		}
		sliceType, isGoSlice := v.X.Type().Underlying().(*types.Slice)
		elemSize := int64(1)
		if isGoSlice {
			elemSize = types.SizesFor("gc", "amd64").Sizeof(sliceType.Elem())
		}
		var low genVal
		if v.Low == nil {
			lowDesc := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}", lowDesc)
			low = genVal{goVar: lowDesc, isDesc: true}
		} else {
			low = g.resolveValue(v.Low)
		}
		var high genVal
		if v.High == nil {
			// Go slice syntax x[low:] => high defaults to len(x).
			// For descriptor-backed strings/slices, len is carried in Reg2.
			highDesc := g.allocDesc()
			g.emit("var %s JITValueDesc", highDesc)
			g.emit("ctx.EnsureDesc(&%s)", x.goVar)
			g.emit("if %s.Loc == LocRegPair || %s.Loc == LocRegTriple {", x.goVar, x.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", highDesc, x.goVar)
			g.emit("} else {")
			g.emit("\tpanic(\"Slice with omitted high requires descriptor with length in Reg2\")")
			g.emit("}")
			high = genVal{goVar: highDesc, isDesc: true}
		} else {
			high = g.resolveValue(v.High)
		}
		if x.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", x.goVar)
		}
		if low.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", low.goVar)
		}
		if high.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", high.goVar)
		}
		dv := g.allocDesc()
		// Compute new length: high - low
		lenDesc := g.allocDesc()
		g.emit("var %s JITValueDesc", lenDesc)
		g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", high.goVar, low.goVar)
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() - %s.Imm.Int())}", lenDesc, high.goVar, low.goVar)
		g.emit("} else {")
		lenReg := g.allocReg()
		g.emit("\t%s := ctx.AllocReg()", lenReg)
		g.emit("\tif %s.Loc == LocImm {", high.goVar)
		g.emit("\t\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", lenReg, high.goVar)
		g.emit("\t} else {")
		g.emit("\t\tctx.EmitMovRegReg(%s, %s.Reg)", lenReg, high.goVar)
		g.emit("\t}")
		g.emit("\tif %s.Loc == LocImm {", low.goVar)
		g.emit("\t\tctx.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
		g.emit("\t\tctx.EmitSubInt64(%s, RegR11)", lenReg)
		g.emit("\t} else {")
		g.emit("\t\tctx.EmitSubInt64(%s, %s.Reg)", lenReg, low.goVar)
		g.emit("\t}")
		g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", lenDesc, lenReg)
		g.emit("}")
		// Compute new data pointer: x.ptr + low*element-size.
		ptrDesc := g.allocDesc()
		g.emit("var %s JITValueDesc", ptrDesc)
		if x.isDesc {
			ptrReg := g.allocReg()
			g.emit("%s := ctx.EmitSliceDataAfterLow(&%s, &%s, %d)", ptrReg, x.goVar, low.goVar, elemSize)
			g.emit("%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", ptrDesc, ptrReg)
			g.emit("ctx.BindReg(%s, &%s)", ptrReg, ptrDesc)
		} else {
			panic(fmt.Sprintf("Slice on non-desc: %s", v))
		}
		// Combine into the ABI header expected by the value's Go type.
		dv2 := g.allocDesc()
		g.emit("var %s JITValueDesc", dv2)
		// Always materialize string/slice headers as register pairs. A single
		// LocImm Scmer cannot represent Go string header (ptr+len) correctly.
		finalPtr := g.allocReg()
		finalLen := g.allocReg()
		g.emit("var %s Reg", finalPtr)
		g.emit("var %s Reg", finalLen)
		g.emit("ctx.SyncDesc(&%s)", ptrDesc)
		g.emit("ctx.EnsureDesc(&%s)", ptrDesc)
		g.emit("if %s.Loc == LocImm {", ptrDesc)
		g.emit("\t%s = ctx.AllocReg()", finalPtr)
		g.emit("\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", finalPtr, ptrDesc)
		g.emit("} else {")
		g.emit("\t%s = %s.Reg", finalPtr, ptrDesc)
		g.emit("}")
		g.emit("ctx.ProtectReg(%s)", finalPtr)
		g.emit("ctx.SyncDesc(&%s)", lenDesc)
		g.emit("ctx.EnsureDesc(&%s)", lenDesc)
		g.emit("if %s.Loc == LocImm {", lenDesc)
		g.emit("\t%s = ctx.AllocReg()", finalLen)
		g.emit("\tctx.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", finalLen, lenDesc)
		g.emit("} else {")
		g.emit("\t%s = %s.Reg", finalLen, lenDesc)
		g.emit("}")
		g.emit("ctx.ProtectReg(%s)", finalLen)
		if isGoSlice {
			finalCap := g.allocReg()
			g.emit("%s := ctx.EmitSliceCapAfterLow(&%s, &%s, %s, %s)", finalCap, x.goVar, low.goVar, finalPtr, finalLen)
			g.emit("ctx.UnprotectReg(%s)", finalLen)
			g.emit("ctx.UnprotectReg(%s)", finalPtr)
			g.emit("%s = JITValueDesc{Loc: LocRegTriple, Reg: %s, Reg2: %s, Reg3: %s}", dv2, finalPtr, finalLen, finalCap)
			g.emit("ctx.BindReg(%s, &%s)", finalPtr, dv2)
			g.emit("ctx.BindReg(%s, &%s)", finalLen, dv2)
			g.emit("ctx.BindReg(%s, &%s)", finalCap, dv2)
		} else {
			g.emit("ctx.UnprotectReg(%s)", finalLen)
			g.emit("ctx.UnprotectReg(%s)", finalPtr)
			g.emit("%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv2, finalPtr, finalLen)
			g.emit("ctx.BindReg(%s, &%s)", finalPtr, dv2)
			g.emit("ctx.BindReg(%s, &%s)", finalLen, dv2)
		}
		_ = dv
		marker := "_gostring"
		if isGoSlice {
			marker = "_slice"
		}
		g.vals[name] = genVal{goVar: dv2, isDesc: true, marker: marker}

	case *ssa.MakeSlice:
		elem := v.Type().Underlying().(*types.Slice).Elem()
		if !isScmerType(elem) && !isByteType(elem) {
			panic(fmt.Sprintf("MakeSlice of unsupported element type: %s", v))
		}
		length := g.resolveValue(v.Len)
		capacity := length
		if v.Cap != nil {
			capacity = g.resolveValue(v.Cap)
		}
		g.emit("ctx.ReclaimUntrackedRegs()")
		g.emit("ctx.EnsureDesc(&%s)", length.goVar)
		g.emit("ctx.EnsureDesc(&%s)", capacity.goVar)
		callResults := g.allocTemp("callResults")
		dv := g.allocDesc()
		helper := "jitMakeScmerSlice"
		if isByteType(elem) {
			helper = "jitMakeByteSlice"
		}
		g.emit("%s := JITEmitGoCallResults(ctx, GoFuncAddr(%s), []JITValueDesc{%s, %s}, []uint8{3}, []uint8{1})", callResults, helper, length.goVar, capacity.goVar)
		g.emit("%s := %s[0]", dv, callResults)
		g.emit("%s.Type = tagSlice", dv)
		g.recordSliceResult(name, v, dv)

	default:
		panic(instrDesc(instr))
	}
}

// emitReturnSingleBlock handles Return for single-block functions (with constant propagation).
func (g *codeGen) emitReturnSingleBlock(v *ssa.Return) {
	if len(g.bbLabels) > 0 {
		g.emit("ctx.ResolveFixups()")
	}
	if len(v.Results) == 0 {
		g.emit("if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: NewNil()} }")
		g.emit("ctx.EmitMakeNil(result)")
		g.emit("result.Type = tagNil")
		g.emit("return result")
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newargslice":
		g.emit("return jitMaterializeVirtualSlice(ctx, %s, result)", res.goVar)
	case "_newbool":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tctx.BindReg(result.Reg, &result)")
		g.emit("\tctx.BindReg(result.Reg2, &result)")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.EmitMakeBool(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.EmitMakeBool(result, %s)", res.goVar)
		if res.resultTargetVar != "" {
			g.emit("\tif !%s { ctx.FreeReg(%s.Reg) }", res.resultTargetVar, res.goVar)
		} else {
			g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		}
		g.emit("}")
		g.emit("result.Type = tagBool")
		g.emit("return result")
	case "_newint":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tctx.BindReg(result.Reg, &result)")
		g.emit("\tctx.BindReg(result.Reg2, &result)")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.EmitMakeInt(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.EmitMakeInt(result, %s)", res.goVar)
		if res.resultTargetVar != "" {
			g.emit("\tif !%s { ctx.FreeReg(%s.Reg) }", res.resultTargetVar, res.goVar)
		} else {
			g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		}
		g.emit("}")
		g.emit("result.Type = tagInt")
		g.emit("return result")
	case "_newfloat":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tctx.BindReg(result.Reg, &result)")
		g.emit("\tctx.BindReg(result.Reg2, &result)")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.EmitMakeFloat(result, %s)", res.goVar)
		if res.resultTargetVar != "" {
			g.emit("\tif !%s { ctx.FreeReg(%s.Reg) }", res.resultTargetVar, res.goVar)
		} else {
			g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		}
		g.emit("}")
		g.emit("result.Type = tagFloat")
		g.emit("return result")
	case "_newnil":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("\tctx.BindReg(result.Reg, &result)")
		g.emit("\tctx.BindReg(result.Reg2, &result)")
		g.emit("}")
		g.emit("ctx.EmitMakeNil(result)")
		g.emit("result.Type = tagNil")
		g.emit("return result")
	case "_newstring":
		// NewString(s string) Scmer — arg is Go string {ptr, len} (2 words), result is Scmer (2 words)
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{%s}, 2)", dv, res.goVar)
		g.emit("if result.Loc == LocAny { return %s }", dv)
		g.emit("ctx.EmitMovPairToResult(&%s, &result)", dv)
		g.emit("result.Type = tagString")
		g.emit("return result")
	default:
		if res.isDesc {
			// Constant folding: LocImm values need no materialization.
			g.emit("if %s.Loc == LocImm {", res.goVar)
			g.emit("\tif result.Loc == LocAny { return %s }", res.goVar)
			g.emit("}")
			g.emit("if result.Loc == LocAny {")
			g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
			g.emit("\tctx.BindReg(result.Reg, &result)")
			g.emit("\tctx.BindReg(result.Reg2, &result)")
			g.emit("}")
			g.emit("ctx.SyncDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair || %s.Loc == LocStackPair || %s.Loc == LocInputPair {", res.goVar, res.goVar, res.goVar)
			g.emit("\tctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("\tresult.Type = %s.Type", res.goVar)
			g.emit("} else {")
			g.emit("\tswitch %s.Type {", res.goVar)
			g.emit("\tcase tagBool:")
			g.emit("\t\tctx.EmitMakeBool(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagBool")
			g.emit("\tcase tagInt:")
			g.emit("\t\tctx.EmitMakeInt(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagInt")
			g.emit("\tcase tagFloat:")
			g.emit("\t\tctx.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagFloat")
			g.emit("\tcase tagNil:")
			g.emit("\t\tctx.EmitMakeNil(result)")
			g.emit("\t\tresult.Type = tagNil")
			g.emit("\tdefault:")
			g.emit("\t\tpanic(\"jit: single-block scalar return with unknown type\")")
			g.emit("\t}")
			g.emit("}")
			g.emit("return result")
		} else {
			panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
		}
	}
}

// emitReturnMultiBlock handles Return for multi-block functions.
// Emits machine code to construct the result + JMP to the shared epilogue.
func (g *codeGen) emitReturnMultiBlock(v *ssa.Return) {
	if g.storageMode {
		if g.returnPhiReg == "" || g.returnPhiReg2 == "" {
			panic("jit: storage return-phi registers not initialized")
		}
		retDesc := g.allocDesc()
		g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", retDesc, g.returnPhiReg, g.returnPhiReg2)
		if len(v.Results) == 0 {
			g.emit("ctx.EmitMakeNil(%s)", retDesc)
			g.emit("ctx.EmitJmp(%s)", g.endLabel)
			return
		}
		res := g.vals[v.Results[0].Name()]
		switch res.marker {
		case "_newbool":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.EmitMakeBool(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newint":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.EmitMakeInt(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newfloat":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.EmitMakeFloat(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newnil":
			g.emit("ctx.EmitMakeNil(%s)", retDesc)
		case "_newstring":
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{%s}, 2)", dv, res.goVar)
			g.emit("ctx.EmitMovPairToResult(&%s, &%s)", dv, retDesc)
		default:
			if res.isDesc {
				g.emit("ctx.EnsureDesc(&%s)", res.goVar)
				g.emit("if %s.Loc == LocRegPair {", res.goVar)
				g.emit("\tctx.EmitMovPairToResult(&%s, &%s)", res.goVar, retDesc)
				g.emit("} else {")
				// Scalar descriptors carry only payload in one register/immediate.
				// EmitMake* reconstructs the Scmer ptr+aux layout (including sentinel ptr for int/float)
				// without requiring a separate runtime type register.
				g.emit("\tswitch %s.Type {", res.goVar)
				g.emit("\tcase tagBool:")
				g.emit("\t\tctx.EmitMakeBool(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagInt:")
				g.emit("\t\tctx.EmitMakeInt(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagFloat:")
				g.emit("\t\tctx.EmitMakeFloat(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagNil:")
				g.emit("\t\tctx.EmitMakeNil(%s)", retDesc)
				g.emit("\tdefault:")
				g.emit("\t\tctx.EmitMovPairToResult(&%s, &%s)", res.goVar, retDesc)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
			}
		}
		g.emit("ctx.EmitJmp(%s)", g.endLabel)
		return
	}

	if len(v.Results) == 0 {
		g.emit("ctx.EmitMakeNil(result)")
		g.emit("ctx.EmitJmp(%s)", g.endLabel)
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newbool":
		g.emitScalarReturnIntoResult(res, "Bool", "tagBool")
		g.emit("result.Type = tagBool")
	case "_newint":
		g.emitScalarReturnIntoResult(res, "Int", "tagInt")
		g.emit("result.Type = tagInt")
	case "_newfloat":
		g.emitScalarReturnIntoResult(res, "Float", "tagFloat")
		g.emit("result.Type = tagFloat")
	case "_newnil":
		g.emit("ctx.EmitMakeNil(result)")
		g.emit("result.Type = tagNil")
	case "_newstring":
		// NewString(s string) Scmer — arg is Go string {ptr, len} (2 words), result is Scmer (2 words)
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{%s}, 2)", dv, res.goVar)
		g.emit("ctx.EmitMovPairToResult(&%s, &result)", dv)
		g.emit("result.Type = tagString")
	default:
		// Already-materialized Scmer in LocRegPair — MOV to result registers
		if res.isDesc {
			g.emit("ctx.SyncDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair || %s.Loc == LocStackPair || %s.Loc == LocInputPair {", res.goVar, res.goVar, res.goVar)
			g.emit("\tctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("\tresult.Type = %s.Type", res.goVar)
			g.emit("} else {")
			// Known scalar type => no additional tag register allocation.
			// The concrete Scmer pair is materialized directly into result registers.
			g.emit("\tswitch %s.Type {", res.goVar)
			g.emit("\tcase tagBool:")
			g.emit("\t\tctx.EmitMakeBool(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagBool")
			g.emit("\tcase tagInt:")
			g.emit("\t\tctx.EmitMakeInt(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagInt")
			g.emit("\tcase tagFloat:")
			g.emit("\t\tctx.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagFloat")
			g.emit("\tcase tagNil:")
			g.emit("\t\tctx.EmitMakeNil(result)")
			g.emit("\t\tresult.Type = tagNil")
			g.emit("\tdefault:")
			g.emit("\t\tctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("\t\tresult.Type = %s.Type", res.goVar)
			g.emit("\t}")
			g.emit("}")
		} else {
			panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
		}
	}
	g.emit("ctx.EmitJmp(%s)", g.endLabel)
}
func (g *codeGen) emitScalarReturnIntoResult(res genVal, constructor, tag string) {
	payload := g.allocDesc()
	g.emit("if %s.Loc == LocImm {", res.goVar)
	g.emit("\tctx.EmitMake%s(result, %s)", constructor, res.goVar)
	g.emit("} else {")
	g.emit("\tctx.EmitMovToReg(result.Reg2, %s)", res.goVar)
	g.emit("\t%s := JITValueDesc{Loc: LocReg, Type: %s, Reg: result.Reg2, ID: 0}", payload, tag)
	g.emit("\tctx.EmitMake%s(result, %s)", constructor, payload)
	g.emit("\tif %s.Loc == LocReg && %s.Reg != result.Reg2 { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar, res.goVar)
	g.emit("}")
}

// emitInlineReturn handles Return inside an inlined function (multi-block).
// Moves the return value to the pre-allocated inline result register(s) and JMPs to end.
func (g *codeGen) emitInlineReturn(v *ssa.Return) {
	if len(v.Results) == 0 {
		// void return — shouldn't happen for inlined value-returning functions
		g.emit("ctx.EmitJmp(%s)", g.inlineEndLabel)
		return
	}
	if len(g.inlineReturnTuple) > 0 {
		if len(v.Results) != len(g.inlineReturnTuple) {
			panic(fmt.Sprintf("inline return result count %d does not match %d destinations", len(v.Results), len(g.inlineReturnTuple)))
		}
		for i, returned := range v.Results {
			res := g.resolveValue(returned)
			words := goCallWordCount(returned.Type())
			if words < 1 || words > 3 {
				panic(fmt.Sprintf("unsupported inline tuple result %s", returned))
			}
			if res.marker == fmt.Sprintf("_gozero:%d", words) {
				g.emit("ctx.EmitZeroDescWords(&%s, %d)", g.inlineReturnTuple[i].goVar, words)
				continue
			}
			if !res.isDesc {
				panic(fmt.Sprintf("unsupported inline tuple result %s", returned))
			}
			g.emit("ctx.EmitCopyDescWords(&%s, &%s, %d)", g.inlineReturnTuple[i].goVar, res.goVar, words)
		}
		g.emit("ctx.EmitJmp(%s)", g.inlineEndLabel)
		return
	}
	if g.inlineReturnsScm {
		if g.inlineReturnRegVar == "" {
			g.inlineReturnRegVar = g.allocReg()
			g.emit("%s := ctx.AllocReg()", g.inlineReturnRegVar)
			g.inlineReturnReg2Var = g.allocReg()
			g.emit("%s := ctx.AllocRegExcept(%s)", g.inlineReturnReg2Var, g.inlineReturnRegVar)
		}
		// Scmer pair return: construct Scmer into the two pre-allocated registers
		res := g.vals[v.Results[0].Name()]
		irDesc := g.allocDesc()
		g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", irDesc, g.inlineReturnRegVar, g.inlineReturnReg2Var)
		switch res.marker {
		case "_newbool":
			g.emit("ctx.EmitMakeBool(%s, %s)", irDesc, res.goVar)
		case "_newint":
			g.emit("ctx.EmitMakeInt(%s, %s)", irDesc, res.goVar)
		case "_newfloat":
			g.emit("ctx.EmitMakeFloat(%s, %s)", irDesc, res.goVar)
		case "_newnil":
			g.emit("ctx.EmitMakeNil(%s)", irDesc)
		case "_newstring":
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{%s}, 2)", dv, res.goVar)
			g.emit("ctx.EmitMovPairToResult(&%s, &%s)", dv, irDesc)
		default:
			if res.isDesc {
				g.emit("ctx.EmitMovPairToResult(&%s, &%s)", res.goVar, irDesc)
			} else {
				panic(fmt.Sprintf("unsupported inline Scmer return for %s (marker=%q)", v.Results[0], res.marker))
			}
		}
	} else {
		if g.inlineReturnRegVar == "" {
			g.inlineReturnRegVar = g.allocReg()
			g.emit("%s := ctx.AllocReg()", g.inlineReturnRegVar)
		}
		// Scalar return: move single value to result register
		res := g.resolveValue(v.Results[0])
		if res.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair {", res.goVar)
			g.emit("\tpanic(\"jit: scalar inline return has LocRegPair\")")
			g.emit("} else {")
			g.emit("\tctx.EmitMovToReg(%s, %s)", g.inlineReturnRegVar, res.goVar)
			g.emit("}")
		} else {
			g.emit("ctx.EmitMovRegReg(%s, %s)", g.inlineReturnRegVar, res.goVar)
		}
	}
	g.emit("ctx.EmitJmp(%s)", g.inlineEndLabel)
}

func (g *codeGen) lookup(v ssa.Value) genVal {
	v = g.rewriteSSAValue(v)
	if gv, ok := g.vals[v.Name()]; ok {
		if gv.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", gv.goVar)
		}
		return gv
	}
	panic(fmt.Sprintf("unresolved SSA value: %s", v))
}

var (
	locRegAssignRe       = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?JITValueDesc\{Loc:\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?LocReg,\s*(?:Type:\s*[^,}]+,\s*)?Reg:\s*([^,}]+)`)
	locRegPairAssignRe   = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?JITValueDesc\{Loc:\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?LocRegPair,\s*(?:Type:\s*[^,}]+,\s*)?Reg:\s*([^,}]+),\s*Reg2:\s*([^,}]+)`)
	locRegTripleAssignRe = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?JITValueDesc\{Loc:\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?LocRegTriple,\s*(?:Type:\s*[^,}]+,\s*)?Reg:\s*([^,}]+),\s*Reg2:\s*([^,}]+),\s*Reg3:\s*([^,}]+)`)
	regExprRe            = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?$`)
)

func bindableRegExpr(expr string) bool {
	return regExprRe.MatchString(strings.TrimSpace(expr))
}

func injectBindRegCalls(code string) string {
	lines := strings.Split(code, "\n")
	out := make([]string, 0, len(lines)+len(lines)/3)
	for _, line := range lines {
		out = append(out, line)
		// ID 0 explicitly marks a borrowed register view. Its source descriptor
		// remains the spill/liveness owner; rebinding the register here would make
		// later consumers overwrite or free that source (for example a Go slice's
		// len register in a loop).
		if strings.Contains(line, "ID: 0") {
			continue
		}
		if m := locRegTripleAssignRe.FindStringSubmatch(line); m != nil {
			indent, descVar := m[1], m[2]
			for _, expr := range m[3:6] {
				expr = strings.TrimSpace(expr)
				if bindableRegExpr(expr) {
					out = append(out, fmt.Sprintf("%sctx.BindReg(%s, &%s)", indent, expr, descVar))
				}
			}
			continue
		}
		if m := locRegAssignRe.FindStringSubmatch(line); m != nil {
			indent, descVar, regExpr := m[1], m[2], strings.TrimSpace(m[3])
			if bindableRegExpr(regExpr) {
				out = append(out, fmt.Sprintf("%sctx.BindReg(%s, &%s)", indent, regExpr, descVar))
			}
			continue
		}
		if m := locRegPairAssignRe.FindStringSubmatch(line); m != nil {
			indent, descVar := m[1], m[2]
			regExpr1 := strings.TrimSpace(m[3])
			regExpr2 := strings.TrimSpace(m[4])
			if bindableRegExpr(regExpr1) {
				out = append(out, fmt.Sprintf("%sctx.BindReg(%s, &%s)", indent, regExpr1, descVar))
			}
			if bindableRegExpr(regExpr2) {
				out = append(out, fmt.Sprintf("%sctx.BindReg(%s, &%s)", indent, regExpr2, descVar))
			}
		}
	}
	return strings.Join(out, "\n")
}

// resolveValue resolves any SSA value to a genVal: constants become LocImm
// descriptors, everything else is looked up from g.vals (must be pre-computed).
func (g *codeGen) resolveValue(v ssa.Value) genVal {
	v = g.rewriteSSAValue(v)
	if global, ok := v.(*ssa.Global); ok {
		expr := global.Name()
		if global.Pkg != nil && global.Pkg.Pkg != nil && global.Pkg.Pkg.Path() != g.topLevelPkgPath {
			alias, imported := g.importedPkgAlias[global.Pkg.Pkg.Path()]
			if !imported || !token.IsExported(global.Name()) {
				panic(fmt.Sprintf("unresolved SSA value: %s", v))
			}
			expr = alias + "." + expr
		}
		dv := g.allocDesc()
		g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&%s)))), NoHeapPointer: true, Rooted: true}", dv, expr)
		return genVal{goVar: dv, isDesc: true, marker: "_goptr"}
	}
	if c, ok := v.(*ssa.Const); ok {
		dv := g.allocDesc()
		if c.Value == nil {
			if words := goCallWordCount(c.Type()); words > 0 {
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}", dv)
				return genVal{goVar: dv, isDesc: true, marker: fmt.Sprintf("_gozero:%d", words)}
			}
			g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}", dv)
		} else {
			switch c.Value.Kind() {
			case constant.Int:
				ival, _ := constant.Int64Val(c.Value)
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%d)}", dv, ival)
			case constant.Float:
				fval, _ := constant.Float64Val(c.Value)
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%v)}", dv, fval)
			case constant.Bool:
				bval := constant.BoolVal(c.Value)
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%v)}", dv, bval)
			case constant.String:
				sval := constant.StringVal(c.Value)
				g.emit("%s := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(%q)}", dv, sval)
			default:
				panic(fmt.Sprintf("unsupported constant kind: %s", c.Value.Kind()))
			}
		}
		return genVal{goVar: dv, isDesc: true}
	}
	if existing, ok := g.vals[v.Name()]; ok && strings.HasPrefix(existing.marker, "_sliceaddr:") {
		parts := strings.SplitN(existing.marker, ":", 3)
		if len(parts) != 3 {
			panic(fmt.Sprintf("malformed slice address marker: %q", existing.marker))
		}
		idxDescVar := g.overlayDescVar(existing.argIdxVar, existing.deferredIndexSSA)
		baseDescVar := g.overlayDescVar(parts[2], existing.deferredBaseSSA)
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitSliceElementAddress(&%s, &%s, int32(%s))", dv, baseDescVar, idxDescVar, parts[1])
		return genVal{goVar: dv, isDesc: true, marker: "_goptr"}
	}
	if existing, ok := g.vals[v.Name()]; ok && strings.HasPrefix(existing.marker, "_stackaddr:") {
		dv := g.allocDesc()
		reg := g.allocReg()
		g.emit("%s := ctx.AllocReg()", reg)
		g.emit("ctx.EmitLeaRegMem(%s, ctx.StackReg, %s)", reg, existing.offsetExpr)
		g.emit("%s := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s, NoHeapPointer: true}", dv, reg)
		return genVal{goVar: dv, isDesc: true, marker: "_goptr"}
	}
	return g.lookup(v)
}

// constInt extracts the int64 from a constant SSA value.
func constInt(v ssa.Value) int64 {
	c, ok := v.(*ssa.Const)
	if !ok {
		panic(fmt.Sprintf("expected constant, got %s", v))
	}
	val, ok := constInt64Value(c.Value)
	if !ok {
		panic(fmt.Sprintf("constant not int64: %s", c))
	}
	return val
}

func constInt64Value(v constant.Value) (val int64, ok bool) {
	defer func() {
		if recover() != nil {
			val = 0
			ok = false
		}
	}()
	return constant.Int64Val(v)
}

func constFloat64Value(v constant.Value) (val float64, ok bool) {
	defer func() {
		if recover() != nil {
			val = 0
			ok = false
		}
	}()
	return constant.Float64Val(v)
}

// opToCC maps a Go comparison token to the JIT condition code constant name.
func opToCC(op token.Token) string {
	switch op {
	case token.EQL:
		return "CondEqual"
	case token.NEQ:
		return "CondNotEqual"
	case token.LSS:
		return "CondSignedLess"
	case token.GTR:
		return "CondSignedGreater"
	case token.LEQ:
		return "CondSignedLessOrEqual"
	case token.GEQ:
		return "CondSignedGreaterOrEqual"
	default:
		return ""
	}
}

func opToCCUnsigned(op token.Token) string {
	switch op {
	case token.EQL:
		return "CondEqual"
	case token.NEQ:
		return "CondNotEqual"
	case token.LSS:
		return "CondUnsignedBelow"
	case token.GTR:
		return "CondUnsignedAbove"
	case token.LEQ:
		return "CondUnsignedBelowOrEqual"
	case token.GEQ:
		return "CondUnsignedAboveOrEqual"
	default:
		return ""
	}
}

// goOpStr maps a Go token to the Go operator string for codegen.
func goOpStr(op token.Token) string {
	switch op {
	case token.EQL:
		return "=="
	case token.NEQ:
		return "!="
	case token.LSS:
		return "<"
	case token.GTR:
		return ">"
	case token.LEQ:
		return "<="
	case token.GEQ:
		return ">="
	case token.ADD:
		return "+"
	case token.SUB:
		return "-"
	case token.MUL:
		return "*"
	case token.QUO:
		return "/"
	default:
		return ""
	}
}

// aluEmitFunc maps an arithmetic token to the JITContext emit method name for int64.
func aluEmitFunc(op token.Token) string {
	switch op {
	case token.ADD:
		return "EmitAddInt64"
	case token.SUB:
		return "EmitSubInt64"
	case token.MUL:
		return "EmitImulInt64"
	default:
		return ""
	}
}

func floatAluEmitFunc(op token.Token) string {
	switch op {
	case token.ADD:
		return "EmitAddFloat64"
	case token.SUB:
		return "EmitSubFloat64"
	case token.MUL:
		return "EmitMulFloat64"
	case token.QUO:
		return "EmitDivFloat64"
	default:
		return ""
	}
}

func isFloat64Type(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return b.Kind() == types.Float64 || b.Kind() == types.UntypedFloat
}

func intTypeInfo(t types.Type) (signed bool, bits int, ok bool) {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false, 0, false
	}
	switch b.Kind() {
	case types.Int8:
		return true, 8, true
	case types.Int16:
		return true, 16, true
	case types.Int32:
		return true, 32, true
	case types.Int64:
		return true, 64, true
	case types.Int, types.UntypedInt:
		return true, 64, true
	case types.Uint8:
		return false, 8, true
	case types.Uint16:
		return false, 16, true
	case types.Uint32:
		return false, 32, true
	case types.Uint64:
		return false, 64, true
	case types.Uint, types.Uintptr:
		return false, 64, true
	default:
		return false, 0, false
	}
}

func isNoopPointerConvert(src types.Type, dst types.Type) bool {
	isPointerLike := func(t types.Type) bool {
		switch tt := t.Underlying().(type) {
		case *types.Pointer:
			return true
		case *types.Basic:
			return tt.Kind() == types.UnsafePointer || tt.Kind() == types.Uintptr
		default:
			return false
		}
	}
	return isPointerLike(src) && isPointerLike(dst)
}

func isPhiPairType(t types.Type) bool {
	switch tt := t.(type) {
	case *types.Named:
		if tt.Obj() != nil && tt.Obj().Name() == "Scmer" {
			return true
		}
		return isPhiPairType(tt.Underlying())
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return u.Kind() == types.String
	case *types.Struct:
		// Scmer-like two-word structs.
		return elemSizeOf(t) == 16
	default:
		return false
	}
}

func isPhiTripleType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}

func intTypeName(signed bool, bits int) string {
	if signed {
		switch bits {
		case 8:
			return "int8"
		case 16:
			return "int16"
		case 32:
			return "int32"
		case 64:
			return "int64"
		}
		return ""
	}
	switch bits {
	case 8:
		return "uint8"
	case 16:
		return "uint16"
	case 32:
		return "uint32"
	case 64:
		return "uint64"
	}
	return ""
}

func normalizeIntConstForType(v int64, signed bool, bits int) int64 {
	if bits <= 0 || bits >= 64 {
		return v
	}
	mask := (uint64(1) << uint(bits)) - 1
	u := uint64(v) & mask
	if signed {
		signBit := uint64(1) << uint(bits-1)
		if (u & signBit) != 0 {
			u |= ^mask
		}
	}
	return int64(u)
}

// elemSizeOf returns the size in bytes of a Go type (for array/slice element sizing).
func elemSizeOf(t types.Type) int {
	switch tt := t.Underlying().(type) {
	case *types.Basic:
		switch tt.Kind() {
		case types.String:
			// Go string headers are 2 words: data pointer + length.
			return 16
		case types.Bool, types.Uint8, types.Int8:
			return 1
		case types.Uint16, types.Int16:
			return 2
		case types.Uint32, types.Int32, types.Float32:
			return 4
		case types.Uint64, types.Int64, types.Float64, types.Uint, types.Int, types.Uintptr:
			return 8
		}
	case *types.Struct:
		// For Scmer-like structs (2 pointers = 16 bytes)
		return 16
	case *types.Pointer:
		return 8
	}
	return 8 // default
}

// fieldVarsOf extracts the field variables from a struct type for use with types.Sizes.Offsetsof.
func fieldVarsOf(s *types.Struct) []*types.Var {
	vars := make([]*types.Var, s.NumFields())
	for i := 0; i < s.NumFields(); i++ {
		vars[i] = s.Field(i)
	}
	return vars
}

// isIntegerKind returns true for all integer basic kinds (signed, unsigned, uintptr).
func isIntegerKind(k types.BasicKind) bool {
	switch k {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr:
		return true
	}
	return false
}

func instrDesc(instr ssa.Instruction) string {
	if v, ok := instr.(*ssa.If); ok {
		return fmt.Sprintf("If: %s (cond=%s:%s; expected cond.Loc in {LocImm,LocReg})",
			v, v.Cond.Name(), v.Cond.Type())
	}
	typeName := fmt.Sprintf("%T", instr)
	typeName = strings.TrimPrefix(typeName, "*ssa.")
	return fmt.Sprintf("%s: %s", typeName, instr)
}

// --- patching ---

type patchEntry struct {
	startOff int
	endOff   int
	newText  string
	opName   string
}

func applyPatches(path string, patches []patchEntry) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  error reading %s: %v\n", path, err)
		return
	}

	for i := len(patches) - 1; i >= 0; i-- {
		p := patches[i]
		// Extend endOff past any trailing /* ... */ comment
		endOff := p.endOff
		rest := src[endOff:]
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		if j+1 < len(rest) && rest[j] == '/' && rest[j+1] == '*' {
			if k := strings.Index(string(rest[j:]), "*/"); k >= 0 {
				endOff += j + k + 2
			}
		}

		old := string(src[p.startOff:endOff])
		if old == p.newText {
			continue
		}
		src = append(src[:p.startOff], append([]byte(p.newText), src[endOff:]...)...)
		fmt.Printf("  %s: patched %s\n", path, p.opName)
	}

	// Auto-manage "unsafe" import based on whether generated code uses it
	src = manageUnsafeImport(src)

	if err := os.WriteFile(path, src, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", path, err)
	}
}

// manageUnsafeImport adds or removes `import "unsafe"` based on whether the file uses `unsafe.`.
// needsUnsafeImport returns true if content uses unsafe. outside of comments.
func needsUnsafeImport(content string) bool {
	i := 0
	for i < len(content) {
		// Skip /* ... */ block comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			end := strings.Index(content[i+2:], "*/")
			if end >= 0 {
				i += end + 4
			} else {
				return false // unterminated comment
			}
			continue
		}
		// Skip // line comments
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '/' {
			nl := strings.Index(content[i:], "\n")
			if nl >= 0 {
				i += nl + 1
			} else {
				return false
			}
			continue
		}
		// Skip string literals
		if content[i] == '"' {
			i++
			for i < len(content) && content[i] != '"' {
				if content[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(content) {
				i++
			}
			continue
		}
		// Check for unsafe. in code
		if i+7 <= len(content) && content[i:i+7] == "unsafe." {
			return true
		}
		i++
	}
	return false
}

func manageUnsafeImport(src []byte) []byte {
	content := string(src)
	needsUnsafe := needsUnsafeImport(content)
	// Check both single-line and grouped import forms
	hasUnsafe := strings.Contains(content, `import "unsafe"`) ||
		strings.Contains(content, `"unsafe"`) && strings.Contains(content, "import (")

	if needsUnsafe && !hasUnsafe {
		// Add import "unsafe" after the last import line/block
		pkgIdx := strings.Index(content, "\npackage ")
		if pkgIdx < 0 {
			pkgIdx = strings.Index(content, "package ")
		} else {
			pkgIdx++
		}
		if pkgIdx >= 0 {
			eol := strings.Index(content[pkgIdx:], "\n")
			if eol >= 0 {
				insertPos := pkgIdx + eol + 1
				lastImportEnd := insertPos
				pos := insertPos
				for {
					nlIdx := strings.Index(content[pos:], "\n")
					if nlIdx < 0 {
						break
					}
					line := strings.TrimSpace(content[pos : pos+nlIdx])
					if strings.HasPrefix(line, `import "`) {
						lastImportEnd = pos + nlIdx + 1
					} else if strings.HasPrefix(line, `import (`) {
						closeIdx := strings.Index(content[pos:], "\n)\n")
						if closeIdx >= 0 {
							lastImportEnd = pos + closeIdx + 3
							pos = lastImportEnd
							continue
						}
					} else if line == "" || strings.HasPrefix(line, "//") {
						// blank or comment, keep scanning
					} else {
						break
					}
					pos = pos + nlIdx + 1
				}
				content = content[:lastImportEnd] + "import \"unsafe\"\n" + content[lastImportEnd:]
				fmt.Printf("  added import \"unsafe\"\n")
			}
		}
		return []byte(content)
	} else if !needsUnsafe && hasUnsafe {
		// Remove single-line import "unsafe"
		if strings.Contains(content, "import \"unsafe\"\n") {
			content = strings.Replace(content, "import \"unsafe\"\n", "", 1)
			fmt.Printf("  removed import \"unsafe\"\n")
		}
		// Also handle grouped import: remove "unsafe" line from import ( ... ) blocks
		// Match: \t"unsafe"\n or \n\t"unsafe"\n within import blocks
		if strings.Contains(content, "\t\"unsafe\"\n") {
			content = strings.Replace(content, "\t\"unsafe\"\n", "", 1)
			fmt.Printf("  removed \"unsafe\" from grouped import\n")
		}
		return []byte(content)
	}
	return src
}

// wipeFiles resets JITEmit bodies in the given files to fallback stubs.
// For storage files: replaces JITEmit method bodies with Go call fallback.
// For scm files: resets JIT emit closures to nil.
func wipeFiles(files []string) {
	const jitSig = ") JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {"
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error reading %s: %v\n", path, err)
			continue
		}
		content := string(src)
		changed := false
		searchFrom := 0

		for {
			idx := strings.Index(content[searchFrom:], jitSig)
			if idx < 0 {
				break
			}
			idx += searchFrom // absolute position

			// Find the type name from the receiver: look backwards for "func (s *"
			prefix := content[:idx]
			funcIdx := strings.LastIndex(prefix, "func (s *")
			if funcIdx < 0 {
				searchFrom = idx + len(jitSig)
				continue
			}
			typeName := prefix[funcIdx+len("func (s *"):]

			// Find opening brace
			braceIdx := idx + len(jitSig)
			// Find matching closing brace (handle nested braces)
			depth := 1
			pos := braceIdx
			for pos < len(content) && depth > 0 {
				if content[pos] == '{' {
					depth++
				} else if content[pos] == '}' {
					depth--
				}
				if depth > 0 {
					pos++
				}
			}
			if depth != 0 {
				fmt.Fprintf(os.Stderr, "  %s: unmatched braces in JITEmit for %s\n", path, typeName)
				break
			}

			// Replace body with fallback
			fallback := fmt.Sprintf("\n\treturn ctx.EmitGoCallScalar(scm.GoFuncAddr((*%s).GetValue), []scm.JITValueDesc{thisptr, idx}, 2)\n", typeName)
			content = content[:braceIdx] + fallback + content[pos:]
			fmt.Printf("  %s: wiped %s.JITEmit\n", path, typeName)
			changed = true
			searchFrom = braceIdx + len(fallback)
		}

		if changed {
			// Remove unsafe import if no longer needed
			result := manageUnsafeImport([]byte(content))
			if err := os.WriteFile(path, result, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", path, err)
			}
		} else {
			fmt.Printf("  %s: no JITEmit methods found\n", path)
		}
	}
}
