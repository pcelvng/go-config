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

	"github.com/iancoleman/strcase"
)

var (
	flagTag   = "flag"   // Expected env struct tag name.
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
	helpTag   = "help" // Only used for encoding.
)

// A FlagField represents the state of a flag.
type FlagField struct {
	Name      string        // name as it appears on command line
	Shorthand string        // one-letter abbreviated flag
	Usage     string        // help message
	Value     reflect.Value // ref to actual reflect.Value.
	ValueType string        // one of: string, bool, duration, float, time, int, uint, intSlice, uintSlice, stringSlice, boolSlice

	StrPtr   *string // placeholder string for storing the value of slices.
	DefValue string  // default value (as text); for usage message
	TimeFmt  string  // time formatting for time.Time fields.
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

func NewDecoder(helpBlock string) *Decoder {
	return &Decoder{
		fs:       flag.NewFlagSet(os.Args[0], flag.ExitOnError),
		fFields:  make([]*FlagField, 0),
		defaults: make(map[string]string),
		helpMenu: helpBlock + "\n",
	}
}

type Decoder struct {
	fs       *flag.FlagSet
	fFields  []*FlagField
	fGroups  [][]*FlagField
	defaults map[string]string
	helpMenu string
}

// Unmarshal implements the go-config/encoding.Unmarshaler interface.
func (d *Decoder) Unmarshal(cfigs ...interface{}) error {
	for _, v := range cfigs {
		err := d.genFields("", v)
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

	d.appendHelp(100)
	d.registerUsage()

	return d.fs.Parse(os.Args[1:])
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

				i, _ := strconv.ParseInt(ff.DefValue, 10, 0)
				d.fs.Int64Var(
					ff.Value.Interface().(*int64),
					ff.Name,
					i,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Int64Var(
						ff.Value.Interface().(*int64),
						ff.Shorthand,
						i,
						ff.Usage,
					)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				ff.ValueType = "uint"
				i, _ := strconv.ParseUint(ff.DefValue, 10, 0)
				d.fs.Uint64Var(
					ff.Value.Interface().(*uint64),
					ff.Name,
					i,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Uint64Var(
						ff.Value.Interface().(*uint64),
						ff.Shorthand,
						i,
						ff.Usage,
					)
				}
			case reflect.Float32, reflect.Float64:
				ff.ValueType = "float"
				f, _ := strconv.ParseFloat(ff.DefValue, 0)
				d.fs.Float64Var(
					ff.Value.Interface().(*float64),
					ff.Name,
					f,
					ff.Usage,
				)

				if ff.Shorthand != "" {
					d.fs.Float64Var(
						ff.Value.Interface().(*float64),
						ff.Name,
						f,
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

			// structs as values are simply ignored. They don't map cleanly for environment variables.
			case reflect.Struct:
				return errors.New("should not be struct")
			default:
				return fmt.Errorf("unsupported type '%v'", v.Kind())
			}
		}
	}

	return nil
}

// populate is a recursive function for populating struct values from env variables.
//
// The case-sensitive value of prefix is pre-pended to each returned expected env variable
// separated by a dash '-'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func (d *Decoder) genFields(prefix string, v interface{}) error {
	// Verify that v is struct pointer. Should not be nil.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))

		// Must be pointing to a struct.
	} else if pv := reflect.Indirect(value); pv.Kind() != reflect.Struct {
		return fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
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

		// env tag name, if present, trumps the generated field name.
		//
		// If the field name is used it is converted to screaming snake case (uppercase with underscores).
		name := mField.Name
		fTag := mField.Tag.Get(flagTag)
		hTag := mField.Tag.Get(helpTag)

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
			err := d.genFields(name, field.Addr().Interface())
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
				err := d.genFields(name, field.Interface())
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

// setTime expects value to be time.Time.
//
// tFmt can be any time package handy time format like "RFC3339Nano".
// Default format is time.RFC3339.
func setTime(value reflect.Value, tv, tFmt string) (string, error) {
	tFmt = getTimeFormat(tFmt)

	t, err := time.Parse(tFmt, tv)
	if err != nil {
		return tFmt, err
	}

	tStruct := reflect.ValueOf(t)
	value.Set(tStruct)

	return tFmt, nil
}

func toStr(field reflect.Value) (string, error) {
	// TODO: handle formatting of zero values.
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

// FlagUsagesWrapped returns a string containing the usage information
// for all flags in the FlagSet. Wrapped to `cols` columns (0 for no
// wrapping)
//func (d *Decoder) FlagUsagesWrapped(cols int) string {
//	buf := new(bytes.Buffer)
//
//	lines := make([]string, 0, len(f.formal))
//
//	maxlen := 0
//	f.VisitAll(func(flag *Flag) {
//		if flag.Hidden {
//			return
//		}
//
//		line := ""
//		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
//			line = fmt.Sprintf("  -%s, --%s", flag.Shorthand, flag.Name)
//		} else {
//			line = fmt.Sprintf("      --%s", flag.Name)
//		}
//
//		varname, usage := UnquoteUsage(flag)
//		if varname != "" {
//			line += " " + varname
//		}
//		if flag.NoOptDefVal != "" {
//			switch flag.Value.Type() {
//			case "string":
//				line += fmt.Sprintf("[=\"%s\"]", flag.NoOptDefVal)
//			case "bool":
//				if flag.NoOptDefVal != "true" {
//					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
//				}
//			case "count":
//				if flag.NoOptDefVal != "+1" {
//					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
//				}
//			default:
//				line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
//			}
//		}
//
//		// This special character will be replaced with spacing once the
//		// correct alignment is calculated
//		line += "\x00"
//		if len(line) > maxlen {
//			maxlen = len(line)
//		}
//
//		line += usage
//		if !flag.defaultIsZeroValue() {
//			if flag.Value.Type() == "string" {
//				line += fmt.Sprintf(" (default %q)", flag.DefValue)
//			} else {
//				line += fmt.Sprintf(" (default %s)", flag.DefValue)
//			}
//		}
//		if len(flag.Deprecated) != 0 {
//			line += fmt.Sprintf(" (DEPRECATED: %s)", flag.Deprecated)
//		}
//
//		lines = append(lines, line)
//	})
//
//	for _, line := range lines {
//		sidx := strings.Index(line, "\x00")
//		spacing := strings.Repeat(" ", maxlen-sidx)
//		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
//		fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
//	}
//
//	return buf.String()
//}
