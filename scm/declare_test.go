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
	"strconv"
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
		"    - **visit** (`func`): handles one row",
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
	for _, expected := range []string{"**rows** (`list<list<any>>`)", "**visit** (`func`)", "**results** (`list<string>`)", "**result** (`string`)"} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("generated documentation does not contain %q:\n%s", expected, doc)
		}
	}
	if strings.Contains(doc, "**visit** (`func(row:list<any>) -> bool`)") {
		t.Fatalf("generated documentation repeats an expanded callback signature:\n%s", doc)
	}
}

func TestHelpUsesRecursiveTypeDescriptors(t *testing.T) {
	oldTitles, oldDeclarations, oldHashes := declaration_titles, declarations, declarations_hash
	defer func() {
		declaration_titles, declarations, declarations_hash = oldTitles, oldDeclarations, oldHashes
	}()
	declaration_titles = nil
	declarations = make(map[string]*Declaration)
	declarations_hash = make(map[string]*Declaration)

	env := Env{Vars: make(Vars)}
	Declare(&env, &Declaration{
		Name: "nested-help",
		Fn:   func(...Scmer) Scmer { return NewNil() },
		Type: &TypeDescriptor{
			Kind:        "func",
			Description: "documents nested values",
			Params:      []*TypeDescriptor{nestedDocumentationType()},
			Return:      &TypeDescriptor{Kind: "bool", Label: "success"},
		},
	})

	help := Help(NewString("nested-help"))
	for _, expected := range []string{
		" - options (assoc): configuration fields",
		"   - rows (list<list<any>>): input rows",
		"     - row (list<any>)",
		"       - value (any)",
		"   - visit (func): handles one row",
		"     Parameters:",
		"       - row (list<any>)",
		"     Returns:",
		"       - accepted (bool)",
		"Returns:\n - success (bool)",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("recursive help does not contain %q:\n%s", expected, help)
		}
	}
	if strings.Contains(help, "visit (func(row:list<any>) -> bool)") {
		t.Fatalf("recursive help repeats an expanded callback signature:\n%s", help)
	}
}

func TestFunctionFactoriesDescribeReturnedFunctions(t *testing.T) {
	tests := []struct {
		name              string
		returnLabel       string
		nestedReturnLabel string
	}{
		{name: "make_structural_index", returnLabel: "lookup"},
		{name: "make_structural_catalog", returnLabel: "catalog", nestedReturnLabel: "value"},
		{name: "newpromise", returnLabel: "promise"},
		{name: "newsession", returnLabel: "session"},
		{name: "once", returnLabel: "once_wrapper"},
		{name: "mutex", returnLabel: "locked"},
		{name: "parser", returnLabel: "parser"},
		{name: "lambda", returnLabel: "lambda"},
		{name: "collate", returnLabel: "relation"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := declarations[test.name]
			if declaration == nil || declaration.Type == nil || declaration.Type.Return == nil {
				t.Fatalf("%s has no declared return type", test.name)
			}
			returned := declaration.Type.Return
			if !hasTypeKind(returned.Kind, "func") {
				t.Fatalf("%s return kind = %q, want a function", test.name, returned.Kind)
			}
			if returned.Label != test.returnLabel || returned.Description == "" {
				t.Fatalf("%s returned function label/description = %q/%q", test.name, returned.Label, returned.Description)
			}
			if returned.Return == nil || returned.Return.Label == "" || returned.Return.Description == "" {
				t.Fatalf("%s returned function has an undocumented result: %#v", test.name, returned.Return)
			}
			if test.nestedReturnLabel != "" {
				if !hasTypeKind(returned.Return.Kind, "func") || returned.Return.Return == nil {
					t.Fatalf("%s does not describe its second-level returned function", test.name)
				}
				if returned.Return.Return.Label != test.nestedReturnLabel || returned.Return.Return.Description == "" {
					t.Fatalf("%s second-level result is undocumented: %#v", test.name, returned.Return.Return)
				}
			}
		})
	}
}

func TestCallbackDescriptionsDoNotRepeatStructuredSignatures(t *testing.T) {
	var inspect func(string, *TypeDescriptor)
	inspect = func(path string, descriptor *TypeDescriptor) {
		if descriptor == nil {
			return
		}
		if hasTypeKind(descriptor.Kind, "func") && (strings.Contains(descriptor.Description, "func(") || strings.Contains(descriptor.Description, "lambda(")) {
			t.Errorf("%s repeats a structured function signature in its description: %q", path, descriptor.Description)
		}
		for i, param := range descriptor.Params {
			inspect(path+".param["+strconv.Itoa(i)+"]", param)
		}
		inspect(path+".return", descriptor.Return)
		inspect(path+".element", descriptor.Element)
		for key, value := range descriptor.Keys {
			inspect(path+".key["+key+"]", value)
		}
	}

	for name, declaration := range declarations {
		inspect(name, declaration.Type)
	}
}
