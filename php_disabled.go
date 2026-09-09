//go:build !php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "fmt"
import "strings"

func startPHP(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--php-") {
			return fmt.Errorf("PHP support is not built in; use make php")
		}
	}
	return nil
}

func stopPHP() {}
