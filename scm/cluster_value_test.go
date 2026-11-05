/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "sync"
import "time"
import "bytes"
import "strings"
import "github.com/google/uuid"
import "testing"
import "runtime"
import "sync/atomic"

func valuePeerPair(t testing.TB) (*SerializationPeer, *SerializationPeer, *atomic.Int64) {
	t.Helper()
	var mu sync.Mutex
	left, right := map[string]func([]byte) error{}, map[string]func([]byte) error{}
	a, b := new(SerializationPeer), new(SerializationPeer)
	count := new(atomic.Int64)
	register := func(table map[string]func([]byte) error) func(string, func([]byte) error) func() {
		return func(id string, receive func([]byte) error) func() {
			mu.Lock()
			if table[id] != nil {
				t.Fatal("duplicate value channel")
			}
			table[id] = receive
			mu.Unlock()
			return func() { mu.Lock(); delete(table, id); mu.Unlock() }
		}
	}
	send := func(table map[string]func([]byte) error) func(string, []byte) error {
		return func(id string, data []byte) error {
			mu.Lock()
			receive := table[id]
			mu.Unlock()
			if receive != nil {
				return receive(data)
			}
			return nil
		}
	}
	a.Register, b.Register = register(left), register(right)
	a.Send, b.Send = send(right), send(left)
	a.Declare = func(id string, data []byte, lambda bool) error { count.Add(1); return b.Receive(id, data, lambda) }
	b.Declare = func(id string, data []byte, lambda bool) error { return a.Receive(id, data, lambda) }
	t.Cleanup(func() { a.Close(); b.Close() })
	return a, b, count
}
func TestClusterValueLambdaContextsShareDeclaration(t *testing.T) {
	sender, _, declarations := valuePeerPair(t)
	syntax := Read(t.Name(), `(lambda (x) (+ x captured))`, false)
	var closures []Scmer
	var code uintptr
	for i := 0; i < 64; i++ {
		proc := Eval(syntax, &Env{Vars: Vars{Symbol("captured"): NewInt(int64(i))}, Outer: &Globalenv})
		ctx := (&SerializationContext{}).WithPeer(sender)
		data, err := MarshalProcedure(proc, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("lambda_rebind")) || !bytes.Contains(data, []byte("cluster_value")) {
			t.Fatal("procedure was not substituted", string(data))
		}
		closure, err := UnmarshalProcedure(data)
		if err != nil {
			t.Fatal(err, string(data))
		}
		ctx.Queued()
		if got := Apply(closure, NewInt(7)); got.Int() != int64(i+7) {
			t.Fatal("wrong context", i, got)
		}
		if jitEnabled {
			if i == 0 {
				code = closure.Proc().nativeCode()
			}
			if code == 0 || code != closure.Proc().nativeCode() {
				t.Fatal("compiled separately per context")
			}
		}
		closures = append(closures, closure)
	}
	if declarations.Load() != 1 {
		t.Fatal("declared code per binding", declarations.Load())
	}
	runtime.KeepAlive(closures)
}

//go:noinline
func declareCollectibleClusterLambda(t *testing.T, sender *SerializationPeer) {
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x captured))`, false), &Env{Vars: Vars{Symbol("captured"): NewInt(11)}, Outer: &Globalenv})
	ctx := (&SerializationContext{}).WithPeer(sender)
	data, err := MarshalProcedure(proc, ctx)
	if err != nil {
		t.Fatal(err)
	}
	value, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Queued()
	if Apply(value, NewInt(7)).Int() != 18 {
		t.Fatal("wrong result")
	}
}
func TestClusterValueGCRemovesBothPeerRegistries(t *testing.T) {
	sender, receiver, _ := valuePeerPair(t)
	declareCollectibleClusterLambda(t, sender)
	deadline := time.Now().Add(3 * time.Second)
	count := func(table *sync.Map) int { n := 0; table.Range(func(_, _ any) bool { n++; return true }); return n }
	for time.Now().Before(deadline) {
		runtime.GC()
		if count(&sender.known) == 0 && count(&receiver.received) == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("GC retained cluster declarations", count(&sender.known), count(&receiver.received))
}
func TestClusterValueRetirementKeepsPreparedReference(t *testing.T) {
	sender, receiver, _ := valuePeerPair(t)
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x captured))`, false), &Env{Vars: Vars{Symbol("captured"): NewInt(31)}, Outer: &Globalenv})
	ctx := (&SerializationContext{}).WithPeer(sender)
	data, err := MarshalProcedure(proc, ctx)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	receiver.received.Range(func(key, _ any) bool { id = key.(string); return false })
	receiver.retireValue(id)
	if _, ok := receiver.received.Load(id); !ok {
		t.Fatal("freed value while sender still prepared its reference")
	}
	value, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Queued()
	if _, ok := receiver.received.Load(id); ok {
		t.Fatal("registry survived ordered retirement")
	}
	if got := Apply(value, NewInt(7)); got.Int() != 38 {
		t.Fatal("retirement invalidated running closure", got)
	}
}

