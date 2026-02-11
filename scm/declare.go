/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

import "os"
import "fmt"
import "strings"
import "path/filepath"

type Declaration struct {
	Name         string
	Desc         string
	MinParameter int
	MaxParameter int
	Params       []DeclarationParameter
	Returns      string // any | string | number | int | bool | func | list | symbol | nil
	Fn           func(...Scmer) Scmer
	Foldable     bool // safe to constant-fold when all args are literals
	Forbidden    bool // optimizer-only: hidden from help, blocked from .scm code
}

// TypeDescriptor describes the type of any Scmer value at arbitrary depth.
// Uses pointers throughout — nil means "unknown / don't care" (conservative).
type TypeDescriptor struct {
	Kind     string                     // "any"|"string"|"number"|"int"|"bool"|"nil"|"symbol"|"func"|"list"|"assoc"
	Escape   bool                       // value may outlive its scope
	Transfer bool                       // callee receives ownership, can mutate
	Const    bool                       // value is a compile-time constant
	Params   []*TypeDescriptor          // for Kind="func": parameter types
	Return   *TypeDescriptor            // for Kind="func": return type
	Keys     map[string]*TypeDescriptor // for Kind="assoc": per-key type info
	Element  *TypeDescriptor            // for Kind="list": element type
}

type DeclarationParameter struct {
	Name     string
	Type     string          // any | string | number | int | bool | func | list | symbol | nil
	Desc     string
	TypeInfo *TypeDescriptor // rich type info (nil = unknown / no annotation)
}

// NoEscape is a reusable TypeDescriptor annotation for parameters that
// the callee reads but never stores — safe to back with stack-allocated !list.
var NoEscape = &TypeDescriptor{Kind: "any", Escape: false}

var declaration_titles []string
var declarations map[string]*Declaration = make(map[string]*Declaration)
var declarations_hash map[string]*Declaration = make(map[string]*Declaration)

func DeclareTitle(title string) {
	declaration_titles = append(declaration_titles, "#"+title)
}

// returnTypeAnnotations stores rich return-type info for declared functions.
// The optimizer queries this to determine ownership transfer (e.g., fresh allocation).
var returnTypeAnnotations = map[string]*TypeDescriptor{}

// FreshAlloc is a reusable TypeDescriptor for functions whose return value
// is always a fresh allocation — safe for _mut swap by the optimizer.
var FreshAlloc = &TypeDescriptor{Kind: "list", Transfer: true}

// AnnotateReturnType attaches a return TypeDescriptor to named functions.
// Use FreshAlloc for functions that always return freshly allocated values.
func AnnotateReturnType(td *TypeDescriptor, names ...string) {
	for _, name := range names {
		returnTypeAnnotations[name] = td
	}
}

// ReturnTypeOf returns the return TypeDescriptor for a declared function, or nil.
func ReturnTypeOf(name string) *TypeDescriptor {
	return returnTypeAnnotations[name]
}

func Declare(env *Env, def *Declaration) {
	if !def.Forbidden {
		declaration_titles = append(declaration_titles, def.Name)
	}
	declarations[def.Name] = def
	if def.Fn != nil {
		declarations_hash[fmt.Sprintf("%p", def.Fn)] = def
		env.Vars[Symbol(def.Name)] = NewFunc(def.Fn)
	}
}

// slugify makes a filesystem-safe, lowercase slug from a chapter title.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Replace spaces with dashes
	s = strings.ReplaceAll(s, " ", "-")
	// Keep only a–z, 0–9, -, _
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		out = "chapter"
	}
	return out
}

