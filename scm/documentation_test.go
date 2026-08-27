/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDocumentationPreamblePreservesLegacyIntroduction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "strings.md")
	content := "# Strings\n\nA hand-written introduction.\n\n- First topic\n- Second topic\n\n## old_function\n\nstale generated text\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readDocumentationPreamble(path, "Strings")
	if err != nil {
		t.Fatal(err)
	}
	want := "A hand-written introduction.\n\n- First topic\n- Second topic"
	if got != want {
		t.Fatalf("preamble = %q, want %q", got, want)
	}
}

func TestReadDocumentationPreambleMarkersAllowSubheadings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storage.md")
	content := strings.Join([]string{
		"# Storage",
		"",
		documentationPreambleStart,
		"",
		"Storage overview.",
		"",
		"## Design goals",
		"",
		"This heading belongs to the introduction.",
		"",
		documentationPreambleEnd,
		"",
		"## generated_function",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readDocumentationPreamble(path, "Storage")
	if err != nil {
		t.Fatal(err)
	}
	want := "Storage overview.\n\n## Design goals\n\nThis heading belongs to the introduction."
	if got != want {
		t.Fatalf("preamble = %q, want %q", got, want)
	}
}

func TestReadDocumentationPreambleSuppliesDefault(t *testing.T) {
	got, err := readDocumentationPreamble(filepath.Join(t.TempDir(), "date.md"), "Date")
	if err != nil {
		t.Fatal(err)
	}
	if got != documentationPreambles["Date"] {
		t.Fatalf("preamble = %q, want configured Date introduction", got)
	}

	unknown, err := readDocumentationPreamble(filepath.Join(t.TempDir(), "future.md"), "Future Module")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unknown, "Future Module") {
		t.Fatalf("fallback preamble %q does not identify its chapter", unknown)
	}
}

func TestWriteDocumentationPreservesMarkedPreamble(t *testing.T) {
	dir := t.TempDir()
	if err := WriteDocumentation(dir); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "strings.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(data)
	if !strings.Contains(generated, documentationPreambleStart) || !strings.Contains(generated, documentationPreambleEnd) {
		t.Fatalf("generated chapter has no preamble markers:\n%s", generated)
	}

	custom := "A maintained introduction.\n\n## Usage\n\nThis subsection must survive regeneration."
	start := strings.Index(generated, documentationPreambleStart) + len(documentationPreambleStart)
	end := strings.Index(generated[start:], documentationPreambleEnd) + start
	generated = generated[:start] + "\n\n" + custom + "\n\n" + generated[end:]
	if err := os.WriteFile(path, []byte(generated), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := WriteDocumentation(dir); err != nil {
		t.Fatal(err)
	}
	regenerated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(regenerated), custom) {
		t.Fatalf("custom preamble was lost during regeneration:\n%s", regenerated)
	}
	if !strings.Contains(string(regenerated), "## string?") {
		t.Fatalf("generated function catalog is missing after regeneration:\n%s", regenerated)
	}
}
