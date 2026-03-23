/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
import "time"
import "bufio"
import "sync"
import "syscall"
import "runtime"
import "io/ioutil"
import "strings"
import "os/signal"
import "crypto/rand"
import "path/filepath"
import "runtime/pprof"
import "runtime/debug"
import "github.com/google/uuid"
import "github.com/fsnotify/fsnotify"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage"

var IOEnv scm.Env

// printLogFunc is set after lib load to route (print) output to storage.
var printLogFunc func(string)

func getReadfile(path string) func(a ...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		filename := filepath.Join(path, scm.String(a[0]))
		data, err := os.ReadFile(filename)
		if err != nil {
			panic("readfile: " + err.Error())
		}
		return scm.NewString(string(data))
	}
}

func getImport(path string) func(a ...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		filename := filepath.Join(path, scm.String(a[0]))
		// TODO: filepath.Walk for wildcards
		wd := filepath.Dir(filename)
		otherPath := scm.Env{
			Vars: scm.Vars{
				"__DIR__":     scm.NewString(path),
				"__FILE__":    scm.NewString(filename),
				"import":      scm.NewFunc(getImport(wd)),
				"load":        scm.NewFunc(getLoad(wd)),
				"stream":      scm.NewFunc(getStream(wd)),
				"watch":       scm.NewFunc(getWatch(wd)),
				"readfile":    scm.NewFunc(getReadfile(wd)),
				"serveStatic": scm.NewFunc(scm.HTTPStaticGetter(wd)),
			},
			VarsNumbered: nil,
			Outer:        &IOEnv,
			Nodefine:     true,
		}
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		return scm.EvalAll(filename, string(bytes), &otherPath)
	}
}

func getStream(path string) func(a ...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		filename := filepath.Join(path, scm.String(a[0]))
		stream, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		return scm.NewAny(io.Reader(stream))
	}
}

func getLoad(path string) func(a ...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		stream, ok := a[0].Any().(io.Reader)
		if !ok {
			// not a stream? call getStream
			stream = getStream(path)(a[0]).Any().(io.Reader)
		}
		if len(a) > 2 {
			splitter := bufio.NewReader(stream)
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
				scm.Apply(a[1], scm.NewString(str))
			}
		} else {
			// read in whole
			bytes, err := ioutil.ReadAll(stream)
			if err != nil {
				panic(err)
			}
			if len(a) > 1 {
				scm.Apply(a[1], scm.NewString(string(bytes)))
			} else {
				return scm.NewString(string(bytes))
			}
		}
		return scm.NewBool(true)
	}
}

func getWatch(path string) func(a ...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		filename := filepath.Join(path, scm.String(a[0]))
		reread := func() {
			// read in whole
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			scm.Apply(a[1], scm.NewString(string(bytes)))
		}
		reread() // read once at the beginning in sync
		// watch for changes
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}
		go func() {
			for {
				select {
				case /*event :=*/ <-watcher.Events:
					// flush all other events
					for {
						time.Sleep(10 * time.Millisecond) // delay a bit, so we don't read empty files
						select {
						case <-watcher.Events:
							// ignore
						default:
							goto to_reread
						}
					}
				to_reread:
					// now reread the file
					func() {
						defer func() {
							if err := recover(); err != nil {
								// error happens during reload: log to console
								fmt.Println(err)
							}
						}()
						reread()
					}()
					watcher.Add(filename) // text editors rename, so we have to rewatch
				}
			}
		}()
		err = watcher.Add(filename)
		if err != nil {
			panic(err)
		}
		return scm.NewBool(true)
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

