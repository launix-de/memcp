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

// TestJITParserAccumulateSeeded exercises a non-trivial init (seeds a head
// symbol) + a finish that inspects the accumulator - the shape an optimizer
// injection for (cons 'op repeat) / the AND-OR cascade would produce.
func TestJITParserAccumulateSeeded(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	env := &Env{Vars: make(Vars), Outer: &Globalenv}

	seeded := Eval(Read("seeded", `(parser '(
		(atom "X" true)
		(define v (+ (regex "[a-z]+" false true) "," nil
			(lambda () (list (quote head)))
			(lambda (a x) (append_mut a x))
			(lambda (a) a)))
		$
	) v)`), env)
	env.Vars[Symbol("seeded_parser")] = seeded
	sp := seeded.Parser()

	cascade := Eval(Read("cascade", `(parser '(
		(define b (+ (regex "[a-z]+" false true) (atom "+" false) nil
			(lambda () (list))
			(lambda (a x) (append_mut a x))
			(lambda (a) (if (equal? (count a) 1) (nth a 0) (cons (quote op) a)))))
		$
	) b)`), env)
	env.Vars[Symbol("cascade_parser")] = cascade
	cp := cascade.Parser()

	jitCompileEnvironmentParsers(env)
	if sp.Compiled == nil || sp.JITProgram == nil {
		t.Fatal("seeded grammar not JIT-compiled")
	}
	if cp.Compiled == nil || cp.JITProgram == nil {
		t.Fatal("cascade grammar not JIT-compiled")
	}

	if g := sp.Execute(" X a,b,c ", env); String(g) != "(head a b c)" {
		t.Fatalf("seeded JIT = %s, want (head a b c)", String(g))
	}
	if g := sp.Execute(" X solo ", env); String(g) != "(head solo)" {
		t.Fatalf("seeded single = %s, want (head solo)", String(g))
	}
	if g := cp.Execute(" p+q+r ", env); String(g) != "(op p q r)" {
		t.Fatalf("cascade JIT = %s, want (op p q r)", String(g))
	}
	if g := cp.Execute(" lone ", env); String(g) != "lone" {
		t.Fatalf("cascade single = %s, want lone", String(g))
	}
}
