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

// serialExpr is the small prepared-expression layer between callback shape
// recognition and the future fused JIT. It compiles structure once while
// preserving Eval as the compatibility implementation for uncommon forms.
// Every native call node owns its argument frame; instances are serial and
// must never be shared between physical workers.
type serialExpr func(*Env) Scmer

// serialCallFrames owns the nested variadic argument arrays used by one
// serially executed callback. A frame is acquired before evaluating operands,
// so recursive calls naturally use the next depth. Native declarations which
// retain their argument array deliberately bypass this storage.
type serialCallFrames struct {
	frames [][]Scmer
	depth  int
}

// serialExprMayCaptureEnv detects forms which can retain the lexical scope
// created by begin. Without one of these forms the scope is call-local scratch:
// its map can be cleared and reused by the next serial invocation.
func serialExprMayCaptureEnv(expression Scmer) bool {
	expression = serialProcBody(expression)
	if !expression.IsSlice() {
		return false
	}
	items := expression.Slice()
	if len(items) == 0 {
		return false
	}
	if items[0].IsSymbol() {
		switch items[0].String() {
		case "quote":
			return false
		case "lambda", "parser":
			return true
		}
	}
	for _, item := range items {
		if serialExprMayCaptureEnv(item) {
			return true
		}
	}
	return false
}

func (s *serialCallFrames) acquire(size int) []Scmer {
	depth := s.depth
	s.depth++
	if depth == len(s.frames) {
		s.frames = append(s.frames, make([]Scmer, size))
	} else if cap(s.frames[depth]) < size {
		s.frames[depth] = make([]Scmer, size)
	}
	return s.frames[depth][:size]
}

func (s *serialCallFrames) release() {
	s.depth--
}

func prepareSerialExpr(proc *Proc, expression Scmer) serialExpr {
	expression = serialProcBody(expression)
	if expression.IsNthLocalVar() {
		index := int(expression.NthLocalVar())
		return func(en *Env) Scmer { return en.VarsNumbered[index] }
	}
	if !expression.IsSlice() {
		if expression.IsSymbol() {
			symbol := mustSymbol(expression)
			return func(en *Env) Scmer {
				if binding := en.FindRead(symbol); binding != nil {
					if value, ok := binding.Vars[symbol]; ok {
						return value
					}
				}
				return NewNil()
			}
		}
		return func(*Env) Scmer { return expression }
	}

	items := expression.Slice()
	if len(items) == 0 {
		return func(*Env) Scmer { return expression }
	}
	if items[0].IsSymbol() {
		// These control forms intentionally mirror Eval's short-circuit and SQL
		// NULL semantics. Keep both implementations in sync; every other form
		// falls through to Eval instead of growing a second interpreter here.
		switch items[0].String() {
		case "quote":
			value := items[1]
			return func(*Env) Scmer { return value }
		case "begin":
			if !serialExprMayCaptureEnv(expression) {
				operands := prepareSerialOperands(proc, items[1:])
				scope := Env{Vars: make(Vars)}
				return func(en *Env) Scmer {
					clear(scope.Vars)
					scope.VarsNumbered = en.VarsNumbered
					scope.Outer = en
					result := NewNil()
					for _, operand := range operands {
						result = operand(&scope)
					}
					return result
				}
			}
		case "!begin":
			operands := prepareSerialOperands(proc, items[1:])
			return func(en *Env) Scmer {
				result := NewNil()
				for _, operand := range operands {
					result = operand(en)
				}
				return result
			}
		case "and":
			operands := prepareSerialOperands(proc, items[1:])
			return func(en *Env) Scmer {
				unknown := false
				for _, operand := range operands {
					value := operand(en)
					if value.IsNil() {
						unknown = true
					} else if !value.Bool() {
						return NewBool(false)
					}
				}
				if unknown {
					return NewNil()
				}
				return NewBool(true)
			}
		case "or":
			operands := prepareSerialOperands(proc, items[1:])
			return func(en *Env) Scmer {
				unknown := false
				for _, operand := range operands {
					value := operand(en)
					if value.IsNil() {
						unknown = true
					} else if value.Bool() {
						return NewBool(true)
					}
				}
				if unknown {
					return NewNil()
				}
				return NewBool(false)
			}
		case "if":
			operands := prepareSerialOperands(proc, items[1:])
			return func(en *Env) Scmer {
				i := 0
				for i+1 < len(operands) {
					if operands[i](en).Bool() {
						return operands[i+1](en)
					}
					i += 2
				}
				if i < len(operands) {
					return operands[i](en)
				}
				return NewNil()
			}
		case "coalesce", "coalesceNil":
			operands := prepareSerialOperands(proc, items[1:])
			coalesceNil := items[0].String() == "coalesceNil"
			return func(en *Env) Scmer {
				for i, operand := range operands {
					value := operand(en)
					if coalesceNil {
						if !value.IsNil() {
							return value
						}
					} else if i == len(operands)-1 || value.Bool() {
						return value
					}
				}
				return NewNil()
			}
		}
	}

	// Stable non-retaining natives can borrow a call-node-owned frame. A native
	// such as list deliberately falls through to Eval, which supplies an owned
	// frame because the result may retain it.
	native, ok := serialProcResolveNative(proc, items[0])
	if ok {
		declaration := DeclarationForValue(native)
		if declaration != nil && !declaration.RetainsCallArgs {
			fn := native.Func()
			operands := prepareSerialOperands(proc, items[1:])
			frames := serialCallFrames{}
			return func(en *Env) Scmer {
				args := frames.acquire(len(operands))
				defer frames.release()
				for i, operand := range operands {
					args[i] = operand(en)
				}
				return fn(args...)
			}
		}
	}

	return func(en *Env) Scmer { return Eval(expression, en) }
}

func prepareSerialOperands(proc *Proc, expressions []Scmer) []serialExpr {
	result := make([]serialExpr, len(expressions))
	for i, expression := range expressions {
		result[i] = prepareSerialExpr(proc, expression)
	}
	return result
}
