package flg

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pcelvng/go-config/util/node"

	"github.com/pcelvng/go-config/util"

	"github.com/iancoleman/strcase"
)

func NewDecoder(helpBlock string) *Decoder {
	return &Decoder{
		fs:       flag.NewFlagSet(os.Args[0], flag.ExitOnError),
		fFields:  make([]*FlagField, 0),
		fGroups:  make([][]*FlagField, 0),
		ignore:   make([]string, 0),
		helpMsgs: make(map[string]string),
		defaults: make(map[string]string),
		helpMenu: helpBlock + "\n",
	}
}

type Decoder struct {
	fs       *flag.FlagSet
	fFields  []*FlagField
	fGroups  [][]*FlagField
	ignore   []string          // list of field names to ignore -- useful for dynamically adjusting the flag list.
	helpMsgs map[string]string // manually set help messages. key=fieldFefName, value=helpMsg.
	defaults map[string]string
	helpMenu string
}

// IgnoreField works by addressing the struct field name as a string.
// Fields in embedded structs can be addressed by separating field names with a ".".
//
// Ignore must be called before "Load".
//
// Example:
// - "MyField.EmbeddedField"
func (d *Decoder) IgnoreField(fName string) {
	d.ignore = append(d.ignore, fName)
}

func (d *Decoder) isIgnored(fName string) bool {
	for _, v := range d.ignore {
		if fName == v {
			return true
		}
	}

	return false
}

// SetHelp will override an existing field "help" value or create
// one if not provided as a tag value.
//
// Useful for dynamically created help messages or adding help messages
// to struct fields a user does not control.
//
// SetHelp must be called before "Load" to be effective.
func (d *Decoder) SetHelp(fName, helpMsg string) {
	d.helpMsgs[fName] = helpMsg
}

// Load implements the go-config/encoding.Unmarshaler interface.
func (d *Decoder) Unmarshal(cfigs ...interface{}) error {
	for _, v := range cfigs {
		err := d.genFields("", "", v)
		if err != nil {
			return err
		}

		// move to field group.
		d.fGroups = append(d.fGroups, d.fFields)
		d.fFields = make([]*FlagField, 0)
	}

	err := d.registerAll()
	if err != nil {
		return err
	}

	d.appendHelp(175)
	d.registerUsage()

	return d.fs.Parse(os.Args[1:])
}

// genFields is a recursive function for populating struct values from flag variables.
//
// The case-sensitive value of prefix is pre-pended to each returned expected flag variable
// separated by a dash '-'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func (d *Decoder) genFieldNodes(v interface{}) error {
	if _, err := util.IsStructPointer(v); err != nil {
		return err
	}

	nodes := node.MakeNodes(v, node.Options{}).Map()
	for _, n := range nodes {
		heritage := node.Parents(n, nodes)

		// Check if ignored or any parent(s) are ignored.
		//
		// Note that if this node or any ancestor node is ignored
		// then the res	ult is the same - this node is ignored.
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

		// Set field from flag value.
		// TODO: read in flag value.
		//err := setFieldValue(n, os.Getenv(genFullName(n, heritage)))
		//if err != nil {
		//	return err
		//}
	}

	return nil
}

