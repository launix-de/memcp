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
//   go run ./tools/jitgen/ scm/alu.go                    # list operators
//   go run ./tools/jitgen/ -dump=+ scm/alu.go             # SSA dump for +
//   go run ./tools/jitgen/ -patch scm/alu.go              # patch source
package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var dumpOp string
var doPatch bool
var verbose bool

func main() {
	var files []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-dump=") {
			dumpOp = arg[len("-dump="):]
		} else if arg == "-patch" {
			doPatch = true
		} else if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else {
			files = append(files, arg)
		}
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "usage: jitgen [-dump=OP] [-patch] <file.go> ...\n")
		os.Exit(1)
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

	// Index all SSA functions by source position
	ssaFuncs := map[token.Pos]*ssa.Function{}
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Pos().IsValid() {
			ssaFuncs[fn.Pos()] = fn
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
	for _, astFile := range pkg.Syntax {
		fname := fset.Position(astFile.Pos()).Filename
		abs, _ := filepath.Abs(fname)
		if !absFiles[abs] {
			continue
		}
		ops = append(ops, collectOperators(fset, astFile, fname)...)
	}

	// Process each operator
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
	goVar  string
	isDesc bool   // true = JITValueDesc (Scmer pair), false = Reg (scalar)
	argIdx int    // >= 0: deferred arg reference from IndexAddr, not yet loaded
	marker string // "_newbool"/"_newint"/"_newfloat" for deferred constructors
}

type codeGen struct {
	w         strings.Builder
	vals      map[string]genVal
	paramName string
	nextDesc  int
	nextReg   int
	nextLabel int
	fn        *ssa.Function
	bbLabels  map[int]string  // BB index → label var name
	bbDone    map[int]bool    // BB index → already generated
	bbQueue   []int           // queue of BB indices to generate
	phiRegs    map[string]string // SSA phi name → register var name
	curBlock   int              // current BB index being generated
	multiBlock bool             // true if function has >1 block
	endLabel   string           // label for shared epilogue (multi-block)
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

func (g *codeGen) emit(format string, a ...any) {
	fmt.Fprintf(&g.w, "\t\t\t"+format+"\n", a...)
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
				g.emitPhiMov(phiReg, edge)
				break
			}
		}
	}
}

// emitPhiMov emits a machine-code MOV from an SSA value to a phi register.
func (g *codeGen) emitPhiMov(phiReg string, v ssa.Value) {
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			g.emit("ctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(0)})", phiReg)
		} else if c.Value.Kind() == constant.Bool {
			bval := constant.BoolVal(c.Value)
			if bval {
				g.emit("ctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})", phiReg)
			} else {
				g.emit("ctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(0)})", phiReg)
			}
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			g.emit("ctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(%d)})", phiReg, ival)
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			g.emit("ctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewFloat(%v)})", phiReg, fval)
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.isDesc {
			g.emit("ctx.EmitMovToReg(%s, %s)", phiReg, src.goVar)
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
				g.emitPhiMovIndent(phiReg, edge, indent)
				break
			}
		}
	}
}

// emitPhiMovIndent emits a phi MOV with a given indent prefix.
func (g *codeGen) emitPhiMovIndent(phiReg string, v ssa.Value, indent string) {
	if c, ok := v.(*ssa.Const); ok {
		if c.Value == nil {
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(0)})\n", indent, phiReg)
		} else if c.Value.Kind() == constant.Bool {
			bval := constant.BoolVal(c.Value)
			var ival int
			if bval {
				ival = 1
			}
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(%d)})\n", indent, phiReg, ival)
		} else if c.Value.Kind() == constant.Int {
			ival, _ := constant.Int64Val(c.Value)
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewInt(%d)})\n", indent, phiReg, ival)
		} else if c.Value.Kind() == constant.Float {
			fval, _ := constant.Float64Val(c.Value)
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitMovToReg(%s, JITValueDesc{Loc: LocImm, Imm: NewFloat(%v)})\n", indent, phiReg, fval)
		} else {
			panic(fmt.Sprintf("unsupported phi constant: %s", c))
		}
	} else {
		src := g.vals[v.Name()]
		if src.isDesc {
			fmt.Fprintf(&g.w, "\t\t\t%sctx.EmitMovToReg(%s, %s)\n", indent, phiReg, src.goVar)
		} else {
			panic(fmt.Sprintf("phi edge references unknown value: %s", v))
		}
	}
}

// allocPhiRegs pre-scans the function for phis and allocates a register for each.
func (g *codeGen) allocPhiRegs() {
	for _, block := range g.fn.Blocks {
		for _, instr := range block.Instrs {
			phi, ok := instr.(*ssa.Phi)
			if !ok {
				break
			}
			regVar := g.allocReg()
			g.emit("%s := ctx.AllocReg()", regVar)
			g.phiRegs[phi.Name()] = regVar
		}
	}
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
		vals:     map[string]genVal{},
		fn:       fn,
		bbLabels: map[int]string{},
		bbDone:   map[int]bool{},
		phiRegs:  map[string]string{},
	}
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
		}
	}

	// Emit fixup resolution and epilogue
	if g.multiBlock {
		g.emit("ctx.W.MarkLabel(%s)", g.endLabel)
		g.emit("ctx.W.ResolveFixups()")
	} else if len(g.bbLabels) > 0 {
		g.emit("ctx.W.ResolveFixups()")
	}
	if g.multiBlock {
		g.emit("return result")
	}

	result := fmt.Sprintf("func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {\n%s\t\t}",
		g.w.String())
	return result, ""
}

