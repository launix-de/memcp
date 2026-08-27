/*
Copyright (C) 2026  Carl-Philip Haensch

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

func nestedDocumentationType() *TypeDescriptor {
	return &TypeDescriptor{
		Kind:        "assoc",
		Label:       "options",
		Description: "configuration fields",
		Keys: map[string]*TypeDescriptor{
			"rows": {
				Kind:        "list",
				Label:       "rows",
				Description: "input rows",
				Element: &TypeDescriptor{
					Kind:    "list",
					Label:   "row",
					Element: &TypeDescriptor{Kind: "any", Label: "value"},
				},
			},
			"visit": {
				Kind:        "func",
				Label:       "visit",
				Description: "handles one row",
				Params: []*TypeDescriptor{{
					Kind:    "list",
					Label:   "row",
					Element: &TypeDescriptor{Kind: "any", Label: "value"},
				}},
				Return: &TypeDescriptor{Kind: "bool", Label: "accepted"},
			},
		},
	}
}

func TestTypeDescriptorWritesDeepDocumentation(t *testing.T) {
	var output strings.Builder
	nestedDocumentationType().WriteDocumentation(&output, 1)
	doc := output.String()

	for _, expected := range []string{
		"  - **options** (`assoc`): configuration fields",
		"    - **rows** (`list<list<any>>`): input rows",
		"      - **row** (`list<any>`)",
		"        - **value** (`any`)",
		"    - **visit** (`func(row:list<any>) -> bool`): handles one row",
		"      - **Parameters**",
		"        - **row** (`list<any>`)",
		"      - **Returns**",
		"        - **accepted** (`bool`)",
	} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("nested documentation does not contain %q:\n%s", expected, doc)
		}
	}
	if strings.Index(doc, "**rows**") > strings.Index(doc, "**visit**") {
		t.Fatalf("assoc fields are not sorted:\n%s", doc)
	}
}

func TestFormatTypeSignatureHandlesUnionContainers(t *testing.T) {
	typeDesc := &TypeDescriptor{
		Kind:    "list|nil",
		Element: &TypeDescriptor{Kind: "func|string", Params: []*TypeDescriptor{{Kind: "int", Label: "id"}}, Return: &TypeDescriptor{Kind: "bool"}},
	}
	if got, want := FormatTypeSignature(typeDesc), "list<func(id:int) -> bool|string>|nil"; got != want {
		t.Fatalf("FormatTypeSignature() = %q, want %q", got, want)
	}
}

func TestWriteDocumentationUsesRecursiveTypeDescriptors(t *testing.T) {
	oldTitles, oldDeclarations, oldHashes := declaration_titles, declarations, declarations_hash
	defer func() {
		declaration_titles, declarations, declarations_hash = oldTitles, oldDeclarations, oldHashes
	}()
	declaration_titles = nil
	declarations = make(map[string]*Declaration)
	declarations_hash = make(map[string]*Declaration)

	DeclareTitle("Nested")
	env := Env{Vars: make(Vars)}
	Declare(&env, &Declaration{
		Name: "nested",
		Fn:   func(...Scmer) Scmer { return NewNil() },
		Type: &TypeDescriptor{
			Kind:        "func",
			Description: "documents nested values",
			Params:      []*TypeDescriptor{nestedDocumentationType()},
			Return:      &TypeDescriptor{Kind: "list", Label: "results", Element: &TypeDescriptor{Kind: "string", Label: "result"}},
		},
	})

	folder := t.TempDir()
	if err := WriteDocumentation(folder); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(folder, "nested.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(content)
	for _, expected := range []string{"**rows** (`list<list<any>>`)", "**visit** (`func(row:list<any>) -> bool`)", "**results** (`list<string>`)", "**result** (`string`)"} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("generated documentation does not contain %q:\n%s", expected, doc)
		}
	}
}
