package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

var (
	configTag = "config"
	reqTag    = "req"
	showTag   = "show"
	fmtTag    = "fmt"
	ignoreTag = "ignore"
	helpTag   = "help"
	sepTag    = "sep"
	flagTag   = "flag"
	envTag    = "env"
	jsonTag   = "json"
	yamlTag   = "yaml"
	tomlTag   = "toml"

	defaultSep = ","
)

// New should be called before struct values values are populated as
// it marks the initial field value. The field value is marked again
// right before rendering so that the "ValueBefore" and "ValueAfter"
// Field values are correctly populated and rendered.
func New(o Options, nGrps []*node.Nodes) (*Renderer, error) {
	var err error
	r := &Renderer{
		preamble:   o.Preamble,
		conclusion: o.Postamble,
	}

	// field name generator.
	ng := defNameGenerator{}
	ng.nameFrom, ng.formatAs = parseFieldNameFormat(o.FieldNameFormat)
	r.nameFunc = ng.genFieldName

	// render func
	r.renderFunc = defaultRenderer
	if o.RenderFunc != nil {
		r.renderFunc = o.RenderFunc
	}

	// Create field groups.
	r.fGrps, err = r.fieldGroups(nGrps)
	if err != nil {
		return nil, err
	}

	// Record initial field values.
	r.recordVals()

	return r, nil
}

// Field represents a single field to render.
type Field struct {
	Name        string
	ValueBefore string
	ValueAfter  string
	Type        string
	Req         bool
	Show        bool
	TimeFmt     string // Effective 'fmt' value for time.Time fields.

	Node          *node.Node
	valueRecorded bool
}

// recordValue will record the string representation of
// the underlying value. The first time it's called
// the string value is written to "ValueBefore" and any subsequent
// call writes the value to "ValueAfter".
func (f *Field) recordValue() {
	if f.valueRecorded {
		f.ValueAfter = toStr(f.Node)
		return
	}

	f.ValueBefore = toStr(f.Node)
	f.valueRecorded = true
}

func (f *Field) IsZero(val string) bool {
	switch f.Type {
	case "bool":
		return val == "false"
	case "int", "uint", "float":
		return val == "0"
	case "string", "time":
		return val == ""
	case "bools", "durations", "ints", "uints", "strings":
		return val == "[]" || val == ""
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return val == "0" || val == "0s"
	default:
		return false
	}
}

type Options struct {
	// Preamble is optional and is used in the default renderer. The preamble is
	// prepended to the rendering.
	Preamble string

	// Postamble is optional and is used in the default renderer. It is appended
	// to the rendering.
	Postamble string

	// FieldNameFormat defines the name format used when representing the field name when
	// using the default name formatter.
	//
	// Options are:
	// - "env" // env value as SCREAMING_SNAKE_CASE.
	// - "flag" // flag value as kebab-case (with prepended dash).
	// - "json" // json value as snake_case.
	// - "toml" // toml value as snake_case.
	// - "yaml" // yaml value as lowercase.
	// - "field" // struct field name (Dot.Separated for embedded fields).
	// - "snake" // struct field name as snake_case.
	//
	// TODO: support the following:
	// - "screaming" // struct field name as SCREAMING_SNAKE_CASE.
	// - "kebab" // struct field name as kebab-case.
	//
	// Can also provide:
	// - "[env,flag,json,toml,yaml,field] as [snake,kebab,screaming,
	FieldNameFormat string

	// RenderFunc is optional and if provided overrides the default render
	// function. If a custom RenderFunc is provided then "Preamble" and "Postamble" are
	// not used.
	RenderFunc RenderFunc
}

type RenderFunc func(preamble, conclusion string, fieldGroups [][]*Field) []byte

type Renderer struct {
	preamble   string     // Prepended string used in the default renderer.
	conclusion string     // Appended string used in the default renderer.
	fGrps      [][]*Field // One field group per config struct passed.
	renderFunc RenderFunc
	nameFunc   func(n *node.Node, heritage []*node.Node) string
}

func (r *Renderer) Render() []byte {
	r.recordVals() // Record final string values.
	return r.renderFunc(r.preamble, r.conclusion, r.fGrps)
}

// recordVals records the current node string values.
// The first time it's called the "ValueBefore" string value
// is recorded. The second time it's called the "ValueAfter" value
// is recorded.
//
// recordVals should be called once before reading in config values
// and then again after the values have been reconciled.
func (r *Renderer) recordVals() {
	for _, fGrp := range r.fGrps {
		for _, f := range fGrp {
			f.recordValue()
		}
	}
}

