/*
Copyright (C) 2026  Carl-Philip Haensch

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
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func newOptimizerTestEnv() *Env {
	return &Env{Vars: make(Vars), Outer: &Globalenv}
}

func optimizeTestSource(t testing.TB, env *Env, source string) Scmer {
	t.Helper()
	return Optimize(Read("optimizer return test", source), env, nil)
}

func serializedTestExpr(t testing.TB, env *Env, expr Scmer) string {
	t.Helper()
	var out bytes.Buffer
	Serialize(&out, expr, env)
	return out.String()
}

func TestSchemeHelperFreshReturnEnablesMutRewrite(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define fresh_pair (lambda (a b) (list a b)))`, env)

	optimized := optimizeTestSource(t, env, `(append (fresh_pair 1 2) 3)`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "append_mut") {
		t.Fatalf("fresh Scheme helper did not enable append_mut: %s", serialized)
	}
	if got := Eval(optimized, env); !Equal(got, NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})) {
		t.Fatalf("optimized helper call returned %s", String(got))
	}
}

func TestSchemeHelperReturnLengthPropagates(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define fresh_pair_len (lambda (a b) (list a b)))`, env)

	optimized := optimizeTestSource(t, env, `(count (fresh_pair_len 1 2))`)
	if !optimized.IsInt() || optimized.Int() != 2 {
		t.Fatalf("expected exact helper return length to fold count, got %s", serializedTestExpr(t, env, optimized))
	}
}

func TestSchemeHelperReturnMetadataBelongsToBoundProc(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define proc_owned_pair (lambda () (list 1 2)))`, env)

	bound := env.FindRead(Symbol("proc_owned_pair")).Vars[Symbol("proc_owned_pair")]
	if bound.GetTag() != tagProc {
		t.Fatalf("expected procedure binding, got %s", String(bound))
	}
	proc := bound.Proc()
	if proc.OptimizerMeta == nil || proc.OptimizerMeta.Return.Kind() != KindList || proc.OptimizerMeta.Return.Length() != 2 {
		t.Fatalf("return metadata is not attached to procedure: %#v", proc.OptimizerMeta)
	}
}

func TestSchemeHelperReturnMetadataPreservesStructuredTypes(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define structured_pair (lambda (number text) (list (+ number 1) (concat text "x"))))`, env)

	proc := env.FindRead(Symbol("structured_pair")).Vars[Symbol("structured_pair")].Proc()
	if proc.OptimizerMeta == nil || proc.OptimizerMeta.Return.Extra == nil {
		t.Fatal("structured return metadata is missing")
	}
	keys := proc.OptimizerMeta.Return.Extra.Keys
	if keys["0"] == nil || keys["0"].Kind != "number" || keys["1"] == nil || keys["1"].Kind != "string" {
		t.Fatalf("structured return metadata lost element types: %#v", keys)
	}
}

func TestConcurrentSchemeReturnInferenceDoesNotMutateEnvironment(t *testing.T) {
	env := newOptimizerTestEnv()
	expr := NewSlice([]Scmer{
		NewSymbol("define"),
		NewSymbol("concurrent_pair"),
		NewSlice([]Scmer{
			NewSymbol("lambda"),
			NewSlice([]Scmer{NewSymbol("a"), NewSymbol("b")}),
			NewSlice([]Scmer{NewSymbol("list"), NewSymbol("a"), NewSymbol("b")}),
		}),
	})
	callExpr := NewSlice([]Scmer{
		NewSymbol("append"),
		NewSlice([]Scmer{NewSymbol("concurrent_pair"), NewInt(1), NewInt(2)}),
		NewInt(3),
	})

	errors := make(chan error, 16)
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerEnv := &Env{Vars: make(Vars), Outer: env}
			for iteration := 0; iteration < 100; iteration++ {
				optimized := Optimize(CloneOptimizerExpression(expr), env, nil)
				Eval(optimized, workerEnv)
				proc := workerEnv.Vars[Symbol("concurrent_pair")].Proc()
				if proc == nil || proc.OptimizerMeta == nil || proc.OptimizerMeta.Return.Length() != 2 {
					errors <- fmt.Errorf("iteration %d lost return metadata", iteration)
					return
				}
				call := Optimize(CloneOptimizerExpression(callExpr), workerEnv, nil)
				if !strings.Contains(String(call), "append_mut") {
					errors <- fmt.Errorf("iteration %d lost transfer rewrite: %s", iteration, String(call))
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestSchemeHelperReturnMetadataSurvivesReoptimizationAndProcCopies(t *testing.T) {
	env := newOptimizerTestEnv()
	definition := optimizeTestSource(t, env, `(define copied_pair (lambda () (list 1 2)))`)
	definition = Optimize(CloneOptimizerExpression(definition), env, nil)
	Eval(definition, env)

	bound := env.Vars[Symbol("copied_pair")]
	closed := CloseProcedure(bound)
	if bound.Proc().OptimizerMeta == nil || closed.Proc().OptimizerMeta == bound.Proc().OptimizerMeta {
		t.Fatal("closed procedure reused specialization metadata for a changed body/environment")
	}
	if closed.Proc().OptimizerMeta.Return.Length() != bound.Proc().OptimizerMeta.Return.Length() {
		t.Fatal("closed procedure did not preserve immutable return metadata")
	}
}

func TestProcComputeSizeIncludesOptimizerMetadata(t *testing.T) {
	plain := NewProcStruct(Proc{Params: NewSlice(nil), Body: NewNil()})
	typed := *plain.Proc()
	typed.OptimizerMeta = &ProcOptimizerMeta{Return: TypeInfoFromTD(&TypeDescriptor{
		Kind: "list",
		Keys: map[string]*TypeDescriptor{"0": {Kind: "int"}},
	}), HasReturn: true}
	if ComputeSize(NewProcStruct(typed)) <= ComputeSize(plain) {
		t.Fatal("procedure size does not include optimizer metadata")
	}
}

func TestSchemeHelperReturnHintFollowsEnvironmentChain(t *testing.T) {
	parent := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define inherited_pair (lambda (a b) (list a b)))`, parent)
	child := &Env{Vars: make(Vars), Outer: parent}

	optimized := optimizeTestSource(t, child, `(append (inherited_pair 1 2) 3)`)
	if serialized := serializedTestExpr(t, child, optimized); !strings.Contains(serialized, "append_mut") {
		t.Fatalf("inherited helper hint did not enable append_mut: %s", serialized)
	}
}

func TestSchemeHelperLocalBeginDoesNotReplaceOuterHint(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define outer_pair (lambda (a b) (list a b)))`, env)
	optimizeTestSource(t, env, `(begin (define outer_pair (lambda (value) value)) (outer_pair 1))`)

	optimized := optimizeTestSource(t, env, `(append (outer_pair 1 2) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "append_mut") {
		t.Fatalf("local begin replaced outer helper hint: %s", serialized)
	}
}

func TestSchemeHelperSharedReturnStaysImmutable(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define shared_pair (list 1 2))`, env)
	EvalAll("optimizer return test", `(define return_shared_pair (lambda () shared_pair))`, env)

	optimized := optimizeTestSource(t, env, `(append (return_shared_pair) 3)`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("shared Scheme helper return enabled append_mut: %s", serialized)
	}
	Eval(optimized, env)
	if got := env.FindRead("shared_pair").Vars["shared_pair"]; len(got.Slice()) != 2 {
		t.Fatalf("shared helper result was mutated: %s", String(got))
	}
}

func TestProcOwnershipSpecializationPublishesFullProc(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define specialize_append_unique (lambda (values)
		(append_unique values 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25)))`, env)
	base := env.Vars[Symbol("specialize_append_unique")].Proc()
	genericEntry := &JITEntryPoint{}
	base.Compiled = genericEntry

	optimized := optimizeTestSource(t, env, `(lambda (value)
		(specialize_append_unique (list value)))`)
	parts := optimized.Slice()
	call := parts[2].Slice()
	if !call[0].IsProc() {
		t.Fatalf("specialized call head is not a Proc: %s", serializedTestExpr(t, env, optimized))
	}
	variant := call[0].Proc()
	if variant == base || variant.OptimizerMeta == base.OptimizerMeta {
		t.Fatal("specialized Proc did not receive an independent full Proc identity")
	}
	if variant.Compiled == genericEntry {
		t.Fatal("specialized Proc inherited machine code for the generic body")
	}
	if serialized := serializedTestExpr(t, env, variant.Body); !strings.Contains(serialized, "append_unique_mut") {
		t.Fatalf("specialized Proc body did not consume its transferred parameter: %s", serialized)
	}
	snapshot := base.OptimizerMeta.specializations.Load()
	if snapshot == nil || len(snapshot.variants) != 1 {
		t.Fatal("base Proc did not retain exactly one Proc specialization")
	}
	var cached Scmer
	for _, variant := range snapshot.variants {
		cached = variant
	}
	if cached != call[0] {
		t.Fatal("base Proc did not retain the published Proc specialization")
	}
	got := Apply(Eval(optimized, env), NewInt(0))
	if got.Slice()[0].Int() != 0 || len(got.Slice()) != 26 {
		t.Fatalf("specialized Proc returned %s", String(got))
	}
}

func TestProcOwnershipSpecializationFollowsCoalesceNilIntoFilter(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define specialize_optional_filter (lambda (values)
		(filter (coalesceNil values '()) (lambda (value)
			(and (> value 0) (< value 100) (not (equal? value 50)))))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (a b c)
		(specialize_optional_filter (list a b c)))`)
	call := optimized.Slice()[2].Slice()
	if !call[0].IsProc() {
		t.Fatalf("ownership through coalesceNil did not specialize the Proc: %s", serializedTestExpr(t, env, optimized))
	}
	if serialized := serializedTestExpr(t, env, call[0].Proc().Body); !strings.Contains(serialized, "filter_mut") {
		t.Fatalf("specialized Proc did not reuse the optional input list: %s", serialized)
	}
	got := Apply(Eval(optimized, env), NewInt(1), NewInt(50), NewInt(99))
	if !Equal(got, NewSlice([]Scmer{NewInt(1), NewInt(99)})) {
		t.Fatalf("specialized optional filter returned %s", String(got))
	}
}

func TestProcOwnershipSpecializationRetainsNestedOwnershipVariants(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested proc specialization test", `(define specialize_nested_child (lambda (wrapper)
		(match wrapper
			(cons child _tail) (append child 2 3 4 5 6 7 8 9 10)
			_ '())))`, env)
	base := env.Vars[Symbol("specialize_nested_child")].Proc()

	borrowed := optimizeTestSource(t, env, `(lambda (child)
		(specialize_nested_child (list child)))`)
	borrowedCall := borrowed.Slice()[2].Slice()
	if !borrowedCall[0].IsProc() {
		t.Fatalf("fresh wrapper did not specialize: %s", serializedTestExpr(t, env, borrowed))
	}
	if body := serializedTestExpr(t, env, borrowedCall[0].Proc().Body); strings.Contains(body, "append_mut") {
		t.Fatalf("borrowed child was made mutable: %s", body)
	}

	transferred := optimizeTestSource(t, env, `(lambda (value)
		(specialize_nested_child (list (list value))))`)
	transferredCall := transferred.Slice()[2].Slice()
	if !transferredCall[0].IsProc() {
		t.Fatalf("nested fresh child did not specialize: %s", serializedTestExpr(t, env, transferred))
	}
	if body := serializedTestExpr(t, env, transferredCall[0].Proc().Body); !strings.Contains(body, "append_mut") {
		t.Fatalf("nested ownership did not reach the match binding: %s", body)
	}
	if transferredCall[0] == borrowedCall[0] {
		t.Fatal("different nested ownership shapes selected the same Proc")
	}
	snapshot := base.OptimizerMeta.specializations.Load()
	if snapshot == nil || len(snapshot.variants) != 2 {
		t.Fatalf("expected two retained ownership variants, got %#v", snapshot)
	}

	child := NewSlice([]Scmer{NewInt(1)})
	got := Apply(Eval(borrowed, env), child)
	if len(child.Slice()) != 1 || len(got.Slice()) != 10 {
		t.Fatalf("borrowed variant changed its input: child=%s result=%s", String(child), String(got))
	}
	got = Apply(Eval(transferred, env), NewInt(1))
	if len(got.Slice()) != 10 {
		t.Fatalf("transferred variant returned %s", String(got))
	}
}

func TestProcOwnershipSpecializationUsesElementOwnershipForEveryChild(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("element ownership specialization test", `(define specialize_element_child (lambda (wrapper)
		(match wrapper
			(cons child _tail) (append child 2 3 4 5 6 7 8 9 10)
			_ '())))`, env)
	base := env.Vars[Symbol("specialize_element_child")].Proc()
	call := []Scmer{NewSymbol("specialize_element_child"), NewSymbol("fresh")}
	allChildrenOwned := &TypeDescriptor{
		Kind:     "list",
		Transfer: true,
		Length:   UnknownLength,
		Element:  &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength},
	}
	meta := newOptimizerMetainfo()
	variant, ok := trySpecializeProcCall(call, []TypeInfo{tiZero, TypeInfoFromTD(allChildrenOwned)}, env, &meta)
	if !ok || !variant.IsProc() {
		t.Fatal("element ownership did not produce a Proc specialization")
	}
	if body := serializedTestExpr(t, env, variant.Proc().Body); !strings.Contains(body, "append_mut") {
		t.Fatalf("Element.Transfer did not reach a projected child: %s", body)
	}
	if base.OptimizerMeta.specializations.Load() == nil {
		t.Fatal("element ownership specialization was not retained")
	}
}

func TestProcOwnershipSpecializationRejectsUnusedNestedShape(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("unused nested ownership specialization test", `(define specialize_nested_read_only (lambda (wrapper)
		(match wrapper
			(cons child _tail) (car child)
			_ nil)))`, env)

	borrowed := optimizeTestSource(t, env, `(lambda (child)
		(specialize_nested_read_only (list child)))`)
	transferred := optimizeTestSource(t, env, `(lambda (value)
		(specialize_nested_read_only (list (list value))))`)
	borrowedVariant := borrowed.Slice()[2].Slice()[0]
	transferredVariant := transferred.Slice()[2].Slice()[0]
	if !borrowedVariant.IsProc() || transferredVariant.IsProc() {
		t.Fatal("unused nested ownership built a redundant structural Proc body")
	}
	base := env.Vars[Symbol("specialize_nested_read_only")].Proc()
	snapshot := base.OptimizerMeta.specializations.Load()
	if snapshot == nil || len(snapshot.variants) != 1 || len(snapshot.rejected) != 1 {
		t.Fatalf("unused nested ownership was not rejected: %#v", snapshot)
	}
}

func TestProcOwnershipSpecializationRejectsSharedCoalesceFallback(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define shared_filter_fallback (list 7 8 9))`, env)
	EvalAll("proc specialization test", `(define specialize_optional_filter_shared (lambda (values)
		(filter (coalesceNil values shared_filter_fallback) (lambda (value) (> value 0)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value)
		(specialize_optional_filter_shared (list value)))`)
	if call := optimized.Slice()[2].Slice(); call[0].IsProc() {
		t.Fatalf("shared coalesceNil fallback received an ownership specialization: %s", serializedTestExpr(t, env, optimized))
	}
	if got := env.Vars[Symbol("shared_filter_fallback")]; !Equal(got, NewSlice([]Scmer{NewInt(7), NewInt(8), NewInt(9)})) {
		t.Fatalf("shared fallback changed during optimization: %s", String(got))
	}
}

func TestCoalesceConstantBranchesRemainExecutableCode(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(coalesceNil 7 42)`)
	if got := Eval(optimized, env); ToInt(got) != 7 {
		t.Fatalf("optimized coalesceNil returned %s", String(got))
	}

	single := optimizeTestSource(t, env, `(coalesceNil 7)`)
	if ToInt(single) != 7 {
		t.Fatalf("single-argument coalesceNil did not collapse: %s", serializedTestExpr(t, env, single))
	}
}

func TestProcOwnershipSpecializationRequiresLinearParameterUse(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define specialize_shared_twice (lambda (values)
		(list
			(append_unique values 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25)
			values)))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value)
		(specialize_shared_twice (list value)))`)
	call := optimized.Slice()[2].Slice()
	if call[0].IsProc() {
		t.Fatalf("non-linear parameter received an ownership specialization: %s", serializedTestExpr(t, env, optimized))
	}
	got := Apply(Eval(optimized, env), NewInt(0)).Slice()
	if len(got) != 2 || len(got[0].Slice()) != 26 || !Equal(got[1], NewSlice([]Scmer{NewInt(0)})) {
		t.Fatalf("generic helper semantics changed: %s", String(NewSlice(got)))
	}
}

func TestProcOwnershipSpecializationSkipsUnusedTransferFact(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define specialize_read_only (lambda (values)
		(list 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 (car values))))`, env)
	base := env.Vars[Symbol("specialize_read_only")].Proc()

	optimized := optimizeTestSource(t, env, `(lambda (value)
		(specialize_read_only (list value)))`)
	call := optimized.Slice()[2].Slice()
	if call[0].IsProc() {
		t.Fatalf("read-only transfer fact replaced the named call: %s", serializedTestExpr(t, env, optimized))
	}
	snapshot := base.OptimizerMeta.specializations.Load()
	if snapshot != nil {
		t.Fatal("read-only transfer fact reached the specialization cache")
	}
	optimized = optimizeTestSource(t, env, `(lambda (value)
		(specialize_read_only (list value)))`)
	if base.OptimizerMeta.specializations.Load() != snapshot {
		t.Fatal("read-only transfer fact triggered specialization work")
	}
	got := Apply(Eval(optimized, env), NewInt(0)).Slice()
	if len(got) != 26 || got[25].Int() != 0 {
		t.Fatalf("read-only helper returned %s", String(NewSlice(got)))
	}
}

func TestConcurrentProcOwnershipSpecializationPublishesOneProc(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("proc specialization test", `(define specialize_concurrently (lambda (values)
		(append_unique values 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25)))`, env)
	base := env.Vars[Symbol("specialize_concurrently")].Proc()
	call := []Scmer{NewSymbol("specialize_concurrently"), NewSymbol("fresh")}
	argTypes := []TypeInfo{tiZero, tiTransfer.WithTransfer().WithKind(KindList)}

	const workers = 16
	variants := make(chan Scmer, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			meta := newOptimizerMetainfo()
			variant, ok := trySpecializeProcCall(call, argTypes, env, &meta)
			if !ok {
				variants <- NewNil()
				return
			}
			variants <- variant
		}()
	}
	wg.Wait()
	close(variants)

	var published Scmer
	for variant := range variants {
		if !variant.IsProc() {
			t.Fatal("concurrent specialization did not return a Proc")
		}
		if published.IsNil() {
			published = variant
		} else if variant != published {
			t.Fatal("concurrent specialization published multiple Proc identities")
		}
	}
	snapshot := base.OptimizerMeta.specializations.Load()
	if snapshot == nil || len(snapshot.variants) != 1 {
		t.Fatal("concurrent specialization cache does not contain exactly the published Proc")
	}
	for _, variant := range snapshot.variants {
		if variant != published {
			t.Fatal("concurrent specialization cache contains a different Proc")
		}
	}
}

func TestSchemeHelperRecursiveReturnStaysConservative(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define recursive_pair (lambda (n) (if (= n 0) (list n) (recursive_pair (- n 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(append (recursive_pair 2) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("recursive Scheme helper enabled append_mut: %s", serialized)
	}
}

func TestSchemeHelperEvalReturnStaysConservative(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define evaluated_pair (lambda () (eval '(list 1 2))))`, env)

	optimized := optimizeTestSource(t, env, `(append (evaluated_pair) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("eval-using Scheme helper enabled append_mut: %s", serialized)
	}
}

func TestSchemeHelperDynamicRebindInvalidatesReturnInfo(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define rebound_pair (lambda () (list 1 2)))`, env)
	EvalAll("optimizer return test", `(define rebound_pair (lambda (value) value))`, env)

	optimized := optimizeTestSource(t, env, `(append (rebound_pair (list 1 2)) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("dynamically rebound Scheme helper kept stale ownership: %s", serialized)
	}
}

func TestSchemeReturnHintsPreserveNestedMutRewrite(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(set_assoc (filter (list) (lambda (x) true)) "k" "v")`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "set_assoc_mut") {
		t.Fatalf("nested fresh result did not enable set_assoc_mut: %s", serialized)
	}
}

func TestOptimizeInlinesHelperWithCaptureFreeNestedLambda(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define filter_owned (lambda (values)
		(filter values (lambda (x) (> x 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (a b c)
		(filter_owned (list a b c)))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "filter_owned") {
		t.Fatalf("helper with capture-free nested lambda was not inlined: %s", serialized)
	}
	if !strings.Contains(serialized, "!list") && !strings.Contains(serialized, "filter_mut") {
		t.Fatalf("inlined helper did not reuse or stack-allocate its input: %s", serialized)
	}
	if got := Apply(Eval(optimized, env), NewInt(0), NewInt(2), NewInt(3)); !Equal(got, NewSlice([]Scmer{NewInt(2), NewInt(3)})) {
		t.Fatalf("inlined helper returned %s", String(got))
	}
}

func TestInlinedNestedLambdaHelperDoesNotMutateBorrowedParameter(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define filter_borrowed (lambda (values)
		(filter values (lambda (x) (> x 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (filter_borrowed values))`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "filter_mut") {
		t.Fatalf("borrowed helper parameter selected filter_mut: %s", serialized)
	}
	shared := NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)})
	if got := Apply(Eval(optimized, env), shared); !Equal(got, NewSlice([]Scmer{NewInt(2), NewInt(3)})) {
		t.Fatalf("borrowed helper returned %s", String(got))
	}
	if !Equal(shared, NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)})) {
		t.Fatalf("borrowed helper mutated its input: %s", String(shared))
	}
}

func TestOptimizeKeepsNestedLambdaHelperWithOuterCapture(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define add_offset (lambda (offset values)
		(map values (lambda (value) (+ value offset)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (add_offset 2 values))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "add_offset") {
		t.Fatalf("helper with nested outer capture was inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, env), NewSlice([]Scmer{NewInt(1), NewInt(3)}))
	if !Equal(got, NewSlice([]Scmer{NewInt(3), NewInt(5)})) {
		t.Fatalf("capturing helper returned %s", String(got))
	}
}

func TestOptimizeKeepsNestedLambdaHelperFromDifferentEnvironment(t *testing.T) {
	source := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define nested_captured_value 7)`, source)
	EvalAll("nested lambda inline test", `(define nested_captured_helper (lambda (values)
		(map values (lambda (value) (+ value nested_captured_value)))))`, source)

	target := newOptimizerTestEnv()
	target.Vars[Symbol("nested_captured_value")] = NewInt(9)
	target.Vars[Symbol("nested_captured_helper")] = source.Vars[Symbol("nested_captured_helper")]
	optimized := optimizeTestSource(t, target, `(lambda (values) (nested_captured_helper values))`)
	if serialized := serializedTestExpr(t, target, optimized); !strings.Contains(serialized, "nested_captured_helper") {
		t.Fatalf("nested helper from a different environment was inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, target), NewSlice([]Scmer{NewInt(1)}))
	if !Equal(got, NewSlice([]Scmer{NewInt(8)})) {
		t.Fatalf("nested helper used target capture: %s", String(got))
	}
}

func TestNestedLambdaCaptureAnalysisHandlesCycles(t *testing.T) {
	items := make([]Scmer, 2)
	cycle := NewSlice(items)
	items[0] = NewSymbol("lambda")
	items[1] = cycle
	if expressionContainsOuterReference(cycle) {
		t.Fatal("capture analysis treated a small capture-free cycle as an outer reference")
	}
}

func TestOptimizeInlinesSingleUseLeafProc(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_second (lambda (values) (nth values 1)))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (leaf_second values))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "leaf_second") {
		t.Fatalf("single-use leaf proc was not inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, env), NewSlice([]Scmer{NewInt(4), NewInt(9)}))
	if ToInt(got) != 9 {
		t.Fatalf("inlined leaf proc returned %s, want 9", String(got))
	}
}

func TestOptimizeKeepsMultiUseLeafProcCall(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_twice (lambda (value) (+ value value)))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value) (leaf_twice value))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "leaf_twice") {
		t.Fatalf("multi-use leaf proc was inlined: %s", serialized)
	}
}

func TestOptimizeKeepsRecursiveLeafProcCall(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_recurse (lambda (value)
		(if (equal? value 0) 0 (leaf_recurse (- value 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value) (leaf_recurse value))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "leaf_recurse") {
		t.Fatalf("recursive leaf proc was inlined: %s", serialized)
	}
}

func TestOptimizeKeepsLeafProcWithDifferentCapturedBinding(t *testing.T) {
	source := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define captured_value 7)`, source)
	proc := Eval(Optimize(Read("leaf inline test", `(lambda (value) (+ value captured_value))`), source, nil), source)

	target := newOptimizerTestEnv()
	target.Vars[Symbol("captured_value")] = NewInt(9)
	target.Vars[Symbol("captured_leaf")] = proc
	optimized := optimizeTestSource(t, target, `(lambda (value) (captured_leaf value))`)
	if serialized := serializedTestExpr(t, target, optimized); !strings.Contains(serialized, "captured_leaf") {
		t.Fatalf("leaf proc with a different captured binding was inlined: %s", serialized)
	}
}

func BenchmarkOptimizeSchemeHelperFreshReturn(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return benchmark", `(define benchmark_fresh_pair (lambda (a b) (list a b)))`, env)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr := NewSlice([]Scmer{
			NewSymbol("append"),
			NewSlice([]Scmer{NewSymbol("benchmark_fresh_pair"), NewInt(1), NewInt(2)}),
			NewInt(3),
		})
		Optimize(expr, env, nil)
	}
}

func BenchmarkSchemeHelperOwnedReturnAppend(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return benchmark", `(define benchmark_filtered (lambda (a b c d) (filter (list a b c d) (lambda (x) (> x 1)))))`, env)
	optimized := optimizeTestSource(b, env, `(lambda (a b c d e) (append (benchmark_filtered a b c d) e))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	args := []Scmer{NewInt(0), NewInt(2), NewInt(3), NewInt(4), NewInt(5)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fn(args...)
		if len(result.Slice()) != 4 {
			b.Fatal("unexpected result length")
		}
	}
}

// TestRecursiveAccumulatorIfBranchesCountAsSingleUse covers the common
// "return the accumulator on the empty case, otherwise recurse with an
// extended accumulator" idiom. Before this fix, procParameterOwnershipUses
// summed if-branch occurrences instead of treating them as mutually
// exclusive, so the accumulator was always seen as "used twice" and the
// recursive call's append could never be promoted to append_mut.
func TestRecursiveAccumulatorIfBranchesCountAsSingleUse(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("recursive accumulator ownership test", `(define recursive_append_accumulate (lambda (acc remaining)
		(if (equal? remaining '())
			acc
			(recursive_append_accumulate (append acc (list (car remaining))) (cdr remaining)))))`, env)

	base := env.Vars[Symbol("recursive_append_accumulate")].Proc()
	call := []Scmer{NewSymbol("recursive_append_accumulate"), NewSymbol("fresh")}
	freshList := &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}
	meta := newOptimizerMetainfo()
	variant, ok := trySpecializeProcCall(call, []TypeInfo{tiZero, TypeInfoFromTD(freshList)}, env, &meta)
	if !ok || !variant.IsProc() {
		t.Fatal("fresh accumulator did not produce a Proc specialization")
	}
	if body := serializedTestExpr(t, env, variant.Proc().Body); !strings.Contains(body, "append_mut") {
		t.Fatalf("recursive accumulator did not become append_mut: %s", body)
	}
	_ = base

	got := Apply(variant, NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})
	if !Equal(got, want) {
		t.Fatalf("specialized accumulator returned %s, want %s", String(got), String(want))
	}
}

func BenchmarkOptimizeInlinedNestedLambdaHelper(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline benchmark", `(define benchmark_filter_owned (lambda (values) (filter values (lambda (x) (> x 1)))))`, env)
	optimized := optimizeTestSource(b, env, `(lambda (a b c d e)
		(append (benchmark_filter_owned (list a b c d)) e))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	args := []Scmer{NewInt(0), NewInt(2), NewInt(3), NewInt(4), NewInt(5)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fn(args...)
		if len(result.Slice()) != 4 {
			b.Fatal("unexpected result length")
		}
	}
}
