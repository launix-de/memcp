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

import "testing"

func sourceCoverageCounts(source string) (covered, total int) {
	type point struct {
		line int
		col  int
	}
	points := make(map[point]bool)
	for _, info := range sourceCoverageInfos {
		if info != nil && info.source == source {
			key := point{line: info.line, col: info.col}
			points[key] = points[key] || info.wasInterpreted()
		}
	}
	for _, executed := range points {
		total++
		if executed {
			covered++
		}
	}
	return covered, total
}

func TestSourceCoverageTracksInterpreterExecutionOnly(t *testing.T) {
	env := newOptimizerTestEnv()
	optimizedSource := "coverage-optimizer-only.scm"
	Optimize(Read(optimizedSource, `(if true (+ 1 2) (+ 3 4))`), env, nil)
	if covered, total := sourceCoverageCounts(optimizedSource); total != 3 || covered != 0 {
		t.Fatalf("optimizer marked source coverage: covered=%d total=%d", covered, total)
	}

	runtimeSource := "coverage-interpreter.scm"
	Eval(Read(runtimeSource, `(if true (+ 1 2) (+ 3 4))`), env)
	if covered, total := sourceCoverageCounts(runtimeSource); total != 3 || covered != 2 {
		t.Fatalf("interpreter coverage = %d/%d, want selected branch plus root = 2/3", covered, total)
	}

	previous := SettingsTrackSourceCoverage
	SettingsTrackSourceCoverage = true
	defer func() { SettingsTrackSourceCoverage = previous }()
	preservedSource := "coverage-optimized-runtime.scm"
	optimized := Optimize(Read(preservedSource, `(lambda (x) (+ x 1))`), env, nil)
	if covered, total := sourceCoverageCounts(preservedSource); total != 3 || covered != 0 {
		t.Fatalf("coverage-preserving optimizer marked source coverage: covered=%d total=%d", covered, total)
	}
	proc := Eval(optimized, env)
	Apply(proc, NewInt(2))
	if covered, total := sourceCoverageCounts(preservedSource); total != 3 || covered != 2 {
		t.Fatalf("optimized interpreter coverage = %d/%d, want lambda and body but not parameter syntax = 2/3", covered, total)
	}
}
