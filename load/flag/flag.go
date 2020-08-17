package flag

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

var (
	flagTag   = "flag"   // Expected flag struct tag name.
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
	helpTag   = "help" // Only used for encoding.
	ignoreTag = "ignore"
	sepTag    = "sep" // separator for slice values.

	defaultSep = "," // default separator for encoding/decoding slice values.
)

// newFlagSet creates a new flagset and sets the flags.
// The resulting flagset is useful for both getting the flag
// help page bytes and setting values runtime flags.
func newFlagSet(o Options, helpMsgs map[string]string, ignore []string, nGrps []*node.Nodes) (fs *flagSet, err error) {
	if helpMsgs == nil {
		helpMsgs = make(map[string]string)
	}

	fs = &flagSet{
		fs:         flag.NewFlagSet(os.Args[0], flag.ExitOnError),
		fGroups:    make([][]*Flag, 0),
		fNames:     make(map[string]bool),
		helpMsgs:   helpMsgs,
		ignore:     ignore,
		hlpPreTxt:  o.HlpPreText,
		hlpPostTxt: o.HlpPostText,
		hlpFunc:    o.HlpFunc,
	}

	if fs.hlpFunc == nil {
		fs.hlpFunc = fs.defGenHelp
	}

	for _, nGrp := range nGrps {
		err = fs.makeFlags(nGrp)
		if err != nil {
			return nil, err
		}
	}

	// Check that the user hasn't registered -help or -h.
	if fs.fNames["help"] || fs.fNames["h"] {
		return nil, errors.New("cannot use reserved flags 'help' or 'h'")
	}

	fs.registerHelpMenu()
	return fs, nil
}

type flagSet struct {
	fs         *flag.FlagSet
	fGroups    [][]*Flag
	fNames     map[string]bool
	helpMsgs   map[string]string // manually set or override help messages. Help messages can be long.
	ignore     []string          // list of field names to ignore -- useful for dynamically adjusting the flag list.
	hlpPreTxt  string
	hlpPostTxt string
	hlpFunc    GenHelpFunc
}

// SetHelp will override an existing field "help" value or create
// one if not provided as a tag value.
//
// Useful for dynamically created help messages or adding help messages
// to struct fields a user does not control.
//
// SetHelp must be called before "Load" to be effective.
func (fs *flagSet) setHelp(fName, helpMsg string) {
	fs.helpMsgs[fName] = helpMsg
}

func (fs *flagSet) registerHelpMenu() {
	helpMenu := fs.hlpFunc(fs.fGroups)

	fs.fs.Usage = func() {
		fmt.Fprint(os.Stderr, helpMenu)
	}
}

// flagSet registers flags to the underlying flagset and
// created underlying flags.
func (fs *flagSet) makeFlags(nGrp *node.Nodes) error {
	fGroup := make([]*Flag, 0)
	for _, n := range nGrp.List() {
		heritage := node.Parents(n, nGrp.Map())

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
		if getFlagTag(n) == "omitprefix" {
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}

		// Get alias if exists.
		_, alias := nodeFlagName(n)

		// "alias" must be no more than one character.
		if len(alias) > 1 {
			return errors.New("flag name alias '" + alias + "' must be one character")
		}

		f := &Flag{
			Name:  genFullName(n, heritage),
			Alias: alias,
			n:     n,
		}
		f.helpOverride = fs.helpMsgs[f.Name]

		// Check if is on ignore list.
		if fs.isIgnored(f.Name) {
			continue
		}

		// If the flag name or it's alias is already defined then return an error.
		if fs.fNames[f.Name] {
			return errors.New(fmt.Sprintf("flag name '%v' defined more than once", f.Name))
		}

		if fs.fNames[f.Alias] {
			return errors.New(fmt.Sprintf("flag alias '%v' defined more than once", f.Alias))
		}

		// Register name(s)
		fs.register(f)

		fGroup = append(fGroup, f)
	}

	fs.fGroups = append(fs.fGroups, fGroup)

	return nil
}

