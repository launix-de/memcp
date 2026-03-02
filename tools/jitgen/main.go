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
	"os"
	"path/filepath"
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
		for _, e := range pkg.Errors {
			fmt.Fprintf(os.Stderr, "  %v\n", e)
		}
		os.Exit(1)
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
type genVal struct {
	goVar            string
	isDesc           bool   // true = JITValueDesc (Scmer pair), false = Reg (scalar)
	argIdx           int    // >= 0: deferred arg reference from IndexAddr (constant index), not yet loaded
	argIdxVar        string // non-empty: deferred arg reference with variable index (goVar of index desc)
	marker           string // "_newbool"/"_newint"/"_newfloat" for deferred constructors
	deferredIndexSSA string // SSA name of index operand (for deferred IndexAddr on slices)
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
	bbLabels       map[int]string    // BB index → label var name
	bbDone         map[int]bool      // BB index → already generated
	bbQueue        []int             // queue of BB indices to generate
	phiRegs        map[string]string // SSA phi name → stack offset string (e.g. "0", "8", "16")
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
	inlineEndLabel   string // label after inlined blocks

	// Field deduplication: cache FieldAddr+UnOp deref results by field name
	fieldCache map[string]genVal

	// Reference counting for SSA values (remaining uses)
	refCounts map[string]int

	// SSA name aliases (e.g. Convert no-ops redirect to source)
	ssaAliases map[string]string

	// Top-level package path (the output package, not the inlined callee's package)
	topLevelPkgPath string

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
	if lbl, ok := g.bbLabels[bbIdx]; ok {
		return lbl
	}
	lbl := g.allocLabel()
	g.bbLabels[bbIdx] = lbl
	g.emit("%s := ctx.W.ReserveLabel()", lbl)
	return lbl
}

// enqueueBB adds a BB to the processing queue if not already done/queued.
func (g *codeGen) enqueueBB(bbIdx int) {
	if g.bbDone[bbIdx] {
		return
	}
	for _, q := range g.bbQueue {
		if q == bbIdx {
			return
		}
	}
	g.bbQueue = append(g.bbQueue, bbIdx)
}

// emitEdgePhiMoves emits machine-code-level MOVs for phi edges to targetBB from curBlock.
// Each phi in the target BB gets its register set via ctx.EmitMovToReg.
func (g *codeGen) emitEdgePhiMoves(targetBBIdx int) {
	targetBlock := g.fn.Blocks[targetBBIdx]
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		phiReg, ok := g.phiRegs[phi.Name()]
		if !ok {
			continue // no phi reg allocated (shouldn't happen)
		}
		for i, pred := range targetBlock.Preds {
			if pred.Index == g.curBlock {
				edge := phi.Edges[i]
				g.emitPhiMov(phiReg, edge, phi.Type())
				break
			}
		}
	}
}

