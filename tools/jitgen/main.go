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
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var dumpOp string
var doPatch bool
var doWipe bool
var verbose bool

const generatedBanner = "/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */"

func main() {
	var files []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-dump=") {
			dumpOp = arg[len("-dump="):]
		} else if arg == "-patch" {
			doPatch = true
		} else if arg == "-wipe" {
			doWipe = true
		} else if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else {
			files = append(files, arg)
		}
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "usage: jitgen [-dump=OP] [-patch] [-wipe] <file.go> ...\n")
		os.Exit(1)
	}

	if doWipe {
		wipeFiles(files)
		return
	}

	// Determine package from file paths
	pkgDir := "./" + filepath.Dir(files[0])

	// Load package with full type info for SSA
	cfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports | packages.NeedName,
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
			msg := e.Error()
			if strings.Contains(msg, "declared and not used") {
				// Regenerating from a temporarily inconsistent generated file is
				// allowed; the patch pass will rewrite these sections.
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
	for fn := range ssautil.AllFunctions(prog) {
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

	// Process each operator (pattern 1: Declare)
	patches := map[string][]patchEntry{}
	for _, op := range ops {
		ssaFn := ssaFuncs[op.funcLit.Pos()]
		if ssaFn == nil {
			fmt.Fprintf(os.Stderr, "  %s: %s — SSA function not found\n", op.path, op.name)
			continue
		}

		if dumpOp == op.name {
			dumpSSA(ssaFn)
		}

		// Single-pass: try to generate, recover on failure
		newText, genErr := generateClosure(op.name, ssaFn)
		if genErr == "" {
			fmt.Printf("  %s: %s OK\n", op.path, op.name)
		} else {
			fmt.Printf("  %s: %s SKIP: %s\n", op.path, op.name, genErr)
			if verbose {
				dumpSSA(ssaFn)
			}
			newText = fmt.Sprintf("nil /* TODO: %s */", genErr)
		}

		if doPatch && len(op.comp.Elts) >= 11 {
			jitField := op.comp.Elts[10]
			pos := fset.Position(jitField.Pos())
			end := fset.Position(jitField.End())
			patches[op.path] = append(patches[op.path], patchEntry{
				startOff: pos.Offset,
				endOff:   end.Offset,
				newText:  newText,
				opName:   op.name,
			})
		}
	}

	// Process each storage type (pattern 2: ColumnStorage.GetValue → JITEmit)
	for _, si := range stInfos {
		ssaFn := ssaFuncs[si.getValuePos]
		if ssaFn == nil {
			fmt.Fprintf(os.Stderr, "  %s: %s.GetValue — SSA function not found\n", si.path, si.typeName)
			continue
		}

		if dumpOp == si.typeName || dumpOp == si.typeName+".GetValue" {
			dumpSSA(ssaFn)
		}

		newText, genErr := generateStorageBody(si.typeName, ssaFn)
		if genErr == "" {
			fmt.Printf("  %s: %s.GetValue OK\n", si.path, si.typeName)
		} else {
			fmt.Printf("  %s: %s.GetValue SKIP: %s\n", si.path, si.typeName, genErr)
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

// --- AST operator collection (for patching byte offsets) ---

type operatorInfo struct {
	name    string
	path    string
	line    int
	funcLit *ast.FuncLit
	comp    *ast.CompositeLit
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
		if !ok || len(comp.Elts) < 7 {
			return true
		}
		nameLit, ok := comp.Elts[0].(*ast.BasicLit)
		if !ok || nameLit.Kind != token.STRING {
			return true
		}
		funcLit, ok := comp.Elts[6].(*ast.FuncLit)
		if !ok {
			return true
		}
		ops = append(ops, operatorInfo{
			name:    strings.Trim(nameLit.Value, "\""),
			path:    path,
			line:    fset.Position(nameLit.Pos()).Line,
			funcLit: funcLit,
			comp:    comp,
		})
		return true
	})
	return ops
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
	marker           string // "_newbool"/"_newint"/"_newfloat" for deferred constructors
	deferredIndexSSA string // SSA name of index operand (for deferred IndexAddr on slices)
	deferredBaseSSA  string // SSA name of base operand for deferred local FieldAddr deref
	offsetExpr       string // Go expression for byte offset from thisptr (for _fieldaddr/_fieldconst markers)
}

type codeGen struct {
	w              strings.Builder
	vals           map[string]genVal
	paramName      string
	nextDesc       int
	nextReg        int
	nextLabel      int
	fn             *ssa.Function
	bbLabels       map[uint64]string // scoped BB id → label var name
	bbDone         map[uint64]bool   // scoped BB id → already generated
	bbQueued       map[uint64]bool   // scoped BB id → queued for future generation
	bbQueue        []int             // queue of BB indices to generate
	bbScope        uint32            // current BB namespace id
	nextBBScope    uint32            // monotonically increasing fallback namespace id
	inlineCallSeq  map[uint64]uint32 // caller scoped-BB id -> inline call ordinal
	phiRegs        map[string]string // SSA phi name → stack offset string (e.g. "0", "8", "16")
	phiPair        map[string]bool   // SSA phi name → true if value occupies 2 words (16 bytes)
	phiStackSize   int               // total bytes reserved on stack for phi nodes (local to current function/inline)
	globalPhiSize  int               // total bytes across ALL phi slots (outer + inlined)
	phiFrameFixup  string            // Go var name for fixup pointer (set by outer allocPhiRegs)
	phiFrameActive bool              // true if an outer phi frame fixup is active (inline should NOT emit SUB RSP)
	curBlock       int               // current BB index being generated
	multiBlock     bool              // true if function has >1 block
	endLabel       string            // label for shared epilogue (multi-block)
	storageMode    bool              // true for ColumnStorage.GetValue pattern (vs Declare pattern)
	typeName       string            // struct type name for FieldAddr (e.g. "StorageInt")

	// Inline call state (non-empty when processing an inlined function)
	inlineReturnReg  string // register var to MOV result into (multi-block inline)
	inlineReturnReg2 string // second register for Scmer pair returns
	inlineReturnsScm bool   // true when current inline callee returns Scmer
	inlineEndLabel   string // label after inlined blocks
	// Top-level multi-block storage returns are merged through a register-based
	// virtual phi (instead of writing result directly in each return block).
	returnPhiReg  string
	returnPhiReg2 string

	// Field deduplication: cache FieldAddr+UnOp deref results by field name
	fieldCache map[string]genVal

	// Reference counting for SSA values (remaining uses)
	refCounts map[string]int

	// SSA name aliases (e.g. Convert no-ops redirect to source)
	ssaAliases map[string]string

	// Top-level package path (the output package, not the inlined callee's package)
	topLevelPkgPath string
	// True for storage GetValue emitters that materialize idxInt/idxPinned vars.
	hasStorageIdx bool

	// Phi register protection: tracks registers protected during phi loads
	// at a block header. Cleared when the first non-Phi instruction is emitted.
	phiProtectedRegVars []string
}

func (g *codeGen) allocDesc() string {
	name := fmt.Sprintf("d%d", g.nextDesc)
	g.nextDesc++
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
	g.emit("\tctx.W.EmitShlRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("\tctx.W.EmitShrRegImm8(%s.Reg, %d)", descVar, shift)
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
	g.emit("\tctx.W.EmitShlRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("\tctx.W.EmitSarRegImm8(%s.Reg, %d)", descVar, shift)
	g.emit("}")
}

func (g *codeGen) emit(format string, a ...any) {
	fmt.Fprintf(&g.w, "\t\t\t"+format+"\n", a...)
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
		// Go slice is 3 words (ptr,len,cap), currently not representable by JITValueDesc.
		return 0
	case *types.Struct:
		sz := types.SizesFor("gc", "amd64").Sizeof(t)
		if sz == 8 {
			return 1
		}
		if sz == 16 {
			return 2
		}
		return 0
	default:
		return 0
	}
}

func (g *codeGen) staticFuncExpr(callee *ssa.Function) (string, bool) {
	if callee == nil || callee.Signature == nil || callee.Signature.Recv() != nil {
		return "", false
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return "", false
	}
	if callee.Pkg.Pkg.Path() == g.topLevelPkgPath {
		return callee.Name(), true
	}
	return callee.Pkg.Pkg.Name() + "." + callee.Name(), true
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
	if params.Len() != len(args) {
		return false
	}
	resolved := make([]genVal, len(args))
	argVars := make([]string, len(args))
	for i, a := range args {
		resolved[i] = g.resolveValue(a)
		argVars[i] = resolved[i].goVar
		switch goCallWordCount(params.At(i).Type()) {
		case 1:
			// If the value is currently a pair, this call shape is not representable.
			g.emit("if %s.Loc == LocRegPair || %s.Loc == LocStackPair {", resolved[i].goVar, resolved[i].goVar)
			g.emit("\tpanic(\"jit: generic call arg expects 1-word value\")")
			g.emit("}")
		case 2:
			// Pair args must be materialized to avoid LocImm flattening to one word.
			g.emit("ctx.EnsureDesc(&%s)", resolved[i].goVar)
			g.emit("if %s.Loc != LocRegPair && %s.Loc != LocStackPair {", resolved[i].goVar, resolved[i].goVar)
			g.emit("\tpanic(\"jit: generic call arg expects 2-word value\")")
			g.emit("}")
		default:
			return false
		}
	}
	argList := strings.Join(argVars, ", ")
	results := sig.Results()
	if results.Len() == 0 {
		g.emit("ctx.EmitGoCallVoid(GoFuncAddr(%s), []JITValueDesc{%s})", funcExpr, argList)
		return true
	}
	if results.Len() != 1 || name == "" {
		return false
	}
	retType := results.At(0).Type()
	retWords := goCallWordCount(retType)
	if retWords != 1 && retWords != 2 {
		return false
	}
	dv := g.allocDesc()
	g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(%s), []JITValueDesc{%s}, %d)", dv, funcExpr, argList, retWords)
	marker := ""
	if bt, ok := retType.Underlying().(*types.Basic); ok && bt.Kind() == types.String {
		marker = "_gostring"
	}
	g.vals[name] = genVal{goVar: dv, isDesc: true, marker: marker}
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
		g.emit("%sctx.W.EmitMovRegImm64(%s, 0)", indent, regExpr)
	case 1:
		// no-op
	case 2:
		g.emit("%sctx.W.EmitAddInt64(%s, %s)", indent, regExpr, regExpr)
	default:
		if fitsInt32(k) {
			g.emit("%sctx.W.EmitImulRegImm32(%s, int32(%d))", indent, regExpr, k)
		} else {
			g.emit("%sctx.W.EmitMovRegImm64(RegR11, uint64(%d))", indent, k)
			g.emit("%sctx.W.EmitImulInt64(%s, RegR11)", indent, regExpr)
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
	g.emit("%s := ctx.W.ReserveLabel()", lbl)
	return lbl
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

// emitEdgePhiMoves emits machine-code-level MOVs for phi edges to targetBB from
// the successor edge succPos of the current block.
func (g *codeGen) emitEdgePhiMoves(targetBBIdx int, succPos int) {
	targetBlock := g.fn.Blocks[targetBBIdx]
	edgeIdx, ok := g.phiEdgeIndexForSucc(targetBBIdx, succPos)
	if !ok {
		return
	}
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		phiReg, ok := g.phiRegs[phi.Name()]
		if !ok {
			continue // no phi reg allocated (shouldn't happen)
		}
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			panic(fmt.Sprintf("phi edge index out of range for %s: edge=%d len=%d", phi.Name(), edgeIdx, len(phi.Edges)))
		}
		edge := phi.Edges[edgeIdx]
		g.emitPhiMov(phiReg, edge, phi.Type())
	}
}

// emitPhiMov emits a machine-code store from an SSA value to a phi stack slot.
// phiOff is the stack offset string (e.g. "0", "8", "16").
func (g *codeGen) emitPhiMov(phiOff string, v ssa.Value, phiType types.Type) {
	phiPair := isPhiPairType(phiType)
	phiOffHi := "(" + phiOff + ")+8"
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOff)
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
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, %s)", phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOff)
			}
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			if signed, bits, ok := intTypeInfo(phiType); ok {
				ival = normalizeIntConstForType(ival, signed, bits)
			}
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(%d)}, %s)", ival, phiOff)
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewFloat(%v)}, %s)", fval, phiOff)
			if phiPair {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
			}
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.isDesc {
			edgeSrc := g.allocDesc()
			g.emit("%s := %s", edgeSrc, src.goVar)
			g.emit("if %s.Loc == LocNone { panic(\"jit: phi source has no location\") }", edgeSrc)
			g.emit("ctx.EnsureDesc(&%s)", edgeSrc)
			if phiPair {
				g.emit("if %s.Loc == LocRegPair || %s.Loc == LocImm {", edgeSrc, edgeSrc)
				g.emit("\tctx.EmitStoreScmerToStack(%s, %s)", edgeSrc, phiOff)
				g.emit("} else {")
				g.emit("\tctx.EmitStoreToStack(%s, %s)", edgeSrc, phiOff)
				g.emit("\tctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOffHi)
				g.emit("}")
				return
			}
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
			panic(fmt.Sprintf("phi edge references unknown value: %s", v))
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
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		phiReg, ok := g.phiRegs[phi.Name()]
		if !ok {
			continue
		}
		_ = indent
		if edgeIdx < 0 || edgeIdx >= len(phi.Edges) {
			panic(fmt.Sprintf("phi edge index out of range for %s: edge=%d len=%d", phi.Name(), edgeIdx, len(phi.Edges)))
		}
		edge := phi.Edges[edgeIdx]
		g.emitPhiMov(phiReg, edge, phi.Type())
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

// allocPhiRegs pre-scans the function for phis and assigns stack slots.
// Phi values live on the stack at [RSP + offset] to avoid register pressure.
// A temp register is allocated on each read and freed after use.
//
// When phiFrameActive is true (inside an inline call), offsets continue from
// globalPhiSize and no SUB RSP is emitted — the outer frame covers all slots.
// When phiFrameActive is false (outer function), a SUB RSP fixup placeholder
// is emitted and patched at the end with the total size.
func (g *codeGen) allocPhiRegs() {
	offset := g.globalPhiSize // continue from global counter
	localStart := offset
	for _, block := range g.fn.Blocks {
		for _, instr := range block.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				break
			}
			phiName := phi.Name()
			pair := isPhiPairType(phi.Type())
			g.phiRegs[phiName] = fmt.Sprintf("%d", offset)
			g.phiPair[phiName] = pair
			if pair {
				offset += 16
			} else {
				offset += 8
			}
		}
	}
	g.phiStackSize = offset - localStart
	g.globalPhiSize = offset

	if !g.phiFrameActive {
		// Outer function: emit SUB RSP fixup placeholder
		if offset > 0 {
			fixup := g.allocReg() // reuse allocReg for unique var names
			g.emit("%s := ctx.W.EmitSubRSP32Fixup()", fixup)
			g.phiFrameFixup = fixup
			g.phiFrameActive = true
		}
	}
	// Inline: no SUB RSP emitted; slots are in the outer frame
}

