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

import "strings"

import "github.com/launix-de/memcp/scm"

type maintenanceCapabilities struct {
	class           string
	canDrop         bool
	canTruncate     bool
	canRename       bool
	canAlter        bool
	canChangeEngine bool
}

type maintenanceOperation string

const (
	maintenanceDrop         maintenanceOperation = "drop"
	maintenanceTruncate     maintenanceOperation = "truncate"
	maintenanceRename       maintenanceOperation = "rename"
	maintenanceAlter        maintenanceOperation = "alter"
	maintenanceChangeEngine maintenanceOperation = "change engine"
)

func unrestrictedMaintenanceCapabilities() maintenanceCapabilities {
	return maintenanceCapabilities{
		class:           "user",
		canDrop:         true,
		canTruncate:     true,
		canRename:       true,
		canAlter:        true,
		canChangeEngine: true,
	}
}

func databaseMaintenanceCapabilities(schema string) maintenanceCapabilities {
	switch strings.ToLower(schema) {
	case "system":
		return maintenanceCapabilities{class: "system_catalog"}
	case "system_statistic":
		return maintenanceCapabilities{class: "telemetry"}
	default:
		return unrestrictedMaintenanceCapabilities()
	}
}

func tableMaintenanceCapabilities(schema, name string) maintenanceCapabilities {
	if strings.EqualFold(name, ".blobs") {
		return maintenanceCapabilities{class: "engine_internal"}
	}
	switch strings.ToLower(schema) {
	case "system":
		return maintenanceCapabilities{class: "system_catalog"}
	case "system_statistic":
		return maintenanceCapabilities{class: "telemetry", canTruncate: true}
	default:
		return unrestrictedMaintenanceCapabilities()
	}
}

func (capabilities maintenanceCapabilities) allows(operation maintenanceOperation) bool {
	switch operation {
	case maintenanceDrop:
		return capabilities.canDrop
	case maintenanceTruncate:
		return capabilities.canTruncate
	case maintenanceRename:
		return capabilities.canRename
	case maintenanceAlter:
		return capabilities.canAlter
	case maintenanceChangeEngine:
		return capabilities.canChangeEngine
	default:
		return false
	}
}

func maintenanceCapabilitiesScmer(capabilities maintenanceCapabilities) scm.Scmer {
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("Class"), scm.NewString(capabilities.class),
		scm.NewString("CanDrop"), scm.NewBool(capabilities.canDrop),
		scm.NewString("CanTruncate"), scm.NewBool(capabilities.canTruncate),
		scm.NewString("CanRename"), scm.NewBool(capabilities.canRename),
		scm.NewString("CanAlter"), scm.NewBool(capabilities.canAlter),
		scm.NewString("CanChangeEngine"), scm.NewBool(capabilities.canChangeEngine),
	})
}

func requireDatabaseMaintenance(schema string, operation maintenanceOperation) {
	if !databaseMaintenanceCapabilities(schema).allows(operation) {
		panic("Database " + schema + " is required by MemCP and does not allow " + string(operation))
	}
}

func requireTableMaintenance(schema, name string, operation maintenanceOperation) {
	if !tableMaintenanceCapabilities(schema, name).allows(operation) {
		panic("Table " + schema + "." + name + " is required by MemCP and does not allow " + string(operation))
	}
}
