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
	"strings"
	"time"

	"github.com/jtolds/gls"
)

var specialFormsByName map[Symbol]Scmer

const (
	specialFormCallbackKind uint8 = iota
	specialFormEvalKind
	specialFormIfKind
	specialFormMatchKind
	specialFormMatchMutKind
	specialFormBeginMutKind
	specialFormBangBeginKind
)

func registerSpecialForms() {
	specialFormsByName = make(map[Symbol]Scmer)
	register := func(name string, kind uint8, fn SpecialForm) {
		value := NewSpecialForm(name, kind, fn)
		specialFormsByName[Symbol(name)] = value
	}
	register("outer", specialFormCallbackKind, specialOuter)
	register("quote", specialFormCallbackKind, specialQuote)
	register("eval", specialFormEvalKind, nil)
	register("time", specialFormCallbackKind, specialTime)
	register("if", specialFormIfKind, nil)
	register("and", specialFormCallbackKind, specialAnd)
	register("or", specialFormCallbackKind, specialOr)
	register("coalesce", specialFormCallbackKind, specialCoalesce)
	register("coalesceNil", specialFormCallbackKind, specialCoalesceNil)
	register("match", specialFormMatchKind, nil)
	register("match_mut", specialFormMatchMutKind, nil)
	register("define", specialFormCallbackKind, specialDefine)
	register("set", specialFormCallbackKind, specialDefine)
	register("setN", specialFormCallbackKind, specialSetN)
	register("parser", specialFormCallbackKind, specialParser)
	register("optimizer_proc_return", specialFormCallbackKind, specialOptimizerProcReturn)
	register("lambda", specialFormCallbackKind, specialLambda)
	register("begin", specialFormCallbackKind, specialBegin)
	register("begin_mut", specialFormBeginMutKind, nil)
	register("!begin", specialFormBangBeginKind, nil)
	register("!list", specialFormCallbackKind, specialBangList)
	register("!!list", specialFormCallbackKind, specialBangBangList)
	register("parallel", specialFormCallbackKind, specialParallel)
}

func specialFormForSymbol(value Scmer) (Scmer, bool) {
	name, ok := scmerSymbol(value)
	if !ok {
		return NewNil(), false
	}
	form, ok := specialFormsByName[name]
	return form, ok
}

func specialOuter(code []Scmer, en *Env) Scmer {
	if en.Outer == nil {
		return NewNil()
	}
	if code[0].IsSymbol() {
		sym := code[0].Symbol()
		if env := en.Outer.FindRead(sym); env != nil {
			if val, ok := env.Vars[sym]; ok {
				return val
			}
		}
		symStr := string(sym)
		if strings.Contains(symStr, ".") && !strings.Contains(symStr, "\x00") {
			suffix := "\x00" + symStr
			for env := en.Outer; env != nil; env = env.Outer {
				for key, val := range env.Vars {
					if strings.HasSuffix(string(key), suffix) {
						return val
					}
				}
			}
		}
	}
	return Eval(code[0], en.Outer)
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

func specialBegin(code []Scmer, en *Env) Scmer {
	beginEnv := &Env{Vars: make(Vars), VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: false}
	for _, form := range code[:len(code)-1] {
		Eval(form, beginEnv)
	}
	return Eval(code[len(code)-1], beginEnv)
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