// emitPhiMov emits a machine-code store from an SSA value to a phi stack slot.
// phiOff is the stack offset string (e.g. "0", "8", "16").
func (g *codeGen) emitPhiMov(phiOff string, v ssa.Value, phiType types.Type) {
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOff)
		} else if c.Value.Kind() == constant.Bool {
			bval := constant.BoolVal(c.Value)
			if bval {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, %s)", phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)", phiOff)
			}
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			if signed, bits, ok := intTypeInfo(phiType); ok {
				ival = normalizeIntConstForType(ival, signed, bits)
			}
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(%d)}, %s)", ival, phiOff)
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			g.emit("ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewFloat(%v)}, %s)", fval, phiOff)
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.isDesc {
			if signed, bits, ok := intTypeInfo(phiType); ok && bits > 0 && bits < 64 {
				tmp := g.allocDesc()
				g.emit("%s := %s", tmp, src.goVar)
				if signed {
					g.emitNormalizeSignedNarrow(tmp, bits)
				} else {
					g.emitNormalizeUnsignedNarrow(tmp, bits)
				}
				g.emit("ctx.EmitStoreToStack(%s, %s)", tmp, phiOff)
			} else {
				g.emit("ctx.EmitStoreToStack(%s, %s)", src.goVar, phiOff)
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
func (g *codeGen) emitEdgePhiMovesIndent(targetBBIdx int, indent string) {
	targetBlock := g.fn.Blocks[targetBBIdx]
	for _, instr := range targetBlock.Instrs {
		phi, ok := instr.(*ssa.Phi)
		if !ok {
			break
		}
		phiReg, ok := g.phiRegs[phi.Name()]
		if !ok {
			continue
		}
		for i, pred := range targetBlock.Preds {
			if pred.Index == g.curBlock {
				edge := phi.Edges[i]
				_ = indent
				g.emitPhiMov(phiReg, edge, phi.Type())
				break
			}
		}
	}
}

// emitPhiMovIndent emits a phi stack store with a given indent prefix.
func (g *codeGen) emitPhiMovIndent(phiOff string, v ssa.Value, indent string) {
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, %s)\n", indent, phiOff)
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
			g.phiRegs[phi.Name()] = fmt.Sprintf("%d", offset)
			offset += 8
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
	savedBBLabels := g.bbLabels
	savedCurBlock := g.curBlock
	savedPhiRegs := g.phiRegs
	savedPhiStackSize := g.phiStackSize
	savedVals := g.vals
	savedMultiBlock := g.multiBlock
	savedEndLabel := g.endLabel
	savedInlineReturnReg := g.inlineReturnReg
	savedInlineReturnReg2 := g.inlineReturnReg2
	savedInlineEndLabel := g.inlineEndLabel
	savedRefCounts := g.refCounts
	savedAliases := g.ssaAliases

	// Set up callee state
	g.fn = callee
	g.bbQueue = nil
	g.bbDone = map[int]bool{}
	g.bbLabels = map[int]string{}
	g.phiRegs = map[string]string{}
	g.vals = map[string]genVal{}
	g.refCounts = computeRefCounts(callee)
	g.ssaAliases = map[string]string{}
	// fieldCache is intentionally shared across inline boundary (same receiver)

	// Map callee params → resolved caller args
	for i, param := range callee.Params {
		g.vals[param.Name()] = resolvedArgs[i]
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
		if savedRefCounts[argName] > 1 {
			// Caller still needs this value after the call.
			// Bump callee refcount to prevent destructive ALU.
			g.refCounts[callee.Params[i].Name()]++
			// Protect the register from spilling.
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

	// For multi-block, allocate result register(s) and end label
	var resultReg, resultReg2 string
	if isMultiBlock {
		resultReg = g.allocReg()
		g.emit("%s := ctx.AllocReg()", resultReg)
		g.inlineReturnReg = resultReg

		if returnsScmer {
			resultReg2 = g.allocReg()
			g.emit("%s := ctx.AllocReg()", resultReg2)
			g.inlineReturnReg2 = resultReg2
		} else {
			g.inlineReturnReg2 = ""
		}

		inlineEnd := g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", inlineEnd)
		g.inlineEndLabel = inlineEnd
		g.endLabel = "" // don't use outer endLabel
	} else {
		g.inlineReturnReg = ""
		g.inlineReturnReg2 = ""
		g.inlineEndLabel = ""
		g.endLabel = ""
	}

	// Process callee blocks
	var singleBlockResult genVal
	g.bbQueue = []int{0}
	for len(g.bbQueue) > 0 {
		bbIdx := g.bbQueue[0]
		g.bbQueue = g.bbQueue[1:]
		if g.bbDone[bbIdx] {
			continue
		}
		g.bbDone[bbIdx] = true
		g.curBlock = bbIdx

		if lbl, ok := g.bbLabels[bbIdx]; ok {
			g.emit("ctx.W.MarkLabel(%s)", lbl)
		}

		block := callee.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			if ret, ok := instr.(*ssa.Return); ok && !isMultiBlock {
				// Single-block: capture return value directly, no code emitted
				if len(ret.Results) > 0 {
					singleBlockResult = g.resolveValue(ret.Results[0])
				}
			} else {
				g.emitInstr(instr)
				g.freeDeadOperands(instr)
			}
		}
	}

	if isMultiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.inlineEndLabel)
	}
	if len(g.bbLabels) > 0 || isMultiBlock {
		g.emit("ctx.W.ResolveFixups()")
	}
	// Note: no ADD RSP for inlined callee's phis — the unified phi frame
	// is managed by the outer function (allocated via fixup, freed at end).

	// Determine result
	var result genVal
	if isMultiBlock {
		dv := g.allocDesc()
		if returnsScmer {
			// Wrap the register pair in a JITValueDesc (Scmer = 2 words)
			g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, resultReg, resultReg2)
		} else {
			// Wrap the bare register in a JITValueDesc for type safety
			g.emit("%s := JITValueDesc{Loc: LocReg, Reg: %s}", dv, resultReg)
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
	g.bbLabels = savedBBLabels
	g.curBlock = savedCurBlock
	g.phiRegs = savedPhiRegs
	g.phiStackSize = savedPhiStackSize
	g.vals = savedVals
	g.multiBlock = savedMultiBlock
	g.endLabel = savedEndLabel
	g.inlineReturnReg = savedInlineReturnReg
	g.inlineReturnReg2 = savedInlineReturnReg2
	g.inlineEndLabel = savedInlineEndLabel
	g.refCounts = savedRefCounts
	g.ssaAliases = savedAliases

	return result
}

// generateClosure tries to generate a JIT emitter closure for the given SSA function.
// Returns (closureCode, "") on success, or ("", errorDescription) on failure.
func generateClosure(opName string, fn *ssa.Function) (code string, errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			code = ""
			errMsg = fmt.Sprintf("%v", r)
		}
	}()

	g := &codeGen{
		vals:            map[string]genVal{},
		fn:              fn,
		bbLabels:        map[int]string{},
		bbDone:          map[int]bool{},
		phiRegs:         map[string]string{},
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
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.endLabel = g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", g.endLabel)
	}

	// Process BBs via queue, starting from BB0
	g.bbQueue = []int{0}
	for len(g.bbQueue) > 0 {
		bbIdx := g.bbQueue[0]
		g.bbQueue = g.bbQueue[1:]
		if g.bbDone[bbIdx] {
			continue
		}
		g.bbDone[bbIdx] = true
		g.curBlock = bbIdx

		// Emit label if one was reserved for this BB
		if lbl, ok := g.bbLabels[bbIdx]; ok {
			g.emit("ctx.W.MarkLabel(%s)", lbl)
		}

		block := fn.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			g.emitInstr(instr)
			g.freeDeadOperands(instr)
		}
	}

	// Emit fixup resolution and epilogue
	if g.multiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.endLabel)
		g.emit("ctx.W.ResolveFixups()")
	} else if len(g.bbLabels) > 0 {
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
		g.w.String())
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
		bbLabels:        map[int]string{},
		bbDone:          map[int]bool{},
		phiRegs:         map[string]string{},
		fieldCache:      map[string]genVal{},
		refCounts:       computeRefCounts(fn),
		ssaAliases:      map[string]string{},
		storageMode:     true,
		typeName:        typeName,
		topLevelPkgPath: fn.Pkg.Pkg.Path(),
	}
	fmt.Fprintf(&g.w, "\t%s\n", generatedBanner)

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
		g.emit("\tctx.W.EmitShlRegImm8(idxInt.Reg, 32)")
		g.emit("\tctx.W.EmitShrRegImm8(idxInt.Reg, 32)")
		g.emit("}")
		g.vals[fn.Params[1].Name()] = genVal{goVar: "idxInt", isDesc: true}
	}

	g.multiBlock = len(fn.Blocks) > 1

	// Pre-allocate registers for all phi nodes
	g.allocPhiRegs()

	if g.multiBlock {
		g.emit("if result.Loc == LocAny {")
		g.emit("\tresult = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}")
		g.emit("}")
		g.endLabel = g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", g.endLabel)
	}

	// Process BBs via queue, starting from BB0
	g.bbQueue = []int{0}
	for len(g.bbQueue) > 0 {
		bbIdx := g.bbQueue[0]
		g.bbQueue = g.bbQueue[1:]
		if g.bbDone[bbIdx] {
			continue
		}
		g.bbDone[bbIdx] = true
		g.curBlock = bbIdx

		if lbl, ok := g.bbLabels[bbIdx]; ok {
			g.emit("ctx.W.MarkLabel(%s)", lbl)
		}

		block := fn.Blocks[bbIdx]
		for _, instr := range block.Instrs {
			g.emitInstr(instr)
			g.freeDeadOperands(instr)
		}
	}

	if g.multiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.endLabel)
		g.emit("ctx.W.ResolveFixups()")
	} else if len(g.bbLabels) > 0 {
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

	code = g.w.String()
	// In storage mode, generated code goes in the storage package and needs scm. prefix
	if g.storageMode {
		code = addScmPrefix(code)
	}
	return code, ""
}

