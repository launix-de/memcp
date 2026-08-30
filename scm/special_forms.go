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
	"fmt"
	"runtime"
	"time"

	"github.com/jtolds/gls"
)

var specialFormNames = make(map[*byte]string)

// DeclareSpecialForm registers syntax as a normal global callable while
// preserving its unevaluated-operand calling convention. Optimize and JIT
// hooks live on def.Type just like they do for ordinary declarations.
func DeclareSpecialForm(env *Env, def *Declaration, fn SpecialForm) {
	Declare(env, def)
	value := NewSpecialForm(fn)
	env.Vars[Symbol(def.Name)] = value
	specialFormNames[value.ptr] = def.Name
}

func registerSpecialForms() {
	register := func(name string, fn SpecialForm) {
		value := NewSpecialForm(fn)
		Globalenv.Vars[Symbol(name)] = value
		specialFormNames[value.ptr] = name
	}
	register("outer", nil)
	register("setN", specialSetN)
	register("parser", specialParser)
	register("optimizer_proc_return", specialOptimizerProcReturn)
	register("!list", specialBangList)
	register("!!list", specialBangBangList)
	register("match_mut", nil)
	register("begin_mut", nil)
	register("!begin", nil)
}

func specialFormName(value Scmer) string {
	if name, ok := specialFormNames[value.ptr]; ok {
		return name
	}
	panic("unregistered special form")
}

func resolveSpecialFormSymbol(value Scmer, env *Env) (Scmer, bool) {
	symbol, symbolic := scmerSymbol(value)
	if !symbolic {
		return NewNil(), false
	}
	if owner := env.FindRead(symbol); owner != nil {
		if resolved, exists := owner.Vars[symbol]; exists {
			if resolved.GetTag() == tagSpecialForm {
				return resolved, true
			}
			return NewNil(), false
		}
	}
	resolved, exists := Globalenv.Vars[symbol]
	return resolved, exists && resolved.GetTag() == tagSpecialForm
}

func specialQuote(code []Scmer, _ *Env) Scmer { return code[0] }

func specialTime(code []Scmer, en *Env) Scmer {
	var start time.Time
	if TracePrint {
		start = time.Now()
	}
	var result Scmer
	if Trace != nil {
		label := "(time)"
		if len(code) > 1 {
			label = String(Eval(code[1], en))
		}
		Trace.Duration(label, "scm", func() { result = Eval(code[0], en) })
	} else {
		result = Eval(code[0], en)
	}
	if TracePrint {
		message := "trace " + time.Since(start).String()
		if len(code) > 1 {
			message += " " + String(Eval(code[1], en))
		}
		TracePrintFunc(message)
	}
	return result
}