// setFieldValue sets the field value. It takes into account
// special cases such as time.Time and slices.
//
// If 'flagVal' is empty then nothing is set and nil is returned.
func setFieldValue(n *node.Node, flagVal string) error {
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

// splitSlice splits an flag string.
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

// genFields is a recursive function for populating struct values from flag values.
//
// The case-sensitive value of prefix is pre-pended to each returned expected flag value
// separated by a dash '-'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func (d *Decoder) genFields(prefix, refNamePrefix string, v interface{}) error {
	if _, err := util.IsStructPointer(v); err != nil {
		return err
	}

	// iterate through struct fields.
	vStruct := reflect.ValueOf(v).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)

		if !field.CanSet() { // skip private variables
			continue
		}

		mField := vStruct.Type().Field(i)

		// Check general 'config' tag value. if it has a "ignore" value
		// then skip it entirely.
		if cfgV := mField.Tag.Get(configTag); cfgV == "ignore" {
			continue
		}

		// flag tag name, if present, trumps the generated tag name.
		//
		// If the generated flag name is used it is converted to kebab case.
		name := mField.Name
		refName := genRefName(refNamePrefix, name)
		fTag := mField.Tag.Get(flagTag)

		hTag := mField.Tag.Get(helpTag)
		if msg, ok := d.helpMsgs[refName]; ok {
			hTag = msg
		}

		// Check if ignored.
		if d.isIgnored(refName) {
			continue
		}

		// split tag value on "," for alias names.
		flagNames := strings.Split(fTag, ",")
		var fName, fAlias string
		fName = fTag
		if len(flagNames) >= 2 {
			fName = flagNames[0]
			fAlias = flagNames[1]
		}

		// fAlias must be no more than one character.
		if len(fAlias) > 1 {
			return errors.New("alias flag name '" + fAlias + "' must be one character.")
		}

		switch fName {
		case "-":
			continue // ignore field
		case "omitprefix":
			// Should only be used on struct field types, in
			// which case an existing prefix is passed through
			// to the struct fields. The immediate struct field
			// has no prefix.
			name = ""
		case "":
			name = strcase.ToKebab(name)
		default:
			name = fName
		}

		// prepend prefix
		if prefix != "" {
			// An empty name takes on the prefix so that
			// it can pass through if the type is a struct or pointer struct.
			if name == "" {
				name = prefix
			} else {
				// An existing dash means there will be 2 dashes.
				name = prefix + "-" + name
			}
		}

		// if the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// explicitly ignored list of types.
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		case reflect.Struct:
			// time.Time special struct case
			if field.Type().String() == "time.Time" {

				tFmt := getTimeFormat(mField.Tag.Get(fmtTag))
				d.fFields = append(d.fFields, &FlagField{
					RefName:   refName,
					Name:      name,
					Shorthand: fAlias,
					Usage:     hTag,
					Value:     field.Addr(),
					ValueType: "time",
					DefValue:  field.Interface().(time.Time).Format(tFmt),
					TimeFmt:   tFmt,
				})

				continue
			}

			// get a pointer and recurse
			err := d.genFields(name, refName, field.Addr().Interface())
			if err != nil {
				return err
			}
		case reflect.Ptr:
			// if it's a ptr to a struct then recurse otherwise fallthrough
			if field.IsNil() {
				z := reflect.New(field.Type().Elem())
				field.Set(z)
			}

			// check if it's pointing to a struct
			if reflect.Indirect(field).Kind() == reflect.Struct {
				if reflect.Indirect(field).Type().String() == "time.Time" {
					tFmt := getTimeFormat(mField.Tag.Get(fmtTag))
					d.fFields = append(d.fFields, &FlagField{
						RefName:   refName,
						Name:      name,
						Shorthand: fAlias,
						Usage:     hTag,
						Value:     field,
						ValueType: "time",
						DefValue:  reflect.Indirect(field).Interface().(time.Time).Format(tFmt),
						TimeFmt:   tFmt,
					})

					continue
				}

				// recurse on ptr struct
				err := d.genFields(name, refName, field.Interface())
				if err != nil {
					return err
				}

				continue
			}

			defValue, err := toStr(reflect.Indirect(field))
			if err != nil {
				return err
			}

			d.fFields = append(d.fFields, &FlagField{
				RefName:   refName,
				Name:      name,
				Shorthand: fAlias,
				Usage:     hTag,
				Value:     field,
				DefValue:  defValue,
			})

		default:
			// Validate "omitprefix" usage.
			// Cannot be used on non-struct field types.
			if fName == "omitprefix" {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			defValue, err := toStr(field)
			if err != nil {
				return err
			}
			d.fFields = append(d.fFields, &FlagField{
				RefName:   refName,
				Name:      name,
				Shorthand: fAlias,
				Usage:     hTag,
				Value:     field.Addr(),
				DefValue:  defValue,
			})
		}
	}

	return nil
}

