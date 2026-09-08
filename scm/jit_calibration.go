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
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	jitCalibrationSamples      = 5
	jitCalibrationMinimumRound = 100 * time.Microsecond
	jitCalibrationBufferSize   = 256 * 1024
)

// JITCostCalibration converts architecture-neutral emission and predicate
// work estimates into costs measured on the machine running MemCP. The value is
// immutable after startup publication; query compilation can therefore read it
// without locks or hot-path calibration branches.
type JITCostCalibration struct {
	Enabled                   bool
	ShapeCount                int
	SamplesPerShape           int
	CalibrationNS             int64
	EmitFixedNS               float64
	EmitUnitNS                float64
	PublishFixedNS            float64
	DirectCallNS              float64
	PredicateUnitNS           float64
	BufferCompileNS           float64
	BufferCompileUnitNS       float64
	BufferCurrentNS           float64
	BufferFusedNS             float64
	BufferSavedNS             float64
	BufferMaxUnits            int
	FilterBufferCompileNS     float64
	FilterBufferCompileUnitNS float64
	FilterBufferCurrentNS     float64
	FilterBufferFusedNS       float64
	FilterBufferSavedNS       float64
	FilterBufferMaxUnits      int
	FilterBufferMinimumRows   int
}

// EstimateCompileNS estimates one query-local emission and publication. The
// expression cost is supplied by JITExpressionCost from jitgen's
// architecture-neutral planning metadata; startup calibration converts that
// structural estimate to this machine's time.
func (calibration JITCostCalibration) EstimateCompileNS(expressionCost int) float64 {
	if expressionCost < 0 {
		expressionCost = 0
	}
	return calibration.EmitFixedNS + calibration.EmitUnitNS*float64(expressionCost) + calibration.PublishFixedNS
}

// EstimateFilterCallNS estimates the current per-row Proc boundary plus the
// predicate work represented by jitgen cost units.
func (calibration JITCostCalibration) EstimateFilterCallNS(predicateUnits int) float64 {
	if predicateUnits < 0 {
		predicateUnits = 0
	}
	return calibration.DirectCallNS + calibration.PredicateUnitNS*float64(predicateUnits)
}

// BreakEvenCandidates returns how many candidate rows must pass through the
// current filter boundary before one compilation amortizes. No safety factor is
// applied: measurement stability comes from the calibration sample set rather
// than systematically preferring either implementation.
func (calibration JITCostCalibration) BreakEvenCandidates(expressionCost, predicateUnits int, fusedPerCandidateNS float64, expectedExecutions int) int {
	if expectedExecutions < 1 {
		expectedExecutions = 1
	}
	saved := calibration.EstimateFilterCallNS(predicateUnits) - fusedPerCandidateNS
	if saved <= 0 {
		return math.MaxInt
	}
	return int(math.Ceil(calibration.EstimateCompileNS(expressionCost) / (saved * float64(expectedExecutions))))
}

// MapReduceBufferBreakEven returns the number of rows needed to amortize one
// typed buffer-loop compilation. The startup probe measures the same simple
// accumulator shape admitted by this fast criterion.
func (calibration JITCostCalibration) MapReduceBufferBreakEven(expressionCost, columnCount int) int {
	if !calibration.Enabled || columnCount < 0 || columnCount > 1 || expressionCost > calibration.BufferMaxUnits || calibration.BufferSavedNS <= 0 {
		return math.MaxInt
	}
	// Require twice the measured break-even before switching. This absorbs timer
	// noise and keeps short scans on the allocation-free existing callback loop.
	return int(math.Ceil(2 * calibration.BufferCompileNS / calibration.BufferSavedNS))
}

// MapReduceBufferProbeBreakEven bounds the one-time cost of compiling and
// comparing an uncalibrated reducer shape to one percent of the whole scan.
// Unlike MapReduceBufferBreakEven, this admits arbitrary callback arity and
// expression complexity: the actual reducer decides its own best path on its
// first representative buffer.
func (calibration JITCostCalibration) MapReduceBufferProbeBreakEven(expressionCost, columnCount, probeRows int) int {
	if !calibration.Enabled || expressionCost < 0 || columnCount < 0 || calibration.BufferCurrentNS <= 0 {
		return math.MaxInt
	}
	compileNS := calibration.BufferCompileNS
	if expressionCost > calibration.BufferMaxUnits {
		if calibration.BufferCompileUnitNS <= 0 {
			return math.MaxInt
		}
		compileNS += float64(expressionCost-calibration.BufferMaxUnits) * calibration.BufferCompileUnitNS
	}
	probeNS := float64(probeRows) * (calibration.BufferCurrentNS + calibration.BufferFusedNS)
	return int(math.Ceil(100 * (compileNS + probeNS) / calibration.BufferCurrentNS))
}

