//go:build !ceph

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

// CephFactory is a stub when Ceph support is not compiled in.
// Build with -tags=ceph to enable Ceph support.
type CephFactory struct {
	UserName    string
	ClusterName string
	ConfFile    string
	Pool        string
	Prefix      string
}

func (f *CephFactory) CreateDatabase(schema string) PersistenceEngine {
	panic("Ceph support not compiled in. Build with: go build -tags=ceph")
}