func (d *Decoder) registerUsage() {
	d.fs.Usage = func() {
		fmt.Fprint(os.Stderr, d.helpMenu)
	}
}

func (d *Decoder) appendHelp(cols int) {
	for _, fg := range d.fGroups {
		buf := new(bytes.Buffer)
		lines := make([]string, 0, len(fg))

		maxlen := 0
		for _, f := range fg {
			line := ""
			if f.Shorthand != "" {
				line = fmt.Sprintf("  -%s, --%s", f.Shorthand, f.Name)
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

			line += usage
			if usage != "" && f.DefValue != "" {
				line += " "
			}
			if !f.defaultIsZeroValue() {
				if f.ValueType == "string" {
					line += fmt.Sprintf("[default: %q]", f.DefValue)
				} else {
					line += fmt.Sprintf("[default: %s]", f.DefValue)
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

		d.helpMenu += buf.String() + "\n"
	}
}

// registerAll will register all Fields with the flag package.
func (d *Decoder) registerAll() error {
	for _, fg := range d.fGroups {
		// Use field info to populate the flag set.
		for _, ff := range fg {
			v := reflect.Indirect(ff.Value)
			switch v.Kind() {
			case reflect.String:
				ff.ValueType = "string"
				d.fs.StringVar(
					ff.Value.Interface().(*string),
					ff.Name,
					ff.DefValue,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.StringVar(
						ff.Value.Interface().(*string),
						ff.Shorthand,
						ff.DefValue,
						ff.Usage,
					)
				}
			case reflect.Bool:
				ff.ValueType = "bool"
				b, _ := strconv.ParseBool(ff.DefValue)
				d.fs.BoolVar(
					ff.Value.Interface().(*bool),
					ff.Name,
					b,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.BoolVar(
						ff.Value.Interface().(*bool),
						ff.Shorthand,
						b,
						ff.Usage,
					)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				ff.ValueType = "int"
				// Check for time.Duration int64 special case.
				//
				// TODO: check if this still works when time package is vendored or there is a way to fake this.
				if v.Type().String() == "time.Duration" {
					ff.ValueType = "duration"
					dd, _ := time.ParseDuration(ff.DefValue)
					d.fs.DurationVar(
						ff.Value.Interface().(*time.Duration),
						ff.Name,
						dd,
						ff.Usage,
					)

					continue
				}

				d.fs.Var(
					ff,
					ff.Name,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Var(
						ff,
						ff.Shorthand,
						ff.Usage,
					)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				ff.ValueType = "uint"
				d.fs.Var(
					ff,
					ff.Name,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Var(
						ff,
						ff.Shorthand,
						ff.Usage,
					)
				}
			case reflect.Float32, reflect.Float64:
				ff.ValueType = "float"
				d.fs.Var(
					ff,
					ff.Name,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Var(
						ff,
						ff.Shorthand,
						ff.Usage,
					)
				}
			case reflect.Ptr:
				return errors.New("should not be pointer")
			case reflect.Slice:
				ff.ValueType = "stringSlice"
				d.fs.StringVar(
					ff.StrPtr,
					ff.Name,
					ff.DefValue,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.StringVar(
						ff.StrPtr,
						ff.Name,
						ff.DefValue,
						ff.Usage,
					)
				}

			// structs as values are simply ignored. They don't map cleanly for flag variables.
			case reflect.Struct:
				if v.Type().String() == "time.Time" {
					ff.ValueType = "time"

					d.fs.Var(ff, ff.Name, ff.Usage)
				} else {
					return errors.New("should not be struct")
				}
			default:
				return fmt.Errorf("unsupported type '%v'", v.Kind())
			}
		}
	}

	return nil
}

// genRefName will create a complete field reference
// name by pre-pending "prefix" + "." + refField if
// prefix exists, otherwise returns refField.
func genRefName(prefix, refField string) string {
	if prefix != "" {
		return prefix + "." + refField
	}
	return refField
}

// A FlagField represents the state of a flag.
type FlagField struct {
	RefName   string        // actual struct "." separated field name. ie "MyField.EmbeddedField".
	Name      string        // name as it appears on command line
	Shorthand string        // one-letter abbreviated flag
	Usage     string        // help message
	Value     reflect.Value // ref to actual reflect.Value.
	ValueType string        // one of: string, bool, duration, float, time, int, uint, intSlice, uintSlice, stringSlice, boolSlice

	StrPtr   *string // placeholder string for storing the value of slices.
	DefValue string  // default value (as text); for usage message
	TimeFmt  string  // time formatting for time.Time fields.
}

func (f *FlagField) String() string {
	s, _ := toStr(f.Value)
	return s
}

func (f *FlagField) Set(s string) error {
	if f.Value.Type().String() == "time.Time" || reflect.Indirect(f.Value).Type().String() == "time.Time" {
		_, err := setTime(f.Value, s, f.TimeFmt)
		return err
	}

	return setField(f.Value, s)
}

// defaultIsZeroValue returns true if the default value for this flag represents
// a zero value.
func (f *FlagField) defaultIsZeroValue() bool {
	switch f.ValueType {
	case "bool":
		return f.DefValue == "false"
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return f.DefValue == "0" || f.DefValue == "0s"
	case "int", "uint":
		return f.DefValue == "0"
	case "string":
		return f.DefValue == ""
	case "intSlice", "uintSlice", "stringSlice", "boolSlice":
		return f.DefValue == "[]"
	default:
		switch f.Value.String() {
		case "false":
			return true
		case "<nil>":
			return true
		case "":
			return true
		case "0":
			return true
		}
		return false
	}
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for a flag and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the flag's value, or the empty string if the flag is boolean.
func UnquoteUsage(f *FlagField) (name string, usage string) {
	// Look for a back-quoted name, but avoid the strings package.
	usage = f.Usage
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

	name = f.ValueType
	switch name {
	case "bool":
		name = ""
	case "float":
		name = "float"
	case "int":
		name = "int"
	case "uint":
		name = "uint"
	case "stringSlice":
		name = "strings"
	case "intSlice":
		name = "ints"
	case "uintSlice":
		name = "uints"
	case "boolSlice":
		name = "bools"
	}

	return
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

// setTime expects value to be time.Time.
//
// tFmt can be any time package handy time format like "RFC3339Nano".
// Default format is time.RFC3339.
func setTime(value reflect.Value, tv, tFmt string) (string, error) {
	if value.Kind() == reflect.Ptr {
		value = reflect.Indirect(value)
	}

	tFmt = getTimeFormat(tFmt)

	t, err := time.Parse(tFmt, tv)
	if err != nil {
		return tFmt, err
	}

	tStruct := reflect.ValueOf(t)
	value.Set(tStruct)

	return tFmt, nil
}

// setField converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
//
// All structs that implement encoding.TextUnmarshaler are supported
//
// Does not support array literals.
func setField(value reflect.Value, s string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(s)
	case reflect.Bool:
		switch strings.ToLower(s) {
		case "true":
			value.SetBool(true)
		case "false", "":
			value.SetBool(false)
		default:
			// the bool value should be explicit to tell user
			// something is amiss.
			return fmt.Errorf("cannot assign '%v' to bool type", s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// check for time.Duration int64 special case.
		//
		// TODO: check if this still works when time package is vendored or if there is a way to fake this.
		if value.Type().String() == "time.Duration" {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}

			value.SetInt(int64(d))
			return nil
		}

		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 0)
		if err != nil {
			return err
		}
		value.SetFloat(f)
	case reflect.Ptr:
		// Create non-pointer type and recursively assign.
		if value.IsNil() {
			z := reflect.New(value.Type().Elem())
			value.Set(z)
		}

		return setField(reflect.Indirect(value), s)

	case reflect.Slice:
		// TODO: underlying slice type must not be complex.
		// TODO: consider supporting native bash arrays.
		// create a slice and recursively assign the elements
		baseType := reflect.TypeOf(value.Interface()).Elem()
		s = strings.Trim(s, "[]") // trim brackets for bracket support.
		vals := strings.Split(s, ",")

		slice := reflect.MakeSlice(value.Type(), 0, len(vals))
		for _, v := range vals {
			// trim whitespace from each value to support comma-separated with spaces.
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)

			// each item must be the correct type.
			baseValue := reflect.New(baseType).Elem()
			err := setField(baseValue, v)
			if err != nil {
				return err
			}
			slice = reflect.Append(slice, baseValue)
		}

		value.Set(slice)

	// structs as values are simply ignored. They don't map cleanly for flag variables.
	case reflect.Struct:
		return nil
	default:
		return fmt.Errorf("unsupported type '%v'", value.Kind())
	}

	return nil
}

func toStr(field reflect.Value) (string, error) {
	switch field.Kind() {
	case reflect.String:
		return field.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(field.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Support case of int64 as a time.Duration.
		if field.Type().String() == "time.Duration" {
			return field.Interface().(time.Duration).String(), nil
		}

		return strconv.FormatInt(field.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", field.Float()), nil
	case reflect.Slice:
		// Create a slice and recursively assign the elements.
		baseType := reflect.TypeOf(field.Interface()).Elem()

		// Handle empty slice - no defaults.
		if field.Len() == 0 {
			return "", nil
		}

		outValue := ""
		sep := ","
		for i := 0; i < field.Len(); i++ {
			item := field.Index(i)

		typeCheck:
			switch baseType.Kind() {
			case reflect.String:
				outValue += item.String()
			case reflect.Bool:
				outValue += strconv.FormatBool(item.Bool())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				outValue += strconv.FormatInt(item.Int(), 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				outValue += strconv.FormatUint(item.Uint(), 10)
			case reflect.Float32, reflect.Float64:
				outValue += fmt.Sprintf("%f", item.Float())
			case reflect.Ptr:
				item = item.Elem()
				goto typeCheck
			default:
				// Skip all other types. Structs, for example, do not map.
				// Only simple types supported here.
				return "", nil
			}

			if i < field.Len() {
				outValue += sep
			}
		}

		return outValue, nil

	// structs as values are simply ignored. They don't map cleanly for flag variables.
	case reflect.Struct:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported type '%v'", field.Kind())
	}

	return "", nil
}

func getTimeFormat(timeFmt string) string {
	if timeFmt == "" {
		timeFmt = time.RFC3339 // default format
	}
	switch timeFmt {
	case "ANSIC":
		timeFmt = time.ANSIC
	case "UnixDate":
		timeFmt = time.UnixDate
	case "RubyDate":
		timeFmt = time.RubyDate
	case "RFC822":
		timeFmt = time.RFC822
	case "RFC822Z":
		timeFmt = time.RFC822Z
	case "RFC850":
		timeFmt = time.RFC850
	case "RFC1123":
		timeFmt = time.RFC1123
	case "RFC1123Z":
		timeFmt = time.RFC1123Z
	case "RFC3339":
		timeFmt = time.RFC3339
	case "RFC3339Nano":
		timeFmt = time.RFC3339Nano
	case "Kitchen":
		timeFmt = time.Kitchen
	case "Stamp":
		timeFmt = time.Stamp
	case "StampMilli":
		timeFmt = time.StampMilli
	case "StampMicro":
		timeFmt = time.StampMicro
	case "StampNano":
		timeFmt = time.StampNano
	}
	return timeFmt
}