func (g *codeGen) emitInstr(instr ssa.Instruction) {
	val, isVal := instr.(ssa.Value)
	name := ""
	if isVal {
		name = val.Name()
	}

	switch v := instr.(type) {
	case *ssa.IndexAddr:
		if v.X.Name() == g.paramName {
			idx := constInt(v.Index)
			g.vals[name] = genVal{argIdx: int(idx)}
		} else {
			panic(fmt.Sprintf("IndexAddr on non-parameter: %s", v))
		}

	case *ssa.UnOp:
		if v.Op == token.MUL {
			src := g.vals[v.X.Name()]
			if src.argIdx >= 0 {
				// Fused IndexAddr+Deref → args[i] already describes this argument
				dv := g.allocDesc()
				g.emit("%s := args[%d]", dv, src.argIdx)
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
					panic(fmt.Sprintf("len on non-parameter: %s", v))
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
		case "IsSlice":
			arg := g.vals[v.Call.Args[0].Name()]
			dv := g.allocDesc()
			g.emit("%s := ctx.EmitTagEquals(&%s, tagSlice, JITValueDesc{Loc: LocAny})", dv, arg.goVar)
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
		case "NewBool":
			src := g.lookup(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newbool"}
		case "NewInt":
			src := g.lookup(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newint"}
		case "NewFloat":
			src := g.lookup(v.Call.Args[0])
			g.vals[name] = genVal{goVar: src.goVar, marker: "_newfloat"}
		case "NewNil":
			g.vals[name] = genVal{goVar: "", marker: "_newnil"}
		default:
			panic(fmt.Sprintf("unsupported call: %s", v))
		}

	case *ssa.BinOp:
		xVal := g.lookup(v.X)
		cc := opToCC(v.Op)
		goOp := goOpStr(v.Op)
		if cc != "" {
			dv := g.allocDesc()
			if c, ok := v.Y.(*ssa.Const); ok {
				cmpVal := c.Int64()
				// Constant-fold if x is LocImm
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm {", xVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Int() %s %d)}", dv, xVal.goVar, goOp, cmpVal)
				g.emit("} else {")
				// Fresh register for result — CMP is non-destructive, SetCC writes only the target
				rv := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv)
				g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg, %d)", xVal.goVar, cmpVal)
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
				g.emit("}")
			} else {
				yVal := g.lookup(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", xVal.goVar, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOp, yVal.goVar)
				g.emit("} else {")
				rv := g.allocReg()
				g.emit("\t%s := ctx.AllocReg()", rv)
				g.emit("\tctx.W.EmitCmpInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
				g.emit("\tctx.W.EmitSetcc(%s, %s)", rv, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: %s}", dv, rv)
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
				// Materialize constant into scratch reg, then ALU
				g.emit("\tscratch := ctx.AllocReg()")
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%d))", cmpVal)
				g.emit("\tctx.W.%s(%s.Reg, scratch)", aluOp, xVal.goVar)
				g.emit("\tctx.FreeReg(scratch)")
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				g.emit("}")
			} else {
				yVal := g.lookup(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", xVal.goVar, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOpStr(v.Op), yVal.goVar)
				g.emit("} else if %s.Loc == LocImm {", xVal.goVar)
				// x is const, y is reg → materialize x, ALU
				g.emit("\tscratch := ctx.AllocReg()")
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", xVal.goVar)
				g.emit("\tctx.W.%s(scratch, %s.Reg)", aluOp, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}", dv)
				g.emit("} else if %s.Loc == LocImm {", yVal.goVar)
				// y is const, x is reg → materialize y, ALU
				g.emit("\tscratch := ctx.AllocReg()")
				g.emit("\tctx.W.EmitMovRegImm64(scratch, uint64(%s.Imm.Int()))", yVal.goVar)
				g.emit("\tctx.W.%s(%s.Reg, scratch)", aluOp, xVal.goVar)
				g.emit("\tctx.FreeReg(scratch)")
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				g.emit("} else {")
				g.emit("\tctx.W.%s(%s.Reg, %s.Reg)", aluOp, xVal.goVar, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: %s.Reg}", dv, xVal.goVar)
				g.emit("}")
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("unsupported BinOp %s", v.Op))
		}

	case *ssa.Return:
		if g.multiBlock {
			g.emitReturnMultiBlock(v)
		} else {
			g.emitReturnSingleBlock(v)
		}

	case *ssa.Phi:
		// Phi values are set by predecessor edges (emitEdgePhiMoves).
		// Create a variable wrapping the phi's pre-allocated register.
		if regVar, ok := g.phiRegs[name]; ok {
			dv := g.allocDesc()
			g.emit("%s := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: %s}", dv, regVar)
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("phi %s has no allocated register", name))
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
	default:
		panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
	}
	g.emit("ctx.W.EmitJmp(%s)", g.endLabel)
}

func (g *codeGen) lookup(v ssa.Value) genVal {
	if gv, ok := g.vals[v.Name()]; ok {
		return gv
	}
	panic(fmt.Sprintf("unresolved SSA value: %s", v))
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
		if old != "nil" && !strings.HasPrefix(old, "func(") && !strings.HasPrefix(old, "nil ") {
			fmt.Printf("  %s: %s JITEmit field is %q — skipping\n", path, p.opName, old)
			continue
		}
		if old == p.newText {
			continue
		}
		src = append(src[:p.startOff], append([]byte(p.newText), src[endOff:]...)...)
		fmt.Printf("  %s: patched %s\n", path, p.opName)
	}

	if err := os.WriteFile(path, src, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", path, err)
	}
}