// WriteDocumentation generates Markdown docs:
// - index.md with links to chapters
// - one <chapter>.md file per chapter, containing all functions of that chapter
func WriteDocumentation(folder string) error {
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return fmt.Errorf("failed to create folder %q: %w", folder, err)
	}

	type Chapter struct {
		Title string
		Slug  string
		Fns   []*Declaration
	}

	var chapters []*Chapter
	var current *Chapter

	// We’ll add a default "General" chapter if we see functions before any heading.
	defaultChapter := &Chapter{Title: "General", Slug: slugify("General")}
	usedSlugs := map[string]int{}

	uniqSlug := func(s string) string {
		base := slugify(s)
		if usedSlugs[base] == 0 {
			usedSlugs[base] = 1
			return base
		}
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s-%d", base, i)
			if usedSlugs[candidate] == 0 {
				usedSlugs[candidate] = 1
				return candidate
			}
		}
	}

	// Build chapter -> functions from the ordered declaration_titles
	for _, t := range declaration_titles {
		if len(t) > 0 && t[0] == '#' {
			title := strings.TrimSpace(t[1:])
			ch := &Chapter{Title: title, Slug: uniqSlug(title)}
			chapters = append(chapters, ch)
			current = ch
			continue
		}
		// function name
		def, ok := declarations[t]
		if !ok {
			// unknown entry — ignore gracefully
			continue
		}
		if current == nil {
			// First functions before any chapter title: create/use "General".
			if usedSlugs[defaultChapter.Slug] == 0 {
				usedSlugs[defaultChapter.Slug] = 1
				chapters = append(chapters, defaultChapter)
			}
			current = defaultChapter
		}
		current.Fns = append(current.Fns, def)
	}

	// Write index.md (chapters only)
	indexPath := filepath.Join(folder, "index.md")
	indexFile, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", indexPath, err)
	}
	defer indexFile.Close()

	fmt.Fprintln(indexFile, "# Documentation\n")
	for _, ch := range chapters {
		if len(ch.Fns) == 0 {
			// Skip empty chapters
			continue
		}
		fmt.Fprintf(indexFile, "- [%s](%s.md)\n", ch.Title, ch.Slug)
	}

	// Write one file per chapter
	for _, ch := range chapters {
		if len(ch.Fns) == 0 {
			continue
		}
		fp := filepath.Join(folder, ch.Slug+".md")
		f, err := os.Create(fp)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", fp, err)
		}

		// Chapter header
		fmt.Fprintf(f, "# %s\n\n", ch.Title)

		// Functions in this chapter
		for _, def := range ch.Fns {
			fmt.Fprintf(f, "## %s\n\n", def.Name)
			if def.Desc != "" {
				fmt.Fprintf(f, "%s\n\n", def.Desc)
			}
			fmt.Fprintf(f, "**Allowed number of parameters:** %d–%d\n\n", def.MinParameter, def.MaxParameter)

			fmt.Fprintln(f, "### Parameters\n")
			if len(def.Params) == 0 {
				fmt.Fprintln(f, "_This function has no parameters._\n")
			} else if d, ok := declarations[def.Name]; ok && !d.Forbidden {
				for _, p := range def.Params {
					fmt.Fprintf(f, "- **%s** (`%s`): %s\n", p.Name, p.Type, p.Desc)
				}
				fmt.Fprintln(f)
			}

			fmt.Fprintf(f, "### Returns\n\n`%s`\n\n", def.Returns)
		}

		_ = f.Close()
	}

	return nil
}

func types_match(given string, required string) bool {
	// handle type alternatives
	required_ := strings.Split(required, "|")
	given_ := strings.Split(given, "|")
	if len(required_) > 1 || len(given_) > 1 {
		for _, r := range required_ {
			for _, g := range given_ {
				if types_match(g, r) {
					return true // if any given fits any required, the value is allowed
				}
			}
		}
		return false
	}
	// single type comparison
	if given == required {
		return true // exact match
	}
	if given == "any" {
		return true // be graceful, we can't check it
	}
	if required == "any" {
		return true // this is always allowed
	}
	if given == "int" && required == "number" {
		return true // we allow int to number but not otherwise
	}
	// TODO: in case of func: compare signatures??
	// TODO: list(subtype)
	return false // not a single match
}

func types_merge(given, newtype string) string {
	if given == "" {
		return newtype
	}
	if types_match(given, newtype) {
		return given
	}
	if types_match(newtype, given) {
		return newtype
	}
	return given + "|" + newtype
}