func (fs *flagSet) register(f *Flag) {
	fs.fNames[f.Name] = true
	if f.Alias != "" {
		fs.fNames[f.Alias] = true
	}

	// bools need to be registered separately because command
	// line behavior is different.
	if f.n.IsBool() {
		bp := f.n.FieldValue.Addr().Interface().(*bool)
		fs.fs.BoolVar(bp, f.Name, false, "")

		if f.Alias != "" {
			fs.fs.BoolVar(bp, f.Alias, false, "")
		}
		return
	}

	fs.fs.Var(f, f.Name, "")
	if f.Alias != "" {
		fs.fs.Var(f, f.Alias, "")
	}
}

func (fs *flagSet) isIgnored(fName string) bool {
	for _, v := range fs.ignore {
		if fName == v {
			return true
		}
	}

	return false
}

type Flag struct {
	Name         string // full flag name
	Alias        string // flag alias - if exists
	helpOverride string

	n *node.Node
}

// String implements flag.ValueBefore interface and gets
// the string value of the struct field.
func (f *Flag) String() string {
	return toStr(f.n)
}

// Set implements flag.ValueBefore interface and sets the
// struct field value.
func (f *Flag) Set(s string) error {
	return set(f.n, s)
}

func (f *Flag) Help() string {
	if f.helpOverride == "" {
		return f.n.GetTag(helpTag)
	}

	return f.helpOverride
}

// ValueType returns the string representation of the type
// in simple terms.
func (f *Flag) ValueType() string {
	return node.ValueType(f.n)
}

// set sets the field value. It takes into account
// special cases such as time.Time and slices.
//
// If 'flagVal' is empty then nothing is set and nil is returned.
func set(n *node.Node, flagVal string) error {
	if n == nil {
		return nil
	}

	if flagVal == "" {
		return nil
	}

	if n.IsTime() {
		_, err := n.SetTime(flagVal, n.GetTag(fmtTag))
		return err
	} else if n.IsSlice() {
		return n.SetSlice(splitSlice(flagVal, n.GetTag(sepTag), isFlagString(n)))
	}

	return n.SetFieldValue(flagVal)
}

// splitSlice splits a flag string.
// the 'isString' option reads in the values as possibly string quoted.
// the result is `"1"` is read in as `1` with the quotes stripped away
// before reading in the value.
//
// TODO: allow hook for a custom implementation of this function.
func splitSlice(flagValue string, sep string, isString bool) []string {
	if sep == "" {
		sep = defaultSep
	}

	// Trim brackets for bracket support.
	vals := strings.Split(strings.Trim(flagValue, "[]"), sep)

	// Trim out single and double quotes and spaces.
	for i := range vals {
		vals[i] = strings.TrimSpace(vals[i])
		if isString {
			// Strip away possible string quoted values.
			vals[i] = strings.Trim(vals[i], `"'`)
		}
	}

	return vals
}

// genFullName generates the full flag name including the prefix.
var genFullName = func(n *node.Node, heritage []*node.Node) (fullName string) {
	return genPrefix(append(heritage, n))
}

// genPrefix generates the flag name prefix.
//
// 'heritage' is expected to be ordered from most to least distant relative.
func genPrefix(heritage []*node.Node) (prefix string) {
	for _, hn := range heritage {
		flagName, _ := nodeFlagName(hn)
		if flagName == "" {
			continue
		}

		if prefix == "" {
			prefix = flagName
		} else {
			prefix += "-" + flagName
		}
	}

	return prefix
}