// inlineCall inlines a callee's SSA into the current code generation.
// The callee's params are mapped to the caller's args, and the callee's
// return value is captured. Returns the genVal representing the result.
func (g *codeGen) inlineCall(callee *ssa.Function, callArgs []ssa.Value) genVal {
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
	savedBBScope := g.bbScope
	savedCurBlock := g.curBlock
	savedPhiRegs := g.phiRegs
	savedPhiPair := g.phiPair
	savedPhiStackSize := g.phiStackSize
	savedVals := g.vals
	savedMultiBlock := g.multiBlock
	savedEndLabel := g.endLabel
	savedInlineReturnReg := g.inlineReturnReg
	savedInlineReturnReg2 := g.inlineReturnReg2
	savedInlineReturnsScm := g.inlineReturnsScm
	savedInlineEndLabel := g.inlineEndLabel
	savedReturnPhiReg := g.returnPhiReg
	savedReturnPhiReg2 := g.returnPhiReg2
	savedRefCounts := g.refCounts
	savedAliases := g.ssaAliases
	savedFieldCache := g.fieldCache
	savedPhiProtected := g.phiProtectedRegVars
	savedTypeName := g.typeName

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
	// Allocate a globally unique namespace for each inline call.
	g.nextBBScope++
	g.bbScope = g.nextBBScope
	g.phiRegs = map[string]string{}
	g.phiPair = map[string]bool{}
	g.vals = map[string]genVal{}
	g.refCounts = computeRefCounts(callee)
	g.ssaAliases = map[string]string{}
	// Do not share cached field loads across inline boundaries:
	// the callee receiver may be a different sub-struct (e.g. multiple inlined
	// StorageInt receivers inside StorageString/StorageSeq).
	g.fieldCache = map[string]genVal{}

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
			copied := arg
			copied.goVar = pv
			g.vals[param.Name()] = copied
		} else {
			g.vals[param.Name()] = arg
		}
	}

	// Protect caller registers across inline boundary.
	// For each argument that the caller still needs after this call:
	// 1. Bump the callee's refcount so it won't destructively modify the register.
	// 2. Mark the register as protected so AllocReg won't spill it.
	// Without both protections, the callee could destroy the caller's live values
	// through destructive ALU operations or register spilling.
	type protectedArg struct {
		activeVar string
		regVar    string
	}
	var protectedArgs []protectedArg
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
		resolved := resolvedArgs[i]
		if resolved.isDesc {
			active := g.allocReg()
			reg := g.allocReg()
			g.emit("%s := %s.Loc == LocReg", active, resolved.goVar)
			g.emit("%s := %s.Reg", reg, resolved.goVar)
			g.emit("if %s { ctx.ProtectReg(%s) }", active, reg)
			protectedArgs = append(protectedArgs, protectedArg{activeVar: active, regVar: reg})
		}
	}

	// Pre-allocate phi regs for callee
	g.allocPhiRegs()

	isMultiBlock := len(callee.Blocks) > 1
	g.multiBlock = isMultiBlock

	// Detect if callee returns Scmer (2-word pair) or scalar (1 word)
	returnsScmer := false
	if results := callee.Signature.Results(); results.Len() == 1 {
		if named, ok := results.At(0).Type().(*types.Named); ok && named.Obj().Name() == "Scmer" {
			returnsScmer = true
		}
	}

	// For multi-block, reserve only an end label.
	// Return registers are allocated lazily on first encountered Return.
	if isMultiBlock {
		g.inlineReturnReg = ""
		g.inlineReturnReg2 = ""
		g.inlineReturnsScm = returnsScmer

		inlineEnd := g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", inlineEnd)
		g.inlineEndLabel = inlineEnd
		g.endLabel = "" // don't use outer endLabel
	} else {
		g.inlineReturnReg = ""
		g.inlineReturnReg2 = ""
		g.inlineReturnsScm = false
		g.inlineEndLabel = ""
		g.endLabel = ""
	}

	// Process callee blocks
	var singleBlockResult genVal
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

		lbl := g.ensureBBLabel(bbIdx)
		g.emit("ctx.W.MarkLabel(%s)", lbl)
		g.resetAllPhiDescsToStack()

		block := callee.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			if ret, ok := instr.(*ssa.Return); ok && !isMultiBlock {
				// Single-block: capture return value directly, no code emitted
				if len(ret.Results) > 0 {
					singleBlockResult = g.resolveValue(ret.Results[0])
				}
				break
			} else {
				g.emitInstr(instr)
				g.freeDeadOperands(instr)
				if _, isRet := instr.(*ssa.Return); isRet {
					break
				}
			}
		}
	}

	if isMultiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.inlineEndLabel)
	}
	// Resolve fixups only once at top-level end. Inline bodies may run while
	// outer-function labels are still pending.
	// Note: no ADD RSP for inlined callee's phis — the unified phi frame
	// is managed by the outer function (allocated via fixup, freed at end).

	// Determine result
	var result genVal
	if isMultiBlock {
		if g.inlineReturnReg == "" {
			panic(fmt.Sprintf("inline callee has no return register: %s", callee))
		}
		dv := g.allocDesc()
		if g.inlineReturnsScm {
			// Wrap the register pair in a JITValueDesc (Scmer = 2 words)
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, g.inlineReturnReg, g.inlineReturnReg2)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnReg, dv)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnReg2, dv)
		} else {
			// Wrap the bare register in a JITValueDesc for type safety
			g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, g.inlineReturnReg)
			g.emit("ctx.BindReg(%s, &%s)", g.inlineReturnReg, dv)
		}
		result = genVal{goVar: dv, isDesc: true}
	} else {
		result = singleBlockResult
	}

	// Unprotect caller registers after inline body completes
	for _, p := range protectedArgs {
		g.emit("if %s { ctx.UnprotectReg(%s) }", p.activeVar, p.regVar)
	}

	// Restore caller state
	g.fn = savedFn
	g.bbQueue = savedBBQueue
	g.bbDone = savedBBDone
	g.bbQueued = savedBBQueued
	g.bbLabels = savedBBLabels
	g.bbScope = savedBBScope
	g.curBlock = savedCurBlock
	g.phiRegs = savedPhiRegs
	g.phiPair = savedPhiPair
	g.phiStackSize = savedPhiStackSize
	g.vals = savedVals
	g.multiBlock = savedMultiBlock
	g.endLabel = savedEndLabel
	g.inlineReturnReg = savedInlineReturnReg
	g.inlineReturnReg2 = savedInlineReturnReg2
	g.inlineReturnsScm = savedInlineReturnsScm
	g.inlineEndLabel = savedInlineEndLabel
	g.returnPhiReg = savedReturnPhiReg
	g.returnPhiReg2 = savedReturnPhiReg2
	g.refCounts = savedRefCounts
	g.ssaAliases = savedAliases
	g.fieldCache = savedFieldCache
	g.phiProtectedRegVars = savedPhiProtected
	g.typeName = savedTypeName

	return result
}