// FilterBufferBreakEven returns the number of rows needed to amortize a typed
// predicate loop. Startup calibration compares the same row-major buffer and
// compaction work with and without the fused loop; bulk column decoding is an
// additional storage-side benefit and is deliberately not needed to justify it.
func (calibration JITCostCalibration) FilterBufferBreakEven(expressionCost, columnCount int) int {
	if !calibration.Enabled || expressionCost < 0 || columnCount < 1 || calibration.FilterBufferSavedNS <= 0 {
		return math.MaxInt
	}
	compileNS := calibration.FilterBufferCompileNS
	if expressionCost > calibration.FilterBufferMaxUnits {
		if calibration.FilterBufferCompileUnitNS <= 0 {
			return math.MaxInt
		}
		compileNS += float64(expressionCost-calibration.FilterBufferMaxUnits) * calibration.FilterBufferCompileUnitNS
	}
	// Keep timer noise and unmeasured gather costs away from point queries.
	return int(math.Ceil(2 * compileNS / calibration.FilterBufferSavedNS))
}

type jitCalibrationShape struct {
	source  string
	argSets [][]Scmer
}

type jitCalibrationObservation struct {
	workUnits int
	emitNS    float64
	publishNS float64
	callNS    float64
}

var (
	jitCalibrationOnce sync.Once
	jitCalibrationData atomic.Pointer[JITCostCalibration]
	jitCalibrationSink Scmer
)

// CalibrateJITCosts runs once per process. It deliberately executes after the
// Go runtime and JIT arena are available but before Scheme libraries are
// imported, so import and query-plan compilation see the same immutable costs.
func CalibrateJITCosts() JITCostCalibration {
	jitCalibrationOnce.Do(func() {
		started := time.Now()
		calibration := JITCostCalibration{}
		if jitEnabled {
			observations := func() []jitCalibrationObservation {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				return measureJITCalibrationShapes()
			}()
			calibration = summarizeJITCalibration(observations)
			calibration.BufferCompileNS, calibration.BufferCurrentNS, calibration.BufferFusedNS, calibration.BufferMaxUnits = measureMapReduceBufferCalibration()
			calibration.BufferCompileUnitNS = measureMapReduceBufferCompileUnit(calibration.BufferCompileNS, calibration.BufferMaxUnits)
			calibration.BufferSavedNS = math.Max(0, calibration.BufferCurrentNS-calibration.BufferFusedNS)
			calibration.FilterBufferCompileNS, calibration.FilterBufferCurrentNS, calibration.FilterBufferFusedNS, calibration.FilterBufferMaxUnits = measureFilterBufferCalibration()
			calibration.FilterBufferCompileUnitNS = measureFilterBufferCompileUnit(calibration.FilterBufferCompileNS, calibration.FilterBufferMaxUnits)
			calibration.FilterBufferSavedNS = math.Max(0, calibration.FilterBufferCurrentNS-calibration.FilterBufferFusedNS)
			calibration.FilterBufferMinimumRows = calibration.FilterBufferBreakEven(0, 1)
		}
		calibration.CalibrationNS = time.Since(started).Nanoseconds()
		jitCalibrationData.Store(&calibration)
	})
	return CurrentJITCosts()
}