// nodeFlagName generates the flag name of the node. Does
// not include the prefix and optionally returns an alias name.
func nodeFlagName(n *node.Node) (name, alias string) {
	flagVal := getFlagTag(n)

	vals := strings.Split(flagVal, ",")
	if len(vals) > 1 {
		alias = vals[1]
	}
	name = vals[0]

	switch name {
	case "omitprefix":
		return "", ""
	case "":
		return util.ToKebab(n.FieldName()), ""
	default:
		return name, alias
	}
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
// - `flag:"-"`
func isIgnored(n *node.Node) bool {
	// "ignore" tag or "config" tag has ("ignore" value)
	if n.GetBoolTag(ignoreTag) ||
		n.GetTag(configTag) == "ignore" ||
		getFlagTag(n) == "-" {
		return true
	}

	return false
}

// getFlagTag returns the 'flag' tag value. It
// knows to exclude the supported ',string' suffix option if present.
func getFlagTag(n *node.Node) string {
	return strings.Replace(n.GetTag(flagTag), ",string", "", -1)
}

// isFlagString returns true when the flag tag value has the suffix ",string".
func isFlagString(n *node.Node) bool {
	return strings.HasSuffix(n.GetTag(flagTag), ",string")
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

// toStr handles the converting an existing/default field
// value to a string as it would be represented as an flag value.
//
// The value includes double quotes for fields with the ",string"
// flag tag suffix.
func toStr(n *node.Node) string {
	if n.IsTime() {
		return n.TimeString(n.GetTag(fmtTag))
	} else if n.IsSlice() {
		vals := n.SliceString()
		if isFlagString(n) {
			for i := range vals {
				vals[i] = `"` + vals[i] + `"`
			}
		}

		return `[` + strings.Join(vals, getSep(n)) + `]`
	}

	val := n.String()
	if isFlagString(n) {
		val = `"` + val + `"`
	}

	return val
}

type GenHelpFunc func([][]*Flag) string

func (fs *flagSet) defGenHelp(fGroups [][]*Flag) string {
	cols := 175
	helpMenu := strings.TrimRight(fs.hlpPreTxt, "\n") + "\n\n"

	for _, fg := range fGroups {
		buf := new(bytes.Buffer)
		lines := make([]string, 0, len(fg))

		maxlen := 0
		for _, f := range fg {
			line := ""
			if f.Alias != "" {
				line = fmt.Sprintf("  -%s, --%s", f.Alias, f.Name)
			} else {
				line = fmt.Sprintf("      --%s", f.Name)
			}

			varname, usage := UnquoteUsage(f)
			if varname != "" {
				line += " " + varname
			}

			// This special character will be replaced with spacing once the
			// correct alignment is calculated
			line += "\x00"
			if len(line) > maxlen {
				maxlen = len(line)
			}

			valueType := f.ValueType()
			defValue := f.String()
			if f.n.IsTime() {
				fmtV := f.n.GetTag(fmtTag)
				if fmtV == "" {
					fmtV = node.NormTimeFormat("")
				}
				fmtV = "fmt: " + fmtV

				if usage == "" {
					usage = fmtV
				} else {
					usage = usage + " " + fmtV
				}
			}
			line += usage
			if usage != "" && defValue != "" {
				line += " "
			}
			if !isZeroValue(valueType, defValue) {
				if valueType == "string" {
					line += fmt.Sprintf("(default: %q)", defValue)
				} else {
					line += fmt.Sprintf("(default: %s)", defValue)
				}
			}

			lines = append(lines, line)
		}

		for _, line := range lines {
			sidx := strings.Index(line, "\x00")
			spacing := strings.Repeat(" ", maxlen-sidx)
			// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
			fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
		}

		helpMenu += buf.String() + "\n"
	}

	return helpMenu + fs.hlpPostTxt
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

// Splits the string `s` on whitespace into an initial substring up to
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

func isZeroValue(valueType, strValue string) bool {
	switch valueType {
	case "bool":
		return strValue == "false"
	case "int", "uint", "float":
		return strValue == "0"
	case "string", "time":
		return strValue == ""
	case "bools", "durations", "ints", "uints", "strings":
		return strValue == "[]" || strValue == ""
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return strValue == "0" || strValue == "0s"
	default:
		return false
	}
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for a flag and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the flag's value, or the empty string if the flag is boolean.
func UnquoteUsage(f *Flag) (name string, usage string) {
	// Look for a back-quoted name, but avoid the strings package.
	usage = f.Help()
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break // Only one back quote; use type name.
		}
	}

	return f.ValueType(), usage
}