func setupIO(wd string) {
	// define some IO functions (scm will not provide them since it is sandboxable)
	IOEnv = scm.Env{
		Vars:         scm.Vars{},
		VarsNumbered: nil,
		Outer:        &scm.Globalenv,
		Nodefine:     true, // other defines go into Globalenv
	}
	scm.DeclareTitle("IO")
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "print",
		Desc: "Prints values to stdout (only in IO environment)",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			var msg string
			for _, s := range a {
				msg += scm.String(s)
			}
			fmt.Println(msg)
			if printLogFunc != nil {
				printLogFunc(msg)
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", ParamName: "value...", ParamDesc: "values to print", Variadic: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "env",
		Desc: "returns the content of a environment variable",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) > 1 {
				if val, ok := os.LookupEnv(scm.String(a[0])); ok {
					return scm.NewString(val)
				}
				return a[1]
			}
			return scm.NewString(os.Getenv(scm.String(a[0])))
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "var", ParamDesc: "envvar"},
				{Kind: "string", ParamName: "default", ParamDesc: "default if the env is not found", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "help",
		Desc: "Lists all functions or returns help for a specific function as a string",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				return scm.NewString(scm.Help(scm.NewNil()))
			} else {
				return scm.NewString(scm.Help(a[0]))
			}
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "topic", ParamDesc: "function to get help about", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "import",
		Desc: "Imports a file .scm file into current namespace",
		Fn: (func(...scm.Scmer) scm.Scmer)(getImport(wd)),
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "filename", ParamDesc: "filename relative to folder of source file"},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "load",
		Desc: "Loads a file or stream and returns the string or iterates line-wise",
		Fn: (func(...scm.Scmer) scm.Scmer)(getLoad(wd)),
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string|stream", ParamName: "filenameOrStream", ParamDesc: "filename relative to folder of source file or stream to read from"},
				{Kind: "func", ParamName: "linehandler", ParamDesc: "handler that reads each line; each line may end with delimiter", Optional: true},
				{Kind: "string", ParamName: "delimiter", ParamDesc: "delimiter to extract; if no delimiter is given, the file is read as whole and returned or passed to linehandler", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string|bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "stream",
		Desc: "Opens a file readonly as stream",
		Fn: (func(...scm.Scmer) scm.Scmer)(getStream(wd)),
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "filename", ParamDesc: "filename relative to folder of source file"},
			},
			Return: &scm.TypeDescriptor{Kind: "stream"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "watch",
		Desc: "Loads a file and calls the callback. Whenever the file changes on disk, the file is load again.",
		Fn: (func(...scm.Scmer) scm.Scmer)(getWatch(wd)),
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "filename", ParamDesc: "filename relative to folder of source file"},
				{Kind: "func", ParamName: "updatehandler", ParamDesc: "handler that receives the file content func(content)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "serve",
		Desc: "Opens a HTTP server at a given port",
		Fn: scm.HTTPServe,
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "number", ParamName: "port", ParamDesc: "port number for HTTP server"},
				{Kind: "func", ParamName: "handler", ParamDesc: "handler: lambda(req res) that handles the http request (TODO: detailed documentation)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "serveStatic",
		Desc: "creates a static handler for use as a callback in (serve) - returns a handler lambda(req res)",
		Fn: (func(...scm.Scmer) scm.Scmer)(scm.HTTPStaticGetter(wd)),
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "directory", ParamDesc: "folder with the files to serve"},
			},
			Return: &scm.TypeDescriptor{Kind: "func"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "mysql",
		Desc: "Imports a file .scm file into current namespace",
		Fn: scm.MySQLServe,
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "number", ParamName: "port", ParamDesc: "port number for MySQL server"},
				{Kind: "func", ParamName: "getPassword", ParamDesc: "lambda(username string) string|nil has to return the password for a user or nil to deny login"},
				{Kind: "func", ParamName: "schemacallback", ParamDesc: "lambda(username schema) bool handler check whether user is allowed to schem (string) - you should check access rights here"},
				{Kind: "func", ParamName: "handler", ParamDesc: "lambda(schema sql resultrow session) handler to process sql query (string) in schema (string). resultrow is a lambda(list)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "mysql_socket",
		Desc: "Listen on a Unix domain socket for MySQL protocol",
		Fn: scm.MySQLServeSocket,
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "socketpath", ParamDesc: "path to the Unix domain socket"},
				{Kind: "func", ParamName: "getPassword", ParamDesc: "lambda(username string) string|nil has to return the password for a user or nil to deny login"},
				{Kind: "func", ParamName: "schemacallback", ParamDesc: "lambda(username schema) bool handler check whether user is allowed to schema (string) - you should check access rights here"},
				{Kind: "func", ParamName: "handler", ParamDesc: "lambda(schema sql resultrow session) handler to process sql query (string) in schema (string). resultrow is a lambda(list)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "password",
		Desc: "Hashes a password with sha1 (for mysql user authentication)",
		Fn: scm.MySQLPassword,
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "password", ParamDesc: "plain text password to hash"},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

	// Graceful shutdown callable from Scheme/SQL (SHUTDOWN)
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "shutdown",
		Desc: "Initiates a graceful shutdown of memcp after a short delay",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if scm.ReplInstance != nil {
				scm.ReplInstance.Close()
			} else {
				// no-repl mode: send SIGTERM to self to unblock signal wait
				syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	// Hard crash for crash testing — equivalent to kill -9, no cleanup
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "crash",
		Desc: "Hard process exit with no cleanup (kill -9 equivalent) for crash testing. SCM only, not exposed to SQL.",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			os.Exit(137) // 128 + SIGKILL(9)
			return scm.NewBool(false) // unreachable
		},
		Type: &scm.TypeDescriptor{
			HasSideEffects: true,
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "args",
		Desc: "Returns command line arguments",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			args := make([]scm.Scmer, len(os.Args))
			for i, arg := range os.Args {
				args[i] = scm.NewString(arg)
			}
			return scm.NewSlice(args)
		},
		Type: &scm.TypeDescriptor{
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		Name: "arg",
		Desc: "Gets a command line argument value",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			longname := scm.String(a[0])
			var shortname string
			var defaultValue scm.Scmer

			if len(a) == 2 {
				// (arg "longname" defaultValue)
				defaultValue = a[1]
			} else {
				// (arg "longname" "s" defaultValue)
				if !a[1].IsString() {
					panic("arg: shortname must be string when provided")
				}
				shortname = scm.String(a[1])
				defaultValue = a[2]
			}

			// Check for --longname=value or --longname value
			longPrefix := "--" + longname
			for i, arg := range os.Args {
				if arg == longPrefix && i+1 < len(os.Args) {
					return scm.NewString(os.Args[i+1])
				}
				if len(arg) > len(longPrefix) && arg[:len(longPrefix)+1] == longPrefix+"=" {
					return scm.NewString(arg[len(longPrefix)+1:])
				}
			}

			// Check for -shortname value if shortname provided
			if shortname != "" {
				shortPrefix := "-" + shortname
				for i, arg := range os.Args {
					if arg == shortPrefix && i+1 < len(os.Args) {
						return scm.NewString(os.Args[i+1])
					}
				}
			}

			// Check for boolean flags (--longname without value means true)
			for _, arg := range os.Args {
				if arg == longPrefix {
					return scm.NewBool(true)
				}
				if shortname != "" && arg == "-"+shortname {
					return scm.NewBool(true)
				}
			}

			return defaultValue
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "longname", ParamDesc: "long argument name (without --)"},
				{Kind: "string|any", ParamName: "shortname", ParamDesc: "short argument name (without -) or default value if only 2 args"},
				{Kind: "any", ParamName: "default", ParamDesc: "default value if argument not found", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
}