// addScmPrefix adds "scm." prefix to scm package identifiers in generated code.
// This is needed when the generated code goes into the storage package.
func addScmPrefix(code string) string {
	// Words that need the scm. prefix — these are exported identifiers from the scm package
	scmIdents := map[string]bool{
		"JITValueDesc": true, "JITTypeUnknown": true, "JITContext": true,
		"LocNone": true, "LocReg": true, "LocRegPair": true,
		"LocStack": true, "LocMem": true, "LocImm": true, "LocAny": true,
		"NewInt": true, "NewFloat": true, "NewBool": true, "NewNil": true, "NewString": true,
		"NewFastDict": true, "NewFastDictValue": true,
		"Scmer": true, "GoFuncAddr": true, "JITBuildMergeClosure": true,
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
				idx, _ := constant.Int64Val(c.Value)
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
			if src.marker == "_slice" {
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
				panic(fmt.Sprintf("IndexAddr on non-parameter: %s", v))
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
		default:
			sizeStr = "8"
		}

		// Create marker with offsetExpr
		if isImmutable && sizeStr == "slice" {
			g.vals[name] = genVal{marker: "_fieldconst:slice:" + cacheKey, offsetExpr: offsetExpr}
		} else if isImmutable && goTypeName != "" {
			g.vals[name] = genVal{marker: "_fieldconst:" + goTypeName + ":" + cacheKey, offsetExpr: offsetExpr}
		} else {
			g.vals[name] = genVal{marker: "_fieldaddr:" + sizeStr + ":" + cacheKey, offsetExpr: offsetExpr}
		}

	case *ssa.UnOp:
		if v.Op == token.MUL {
			src := g.vals[v.X.Name()]
			if strings.HasPrefix(src.marker, "_fieldconst:") {
				// Deref of immutable FieldAddr → constant-fold (LocImm thisptr) or runtime load (LocReg thisptr).
				parts := strings.SplitN(src.marker, ":", 3) // "_fieldconst", goType, fieldName
				goType := parts[1]
				fieldName := parts[2]

				if goType == "slice" {
					// Immutable slice: constant-fold data ptr at JIT compile time.
					// Length is loaded lazily only if len() is actually called.
					// Uses field cache to avoid re-reading the same field.
					cacheKey := fieldName
					if cached, ok := g.fieldCache[cacheKey]; ok {
						g.vals[name] = cached
						break
					}
					dv := g.allocDesc()
					g.emit("var %s JITValueDesc", dv)
					g.emit("if thisptr.Loc == LocImm {")
					// LocImm: read data pointer at JIT compile time as immediate (NO register allocated).
					// The pointer will be materialized into a temp register only when actually used
					// for element access, keeping register pressure low.
					g.emit("\tfieldAddr := uintptr(thisptr.Imm.Int()) + %s", src.offsetExpr)
					g.emit("\tdataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))")
					g.emit("\tsliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))")
					g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}", dv)
					g.emit("} else {")
					// LocReg: register-relative load of data pointer only
					g.emit("\toff := int32(%s)", src.offsetExpr)
					ptrReg2 := g.allocReg()
					g.emit("\t%s := ctx.AllocReg()", ptrReg2)
					g.emit("\tctx.W.EmitMovRegMem(%s, thisptr.Reg, off)", ptrReg2)
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s}", dv, ptrReg2)
					g.emit("}")
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
				}
				g.emit("}")
				if sizeStr == "slice" {
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
				// Variable-index IndexAddr+Deref on args slice → emit runtime load from sliceBase
				dv := g.allocDesc()
				scratch := g.allocReg()
				g.emit("%s := ctx.AllocReg()", scratch)
				g.emit("ctx.W.EmitMovRegReg(%s, %s.Reg)", scratch, src.argIdxVar)
				g.emit("ctx.W.EmitShlRegImm8(%s, 4)", scratch) // *16 (Scmer pair)
				g.emit("ctx.W.EmitAddInt64(%s, ctx.SliceBase)", scratch)
				ptrReg := g.allocReg()
				auxReg := g.allocReg()
				g.emit("%s := ctx.AllocReg()", ptrReg)
				g.emit("%s := ctx.AllocReg()", auxReg)
				g.emit("ctx.W.EmitMovRegMem(%s, %s, 0)", ptrReg, scratch)
				g.emit("ctx.W.EmitMovRegMem(%s, %s, 8)", auxReg, scratch)
				g.emit("ctx.FreeReg(%s)", scratch)
				g.emit("%s := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: %s, Reg2: %s}", dv, ptrReg, auxReg)
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
					// len of a local slice variable (e.g. from Slice())
					src := g.vals[arg.Name()]
					if src.marker == "_slice" {
						dv := g.allocDesc()
						g.emit("var %s JITValueDesc", dv)
						g.emit("if %s.Loc == LocImm {", src.goVar)
						// LocImm slice: length was stored in StackOff at JIT compile time
						g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(%s.StackOff))}", dv, src.goVar)
						g.emit("} else {")
						// LocReg/LocRegPair: Reg2 = length register
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", dv, src.goVar)
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
			g.emit("%s := ctx.EmitTagEquals(&%s, tagNil, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsInt":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagInt, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFloat":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagFloat, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsBool":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagBool, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsString":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagString, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagSlice, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "IsFastDict":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagFastDict, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Int":
			// (Scmer).Int() — extract int64 from Scmer
			// LocImm: compile-time extraction; LocRegPair: aux IS the raw int64
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int())}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.FreeReg(%s.Reg)", arg.goVar) // free ptr, keep aux
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg2}", dv, arg.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "Float":
			// (Scmer).Float() — extract float64 from Scmer
			// LocImm: compile-time extraction; LocRegPair: aux holds float64 bits
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", arg.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(%s.Imm.Float())}", dv, arg.goVar)
			g.emit("} else {")
			g.emit("\tctx.FreeReg(%s.Reg)", arg.goVar) // free ptr, keep aux (float bits)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg2}", dv, arg.goVar)
			g.emit("}")
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		case "String":
			// (Scmer).String() string — extract Go string from Scmer
			// arg: Scmer (2 words), result: Go string (2 words: ptr+len)
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{%s}, 2)", dv, arg.goVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true, marker: "_gostring"}
		case "NewBool":
			src := g.lookup(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newbool"}
		case "NewInt":
			src := g.lookup(v.Call.Args[0])
			g.keepAliveForMarker(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newint"}
		case "NewFloat":
			src := g.lookup(v.Call.Args[0])
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
				g.emit("\tctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + %s))", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tscratch := ctx.AllocReg()")
				g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(scratch, baseReg, 0)")
				g.emit("\t\tctx.FreeReg(scratch)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, baseReg, 0)", val.goVar)
				g.emit("\t}")
				g.emit("\tctx.FreeReg(baseReg)")
				g.emit("} else {")
				g.emit("\toff := int32(%s)", dst.offsetExpr)
				g.emit("\tif %s.Loc == LocImm {", val.goVar)
				g.emit("\t\tscratch := ctx.AllocReg()")
				g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", val.goVar)
				g.emit("\t\tctx.W.EmitStoreRegMem(scratch, thisptr.Reg, off)")
				g.emit("\t\tctx.FreeReg(scratch)")
				g.emit("\t} else {")
				g.emit("\t\tctx.W.EmitStoreRegMem(%s.Reg, thisptr.Reg, off)", val.goVar)
				g.emit("\t}")
				g.emit("}")
			} else {
				panic(fmt.Sprintf("StoreInt64 dst is not a field address: marker=%q", dst.marker))
			}
		default:
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
		if g.isFieldCachedDesc(xVal.goVar) {
			xMultiUse = true
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
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal := c.Int64()
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
					g.emit("\tscratch := ctx.AllocReg()")
					g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%d))", cmpVal)
					g.emit("\tctx.W.EmitCmpInt64(%s.Reg, scratch)", xVal.goVar)
					g.emit("\tctx.FreeReg(scratch)")
				}
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("}")
			} else {
				yVal := g.resolveValue(v.Y)
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
				g.emit("\t\tscratch := ctx.AllocReg()")
				g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", yVal.goVar)
				g.emit("\t\tctx.W.EmitCmpInt64(%s.Reg, scratch)", xVal.goVar)
				g.emit("\t\tctx.FreeReg(scratch)")
				g.emit("\t}")
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				// x is imm, y is reg → materialize x, CMP
				rv2 := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv2)
				g.emit("\tscratch := ctx.AllocReg()")
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.EmitCmpInt64(scratch, %s.Reg)", yVal.goVar)
				g.emit("\tctx.FreeReg(scratch)")
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
				cmpVal := c.Int64()
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
						if fitsInt32(cmpVal) {
							if v.Op == token.ADD {
								g.emit("\tctx.W.EmitAddRegImm32(scratch, int32(%d))", cmpVal)
							} else if v.Op == token.MUL {
								g.emit("\tctx.W.EmitImulRegImm32(scratch, int32(%d))", cmpVal)
							} else {
								g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
								g.emit("\tctx.W.%s(scratch, RegR11)", aluOp)
							}
						} else {
							g.emit("\tctx.W.EmitMovRegImm64(RegR11, uint64(%d))", cmpVal)
							g.emit("\tctx.W.%s(scratch, RegR11)", aluOp)
						}
						g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
					}
				} else {
					// x is consumed; prefer immediate-form ALU to avoid materializing constants in a temp register.
					if fitsInt32(cmpVal) {
						switch v.Op {
						case token.ADD:
							g.emit("\tctx.W.EmitAddRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						case token.SUB:
							g.emit("\tctx.W.EmitSubRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
						case token.MUL:
							g.emit("\tctx.W.EmitImulRegImm32(%s.Reg, int32(%d))", xVal.goVar, cmpVal)
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
				g.emit("\tscratch := ctx.AllocReg()")
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
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
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
				panic(fmt.Sprintf("non-const division: %s", v))
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
				panic(fmt.Sprintf("non-const modulo: %s", v))
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
				g.emit("\tscratch := ctx.AllocReg()")
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
					g.emit("\t\tscratch := ctx.AllocReg()")
					g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitOrInt64(%s, scratch)", copyReg)
					g.emit("\t\tctx.FreeReg(scratch)")
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s}", dv, copyReg)
				} else {
					g.emit("\tif %s.Imm.Int() >= -2147483648 && %s.Imm.Int() <= 2147483647 {", yVal.goVar, yVal.goVar)
					g.emit("\t\tctx.W.EmitOrRegImm32(%s.Reg, int32(%s.Imm.Int()))", xVal.goVar, yVal.goVar)
					g.emit("\t} else {")
					g.emit("\t\tscratch := ctx.AllocReg()")
					g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", yVal.goVar)
					g.emit("\t\tctx.W.EmitOrInt64(%s.Reg, scratch)", xVal.goVar)
					g.emit("\t\tctx.FreeReg(scratch)")
					g.emit("\t}")
					g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				}
				g.emit("} else {")
				if xMultiUse {
					copyReg := g.allocReg()
					g.emitAllocRegExcept(copyReg, "\t", true, xVal)
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
		if g.inlineReturnReg != "" {
			// Inlined multi-block function: MOV result to designated register, JMP to end
			g.emitInlineReturn(v)
		} else if g.multiBlock {
			g.emitReturnMultiBlock(v)
		} else {
			g.emitReturnSingleBlock(v)
		}

	case *ssa.Phi:
		// Phi values live on the stack. Load into a temp register on each read.
		// Use AllocRegExcept(prev...) so loading phi N does not evict any
		// already-loaded phi register (mutual exclusion during loading).
		// Then ProtectReg keeps each register live until the first non-phi
		// instruction, preventing subsequent AllocReg calls from evicting it.
		if phiOff, ok := g.phiRegs[name]; ok {
			rv := g.allocReg()
			if len(g.phiProtectedRegVars) == 0 {
				g.emit("%s := ctx.AllocReg()", rv)
			} else {
				g.emit("%s := ctx.AllocRegExcept(%s)", rv, strings.Join(g.phiProtectedRegVars, ", "))
			}
			g.emit("ctx.EmitLoadFromStack(%s, %s)", rv, phiOff)
			g.emit("ctx.ProtectReg(%s)", rv)
			g.phiProtectedRegVars = append(g.phiProtectedRegVars, rv)
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: %s}", dv, rv)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("phi %s has no allocated stack slot", name))
		}

	case *ssa.If:
		cond := g.vals[v.Cond.Name()]
		thenBB := v.Block().Succs[0].Index
		elseBB := v.Block().Succs[1].Index
		if !cond.isDesc {
			panic(fmt.Sprintf("If condition is not a desc: %s", v.Cond))
		}
		// Ensure labels for both targets
		thenLbl := g.ensureBBLabel(thenBB)
		elseLbl := g.ensureBBLabel(elseBB)
		// Reserve intermediate label for then-edge code
		thenEdgeLbl := g.allocLabel()
		g.emit("%s := ctx.W.ReserveLabel()", thenEdgeLbl)

		g.emit("if %s.Loc == LocImm {", cond.goVar)
		g.emit("\tif %s.Imm.Bool() {", cond.goVar)
		// Constant true: emit then-edge phi moves + JMP thenBB
		g.emitEdgePhiMovesIndent(thenBB, "\t\t")
		g.emit("\t\tctx.W.EmitJmp(%s)", thenLbl)
		g.emit("\t} else {")
		// Constant false: emit else-edge phi moves + JMP elseBB
		g.emitEdgePhiMovesIndent(elseBB, "\t\t")
		g.emit("\t\tctx.W.EmitJmp(%s)", elseLbl)
		g.emit("\t}")
		g.emit("} else {")
		// Runtime: CMP + JNE to then-edge, fall through to else-edge
		g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg, 0)", cond.goVar)
		g.emit("\tctx.W.EmitJcc(CcNE, %s)", thenEdgeLbl)
		// Else-edge phi moves + JMP elseBB
		g.emitEdgePhiMovesIndent(elseBB, "\t")
		g.emit("\tctx.W.EmitJmp(%s)", elseLbl)
		// Then-edge label + phi moves + JMP thenBB
		g.emit("\tctx.W.MarkLabel(%s)", thenEdgeLbl)
		g.emitEdgePhiMovesIndent(thenBB, "\t")
		g.emit("\tctx.W.EmitJmp(%s)", thenLbl)
		g.emit("}")
		g.enqueueBB(elseBB)
		g.enqueueBB(thenBB)

	case *ssa.Jump:
		targetBB := v.Block().Succs[0].Index
		g.emitEdgePhiMoves(targetBB)
		if g.bbDone[targetBB] {
			lbl := g.ensureBBLabel(targetBB)
			g.emit("ctx.W.EmitJmp(%s)", lbl)
		} else {
			lbl := g.ensureBBLabel(targetBB)
			g.emit("ctx.W.EmitJmp(%s)", lbl)
			g.enqueueBB(targetBB)
		}

	case *ssa.Convert:
		src := g.resolveValue(v.X)
		dv := g.allocDesc()
		srcType := v.X.Type().Underlying()
		dstType := v.Type().Underlying()
		srcBasic, srcOk := srcType.(*types.Basic)
		dstBasic, dstOk := dstType.(*types.Basic)
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
			if g.storageMode && g.typeName == "StorageSeq" {
				// StorageSeq currently miscompiles int->float in JIT lowering.
				// Force fallback to Go GetValue until the conversion path is fixed.
				panic(fmt.Sprintf("unsupported Convert %s → %s", v.X.Type(), v.Type()))
			}
			// int → float64: emit CVTSI2SD
			g.emit("var %s JITValueDesc", dv)
			g.emit("if %s.Loc == LocImm {", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(%s.Imm.Int()))}", dv, src.goVar)
			g.emit("} else {")
			g.emit("\tctx.W.EmitCvtInt64ToFloat64(RegX0, %s.Reg)", src.goVar)
			g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: %s.Reg}", dv, src.goVar)
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
			ptrReg := g.allocReg()
			lenReg := g.allocReg()
			dv := g.allocDesc()
			excludedRegs := []string{}
			if x.isDesc {
				excludedRegs = append(excludedRegs, x.goVar+".Reg")
				excludedRegs = append(excludedRegs, x.goVar+".Reg2")
			}
			if low.isDesc {
				excludedRegs = append(excludedRegs, low.goVar+".Reg")
				excludedRegs = append(excludedRegs, low.goVar+".Reg2")
			}
			if high.isDesc {
				excludedRegs = append(excludedRegs, high.goVar+".Reg")
				excludedRegs = append(excludedRegs, high.goVar+".Reg2")
			}
			if len(excludedRegs) > 0 {
				g.emit("%s := ctx.AllocRegExcept(%s)", ptrReg, strings.Join(excludedRegs, ", "))
				g.emit("%s := ctx.AllocRegExcept(%s, %s)", lenReg, strings.Join(excludedRegs, ", "), ptrReg)
			} else {
				g.emit("%s := ctx.AllocReg()", ptrReg)
				g.emit("%s := ctx.AllocRegExcept(%s)", lenReg, ptrReg)
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
			g.emit("\t\t\tscratch := ctx.AllocReg()")
			g.emit("\t\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\t\tctx.W.EmitAddInt64(%s, scratch)", ptrReg)
			g.emit("\t\t\tctx.FreeReg(scratch)")
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
			g.emit("\t\tscratch := ctx.AllocReg()")
			g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\tctx.W.EmitSubInt64(%s, scratch)", lenReg)
			g.emit("\t\tctx.FreeReg(scratch)")
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
		low := g.resolveValue(v.Low)
		high := g.resolveValue(v.High)
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
		g.emit("\t\tscratch := ctx.AllocReg()")
		g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", low.goVar)
		g.emit("\t\tctx.W.EmitSubInt64(%s, scratch)", lenReg)
		g.emit("\t\tctx.FreeReg(scratch)")
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
			g.emit("\t\tscratch := ctx.AllocReg()")
			g.emit("\t\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", low.goVar)
			g.emit("\t\tctx.W.EmitAddInt64(%s, scratch)", ptrReg)
			g.emit("\t\tctx.FreeReg(scratch)")
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
	if len(v.Results) == 0 {
		g.emit("if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: NewNil()} }")
		g.emit("ctx.W.EmitMakeNil(result)")
		g.emit("return result")
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newbool":
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tif result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: %s.Imm} }", res.goVar)
		g.emit("\tctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tif result.Loc == LocAny { return %s }", res.goVar)
		g.emit("\tctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newint":
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tif result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: %s.Imm} }", res.goVar)
		g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tif result.Loc == LocAny { return %s }", res.goVar)
		g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newfloat":
		g.emit("if %s.Loc == LocImm {", res.goVar)
		g.emit("\tif result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: %s.Imm} }", res.goVar)
		g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("} else {")
		g.emit("\tif result.Loc == LocAny { return %s }", res.goVar)
		g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
		g.emit("\tctx.FreeReg(%s.Reg)", res.goVar)
		g.emit("}")
		g.emit("return result")
	case "_newnil":
		g.emit("if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: NewNil()} }")
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
		panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
	}
}

