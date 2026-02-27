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
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// ActiveHTTPConnections tracks the current number of active HTTP connections.
// Incremented/decremented via ConnState callback — single atomic, no mutex.
var ActiveHTTPConnections int64

// TotalHTTPRequests is atomically incremented on each new HTTP request.
// The background sampler reads this to compute requests/sec without any hot-path mutex.
var TotalHTTPRequests int64

// metricsSnapshot holds all sampled values, atomically swapped by the background goroutine.
// Readers load the pointer atomically — zero contention on the hot path.
type metricsSnapshot struct {
	cpuUsage     float64 // 0–100
	rps          float64 // requests per second (averaged over last 10s)
	maxConn10min int64   // max active connections over last 10 minutes
}

var currentSnapshot unsafe.Pointer // *metricsSnapshot

func loadSnapshot() *metricsSnapshot {
	p := atomic.LoadPointer(&currentSnapshot)
	if p == nil {
		return &metricsSnapshot{}
	}
	return (*metricsSnapshot)(p)
}

// initMetricsSampler starts a single background goroutine that samples all metrics.
// CPU is read from /proc/stat, RPS from atomic counter delta, max-connections from atomic load.
func initMetricsSampler() {
	snap := &metricsSnapshot{maxConn10min: 1}
	atomic.StorePointer(&currentSnapshot, unsafe.Pointer(snap))

	go func() {
		var prevIdle, prevTotal uint64
		var prevRequests int64

		// circular buffer: 10 one-second RPS samples
		const rpsBuckets = 10
		rpsBuf := [rpsBuckets]float64{}
		rpsIdx := 0

		// circular buffer: 600 one-second max-connection samples (10 min)
		const connBuckets = 600
		connBuf := [connBuckets]int64{}
		connIdx := 0

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// CPU usage from /proc/stat delta
			cpuVal := float64(0)
			idle, total := readCPUStat()
			if prevTotal > 0 && total > prevTotal {
				deltaIdle := idle - prevIdle
				deltaTotal := total - prevTotal
				cpuVal = (1.0 - float64(deltaIdle)/float64(deltaTotal)) * 100.0
			}
			prevIdle = idle
			prevTotal = total

			// RPS: delta of atomic counter
			curRequests := atomic.LoadInt64(&TotalHTTPRequests)
			delta := curRequests - prevRequests
			prevRequests = curRequests
			rpsBuf[rpsIdx%rpsBuckets] = float64(delta)
			rpsIdx++
			rpsSum := float64(0)
			rpsCount := rpsBuckets
			if rpsIdx < rpsBuckets {
				rpsCount = rpsIdx
			}
			for i := 0; i < rpsCount; i++ {
				rpsSum += rpsBuf[i]
			}
			rpsVal := rpsSum / float64(rpsCount)

			// Max connections: sample current and keep 10-min window
			curConn := atomic.LoadInt64(&ActiveHTTPConnections)
			connBuf[connIdx%connBuckets] = curConn
			connIdx++
			maxConn := curConn
			maxCount := connBuckets
			if connIdx < connBuckets {
				maxCount = connIdx
			}
			for i := 0; i < maxCount; i++ {
				if connBuf[i] > maxConn {
					maxConn = connBuf[i]
				}
			}
			if maxConn < 1 {
				maxConn = 1
			}

			// atomically publish new snapshot
			newSnap := &metricsSnapshot{
				cpuUsage:     cpuVal,
				rps:          math.Round(rpsVal*10) / 10,
				maxConn10min: maxConn,
			}
			atomic.StorePointer(&currentSnapshot, unsafe.Pointer(newSnap))
		}
	}()
}

// readCPUStat reads the first cpu line from /proc/stat and returns (idle, total).
func readCPUStat() (uint64, uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0, 0
			}
			var total uint64
			var idle uint64
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 {
					idle = val
				}
			}
			return idle, total
		}
	}
	return 0, 0
}

// readMemInfo reads MemTotal and MemAvailable from /proc/meminfo.
func readMemInfo() (memTotal, memAvailable int64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					memTotal = kb * 1024
				}
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					memAvailable = kb * 1024
				}
			}
		}
		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}
	return
}

// readProcessRSS reads the RSS (resident set size) of this process from /proc/self/statm.
func readProcessRSS() int64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return pages * int64(os.Getpagesize())
}

func init() {
	initMetricsSampler()
}

// InitMetricsDeclarations registers dashboard metric Scheme functions.
// Must be called after Globalenv is initialized (from scm.go init).
func InitMetricsDeclarations() {
	DeclareTitle("Dashboard Metrics")

	Declare(&Globalenv, &Declaration{
		"cpu_usage", "Returns current CPU usage as a percentage (0-100)",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(loadSnapshot().cpuUsage)
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"available_memory", "Returns available memory in bytes from /proc/meminfo",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			_, avail := readMemInfo()
			return NewInt(avail)
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"total_memory", "Returns total physical memory in bytes from /proc/meminfo",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			total, _ := readMemInfo()
			return NewInt(total)
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"process_memory", "Returns the RSS (resident set size) of this process in bytes",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			return NewInt(readProcessRSS())
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"active_connections", "Returns the current number of active HTTP connections",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			return NewInt(atomic.LoadInt64(&ActiveHTTPConnections))
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"max_connections", "Returns the maximum number of HTTP connections over the last 10 minutes",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			return NewInt(loadSnapshot().maxConn10min)
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"requests_per_second", "Returns the average number of HTTP requests per second over the last 10 seconds",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(loadSnapshot().rps)
		}, false, false, nil,
	})

	Declare(&Globalenv, &Declaration{
		"readfile", "Reads a file from the working directory and returns its contents as a string",
		1, 1,
		[]DeclarationParameter{
			{"filename", "string", "path to the file to read", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			filename := String(a[0])
			data, err := os.ReadFile(filename)
			if err != nil {
				panic("readfile: " + err.Error())
			}
			return NewString(string(data))
		}, false, false, nil,
	})
}
