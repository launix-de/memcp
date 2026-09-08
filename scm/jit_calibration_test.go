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
	"math"
	"testing"
)

func TestJITCalibrationRobustLinearFitIgnoresTimingOutlier(t *testing.T) {
	x := []float64{0, 1, 2, 3, 4, 5, 6}
	y := []float64{7, 10, 13, 16, 19, 22, 1000}
	intercept, slope := robustLinearFit(x, y)
	if math.Abs(intercept-7) > 0.001 || math.Abs(slope-3) > 0.001 {
		t.Fatalf("robust fit = %g + %g*x, want 7 + 3*x", intercept, slope)
	}
}

func TestJITCalibrationBreakEvenHasNoHiddenSafetyFactor(t *testing.T) {
	calibration := JITCostCalibration{
		EmitFixedNS:     100,
		EmitUnitNS:      2,
		PublishFixedNS:  50,
		DirectCallNS:    20,
		PredicateUnitNS: 5,
	}
	// Compile: 100 + 10*2 + 50 = 170ns. Saving: (20 + 2*5) - 13 =
	// 17ns/candidate. The exact break-even is therefore ten candidates.
	if got := calibration.BreakEvenCandidates(10, 2, 13, 1); got != 10 {
		t.Fatalf("break-even candidates = %d, want 10", got)
	}
	if got := calibration.BreakEvenCandidates(10, 2, 30, 1); got != math.MaxInt {
		t.Fatalf("non-profitable fusion break-even = %d, want MaxInt", got)
	}
}

func TestJITCalibrationMapReduceBufferBestOf(t *testing.T) {
	calibration := JITCostCalibration{
		Enabled:              true,
		BufferCompileNS:      100,
		BufferCompileUnitNS:  4,
		BufferCurrentNS:      10,
		BufferFusedNS:        5,
		BufferSavedNS:        2,
		BufferProbeCurrentNS: 7,
		BufferProbeFusedNS:   5,
		BufferMaxUnits:       40,
	}
	if got := calibration.MapReduceBufferBreakEven(40, 1); got != 100 {
		t.Fatalf("buffer break-even rows = %d, want 100", got)
	}
	for _, test := range []struct {
		units, columns int
	}{
		{units: 41, columns: 1},
		{units: 40, columns: 2},
	} {
		if got := calibration.MapReduceBufferBreakEven(test.units, test.columns); got != math.MaxInt {
			t.Fatalf("ineligible buffer shape (%d units, %d columns) break-even = %d, want MaxInt", test.units, test.columns, got)
		}
	}
	if got := calibration.MapReduceBufferProbeBreakEven(80, 3, 1024); got == math.MaxInt {
		t.Fatal("arbitrary-arity expression was not admitted for an adaptive probe")
	}
	// 260ns compilation + 5120ns duplicate execution, twice amortized at
	// 2ns/row, plus the probe rows themselves. No hidden >60k floor.
	if got := calibration.MapReduceBufferProbeBreakEven(80, 3, 1024); got != 6404 {
		t.Fatalf("probe break-even = %d, want 6404", got)
	}
	calibration.BufferProbeFusedNS = 9
	if got := calibration.MapReduceBufferProbeBreakEven(80, 3, 1024); got != math.MaxInt {
		t.Fatal("a losing probe with no measured call saving was admitted")
	}
	calibration.DirectCallNS = 2
	if got := calibration.MapReduceBufferProbeBreakEven(80, 3, 1024); got != 10500 {
		t.Fatalf("call-boundary trial break-even = %d, want 10500", got)
	}
}

func TestJITBufferProbeRequiresPureNumericExpression(t *testing.T) {
	for _, test := range []struct {
		source string
		arity  int
		want   bool
	}{
		{"(lambda (acc a b c) (+ acc (+ a (* b c))))", 4, true},
		{"(lambda (acc a) (sql_sum_reduce acc a))", 2, true},
		{"(lambda (acc a) (set_assoc acc a true))", 2, false},
		{"(lambda (acc a) (begin (print a) (+ acc a)))", 2, false},
		{"(lambda (acc a) (a acc))", 2, false},
	} {
		proc := calibrationProcedure(test.source)
		if got := JITBufferProbeSafe(proc, test.arity); got != test.want {
			t.Errorf("probe safety %s = %v, want %v; body=%s", test.source, got, test.want, String(proc.Body))
		}
	}
}

