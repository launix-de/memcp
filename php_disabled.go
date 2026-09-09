//go:build !php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"

func startPHP(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--php-") {
			return fmt.Errorf("PHP support is not built in; use make php")
		}
	}
	return nil
}

func stopPHP() {}

func getServePHP(wd string) func(...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer { panic("PHP support is not built in; use make php") }
}