// generateClosure tries to generate a JIT emitter closure for the given SSA function.
// Returns (closureCode, "") on success, or ("", errorDescription) on failure.
func generateClosure(opName string, fn *ssa.Function) (code string, errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			if os.Getenv("JITGEN_DEBUG_PANIC") == "1" && (dumpOp == "" || dumpOp == opName) {
				panic(r)
			}
			code = ""
			errMsg = fmt.Sprintf("%v", r)
		}
	}()

	g := &codeGen{
		vals:            map[string]genVal{},
		fn:              fn,
		bbLabels:        map[uint64]string{},
		bbDone:          map[uint64]bool{},
		bbQueued:        map[uint64]bool{},
		inlineCallSeq:   map[uint64]uint32{},
		phiRegs:         map[string]string{},
		phiPair:         map[string]bool{},
		fieldCache:      map[string]genVal{},
		refCounts:       computeRefCounts(fn),
		ssaAliases:      map[string]string{},
		topLevelPkgPath: fn.Pkg.Pkg.Path(),
	}
	fmt.Fprintf(&g.w, "\t\t%s\n", generatedBanner)
	if len(fn.Params) > 0 {
		g.paramName = fn.Params[0].Name()
	}

	g.multiBlock = len(fn.Blocks) > 1

	// Pre-allocate registers for all phi nodes
	g.allocPhiRegs()

	// For multi-block functions: ensure result has a concrete location,
	// and reserve an end label for the shared epilogue.
	if g.multiBlock {
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.endLabel = g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", g.endLabel)
	}

	// Process BBs via queue, starting from BB0
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

		// Ensure every BB has a concrete entry label, even if first reached by
		// queue/fallthrough before any forward jump reserved one.
		lbl := g.ensureBBLabel(bbIdx)
		g.emit("ctx.W.MarkLabel(%s)", lbl)
		g.resetAllPhiDescsToStack()

		block := fn.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			g.emitInstr(instr)
			g.freeDeadOperands(instr)
			if _, isRet := instr.(*ssa.Return); isRet {
				break
			}
		}
	}

	// Emit fixup resolution and epilogue
	if g.multiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.endLabel)
		g.emit("ctx.W.ResolveFixups()")
	}
	// Deallocate unified phi stack frame (patch fixup + emit cleanup)
	if g.globalPhiSize > 0 {
		g.emit("ctx.W.PatchInt32(%s, int32(%d))", g.phiFrameFixup, g.globalPhiSize)
		g.emit("ctx.W.EmitAddRSP32(int32(%d))", g.globalPhiSize)
	}
	if g.multiBlock {
		g.emit("return result")
	}

	result := fmt.Sprintf("func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {\n%s\t\t}",
		injectBindRegCalls(g.w.String()))
	return result, ""
}