func TestJITCalibrationFilterBufferBestOf(t *testing.T) {
	calibration := JITCostCalibration{
		Enabled:                   true,
		FilterBufferCompileNS:     100,
		FilterBufferCompileUnitNS: 4,
		FilterBufferSavedNS:       5,
		FilterBufferMaxUnits:      20,
	}
	if got := calibration.FilterBufferBreakEven(20, 3); got != 40 {
		t.Fatalf("filter buffer break-even rows = %d, want 40", got)
	}
	if got := calibration.FilterBufferBreakEven(30, 3); got != 56 {
		t.Fatalf("complex filter buffer break-even rows = %d, want 56", got)
	}
	if got := calibration.FilterBufferBreakEven(20, 0); got != math.MaxInt {
		t.Fatalf("zero-column filter break-even = %d, want MaxInt", got)
	}
}

func TestJITCalibrationAnchorsCallBoundaryToTrivialShapes(t *testing.T) {
	observations := []jitCalibrationObservation{
		{workUnits: 0, callNS: 4},
		{workUnits: 0, callNS: 6},
		{workUnits: 10, callNS: 25},
		{workUnits: 20, callNS: 45},
		{workUnits: 30, callNS: 1000}, // scheduler outlier
	}
	boundary := calibrationCallBoundary(observations)
	if boundary != 5 {
		t.Fatalf("call boundary = %g, want median trivial-call cost 5", boundary)
	}
	if unit := calibrationPredicateUnit(observations, boundary); math.Abs(unit-2) > 0.001 {
		t.Fatalf("predicate unit = %g, want robust median 2", unit)
	}
}

func TestJITExpressionCostUsesDeclarationMetadata(t *testing.T) {
	procedure := calibrationProcedure(`(lambda (value) (> (+ value 3) 50))`)
	got := JITExpressionCost(procedure.Body)
	if got <= 2 {
		t.Fatalf("expression cost = %d, expected generated declaration costs", got)
	}
	if got != JITExpressionCost(procedure.Body) {
		t.Fatal("expression cost is not deterministic")
	}
}

func TestJITExpressionCostDoesNotInterpretQuotedData(t *testing.T) {
	quoted := Read("jit calibration quoted data", `(quote ((+ 1 2) (> 3 2)))`)
	for quoted.GetTag() == tagSourceInfo {
		quoted = quoted.SourceInfo().value
	}
	declaration := DeclarationForValue(quoted.Slice()[0])
	want := 1
	if declaration != nil && declaration.Type != nil && declaration.Type.JITInlineCost != 0 {
		want = int(declaration.Type.JITInlineCost)
	}
	if got := JITExpressionCost(quoted); got != want {
		t.Fatalf("quoted expression cost = %d, want only quote emitter cost %d", got, want)
	}
}

func TestJITStartupCalibrationPublishesImmutableCostModel(t *testing.T) {
	first := CalibrateJITCosts()
	second := CurrentJITCosts()
	if first != second {
		t.Fatalf("calibration changed after publication: first=%+v second=%+v", first, second)
	}
	if !jitEnabled {
		if first.Enabled {
			t.Fatal("calibration is enabled in a build without the JIT experiment")
		}
		return
	}
	if !first.Enabled || first.ShapeCount != len(jitCalibrationShapes()) {
		t.Fatalf("unexpected calibrated shape set: %+v", first)
	}
	if first.SamplesPerShape != jitCalibrationSamples || first.CalibrationNS <= 0 {
		t.Fatalf("invalid calibration sampling metadata: %+v", first)
	}
	if first.EmitFixedNS < 0 || first.EmitUnitNS < 0 || first.PublishFixedNS < 0 ||
		first.DirectCallNS < 0 || first.PredicateUnitNS < 0 || first.BufferCompileNS < 0 || first.BufferSavedNS < 0 {
		t.Fatalf("calibration published a negative cost: %+v", first)
	}
	t.Logf("JIT startup calibration: %+v", first)
}
