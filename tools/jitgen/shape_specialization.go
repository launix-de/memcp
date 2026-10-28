/*
	Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; see <https://www.gnu.org/licenses/>.
*/
package main

import "fmt"
import "go/types"
import "go/token"
import "go/constant"
import "golang.org/x/tools/go/ssa"

type shapeExpression struct {
	code       string
	structural bool
}

// callShapeCondition finds an SSA condition which depends solely on the
// emitter's argument directory. Its Go expression executes during compilation,
// not in the emitted query. Runtime data and loop-carried values are never
// guessed. The ordinary inline budget bounds specialization, as for helpers.
func (g *codeGen) callShapeCondition(fn *ssa.Function, args []ssa.Value, proof *ssaReachability) (*ssa.If, string) {
	// Storage emitters expose a receiver and index, not a virtual variadic
	// argument directory. Their native ABI remains independent of args.
	if g.storageMode {
		return nil, ""
	}
	lengths := make(map[ssa.Value]string)
	for i, arg := range args {
		value := g.vals[g.rewriteSSAValue(arg).Name()]
		if value.marker == "_variadic_args" {
			code := fmt.Sprintf("(len(args)-%d)", value.variadicOffset)
			if value.variadicLenKnown {
				code = fmt.Sprint(value.variadicLen)
			}
			lengths[fn.Params[i]] = code
		}
	}
	visiting := make(map[ssa.Value]bool)
	var expression func(ssa.Value) shapeExpression
	var length func(ssa.Value) shapeExpression
	length = func(v ssa.Value) shapeExpression {
		v = proof.replacement(v)
		if code, ok := lengths[v]; ok {
			return shapeExpression{code, true}
		}
		if c, ok := v.(*ssa.Const); ok && c.Value == nil {
			return shapeExpression{"0", false}
		}
		if slice, ok := v.(*ssa.Slice); ok {
			high := length(slice.X)
			if slice.High != nil {
				high = expression(slice.High)
			}
			low := shapeExpression{"0", false}
			if slice.Low != nil {
				low = expression(slice.Low)
			}
			if high.code != "" && low.code != "" {
				return shapeExpression{"(" + high.code + "-" + low.code + ")", high.structural || low.structural}
			}
		}
		return shapeExpression{}
	}
	expression = func(v ssa.Value) shapeExpression {
		v = proof.replacement(v)
		if visiting[v] {
			return shapeExpression{}
		}
		visiting[v] = true
		defer delete(visiting, v)
		if c, ok := v.(*ssa.Const); ok && c.Value != nil && (c.Value.Kind() == constant.Int || c.Value.Kind() == constant.Bool) {
			return shapeExpression{c.Value.ExactString(), false}
		}
		switch value := v.(type) {
		case *ssa.Call:
			if builtin, ok := value.Call.Value.(*ssa.Builtin); ok && builtin.Name() == "len" && len(value.Call.Args) == 1 {
				return length(value.Call.Args[0])
			}
		case *ssa.BinOp:
			left, right := expression(value.X), expression(value.Y)
			if left.code == "" || right.code == "" {
				return shapeExpression{}
			}
			switch value.Op {
			case token.ADD, token.SUB, token.REM, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.AND:
				if value.Op == token.REM {
					c, ok := value.Y.(*ssa.Const)
					if !ok || c.Value == nil || constant.Sign(c.Value) == 0 {
						return shapeExpression{}
					}
				}
				return shapeExpression{"(" + left.code + value.Op.String() + right.code + ")", left.structural || right.structural}
			}
		case *ssa.UnOp:
			if value.Op == token.NOT {
				child := expression(value.X)
				if child.code != "" {
					child.code = "(!" + child.code + ")"
					return child
				}
			}
		}
		return shapeExpression{}
	}
	for _, block := range fn.Blocks {
		if !proof.blocks[block] {
			continue
		}
		branch, ok := block.Instrs[len(block.Instrs)-1].(*ssa.If)
		if !ok {
			continue
		}
		if _, known := proof.successor(branch); known {
			continue
		}
		if staged := expression(branch.Cond); staged.code != "" && staged.structural {
			return branch, staged.code
		}
	}
	return nil, ""
}

// tryShapeSpecialization stages a structural branch around two independently
// lowered helpers. Exactly one arm emits machine instructions. Unsupported
// arms keep the existing generic Go ABI boundary; no partial machine code is
// emitted for them. Nothing here depends on an operator or helper name.
func (g *codeGen) tryShapeSpecialization(fn *ssa.Function, args []ssa.Value, bindings map[ssa.Value]*ssa.Const, proof *ssaReachability) (result genVal, ok bool) {
	if fn.Signature.Results().Len() != 1 {
		return
	}
	words := goCallWordCount(fn.Signature.Results().At(0).Type())
	if words < 1 || words > 3 {
		return
	}
	branch, condition := g.callShapeCondition(fn, args, proof)
	if branch == nil {
		return
	}
	trial := g.clone()
	defer func() {
		if r := recover(); r != nil {
			if verbose {
				fmt.Printf("jitgen: structural specialization of %s: %v\n", fn, r)
			}
			result = genVal{}
			ok = false
		}
	}()
	output := trial.allocDesc()
	trial.emit("var %s JITValueDesc", output)
	trial.emit("if %s {", condition)
	anyInline := false
	for arm, selected := range []bool{true, false} {
		if arm == 1 {
			trial.emit("} else {")
		}
		assumptions := cloneMap(bindings)
		assumptions[branch.Cond] = ssa.NewConst(constant.MakeBool(selected), types.Typ[types.Bool])
		// Exactly one arm emits query code, but both consume generator output.
		// Charge alternatives to the existing expansion budget as well; a
		// sequence of structural branches must not produce an exponential
		// emitter. The static inline cost remains a conservative upper bound.
		trial.inlineInstructions += inlineInstructionCount(fn, proof)
		child, inline := trial.tryInlineCall(fn, args, assumptions)
		if !inline {
			name := trial.allocTemp("shapeCall")
			if !trial.emitGenericStaticCall(name, fn, args) {
				return genVal{}, false
			}
			child = trial.vals[name]
		}
		if !child.isDesc || child.goVar == "" {
			return genVal{}, false
		}
		anyInline = anyInline || inline
		trial.emit("%s = %s", output, child.goVar)
	}
	trial.emit("}")
	if !anyInline {
		return
	}
	result = genVal{goVar: output, isDesc: true}
	if _, slice := fn.Signature.Results().At(0).Type().Underlying().(*types.Slice); slice {
		result.marker = "_slice"
	}
	generated, declarations := trial.w.String(), trial.wDecl.String()
	trial.w.Reset()
	trial.wDecl.Reset()
	*g = *trial
	g.w.WriteString(generated)
	g.wDecl.WriteString(declarations)
	return result, true
}