// generateStorageBody generates the body of a JITEmit method from GetValue SSA.
// The generated code lives inside:
//
//	func (s *StorageXxx) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc { ... }
func generateStorageBody(typeName string, fn *ssa.Function) (code string, errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			code = ""
			errMsg = fmt.Sprintf("%v", r)
		}
	}()
	g := &codeGen{
		vals:            map[string]genVal{},
		fn:              fn,
		bbLabels:        map[uint64]string{},
		bbDone:          map[uint64]bool{},
		bbQueued:        map[uint64]bool{},
		inlineCallSeq:   map[uint64]uint32{},
		phiRegs:         map[string]string{},
		phiPair:         map[string]bool{},
		fieldCache:      map[string]genVal{},
		refCounts:       computeRefCounts(fn),
		ssaAliases:      map[string]string{},
		storageMode:     true,
		typeName:        typeName,
		topLevelPkgPath: fn.Pkg.Pkg.Path(),
	}
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
	if len(fn.Params) >= 2 {
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
		g.emit("\tctx.W.EmitShlRegImm8(idxInt.Reg, 32)")
		g.emit("\tctx.W.EmitShrRegImm8(idxInt.Reg, 32)")
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

	// Pre-allocate registers for all phi nodes
	g.allocPhiRegs()

	if g.multiBlock {
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		// Register-based return phi for multi-block storage emitters.
		g.returnPhiReg = g.allocReg()
		g.returnPhiReg2 = g.allocReg()
		g.emit("%s := ctx.AllocReg()", g.returnPhiReg)
		g.emit("%s := ctx.AllocRegExcept(%s)", g.returnPhiReg2, g.returnPhiReg)
		g.endLabel = g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", g.endLabel)
	}

	// Process BBs via queue, starting from BB0
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

		lbl := g.ensureBBLabel(bbIdx)
		g.emit("ctx.W.MarkLabel(%s)", lbl)
		g.resetAllPhiDescsToStack()

		block := fn.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			g.emitInstr(instr)
			g.freeDeadOperands(instr)
			if _, isRet := instr.(*ssa.Return); isRet {
				break
			}
		}
	}

	if g.multiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.endLabel)
		if g.returnPhiReg != "" && g.returnPhiReg2 != "" {
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, g.returnPhiReg, g.returnPhiReg2)
			g.emit("ctx.EmitMovPairToResult(&%s, &result)", dv)
			g.emit("ctx.FreeReg(%s)", g.returnPhiReg)
			g.emit("ctx.FreeReg(%s)", g.returnPhiReg2)
		}
		g.emit("ctx.W.ResolveFixups()")
	}
	if g.hasStorageIdx {
		g.emit("if idxPinned { ctx.UnprotectReg(idxPinnedReg) }")
		if !g.multiBlock {
			// Keep Go return-checker happy when single-block body already emitted
			// returns above: this tail return is unreachable but well-formed.
			g.emit("return result")
		}
	}
	// Deallocate unified phi stack frame (patch fixup + emit cleanup)
	if g.globalPhiSize > 0 {
		g.emit("ctx.W.PatchInt32(%s, int32(%d))", g.phiFrameFixup, g.globalPhiSize)
		g.emit("ctx.W.EmitAddRSP32(int32(%d))", g.globalPhiSize)
	}
	if g.multiBlock {
		g.emit("return result")
	}

	code = g.w.String()
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
		"LocNone": true, "LocReg": true, "LocRegPair": true,
		"LocStack": true, "LocStackPair": true, "LocMem": true, "LocImm": true, "LocAny": true,
		"NewInt": true, "NewFloat": true, "NewBool": true, "NewNil": true, "NewString": true,
		"NewFastDict": true, "NewFastDictValue": true,
		"Scmer": true, "GoFuncAddr": true, "JITBuildMergeClosure": true,
		"EnsureDesc":                   true,
		"ConcatStrings":                true,
		"OptimizeProcToSerialFunction": true,
		"CcE":                          true, "CcNE": true, "CcL": true, "CcG": true, "CcLE": true, "CcGE": true,
		"CcB": true, "CcAE": true, "CcBE": true, "CcA": true,
		"RegRAX": true, "RegRBX": true, "RegRCX": true, "RegRDX": true,
		"RegRSI": true, "RegRDI": true, "RegRSP": true, "RegRBP": true,
		"RegR8": true, "RegR9": true, "RegR10": true, "RegR11": true,
		"RegR12": true, "RegR13": true, "RegR14": true, "RegR15": true,
		"RegX0": true, "RegX1": true, "RegX2": true, "RegX3": true, "RegX4": true, "RegX5": true,
	}
	// Map unexported tag constants to their exported equivalents
	scmTagMap := map[string]string{
		"tagNil": "scm.TagNil", "tagBool": "scm.TagBool", "tagInt": "scm.TagInt",
		"tagFloat": "scm.TagFloat", "tagString": "scm.TagString",
		"tagSlice": "scm.TagSlice", "tagFastDict": "scm.TagFastDict",
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
	for phiName, phiOff := range g.phiRegs {
		gv, ok := g.vals[phiName]
		if !ok || !gv.isDesc {
			// Phi descriptors are declared lazily when lowering the phi
			// instruction itself. Skip unseen phis here to avoid generating
			// unused temporary declarations in functions where some phi values
			// are not materialized on a given path.
			continue
		}
		if g.phiPair[phiName] {
			g.emit("%s = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s)}", gv.goVar, phiOff)
		} else {
			g.emit("%s = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(%s)}", gv.goVar, phiOff)
		}
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
		} else if _, isGlobal := v.X.(*ssa.Global); isGlobal {
			// IndexAddr on a global array/slice (e.g. &pow10f[idx])
			globalName := v.X.Name()
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
			if strings.HasPrefix(src.marker, "_fieldaddr:array:") {
				// Direct indexing on receiver array field address, e.g. &s.thresholds[i].
				idxVal := g.resolveValue(v.Index)
				elemType := v.Type().Underlying().(*types.Pointer).Elem().Underlying()
				elemSize := elemSizeOf(elemType)
				baseDesc := g.allocDesc()
				baseReg := g.allocReg()
				g.emit("var %s JITValueDesc", baseDesc)
				g.emit("%s := ctx.AllocReg()", baseReg)
				g.emit("if thisptr.Loc == LocImm {")
				g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(uintptr(thisptr.Imm.Int()) + %s))", baseReg, src.offsetExpr)
				g.emit("} else {")
				g.emit("\tctx.W.EmitMovRegReg(%s, thisptr.Reg)", baseReg)
				g.emit("\tctx.W.EmitAddRegImm32(%s, int32(%s))", baseReg, src.offsetExpr)
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

		// Extract field info from SSA types
		ptrType := v.X.Type().Underlying().(*types.Pointer)
		structType := ptrType.Elem().Underlying().(*types.Struct)
		field := structType.Field(v.Field)
		fieldName := field.Name()
		fieldType := field.Type().Underlying()

		// Determine the offset expression and struct type name for this field
		var offsetExpr string
		var cacheKey string
		var isImmutable bool
		localFieldAddr := false

		if src.marker == "_storage_recv" {
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

		// Create marker with offsetExpr
		if localFieldAddr {
			g.vals[name] = genVal{
				goVar:           src.goVar,
				marker:          "_fieldaddrlocal:" + sizeStr,
				offsetExpr:      offsetExpr,
				deferredBaseSSA: v.X.Name(),
			}
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
			g.emit("\t\tctx.W.EmitMovRegImm64(%s, 0)", negReg)
			g.emit("\t\tctx.W.EmitSubFloat64(%s, %s.Reg)", negReg, src.goVar)
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s}", dv, negReg)
			g.emit("\t} else {")
			negIntReg := g.allocReg()
			g.emit("\t\t%s := ctx.AllocRegExcept(%s.Reg)", negIntReg, src.goVar)
			g.emit("\t\tctx.W.EmitMovRegImm64(%s, 0)", negIntReg)
			g.emit("\t\tctx.W.EmitSubInt64(%s, %s.Reg)", negIntReg, src.goVar)
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
			g.emit("\t\tctx.W.EmitMovRegReg(negReg, %s.Reg2)", src.goVar)
			g.emit("\t\tctx.W.EmitAndRegImm32(negReg, 1)")
			g.emit("\t\tctx.W.EmitCmpRegImm32(negReg, 0)")
			g.emit("\t\tctx.W.EmitSetcc(negReg, CcE)")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}", dv)
			g.emit("\t} else if %s.Loc == LocReg {", src.goVar)
			g.emit("\t\tctx.W.EmitMovRegReg(negReg, %s.Reg)", src.goVar)
			g.emit("\t\tctx.W.EmitAndRegImm32(negReg, 1)")
			g.emit("\t\tctx.W.EmitCmpRegImm32(negReg, 0)")
			g.emit("\t\tctx.W.EmitSetcc(negReg, CcE)")
			g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}", dv)
			g.emit("\t} else {")
			g.emit("\t\tpanic(\"UnOp ! unsupported source location\")")
			g.emit("\t}")
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if v.Op == token.MUL {
			src := g.vals[v.X.Name()]
			if strings.HasPrefix(src.marker, "_descfield:") {
				// Deref of descriptor-backed Scmer field address.
				// marker format: "_descfield:<ptr|aux>:<descVar>"
				parts := strings.SplitN(src.marker, ":", 3)
				fieldName := parts[1]
				base := parts[2]
				dv := g.allocDesc()
				g.emit("var %s JITValueDesc", dv)
				g.emit("ctx.EnsureDesc(&%s)", base)
				g.emit("if %s.Loc == LocImm {", base)
				g.emit("\tptrWord, auxWord := %s.Imm.RawWords()", base)
				if fieldName == "ptr" {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}", dv)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}", dv)
				}
				g.emit("} else {")
				g.emit("\tif %s.Loc != LocRegPair { panic(\"jitgen: desc field base is not LocRegPair\") }", base)
				rv := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv)
				if fieldName == "ptr" {
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", rv, base)
				} else {
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg2)", rv, base)
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
				case "1":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem8(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem16(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem32(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocReg()", lenReg)
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", ptrReg)
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr+8)", lenReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, lenReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(fieldAddr))", ptrReg)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg)
				}
				g.emit("} else {")
				g.emit("\toff := int32(%s)", src.offsetExpr)
				g.emit("\tbaseReg := %s.Reg", base)
				switch sizeStr {
				case "1":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.W.EmitMovRegMemB(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.W.EmitMovRegMemW(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.W.EmitMovRegMemL(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", rv)
					g.emit("\tctx.W.EmitMovRegMem(%s, baseReg, off)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", ptrReg)
					g.emit("\t%s := ctx.AllocRegExcept(baseReg, %s)", lenReg, ptrReg)
					g.emit("\tctx.W.EmitMovRegMem(%s, baseReg, off)", ptrReg)
					g.emit("\tctx.W.EmitMovRegMem(%s, baseReg, off+8)", lenReg)
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, lenReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(baseReg)", ptrReg)
					g.emit("\tctx.W.EmitMovRegReg(%s, baseReg)", ptrReg)
					g.emit("\tctx.W.EmitAddRegImm32(%s, off)", ptrReg)
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
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(dataPtr))", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s, StackOff: int32(sliceLen)}", dv, ptrReg2)
					g.emit("} else {")
					// Register receiver: load data pointer from field.
					g.emit("\toff := int32(%s)", src.offsetExpr)
					g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg2)
					g.emit("}")
					g.emit("ctx.BindReg(%s, &%s)", ptrReg2, dv)
					gv := genVal{goVar: dv, isDesc: true, marker: "_slice"}
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
				g.emit("\tctx.W.%s(%s, thisptr.Reg, off)", emitLoadRel, rv)
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
					g.emit("\tctx.W.EmitMovRegMem8(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "2":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem16(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "4":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem32(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "8":
					rv := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", rv)
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", rv)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
				case "slice":
					// Load slice header: ptr (8 bytes), len (8 bytes), cap (8 bytes)
					ptrReg := g.allocReg()
					lenReg := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\t%s := ctx.AllocReg()", lenReg)
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", ptrReg)   // data ptr
					g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr+8)", lenReg) // length
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, lenReg)
				case "array":
					ptrReg := g.allocReg()
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\t%s := ctx.AllocReg()", ptrReg)
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(fieldAddr))", ptrReg)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg)
				}
				g.emit("} else {")
				// thisptr is in a register → emit register-relative loads
				g.emit("\toff := int32(%s)", src.offsetExpr)
				switch sizeStr {
				case "1":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.W.EmitMovRegMemB(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "2":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.W.EmitMovRegMemW(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "4":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.W.EmitMovRegMemL(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "8":
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", rv2)
					g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", rv2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv2)
				case "slice":
					ptrReg2 := g.allocReg()
					lenReg2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg2)
					g.emit("\t%s := ctx.AllocReg()", lenReg2)
					g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg2)   // data ptr
					g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off+8)", lenReg2) // length
					g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg2, lenReg2)
				case "array":
					ptrReg2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg2)
					g.emit("\tctx.W.EmitMovRegReg(%s, thisptr.Reg)", ptrReg2)
					g.emit("\tctx.W.EmitAddRegImm32(%s, off)", ptrReg2)
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
				dv := g.allocDesc()
				scratch := g.allocReg()
				g.emit("%s := ctx.AllocReg()", scratch)
				// Both index and base descriptor can be spilled between SSA steps.
				// Always materialize before touching .Reg.
				g.emit("ctx.EnsureDesc(&%s)", src.argIdxVar)
				g.emit("ctx.EnsureDesc(&%s)", sliceDescVar)
				// Compute byte offset: idx * elemSize
				g.emit("if %s.Loc == LocImm {", src.argIdxVar)
				switch elemSize {
				case "8":
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()) * 8)", scratch, src.argIdxVar)
				case "16":
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()) * 16)", scratch, src.argIdxVar)
				default:
					g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()) * %s)", scratch, src.argIdxVar, elemSize)
				}
				g.emit("} else {")
				g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", scratch, src.argIdxVar)
				switch elemSize {
				case "8":
					g.emit("\tctx.W.EmitShlRegImm8(%s, 3)", scratch) // *8
				case "16":
					g.emit("\tctx.W.EmitShlRegImm8(%s, 4)", scratch) // *16
				default:
					g.emit("\t// multiply by %s", elemSize)
					g.emit("\tscratch2 := ctx.AllocReg()")
					g.emit("\tctx.W.EmitMovRegImm64(scratch2, %s)", elemSize)
					g.emit("\tctx.W.EmitImulInt64(%s, scratch2)", scratch)
					g.emit("\tctx.FreeReg(scratch2)")
				}
				g.emit("}")
				// Add base pointer (handle LocImm for immutable slices: use R11 scratch)
				g.emit("if %s.Loc == LocImm {", sliceDescVar)
				g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", sliceDescVar)
				g.emit("\tctx.W.EmitAddInt64(%s, RegR11)", scratch)
				g.emit("} else {")
				g.emit("\tctx.W.EmitAddInt64(%s, %s.Reg)", scratch, sliceDescVar)
				g.emit("}")
				switch elemSize {
				case "8":
					// Single 8-byte element → LocReg
					// AllocRegExcept(scratch) prevents the eviction-alias bug: AllocReg might
					// otherwise return scratch itself, making EmitMovRegMem a self-load.
					rv := g.allocReg()
					g.emit("%s := ctx.AllocRegExcept(%s)", rv, scratch)
					g.emit("ctx.W.EmitMovRegMem(%s, %s, 0)", rv, scratch)
					g.emit("ctx.FreeReg(%s)", scratch)
					g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, rv)
					g.vals[name] = genVal{goVar: dv, isDesc: true}
				default:
					// 16-byte element (Scmer pair) → LocRegPair
					// AllocRegExcept prevents scratch from being evicted for either alloc.
					ptrReg := g.allocReg()
					auxReg := g.allocReg()
					g.emit("%s := ctx.AllocRegExcept(%s)", ptrReg, scratch)
					g.emit("%s := ctx.AllocRegExcept(%s, %s)", auxReg, scratch, ptrReg)
					g.emit("ctx.W.EmitMovRegMem(%s, %s, 0)", ptrReg, scratch)
					g.emit("ctx.W.EmitMovRegMem(%s, %s, 8)", auxReg, scratch)
					g.emit("ctx.FreeReg(%s)", scratch)
					g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
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
				g.emit("%s := ctx.AllocReg()", scratch)
				// Load base address of global array at compile time
				g.emit("ctx.W.EmitMovRegImm64(%s, uint64(uintptr(unsafe.Pointer(&%s[0]))))", scratch, globalName)
				// Compute byte offset: idx * elemSize, add to base
				idxReg := g.allocReg()
				g.emit("%s := ctx.AllocReg()", idxReg)
				g.emit("if %s.Loc == LocImm {", src.argIdxVar)
				g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()) * %s)", idxReg, src.argIdxVar, elemSize)
				g.emit("} else {")
				g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", idxReg, src.argIdxVar)
				switch elemSize {
				case "8":
					g.emit("\tctx.W.EmitShlRegImm8(%s, 3)", idxReg) // *8
				default:
					g.emit("\tscratch2 := ctx.AllocReg()")
					g.emit("\tctx.W.EmitMovRegImm64(scratch2, %s)", elemSize)
					g.emit("\tctx.W.EmitImulInt64(%s, scratch2)", idxReg)
					g.emit("\tctx.FreeReg(scratch2)")
				}
				g.emit("}")
				// Add base pointer
				g.emit("ctx.W.EmitAddInt64(%s, %s)", scratch, idxReg)
				g.emit("ctx.FreeReg(%s)", idxReg)
				// Load value
				// Protect scratch so AllocReg cannot spill it and alias rv==scratch.
				rv := g.allocReg()
				g.emit("%s := ctx.AllocRegExcept(%s)", rv, scratch)
				g.emit("ctx.W.EmitMovRegMem(%s, %s, 0)", rv, scratch)
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
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else if src.argIdxVar != "" {
				// Variable-index IndexAddr+Deref on emitter args.
				// If the index is known at emit-time, reuse args[idx] directly.
				// Otherwise, emit runtime selection across the fixed args list.
				dv := g.allocDesc()
				idxDescVar := src.argIdxVar
				ptrReg := g.allocReg()
				auxReg := g.allocReg()
				doneLbl := g.allocLabel()
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", idxDescVar)
				g.emit("\tidx := int(%s.Imm.Int())", idxDescVar)
				g.emit("\tif idx < 0 || idx >= len(args) {")
				g.emit("\t\tpanic(\"jitgen: dynamic args index out of range\")")
				g.emit("\t}")
				g.emit("\t%s = args[idx]", dv)
				g.emit("} else {")
				g.emit("\tprotected := make([]Reg, 0, len(args)*2+1)")
				g.emit("\tseen := make(map[Reg]bool)")
				g.emit("\tif !seen[%s.Reg] {", idxDescVar)
				g.emit("\t\tctx.ProtectReg(%s.Reg)", idxDescVar)
				g.emit("\t\tseen[%s.Reg] = true", idxDescVar)
				g.emit("\t\tprotected = append(protected, %s.Reg)", idxDescVar)
				g.emit("\t}")
				g.emit("\tfor _, ai := range args {")
				g.emit("\t\tif ai.Loc == LocReg {")
				g.emit("\t\t\tif !seen[ai.Reg] {")
				g.emit("\t\t\t\tctx.ProtectReg(ai.Reg)")
				g.emit("\t\t\t\tseen[ai.Reg] = true")
				g.emit("\t\t\t\tprotected = append(protected, ai.Reg)")
				g.emit("\t\t\t}")
				g.emit("\t\t} else if ai.Loc == LocRegPair {")
				g.emit("\t\t\tif !seen[ai.Reg] {")
				g.emit("\t\t\t\tctx.ProtectReg(ai.Reg)")
				g.emit("\t\t\t\tseen[ai.Reg] = true")
				g.emit("\t\t\t\tprotected = append(protected, ai.Reg)")
				g.emit("\t\t\t}")
				g.emit("\t\t\tif !seen[ai.Reg2] {")
				g.emit("\t\t\t\tctx.ProtectReg(ai.Reg2)")
				g.emit("\t\t\t\tseen[ai.Reg2] = true")
				g.emit("\t\t\t\tprotected = append(protected, ai.Reg2)")
				g.emit("\t\t\t}")
				g.emit("\t\t} else if ai.Loc == LocStackPair {")
				g.emit("\t\t\t// no direct registers to protect")
				g.emit("\t\t}")
				g.emit("\t}")
				g.emit("\t%s := ctx.AllocReg()", ptrReg)
				g.emit("\t%s := ctx.AllocRegExcept(%s)", auxReg, ptrReg)
				g.emit("\t%s := ctx.W.ReserveLabel()", doneLbl)
				g.emit("\tfor i := 0; i < len(args); i++ {")
				g.emit("\t\tnextLbl := ctx.W.ReserveLabel()")
				g.emit("\t\tctx.W.EmitCmpRegImm32(%s.Reg, int32(i))", idxDescVar)
				g.emit("\t\tctx.W.EmitJcc(CcNE, nextLbl)")
				g.emit("\t\tai := args[i]")
				g.emit("\t\tswitch ai.Loc {")
				g.emit("\t\tcase LocRegPair:")
				g.emit("\t\t\tctx.W.EmitMovRegReg(%s, ai.Reg)", ptrReg)
				g.emit("\t\t\tctx.W.EmitMovRegReg(%s, ai.Reg2)", auxReg)
				g.emit("\t\tcase LocStackPair:")
				g.emit("\t\t\ttmp := ai")
				g.emit("\t\t\tctx.EnsureDesc(&tmp)")
				g.emit("\t\t\tif tmp.Loc != LocRegPair {")
				g.emit("\t\t\t\tpanic(\"jitgen: emitter args index expected Scmer pair\")")
				g.emit("\t\t\t}")
				g.emit("\t\t\tctx.W.EmitMovRegReg(%s, tmp.Reg)", ptrReg)
				g.emit("\t\t\tctx.W.EmitMovRegReg(%s, tmp.Reg2)", auxReg)
				g.emit("\t\t\tctx.FreeDesc(&tmp)")
				g.emit("\t\tcase LocImm:")
				g.emit("\t\t\tpair := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", ptrReg, auxReg)
				g.emit("\t\t\tif ai.Imm.GetTag() == tagInt {")
				g.emit("\t\t\t\tsrc := ai")
				g.emit("\t\t\t\tsrc.Type = tagInt")
				g.emit("\t\t\t\tsrc.Imm = NewInt(ai.Imm.Int())")
				g.emit("\t\t\t\tctx.W.EmitMakeInt(pair, src)")
				g.emit("\t\t\t} else if ai.Imm.GetTag() == tagFloat {")
				g.emit("\t\t\t\tsrc := ai")
				g.emit("\t\t\t\tsrc.Type = tagFloat")
				g.emit("\t\t\t\tsrc.Imm = NewFloat(ai.Imm.Float())")
				g.emit("\t\t\t\tctx.W.EmitMakeFloat(pair, src)")
				g.emit("\t\t\t} else if ai.Imm.GetTag() == tagBool {")
				g.emit("\t\t\t\tsrc := ai")
				g.emit("\t\t\t\tsrc.Type = tagBool")
				g.emit("\t\t\t\tsrc.Imm = NewBool(ai.Imm.Bool())")
				g.emit("\t\t\t\tctx.W.EmitMakeBool(pair, src)")
				g.emit("\t\t\t} else if ai.Imm.GetTag() == tagNil {")
				g.emit("\t\t\t\tctx.W.EmitMakeNil(pair)")
				g.emit("\t\t\t} else {")
				g.emit("\t\t\t\tptrWord, auxWord := ai.Imm.RawWords()")
				g.emit("\t\t\t\tctx.W.EmitMovRegImm64(%s, uint64(ptrWord))", ptrReg)
				g.emit("\t\t\t\tctx.W.EmitMovRegImm64(%s, auxWord)", auxReg)
				g.emit("\t\t\t}")
				g.emit("\t\tdefault:")
				g.emit("\t\t\tpanic(\"jitgen: emitter args index expected Scmer pair\")")
				g.emit("\t\t}")
				g.emit("\t\tctx.W.EmitJmp(%s)", doneLbl)
				g.emit("\t\tctx.W.MarkLabel(nextLbl)")
				g.emit("\t}")
				g.emit("\tctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range")
				g.emit("\tctx.W.MarkLabel(%s)", doneLbl)
				g.emit("\tfor _, r := range protected {")
				g.emit("\t\tctx.UnprotectReg(r)")
				g.emit("\t}")
				g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
				g.emit("}")
				g.vals[name] = genVal{goVar: dv, isDesc: true}
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
					if src.marker == "_slice" || src.marker == "_gostring" || src.isDesc {
						dv := g.allocDesc()
						g.emit("var %s JITValueDesc", dv)
						g.emit("if %s.Loc == LocImm {", src.goVar)
						if src.marker == "_gostring" {
							// LocImm Scmer string constant: derive Go-string length.
							g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(%s.Imm.String())))}", dv, src.goVar)
						} else {
							// Legacy LocImm slice path stores length in StackOff.
							g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.StackOff))}", dv, src.goVar)
						}
						g.emit("} else {")
						g.emit("\tctx.EnsureDesc(&%s)", src.goVar)
						g.emit("\tif %s.Loc == LocRegPair {", src.goVar)
						g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", dv, src.goVar)
						g.emit("\t\tctx.BindReg(%s.Reg2, &%s)", src.goVar, dv)
						g.emit("\t} else if %s.Loc == LocReg {", src.goVar)
						g.emit("\t\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, src.goVar)
						g.emit("\t\tctx.BindReg(%s.Reg, &%s)", src.goVar, dv)
						g.emit("\t} else {")
						g.emit("\t\tpanic(\"len on unsupported descriptor location\")")
						g.emit("\t}")
						g.emit("}")
						g.vals[name] = genVal{goVar: dv, isDesc: true}
					} else {
						panic(fmt.Sprintf("len on non-parameter: %s", v))
					}
				}
			default:
				panic(fmt.Sprintf("unsupported builtin: %s", builtin.Name()))
			}
			break
		}
		callee := v.Call.StaticCallee()
		if callee == nil {
			panic(fmt.Sprintf("dynamic call: %s", v))
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
				g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", rv)
				g.emit("} else {")
				g.emit("\toff := int32(%s)", arg.offsetExpr)
				g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", rv)
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
				g.emit("\tctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + %s))", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(RegR11, baseReg, 0)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, baseReg, 0)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.FreeReg(baseReg)")
				g.emit("} else {")
				g.emit("\toff := int32(%s)", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(RegR11, thisptr.Reg, off)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, thisptr.Reg, off)", val.goVar)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("StoreInt64 dst is not a field address: marker=%q", dst.marker))
			}
			break
		}
		switch callee.Name() {
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
			g.emit("%s := ctx.EmitTagEquals(&%s, tagNil, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsInt":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagInt, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFloat":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagFloat, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsBool":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagBool, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsString":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagString, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagSlice, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFastDict":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			tmp := g.allocDesc()
			g.emit("%s := %s", tmp, arg.goVar)
			g.emit("%s := ctx.EmitTagEquals(&%s, tagFastDict, JITValueDesc{Loc: LocAny})", dv, tmp)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Bool":
			// (Scmer).Bool() — extract bool from Scmer.
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Bool())}", dv, arg.goVar)
			g.emit("} else if %s.Type == tagBool && %s.Loc == LocRegPair {", arg.goVar, arg.goVar)
			g.emit("\tctx.FreeReg(%s.Reg)", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s.Reg2}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg2, &%s)", arg.goVar, dv)
			g.emit("} else if %s.Type == tagBool && %s.Loc == LocReg {", arg.goVar, arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s.Reg}", dv, arg.goVar)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", arg.goVar, dv)
			g.emit("} else {")
			g.emit("\t%s = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.emit("\t%s.Type = tagBool", dv)
			g.emit("\tctx.BindReg(%s.Reg, &%s)", dv, dv)
			g.emit("}")
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
			g.emit("if %s.Loc != LocImm && %s.Type == JITTypeUnknown {", arg.goVar, arg.goVar)
			g.emit("\tpanic(\"jit: Scmer.String on unknown dynamic type\")")
			g.emit("}")
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{%s}, 2)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
		case "NewBool":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newbool"}
		case "NewInt":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newint"}
		case "NewFloat":
			src := g.resolveValue(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newfloat"}
		case "NewNil":
			g.vals[name] = genVal{goVar: "", marker: "_newnil"}
		case "NewString":
			// NewString(s string) Scmer — arg is a Go string (2 words: ptr+len), result is Scmer (2 words)
			arg := g.vals[v.Call.Args[0].Name()]
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: arg.goVar, isDesc: arg.isDesc, marker: "_newstring"}
		case "NewFastDict":
			// NewFastDict(fd *FastDict) Scmer — construct Scmer from *FastDict ptr
			// arg: 1 word (raw pointer), result: 2 words (Scmer)
			src := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\tpanic(\"NewFastDict: LocImm not expected at JIT compile time\")")
			g.emit("} else {")
			auxReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", auxReg)
			g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(tagFastDict) << 48)", auxReg)
			g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: %s.Reg, Reg2: %s}", dv, src.goVar, auxReg)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "NewFastDictValue":
			// NewFastDictValue(cap int) *FastDict — Go call, returns 1 word
			arg := g.resolveValue(v.Call.Args[0])
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "OptimizeProcToSerialFunction":
			// OptimizeProcToSerialFunction(Scmer) func(...Scmer) Scmer
			// arg: Scmer (2 words), result: func value (1 word)
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(OptimizeProcToSerialFunction), []JITValueDesc{%s}, 1)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "FastDict":
			// (Scmer).FastDict() *FastDict — extract ptr field, free aux
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\tpanic(\"FastDict: LocImm not expected at JIT compile time\")")
			g.emit("} else {")
			g.emit("\tctx.FreeReg(%s.Reg2)", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s.Reg}", dv, arg.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Set":
			// (*FastDict).Set(fd, key, value, mergeFn) — void Go call
			recv := g.vals[v.Call.Args[0].Name()]     // *FastDict (1 word)
			key := g.vals[v.Call.Args[1].Name()]      // Scmer (2 words)
			val := g.vals[v.Call.Args[2].Name()]      // Scmer (2 words)
			mergeFn := g.resolveValue(v.Call.Args[3]) // func (1 word)
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
			g.emit("\tctx.W.EmitCvtFloatBitsToInt64(truncInt, truncSrc)")
			g.emit("\tctx.W.EmitCvtInt64ToFloat64(RegX0, truncInt)")
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
			// (Scmer).Slice() []Scmer — extract data ptr and length from Scmer
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			// ptr field = data pointer (Reg), aux lower 48 bits = length
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\tslice := %s.Imm.Slice()", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}", dv)
			g.emit("} else {")
			// Extract length from aux: AND with mask, SHR not needed (auxVal = aux & ((1<<48)-1))
			lenReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", lenReg)
			g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg2)", lenReg, arg.goVar)
			g.emit("\tctx.W.EmitShlRegImm8(%s, 16)", lenReg) // clear top 16 bits (tag)
			g.emit("\tctx.W.EmitShrRegImm8(%s, 16)", lenReg)
			g.emit("\tctx.FreeReg(%s.Reg2)", arg.goVar) // free aux
			g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s.Reg, Reg2: %s}", dv, arg.goVar, lenReg)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_slice"}
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
				g.emit("\tctx.W.EmitMovRegMem64(%s, fieldAddr)", rv)
				g.emit("} else {")
				g.emit("\toff := int32(%s)", arg.offsetExpr)
				g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", rv)
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
				g.emit("\tctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + %s))", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(RegR11, baseReg, 0)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, baseReg, 0)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.FreeReg(baseReg)")
				g.emit("} else {")
				g.emit("\toff := int32(%s)", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(RegR11, thisptr.Reg, off)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, thisptr.Reg, off)", val.goVar)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("StoreInt64 dst is not a field address: marker=%q", dst.marker))
			}
		default:
			if g.emitGenericStaticCall(name, callee, v.Call.Args) {
				break
			}
			// Try to inline: if callee SSA is available, inline its body
			if callee.Blocks != nil {
				result := g.inlineCall(callee, v.Call.Args)
				if name != "" {
					g.vals[name] = result
				}
			} else {
				panic(fmt.Sprintf("unsupported call: %s", v))
			}
		}

	case *ssa.BinOp:
		// Check for string concatenation before integer path
		if v.Op == token.ADD {
			if basic, ok := v.X.Type().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
				// String concatenation: call runtime concat function
				xVal := g.resolveValue(v.X)
				yVal := g.resolveValue(v.Y)
				dv := g.allocDesc()
				g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(ConcatStrings), []JITValueDesc{%s, %s}, 2)", dv, xVal.goVar, yVal.goVar)
				g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
				break
			}
		}
		xVal := g.resolveValue(v.X)
		// Check if v.X has more remaining uses (excluding this one).
		// If so, destructive operations must copy before modifying.
		xMultiUse := false
		if _, isConst := v.X.(*ssa.Const); !isConst {
			xMultiUse = g.ssaValueUsesRemaining(v.X.Name()) > 1
		}
		if v.Op == token.SUB {
			// Conservative for subtraction: avoid destructive updates on x to
			// prevent alias/overwrite corner cases in complex SSA blocks.
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
					g.emitAllocRegExcept("scratch", "\t", true, xVal)
					g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.W.%s(scratch, RegR11)", floatAluOp)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				} else {
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.W.%s(%s.Reg, RegR11)", floatAluOp, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				if yVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
				}
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%s.Imm.Float() %s %s.Imm.Float())}", dv, xVal.goVar, goOp, yVal.goVar)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				g.emitAllocRegExcept("scratch", "\t", true, yVal)
				g.emit("\t_, xBits := %s.Imm.RawWords()", xVal.goVar)
				g.emit("\tctx.W.EmitMovRegImm64(scratch, xBits)")
				g.emit("\tctx.W.%s(scratch, %s.Reg)", floatAluOp, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					g.emitAllocRegExcept("scratch", "\t", true, xVal)
					g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.W.%s(scratch, RegR11)", floatAluOp)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}", dv)
				} else {
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.W.%s(%s.Reg, RegR11)", floatAluOp, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.%s(%s, %s.Reg)", floatAluOp, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.%s(%s.Reg, %s.Reg)", floatAluOp, xVal.goVar, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			}
			g.emit("if %s.Loc == LocReg && %s.Loc == LocReg && %s.Reg == %s.Reg {", dv, xVal.goVar, dv, xVal.goVar)
			g.emit("\tctx.TransferReg(%s.Reg)", xVal.goVar)
			g.emit("\t%s.Loc = LocNone", xVal.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
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
							g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg2, 0)", xVal.goVar)
							g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
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
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", bits)
					g.emit("\tctx.W.EmitCmpFloat64Setcc(%s, %s.Reg, RegR11, %s)", rv, xVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
					g.emit("}")
				} else {
					yVal := g.resolveValue(v.Y)
					if xVal.isDesc {
						g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
					}
					if yVal.isDesc {
						g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
					}
					g.emit("var %s JITValueDesc", dv)
					g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Float() %s %s.Imm.Float())}", dv, xVal.goVar, goOp, yVal.goVar)
					g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
					rv := g.allocReg()
					g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
					g.emit("\t_, yBits := %s.Imm.RawWords()", yVal.goVar)
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, yBits)")
					g.emit("\tctx.W.EmitCmpFloat64Setcc(%s, %s.Reg, RegR11, %s)", rv, xVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
					g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
					rv2 := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg)", rv2, yVal.goVar)
					g.emit("\t_, xBits := %s.Imm.RawWords()", xVal.goVar)
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, xBits)")
					g.emit("\tctx.W.EmitCmpFloat64Setcc(%s, RegR11, %s.Reg, %s)", rv2, yVal.goVar, cc)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv2)
					g.emit("} else {")
					rv3 := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", rv3, xVal.goVar, yVal.goVar)
					g.emit("\tctx.W.EmitCmpFloat64Setcc(%s, %s.Reg, %s.Reg, %s)", rv3, xVal.goVar, yVal.goVar, cc)
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
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(%s.Imm.Int()) %s uint64(%d))}", dv, xVal.goVar, goOp, cmpVal)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Int() %s %d)}", dv, xVal.goVar, goOp, cmpVal)
				}
				g.emit("} else {")
				// Fresh register for result — CMP is non-destructive, SetCC writes only the target.
				// Protect xVal.Reg when multi-use: AllocReg must not return xVal.Reg (SetCC would clobber it).
				rv := g.allocReg()
				g.emitAllocRegExcept(rv, "\t", xMultiUse, xVal)
				if fitsInt32(cmpVal) {
					g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg, %d)", xVal.goVar, cmpVal)
				} else {
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
					g.emit("\tctx.W.EmitCmpInt64(%s.Reg, RegR11)", xVal.goVar)
				}
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				// Conservative spill safety: EnsureDesc(y) may spill x when register
				// pressure is high. Re-ensure both operands before emitting compare code.
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				if yVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
				}
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				if yVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
				}
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
				g.emit("\t\tctx.W.EmitCmpRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
				g.emit("\t\tctx.W.EmitCmpInt64(%s.Reg, RegR11)", xVal.goVar)
				g.emit("\t}")
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				// x is imm, y is reg → materialize x, CMP
				rv2 := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv2)
				g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.EmitCmpInt64(RegR11, %s.Reg)", yVal.goVar)
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv2, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv2)
				g.emit("} else {")
				// Both regs: protect xVal.Reg when multi-use (SetCC would clobber if rv3==xVal.Reg).
				rv3 := g.allocReg()
				g.emitAllocRegExcept(rv3, "\t", xMultiUse, xVal)
				g.emit("\tctx.W.EmitCmpInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv3, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv3)
				g.emit("}")
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else if aluOp := aluEmitFunc(v.Op); aluOp != "" {
			// Arithmetic BinOp: ADD, SUB, MUL
			dv := g.allocDesc()
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
						g.emitAllocRegExcept("scratch", "\t", true, xVal)
						g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						if fitsInt32(cmpVal) {
							g.emit("\tctx.W.EmitSubRegImm32(scratch, int32(%d))", cmpVal)
						} else {
							g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
							g.emit("\tctx.W.EmitSubInt64(scratch, RegR11)")
						}
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					} else {
						// ADD/MUL: commutative, order doesn't matter
						g.emitAllocRegExcept("scratch", "\t", true, xVal)
						g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						if v.Op == token.MUL {
							g.emitMulConstOnReg("scratch", cmpVal, "\t")
						} else if fitsInt32(cmpVal) {
							g.emit("\tctx.W.EmitAddRegImm32(scratch, int32(%d))", cmpVal)
						} else {
							g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
							g.emit("\tctx.W.%s(scratch, RegR11)", aluOp)
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
							g.emit("\tctx.W.EmitAddRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						case token.SUB:
							g.emit("\tctx.W.EmitSubRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						default:
							g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
							g.emit("\tctx.W.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
						}
					} else {
						g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
						g.emit("\tctx.W.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
				// Conservative spill safety for arithmetic BinOps as well.
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				if yVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
				}
				if xVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", xVal.goVar)
				}
				if yVal.isDesc {
					g.emit("ctx.EnsureDesc(&%s)", yVal.goVar)
				}
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s {", bothImmCond(xVal.goVar, yVal.goVar))
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOpStr(v.Op), yVal.goVar)
				// Identity optimizations: ADD/SUB 0 is no-op
				if v.Op == token.ADD || v.Op == token.SUB {
					// y is LocImm 0 → x + 0 = x, x - 0 = x
					g.emit("} else if %s.Loc == LocImm && %s.Imm.Int() == 0 {", yVal.goVar, yVal.goVar)
					if xMultiUse {
						copyReg := g.allocReg()
						g.emitAllocRegExcept(copyReg, "\t", true, xVal)
						g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
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
				g.emitAllocRegExcept("scratch", "\t", true, yVal)
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.%s(scratch, %s.Reg)", aluOp, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				// y is const, x is reg → use R11 for constant (result in x.Reg or scratch)
				if xMultiUse {
					if v.Op == token.SUB {
						// SUB is non-commutative: copy x, then subtract y
						g.emitAllocRegExcept("scratch", "\t", true, xVal)
						g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
						g.emit("\t\tctx.W.EmitSubRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t} else {")
						g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.W.EmitSubInt64(scratch, RegR11)")
						g.emit("\t}")
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					} else {
						// ADD/MUL: commutative, order doesn't matter
						g.emitAllocRegExcept("scratch", "\t", true, xVal)
						g.emit("\tctx.W.EmitMovRegReg(scratch, %s.Reg)", xVal.goVar)
						g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
						if v.Op == token.ADD {
							g.emit("\t\tctx.W.EmitAddRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						} else if v.Op == token.MUL {
							g.emit("\t\tctx.W.EmitImulRegImm32(scratch, int32(%s.Imm.Int()))", yVal.goVar)
						} else {
							g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
							g.emit("\t\tctx.W.%s(scratch, RegR11)", aluOp)
						}
						g.emit("\t} else {")
						g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.W.%s(scratch, RegR11)", aluOp)
						g.emit("\t}")
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					}
				} else {
					// x consumed, y constant: immediate-form ALU when possible.
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					if v.Op == token.ADD {
						g.emit("\t\tctx.W.EmitAddRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else if v.Op == token.SUB {
						g.emit("\t\tctx.W.EmitSubRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else if v.Op == token.MUL {
						g.emit("\t\tctx.W.EmitImulRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					} else {
						g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
						g.emit("\t\tctx.W.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					}
					g.emit("\t} else {")
					g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\tctx.W.%s(%s.Reg, RegR11)", aluOp, xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.%s(%s, %s.Reg)", aluOp, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.%s(%s.Reg, %s.Reg)", aluOp, xVal.goVar, yVal.goVar)
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
			g.vals[name] = genVal{goVar: dv, isDesc: true}
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						shift := 0
						for d := divisor; d > 1; d >>= 1 {
							shift++
						}
						g.emit("\tctx.W.EmitShrRegImm8(%s, %d)", copyReg, shift)
					} else {
						g.emit("\tctx.W.EmitIdivRegImm(%s, %d)", copyReg, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						shift := 0
						for d := divisor; d > 1; d >>= 1 {
							shift++
						}
						g.emit("\tctx.W.EmitShrRegImm8(%s.Reg, %d)", xVal.goVar, shift)
					} else {
						g.emit("\tctx.W.EmitIdivRegImm(%s.Reg, %d)", xVal.goVar, divisor)
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						g.emit("\tctx.W.EmitAndRegImm32(%s, %d)", copyReg, divisor-1)
					} else {
						g.emit("\tctx.W.EmitIremRegImm(%s, %d)", copyReg, divisor)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if divisor > 0 && (divisor&(divisor-1)) == 0 {
						g.emit("\tctx.W.EmitAndRegImm32(%s.Reg, %d)", xVal.goVar, divisor-1)
					} else {
						g.emit("\tctx.W.EmitIremRegImm(%s.Reg, %d)", xVal.goVar, divisor)
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.%s(%s, %d)", immFn, copyReg, shiftAmt)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.%s(%s.Reg, %d)", immFn, xVal.goVar, shiftAmt)
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.%s(%s, uint8(%s.Imm.Int()))", immFn, copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.%s(%s.Reg, uint8(%s.Imm.Int()))", immFn, xVal.goVar, yVal.goVar)
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
					g.emit("\t\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t\tshiftSrc = %s", copyReg)
				} else {
					g.emit("\t\tif shiftSrc == RegRCX {")
					g.emit("\t\t\tnewReg := ctx.AllocReg()")
					g.emit("\t\t\tctx.W.EmitMovRegReg(newReg, RegRCX)")
					g.emit("\t\t\tshiftSrc = newReg")
					g.emit("\t\t}")
				}
				g.emit("\t\trcxUsed := ctx.FreeRegs & (1 << uint(RegRCX)) == 0 && %s.Reg != RegRCX", yVal.goVar)
				g.emit("\t\tif rcxUsed {")
				g.emit("\t\t\tctx.W.EmitMovRegReg(RegR11, RegRCX)") // save RCX in scratch R11
				g.emit("\t\t}")
				g.emit("\t\tif %s.Reg != RegRCX {", yVal.goVar)
				g.emit("\t\t\tctx.W.EmitMovRegReg(RegRCX, %s.Reg)", yVal.goVar)
				g.emit("\t\t}")
				g.emit("\t\tctx.W.%s(shiftSrc)", emitFn)
				g.emit("\t\tif rcxUsed {")
				g.emit("\t\t\tctx.W.EmitMovRegReg(RegRCX, RegR11)") // restore RCX from R11
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tif %d >= -2147483648 && %d <= 2147483647 {", cmpVal, cmpVal)
					g.emit("\t\tctx.W.EmitAndRegImm32(%s, int32(%d))", copyReg, cmpVal)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
					g.emit("\t\tctx.W.EmitAndInt64(%s, RegR11)", copyReg)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %d >= -2147483648 && %d <= 2147483647 {", cmpVal, cmpVal)
					g.emit("\t\tctx.W.EmitAndRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
					g.emit("\t\tctx.W.EmitAndInt64(%s.Reg, RegR11)", xVal.goVar)
					g.emit("\t}")
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
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.EmitAndInt64(scratch, %s.Reg)", yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.W.EmitAndRegImm32(%s, int32(%s.Imm.Int()))", copyReg, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitAndInt64(%s, RegR11)", copyReg)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.W.EmitAndRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitAndInt64(%s.Reg, RegR11)", xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.EmitAndInt64(%s, %s.Reg)", copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.EmitAndInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					if fitsInt32(cmpVal) {
						g.emit("\tctx.W.EmitOrRegImm32(%s, int32(%d))", copyReg, cmpVal)
					} else {
						g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
						g.emit("\tctx.W.EmitOrInt64(%s, RegR11)", copyReg)
					}
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					if fitsInt32(cmpVal) {
						g.emit("\tctx.W.EmitOrRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
					} else {
						g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
						g.emit("\tctx.W.EmitOrInt64(%s.Reg, RegR11)", xVal.goVar)
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
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				g.emitAllocRegExcept("scratch", "\t", true, yVal)
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.EmitOrInt64(scratch, %s.Reg)", yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.W.EmitOrRegImm32(%s, int32(%s.Imm.Int()))", copyReg, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitOrInt64(%s, RegR11)", copyReg)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.W.EmitOrRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitOrInt64(%s.Reg, RegR11)", xVal.goVar)
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emit("\t%s := ctx.AllocRegExcept(%s.Reg, %s.Reg)", copyReg, xVal.goVar, yVal.goVar)
					g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", copyReg, xVal.goVar)
					g.emit("\tctx.W.EmitOrInt64(%s, %s.Reg)", copyReg, yVal.goVar)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tctx.W.EmitOrInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
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
		if phiOff, ok := g.phiRegs[name]; ok {
			if g.phiPair[name] {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(%s)}", dv, phiOff)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			} else {
				dv := g.allocDesc()
				g.emit("%s := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(%s)}", dv, phiOff)
				g.vals[name] = genVal{goVar: dv, isDesc: true}
			}
		} else {
			panic(fmt.Sprintf("phi %s has no allocated stack slot", name))
		}

	case *ssa.If:
		thenBB := v.Block().Succs[0].Index
		elseBB := v.Block().Succs[1].Index
		// SSA-constant condition: emit only taken edge and enqueue exactly one BB.
		if c, ok := v.Cond.(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.Bool {
			takenBB := elseBB
			if constant.BoolVal(c.Value) {
				takenBB = thenBB
			}
			_ = g.ensureBBLabel(takenBB)
			if takenBB == thenBB {
				g.emitEdgePhiMoves(takenBB, 0)
			} else {
				g.emitEdgePhiMoves(takenBB, 1)
			}
			lbl := g.ensureBBLabel(takenBB)
			g.emit("ctx.W.EmitJmp(%s)", lbl)
			if !g.bbDone[g.scopedBBID(takenBB)] {
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
		g.emit("%s := ctx.W.ReserveLabel()", thenEdgeLbl)
		g.emit("%s := ctx.W.ReserveLabel()", elseEdgeLbl)

		g.emit("if %s.Loc == LocImm {", condVar)
		g.emit("\tif %s.Imm.Bool() {", condVar)
		// Constant true: still route through helper edge BB.
		g.emit("\t\tctx.W.EmitJmp(%s)", thenEdgeLbl)
		g.emit("\t} else {")
		// Constant false: still route through helper edge BB.
		g.emit("\t\tctx.W.EmitJmp(%s)", elseEdgeLbl)
		g.emit("\t}")
		g.emit("} else {")
		// Runtime: CMP + JNE to then-edge helper, otherwise else-edge helper.
		g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg, 0)", condVar)
		g.emit("\tctx.W.EmitJcc(CcNE, %s)", thenEdgeLbl)
		g.emit("\tctx.W.EmitJmp(%s)", elseEdgeLbl)
		g.emit("}")
		// Helper edges are always emitted, independent of cond materialization.
		g.emit("ctx.W.MarkLabel(%s)", thenEdgeLbl)
		g.emitEdgePhiMoves(thenBB, 0)
		g.emit("ctx.W.EmitJmp(%s)", thenLbl)
		g.emit("ctx.W.MarkLabel(%s)", elseEdgeLbl)
		g.emitEdgePhiMoves(elseBB, 1)
		g.emit("ctx.W.EmitJmp(%s)", elseLbl)
		g.enqueueBB(elseBB)
		g.enqueueBB(thenBB)

	case *ssa.Jump:
		targetBB := v.Block().Succs[0].Index
		_ = g.ensureBBLabel(targetBB)
		g.emitEdgePhiMoves(targetBB, 0)
		lbl := g.ensureBBLabel(targetBB)
		g.emit("ctx.W.EmitJmp(%s)", lbl)
		if !g.bbDone[g.scopedBBID(targetBB)] {
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
			g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", tmpReg, src.goVar)
			// Normalize source width/sign first.
			if srcBits > 0 && srcBits < 64 {
				shift := 64 - srcBits
				g.emit("\tctx.W.EmitShlRegImm8(%s, %d)", tmpReg, shift)
				if srcSigned {
					g.emit("\tctx.W.EmitSarRegImm8(%s, %d)", tmpReg, shift)
				} else {
					g.emit("\tctx.W.EmitShrRegImm8(%s, %d)", tmpReg, shift)
				}
			}
			// Then normalize destination width/sign for actual conversion target.
			if dstBits > 0 && dstBits < 64 {
				shift := 64 - dstBits
				g.emit("\tctx.W.EmitShlRegImm8(%s, %d)", tmpReg, shift)
				if dstSigned {
					g.emit("\tctx.W.EmitSarRegImm8(%s, %d)", tmpReg, shift)
				} else {
					g.emit("\tctx.W.EmitShrRegImm8(%s, %d)", tmpReg, shift)
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
			g.emit("\tctx.W.EmitCvtInt64ToFloat64(RegX0, %s.Reg)", src.goVar)
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
			g.emit("\tctx.W.EmitCvtFloatBitsToInt64(%s, %s.Reg)", tmpReg, src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, tmpReg)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("unsupported Convert %s → %s", v.X.Type(), v.Type()))
		}

	case *ssa.Alloc:
		// Track allocation — no machine code emitted.
		// The actual value will be set by Store and consumed by MakeClosure.
		g.vals[name] = genVal{marker: "_alloc"}

	case *ssa.Store:
		dst := g.vals[v.Addr.Name()]
		if dst.marker == "_alloc" {
			// Storing to an allocation: just remember the stored value
			src := g.vals[v.Val.Name()]
			g.vals[v.Addr.Name()] = genVal{goVar: src.goVar, isDesc: src.isDesc, marker: "_alloc_stored"}
		} else {
			panic(fmt.Sprintf("unsupported Store: %s", v))
		}

	case *ssa.MakeClosure:
		// Construct closure from captured variables.
		// The binding should reference an _alloc_stored value (the func to wrap).
		if len(v.Bindings) != 1 {
			panic(fmt.Sprintf("MakeClosure with %d bindings", len(v.Bindings)))
		}
		binding := g.vals[v.Bindings[0].Name()]
		if binding.marker != "_alloc_stored" {
			panic("MakeClosure binding not an alloc-stored value")
		}
		// The stored value is a func(...Scmer) Scmer (1 word in a register).
		// Call JITBuildMergeClosure to wrap it.
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(JITBuildMergeClosure), []JITValueDesc{%s}, 1)", dv, binding.goVar)
		g.vals[name] = genVal{goVar: dv, isDesc: true}

	case *ssa.MakeInterface:
		// MakeInterface is typically before panic — we can't JIT this but it's dead code
		// in guarded paths. Store a dummy.
		g.vals[name] = genVal{marker: "_interface"}

	case *ssa.Panic:
		// Panic: emit a trap/unreachable instruction (INT3 on amd64)
		g.emit("ctx.W.EmitByte(0xCC)") // INT3

	case *ssa.Slice:
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
			g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", ptrReg, x.goVar)
			g.emit("} else if %s.Loc == LocRegPair {", x.goVar)
			g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", ptrReg, x.goVar)
			g.emit("} else {")
			g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", ptrReg, x.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", low.goVar)
			g.emit("\tif %s.Imm.Int() != 0 {", low.goVar)
			g.emit("\t\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", low.goVar, low.goVar)
			g.emit("\t\t\tctx.W.EmitAddRegImm32(%s, int32(%s.Imm.Int()))", ptrReg, low.goVar)
			g.emit("\t\t} else {")
			g.emit("\t\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\t\tctx.W.EmitAddInt64(%s, RegR11)", ptrReg)
			g.emit("\t\t}")
			g.emit("\t}")
			g.emit("} else {")
			g.emit("\tctx.W.EmitAddInt64(%s, %s.Reg)", ptrReg, low.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", high.goVar)
			g.emit("\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", lenReg, high.goVar)
			g.emit("} else {")
			g.emit("\tctx.W.EmitMovRegReg(%s, %s.Reg)", lenReg, high.goVar)
			g.emit("}")
			g.emit("if %s.Loc == LocImm {", low.goVar)
			g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", low.goVar, low.goVar)
			g.emit("\t\tctx.W.EmitSubRegImm32(%s, int32(%s.Imm.Int()))", lenReg, low.goVar)
			g.emit("\t} else {")
			g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\tctx.W.EmitSubInt64(%s, RegR11)", lenReg)
			g.emit("\t}")
			g.emit("} else {")
			g.emit("\tctx.W.EmitSubInt64(%s, %s.Reg)", lenReg, low.goVar)
			g.emit("}")
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv, ptrReg, lenReg)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
			break
		}
		// Sub-slice: str[low:high] or slice[low:high]
		// Result is a LocRegPair {dataPtr, length} representing a Go string/slice header.
		x := g.vals[v.X.Name()]
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
			g.emit("if %s.Loc == LocRegPair {", x.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", highDesc, x.goVar)
			g.emit("} else {")
			g.emit("\tpanic(\"Slice with omitted high requires descriptor with length in Reg2\")")
			g.emit("}")
			high = genVal{goVar: highDesc, isDesc: true}
		} else {
			high = g.resolveValue(v.High)
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
		g.emit("\t\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", lenReg, high.goVar)
		g.emit("\t} else {")
		g.emit("\t\tctx.W.EmitMovRegReg(%s, %s.Reg)", lenReg, high.goVar)
		g.emit("\t}")
		g.emit("\tif %s.Loc == LocImm {", low.goVar)
		g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
		g.emit("\t\tctx.W.EmitSubInt64(%s, RegR11)", lenReg)
		g.emit("\t} else {")
		g.emit("\t\tctx.W.EmitSubInt64(%s, %s.Reg)", lenReg, low.goVar)
		g.emit("\t}")
		g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", lenDesc, lenReg)
		g.emit("}")
		// Compute new data pointer: x.ptr + low
		ptrDesc := g.allocDesc()
		g.emit("var %s JITValueDesc", ptrDesc)
		if x.isDesc {
			// x is a string/slice descriptor: Reg=dataPtr (or LocImm for const fold)
			g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", x.goVar, low.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() + %s.Imm.Int())}", ptrDesc, x.goVar, low.goVar)
			g.emit("} else {")
			ptrReg := g.allocReg()
			g.emit("\t%s := ctx.AllocReg()", ptrReg)
			g.emit("\tif %s.Loc == LocImm {", x.goVar)
			g.emit("\t\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", ptrReg, x.goVar)
			g.emit("\t} else {")
			g.emit("\t\tctx.W.EmitMovRegReg(%s, %s.Reg)", ptrReg, x.goVar)
			g.emit("\t}")
			g.emit("\tif %s.Loc == LocImm {", low.goVar)
			g.emit("\t\tctx.W.EmitMovRegImm64(RegR11, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\tctx.W.EmitAddInt64(%s, RegR11)", ptrReg)
			g.emit("\t} else {")
			g.emit("\t\tctx.W.EmitAddInt64(%s, %s.Reg)", ptrReg, low.goVar)
			g.emit("\t}")
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", ptrDesc, ptrReg)
			g.emit("}")
		} else {
			panic(fmt.Sprintf("Slice on non-desc: %s", v))
		}
		// Combine into LocRegPair {ptr, len} — same layout as Go string header
		dv2 := g.allocDesc()
		g.emit("var %s JITValueDesc", dv2)
		g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", ptrDesc, lenDesc)
		g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(%s.Imm.Int())}", dv2, ptrDesc)
		g.emit("\t_ = %s", lenDesc) // length carried in separate LocImm
		g.emit("} else {")
		// Materialize both into registers
		finalPtr := g.allocReg()
		finalLen := g.allocReg()
		g.emit("\t%s := ctx.AllocReg()", finalPtr)
		g.emit("\t%s := ctx.AllocReg()", finalLen)
		g.emit("\tif %s.Loc == LocImm {", ptrDesc)
		g.emit("\t\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", finalPtr, ptrDesc)
		g.emit("\t} else {")
		g.emit("\t\tctx.W.EmitMovRegReg(%s, %s.Reg)", finalPtr, ptrDesc)
		g.emit("\t\tctx.FreeReg(%s.Reg)", ptrDesc)
		g.emit("\t}")
		g.emit("\tif %s.Loc == LocImm {", lenDesc)
		g.emit("\t\tctx.W.EmitMovRegImm64(%s, uint64(%s.Imm.Int()))", finalLen, lenDesc)
		g.emit("\t} else {")
		g.emit("\t\tctx.W.EmitMovRegReg(%s, %s.Reg)", finalLen, lenDesc)
		g.emit("\t\tctx.FreeReg(%s.Reg)", lenDesc)
		g.emit("\t}")
		g.emit("\t%s = JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", dv2, finalPtr, finalLen)
		g.emit("}")
		_ = dv
		g.vals[name] = genVal{goVar: dv2, isDesc: true, marker: "_gostring"}

	default:
		panic(instrDesc(instr))
	}
}

// emitReturnSingleBlock handles Return for single-block functions (with constant propagation).
func (g *codeGen) emitReturnSingleBlock(v *ssa.Return) {
	if len(g.bbLabels) > 0 {
		g.emit("ctx.W.ResolveFixups()")
	}
	if len(v.Results) == 0 {
		g.emit("if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: NewNil()} }")
		g.emit("ctx.W.EmitMakeNil(result)")
		g.emit("return result")
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newbool":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newint":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newfloat":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newnil":
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.emit("ctx.W.EmitMakeNil(result)")
		g.emit("return result")
	case "_newstring":
		// NewString(s string) Scmer — arg is Go string {ptr, len} (2 words), result is Scmer (2 words)
		dv := g.allocDesc()
		g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{%s}, 2)", dv, res.goVar)
		g.emit("if result.Loc == LocAny { return %s }", dv)
		g.emit("ctx.EmitMovPairToResult(&%s, &result)", dv)
		g.emit("return result")
	default:
		if res.isDesc {
			g.emit("if result.Loc == LocAny {")
			g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
			g.emit("}")
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair {", res.goVar)
			g.emit("\tctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("} else {")
			g.emit("\tswitch %s.Type {", res.goVar)
			g.emit("\tcase tagBool:")
			g.emit("\t\tctx.W.EmitMakeBool(result, %s)", res.goVar)
			g.emit("\tcase tagInt:")
			g.emit("\t\tctx.W.EmitMakeInt(result, %s)", res.goVar)
			g.emit("\tcase tagFloat:")
			g.emit("\t\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("\tcase tagNil:")
			g.emit("\t\tctx.W.EmitMakeNil(result)")
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
			g.emit("ctx.W.EmitMakeNil(%s)", retDesc)
			g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
			return
		}
		res := g.vals[v.Results[0].Name()]
		switch res.marker {
		case "_newbool":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.W.EmitMakeBool(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newint":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.W.EmitMakeInt(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newfloat":
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("ctx.W.EmitMakeFloat(%s, %s)", retDesc, res.goVar)
			g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		case "_newnil":
			g.emit("ctx.W.EmitMakeNil(%s)", retDesc)
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
				g.emit("\t\tctx.W.EmitMakeBool(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagInt:")
				g.emit("\t\tctx.W.EmitMakeInt(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagFloat:")
				g.emit("\t\tctx.W.EmitMakeFloat(%s, %s)", retDesc, res.goVar)
				g.emit("\tcase tagNil:")
				g.emit("\t\tctx.W.EmitMakeNil(%s)", retDesc)
				g.emit("\tdefault:")
				g.emit("\t\tctx.EmitMovPairToResult(&%s, &%s)", res.goVar, retDesc)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
			}
		}
		g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
		return
	}

	if len(v.Results) == 0 {
		g.emit("ctx.W.EmitMakeNil(result)")
		g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newbool":
		g.emit("ctx.EnsureDesc(&%s)", res.goVar)
		g.emit("ctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		g.emit("result.Type = tagBool")
	case "_newint":
		g.emit("ctx.EnsureDesc(&%s)", res.goVar)
		g.emit("ctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		g.emit("result.Type = tagInt")
	case "_newfloat":
		g.emit("ctx.EnsureDesc(&%s)", res.goVar)
		g.emit("ctx.W.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		g.emit("result.Type = tagFloat")
	case "_newnil":
		g.emit("ctx.W.EmitMakeNil(result)")
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
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair {", res.goVar)
			g.emit("\tctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("\tresult.Type = %s.Type", res.goVar)
			g.emit("} else {")
			// Known scalar type => no additional tag register allocation.
			// The concrete Scmer pair is materialized directly into result registers.
			g.emit("\tswitch %s.Type {", res.goVar)
			g.emit("\tcase tagBool:")
			g.emit("\t\tctx.W.EmitMakeBool(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagBool")
			g.emit("\tcase tagInt:")
			g.emit("\t\tctx.W.EmitMakeInt(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagInt")
			g.emit("\tcase tagFloat:")
			g.emit("\t\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("\t\tresult.Type = tagFloat")
			g.emit("\tcase tagNil:")
			g.emit("\t\tctx.W.EmitMakeNil(result)")
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
	g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
}

// emitInlineReturn handles Return inside an inlined function (multi-block).
// Moves the return value to the pre-allocated inline result register(s) and JMPs to end.
func (g *codeGen) emitInlineReturn(v *ssa.Return) {
	if len(v.Results) == 0 {
		// void return — shouldn't happen for inlined value-returning functions
		g.emit("ctx.W.EmitJmp(%s)", g.inlineEndLabel)
		return
	}

	if g.inlineReturnsScm {
		if g.inlineReturnReg == "" {
			g.inlineReturnReg = g.allocReg()
			g.emit("%s := ctx.AllocReg()", g.inlineReturnReg)
			g.inlineReturnReg2 = g.allocReg()
			g.emit("%s := ctx.AllocRegExcept(%s)", g.inlineReturnReg2, g.inlineReturnReg)
		}
		// Scmer pair return: construct Scmer into the two pre-allocated registers
		res := g.vals[v.Results[0].Name()]
		irDesc := g.allocDesc()
		g.emit("%s := JITValueDesc{Loc: LocRegPair, Reg: %s, Reg2: %s}", irDesc, g.inlineReturnReg, g.inlineReturnReg2)
		switch res.marker {
		case "_newbool":
			g.emit("ctx.W.EmitMakeBool(%s, %s)", irDesc, res.goVar)
		case "_newint":
			g.emit("ctx.W.EmitMakeInt(%s, %s)", irDesc, res.goVar)
		case "_newfloat":
			g.emit("ctx.W.EmitMakeFloat(%s, %s)", irDesc, res.goVar)
		case "_newnil":
			g.emit("ctx.W.EmitMakeNil(%s)", irDesc)
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
		if g.inlineReturnReg == "" {
			g.inlineReturnReg = g.allocReg()
			g.emit("%s := ctx.AllocReg()", g.inlineReturnReg)
		}
		// Scalar return: move single value to result register
		res := g.resolveValue(v.Results[0])
		if res.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", res.goVar)
			g.emit("if %s.Loc == LocRegPair {", res.goVar)
			g.emit("\tpanic(\"jit: scalar inline return has LocRegPair\")")
			g.emit("} else {")
			g.emit("\tctx.EmitMovToReg(%s, %s)", g.inlineReturnReg, res.goVar)
			g.emit("}")
		} else {
			g.emit("ctx.W.EmitMovRegReg(%s, %s)", g.inlineReturnReg, res.goVar)
		}
	}
	g.emit("ctx.W.EmitJmp(%s)", g.inlineEndLabel)
}

func (g *codeGen) lookup(v ssa.Value) genVal {
	if gv, ok := g.vals[v.Name()]; ok {
		if gv.isDesc {
			g.emit("ctx.EnsureDesc(&%s)", gv.goVar)
		}
		return gv
	}
	panic(fmt.Sprintf("unresolved SSA value: %s", v))
}

var (
	locRegAssignRe     = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?JITValueDesc\{Loc:\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?LocReg,\s*(?:Type:\s*[^,}]+,\s*)?Reg:\s*([^,}]+)`)
	locRegPairAssignRe = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?JITValueDesc\{Loc:\s*(?:[A-Za-z_][A-Za-z0-9_]*\.)?LocRegPair,\s*(?:Type:\s*[^,}]+,\s*)?Reg:\s*([^,}]+),\s*Reg2:\s*([^,}]+)`)
	regExprRe          = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?$`)
)

func bindableRegExpr(expr string) bool {
	return regExprRe.MatchString(strings.TrimSpace(expr))
}

func injectBindRegCalls(code string) string {
	lines := strings.Split(code, "\n")
	out := make([]string, 0, len(lines)+len(lines)/3)
	for _, line := range lines {
		out = append(out, line)
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
	if c, ok := v.(*ssa.Const); ok {
		dv := g.allocDesc()
		if c.Value == nil {
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
		return "CcE"
	case token.NEQ:
		return "CcNE"
	case token.LSS:
		return "CcL"
	case token.GTR:
		return "CcG"
	case token.LEQ:
		return "CcLE"
	case token.GEQ:
		return "CcGE"
	default:
		return ""
	}
}

func opToCCUnsigned(op token.Token) string {
	switch op {
	case token.EQL:
		return "CcE"
	case token.NEQ:
		return "CcNE"
	case token.LSS:
		return "CcB"
	case token.GTR:
		return "CcA"
	case token.LEQ:
		return "CcBE"
	case token.GEQ:
		return "CcAE"
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
	default:
		return ""
	}
}

// aluEmitFunc maps an arithmetic token to the JITWriter emit method name for int64.
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
	case *types.Slice:
		return true
	case *types.Struct:
		// Scmer-like two-word structs.
		return elemSizeOf(t) == 16
	default:
		return false
	}
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