//go:noinline
func forwardCachedLambda(t *testing.T, first, second *SerializationPeer) {
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x captured))`, false), &Env{Vars: Vars{Symbol("captured"): NewInt(13)}, Outer: &Globalenv})
	send := func(peer *SerializationPeer, value Scmer) Scmer {
		ctx := (&SerializationContext{}).WithPeer(peer)
		data, err := MarshalProcedure(value, ctx)
		if err != nil {
			t.Fatal(err)
		}
		result, err := UnmarshalProcedure(data)
		ctx.Queued()
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	middle := send(first, proc)
	last := send(second, middle)
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	retained := false
	first.known.Range(func(_, _ any) bool { retained = true; return false })
	if !retained {
		t.Fatal("forwarding dropped original code ownership")
	}
	if Apply(last, NewInt(7)).Int() != 20 {
		t.Fatal("forwarded context changed")
	}
	runtime.KeepAlive(last)
}
func TestClusterValueForwardingGCCascades(t *testing.T) {
	a, _, _ := valuePeerPair(t)
	c, _, _ := valuePeerPair(t)
	forwardCachedLambda(t, a, c)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		left, right := 0, 0
		a.known.Range(func(_, _ any) bool { left++; return true })
		c.known.Range(func(_, _ any) bool { right++; return true })
		if left == 0 && right == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("forwarded code registrations did not cascade")
}
func TestClusterValuePeerlessSerializationIsSelfContained(t *testing.T) {
	sender, _, _ := valuePeerPair(t)
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x captured))`, false), &Env{Vars: Vars{Symbol("captured"): NewInt(31)}, Outer: &Globalenv})
	ctx := (&SerializationContext{}).WithPeer(sender)
	data, err := MarshalProcedure(proc, ctx)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalProcedure(data)
	ctx.Queued()
	if err != nil {
		t.Fatal(err)
	}
	var persistent bytes.Buffer
	Serialize(&persistent, decoded, &Globalenv, new(SerializationContext))
	if bytes.Contains(persistent.Bytes(), []byte("cluster_value")) {
		t.Fatal("persistent value depends on peer cache")
	}
	restored, err := UnmarshalProcedure(append([]byte{procedureWireVersion}, persistent.Bytes()...))
	if err != nil {
		t.Fatal(err)
	}
	if Apply(restored, NewInt(7)).Int() != 38 {
		t.Fatal("persistent context changed")
	}
}
func BenchmarkClusterLambdaContexts(b *testing.B) {
	for _, mode := range []string{"full-procedure", "peer-template"} {
		b.Run(mode, func(b *testing.B) {
			sender, _, declarations := valuePeerPair(b)
			syntax := Read(b.Name(), `(lambda (x) (+ x captured))`, false)
			compilations := 0
			closures := make([]Scmer, 0, b.N)
			totalBytes := 0
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				proc := Eval(syntax, &Env{Vars: Vars{Symbol("captured"): NewInt(int64(i))}, Outer: &Globalenv})
				ctx := new(SerializationContext)
				if mode == "peer-template" {
					ctx = ctx.WithPeer(sender)
				}
				data, err := MarshalProcedure(proc, ctx)
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += len(data)
				value, err := UnmarshalProcedure(data)
				ctx.Queued()
				if err != nil {
					b.Fatal(err)
				}
				if Apply(value, NewInt(7)).Int() != int64(i+7) {
					b.Fatal("context changed")
				}
				if mode == "full-procedure" {
					compilations++
				}
				closures = append(closures, value)
			}
			b.StopTimer()
			b.ReportMetric(float64(compilations), "factory-builds")
			b.ReportMetric(float64(declarations.Load()), "declarations")
			b.ReportMetric(float64(totalBytes)/float64(b.N), "task-B/op")
			runtime.KeepAlive(closures)
		})
	}
}

