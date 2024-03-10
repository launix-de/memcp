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

import "os"
import "bufio"
import "strings"
import "github.com/launix-de/memcp/scm"

func LoadCSV(schema, table, filename, delimiter string) {
	f, _ := os.Open(filename)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	lines := make(chan string, 64)

	go func () {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	db := GetDatabase(schema)
	if db == nil {
		panic("database " + schema + " does not exist")
	}
	t := db.Tables.Get(table)
	if t == nil {
		panic("table " + table + " does not exist")
	}
	for s := range(lines) {
		if s == "" {
			// ignore
		} else {
			arr := strings.Split(s, delimiter)
			x := make([]scm.Scmer, 2*len(t.Columns))
			for i, col := range t.Columns {
				x[2*i] = col.Name
				if i < len(arr) {
					x[2*i+1] = scm.Simplify(arr[i])
				}
			}
			t.Insert(x)
		}
	}
}

