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

import "github.com/launix-de/memcp/scm"

type SettingsT struct {
	Backtrace bool
	PartitionMaxDimensions int
	DefaultEngine string
	ShardSize int
}

var Settings SettingsT = SettingsT{false, 10, "safe", 60000}

// call this after you filled Settings
func InitSettings() {
	scm.SettingsHaveGoodBacktraces = Settings.Backtrace
}

func ChangeSettings(a ...scm.Scmer) scm.Scmer {
	// schema, filename
	if len(a) == 1 {
		switch scm.String(a[0]) {
			case "Backtrace":
				return Settings.Backtrace
			case "PartitionMaxDimensions":
				return float64(Settings.PartitionMaxDimensions)
			case "DefaultEngine":
				return Settings.DefaultEngine
			case "ShardSize":
				return float64(Settings.ShardSize)
			default:
				panic("unknown setting: " + scm.String(a[0]))
		}
	} else {
		switch scm.String(a[0]) {
			case "Backtrace":
				scm.SettingsHaveGoodBacktraces = Settings.Backtrace
				Settings.Backtrace = scm.ToBool(a[1])
			case "PartitionMaxDimensions":
				Settings.PartitionMaxDimensions = scm.ToInt(a[1])
			case "DefaultEngine":
				Settings.DefaultEngine = scm.String(a[1])
			case "ShardSize":
				Settings.ShardSize = scm.ToInt(a[1])
			default:
				panic("unknown setting: " + scm.String(a[0]))
		}
		return true
	}
}
