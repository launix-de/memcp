/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
package storage

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/dc0d/onexit"
	"github.com/launix-de/memcp/scm"
)

type SettingsT struct {
	Backtrace              bool
	Trace                  bool
	TracePrint             bool
	PartitionMaxDimensions int
	DefaultEngine          string
	ShardSize              uint
	AnalyzeMinItems        int
	MaxRamPercent          int   // 0 = default (50%), otherwise 1-100; total memory budget
	MaxRamBytes            int64 // 0 = use MaxRamPercent; >0 = override total budget in bytes
	MaxPersistPercent      int   // 0 = default (30%), otherwise 1-100; budget for persisted shards+indexes
	MaxPersistBytes        int64 // 0 = use MaxPersistPercent; >0 = override persisted budget in bytes
	MetricsTracing         bool  // when true, periodically insert metrics into system.perf_metrics
	MetricsTracingInterval int   // interval in seconds (0 = default 60s)
	ShutdownDrainSeconds   int   // seconds to wait for in-flight requests during shutdown (0 = default 10s)
}

var Settings SettingsT = SettingsT{false, false, false, 10, "safe", 60000, 50, 0, 0, 0, 0, false, 0, 0}

// call this after you filled Settings
func InitSettings() {
	scm.SettingsHaveGoodBacktraces = Settings.Backtrace
	scm.SetTrace(Settings.Trace)
	scm.TracePrint = Settings.TracePrint
	onexit.Register(func() { scm.SetTrace(false) }) // close trace file on exit
	InitCacheManager()
}