func measureFilterBufferCalibration() (compileNS, currentNS, fusedNS float64, expressionUnits int) {
	template := calibrationProcedure(`(lambda (value) (> value 127))`)
	compiled := CompileJIT(NewProcStruct(*cloneCalibrationProcedure(template)), true)
	proc := compiled.Proc()
	if proc == nil || proc.Compiled == nil {
		return 0, 0, 0, 0
	}
	expressionUnits = JITExpressionCost(proc.Body)
	compileSamples := make([]float64, jitCalibrationSamples)
	var fused JITFilterBufferFunc
	for sample := range compileSamples {
		started := time.Now()
		fused = CompileJITFilterBuffer(proc, []uint8{tagInt})
		compileSamples[sample] = float64(time.Since(started).Nanoseconds())
	}
	if fused == nil {
		return 0, 0, 0, 0
	}
	const rows = 16 * 1024
	values := make([]Scmer, rows)
	ids := make([]uint32, rows)
	for index := range values {
		values[index] = NewInt(int64(index & 255))
		ids[index] = uint32(index)
	}
	currentSamples := make([]float64, jitCalibrationSamples)
	fusedSamples := make([]float64, jitCalibrationSamples)
	args := []Scmer{NewInt(0)}
	for sample := range currentSamples {
		started := time.Now()
		out := 0
		for index, value := range values {
			args[0] = value
			if ToBool(proc.Compiled.Native(args...)) {
				ids[out] = uint32(index)
				out++
			}
		}
		currentSamples[sample] = float64(time.Since(started).Nanoseconds()) / rows
		jitCalibrationSink = NewInt(int64(out))

		for index := range ids {
			ids[index] = uint32(index)
		}
		started = time.Now()
		out = fused(ids, values)
		fusedSamples[sample] = float64(time.Since(started).Nanoseconds()) / rows
		jitCalibrationSink = NewInt(int64(out))
	}
	compileNS = medianFloat64(compileSamples)
	currentNS = medianFloat64(currentSamples)
	fusedNS = medianFloat64(fusedSamples)
	runtime.KeepAlive(compiled)
	runtime.KeepAlive(fused)
	return compileNS, currentNS, fusedNS, expressionUnits
}

func measureFilterBufferCompileUnit(simpleNS float64, simpleUnits int) float64 {
	template := calibrationProcedure("(lambda (a b c) (and (> (+ a (* b c)) 127) (< (- c a) 511)))")
	compiled := CompileJIT(NewProcStruct(*cloneCalibrationProcedure(template)), true)
	proc := compiled.Proc()
	if proc == nil || proc.Compiled == nil {
		return 0
	}
	units := JITExpressionCost(proc.Body)
	if units <= simpleUnits {
		return 0
	}
	samples := make([]float64, jitCalibrationSamples)
	for sample := range samples {
		started := time.Now()
		kernel := CompileJITFilterBuffer(proc, []uint8{tagInt, tagInt, tagInt})
		samples[sample] = float64(time.Since(started).Nanoseconds())
		if kernel == nil {
			return 0
		}
		runtime.KeepAlive(kernel)
	}
	return math.Max(0, (medianFloat64(samples)-simpleNS)/float64(units-simpleUnits))
}

func measureMapReduceBufferCalibration() (compileNS, currentNS, fusedNS float64, expressionUnits int) {
	template := calibrationProcedure(`(lambda (acc value) (sql_sum_reduce acc value))`)
	compiled := CompileJIT(NewProcStruct(*cloneCalibrationProcedure(template)), true)
	proc := compiled.Proc()
	if proc == nil || proc.Compiled == nil {
		return 0, 0, 0, 0
	}
	expressionUnits = JITExpressionCost(proc.Body)
	compileSamples := make([]float64, jitCalibrationSamples)
	var fused JITMapReduceBufferFunc
	for sample := range compileSamples {
		started := time.Now()
		fused = CompileJITMapReduceBuffer(proc, []uint8{tagInt})
		compileSamples[sample] = float64(time.Since(started).Nanoseconds())
	}
	if fused == nil {
		return 0, 0, 0, 0
	}
	values := make([]Scmer, 16*1024)
	for index := range values {
		values[index] = NewInt(int64(index & 255))
	}
	currentSamples := make([]float64, jitCalibrationSamples)
	fusedSamples := make([]float64, jitCalibrationSamples)
	args := []Scmer{NewInt(0), NewInt(0)}
	for sample := range currentSamples {
		started := time.Now()
		acc := NewInt(0)
		for _, value := range values {
			args[0], args[1] = acc, value
			acc = proc.Compiled.Native(args...)
		}
		currentSamples[sample] = float64(time.Since(started).Nanoseconds()) / float64(len(values))
		jitCalibrationSink = acc

		started = time.Now()
		acc = fused(NewInt(0), values, len(values))
		fusedSamples[sample] = float64(time.Since(started).Nanoseconds()) / float64(len(values))
		jitCalibrationSink = acc
	}
	compileNS = medianFloat64(compileSamples)
	currentNS = medianFloat64(currentSamples)
	fusedNS = medianFloat64(fusedSamples)
	runtime.KeepAlive(compiled)
	runtime.KeepAlive(fused)
	return compileNS, currentNS, fusedNS, expressionUnits
}