// readConfigFile reads a config file where each non-blank, non-comment line
// is one CLI argument (e.g. "--api-port=4321" or "-data /var/lib/memcp").
func readConfigFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var args []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "-flag value" → two args; "--flag=value" → one arg (no split needed)
		if i := strings.Index(line, " "); i >= 0 {
			args = append(args, line[:i], strings.TrimSpace(line[i+1:]))
		} else {
			args = append(args, line)
		}
	}
	return args, nil
}

func main() {
	fmt.Print(`memcp Copyright (C) 2023 - 2025   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;

`)

	// init random generator for UUIDs
	uuid.SetRand(rand.Reader)

	// Set thread limit - while loading there is a lot of thread pressure because of waiting for IO
	debug.SetMaxThreads(10000000)

	// process --config=FILE early: inject its lines as CLI args (before explicit args, so CLI overrides)
	{
		remaining := []string{os.Args[0]}
		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]
			var configPath string
			if strings.HasPrefix(arg, "--config=") {
				configPath = arg[len("--config="):]
			} else if arg == "--config" && i+1 < len(os.Args) {
				i++
				configPath = os.Args[i]
			} else {
				remaining = append(remaining, arg)
				continue
			}
			configArgs, err := readConfigFile(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: reading config file %s: %v\n", configPath, err)
				os.Exit(1)
			}
			// prepend config args so explicit CLI flags take precedence
			remaining = append(remaining, configArgs...)
		}
		os.Args = remaining
	}

	// parse command line options
	var commands arrayFlags
	flag.Var(&commands, "c", "Execute scm command")

	basepath := "data"
	flag.StringVar(&basepath, "data", "data", "Data folder for persistence")

	profile := ""
	flag.StringVar(&profile, "profile", "", "Data folder for persistence")

	wd, _ := os.Getwd() // libraries are relative to working directory... or change with -wd PATH
	flag.StringVar(&wd, "wd", wd, "Working Directory for (import) and (load) (Default: .)")

	writeDocu := ""
	flag.StringVar(&writeDocu, "write-docu", "", "Write documentation as .md documents to that folder and exit")

	noRepl := false
	flag.BoolVar(&noRepl, "no-repl", false, "Run without interactive REPL (wait for SIGTERM/SIGINT instead)")

	// Parse only known flags, ignore unknown ones for Scheme to handle
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nAdditional arguments (handled by Scheme):\n")
		fmt.Fprintf(os.Stderr, "  --api-port=PORT        HTTP API port (default 4321)\n")
		fmt.Fprintf(os.Stderr, "  --mysql-port=PORT      MySQL protocol port (default 3307)\n")
		fmt.Fprintf(os.Stderr, "  --disable-api          Disable HTTP API server\n")
		fmt.Fprintf(os.Stderr, "  --mysql-socket=PATH    Unix socket path (default /tmp/memcp.sock, empty to disable)\n")
		fmt.Fprintf(os.Stderr, "  --disable-mysql        Disable MySQL protocol server\n")
		fmt.Fprintf(os.Stderr, "... and much more (please refer to your module's documentation)\n\n")
	}

	// Parse until first unknown flag
	knownArgs := []string{}
	schemeArgs := []string{}
	importFiles := []string{}
	skipNext := false

	for i, arg := range os.Args[1:] {
		if skipNext {
			skipNext = false
			continue
		}

		if arg == "-c" || arg == "-data" || arg == "-profile" || arg == "-wd" || arg == "-write-docu" {
			knownArgs = append(knownArgs, arg)
			if i+1 < len(os.Args[1:]) {
				knownArgs = append(knownArgs, os.Args[i+2])
				skipNext = true
			}
		} else if arg == "-no-repl" || arg == "--no-repl" {
			knownArgs = append(knownArgs, "-no-repl")
		} else if arg == "-h" || arg == "-help" || arg == "--help" {
			knownArgs = append(knownArgs, arg)
		} else if len(arg) > 2 && arg[:2] == "--" {
			// This is a long flag for Scheme - don't treat as import file
			schemeArgs = append(schemeArgs, arg)
		} else if len(arg) > 1 && arg[0] == '-' {
			// This looks like a short flag but we don't recognize it - also for Scheme
			schemeArgs = append(schemeArgs, arg)
		} else {
			// This is probably an import file
			importFiles = append(importFiles, arg)
		}
	}

	// Set os.Args to include all arguments for Scheme
	fullArgs := append(append([]string{os.Args[0]}, knownArgs...), schemeArgs...)

	// Parse only known Go flags
	os.Args = append([]string{os.Args[0]}, knownArgs...)
	flag.Parse()
	imports := append(flag.Args(), importFiles...)

	// Restore full args for Scheme
	os.Args = fullArgs

	// storage initialization
	setupIO(wd)
	storage.Init(scm.Globalenv)

	if writeDocu != "" {
		scm.WriteDocumentation(writeDocu)
		os.Exit(0)
	}

	storage.Basepath = basepath
	// Run initialization inside a gls.Go goroutine so that storage operations
	// (scans, inserts, triggers) always have a valid GLS goroutine ID.
	// Without this, hasWriteOwner/enterMutationOwner silently no-op (goid==0),
	// breaking reentrancy guards and causing deadlocks.
	initDone := make(chan struct{})
	gls.Go(func() {
		defer close(initDone)
		storage.LoadDatabases()
		go storage.Clean() // remove crash-orphaned blobs and shard files in background
		// scripts initialization
		if len(imports) == 0 {
			// search for lib/main.scm in well-known locations
			exePath, _ := os.Executable()
			exeDir := filepath.Dir(exePath)
			candidates := []string{
				filepath.Join(wd, "lib/main.scm"),
				filepath.Join(exeDir, "lib/main.scm"),
				"/usr/local/lib/memcp/lib/main.scm",
				"/usr/lib/memcp/lib/main.scm",
			}
			found := false
			for _, candidate := range candidates {
				if _, err := os.Stat(candidate); err == nil {
					fmt.Println("Loading " + candidate + " ...")
					dir := filepath.Dir(candidate)
					base := filepath.Base(candidate)
					getImport(dir)(scm.NewString(base))
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "error: could not find lib/main.scm; tried: %v\n", candidates)
				os.Exit(1)
			}
		} else {
			// load scripts from command line
			for _, scmfile := range imports {
				fmt.Println("Loading " + scmfile + " ...")
				IOEnv.Vars["import"].Func()(scm.NewString(scmfile))
			}
		}
		// wire up print log: (print) and (time) → system_statistic.logs
		logFn := storage.MakePrintLogFunc()
		printLogFunc = logFn
		origTracePrint := scm.TracePrintFunc
		scm.TracePrintFunc = func(msg string) {
			origTracePrint(msg)
			if logFn != nil {
				logFn(msg)
			}
		}
		for _, command := range commands {
			fmt.Println("Executing " + command + " ...")
			code := scm.Read("command line", command)
			scm.Validate(code, "any")
			code = scm.Optimize(code, &IOEnv)
			scm.Eval(code, &IOEnv)
		}
	})
	<-initDone

	// install exit handler
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	go (func() {
		<-cancelChan
		exitroutine()
		os.Exit(1)
	})()

	fmt.Print(`

    Type (help) to show help

`)
	// init profiling
	if profile != "" {
		f, err := os.Create(profile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// start cron (inside gls.Go so storage scans in Rebuild have a valid GLS id)
	exitable.Add(1)
	gls.Go(cronroutine)

	// REPL shell or wait for signal
	if noRepl {
		signal.Stop(cancelChan) // stop duplicate handler; this goroutine handles the signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
	} else {
		signal.Stop(cancelChan) // let readline handle SIGINT/SIGTERM in REPL mode
		replDone := make(chan struct{})
		gls.Go(func() {
			defer close(replDone)
			scm.Repl(&IOEnv)
		})
		<-replDone
	}

	// normal shutdown
	exitroutine()
}

var exitsignal chan bool = make(chan bool, 1) // set true to start shutdown routine and wait for all jobs
var exitable sync.WaitGroup
var exitOnce sync.Once

func cronroutine() {
	defer exitable.Done()

	sched := &scm.DefaultScheduler
	const rebuildInterval = 15 * time.Minute

	var scheduleRebuild func(time.Duration)
	scheduleRebuild = func(delay time.Duration) {
		if _, ok := sched.ScheduleAfter(delay, func() {
			fmt.Println("running 15min cron ...")
			fmt.Println("table compression done in ", storage.Rebuild(false, true))
			scheduleRebuild(rebuildInterval)
		}); !ok {
			fmt.Println("scheduler stopped before scheduling rebuild job")
		}
	}

	scheduleRebuild(rebuildInterval)

	<-exitsignal
	sched.Stop()
}

func exitroutine() {
	exitOnce.Do(func() {
		drainSecs := storage.Settings.ShutdownDrainSeconds
		scm.ShutdownServers(drainSecs)
		exitsignal <- true
		exitable.Wait()
		fmt.Println("Exit procedure...")
		if scm.ReplInstance != nil {
			// in case it dosen't exit properly
			scm.ReplInstance.Close()
		}
		fmt.Println("finalizing storage...")
		func() {
			defer func() {
				if r := recover(); r != nil {
					// ensure shutdown continues even if saving panics
					fmt.Println("error: UnloadDatabases failed:", r)
				}
			}()
			storage.UnloadDatabases()
		}()
		fmt.Println("finalizing memory...")
		runtime.GC() // this will call the finalizers on shards
		fmt.Println("Exit procedure finished")
	})
}
