package show

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
)

var (
	flagTag   = "flag"
	envTag    = "env"
	jsonTag   = "json"
	yamlTag   = "yaml"
	tomlTag   = "toml"
	reqTag    = "req"
	showTag   = "show"
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
)

func NewEncoder(showMsg string) *Encoder {
	return &Encoder{
		showMsg: showMsg + "\n",
	}

}

type Encoder struct {
	showMsg string
}

// Load implements the go-config/encoding.Unmarshaler interface.
func (d *Encoder) Unmarshal(dCfg, cfg interface{}) (string, error) {
	// Fields for default values.
	defFields, err := d.genFields(make([]*FlagField, 0), "", "", dCfg)
	if err != nil {
		return "", err
	}

	defFieldsM := make(map[string]*FlagField)
	for _, v := range defFields {
		defFieldsM[v.RefName] = v
	}

	// Fields for resolved values.
	fields, err := d.genFields(make([]*FlagField, 0), "", "", cfg)
	if err != nil {
		return "", err
	}

	d.genShowMsg(175, defFieldsM, fields)

	return d.showMsg, nil
}

// genFields is a recursive function for populating struct values from env variables.
//
// The case-sensitive value of prefix is pre-pended to each returned expected env variable
// separated by a dash '-'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func (d *Encoder) genFields(fields []*FlagField, prefix, refNamePrefix string, v interface{}) ([]*FlagField, error) {
	// Verify that v is struct pointer. Should not be nil.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))

		// Must be pointing to a struct.
	} else if pv := reflect.Indirect(value); pv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
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

		// pick a tag name override, if present, trumps the generated field name.
		//
		// If the field name is used it is converted to snake case.
		fName := mField.Name
		name := fName
		flagName := mField.Tag.Get(flagTag)
		envName := mField.Tag.Get(envTag)
		jsonName := mField.Tag.Get(jsonTag)
		yamlName := mField.Tag.Get(yamlTag)
		tomlName := mField.Tag.Get(tomlTag)
		reqV := mField.Tag.Get(reqTag)
		if reqV == "" {
			reqV = "false"
		}
		isReq, err := strconv.ParseBool(reqV)
		if err != nil {
			return nil, err
		}
		showV := mField.Tag.Get(showTag)
		if showV == "" {
			showV = "true"
		}
		showValue, err := strconv.ParseBool(showV)
		if err != nil {
			return nil, err
		}

		// Any tag with this field turned off works.
		switch "-" {
		case flagName, envName, jsonName, yamlName, tomlName:
			continue
		}

		switch "omitprefix" {
		case flagName, envName, jsonName, yamlName, tomlName:
			// Should only be used on struct field types, in
			// which case an existing prefix is passed through
			// to the struct fields. The immediate struct field
			// has no prefix.
			name = ""
		default:
			// Pick a tag name if any are provided.
			switch {
			case flagName != "":
				name = strcase.ToSnake(flagName)
			case envName != "":
				name = strcase.ToSnake(envName)
			case jsonName != "":
				name = strcase.ToSnake(jsonName)
			case yamlName != "":
				name = strcase.ToSnake(yamlName)
			case tomlName != "":
				name = strcase.ToSnake(tomlName)
			default:
				name = strcase.ToSnake(name)
			}
		}

		refName := genRefName(refNamePrefix, name)

		// prepend prefix
		if prefix != "" {
			// An empty name takes on the prefix so that
			// it can pass through if the type is a struct or pointer struct.
			if name == "" {
				name = prefix
			} else {
				// An existing underscore means there will be 2 underscores.
				name = prefix + "_" + name
			}
		}

		// If the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// Explicitly ignored list of types.
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		case reflect.Struct:
			// time.Time special struct case
			if field.Type().String() == "time.Time" {
				strV, strT, tFmt := timeToStr(field, mField.Tag.Get(fmtTag))
				fields = append(fields, &FlagField{
					RefName:    refName,
					Name:       name,
					Value:      field,
					ValueType:  strT,
					StrValue:   strV,
					TimeFmt:    tFmt,
					IsRequired: isReq,
					ShowValue:  showValue,
				})

				continue
			}

			// get a pointer and recurse
			fields, err = d.genFields(fields, name, refName, field.Addr().Interface())
			if err != nil {
				return fields, err
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
					strV, strT, tFmt := timeToStr(reflect.Indirect(field), mField.Tag.Get(fmtTag))
					fields = append(fields, &FlagField{
						RefName:    refName,
						Name:       name,
						Value:      field,
						ValueType:  strT,
						StrValue:   strV,
						TimeFmt:    tFmt,
						IsRequired: isReq,
						ShowValue:  showValue,
					})

					continue
				}

				// recurse on ptr struct
				fields, err = d.genFields(fields, name, refName, field.Interface())
				if err != nil {
					return nil, err
				}

				continue
			}

			strV, strType, err := toStr(reflect.Indirect(field))
			if err != nil {
				return nil, err
			}

			fields = append(fields, &FlagField{
				RefName:    refName,
				Name:       name,
				Value:      reflect.Indirect(field),
				ValueType:  strType,
				StrValue:   strV,
				IsRequired: isReq,
				ShowValue:  showValue,
			})
		default:
			// Validate "omitprefix" usage.
			// Cannot be used on non-struct field types.
			switch "omitprefix" {
			case flagName, envName, jsonName, yamlName, tomlName:
				// Should only be used on struct field types, in
				// which case an existing prefix is passed through
				// to the struct fields. The immediate struct field
				// has no prefix.
				return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			strV, strType, err := toStr(field)
			if err != nil {
				return nil, err
			}
			fields = append(fields, &FlagField{
				RefName:    refName,
				Name:       name,
				Value:      field,
				ValueType:  strType,
				StrValue:   strV,
				IsRequired: isReq,
				ShowValue:  showValue,
			})
		}
	}

	return fields, nil
}