func measureMapReduceBufferCompileUnit(simpleNS float64, simpleUnits int) float64 {
	template := calibrationProcedure("(lambda (acc a b c) (+ acc (+ a (* b c))))")
	compiled := CompileJIT(NewProcStruct(*cloneCalibrationProcedure(template)), true)
	proc := compiled.Proc()
	if proc == nil || proc.Compiled == nil {
		return 0
	}
	units := JITExpressionCost(proc.Body)
	if units <= simpleUnits {
		return 0
	}
	samples := make([]float64, jitCalibrationSamples)
	for sample := range samples {
		started := time.Now()
		kernel := CompileJITMapReduceBuffer(proc, []uint8{tagInt, tagInt, tagInt})
		samples[sample] = float64(time.Since(started).Nanoseconds())
		if kernel == nil {
			return 0
		}
		runtime.KeepAlive(kernel)
	}
	return math.Max(0, (medianFloat64(samples)-simpleNS)/float64(units-simpleUnits))
}

// CurrentJITCosts returns a value copy so callers cannot mutate the globally
// published calibration. Embedders which did not call CalibrateJITCosts during
// startup trigger the same once-only initialization on first use.
func CurrentJITCosts() JITCostCalibration {
	if calibration := jitCalibrationData.Load(); calibration != nil {
		return *calibration
	}
	return CalibrateJITCosts()
}

func measureJITCalibrationShapes() []jitCalibrationObservation {
	shapes := jitCalibrationShapes()
	// JITContext keeps both the current and one-past-the-end code pointers.
	// Therefore the scratch buffer must not be a Go heap object: a concurrent GC
	// is allowed to inspect the context while emission is running and correctly
	// rejects its one-past pointer as an invalid heap pointer. The private mapping
	// is never executed or published and is deterministically released below.
	codeBuffer, err := syscall.Mmap(-1, 0, jitCalibrationBufferSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		panic("jit calibration: scratch mmap failed: " + err.Error())
	}
	defer func() {
		if err := syscall.Munmap(codeBuffer); err != nil {
			panic("jit calibration: scratch munmap failed: " + err.Error())
		}
	}()
	observations := make([]jitCalibrationObservation, 0, len(shapes))
	for _, shape := range shapes {
		template := calibrationProcedure(shape.source)
		workUnits := JITExpressionCost(template.Body)
		emitNS, _ := measureJITCalibrationEmit(template, codeBuffer)

		procedure := cloneCalibrationProcedure(template)
		before := time.Now()
		compiled := CompileJIT(NewProcStruct(*procedure), true)
		publishedNS := float64(time.Since(before).Nanoseconds())
		if compiled.Proc() == nil || compiled.Proc().Compiled == nil || compiled.Proc().Compiled.Native == nil {
			panic("jit calibration: fixed predicate did not publish")
		}
		callNS := measureJITCalibrationCall(compiled.Proc().Compiled.Native, shape.argSets)
		observations = append(observations, jitCalibrationObservation{
			workUnits: workUnits,
			emitNS:    emitNS,
			publishNS: math.Max(0, publishedNS-emitNS),
			callNS:    callNS,
		})
		runtime.KeepAlive(compiled)
	}
	return observations
}

func measureJITCalibrationEmit(template *Proc, codeBuffer []byte) (float64, int) {
	iterations := 1
	codeLen := 0
	for {
		elapsed, measuredLen := runJITCalibrationEmits(template, codeBuffer, iterations)
		codeLen = measuredLen
		if elapsed >= jitCalibrationMinimumRound || iterations >= 1<<16 {
			break
		}
		iterations *= 2
	}
	samples := make([]float64, jitCalibrationSamples)
	for sample := range samples {
		elapsed, measuredLen := runJITCalibrationEmits(template, codeBuffer, iterations)
		if measuredLen != codeLen {
			panic("jit calibration: predicate emission is not deterministic")
		}
		samples[sample] = float64(elapsed.Nanoseconds()) / float64(iterations)
	}
	return medianFloat64(samples), codeLen
}

func runJITCalibrationEmits(template *Proc, codeBuffer []byte, iterations int) (time.Duration, int) {
	codeLen := 0
	started := time.Now()
	for range iterations {
		procedure := cloneCalibrationProcedure(template)
		buf := &execBuf{
			ptr: unsafe.Pointer(&codeBuffer[0]),
			n:   len(codeBuffer),
		}
		measuredLen, _, _, overflow, _, _, _ := jitCompileProcToExec(procedure, buf, true)
		if overflow || measuredLen == 0 {
			panic("jit calibration: fixed predicate did not emit")
		}
		codeLen = measuredLen
	}
	return time.Since(started), codeLen
}

