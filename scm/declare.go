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

import "fmt"
import "io"
import "os"
import "path/filepath"
import "sort"
import "strings"

// Declaration describes a built-in or Scheme-defined function.
type Declaration struct {
	Name                     string
	Fn                       func(...Scmer) Scmer
	Type                     *TypeDescriptor
	RetainsCallArgs          bool // native result or state may retain the variadic argument array
	// Optimize owns declaration-specific rewrites. When set, the optimizer calls
	// it instead of the default argument optimization and post-processing path.
	Optimize                 func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor)
	OptimizeFirstArgTransfer bool // the optimizer hook can consume ownership of its first argument
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
	Kind           string                     // "any"|"string"|"number"|"int"|"bool"|"nil"|"symbol"|"func"|"list"|"assoc"
	NoEscape       bool                       // true = value will NOT outlive its scope (safe for stack alloc); default false = may escape (conservative)
	Transfer       bool                       // callee receives ownership, can mutate
	Const          bool                       // value is a compile-time constant; for func: safe to constant-fold
	Length         int                        // exact positive list/assoc length; -1 = unknown
	Optional       bool                       // for func params: parameter is optional
	Variadic       bool                       // for func params: last param accepts 0+ values
	Forbidden      bool                       // for func: optimizer-only, hidden from help
	HasSideEffects bool                       // for func: true = call has side effects, cannot be eliminated even if result unused
	Label          string                     // human-readable label at any nesting level
	Description    string                     // user-facing documentation at any nesting level
	Params         []*TypeDescriptor          // for Kind="func": parameter types
	Return         *TypeDescriptor            // for Kind="func": return type
	Keys           map[string]*TypeDescriptor // for Kind="assoc": per-key type info
	Element        *TypeDescriptor            // for Kind="list": element type
	// Optional JIT emitter for native code generation.
	JITEmit func(ctx *JITContext, args []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc
	// JITVirtualArgs lets an emitter consume the caller's argument array as SSA
	// data. Numbered parameters stay in their existing stack slots and constants
	// stay immediate until an operation actually needs to materialize them.
	JITVirtualArgs bool
	// JITInlineCallbacks is generated from the builtin's Go SSA. It permits the
	// declaration emitter to inline known lambdas only when callback results do
	// not currently cross the builtin's own control-flow merges.
	JITInlineCallbacks bool
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
	kind   uint8
	flags  uint8
	length int
	Extra  *TypeDescriptor // nil in common case; only allocated for sub-structure info
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

const UnknownLength = -1

func (ti TypeInfo) Transfer() bool { return ti.flags&FlagTransfer != 0 }
func (ti TypeInfo) Const() bool    { return ti.flags&FlagConst != 0 }
func (ti TypeInfo) Escape() bool   { return ti.flags&FlagEscape != 0 }
func (ti TypeInfo) Kind() uint8    { return ti.kind }
func (ti TypeInfo) Length() int {
	if ti.length <= 0 {
		return UnknownLength
	}
	return ti.length
}

func (ti TypeInfo) WithTransfer() TypeInfo {
	ti.flags |= FlagTransfer
	return ti
}
func (ti TypeInfo) WithoutTransfer() TypeInfo { ti.flags &^= FlagTransfer; return ti }
func (ti TypeInfo) WithConst() TypeInfo       { ti.flags |= FlagConst; return ti }
func (ti TypeInfo) WithoutConst() TypeInfo    { ti.flags &^= FlagConst; return ti }
func (ti TypeInfo) WithKind(k uint8) TypeInfo { ti.kind = k; return ti }
func (ti TypeInfo) WithLength(n int) TypeInfo {
	if n <= 0 {
		ti.length = UnknownLength
	} else {
		ti.length = n
	}
	return ti
}
func (ti TypeInfo) WithExtra(td *TypeDescriptor) TypeInfo { ti.Extra = td; return ti }

// MakeTypeInfo builds a TypeInfo from transfer/const bools (no heap allocation).
func MakeTypeInfo(transfer, constant bool) TypeInfo {
	ti := TypeInfo{length: UnknownLength}
	if transfer {
		ti.flags |= FlagTransfer
	}
	if constant {
		ti.flags |= FlagConst
	}
	return ti
}

// TypeInfoFromTD converts a *TypeDescriptor to a stack-allocated TypeInfo.
func TypeInfoFromTD(td *TypeDescriptor) TypeInfo {
	if td == nil {
		return TypeInfo{length: UnknownLength}
	}
	ti := TypeInfo{length: UnknownLength}
	if td.Transfer {
		ti.flags |= FlagTransfer
	}
	if td.Const {
		ti.flags |= FlagConst
	}
	switch td.Kind {
	case "string":
		ti.kind = KindString
	case "int":
		ti.kind = KindInt
	case "number":
		ti.kind = KindFloat
	case "bool":
		ti.kind = KindBool
	case "nil":
		ti.kind = KindNil
	case "symbol":
		ti.kind = KindSymbol
	case "func":
		ti.kind = KindFunc
	case "list":
		ti.kind = KindList
	case "assoc":
		ti.kind = KindAssoc
	}
	if td.Length > 0 {
		ti.length = td.Length
	}
	if td.Length > 0 || len(td.Params) > 0 || td.Return != nil || len(td.Keys) > 0 || td.Element != nil {
		ti.Extra = td
	}
	return ti
}

// ToTypeDescriptor converts to a heap-allocated TypeDescriptor (for APIs that need it).
func (ti TypeInfo) ToTypeDescriptor() *TypeDescriptor {
	if ti.kind == KindAny && ti.flags == 0 && ti.Extra == nil && ti.Length() == UnknownLength {
		return nil
	}
	td := &TypeDescriptor{Transfer: ti.Transfer(), Const: ti.Const(), NoEscape: !ti.Escape(), Length: ti.Length()}
	if ti.Extra != nil {
		*td = *ti.Extra
		td.Transfer = ti.Transfer()
		td.Const = ti.Const()
		td.NoEscape = !ti.Escape()
		td.Length = ti.Length()
	}
	if td.Kind == "" {
		td.Kind = ti.kindName()
	}
	return td
}

func (ti TypeInfo) kindName() string {
	switch ti.kind {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "number"
	case KindBool:
		return "bool"
	case KindNil:
		return "nil"
	case KindSymbol:
		return "symbol"
	case KindFunc:
		return "func"
	case KindList:
		return "list"
	case KindAssoc:
		return "assoc"
	default:
		return ""
	}
}

// NoEscape is a reusable TypeDescriptor annotation for parameters that
// the callee reads but never stores — safe to back with stack-allocated !list.
var NoEscape = &TypeDescriptor{Kind: "any", NoEscape: true, Length: UnknownLength}

var declaration_titles []string
var declarations map[string]*Declaration = make(map[string]*Declaration)

// Keying by the complete function identity preserves distinct closure contexts
// and avoids fmt.Sprintf allocating on every prepared callback lookup.
var declarationsByFunction map[uintptr]*Declaration = make(map[uintptr]*Declaration)

func DeclareTitle(title string) {
	declaration_titles = append(declaration_titles, "#"+title)
}

// FreshAlloc is a reusable TypeDescriptor for functions whose return value
// is always a fresh allocation — safe for _mut swap by the optimizer.
var FreshAlloc = &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}

