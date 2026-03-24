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

// Declaration describes a built-in or Scheme-defined function.
type Declaration struct {
	Name string
	Desc string
	Fn   func(...Scmer) Scmer
	Type *TypeDescriptor
}

// MinParams returns the minimum number of required parameters.
func (d *Declaration) MinParams() int {
	if d.Type == nil {
		return 0
	}
	count := 0
	for _, p := range d.Type.Params {
		if p != nil && !p.Optional && !p.Variadic {
			count++
		}
	}
	return count
}

// MaxParams returns the maximum number of parameters (10000 if variadic).
func (d *Declaration) MaxParams() int {
	if d.Type == nil {
		return 0
	}
	for _, p := range d.Type.Params {
		if p != nil && p.Variadic {
			return 10000
		}
	}
	return len(d.Type.Params)
}

// TypeDescriptor describes the type of any Scmer value at arbitrary depth.
// Uses pointers throughout — nil means "unknown / don't care" (conservative).
type TypeDescriptor struct {
	Kind      string                     // "any"|"string"|"number"|"int"|"bool"|"nil"|"symbol"|"func"|"list"|"assoc"
	NoEscape  bool                       // true = value will NOT outlive its scope (safe for stack alloc); default false = may escape (conservative)
	Transfer  bool                       // callee receives ownership, can mutate
	Const     bool                       // value is a compile-time constant; for func: safe to constant-fold
	Optional  bool                       // for func params: parameter is optional
	Variadic  bool                       // for func params: last param accepts 0+ values
	Forbidden      bool                  // for func: optimizer-only, hidden from help
	HasSideEffects bool                  // for func: true = call has side effects, cannot be eliminated even if result unused
	ParamName string                     // for func params: documentation name
	ParamDesc string                     // for func params: documentation description
	Params    []*TypeDescriptor          // for Kind="func": parameter types
	Return    *TypeDescriptor            // for Kind="func": return type
	Keys      map[string]*TypeDescriptor // for Kind="assoc": per-key type info
	Element   *TypeDescriptor            // for Kind="list": element type
	// Custom optimizer hook for function types. When set, the optimizer calls this
	// INSTEAD of the default arg optimization + post-processing.
	Optimize  func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor)
	// Optional JIT emitter for native code generation.
	JITEmit   func(ctx *JITContext, args []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc
	// Specialized variants keyed by param-ownership bitmask.
	// Built on-demand by the optimizer when a call site provides owned args.
	// TODO: deoptimization — if the global function is redefined, all callsites
	// referencing cached variants must be invalidated (reset to the original code).
	Variants  map[uint64]Scmer
}

// OptimizerContext is an exported wrapper so packages like storage can use
// optimizer hooks without importing unexported optimizerMetainfo.
type OptimizerContext struct {
	Env *Env
	Ome *optimizerMetainfo
}

// TypeInfo is a compact, stack-allocated type descriptor returned by OptimizeEx.
// No heap allocation for the common case (Kind + Flags). Extra info (sub-structure
// types, function signatures) is stored in an optional *TypeDescriptor pointer.
type TypeInfo struct {
	kind  uint8
	flags uint8
	Extra *TypeDescriptor // nil in common case; only allocated for sub-structure info
}

// Kind constants for TypeInfo
const (
	KindAny    uint8 = iota // 0: unknown
	KindString              // 1
	KindInt                 // 2
	KindFloat               // 3
	KindBool                // 4
	KindNil                 // 5
	KindSymbol              // 6
	KindFunc                // 7
	KindList                // 8
	KindAssoc               // 9
)

// Flag bits for TypeInfo
const (
	FlagTransfer uint8 = 1 << iota // callee receives ownership
	FlagConst                      // compile-time constant
	FlagEscape                     // value may outlive scope
)

func (ti TypeInfo) Transfer() bool { return ti.flags&FlagTransfer != 0 }
func (ti TypeInfo) Const() bool    { return ti.flags&FlagConst != 0 }
func (ti TypeInfo) Escape() bool   { return ti.flags&FlagEscape != 0 }
func (ti TypeInfo) Kind() uint8    { return ti.kind }

func (ti TypeInfo) WithTransfer() TypeInfo { ti.flags |= FlagTransfer; return ti }
func (ti TypeInfo) WithConst() TypeInfo    { ti.flags |= FlagConst; return ti }
func (ti TypeInfo) WithKind(k uint8) TypeInfo { ti.kind = k; return ti }

// ToTypeDescriptor converts to a heap-allocated TypeDescriptor (for APIs that need it).
func (ti TypeInfo) ToTypeDescriptor() *TypeDescriptor {
	if ti.kind == KindAny && ti.flags == 0 && ti.Extra == nil {
		return nil
	}
	td := &TypeDescriptor{Transfer: ti.Transfer(), Const: ti.Const(), NoEscape: !ti.Escape()}
	if ti.Extra != nil {
		*td = *ti.Extra
		td.Transfer = ti.Transfer()
		td.Const = ti.Const()
		td.NoEscape = !ti.Escape()
	}
	return td
}

// NoEscape is a reusable TypeDescriptor annotation for parameters that
// the callee reads but never stores — safe to back with stack-allocated !list.
var NoEscape = &TypeDescriptor{Kind: "any", NoEscape: true}

var declaration_titles []string
var declarations map[string]*Declaration = make(map[string]*Declaration)
var declarations_hash map[string]*Declaration = make(map[string]*Declaration)