// defaultRenderer is the default render function.
func defaultRenderer(preamble, conclusion string, fieldGroups [][]*Field) []byte {
	cols := 175
	buf := new(bytes.Buffer)

	if preamble != "" {
		fmt.Fprintln(buf, preamble)
	}

	for _, fg := range fieldGroups {
		lines := make([]string, 0, len(fg))
		maxlen := 0
		for _, f := range fg {
			line := ""

			// field name
			//
			// The special character will be replaced with spacing once the
			// correct alignment is calculated
			line += f.Name + " (" + f.Type + ")" + ":\x00"
			if len(line) > maxlen {
				maxlen = len(line)
			}

			// Resolved value.
			if f.Show {
				if f.Type == "string" {
					line += fmt.Sprintf("%q", f.ValueAfter)
				} else {
					line += fmt.Sprintf("%s", f.ValueAfter)
				}
			} else {
				line += "[redacted]"
			}

			// Default value.
			if !f.IsZero(f.ValueBefore) && f.Show {
				if f.Type == "string" {
					line += fmt.Sprintf(" (default: %q)", f.ValueBefore)
				} else {
					line += fmt.Sprintf(" (default: %s)", f.ValueBefore)
				}
			}

			// required
			if f.Req {
				line += " (required)"
			}

			lines = append(lines, line)
		}

		for _, line := range lines {
			sidx := strings.Index(line, "\x00")
			spacing := strings.Repeat(" ", maxlen-sidx)
			// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
			fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
		}

		fmt.Fprintln(buf, "") // New line between groups.
	}

	if conclusion != "" {
		fmt.Fprintln(buf, conclusion)
	}

	body := strings.TrimSpace(buf.String())

	return []byte("\r\n" + body + "\r\n")
}

