/*
Copyright (C) 2023  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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

package scm

import (
	"os"
	"io"
	"log"
	"fmt"
	"bytes"
	"regexp"
	"strings"
	"runtime/debug"
	"github.com/chzyer/readline"
)

const newprompt  = "\033[32m>\033[0m "
const contprompt = "\033[32m.\033[0m "
const resultprompt = "\033[31m=\033[0m "

var lambdaExpr *regexp.Regexp = regexp.MustCompile("\\(lambda\\s*\\(([^)]+)\\)")

/* implements interface readline.AutoCompleter */
func (en *Env) Do(line []rune, pos int) (newLine [][]rune, offset int) {
	start := pos
	for start >= 1 && line[start-1] != '(' && line[start-1] != ')' && line[start-1] != ' ' {
		start--
	}
	pfx := string(line[start:pos])
	offset = len(pfx)
	// iterate documentation
	for _, d := range declarations {
		if strings.HasPrefix(d.Name, pfx) && en.FindRead(Symbol(d.Name)) != nil {
			if d.Name == "lambda" {
				newLine = append(newLine, []rune("lambda ("[offset:]))
			} else {
				newLine = append(newLine, []rune(d.Name[offset:]))
			}
		}
	}
	// iterate variables
	for en != nil {
		for s, _ := range en.Vars {
			if strings.HasPrefix(string(s), pfx) {
				newLine = append(newLine, []rune(s[offset:]))
			}
		}
		en = en.Outer // iterate over parent scope
	}
	// find lambda variables in the line
	for _, m := range lambdaExpr.FindAllStringSubmatch(string(line), -1) {
		// each declared parameter of the lambda is also completed
		for _, s := range strings.Split(m[1], " ") {
			if strings.HasPrefix(s, pfx) {
				newLine = append(newLine, []rune(s[offset:]))
			}
		}
	}
	return
}

var ReplInstance *readline.Instance

func Repl(en *Env) {
	l, err := readline.NewEx(&readline.Config {
		Prompt: newprompt,
		HistoryFile: ".memcp-history.tmp",
		AutoComplete: en,
		InterruptPrompt: "^C",
		EOFPrompt: "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	ReplInstance = l
	defer l.Close()
	l.CaptureExitSignal()

	oldline := ""
	for {
		line, err := l.Readline()
		line = oldline + line
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if line == "" {
			continue
		}

		// anti-panic func
		func () {
			defer func () {
				if r := recover(); r != nil {
					if r == "expecting matching )" {
						// keep oldline
						oldline = line + "\n"
						l.SetPrompt(contprompt)
						return
					}
					PrintError(r)
					oldline = ""
					l.SetPrompt(newprompt)
				}
			}()
			var b bytes.Buffer
			code := Read("user prompt", line)
			Validate(code, "any")
			code = Optimize(code, en)
			result := Eval(code, en)
			Serialize(&b, result, en)
			fmt.Print(resultprompt)
			fmt.Println(b.String())
			oldline = ""
			l.SetPrompt(newprompt)
		}()
	}
	ReplInstance = nil
}

var errorlog *log.Logger
func init() {
	errorlog = log.New(os.Stderr, "", 0)
}
func PrintError(r any) {
	s := fmt.Sprint(r)
	numlines := strings.Count(s, "\nin ") * 4 + 9 // skip those stack trace lines that peel out of the error message
	trace := string(debug.Stack())
	for numlines > 0 {
		if trace == "" {
			break
		}
		if trace[0] == '\n' {
			numlines--
		}
		trace = trace[1:]
	}
	errorlog.Println(r, ": \n", trace)
}