// globalFuncTypeInfo stores optimizer type info for Scheme-defined functions.
// Persists across import boundaries so cross-file ownership propagation works.
// Not used for arity checks or documentation — only for optimizer type inference.
var globalFuncTypeInfo = make(map[Symbol]TypeInfo)

func DeclareTitle(title string) {
	declaration_titles = append(declaration_titles, "#"+title)
}

// FreshAlloc is a reusable TypeDescriptor for functions whose return value
// is always a fresh allocation — safe for _mut swap by the optimizer.
var FreshAlloc = &TypeDescriptor{Kind: "list", Transfer: true}

func (d *Declaration) IsForbidden() bool {
	return d.Type != nil && d.Type.Forbidden
}

func (d *Declaration) IsFoldable() bool {
	return d.Type != nil && d.Type.Const
}

func Declare(env *Env, def *Declaration) {
	if !def.IsForbidden() {
		declaration_titles = append(declaration_titles, def.Name)
	}
	declarations[def.Name] = def
	if def.Fn != nil {
		declarations_hash[fmt.Sprintf("%p", def.Fn)] = def
		env.Vars[Symbol(def.Name)] = NewFunc(def.Fn)
	}
}

// DeclareInSection registers a declaration and inserts it at the end of an
// existing named section in the help index. If the section is not found,
// it falls back to a normal Declare (appending at the end).
func DeclareInSection(section string, env *Env, def *Declaration) {
	declarations[def.Name] = def
	if def.Fn != nil {
		declarations_hash[fmt.Sprintf("%p", def.Fn)] = def
		env.Vars[Symbol(def.Name)] = NewFunc(def.Fn)
	}
	if def.IsForbidden() {
		return
	}
	// find the position right before the next section header after sectionName
	insertAt := -1
	inSection := false
	for i, t := range declaration_titles {
		if t == "#"+section {
			inSection = true
		} else if inSection && len(t) > 0 && t[0] == '#' {
			insertAt = i
			break
		}
	}
	if inSection {
		insertAt = len(declaration_titles)
	}
	if insertAt < 0 {
		declaration_titles = append(declaration_titles, def.Name)
		return
	}
	declaration_titles = append(declaration_titles[:insertAt], append([]string{def.Name}, declaration_titles[insertAt:]...)...)
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

	fmt.Fprintln(indexFile, "# Documentation")
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
			fmt.Fprintf(f, "**Allowed number of parameters:** %d–%d\n\n", def.MinParams(), def.MaxParams())

			fmt.Fprintln(f, "### Parameters")
			if def.Type == nil || len(def.Type.Params) == 0 {
				fmt.Fprintln(f, "_This function has no parameters._")
			} else if d, ok := declarations[def.Name]; ok && !d.IsForbidden() {
				for _, p := range def.Type.Params {
					fmt.Fprintf(f, "- **%s** (`%s`): %s\n", p.ParamName, p.Kind, p.ParamDesc)
				}
				fmt.Fprintln(f)
			}

			retKind := "any"
			if def.Type != nil && def.Type.Return != nil && def.Type.Return.Kind != "" {
				retKind = def.Type.Return.Kind
			}
			fmt.Fprintf(f, "### Returns\n\n`%s`\n\n", retKind)
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
				if len(slice)-1 < def.MinParams() {
					panic(source_info.String() + ": function " + def.Name + " expects at least " + fmt.Sprintf("%d", def.MinParams()) + " parameters")
				}
				if len(slice)-1 > def.MaxParams() {
					panic(source_info.String() + ": function " + def.Name + " expects at most " + fmt.Sprintf("%d", def.MaxParams()) + " parameters")
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
					if def != nil && def.Type != nil {
						j := i - 1
						if j >= len(def.Type.Params) {
							j = len(def.Type.Params) - 1
						}
						if j >= 0 && j < len(def.Type.Params) && def.Type.Params[j] != nil {
							subrequired = def.Type.Params[j].Kind
							if subrequired == "" {
								subrequired = "any"
							}
						}
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
				retKind := "any"
				if def.Type != nil && def.Type.Return != nil && def.Type.Return.Kind != "" {
					retKind = def.Type.Return.Kind
				}
				if retKind == "returntype" {
					if returntype == "" {
						panic("return returntype without returntype parameters")
					}
					return returntype
				}
				return retKind
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

func Help(fn Scmer) string {
	var b strings.Builder
	if fn.IsNil() {
		b.WriteString("Available scm functions:\n")
		for _, title := range declaration_titles {
			if title[0] == '#' {
				b.WriteString("\n-- " + title[1:] + " --\n")
			} else if d, ok := declarations[title]; ok && !d.IsForbidden() {
				b.WriteString("  " + title + ": " + strings.Split(d.Desc, "\n")[0] + "\n")
			}
		}
		b.WriteString("\nget further information by typing (help \"functionname\") to get more info\n")
	} else {
		def := DeclarationForValue(fn)
		if def != nil {
			b.WriteString("Help for: " + def.Name + "\n===\n\n")
			b.WriteString(def.Desc + "\n\n")
			b.WriteString(fmt.Sprintf("Allowed nø of parameters: %d-%d\n\n", def.MinParams(), def.MaxParams()))
			if def.Type != nil {
				for _, p := range def.Type.Params {
					if p != nil {
						b.WriteString(" - " + p.ParamName + " (" + p.Kind + "): " + p.ParamDesc + "\n")
					}
				}
			}
			b.WriteString("\n")
		} else {
			panic("function not found: " + String(fn))
		}
	}
	return b.String()
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
