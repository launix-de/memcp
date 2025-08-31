/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "io"
import "bufio"
import "strings"
import "github.com/launix-de/memcp/scm"

func LoadCSV(schema, table string, f io.Reader, delimiter string, firstLine bool) {
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	lines := make(chan string, 512)

	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	db := GetDatabase(schema)
	if db == nil {
		panic("database " + schema + " does not exist")
	}
	t := db.GetTable(table)
	if t == nil {
		panic("table " + table + " does not exist")
	}
	var cols []string
	if firstLine {
		if !scanner.Scan() {
			panic("CSV does not contain header line")
		}
		cols = strings.Split(scanner.Text(), delimiter) // read headerline
	} else {
		// otherwise use the table's column order
		cols = make([]string, len(t.Columns))
		for i, col := range t.Columns {
			cols[i] = col.Name
		}
	}
	buffer := make([][]scm.Scmer, 0, 4096)
	for s := range lines {
		if s == "" {
			// ignore
		} else {
			arr := strings.Split(s, delimiter)
			x := make([]scm.Scmer, len(t.Columns))
			for i, _ := range t.Columns {
				if i < len(arr) {
					x[i] = scm.Simplify(arr[i])
				}
			}
			buffer = append(buffer, x)
			if len(buffer) >= 4096 {
				t.Insert(cols, buffer, nil, nil, false, nil)
				buffer = buffer[:0]
			}
		}
	}
	if len(buffer) > 0 {
		t.Insert(cols, buffer, nil, nil, false, nil)
	}
}
