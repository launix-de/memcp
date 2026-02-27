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
package storage

import "github.com/launix-de/memcp/scm"

func initDashboard(en scm.Env) {
	scm.DeclareTitle("Dashboard")

	scm.Declare(&en, &scm.Declaration{
		Name:         "cache_stat",
		Desc:         "Returns cache statistics as an associative list with current_memory, persisted_budget, memory_budget, persisted_memory",
		MinParameter: 0,
		MaxParameter: 0,
		Params:       []scm.DeclarationParameter{},
		Returns:      "list",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			stat := GlobalCache.Stat()
			return scm.NewSlice([]scm.Scmer{
				scm.NewString("current_memory"), scm.NewInt(stat.CurrentMemory),
				scm.NewString("persisted_budget"), scm.NewInt(stat.PersistedBudget),
				scm.NewString("memory_budget"), scm.NewInt(stat.MemoryBudget),
				scm.NewString("persisted_memory"), scm.NewInt(stat.PersistedMemory),
			})
		},
	})
}