// emitReturnMultiBlock handles Return for multi-block functions.
// Emits machine code to construct the result + JMP to the shared epilogue.
func (g *codeGen) emitReturnMultiBlock(v *ssa.Return) {
	if len(v.Results) == 0 {
		g.emit("ctx.W.EmitMakeNil(result)")
		g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
		return
	}
	res := g.vals[v.Results[0].Name()]
	switch res.marker {
	case "_newbool":
		g.emit("ctx.W.EmitMakeBool(result, %s)", res.goVar)
		g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		g.emit("result.Type = tagBool")
	case "_newint":
		g.emit("ctx.W.EmitMakeInt(result, %s)", res.goVar)
		g.emit("if %s.Loc == LocReg { ctx.FreeReg(%s.Reg) }", res.goVar, res.goVar)
		g.emit("result.Type = tagInt")
	case "_newfloat":
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
			g.emit("ctx.EmitMovPairToResult(&%s, &result)", res.goVar)
			g.emit("result.Type = %s.Type", res.goVar)
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

	if g.inlineReturnReg2 != "" {
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
		// Scalar return: move single value to result register
		res := g.resolveValue(v.Results[0])
		if res.isDesc {
			g.emit("ctx.EmitMovToReg(%s, %s)", g.inlineReturnReg, res.goVar)
		} else {
			g.emit("ctx.W.EmitMovRegReg(%s, %s)", g.inlineReturnReg, res.goVar)
		}
	}
	g.emit("ctx.W.EmitJmp(%s)", g.inlineEndLabel)
}

func (g *codeGen) lookup(v ssa.Value) genVal {
	if gv, ok := g.vals[v.Name()]; ok {
		return gv
	}
	panic(fmt.Sprintf("unresolved SSA value: %s", v))
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
	val, ok := constant.Int64Val(c.Value)
	if !ok {
		panic(fmt.Sprintf("constant not int64: %s", c))
	}
	return val
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