func (d *Encoder) genShowMsg(cols int, dFields map[string]*FlagField, fields []*FlagField) {
	buf := new(bytes.Buffer)
	lines := make([]string, 0, len(fields))

	maxlen := 0
	for _, f := range fields {
		line := ""

		// field name
		//
		// The special character will be replaced with spacing once the
		// correct alignment is calculated
		line += f.Name + ":\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		// field value
		if f.ShowValue {
			if f.ValueType == "string" {
				line += fmt.Sprintf("%q", f.StrValue)
			} else {
				line += fmt.Sprintf("%s", f.StrValue)
			}
		} else {
			line += "[redacted]"
		}

		// default
		if df, ok := dFields[f.RefName]; ok && !df.isZeroValue() && f.ShowValue {
			if f.ValueType == "string" {
				line += fmt.Sprintf(" [default: %q]", df.StrValue)
			} else {
				line += fmt.Sprintf(" [default: %s]", df.StrValue)
			}
		}

		// required
		if f.IsRequired {
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

	d.showMsg += buf.String()
}

// defaultIsZeroValue returns true if the default value for this flag represents
// a zero value.
func (f *FlagField) isZeroValue() bool {
	switch f.ValueType {
	case "bool":
		return f.StrValue == "false"
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return f.StrValue == "0" || f.StrValue == "0s"
	case "int", "uint", "float":
		return f.StrValue == "0"
	case "string":
		return f.StrValue == ""
	case "slice", "intSlice", "uintSlice", "stringSlice", "boolSlice":
		return f.StrValue == "[]"
	case "time":
		return f.StrValue == "<empty>"
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
	RefName    string        // actual struct "." separated field name. ie "MyField.EmbeddedField".
	Name       string        // name as it appears on command line
	Value      reflect.Value // ref to actual reflect.ValueBefore.
	ValueType  string        // one of: string, bool, duration, float, time, int, uint, slice, intSlice, uintSlice, stringSlice, boolSlice
	StrValue   string        // value (as text)
	TimeFmt    string        // time formatting for time.Time fields.
	IsRequired bool
	ShowValue  bool // if false then the value is shown as "[redacted]"
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

func timeToStr(field reflect.Value, timeFmt string) (strV, strT, tFmt string) {
	strT = "time"
	tFmt = getTimeFormat(timeFmt)
	t := field.Interface().(time.Time)
	if t.IsZero() {
		strV = "<empty>"
	} else {
		strV = t.Format(tFmt)
	}

	return
}

func toStr(field reflect.Value) (str string, strType string, err error) {
	switch field.Kind() {
	case reflect.String:
		return field.String(), "string", nil
	case reflect.Bool:
		return strconv.FormatBool(field.Bool()), "bool", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Support case of int64 as a time.Duration.
		if field.Type().String() == "time.Duration" {
			return field.Interface().(time.Duration).String(), "duration", nil
		}

		return strconv.FormatInt(field.Int(), 10), "int", nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10), "uint", nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", field.Float()), "float", nil
	case reflect.Slice:
		// Create a slice and recursively assign the elements.
		baseType := reflect.TypeOf(field.Interface()).Elem()

		// Handle empty slice - no defaults.
		if field.Len() == 0 {
			return "[]", "slice", nil
		}

		outValue := "["
		sep := ","
		for i := 0; i < field.Len(); i++ {
			item := field.Index(i)

		typeCheck:
			switch baseType.Kind() {
			case reflect.String:
				outValue += item.String()
				strType = "stringSlice"
			case reflect.Bool:
				outValue += strconv.FormatBool(item.Bool())
				strType = "boolSlice"
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				outValue += strconv.FormatInt(item.Int(), 10)
				strType = "intSlice"
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				outValue += strconv.FormatUint(item.Uint(), 10)
				strType = "uintSlice"
			case reflect.Float32, reflect.Float64:
				outValue += fmt.Sprintf("%f", item.Float())
				strType = "floatSlice"
			case reflect.Ptr:
				item = item.Elem()
				goto typeCheck
			default:
				// Skip all other types. Structs, for example, do not map.
				// Only simple types supported here.
				return "", "slice", nil
			}

			if i < field.Len() {
				outValue += sep
			}
		}

		return outValue + "]", strType, nil

	// structs as values are simply ignored. They don't map cleanly for flag variables.
	case reflect.Struct:
		return "", "struct", nil
	default:
		return "", "", fmt.Errorf("unsupported type '%v'", field.Kind())
	}

	return "", "", nil
}

func getTimeFormat(tFmt string) string {
	if tFmt == "" {
		tFmt = time.RFC3339 // default format
	}
	switch tFmt {
	case "ANSIC":
		tFmt = time.ANSIC
	case "UnixDate":
		tFmt = time.UnixDate
	case "RubyDate":
		tFmt = time.RubyDate
	case "RFC822":
		tFmt = time.RFC822
	case "RFC822Z":
		tFmt = time.RFC822Z
	case "RFC850":
		tFmt = time.RFC850
	case "RFC1123":
		tFmt = time.RFC1123
	case "RFC1123Z":
		tFmt = time.RFC1123Z
	case "RFC3339":
		tFmt = time.RFC3339
	case "RFC3339Nano":
		tFmt = time.RFC3339Nano
	case "Kitchen":
		tFmt = time.Kitchen
	case "Stamp":
		tFmt = time.Stamp
	case "StampMilli":
		tFmt = time.StampMilli
	case "StampMicro":
		tFmt = time.StampMicro
	case "StampNano":
		tFmt = time.StampNano
	}
	return tFmt
}