func (d *Declaration) IsForbidden() bool {
	return d.Type != nil && d.Type.Forbidden
}

func (d *Declaration) IsFoldable() bool {
	return d.Type != nil && d.Type.Const
}

func Declare(env *Env, def *Declaration) {
	validateDeclaration(def)
	if !def.IsForbidden() {
		declaration_titles = append(declaration_titles, def.Name)
	}
	declarations[def.Name] = def
	if def.Fn != nil {
		declarationsByFunction[FunctionIdentity(def.Fn)] = def
		env.Vars[Symbol(def.Name)] = NewFunc(def.Fn)
	}
}

// DeclareInSection registers a declaration and inserts it at the end of an
// existing named section in the help index. If the section is not found,
// it falls back to a normal Declare (appending at the end).
func DeclareInSection(section string, env *Env, def *Declaration) {
	validateDeclaration(def)
	declarations[def.Name] = def
	if def.Fn != nil {
		declarationsByFunction[FunctionIdentity(def.Fn)] = def
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

func validateDeclaration(def *Declaration) {
	if def == nil || def.Type == nil || def.Type.Kind != "func" {
		panic("declaration requires a function TypeDescriptor")
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

const documentationPreambleStart = "<!-- BEGIN CHAPTER PREAMBLE -->"
const documentationPreambleEnd = "<!-- END CHAPTER PREAMBLE -->"

var documentationPreambles = map[string]string{
	"Arithmetic / Logic":               "Arithmetic and logic functions provide numeric operations, comparisons, type predicates, SQL truth handling, and the reduction primitives used by compiled expressions.",
	"Associative Lists / Dictionaries": "Associative-list functions build and transform key/value data in the functional Scheme runtime, including lookup, filtering, mapping, reduction, and structural indexing.",
	"Dashboard Metrics":                "Dashboard metrics expose process and HTTP activity for operational displays and diagnostics. They report observations from the current MemCP process, not a multi-node cluster.",
	"Date":                             "Date functions parse, format, compare, truncate, and calculate with SQL temporal values while preserving the declared SQL result type where required.",
	"General":                          "General functions are declarations that are available before a more specific documentation chapter is selected.",
	"IO":                               "IO functions provide controlled access to streams, files, environment data, HTTP helpers, serialization, and process-facing input or output facilities.",
	"JIT Compilation":                  "JIT functions inspect and request native compilation of supported Scheme procedures. Unsupported procedures retain interpreted semantics.",
	"Lists":                            "Lists are the primary immutable collection and code representation in MemCP Scheme. This chapter covers construction, traversal, transformation, reduction, and sequence generation.",
	"Parsers":                          "Parser functions construct and run composable packrat parsers used by the SQL frontends and other structured-input modules.",
	"SCM Builtins":                     "SCM builtins form the core Scheme language: evaluation, quoting, functions, control flow, type conversion, pattern matching, optimization, and execution support.",
	"Storage":                          "Storage functions manage databases, tables, columns, scans, indexes, computed data, and persistence. They are the low-level primitives used by generated SQL plans.",
	"Streams":                          "Stream functions adapt producers and consumers for text, compression, decompression, and incremental data processing without requiring one complete in-memory value.",
	"Strings":                          "String functions cover validation, Unicode-aware manipulation, matching, collation, encoding, hashing, JSON conversion, and SQL-compatible text operations.",
	"Sync":                             "Synchronization functions provide thread-safe sessions, promises, caches, locks, and coordination primitives for explicitly shared state in otherwise functional Scheme code.",
	"Timezone":                         "Timezone functions convert temporal values between zones and provide MySQL- and PostgreSQL-compatible current-time and timestamp operations.",
	"Transactions":                     "Transaction functions manage implicit and explicit transaction contexts, including cursor-stability execution and snapshot/OCC commit paths.",
	"Vectors":                          "Vector functions operate on numeric list representations for dot products and compatibility scoring modes. Validate dimensions and the exact mode semantics before treating a result as a mathematical distance.",
	"Window Functions":                 "Window and streaming helpers maintain ordered buffers, emit bounded results, and support the runtime machinery used by compiled window plans.",
}

func defaultDocumentationPreamble(title string) string {
	if preamble := strings.TrimSpace(documentationPreambles[title]); preamble != "" {
		return preamble
	}
	return fmt.Sprintf("This chapter documents the %s functions available in the current MemCP runtime.", title)
}

// readDocumentationPreamble keeps hand-written chapter introductions separate
// from generated function entries. Marker-delimited text may itself contain ##
// headings. Files generated before the markers were introduced retain the text
// between their # chapter heading and first ## function heading.
func readDocumentationPreamble(path string, title string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultDocumentationPreamble(title), nil
		}
		return "", err
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if start := strings.Index(content, documentationPreambleStart); start >= 0 {
		start += len(documentationPreambleStart)
		if end := strings.Index(content[start:], documentationPreambleEnd); end >= 0 {
			if preamble := strings.TrimSpace(content[start : start+end]); preamble != "" {
				return preamble, nil
			}
		}
	}

	heading := "# " + title
	if strings.HasPrefix(content, heading) {
		rest := strings.TrimLeft(content[len(heading):], "\n")
		if end := strings.Index(rest, "\n## "); end >= 0 {
			rest = rest[:end]
		}
		if preamble := strings.TrimSpace(rest); preamble != "" {
			return preamble, nil
		}
	}

	return defaultDocumentationPreamble(title), nil
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
		preamble, err := readDocumentationPreamble(fp, ch.Title)
		if err != nil {
			return fmt.Errorf("failed to preserve preamble from %s: %w", fp, err)
		}
		f, err := os.Create(fp)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", fp, err)
		}

		// Chapter header
		fmt.Fprintf(f, "# %s\n\n", ch.Title)
		fmt.Fprintf(f, "%s\n\n%s\n\n%s\n\n", documentationPreambleStart, preamble, documentationPreambleEnd)

		// Functions in this chapter
		for _, def := range ch.Fns {
			fmt.Fprintf(f, "## %s\n\n", def.Name)
			if def.Type.Description != "" {
				fmt.Fprintf(f, "%s\n\n", def.Type.Description)
			}
			fmt.Fprintf(f, "**Allowed number of parameters:** %d–%d\n\n", def.MinParams(), def.MaxParams())

			fmt.Fprintln(f, "### Parameters")
			if def.Type == nil || len(def.Type.Params) == 0 {
				fmt.Fprintln(f, "_This function has no parameters._")
			} else if d, ok := declarations[def.Name]; ok && !d.IsForbidden() {
				for _, p := range def.Type.Params {
					p.WriteDocumentation(f, 0)
				}
				fmt.Fprintln(f)
			}

			fmt.Fprintln(f, "### Returns")
			fmt.Fprintln(f)
			if def.Type.Return == nil {
				(&TypeDescriptor{Kind: "any"}).writeDocumentation(f, 0, "value")
			} else {
				def.Type.Return.writeDocumentation(f, 0, "value")
			}
			fmt.Fprintln(f)
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
	// TODO: list(subtype)
	return false // not a single match
}

// validateCallbackSignature checks whether a lambda literal matches the
// expected callback signature (parameter count). Returns "" on success
// or an error description.
func validateCallbackSignature(lambdaSlice []Scmer, expectedSig *TypeDescriptor, source_info SourceInfo) string {
	if expectedSig == nil || expectedSig.Kind != "func" || len(expectedSig.Params) == 0 {
		return ""
	}
	// lambdaSlice is (lambda (params...) body [numvars])
	if len(lambdaSlice) < 3 {
		return ""
	}
	if !lambdaSlice[0].IsSymbol() || !lambdaSlice[0].SymbolEquals("lambda") {
		return ""
	}
	paramList, ok := scmerSlice(lambdaSlice[1])
	if !ok {
		return ""
	}

	// Count expected required/max params
	expectedMin := 0
	expectedMax := 0
	hasVariadic := false
	for _, p := range expectedSig.Params {
		if p == nil {
			expectedMax++
			expectedMin++
			continue
		}
		if p.Variadic {
			hasVariadic = true
		} else if !p.Optional {
			expectedMin++
		}
		expectedMax++
	}

	lambdaParams := len(paramList)
	// In this Scheme dialect, excess arguments are silently ignored,
	// so a lambda with FEWER params than the caller provides is valid.
	// Only reject lambdas with MORE params than the caller will provide.
	if !hasVariadic && lambdaParams > expectedMax {
		return fmt.Sprintf("%s: callback provides at most %d arguments, but lambda declares %d parameters",
			source_info.String(), expectedMax, lambdaParams)
	}

	// Validate lambda body against expected return type
	if expectedSig.Return != nil && expectedSig.Return.Kind != "" && expectedSig.Return.Kind != "any" {
		// Recursively validate the body (last expression before optional numvars)
		bodyIdx := 2
		if len(lambdaSlice) > 3 {
			// has numvars suffix — body is at index 2
			bodyIdx = 2
		}
		if bodyIdx < len(lambdaSlice) {
			bodyType := Validate(lambdaSlice[bodyIdx], expectedSig.Return.Kind)
			if !types_match(bodyType, expectedSig.Return.Kind) {
				return fmt.Sprintf("%s: callback should return %s, but body has type %s",
					source_info.String(), expectedSig.Return.Kind, bodyType)
			}
		}
	}

	return ""
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
				if def2, ok := declarationsByFunction[FunctionIdentity(head.Func())]; ok {
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
					// Deep callback signature validation: if param expects func with Params,
					// and the argument is a lambda literal, validate arity and return type.
					if subrequired == "func" && def != nil && def.Type != nil {
						j := i - 1
						if j >= len(def.Type.Params) {
							j = len(def.Type.Params) - 1
						}
						if j >= 0 && j < len(def.Type.Params) && def.Type.Params[j] != nil {
							paramTD := def.Type.Params[j]
							if paramTD.Kind == "func" && len(paramTD.Params) > 0 {
								argVal := slice[i]
								if argVal.IsSourceInfo() {
									argVal = argVal.SourceInfo().value
								}
								if argSlice, ok2 := scmerSlice(argVal); ok2 && len(argSlice) >= 3 {
									if errMsg := validateCallbackSignature(argSlice, paramTD, source_info); errMsg != "" {
										panic(errMsg)
									}
								}
							}
						}
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

// FormatTypeSignature returns a compact, recursively rendered type signature.
func FormatTypeSignature(td *TypeDescriptor) string {
	if td == nil {
		return "any"
	}
	kind := td.Kind
	if kind == "" {
		kind = "any"
	}
	if hasTypeKind(kind, "list") && td.Element != nil {
		kind = replaceTypeKind(kind, "list", "list<"+FormatTypeSignature(td.Element)+">")
	}
	if !hasTypeKind(kind, "func") {
		return kind
	}
	var b strings.Builder
	funcSignature := strings.Builder{}
	funcSignature.WriteString("func(")
	for i, p := range td.Params {
		if i > 0 {
			funcSignature.WriteString(", ")
		}
		if p == nil {
			funcSignature.WriteString("any")
			continue
		}
		if p.Label != "" {
			funcSignature.WriteString(p.Label)
			funcSignature.WriteString(":")
		}
		funcSignature.WriteString(FormatTypeSignature(p))
		if p.Optional {
			funcSignature.WriteString("?")
		}
		if p.Variadic {
			funcSignature.WriteString("...")
		}
	}
	funcSignature.WriteString(")")
	if td.Return != nil {
		funcSignature.WriteString(" -> ")
		funcSignature.WriteString(FormatTypeSignature(td.Return))
	}
	b.WriteString(replaceTypeKind(kind, "func", funcSignature.String()))
	return b.String()
}

func hasTypeKind(kinds, wanted string) bool {
	for _, kind := range strings.Split(kinds, "|") {
		if kind == wanted {
			return true
		}
	}
	return false
}

func replaceTypeKind(kinds, wanted, replacement string) string {
	parts := strings.Split(kinds, "|")
	for i, kind := range parts {
		if kind == wanted {
			parts[i] = replacement
		}
	}
	return strings.Join(parts, "|")
}

// documentedTypeName keeps function signatures compact when their parameters
// and return type are rendered as a nested structure immediately below them.
func documentedTypeName(td *TypeDescriptor) string {
	if td == nil {
		return "any"
	}
	if hasTypeKind(td.Kind, "func") && (len(td.Params) > 0 || td.Return != nil) {
		if td.Kind == "" {
			return "func"
		}
		return td.Kind
	}
	return FormatTypeSignature(td)
}

// WriteDocumentation writes a Markdown list item for a type and recursively
// documents nested lists, assoc fields, callback parameters, and return types.
// depth controls the initial list indentation, using two spaces per level.
func (td *TypeDescriptor) WriteDocumentation(w io.Writer, depth int) {
	td.writeDocumentation(w, depth, "")
}

func (td *TypeDescriptor) writeDocumentation(w io.Writer, depth int, fallbackLabel string) {
	if td == nil {
		td = &TypeDescriptor{Kind: "any"}
	}
	label := td.Label
	if label == "" {
		label = fallbackLabel
	}
	indent := strings.Repeat("  ", depth)
	fmt.Fprint(w, indent+"- ")
	if label != "" {
		fmt.Fprintf(w, "**%s** ", label)
	}
	fmt.Fprintf(w, "(`%s`)", documentedTypeName(td))
	if td.Description != "" {
		fmt.Fprintf(w, ": %s", td.Description)
	}
	if td.Optional {
		fmt.Fprint(w, " _(optional)_")
	}
	if td.Variadic {
		fmt.Fprint(w, " _(variadic)_")
	}
	fmt.Fprintln(w)

	if hasTypeKind(td.Kind, "list") {
		if td.Element != nil {
			td.Element.writeDocumentation(w, depth+1, "elements")
		}
	}
	if hasTypeKind(td.Kind, "assoc") {
		keys := make([]string, 0, len(td.Keys))
		for key := range td.Keys {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			td.Keys[key].writeDocumentation(w, depth+1, key)
		}
	}
	if hasTypeKind(td.Kind, "func") {
		if len(td.Params) > 0 {
			fmt.Fprintln(w, indent+"  - **Parameters**")
			for _, param := range td.Params {
				param.writeDocumentation(w, depth+2, "parameter")
			}
		}
		if td.Return != nil {
			fmt.Fprintln(w, indent+"  - **Returns**")
			td.Return.writeDocumentation(w, depth+2, "value")
		}
	}
}

func (td *TypeDescriptor) writeHelp(w io.Writer, depth int, fallbackLabel string) {
	if td == nil {
		td = &TypeDescriptor{Kind: "any"}
	}
	label := td.Label
	if label == "" {
		label = fallbackLabel
	}
	indent := strings.Repeat("  ", depth)
	fmt.Fprint(w, indent+" - ")
	if label != "" {
		fmt.Fprint(w, label+" ")
	}
	fmt.Fprintf(w, "(%s)", documentedTypeName(td))
	if td.Description != "" {
		fmt.Fprint(w, ": "+td.Description)
	}
	if td.Optional {
		fmt.Fprint(w, " [optional]")
	}
	if td.Variadic {
		fmt.Fprint(w, " [variadic]")
	}
	fmt.Fprintln(w)

	if hasTypeKind(td.Kind, "list") && td.Element != nil {
		td.Element.writeHelp(w, depth+1, "elements")
	}
	if hasTypeKind(td.Kind, "assoc") {
		keys := make([]string, 0, len(td.Keys))
		for key := range td.Keys {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			td.Keys[key].writeHelp(w, depth+1, key)
		}
	}
	if hasTypeKind(td.Kind, "func") {
		if len(td.Params) > 0 {
			fmt.Fprintln(w, indent+"   Parameters:")
			for _, param := range td.Params {
				param.writeHelp(w, depth+2, "parameter")
			}
		}
		if td.Return != nil {
			fmt.Fprintln(w, indent+"   Returns:")
			td.Return.writeHelp(w, depth+2, "value")
		}
	}
}

func Help(fn Scmer) string {
	var b strings.Builder
	if fn.IsNil() {
		b.WriteString("Available scm functions:\n")
		for _, title := range declaration_titles {
			if title[0] == '#' {
				b.WriteString("\n-- " + title[1:] + " --\n")
			} else if d, ok := declarations[title]; ok && !d.IsForbidden() {
				b.WriteString("  " + title + ": " + strings.Split(d.Type.Description, "\n")[0] + "\n")
			}
		}
		b.WriteString("\nget further information by typing (help \"functionname\") to get more info\n")
	} else {
		def := DeclarationForValue(fn)
		if def != nil {
			b.WriteString("Help for: " + def.Name + "\n===\n\n")
			b.WriteString(def.Type.Description + "\n\n")
			b.WriteString(fmt.Sprintf("Allowed nø of parameters: %d-%d\n\n", def.MinParams(), def.MaxParams()))
			if def.Type != nil {
				for _, p := range def.Type.Params {
					if p != nil {
						p.writeHelp(&b, 0, "parameter")
					}
				}
			}
			if def.Type != nil && def.Type.Return != nil {
				b.WriteString("\nReturns:\n")
				def.Type.Return.writeHelp(&b, 0, "value")
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
		if d, ok := declarationsByFunction[FunctionIdentity(v.Func())]; ok {
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
			if d, ok := declarationsByFunction[FunctionIdentity(fn)]; ok {
				return d
			}
		}
	}
	return nil
}