// wrap wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(i, w int, s string) string {
	if w == 0 {
		return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(s, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = wrapN(wrap, slop, s)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", i), -1)

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = wrapN(wrap, slop, s)
		r = r + "\n" + strings.Repeat(" ", i) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	return r
}

// wrapN splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
func wrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

func (r *Renderer) fieldGroups(ngrps []*node.Nodes) ([][]*Field, error) {
	fgs := make([][]*Field, 0)
	for _, ngrp := range ngrps {
		fg, err := r.fieldGroup(ngrp)
		if err != nil {
			return nil, err
		}
		fgs = append(fgs, fg)
	}

	return fgs, nil
}

func (r *Renderer) fieldGroup(ngrp *node.Nodes) ([]*Field, error) {
	fg := make([]*Field, 0)
	for _, n := range ngrp.List() {
		heritage := node.Parents(n, ngrp.Map())

		// Check if ignored or any parent(s) are ignored.
		//
		// Note that if this node or any ancestor node is ignored
		// then the result is the same - this node is ignored.
		if isAnyIgnored(append(heritage, n)) {
			continue
		}

		// Skip fields that are themselves structs (excluding special structs like time.Time).
		//
		// Note: for now time.Time is treated specifically. At some point we want to key
		// off something like non-stringer structs.
		if n.IsStruct() && !n.IsTime() {
			continue
		}

		// Validate that "omitprefix" is not used on value fields.
		switch "omitprefix" {
		case n.GetTag(envTag), n.GetTag(flagTag), n.GetTag(tomlTag), n.GetTag(yamlTag), n.GetTag(jsonTag):
			return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}

		name := r.nameFunc(n, heritage)
		if name == "" {
			continue
		}
		fg = append(fg, &Field{
			Name:    name,
			Type:    node.ValueType(n),
			Req:     n.GetBoolTag(reqTag),
			Show:    isShown(n),
			TimeFmt: timeFmt(n),
			Node:    n,
		})
	}

	return fg, nil
}

// toStr handles the converting an existing/default field
// value to a generic string representation.
func toStr(n *node.Node) string {
	if n.IsTime() {
		return n.TimeString(n.GetTag(fmtTag))
	} else if n.IsSlice() {
		vals := n.SliceString()
		return `[` + strings.Join(vals, getSep(n)) + `]`
	}
	val := n.String()

	return val
}

// getSep returns the separator designed to be used for
// the struct field node if one is provided. If a separator is not
// provided then the default separator is returned.
func getSep(n *node.Node) string {
	sep := n.GetTag(sepTag)
	if sep == "" {
		sep = defaultSep
	}

	return sep
}

// isShown calculates if a node value is shown.
// If no value is present then defaults to "true".
// Otherwise takes the bool value of the "show" tag.
func isShown(n *node.Node) bool {
	show := n.GetTag(showTag)
	if show == "" {
		return true
	}

	return n.GetBoolTag(showTag)
}

func timeFmt(n *node.Node) string {
	fmtV := n.GetTag(fmtTag)
	if fmtV == "" {
		fmtV = node.NormTimeFormat("")
	}

	return fmtV
}

// parseFieldNameFormat
//
// fFmt general form:
// "{nameFrom}[ as {formatAs}]"
//
// nameFrom values:
// - "field" (name is based on the struct field - default)
// - "env" (name based on env input)
// - "flag" (name based on flag input)
// - "toml" (name based on toml input)
// - "yaml" (name based on yaml input)
// - "json" (name based on json input)
//
// formatAs values:
// - "screaming-snake"
// - "snake"
// - "camel"
// - "lower-camel"
// - "screaming-kebab"
// - "kebab"
// - "lower" (everything lowercase squished together)
// - "field" (name matches the struct field name)
//
// Examples:
//
// "" (empty)     -> "field", "field"
// "env"          -> "env", "screaming-snake"
// "flag"         -> "flag", "kebab"
// "toml"         -> "toml", "snake"
// "yaml"         -> "yaml", "lower"
// "json"         -> "json", "snake"
// "field"        -> "field", "field"
// "as snake"     -> "field", "snake"
// "env as snake" -> "env", "snake"
//
// Incorrect input format will result in the default values "field", "field".
func parseFieldNameFormat(fFmt string) (nameFrom, formatAs string) {
	vals := strings.Split(fFmt, "as")
	if len(vals) == 1 {
		nameFrom = strings.TrimSpace(vals[0])
	}
	if len(vals) == 2 {
		nameFrom = strings.TrimSpace(vals[0])
		formatAs = strings.TrimSpace(vals[1])
	}

	switch nameFrom {
	case "field", "env", "flag", "toml", "yaml", "json":
		break
	default:
		nameFrom = "field"
	}

	switch formatAs {
	case "screaming-snake", "snake", "camel", "lower-camel", "screaming-kebab", "kebab", "lower", "field":
		break
	case "":
		// derive formatAs from nameFrom.
		switch nameFrom {
		case "env":
			formatAs = "screaming-snake"
		case "flag":
			formatAs = "kebab"
		case "toml":
			formatAs = "snake"
		case "yaml":
			formatAs = "lower"
		case "json":
			formatAs = "snake" // as "camel" instead?
		default:
			formatAs = "field"
		}
	default:
		formatAs = "field"
	}

	return nameFrom, formatAs
}

// isAnyIgnored checks if any members of 'nodes' is ignored.
// if so, then returns true.
func isAnyIgnored(nodes []*node.Node) bool {
	for _, n := range nodes {
		if isIgnored(n) {
			return true
		}
	}

	return false
}

// isIgnored checks if the node is ignored.
//
// A node is ignored when one or more of the following struct
// field tag cases are met:
// - `ignore:"true"`
// - `config:"ignore"`
func isIgnored(n *node.Node) bool {
	// "ignore" tag or "config" tag has ("ignore" value)
	if n.GetBoolTag(ignoreTag) ||
		n.GetTag(configTag) == "ignore" {
		return true
	}

	return false
}

// defNameGenerator is the default field name generator struct.
type defNameGenerator struct {
	nameFrom string
	formatAs string
}

// genFieldName generates the string representation of the field name.
func (ng *defNameGenerator) genFieldName(n *node.Node, heritage []*node.Node) (fullName string) {
	return genBaseName(append(heritage, n), ng.nameFrom, ng.formatAs)
}

// genPrefix generates the env name prefix.
//
// 'heritage' is expected to be ordered from most to least distant relative.
func genBaseName(heritage []*node.Node, nameFrom, formatAs string) (prefix string) {
	del := "."
	switch formatAs {
	case "screaming-snake", "snake":
		del = "_"
	case "camel", "lower-camel", "lower":
		del = ""
	case "screaming-kebab", "kebab":
		del = "-"
	}

	for _, hn := range heritage {
		name := nodeName(hn, nameFrom, formatAs)
		if name == "" {
			continue
		}

		if prefix == "" {
			prefix = name
		} else {
			prefix += del + name
		}
	}

	return prefix
}

// nodeEnvName generates the name of the node. Does
// not include the prefix.
func nodeName(n *node.Node, nameFrom, formatAs string) string {
	name := ""
	switch nameFrom {
	case envTag, flagTag, tomlTag, yamlTag, jsonTag:
		name = n.GetTag(nameFrom)
		name = strings.Split(name, ",")[0] // handle cases with special "," options like ",string"
		if name == "-" {
			return ""
		}

		if len(name) > 0 {
			return name
		}
	}

	if name == "" {
		name = n.FieldName()
	}

	switch name {
	case "omitprefix", "-":
		return ""
	case "":
		return name
	default:
		switch formatAs {
		case "screaming-snake":
			return util.ToScreamingSnake(name)
		case "snake":
			return util.ToSnake(name)
		case "camel":
			return util.ToCamel(name)
		case "lower-camel":
			return util.ToLowerCamel(name)
		case "screaming-kebab":
			return util.ToScreamingKebab(name)
		case "kebab":
			return util.ToKebab(name)
		case "lower":
			return util.ToLower(name)
		}
	}

	return name
}
