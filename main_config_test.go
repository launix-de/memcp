/*
Copyright (C) 2026  Carl-Philip Haensch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.
*/
package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExpandRootPasswordFileArgs(t *testing.T) {
	t.Parallel()
	passwordFile := filepath.Join(t.TempDir(), "root-password")
	if err := os.WriteFile(passwordFile, []byte("correct horse battery staple\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := expandRootPasswordFileArgs([]string{
		"memcp", "--root-password-file=" + passwordFile, "--api-port=4321",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"memcp", "--root-password=correct horse battery staple", "--api-port=4321"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded args = %#v, want %#v", got, want)
	}
}

func TestExpandRootPasswordFileArgsRejectsEmptySecret(t *testing.T) {
	t.Parallel()
	passwordFile := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(passwordFile, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := expandRootPasswordFileArgs([]string{"memcp", "--root-password-file", passwordFile})
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("expected empty-password error, got %v", err)
	}
}