func ChangeSettings(a ...scm.Scmer) scm.Scmer {
	// schema, filename
	if len(a) == 0 {
		return scm.NewSlice([]scm.Scmer{
			scm.NewString("Backtrace"), scm.NewBool(Settings.Backtrace),
			scm.NewString("Trace"), scm.NewBool(Settings.Trace),
			scm.NewString("TracePrint"), scm.NewBool(Settings.TracePrint),
			scm.NewString("PartitionMaxDimensions"), scm.NewInt(int64(Settings.PartitionMaxDimensions)),
			scm.NewString("DefaultEngine"), scm.NewString(Settings.DefaultEngine),
			scm.NewString("ShardSize"), scm.NewInt(int64(Settings.ShardSize)),
			scm.NewString("AnalyzeMinItems"), scm.NewInt(int64(Settings.AnalyzeMinItems)),
			scm.NewString("MaxRamPercent"), scm.NewInt(int64(Settings.MaxRamPercent)),
			scm.NewString("MaxRamBytes"), scm.NewInt(Settings.MaxRamBytes),
			scm.NewString("MaxPersistPercent"), scm.NewInt(int64(Settings.MaxPersistPercent)),
			scm.NewString("MaxPersistBytes"), scm.NewInt(Settings.MaxPersistBytes),
			scm.NewString("MetricsTracing"), scm.NewBool(Settings.MetricsTracing),
			scm.NewString("MetricsTracingInterval"), scm.NewInt(int64(Settings.MetricsTracingInterval)),
			scm.NewString("ShutdownDrainSeconds"), scm.NewInt(int64(Settings.ShutdownDrainSeconds)),
		})
	} else if len(a) == 1 {
		switch scm.String(a[0]) {
		case "Backtrace":
			return scm.NewBool(Settings.Backtrace)
		case "Trace":
			return scm.NewBool(Settings.Trace)
		case "TracePrint":
			return scm.NewBool(Settings.TracePrint)
		case "PartitionMaxDimensions":
			return scm.NewInt(int64(Settings.PartitionMaxDimensions))
		case "DefaultEngine":
			return scm.NewString(Settings.DefaultEngine)
		case "ShardSize":
			return scm.NewInt(int64(Settings.ShardSize))
		case "AnalyzeMinItems":
			return scm.NewInt(int64(Settings.AnalyzeMinItems))
		case "MaxRamPercent":
			return scm.NewInt(int64(Settings.MaxRamPercent))
		case "MaxRamBytes":
			return scm.NewInt(Settings.MaxRamBytes)
		case "MaxPersistPercent":
			return scm.NewInt(int64(Settings.MaxPersistPercent))
		case "MaxPersistBytes":
			return scm.NewInt(Settings.MaxPersistBytes)
		case "MetricsTracing":
			return scm.NewBool(Settings.MetricsTracing)
		case "MetricsTracingInterval":
			return scm.NewInt(int64(Settings.MetricsTracingInterval))
		case "ShutdownDrainSeconds":
			return scm.NewInt(int64(Settings.ShutdownDrainSeconds))
		default:
			panic("unknown setting: " + scm.String(a[0]))
		}
	} else {
		switch scm.String(a[0]) {
		case "Backtrace":
			scm.SettingsHaveGoodBacktraces = Settings.Backtrace
			Settings.Backtrace = scm.ToBool(a[1])
		case "Trace":
			Settings.Trace = scm.ToBool(a[1])
			scm.SetTrace(Settings.Trace)
		case "TracePrint":
			Settings.TracePrint = scm.ToBool(a[1])
			scm.TracePrint = Settings.TracePrint
		case "PartitionMaxDimensions":
			Settings.PartitionMaxDimensions = scm.ToInt(a[1])
		case "DefaultEngine":
			Settings.DefaultEngine = scm.String(a[1])
		case "ShardSize":
			Settings.ShardSize = uint(scm.ToInt(a[1]))
		case "AnalyzeMinItems":
			Settings.AnalyzeMinItems = scm.ToInt(a[1])
		case "MaxRamPercent":
			Settings.MaxRamPercent = scm.ToInt(a[1])
			total, persisted := computeMemoryBudgets()
			GlobalCache.UpdateBudget(total, persisted)
		case "MaxRamBytes":
			Settings.MaxRamBytes = int64(scm.ToInt(a[1]))
			total, persisted := computeMemoryBudgets()
			GlobalCache.UpdateBudget(total, persisted)
		case "MaxPersistPercent":
			Settings.MaxPersistPercent = scm.ToInt(a[1])
			total, persisted := computeMemoryBudgets()
			GlobalCache.UpdateBudget(total, persisted)
		case "MaxPersistBytes":
			Settings.MaxPersistBytes = int64(scm.ToInt(a[1]))
			total, persisted := computeMemoryBudgets()
			GlobalCache.UpdateBudget(total, persisted)
		case "MetricsTracing":
			Settings.MetricsTracing = scm.ToBool(a[1])
		case "MetricsTracingInterval":
			Settings.MetricsTracingInterval = scm.ToInt(a[1])
		case "ShutdownDrainSeconds":
			Settings.ShutdownDrainSeconds = scm.ToInt(a[1])
		default:
			panic("unknown setting: " + scm.String(a[0]))
		}
		return scm.NewBool(true)
	}
}

// GlobalCache is the singleton CacheManager for memory pressure management.
// Always exists as a struct; methods are no-ops until Init() is called.
var GlobalCache CacheManager

// totalMemoryBytes reads total physical RAM from /proc/meminfo (Linux).
func totalMemoryBytes() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
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
					return kb * 1024
				}
			}
			break
		}
	}
	return 0
}

func computeMemoryBudgets() (total, persisted int64) {
	totalRAM := totalMemoryBytes()

	// total budget
	if Settings.MaxRamBytes > 0 {
		total = Settings.MaxRamBytes
	} else if totalRAM > 0 {
		pct := Settings.MaxRamPercent
		if pct == 0 {
			pct = 50
		}
		total = totalRAM * int64(pct) / 100
	}

	// persisted budget (shards + indexes)
	if Settings.MaxPersistBytes > 0 {
		persisted = Settings.MaxPersistBytes
	} else if totalRAM > 0 {
		pct := Settings.MaxPersistPercent
		if pct == 0 {
			pct = 30
		}
		persisted = totalRAM * int64(pct) / 100
	}

	return
}

// InitCacheManager initializes the global CacheManager. Call from InitSettings().
func InitCacheManager() {
	total, persisted := computeMemoryBudgets()
	if total > 0 || persisted > 0 {
		GlobalCache.Init(total, persisted)
	}
}
