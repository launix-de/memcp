/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "io"
import "sort"
import "bytes"
import "context"
import "strings"
import "encoding/xml"

// XMLCodecFailure describes an actual codec failure. Its language value is a
// generic diagnostic, never a transaction outcome or a SQL error number.
type XMLCodecFailure struct {
	Kind    string
	Message string
}

func (failure *XMLCodecFailure) Error() string { return "XML " + failure.Kind + ": " + failure.Message }
func (failure *XMLCodecFailure) OriginalFailure() any {
	return NewSlice([]Scmer{NewString("xml-codec-error"), NewString(failure.Kind), NewString(failure.Message)})
}

func xmlFailure(kind, message string) error { return &XMLCodecFailure{kind, message} }

type xmlContextReader struct {
	context context.Context
	reader  *strings.Reader
}

func (reader *xmlContextReader) Read(buffer []byte) (int, error) {
	if err := reader.context.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

type xmlNamespaceBinding struct {
	prefix   string
	previous string
	existed  bool
}

type xmlFrame struct {
	raw          xml.Name
	expanded     xml.Name
	declarations []xmlNamespaceBinding
	preserve     bool
}

type xmlDocumentInfo struct{ encoding string }

const xmlNamespaceURI = "http://www.w3.org/XML/1998/namespace"

func xmlDeclaration(data []byte) (xmlDocumentInfo, error) {
	// XML declarations contain pseudo-attributes. Use the real attribute parser
	// rather than a regular expression or a second home-grown XML tokenizer.
	decoder := xml.NewDecoder(strings.NewReader("<declaration " + string(data) + "/>"))
	token, err := decoder.Token()
	if err != nil {
		return xmlDocumentInfo{}, xmlFailure("declaration", err.Error())
	}
	start, ok := token.(xml.StartElement)
	if !ok {
		return xmlDocumentInfo{}, xmlFailure("declaration", "invalid declaration")
	}
	info := xmlDocumentInfo{}
	seen := make(map[string]bool)
	for index, attribute := range start.Attr {
		if attribute.Name.Space != "" || seen[attribute.Name.Local] {
			return info, xmlFailure("declaration", "duplicate or qualified declaration attribute")
		}
		seen[attribute.Name.Local] = true
		switch attribute.Name.Local {
		case "version":
			if index != 0 || attribute.Value != "1.0" {
				return info, xmlFailure("declaration", "unsupported XML version")
			}
		case "encoding":
			if !seen["version"] || seen["standalone"] {
				return info, xmlFailure("declaration", "invalid declaration attribute order")
			}
			info.encoding = attribute.Value
		case "standalone":
			if attribute.Value != "yes" && attribute.Value != "no" {
				return info, xmlFailure("declaration", "invalid standalone declaration")
			}
		default:
			return info, xmlFailure("declaration", "unknown declaration attribute")
		}
	}
	if !seen["version"] {
		return info, xmlFailure("declaration", "missing XML version")
	}
	end, err := decoder.Token()
	if err != nil {
		return info, xmlFailure("declaration", "invalid declaration attributes")
	}
	if element, ok := end.(xml.EndElement); !ok || element.Name != start.Name {
		return info, xmlFailure("declaration", "invalid declaration attributes")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return info, xmlFailure("declaration", "unexpected declaration content")
	}
	return info, nil
}

func xmlQualifiedName(name xml.Name) xml.Name {
	if name.Space == "" {
		return name
	}
	return xml.Name{Local: name.Space + ":" + name.Local}
}

func xmlExpandedName(name xml.Name, namespaces map[string]string, attribute bool) (xml.Name, error) {
	if strings.Contains(name.Local, ":") {
		return xml.Name{}, xmlFailure("namespace", "invalid qualified name")
	}
	if name.Space == "" && attribute {
		return name, nil
	}
	uri, found := namespaces[name.Space]
	if name.Space != "" && !found {
		return xml.Name{}, xmlFailure("namespace-unbound", "undeclared namespace prefix")
	}
	return xml.Name{Space: uri, Local: name.Local}, nil
}

// xmlWalk consumes Unicode XML CONTENT, including multiple top-level elements.
// No external entity resolver exists. Whitespace policy is supplied by the
// caller; names and namespace bindings are validated before a token is emitted.
// Namespace state uses one active map with stack-owned declaration undo, so
// nesting does not retain a copy of every ancestor namespace map. Consumer
// arguments are synchronous borrowed tokenizer state and cannot escape.
func xmlWalk(text string, preserve bool, scope Scmer, consume func(xml.Token, []xmlFrame, map[string]string) error) (xmlDocumentInfo, error) {
	ctx := executionContextFrom(scope)
	if ctx == nil {
		ctx = context.Background()
	}
	decoder := xml.NewDecoder(&xmlContextReader{ctx, strings.NewReader(strings.TrimPrefix(text, "\ufeff"))})
	// The input is already Unicode, not bytes in the declaration's encoding.
	// The caller can inspect that label separately to enforce its input policy.
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	var frames []xmlFrame
	namespaces := map[string]string{"xml": xmlNamespaceURI}
	info := xmlDocumentInfo{}
	seenToken := false
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			if len(frames) != 0 {
				return info, xmlFailure("unexpected-end", "missing end tag")
			}
			return info, nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return info, ctx.Err()
			}
			return info, xmlFailure("syntax", err.Error())
		}
		switch value := token.(type) {
		case xml.ProcInst:
			if strings.EqualFold(value.Target, "xml") {
				if value.Target != "xml" || seenToken {
					return info, xmlFailure("declaration", "XML declaration must be first")
				}
				info, err = xmlDeclaration(value.Inst)
				if err != nil {
					return info, err
				}
				seenToken = true
				continue // encoding is not retained in the Unicode representation
			}
		case xml.Directive:
			if strings.HasPrefix(string(value), "DOCTYPE") {
				return info, xmlFailure("document-type", "document type directives are not supported by this codec")
			}
			return info, xmlFailure("directive", "unknown XML directive")
		case xml.StartElement:
			var bindings []xmlNamespaceBinding
			declared := make(map[string]bool)
			for _, attribute := range value.Attr {
				prefix := ""
				if attribute.Name.Space == "xmlns" {
					prefix = attribute.Name.Local
				} else if attribute.Name.Space != "" || attribute.Name.Local != "xmlns" {
					continue
				}
				if declared[prefix] {
					return info, xmlFailure("duplicate-attribute", "duplicate namespace declaration")
				}
				declared[prefix] = true
				if prefix == "xml" && attribute.Value != xmlNamespaceURI || prefix != "xml" && attribute.Value == xmlNamespaceURI {
					return info, xmlFailure("namespace-reserved", "invalid reserved XML namespace binding")
				}
				if prefix == "xmlns" || attribute.Value == "http://www.w3.org/2000/xmlns/" || prefix != "" && attribute.Value == "" {
					return info, xmlFailure("namespace", "invalid namespace binding")
				}
				previous, existed := namespaces[prefix]
				bindings = append(bindings, xmlNamespaceBinding{prefix, previous, existed})
				namespaces[prefix] = attribute.Value
			}
			expanded, err := xmlExpandedName(value.Name, namespaces, false)
			if err != nil {
				return info, err
			}
			preserveSpace := preserve
			if len(frames) != 0 {
				preserveSpace = frames[len(frames)-1].preserve
			}
			attributes := make(map[xml.Name]bool)
			for i, attribute := range value.Attr {
				if attribute.Name.Space != "xmlns" && !(attribute.Name.Space == "" && attribute.Name.Local == "xmlns") {
					name, err := xmlExpandedName(attribute.Name, namespaces, true)
					if err != nil {
						return info, err
					}
					if attributes[name] {
						return info, xmlFailure("duplicate-attribute", "duplicate expanded attribute name")
					}
					attributes[name] = true
					if name.Space == xmlNamespaceURI && name.Local == "space" {
						switch attribute.Value {
						case "preserve":
							preserveSpace = true
						case "default":
							preserveSpace = preserve
						default:
							return info, xmlFailure("attribute-value", "invalid xml:space attribute")
						}
					}
				}
				value.Attr[i].Name = xmlQualifiedName(attribute.Name)
			}
			frames = append(frames, xmlFrame{value.Name, expanded, bindings, preserveSpace})
			value.Name = xmlQualifiedName(value.Name)
			token = value
		case xml.EndElement:
			if len(frames) == 0 || frames[len(frames)-1].raw != value.Name {
				return info, xmlFailure("mismatched-end", "end tag does not match start tag")
			}
			value.Name = xmlQualifiedName(value.Name)
			token = value
		case xml.CharData:
			preserveSpace := preserve
			if len(frames) != 0 {
				preserveSpace = frames[len(frames)-1].preserve
			}
			if !preserveSpace && strings.Trim(string(value), " \t\r\n") == "" {
				seenToken = true
				continue
			}
		}
		seenToken = true
		if consume != nil {
			if err := consume(token, frames, namespaces); err != nil {
				return info, err
			}
		}
		if _, end := token.(xml.EndElement); end {
			for _, binding := range frames[len(frames)-1].declarations {
				if binding.existed {
					namespaces[binding.prefix] = binding.previous
				} else {
					delete(namespaces, binding.prefix)
				}
			}
			frames = frames[:len(frames)-1]
		}
	}
}

