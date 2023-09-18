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
	memcp smart clusterable distributed database working best on nvme

	https://pkelchte.wordpress.com/2013/12/31/scm-go/

*/
package main

import "os"
import "io"
import "fmt"
import "flag"
import "bufio"
import "syscall"
import "io/ioutil"
import "os/signal"
import "crypto/rand"
import "path/filepath"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage"

var IOEnv scm.Env

func getImport(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
			filename := path + "/" + scm.String(a[0])
			// TODO: filepath.Walk for wildcards
			wd := filepath.Dir(filename)
			otherPath := scm.Env {
				scm.Vars {
					"__DIR__": path,
					"__FILE__": filename,
					"import": getImport(wd),
					"load": getLoad(wd),
				},
				&IOEnv,
				true,
			}
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			return scm.EvalAll(string(bytes), &otherPath)
		}
}

func getLoad(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
			filename := path + "/" + scm.String(a[0])
			if len(a) > 2 {
				file, err := os.Open(filename)
				if err != nil {
					panic(err)
				}
				splitter := bufio.NewReader(file)
				delimiter := scm.String(a[2])
				if len(delimiter) != 1 {
					panic("load delimiter must be 1 byte long")
				}
				for {
					str, err := splitter.ReadString(delimiter[0])
					if err == io.EOF {
						break // file is finished
					}
					if err != nil {
						panic(err)
					}
					// go??
					scm.Apply(a[1], []scm.Scmer{str});
				}
			} else {
				// read in whole
				bytes, err := ioutil.ReadFile(filename)
				if err != nil {
					panic(err)
				}
				if len(a) > 1 {
					scm.Apply(a[1], []scm.Scmer{string(bytes)});
				} else {
					return string(bytes)
				}
			}
			return "ok"
		}
}

// workaround for flags package to allow multiple values
type arrayFlags []string

func (i *arrayFlags) String() string {
    return "dummy"
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

func main() {
	fmt.Print(`memcp Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
`)

	// init random generator for UUIDs
	uuid.SetRand(rand.Reader)

	// parse command line options
	var commands arrayFlags
	flag.Var(&commands, "c", "Execute scm command")
	basepath := "data"
	flag.StringVar(&basepath, "data", "data", "Data folder for persistence")
	flag.Parse()
	imports := flag.Args()

	// define some IO functions (scm will not provide them since it is sandboxable)
	wd, _ := os.Getwd() // libraries are relative to working directory... is that right?
	IOEnv = scm.Env {
		scm.Vars {
			"print": func (a ...scm.Scmer) scm.Scmer {
					for _, s := range a {
						fmt.Print(scm.String(s))
					}
					fmt.Println()
					return "ok"
				},
			"import": getImport(wd),
			"load": getLoad(wd),
			"serve": scm.HTTPServe,
			"mysql": scm.MySQLServe,
			"password": scm.MySQLPassword,
		},
		&scm.Globalenv,
		true, // other defines go into Globalenv
	}
	// storage initialization
	storage.Init(scm.Globalenv)
	storage.Basepath = basepath
	storage.LoadDatabases()
	// scripts initialization
	scm.Eval(scm.Read("(import \"lib/main.scm\")"), &IOEnv)
	// command line initialization
	for _, filename := range imports {
		fmt.Println("Loading " + filename + " ...")
		scm.Eval(scm.Read("(import \"" + filename + "\")"), &IOEnv)
	}
	for _, command := range commands {
		fmt.Println("Executing " + command + " ...")
		scm.Eval(scm.Read(command), &IOEnv)
	}

	// install exit handler
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	go (func () {
		<-cancelChan
		exitroutine()
		os.Exit(1)
	})()

	// REPL shell
	scm.Repl(&IOEnv)

	// normal shutdown
	exitroutine()
}

func exitroutine() {
	fmt.Println("Exit procedure... syncing to disk")
	fmt.Println("table compression done in ", scm.Globalenv.Vars["rebuild"].(func(...scm.Scmer) scm.Scmer)())
	fmt.Println("Exit procedure finished")
}
