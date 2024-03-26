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
	"io"
	"fmt"
	"bytes"
	"runtime/debug"
	"github.com/chzyer/readline"
)

const newprompt  = "\033[32m>\033[0m "
const contprompt = "\033[32m.\033[0m "
const resultprompt = "\033[31m=\033[0m "

func Repl(en *Env) {
	l, err := readline.NewEx(&readline.Config {
		Prompt: newprompt,
		HistoryFile: ".memcp-history.tmp",
		InterruptPrompt: "^C",
		EOFPrompt: "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
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
					fmt.Println("panic:", r, string(debug.Stack()))
					oldline = ""
					l.SetPrompt(newprompt)
				}
			}()
			var b bytes.Buffer
			code := Read("user prompt", line)
			Validate(code, "any")
			code = Optimize(code, en)
			result := Eval(code, en)
			Serialize(&b, result, en, en)
			fmt.Print(resultprompt)
			fmt.Println(b.String())
			oldline = ""
			l.SetPrompt(newprompt)
		}()
	}
}
