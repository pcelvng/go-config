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
	vStruct := reflect.ValueOf(i).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)
		dField := vStruct.Type().Field(i)
		tag := dField.Tag.Get(encode.FlagTag)
		name := strcase.ToKebab(dField.Name)
		desc := dField.Tag.Get(encode.DescTag)
		confTag := dField.Tag.Get(encode.ConfigTag)
		if tag == "" {
			tag = name
		}

		// skip private variables and disabled flags
		if tag == "-" || confTag == "ignore" || !field.CanSet() {
			continue
		}

		if isAlias(field) {
			/*if field.Type().String() == "time.Duration" {
				d := field.Interface().(time.Duration)
				flagSet.String(tag, d.String(), desc)
				continue
			}*/
			if implementsStringer(field) {
				s := field.Interface().(fmt.Stringer).String()
				flagSet.String(tag, s, desc)
				flg.defaults[tag] = s
				continue
			}
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				flagSet.String(tag, string(b), desc)
				flg.defaults[tag] = string(b)
				continue
			}
		}
	switchStart:
		switch field.Kind() {
		// explicit list of unsupported types
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map, reflect.Slice:
			continue
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			i := int(field.Int())
			flagSet.Int(tag, i, desc)
			flg.defaults[tag] = strconv.Itoa(i)
		case reflect.Int64:
			flagSet.Int64(tag, field.Int(), desc)
			flg.defaults[tag] = strconv.FormatInt(field.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			flagSet.Uint(tag, uint(field.Uint()), desc)
			flg.defaults[tag] = strconv.FormatUint(field.Uint(), 10)
		case reflect.Uint64:
			flagSet.Uint64(tag, field.Uint(), desc)
			flg.defaults[tag] = strconv.FormatUint(field.Uint(), 10)
		case reflect.String:
			flagSet.String(tag, field.String(), desc)
			flg.defaults[tag] = field.String()
		case reflect.Bool:
			flagSet.Bool(tag, field.Bool(), desc)
			if field.Bool() {
				flg.defaults[tag] = "true"
			} else {
				flg.defaults[tag] = "false"
			}
		case reflect.Float32, reflect.Float64:
			flagSet.Float64(tag, field.Float(), desc)
			flg.defaults[tag] = strconv.FormatFloat(field.Float(), 'f', -1, 64)
		case reflect.Ptr:
			// if nil create a new instance so we can setup the flag
			if field.IsNil() {
				field = reflect.New(field.Type().Elem())
			}
			field = field.Elem()
			goto switchStart // easiest solution, but do we want a goto statement
		case reflect.Struct:

			if field.Type().String() == "time.Time" {
				timeFmt := dField.Tag.Get(encode.FormatTag)
				timeFmt = getTimeFormat(timeFmt)
				t := field.Interface().(time.Time)
				flagSet.String(tag, t.Format(timeFmt), desc)
				flg.defaults[tag] = t.Format(timeFmt)
				continue
			}

			// support a struct if they implement a marshaler
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				flagSet.String(tag, string(b), desc)
				flg.defaults[tag] = string(b)
			}
		}
	}
	flg.FlagSet = flagSet
	return flg, nil
}

// Parse the internal flags and the user defined flags.
// Positional args may appear before, after, or between flags.
func (f *Flags) Parse() error {
	// add other defined flags
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

	vStruct := reflect.ValueOf(c).Elem()
	errs := appenderr.New()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)
		dField := vStruct.Type().Field(i)
		tag := dField.Tag.Get(encode.FlagTag)
		name := strcase.ToKebab(dField.Name)
		//confTag := dField.Tag.Get(configTag)
		if tag != "" {
			name = tag
		}

		flg := f.FlagSet.Lookup(name)
		if flg == nil {
			// ignore all types without flags
			continue
		}
		// skip flags set to default
		if f.defaults[name] == flg.Value.String() {
			continue
		}
		errs.Add(encode.SetField(field, flg.Value.String(), dField))
	}
	return errs.ErrOrNil()
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
