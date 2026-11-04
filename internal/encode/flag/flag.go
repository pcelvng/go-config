package flg

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/jbsmith7741/go-tools/appenderr"

	"github.com/hydronica/go-config/internal/encode"
)

type Flags struct {
	*flag.FlagSet
	defaults  map[string]string
	remaining []string
}

// New creates a custom flagset based on the struct i.
func New(i interface{}) (*Flags, error) {
	flg := &Flags{
		defaults: make(map[string]string),
	}
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	if i == nil {
		flg.FlagSet = flagSet
		return flg, nil
	}
	if !isValidConfig(i) {
		return nil, errors.New("invalid config, must be pointer to a struct")
	}
	if err := registerFlags(flagSet, flg.defaults, "", reflect.ValueOf(i).Elem()); err != nil {
		return nil, err
	}
	flg.FlagSet = flagSet
	return flg, nil
}

func registerFlags(flagSet *flag.FlagSet, defaults map[string]string, prefix string, vStruct reflect.Value) error {
	for i := 0; i < vStruct.NumField(); i++ {
		sf := vStruct.Type().Field(i)
		field := vStruct.Field(i)

		if !encode.ShouldProcessField(sf, field) {
			continue
		}
		if sf.Tag.Get(encode.ConfigTag) == "ignore" {
			continue
		}
		raw := sf.Tag.Get(encode.FlagTag)
		if raw == "-" {
			continue
		}
		name, omitPrefix, _ := encode.ParseTag(raw)

		if encode.IsRecursableStruct(field) {
			if omitPrefix {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, ".")
			}
			child, ok := encode.StructValue(field)
			if !ok {
				continue
			}
			if err := registerFlags(flagSet, defaults, childPrefix, child); err != nil {
				return err
			}
			continue
		}

		if name == "omitprefix" {
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}
		segment := strcase.ToKebab(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, ".")
		if omitPrefix {
			key = segment
		}
		if err := registerLeafFlag(flagSet, defaults, key, sf.Tag.Get(encode.DescTag), sf, field); err != nil {
			return err
		}
	}
	return nil
}

func registerLeafFlag(flagSet *flag.FlagSet, defaults map[string]string, name, desc string, sf reflect.StructField, field reflect.Value) error {
	if isAlias(field) {
		if implementsStringer(field) {
			s := field.Interface().(fmt.Stringer).String()
			flagSet.String(name, s, desc)
			defaults[name] = s
			return nil
		}
		if implementsMarshaler(field) {
			b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
			flagSet.String(name, string(b), desc)
			defaults[name] = string(b)
			return nil
		}
	}
switchStart:
	switch field.Kind() {
	case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map, reflect.Slice:
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		i := int(field.Int())
		flagSet.Int(name, i, desc)
		defaults[name] = strconv.Itoa(i)
	case reflect.Int64:
		flagSet.Int64(name, field.Int(), desc)
		defaults[name] = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		flagSet.Uint(name, uint(field.Uint()), desc)
		defaults[name] = strconv.FormatUint(field.Uint(), 10)
	case reflect.Uint64:
		flagSet.Uint64(name, field.Uint(), desc)
		defaults[name] = strconv.FormatUint(field.Uint(), 10)
	case reflect.String:
		flagSet.String(name, field.String(), desc)
		defaults[name] = field.String()
	case reflect.Bool:
		flagSet.Bool(name, field.Bool(), desc)
		if field.Bool() {
			defaults[name] = "true"
		} else {
			defaults[name] = "false"
		}
	case reflect.Float32, reflect.Float64:
		flagSet.Float64(name, field.Float(), desc)
		defaults[name] = strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Ptr:
		if field.IsNil() {
			field = reflect.New(field.Type().Elem())
		}
		field = field.Elem()
		goto switchStart
	case reflect.Struct:
		if field.Type().String() == "time.Time" {
			timeFmt := getTimeFormat(sf.Tag.Get(encode.FormatTag))
			t := field.Interface().(time.Time)
			flagSet.String(name, t.Format(timeFmt), desc)
			defaults[name] = t.Format(timeFmt)
			return nil
		}
		if implementsMarshaler(field) {
			b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
			flagSet.String(name, string(b), desc)
			defaults[name] = string(b)
		}
	}
	return nil
}

// Parse the internal flags and the user defined flags.
// Positional args may appear before, after, or between flags.
func (f *Flags) Parse() error {
	flag.VisitAll(func(flg *flag.Flag) {
		f.Var(flg.Value, flg.Name, flg.Usage)
	})

	flagArgs, posArgs := splitFlagArgs(f.FlagSet, os.Args[1:])
	if err := f.FlagSet.Parse(flagArgs); err != nil {
		return err
	}
	if posArgs == nil {
		posArgs = []string{}
	}
	f.remaining = posArgs
	return nil
}

// Args returns non-flag arguments remaining after Parse.
// Unlike flag.FlagSet.Args, this includes positionals that appeared before flags.
func (f *Flags) Args() []string {
	return f.remaining
}

