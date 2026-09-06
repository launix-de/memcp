package scm

import "testing"

// TestJITParserAccumulateRepeat exercises the accumulation form of + :
// accInit/accStep/accFinish instead of pushMark/mergeMark. The result must
// match the plain collecting form.
func TestJITParserAccumulateRepeat(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	env := &Env{Vars: make(Vars), Outer: &Globalenv}

	plain := Eval(Read("plain", `(parser '(
		(atom "SELECT" true)
		(define v (+ (regex "[a-z]+" false true) ","))
		$
	) v)`), env).Parser()
	want := plain.Execute(" select alpha,beta,gamma ", env)

	accParser := Eval(Read("acc", `(parser '(
		(atom "SELECT" true)
		(define v (+ (regex "[a-z]+" false true) "," nil
			(lambda () (list))
			(lambda (a x) (append_mut a x))
			(lambda (a) a)))
		$
	) v)`), env)
	env.Vars[Symbol("acc_parser")] = accParser
	p := accParser.Parser()

	jitCompileEnvironmentParsers(env)
	if p.Compiled == nil {
		t.Fatal("accumulate grammar was not compiled")
	}
	if p.JITProgram == nil {
		t.Fatal("no JIT program")
	}
	got := p.Execute(" select alpha,beta,gamma ", env)
	if !Equal(got, want) {
		t.Fatalf("accumulate JIT = %s, want %s", String(got), String(want))
	}

	// single item, and (via a fresh execute) re-init on a second run
	if g := p.Execute(" select solo ", env); String(g) != "(solo)" {
		t.Fatalf("single item = %s, want (solo)", String(g))
	}
	if g := p.Execute(" select a,b ", env); String(g) != "(a b)" {
		t.Fatalf("re-run = %s, want (a b)", String(g))
	}
}