func specialAnd(code []Scmer, en *Env) Scmer {
	unknown := false
	for _, expression := range code {
		value := Eval(expression, en)
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

func specialOr(code []Scmer, en *Env) Scmer {
	unknown := false
	for _, expression := range code {
		value := Eval(expression, en)
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

func specialCoalesce(code []Scmer, en *Env) Scmer {
	for i, expression := range code {
		value := Eval(expression, en)
		if i == len(code)-1 || value.Bool() {
			return value
		}
	}
	return NewNil()
}

func specialCoalesceNil(code []Scmer, en *Env) Scmer {
	for _, expression := range code {
		value := Eval(expression, en)
		if !value.IsNil() {
			return value
		}
	}
	return NewNil()
}

func specialDefine(code []Scmer, en *Env) Scmer {
	value := Eval(code[1], en)
	target := en
	for target != nil && target.Nodefine {
		target = target.Outer
	}
	if target == nil {
		target = &Globalenv
	}
	if target.Vars == nil {
		target.Vars = make(Vars)
	}
	target.Vars[mustSymbol(code[0])] = value
	return value
}

func specialSetN(code []Scmer, en *Env) Scmer {
	value := Eval(code[1], en)
	idx := mustNthLocalVar(code[0])
	if int(idx) >= len(en.VarsNumbered) {
		buf := make([]byte, 8192)
		n := runtime.Stack(buf, false)
		panic(fmt.Sprintf("setN(%d) out of range (len=%d)\n%s", int(idx), len(en.VarsNumbered), buf[:n]))
	}
	en.VarsNumbered[int(idx)] = value
	return value
}

func specialParser(code []Scmer, en *Env) Scmer {
	if len(code) > 2 {
		return NewScmParser(NewParser(code[0], code[1], code[2], en, true))
	}
	if len(code) > 1 {
		return NewScmParser(NewParser(code[0], code[1], NewNil(), en, true))
	}
	return NewScmParser(NewParser(code[0], NewNil(), NewNil(), en, false))
}

func specialOptimizerProcReturn(code []Scmer, en *Env) Scmer {
	if len(code) != 2 {
		panic("optimizer_proc_return expects procedure and return metadata")
	}
	value := Eval(code[0], en)
	if value.GetTag() != tagProc {
		return value
	}
	metadataValue := Eval(code[1], en)
	if metadataValue.GetTag() != tagAny {
		panic("optimizer_proc_return expects internal return metadata")
	}
	metadata, ok := metadataValue.Any().(optimizerProcReturnTemplate)
	if !ok {
		panic("optimizer_proc_return received invalid return metadata")
	}
	proc := *value.Proc()
	proc.OptimizerMeta = &ProcOptimizerMeta{Return: metadata.Return, HasReturn: metadata.HasReturn}
	return NewProcStruct(proc)
}

func specialLambda(code []Scmer, en *Env) Scmer {
	params := code[0]
	if params.IsSourceInfo() {
		params = params.SourceInfo().value
	}
	numVars := 0
	if len(code) > 2 {
		numVars = int(code[2].Int())
	}
	return NewProcStruct(Proc{
		Params:       params,
		Body:         code[1],
		En:           en,
		NumVars:      numVars,
		NumberedOnly: procCanUseNumberedOnly(params, code[1], numVars),
	})
}

func specialBangList(code []Scmer, en *Env) Scmer {
	start := int(code[0].NthLocalVar())
	count := int(ToInt(code[1]))
	if start+count > len(en.VarsNumbered) {
		buf := make([]byte, 8192)
		n := runtime.Stack(buf, false)
		panic(fmt.Sprintf("!list start=%d count=%d out of range (len=%d)\n%s", start, count, len(en.VarsNumbered), buf[:n]))
	}
	for i := 0; i < count && i+2 < len(code); i++ {
		en.VarsNumbered[start+i] = Eval(code[i+2], en)
	}
	return NewSlice(en.VarsNumbered[start : start+count])
}

func specialBangBangList(code []Scmer, en *Env) Scmer {
	if len(code) == 2 && code[0].IsNthLocalVar() {
		start := int(code[0].NthLocalVar())
		capacity := int(ToInt(code[1]))
		if capacity < 0 {
			capacity = 0
		}
		if start+capacity > len(en.VarsNumbered) {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			panic(fmt.Sprintf("!!list start=%d cap=%d out of range (len=%d)\n%s", start, capacity, len(en.VarsNumbered), buf[:n]))
		}
		return NewSlice(en.VarsNumbered[start : start : start+capacity])
	}
	if len(code) == 1 {
		capacity := int(ToInt(Eval(code[0], en)))
		if capacity < 0 {
			capacity = 0
		}
		return NewSlice(make([]Scmer, 0, capacity))
	}
	panic("!!list expects either (!!list cap) or optimized (!!list NthLocalVar(start) cap)")
}

func specialParallel(code []Scmer, en *Env) Scmer {
	if len(code) == 0 {
		return NewNil()
	}
	errs := make(chan any, len(code))
	for _, expression := range code {
		expression := expression
		gls.Go(func(value Scmer) func() {
			return func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						errs <- recovered
					} else {
						errs <- nil
					}
				}()
				Eval(value, en)
			}
		}(expression))
	}
	for range code {
		if err := <-errs; err != nil {
			panic(err)
		}
	}
	return NewNil()
}
