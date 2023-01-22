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
/*
	cpdb smart clusterable distributed database working best on nvme

	https://pkelchte.wordpress.com/2013/12/31/scm-go/

*/
package main

import "fmt"
import "github.com/launix-de/cpdb/scm"
import "github.com/launix-de/cpdb/storage"
import "github.com/lrita/numa"

func main() {
	fmt.Print(`cpdb Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
`)
	// print some info
	fmt.Println("NUMA support:", numa.Available())

	// define user specific functions
	scm.Globalenv.Vars["print"] = func (a ...scm.Scmer) scm.Scmer {
		for _, s := range a {
			fmt.Print(scm.String(s))
		}
		fmt.Println()
		return "ok"
	}
	// storage initialization
	storage.Init(scm.Globalenv)
	// scripts initialization
	scm.Eval(scm.Read("(print \"loaded test data in: \" (loadJSON \"test.jsonl\"))"), &scm.Globalenv)
	scm.Repl()
}
