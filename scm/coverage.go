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
	"strings"
	"sync"
	"sync/atomic"
)

type sourceCoverageLocation struct {
	source string
	line   int
	col    int
}

// Permanent coverage facts own coordinates and an execution flag only. They
// must never point back to a SourceInfo, its AST, or any runtime environment.
type sourceCoveragePoint struct {
	sourceCoverageLocation
	interpreted atomic.Uint32
}

func (point *sourceCoveragePoint) wasInterpreted() bool {
	return point.interpreted.Load() != 0
}

var (
	sourceCoverageMu     sync.RWMutex
	sourceCoveragePoints = make(map[sourceCoverageLocation]*sourceCoveragePoint)
)

func sourceCoveragePointFor(source string, line, col int) *sourceCoveragePoint {
	location := sourceCoverageLocation{source: source, line: line, col: col}
	sourceCoverageMu.Lock()
	defer sourceCoverageMu.Unlock()
	if point := sourceCoveragePoints[location]; point != nil {
		return point
	}
	// A caller may lend a substring of a request buffer as its source label.
	// Retain the coordinate text itself, not that buffer's backing allocation.
	location.source = strings.Clone(source)
	point := &sourceCoveragePoint{sourceCoverageLocation: location}
	sourceCoveragePoints[location] = point
	return point
}

func sourceCoverageSnapshot() []*sourceCoveragePoint {
	sourceCoverageMu.RLock()
	defer sourceCoverageMu.RUnlock()
	points := make([]*sourceCoveragePoint, 0, len(sourceCoveragePoints))
	for _, point := range sourceCoveragePoints {
		points = append(points, point)
	}
	return points
}

func sourceCoverageMatchesPrefix(source string, prefix string) bool {
	if prefix == "" || strings.HasPrefix(source, prefix) {
		return true
	}
	if strings.HasSuffix(prefix, "/") {
		return strings.Contains(source, "/"+prefix)
	}
	return false
}

func sourceCoverageReport(a ...Scmer) Scmer {
	prefix := ""
	if len(a) > 0 {
		prefix = String(a[0])
	}

	type fileStats struct {
		total        int
		covered      int
		coveredLines map[int]bool
		allLines     map[int]bool
	}

	stats := map[string]*fileStats{}
	for _, point := range sourceCoverageSnapshot() {
		if point.source == "" || !sourceCoverageMatchesPrefix(point.source, prefix) {
			continue
		}
		fs := stats[point.source]
		if fs == nil {
			fs = &fileStats{
				coveredLines: map[int]bool{},
				allLines:     map[int]bool{},
			}
			stats[point.source] = fs
		}
		fs.total++
		fs.allLines[point.line] = true
		if point.wasInterpreted() {
			fs.covered++
			fs.coveredLines[point.line] = true
		}
	}

	total := 0
	covered := 0
	fileRows := make([]Scmer, 0, len(stats))
	for source, fs := range stats {
		total += fs.total
		covered += fs.covered

		uncoveredLines := make([]Scmer, 0)
		for line := range fs.allLines {
			if !fs.coveredLines[line] {
				uncoveredLines = append(uncoveredLines, NewInt(int64(line)))
			}
		}

		percent := 100.0
		if fs.total > 0 {
			percent = float64(fs.covered) * 100.0 / float64(fs.total)
		}
		fileRows = append(fileRows, NewSlice([]Scmer{
			NewString("source"), NewString(source),
			NewString("total"), NewInt(int64(fs.total)),
			NewString("covered"), NewInt(int64(fs.covered)),
			NewString("percent"), NewFloat(percent),
			NewString("uncovered_lines"), NewSlice(uncoveredLines),
		}))
	}

	percent := 100.0
	if total > 0 {
		percent = float64(covered) * 100.0 / float64(total)
	}
	return NewSlice([]Scmer{
		NewString("total"), NewInt(int64(total)),
		NewString("covered"), NewInt(int64(covered)),
		NewString("percent"), NewFloat(percent),
		NewString("files"), NewSlice(fileRows),
	})
}