// splitFlagArgs separates flag arguments from positional arguments so the
// standard library FlagSet (which stops at the first non-flag) can still
// parse flags that appear after positionals.
func splitFlagArgs(fs *flag.FlagSet, args []string) (flagArgs, posArgs []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			posArgs = append(posArgs, a)
			continue
		}

		name := a[1:]
		if strings.HasPrefix(a, "--") {
			name = a[2:]
		}
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			flagArgs = append(flagArgs, a)
			continue
		}

		flagArgs = append(flagArgs, a)
		if isBoolFlag(fs, name) {
			continue
		}
		if i+1 >= len(args) {
			continue
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}
	return flagArgs, posArgs
}

func isBoolFlag(fs *flag.FlagSet, name string) bool {
	flg := fs.Lookup(name)
	if flg == nil {
		return false
	}
	type boolFlag interface {
		IsBoolFlag() bool
	}
	bf, ok := flg.Value.(boolFlag)
	return ok && bf.IsBoolFlag()
}

// Unmarshal the given struct from the flagSet
func (f Flags) Unmarshal(c interface{}) error {
	if !isValidConfig(c) {
		return errors.New("invalid config")
	}

	errs := appenderr.New()
	if err := unmarshalFlags(f.FlagSet, f.defaults, "", reflect.ValueOf(c).Elem(), errs); err != nil {
		return err
	}
	return errs.ErrOrNil()
}

func unmarshalFlags(flagSet *flag.FlagSet, defaults map[string]string, prefix string, vStruct reflect.Value, errs *appenderr.AppendErr) error {
	for i := 0; i < vStruct.NumField(); i++ {
		sf := vStruct.Type().Field(i)
		field := vStruct.Field(i)

		if !encode.ShouldProcessField(sf, field) {
			continue
		}
		if sf.Tag.Get(encode.ConfigTag) == "ignore" {
			continue
		}
		raw := sf.Tag.Get(encode.FlagTag)
		if raw == "-" {
			continue
		}
		name, omitPrefix, _ := encode.ParseTag(raw)

		if encode.IsRecursableStruct(field) {
			if omitPrefix {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, ".")
			}
			if !subtreeHasFlagChanges(flagSet, defaults, childPrefix, field) {
				continue
			}
			child, ok := encode.InitStructValue(field)
			if !ok {
				continue
			}
			if err := unmarshalFlags(flagSet, defaults, childPrefix, child, errs); err != nil {
				return err
			}
			continue
		}

		if name == "omitprefix" {
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}
		segment := strcase.ToKebab(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, ".")
		if omitPrefix {
			key = segment
		}
		flg := flagSet.Lookup(key)
		if flg == nil {
			continue
		}
		if defaults[key] == flg.Value.String() {
			continue
		}
		errs.Add(encode.SetField(field, flg.Value.String(), sf))
	}
	return nil
}

func subtreeHasFlagChanges(flagSet *flag.FlagSet, defaults map[string]string, prefix string, field reflect.Value) bool {
	child, ok := encode.StructValue(field)
	if !ok {
		return false
	}
	for i := 0; i < child.NumField(); i++ {
		sf := child.Type().Field(i)
		f := child.Field(i)
		if !encode.ShouldProcessField(sf, f) {
			continue
		}
		if sf.Tag.Get(encode.ConfigTag) == "ignore" {
			continue
		}
		raw := sf.Tag.Get(encode.FlagTag)
		if raw == "-" {
			continue
		}
		name, omitPrefix, _ := encode.ParseTag(raw)

		if encode.IsRecursableStruct(f) {
			if omitPrefix {
				continue
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, ".")
			}
			if subtreeHasFlagChanges(flagSet, defaults, childPrefix, f) {
				return true
			}
			continue
		}

		if name == "omitprefix" {
			continue
		}
		segment := strcase.ToKebab(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, ".")
		if omitPrefix {
			key = segment
		}
		flg := flagSet.Lookup(key)
		if flg == nil {
			continue
		}
		if defaults[key] != flg.Value.String() {
			return true
		}
	}
	return false
}

// isValidConfig checks if a config can be properly read and written to.
// must be a pointer to a config and not nil
func isValidConfig(i interface{}) bool {
	if i == nil {
		return false
	}
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return false
	}
	if v.Elem().Kind() != reflect.Struct {
		return false
	}
	return true
}

func isAlias(v reflect.Value) bool {
	if v.Kind() == reflect.Struct || v.Kind() == reflect.Ptr {
		return false
	}
	s := fmt.Sprint(v.Type())
	return strings.Contains(s, ".")
}

func implementsUnmarshaler(v reflect.Value) bool {
	return v.Type().Implements(reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem())
}

func implementsMarshaler(v reflect.Value) bool {
	return v.Type().Implements(reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem())
}

func implementsStringer(v reflect.Value) bool {
	return v.Type().Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem())
}

func getTimeFormat(timeFmt string) string {
	if timeFmt == "" {
		timeFmt = time.RFC3339
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
