/*
Copyright (C) 2024  Carl-Philip Hänsch

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

import "github.com/dc0d/onexit"
import "github.com/launix-de/memcp/scm"

type SettingsT struct {
	Backtrace bool
	Trace bool
	PartitionMaxDimensions int
	DefaultEngine string
	ShardSize uint
}

var Settings SettingsT = SettingsT{false, false, 10, "safe", 60000}

// call this after you filled Settings
func InitSettings() {
	scm.SettingsHaveGoodBacktraces = Settings.Backtrace
	scm.SetTrace(Settings.Trace)
	onexit.Register(func() { scm.SetTrace(false) }) // close trace file on exit
}

func ChangeSettings(a ...scm.Scmer) scm.Scmer {
	// schema, filename
	if len(a) == 1 {
		switch scm.String(a[0]) {
			case "Backtrace":
				return Settings.Backtrace
			case "Trace":
				return Settings.Trace
			case "PartitionMaxDimensions":
				return int64(Settings.PartitionMaxDimensions)
			case "DefaultEngine":
				return Settings.DefaultEngine
			case "ShardSize":
				return int64(Settings.ShardSize)
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
			case "PartitionMaxDimensions":
				Settings.PartitionMaxDimensions = scm.ToInt(a[1])
			case "DefaultEngine":
				Settings.DefaultEngine = scm.String(a[1])
			case "ShardSize":
				Settings.ShardSize = uint(scm.ToInt(a[1]))
			default:
				panic("unknown setting: " + scm.String(a[0]))
		}
		return true
	}
}