// JITExpressionCost returns an architecture-neutral estimate for an expression.
// Generated declarations contribute jitgen's offline SSA cost. Handwritten
// emitters and special forms contribute one structural unit because zero in a
// Declaration means "no generated cost metadata", not "free at runtime".
//
// Keeping this walker independent of concrete operator names is important: a
// newly generated builtin automatically participates in startup calibration and
// scan-fusion decisions through its Declaration metadata.
func JITExpressionCost(expression Scmer) int {
	for expression.GetTag() == tagSourceInfo {
		expression = expression.SourceInfo().value
	}
	if expression.GetTag() == tagProc && expression.Proc() != nil {
		return JITExpressionCost(expression.Proc().Body)
	}
	if expression.GetTag() != tagSlice {
		return 0
	}
	list := expression.Slice()
	if len(list) == 0 {
		return 0
	}
	cost := 1
	declaration := DeclarationForValue(list[0])
	if declaration != nil && declaration.Type != nil {
		cost = int(declaration.Type.JITInlineCost)
		if cost == 0 {
			cost = 1
		}
	}
	arguments := list[1:]
	if declaration != nil {
		switch declaration.SyntaxKind {
		case SyntaxQuote:
			return cost
		case SyntaxLambda:
			// Lambda parameter lists are syntax data. Its body is emitted when the
			// callback itself is inlined, so it belongs to the estimate.
			if len(arguments) != 0 {
				arguments = arguments[1:]
			}
		}
	}
	for _, argument := range arguments {
		cost += JITExpressionCost(argument)
	}
	return cost
}

func calibrationProcedure(source string) *Proc {
	parsed := Read("jit startup calibration", source)
	optimized := Optimize(parsed, &Globalenv, nil)
	procedure := Eval(optimized, &Globalenv)
	if procedure.Proc() == nil {
		panic("jit calibration: optimized expression is not a procedure")
	}
	return procedure.Proc()
}

func cloneCalibrationProcedure(template *Proc) *Proc {
	copy := *template
	copy.JITCode = 0
	copy.Compiled = nil
	copy.jitCompiling = 0
	return &copy
}

func measureJITCalibrationCall(function func(...Scmer) Scmer, argSets [][]Scmer) float64 {
	iterations := 256
	for {
		elapsed := runJITCalibrationCalls(function, argSets, iterations)
		if elapsed >= jitCalibrationMinimumRound || iterations >= 1<<24 {
			break
		}
		iterations *= 2
	}
	samples := make([]float64, jitCalibrationSamples)
	for sample := range samples {
		elapsed := runJITCalibrationCalls(function, argSets, iterations)
		samples[sample] = float64(elapsed.Nanoseconds()) / float64(iterations)
	}
	return medianFloat64(samples)
}

func runJITCalibrationCalls(function func(...Scmer) Scmer, argSets [][]Scmer, iterations int) time.Duration {
	mask := len(argSets) - 1
	started := time.Now()
	for iteration := 0; iteration < iterations; iteration++ {
		jitCalibrationSink = function(argSets[iteration&mask]...)
	}
	return time.Since(started)
}

func summarizeJITCalibration(observations []jitCalibrationObservation) JITCostCalibration {
	if len(observations) == 0 {
		return JITCostCalibration{}
	}
	emitX := make([]float64, len(observations))
	emitY := make([]float64, len(observations))
	callX := make([]float64, len(observations))
	callY := make([]float64, len(observations))
	publish := make([]float64, len(observations))
	for index, observation := range observations {
		emitX[index] = float64(observation.workUnits)
		emitY[index] = observation.emitNS
		callX[index] = float64(observation.workUnits)
		callY[index] = observation.callNS
		publish[index] = observation.publishNS
	}
	emitFixed, emitUnit := robustLinearFit(emitX, emitY)
	callFixed := calibrationCallBoundary(observations)
	predicateUnit := calibrationPredicateUnit(observations, callFixed)
	return JITCostCalibration{
		Enabled:         true,
		ShapeCount:      len(observations),
		SamplesPerShape: jitCalibrationSamples,
		EmitFixedNS:     emitFixed,
		EmitUnitNS:      emitUnit,
		PublishFixedNS:  medianFloat64(publish),
		DirectCallNS:    callFixed,
		PredicateUnitNS: predicateUnit,
	}
}