// panics if the code is bad (returns possible datatype, at least "any")
func Validate(val Scmer, require string) string {
	var source_info SourceInfo
	if val.IsSourceInfo() {
		source_info = *val.SourceInfo()
		val = source_info.value
	}
	switch val.GetTag() {
	case tagNil:
		return "nil"
	case tagString:
		return "string"
	case tagSymbol:
		return "any"
	case tagFloat:
		return "number"
	case tagInt:
		return "int"
	case tagBool:
		return "bool"
	case tagFunc:
		return "func"
	case tagSlice:
		slice := val.Slice()
		if len(slice) == 0 {
			return "list"
		}
		if len(slice) > 0 {
			var def *Declaration
			head := slice[0]
			if head.IsSymbol() {
				if def2, ok := declarations[head.String()]; ok {
					def = def2
				}
			} else if head.GetTag() == tagFunc {
				if def2, ok := declarations_hash[fmt.Sprintf("%p", head.Func())]; ok {
					def = def2
				}
			}
			if def != nil {
				if len(slice)-1 < def.MinParameter {
					panic(source_info.String() + ": function " + def.Name + " expects at least " + fmt.Sprintf("%d", def.MinParameter) + " parameters")
				}
				if len(slice)-1 > def.MaxParameter {
					panic(source_info.String() + ": function " + def.Name + " expects at most " + fmt.Sprintf("%d", def.MaxParameter) + " parameters")
				}
			}
			skipFirst := slice[0].IsSymbol() && (slice[0].SymbolEquals("lambda") || slice[0].SymbolEquals("parser"))
			returntype := ""
			for i := 1; i < len(slice); i++ {
				if def != nil && def.Name == "match" && i >= 2 && i%2 == 0 {
					// pattern positions in (match) are not evaluated like regular function args; skip validation
					continue
				}
				if i != 1 || !skipFirst {
					subrequired := "any"
					isReturntype := false
					if def != nil {
						j := i - 1
						if j >= len(def.Params) {
							j = len(def.Params) - 1
						}
						subrequired = def.Params[j].Type
						if subrequired == "returntype" {
							subrequired = require
							isReturntype = true
						}
					}
					typ := Validate(slice[i], subrequired)
					if !types_match(typ, subrequired) {
						panic(fmt.Sprintf("%s: function %s expects parameter %d to be %s, but found value of type %s", source_info.String(), def.Name, i, subrequired, typ))
					}
					if isReturntype {
						returntype = types_merge(returntype, typ)
					}
				}
			}
			if def != nil {
				if def.Returns == "returntype" {
					if returntype == "" {
						panic("return returntype without returntype parameters")
					}
					return returntype
				}
				return def.Returns
			}
			return "any"
		}
	case tagFastDict:
		fd := val.FastDict()
		if fd == nil {
			return "list"
		}
		return Validate(NewSlice(fd.Pairs), require)
	case tagAny:
		if val.Any() == nil {
			return "nil"
		}
		if _, ok := val.Any().(func(...Scmer) Scmer); ok {
			return "func"
		}
	}
	return "any"
}

func Help(fn Scmer) {
	if fn.IsNil() {
		fmt.Println("Available scm functions:")
		for _, title := range declaration_titles {
			if title[0] == '#' {
				fmt.Println("")
				fmt.Println("-- " + title[1:] + " --")
			} else if d, ok := declarations[title]; ok && !d.Forbidden {
				fmt.Println("  " + title + ": " + strings.Split(d.Desc, "\n")[0])
			}
		}
		fmt.Println("")
		fmt.Println("get further information by typing (help \"functionname\") to get more info")
	} else {
		def := DeclarationForValue(fn)
		if def != nil {
			fmt.Println("Help for: " + def.Name)
			fmt.Println("===")
			fmt.Println("")
			fmt.Println(def.Desc)
			fmt.Println("")
			fmt.Println("Allowed nø of parameters: ", def.MinParameter, "-", def.MaxParameter)
			fmt.Println("")
			for _, p := range def.Params {
				fmt.Println(" - " + p.Name + " (" + p.Type + "): " + p.Desc)
			}
			fmt.Println("")
		} else {
			panic("function not found: " + String(fn))
		}
	}
}

// DeclarationForValue resolves a callable head (symbol or native func) to its Declaration.
func DeclarationForValue(v Scmer) *Declaration {
	switch v.GetTag() {
	case tagString:
		if d, ok := declarations[v.String()]; ok {
			return d
		}
	case tagSymbol:
		if d, ok := declarations[v.String()]; ok {
			return d
		}
	case tagFunc:
		if d, ok := declarations_hash[fmt.Sprintf("%p", v.Func())]; ok {
			return d
		}
	case tagAny:
		if s, ok := v.Any().(string); ok {
			if d, ok := declarations[s]; ok {
				return d
			}
		}
		if sym, ok := v.Any().(Symbol); ok {
			if d, ok := declarations[string(sym)]; ok {
				return d
			}
		}
		if fn, ok := v.Any().(func(...Scmer) Scmer); ok {
			if d, ok := declarations_hash[fmt.Sprintf("%p", fn)]; ok {
				return d
			}
		}
	}
	return nil
}
