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
	"sync"
	"testing"
)

func TestKleeneParserWithoutMemoization(t *testing.T) {
	parserValue := Eval(Read("no-memo parser", `(parser '(
		(define values (* (regex "[0-9]+" false false) "," true))
		$
	) values "")`), &Globalenv)
	parser := parserValue.Parser()

	if got := parser.Execute("1,2,3", &Globalenv); String(got) != `(1 2 3)` {
		t.Fatalf("no-memo parser returned %s", String(got))
	}
	if got := parser.Execute("", &Globalenv); String(got) != `()` {
		t.Fatalf("empty no-memo parser returned %s", String(got))
	}
}

// The interpreter path must honour the accumulation form of * / + (go-packrat
// Init/Step/Finish), matching the JIT's emitRepeatAccumulate.
func TestInterpreterAccumulateRepeat(t *testing.T) {
	acc := Eval(Read("acc parser", `(parser '(
		(define v (+ (regex "[a-z]+" false false) "," nil
			(lambda () (list (quote head)))
			(lambda (a x) (append_mut a x))
			(lambda (a) a)))
		$
	) v "")`), &Globalenv).Parser()
	if got := acc.Execute("a,b,c", &Globalenv); String(got) != "(head a b c)" {
		t.Fatalf("accumulate = %s, want (head a b c)", String(got))
	}

	// AND-cascade shape: finish unwraps a lone operand, two or more become (op …).
	cascade := Eval(Read("cascade parser", `(parser '(
		(define b (+ (regex "[a-z]+" false false) (atom "+" false) nil
			(lambda () (list))
			(lambda (a x) (append_mut a x))
			(lambda (a) (if (equal? (count a) 1) (nth a 0) (cons (quote andop) a)))))
		$
	) b "")`), &Globalenv).Parser()
	if got := cascade.Execute("p+q+r", &Globalenv); String(got) != "(andop p q r)" {
		t.Fatalf("cascade multi = %s, want (andop p q r)", String(got))
	}
	if got := cascade.Execute("p", &Globalenv); String(got) != "p" {
		t.Fatalf("cascade single = %s, want p", String(got))
	}
}

func TestSharedParserConcurrentCaptures(t *testing.T) {
	parserValue := Eval(Read("concurrent parser", `(parser '(
		(define value (or
			(regex "value-[0-9]+" false false)
			(regex "word-[a-z]+" false false)))
		$
	) value "")`), &Globalenv)
	parser := parserValue.Parser()

	const workers = 64
	const iterations = 200
	start := make(chan struct{})
	errors := make(chan string, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for iteration := 0; iteration < iterations; iteration++ {
				input := fmt.Sprintf("value-%d", worker*iterations+iteration)
				if got := parser.Execute(input, &Globalenv).String(); got != input {
					errors <- fmt.Sprintf("input %q returned %q", input, got)
					return
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func benchmarkCaptureParser(b *testing.B) *ScmParser {
	b.Helper()
	return Eval(Read("parser benchmark", `(parser '(
		(define values (* (regex "[a-z]+" false false) "," true))
		$
	) values "")`), &Globalenv).Parser()
}

func BenchmarkSharedParserSequential(b *testing.B) {
	parser := benchmarkCaptureParser(b)
	for b.Loop() {
		parser.Execute("alpha,beta,gamma,delta,epsilon", &Globalenv)
	}
}

func BenchmarkSharedParserParallel(b *testing.B) {
	parser := benchmarkCaptureParser(b)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser.Execute("alpha,beta,gamma,delta,epsilon", &Globalenv)
		}
	})
}