// The zero-work lambdas anchor the Proc-call boundary directly. Fitting the
// intercept from all predicates made a rare slow complex predicate look like a
// more expensive call boundary, which is precisely the kind of edge bias the
// calibration must avoid.
func calibrationCallBoundary(observations []jitCalibrationObservation) float64 {
	boundaries := make([]float64, 0, len(observations))
	for _, observation := range observations {
		if observation.workUnits == 0 {
			boundaries = append(boundaries, observation.callNS)
		}
	}
	if len(boundaries) == 0 {
		return 0
	}
	return medianFloat64(boundaries)
}

func calibrationPredicateUnit(observations []jitCalibrationObservation, callBoundary float64) float64 {
	units := make([]float64, 0, len(observations))
	for _, observation := range observations {
		if observation.workUnits > 0 {
			units = append(units, (observation.callNS-callBoundary)/float64(observation.workUnits))
		}
	}
	if len(units) == 0 {
		return 0
	}
	return math.Max(0, medianFloat64(units))
}

// robustLinearFit uses the Theil-Sen median slope. Startup timing occasionally
// contains an interrupt or scheduler outlier; pairwise median slopes retain the
// complete expression set without allowing one delayed sample to move every
// query's JIT threshold.
func robustLinearFit(x, y []float64) (intercept, slope float64) {
	if len(x) != len(y) || len(x) == 0 {
		return 0, 0
	}
	slopes := make([]float64, 0, len(x)*(len(x)-1)/2)
	for left := 0; left < len(x); left++ {
		for right := left + 1; right < len(x); right++ {
			if x[left] == x[right] {
				continue
			}
			slopes = append(slopes, (y[right]-y[left])/(x[right]-x[left]))
		}
	}
	if len(slopes) != 0 {
		slope = math.Max(0, medianFloat64(slopes))
	}
	intercepts := make([]float64, len(x))
	for index := range x {
		intercepts[index] = y[index] - slope*x[index]
	}
	return math.Max(0, medianFloat64(intercepts)), slope
}

func medianFloat64(values []float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 == 0 {
		return (ordered[middle-1] + ordered[middle]) / 2
	}
	return ordered[middle]
}

func jitCalibrationShapes() []jitCalibrationShape {
	values := func(arity int) [][]Scmer {
		sets := make([][]Scmer, 64)
		for row := range sets {
			sets[row] = make([]Scmer, arity)
			for column := range sets[row] {
				sets[row][column] = NewInt(int64((row*17 + column*29) % 101))
			}
		}
		return sets
	}
	strings := func() [][]Scmer {
		words := [...]string{"", "a", "alpha", "beta", "delta", "memcp", "mysql", "wordpress"}
		sets := make([][]Scmer, 64)
		for row := range sets {
			sets[row] = []Scmer{NewString(words[(row*5)%len(words)])}
		}
		return sets
	}
	return []jitCalibrationShape{
		{`(lambda (value) true)`, values(1)},
		{`(lambda (value) false)`, values(1)},
		{`(lambda (value) (> value 50))`, values(1)},
		{`(lambda (value) (< value 50))`, values(1)},
		{`(lambda (value) (equal?? value 50))`, values(1)},
		{`(lambda (value) (and (>= value 20) (< value 80)))`, values(1)},
		{`(lambda (value) (or (< value 20) (> value 80)))`, values(1)},
		{`(lambda (left right) (and (> left 20) (< right 80)))`, values(2)},
		{`(lambda (value) (equal?? (+ value 3) 45))`, values(1)},
		{`(lambda (value) (and (> value 10) (< value 90) (not (equal?? value 50)) (> (+ value 2) 12)))`, values(1)},
		{`(lambda (a b c) (and (> a 10) (< b 90) (equal?? (+ a b) c)))`, values(3)},
		{`(lambda (value) (equal?? value "memcp"))`, strings()},
		{`(lambda (value) (not (equal?? value "")))`, strings()},
		{`(lambda (value) (or (equal?? value "mysql") (equal?? value "wordpress") (equal?? value "memcp")))`, strings()},
		{`(lambda (value) (and (> (* value 3) 30) (< (+ value 7) 90)))`, values(1)},
		{`(lambda (a b) (or (and (> a 10) (< b 40)) (and (> b 70) (< a 90))))`, values(2)},
	}
}