func xmlStringArgument(value Scmer) string {
	if !value.IsString() {
		panic("XML codec requires a Unicode string")
	}
	return value.String()
}

func init_xml_functions() {
	DeclareTitle("XML")
	Declare(&Globalenv, &Declaration{Name: "xml_parse", Fn: func(a ...Scmer) Scmer {
		if len(a) != 3 || !a[1].IsBool() {
			panic("xml_parse requires Unicode text, preserve-whitespace and execution scope")
		}
		var output bytes.Buffer
		encoder := xml.NewEncoder(&output)
		_, err := xmlWalk(xmlStringArgument(a[0]), a[1].Bool(), a[2], func(token xml.Token, _ []xmlFrame, _ map[string]string) error { return encoder.EncodeToken(token) })
		if err != nil {
			panic(err)
		}
		if err := encoder.Flush(); err != nil {
			panic(err)
		}
		return NewString(output.String())
	}, Type: &TypeDescriptor{Kind: "func", Description: "validates Unicode XML CONTENT and returns its normalized Unicode serialization; document type directives are unsupported", Params: []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "bool", Label: "preserve_whitespace"}, {Kind: "any", Label: "scope"}}, Return: &TypeDescriptor{Kind: "string"}}})
	Declare(&Globalenv, &Declaration{Name: "xml_declared_encoding", Fn: func(a ...Scmer) Scmer {
		if len(a) != 2 {
			panic("xml_declared_encoding requires Unicode text and execution scope")
		}
		info, err := xmlWalk(xmlStringArgument(a[0]), true, a[1], nil)
		if err != nil {
			panic(err)
		}
		if info.encoding == "" {
			return NewNil()
		}
		return NewString(info.encoding)
	}, Type: &TypeDescriptor{Kind: "func", Description: "returns an XML declaration's encoding label from a validated Unicode document, or nil when absent", Params: []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "any", Label: "scope"}}, Return: &TypeDescriptor{Kind: "any"}}})
	Declare(&Globalenv, &Declaration{Name: "xml_string_value", Fn: func(a ...Scmer) Scmer {
		if len(a) != 2 {
			panic("xml_string_value requires a Unicode XML fragment and execution scope")
		}
		var output strings.Builder
		_, err := xmlWalk(xmlStringArgument(a[0]), true, a[1], func(token xml.Token, _ []xmlFrame, _ map[string]string) error {
			if text, ok := token.(xml.CharData); ok {
				_, _ = output.Write(text)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		return NewString(output.String())
	}, Type: &TypeDescriptor{Kind: "func", Description: "returns descendant character data in document order; attributes are not text content", Params: []*TypeDescriptor{{Kind: "string", Label: "fragment"}, {Kind: "any", Label: "scope"}}, Return: &TypeDescriptor{Kind: "string"}}})
	Declare(&Globalenv, &Declaration{Name: "xml_children_each", Fn: func(a ...Scmer) Scmer {
		if len(a) != 4 || !a[1].IsSlice() {
			panic("xml_children_each requires Unicode XML, QName steps, emitter and execution scope")
		}
		steps := make([]xml.Name, len(a[1].Slice()))
		for i, item := range a[1].Slice() {
			if !item.IsSlice() || len(item.Slice()) != 2 || !item.Slice()[0].IsString() || !item.Slice()[1].IsString() {
				panic("XML child steps are namespace URI/local-name pairs")
			}
			steps[i] = xml.Name{Space: item.Slice()[0].String(), Local: item.Slice()[1].String()}
		}
		ctx := executionContextFrom(a[3])
		if ctx == nil {
			ctx = context.Background()
		}
		var output *bytes.Buffer
		var encoder *xml.Encoder
		var count int64
		_, err := xmlWalk(xmlStringArgument(a[0]), true, a[3], func(token xml.Token, frames []xmlFrame, namespaces map[string]string) error {
			if _, start := token.(xml.StartElement); start && output == nil && len(frames) == len(steps) {
				matches := true
				for i := range steps {
					matches = matches && frames[i].expanded == steps[i]
				}
				if matches {
					output = new(bytes.Buffer)
					encoder = xml.NewEncoder(output)
					start := token.(xml.StartElement)
					declared := make(map[string]bool)
					for _, attribute := range start.Attr {
						if attribute.Name.Local == "xmlns" {
							declared[""] = true
						} else if strings.HasPrefix(attribute.Name.Local, "xmlns:") {
							declared[strings.TrimPrefix(attribute.Name.Local, "xmlns:")] = true
						}
					}
					prefixes := make([]string, 0, len(namespaces))
					for prefix := range namespaces {
						prefixes = append(prefixes, prefix)
					}
					sort.Strings(prefixes)
					for _, prefix := range prefixes {
						uri := namespaces[prefix]
						if prefix != "xml" && !declared[prefix] {
							name := "xmlns"
							if prefix != "" {
								name += ":" + prefix
							}
							start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: uri})
						}
					}
					token = start
				}
			}
			if output == nil {
				return nil
			}
			if err := encoder.EncodeToken(token); err != nil {
				return err
			}
			if _, end := token.(xml.EndElement); end && len(frames) == len(steps) {
				if err := encoder.Flush(); err != nil {
					return err
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				Apply(a[2], NewString(output.String()))
				count++
				output, encoder = nil, nil
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		return NewInt(count)
	}, Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "synchronously emits selected child-element XML fragments in document order without a result-row list", Params: []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "list", Label: "qname_steps"}, {Kind: "func", Label: "emit", Params: []*TypeDescriptor{{Kind: "string", Label: "fragment"}}, Return: &TypeDescriptor{Kind: "any"}}, {Kind: "any", Label: "scope"}}, Return: &TypeDescriptor{Kind: "int"}}})
}
