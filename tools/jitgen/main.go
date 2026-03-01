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

func (g *codeGen) emit(format string, a ...any) {
	fmt.Fprintf(&g.w, "\t\t\t"+format+"\n", a...)
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

	g := &codeGen{vals: map[string]genVal{}}
	if len(fn.Params) > 0 {
		g.paramName = fn.Params[0].Name()
	}

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			g.emitInstr(instr)
		}
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
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Imm: NewBool(%s.Imm.Int() %s %d)}", dv, xVal.goVar, goOp, cmpVal)
				g.emit("} else {")
				g.emit("\tctx.W.EmitCmpRegImm32(%s.Reg, %d)", xVal.goVar, cmpVal)
				g.emit("\tctx.W.EmitSetcc(%s.Reg, %s)", xVal.goVar, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s.Reg}", dv, xVal.goVar)
				g.emit("}")
			} else {
				yVal := g.lookup(v.Y)
				g.emit("var %s JITValueDesc", dv)
				g.emit("if %s.Loc == LocImm && %s.Loc == LocImm {", xVal.goVar, yVal.goVar)
				g.emit("\t%s = JITValueDesc{Loc: LocImm, Imm: NewBool(%s.Imm.Int() %s %s.Imm.Int())}", dv, xVal.goVar, goOp, yVal.goVar)
				g.emit("} else {")
				g.emit("\tctx.W.EmitCmpInt64(%s.Reg, %s.Reg)", xVal.goVar, yVal.goVar)
				g.emit("\tctx.W.EmitSetcc(%s.Reg, %s)", xVal.goVar, cc)
				g.emit("\t%s = JITValueDesc{Loc: LocReg, Reg: %s.Reg}", dv, xVal.goVar)
				g.emit("}")
			}
			g.vals[name] = genVal{goVar: dv, isDesc: true}
		} else {
			panic(fmt.Sprintf("unsupported BinOp %s", v.Op))
		}

	case *ssa.Return:
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
			g.emit("\tctx.W.EmitMakeBool(result, %s)", res.goVar)
			g.emit("}")
			g.emit("return result")
		case "_newint":
			g.emit("if %s.Loc == LocImm {", res.goVar)
			g.emit("\tif result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: %s.Imm} }", res.goVar)
			g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
			g.emit("} else {")
			g.emit("\tctx.W.EmitMakeInt(result, %s)", res.goVar)
			g.emit("}")
			g.emit("return result")
		case "_newfloat":
			g.emit("if %s.Loc == LocImm {", res.goVar)
			g.emit("\tif result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: %s.Imm} }", res.goVar)
			g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("} else {")
			g.emit("\tctx.W.EmitMakeFloat(result, %s)", res.goVar)
			g.emit("}")
			g.emit("return result")
		case "_newnil":
			g.emit("if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: NewNil()} }")
			g.emit("ctx.W.EmitMakeNil(result)")
			g.emit("return result")
		default:
			panic(fmt.Sprintf("unsupported return type for %s", v.Results[0]))
		}

	default:
		panic(instrDesc(instr))
	}
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

// goOpStr maps a Go comparison token to the Go operator string for codegen.
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
