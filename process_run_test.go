package main

import "testing"

func TestRunProcessPassesExactArgumentsWithoutShell(t *testing.T) {
	exitCode, stdout, stderr := runProcess("/usr/bin/printf", []string{"%s", "$(printf injected); *"}, "")
	if exitCode != 0 || stdout != "$(printf injected); *" || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestRunProcessForwardsBinaryStdin(t *testing.T) {
	input := "plain\x00binary\xffdata"
	exitCode, stdout, stderr := runProcess("/bin/cat", nil, input)
	if exitCode != 0 || stdout != input || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestRunProcessReportsNonZeroExit(t *testing.T) {
	exitCode, stdout, stderr := runProcess("/bin/sh", []string{"-c", "printf out; printf err >&2; exit 7"}, "")
	if exitCode != 7 || stdout != "out" || stderr != "err" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}