func TestClusterValueSnapshotsImperativeCaptureOnce(t *testing.T) {
	sender, _, _ := valuePeerPair(t)
	calls := 0
	env := &Env{Vars: Vars{Symbol("probe"): NewFunc(func(...Scmer) Scmer { calls++; return NewInt(9) })}, Outer: &Globalenv}
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x (outer 1 (probe))))`, false), env)
	ctx := (&SerializationContext{}).WithPeer(sender)
	data, err := MarshalProcedure(proc, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("capture was evaluated more than once", calls)
	}
	value, err := UnmarshalProcedure(data)
	ctx.Queued()
	if err != nil {
		t.Fatal(err)
	}
	if Apply(value, NewInt(7)).Int() != 16 {
		t.Fatal("capture changed")
	}
}
func TestClusterValueNativeJITContextsShareDeclaration(t *testing.T) {
	if !jitEnabled {
		t.Skip("native JIT required")
	}
	sender, _, declarations := valuePeerPair(t)
	factory := CompileJIT(Eval(Read(t.Name(), `(lambda (captured) (lambda (x) (+ x captured)))`, false), &Globalenv), true)
	var code uintptr
	var closures []Scmer
	for i := 0; i < 64; i++ {
		original := Apply(factory, NewInt(int64(i)))
		ctx := (&SerializationContext{}).WithPeer(sender)
		data, err := MarshalProcedure(NewFunc(original.Proc().jitFunction()), ctx)
		if err != nil {
			t.Fatal(err)
		}
		value, err := UnmarshalProcedure(data)
		ctx.Queued()
		if err != nil {
			t.Fatal(err)
		}
		if Apply(value, NewInt(7)).Int() != int64(i+7) {
			t.Fatal("native binding changed")
		}
		if i == 0 {
			code = value.Proc().nativeCode()
		}
		if code == 0 || code != value.Proc().nativeCode() {
			t.Fatal("native lambda was recompiled")
		}
		closures = append(closures, value)
	}
	if declarations.Load() != 1 {
		t.Fatal("native code declared per context", declarations.Load())
	}
	runtime.KeepAlive(closures)
}

func TestClusterValueImmutableDataAndRetirement(t *testing.T) {
	for _, kind := range []string{"string", "list", "dictionary"} {
		t.Run(kind, func(t *testing.T) {
			sender, receiver, declarations := valuePeerPair(t)
			func() {
				value := NewString(strings.Repeat("value", 80))
				if kind == "list" {
					values := make([]Scmer, 64)
					for i := range values {
						values[i] = NewInt(int64(i))
					}
					value = NewSlice(values)
				}
				if kind == "dictionary" {
					pairs := make([]Scmer, 256)
					for i := range pairs {
						pairs[i] = NewInt(int64(i))
					}
					value = NewFastDict(&FastDict{Pairs: pairs})
				}
				ctx := (&SerializationContext{}).WithPeer(sender)
				expr, ok, err := ctx.cachedValue(value)
				if err != nil || !ok {
					t.Fatal("not cached", err)
				}
				decoded := Eval(expr, &Globalenv)
				if !Equal(decoded, value) {
					t.Fatal("value changed")
				}
				again, ok, err := ctx.cachedValue(value)
				if err != nil || !ok || Eval(again, &Globalenv) != decoded || declarations.Load() != 1 {
					t.Fatal("value was not shared", err)
				}
				ctx.Queued()
				runtime.KeepAlive(decoded)
			}()
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				runtime.GC()
				n := 0
				receiver.received.Range(func(_, _ any) bool { n++; return true })
				if n == 0 {
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Fatal("immutable value survived receiver GC")
		})
	}
}
func TestClusterValueRejectsMalformedDeclarations(t *testing.T) {
	_, receiver, _ := valuePeerPair(t)
	for _, source := range []string{"", "(lambda (x) x) 42", "42", "(+ 1 2)", "(lambda (x)"} {
		if err := receiver.Receive(uuid.NewString(), []byte(source), false); err == nil {
			t.Fatal("accepted invalid value", source)
		}
	}
}

func TestClusterValueOptimizedNumberedLocals(t *testing.T) {
	sender, _, _ := valuePeerPair(t)
	proc := Eval(Read(t.Name(), `(lambda (a b c) (+ a b c))`, false), &Globalenv)
	proc.Proc().Body = NewSlice([]Scmer{NewSymbol("+"), NewNthLocalVar(0), NewNthLocalVar(1), NewNthLocalVar(2)})
	proc.Proc().NumVars = 3
	ctx := (&SerializationContext{}).WithPeer(sender)
	data, err := MarshalProcedure(proc, ctx)
	if err != nil {
		t.Fatal(err)
	}
	value, err := UnmarshalProcedure(data)
	ctx.Queued()
	if err != nil {
		t.Fatal(err)
	}
	if Apply(value, NewInt(7), NewInt(11), NewInt(13)).Int() != 31 {
		t.Fatal("numbered locals changed")
	}
}
