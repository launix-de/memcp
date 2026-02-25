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
	AIEstimator            bool
	MaxRamPercent          int // 0 = default (90%), otherwise 1-100
}

var Settings SettingsT = SettingsT{false, false, false, 10, "safe", 60000, 50, false, 0}

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
			scm.NewString("AIEstimator"), scm.NewBool(Settings.AIEstimator),
			scm.NewString("MaxRamPercent"), scm.NewInt(int64(Settings.MaxRamPercent)),
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
		case "AIEstimator":
			return scm.NewBool(Settings.AIEstimator)
		case "MaxRamPercent":
			return scm.NewInt(int64(Settings.MaxRamPercent))
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
			GlobalCache.UpdateBudget(computeMemoryBudget())
		case "AIEstimator":
			prev := Settings.AIEstimator
			Settings.AIEstimator = scm.ToBool(a[1])
			if prev != Settings.AIEstimator {
				// start/stop estimator on change
				if Settings.AIEstimator {
					StartGlobalEstimator()
				} else {
					StopGlobalEstimator()
				}
			} else if Settings.AIEstimator {
				// Setting already true; if estimator not running, try to (re)start
				globalEstimatorMu.Lock()
				est := globalEstimator
				globalEstimatorMu.Unlock()
				if est == nil {
					StartGlobalEstimator()
				}
			}
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

func computeMemoryBudget() int64 {
	totalRAM := totalMemoryBytes()
	if totalRAM <= 0 {
		return 0
	}
	pct := Settings.MaxRamPercent
	if pct == 0 {
		pct = 90
	}
	return totalRAM * int64(pct) / 100
}

// InitCacheManager initializes the global CacheManager. Call from InitSettings().
func InitCacheManager() {
	budget := computeMemoryBudget()
	if budget > 0 {
		GlobalCache.Init(budget)
	}
}
