package flg

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
)

const (
	flagTag = "flag"
	descTag = "comment" // do we want a different tag for the flag vs toml?
	fmtTag  = "fmt"
)

type Flags struct {
	values  map[string]interface{} // map[VariableName]value
	flagSet *flag.FlagSet
}

// New creates a custom flagset based on the struct i.
//
func New(i interface{}) (*Flags, error) {
	if !isValidConfig(i) {
		return nil, errors.New("invalid config")
	}
	f := &Flags{
		values:  make(map[string]interface{}),
		flagSet: flag.NewFlagSet(os.Args[0], flag.ExitOnError),
	}
	vStruct := reflect.ValueOf(i).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)
		dField := vStruct.Type().Field(i)
		tag := dField.Tag.Get(flagTag)
		name := strcase.ToKebab(dField.Name)
		desc := dField.Tag.Get(descTag)
		if tag != "" {
			name = tag
		}

		// skip private variables and disabled flags
		if tag == "-" || !field.CanSet() {
			continue
		}

		if isAlias(field) {
			if field.Type().String() == "time.Duration" {
				d := field.Interface().(time.Duration)
				f.values[name] = f.flagSet.String(name, d.String(), desc)
				continue
			}
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				f.values[name] = f.flagSet.String(name, string(b), desc)
				continue
			}
		}
	switchStart:
		switch field.Kind() {
		// explicit list of unsupported types
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map, reflect.Slice:
			continue
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			f.values[name] = f.flagSet.Int(name, int(field.Int()), desc)
		case reflect.Int64:
			f.values[name] = f.flagSet.Int64(name, field.Int(), desc)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			f.values[name] = f.flagSet.Uint(name, uint(field.Uint()), desc)
		case reflect.Uint64:
			f.values[name] = f.flagSet.Uint64(name, field.Uint(), desc)
		case reflect.String:
			f.values[name] = f.flagSet.String(name, field.String(), desc)
		case reflect.Bool:
			f.values[name] = f.flagSet.Bool(name, field.Bool(), desc)
		case reflect.Float32, reflect.Float64:
			f.values[name] = f.flagSet.Float64(name, field.Float(), desc)
		case reflect.Ptr:
			// todo dereference and handle. use recursion
			field = field.Elem()
			goto switchStart // easiest solution, but do we want a goto statement
		case reflect.Struct:
			// todo: Should we add a prefix for flags? structName-childVar or only
			// support a struct if they implement a marshaler
			if field.Type().String() == "time.Time" {
				timeFmt := dField.Tag.Get(fmtTag)
				timeFmt = getTimeFormat(timeFmt)
				t := field.Interface().(time.Time)
				f.values[name] = f.flagSet.String(name, t.Format(timeFmt), desc)
				continue
			}
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				f.values[name] = f.flagSet.String(name, string(b), desc)
			}
		}
	}
	return f, nil
}

// Parse the internal flags and the user defined flags
func (f *Flags) Parse() error {

	// add other defined flags
	flag.VisitAll(func(flg *flag.Flag) {
		f.flagSet.Var(flg.Value, flg.Name, flg.Usage)
	})

	return f.flagSet.Parse(os.Args[1:])
}

// Unmarshal the given struct from the flagSet
func (f *Flags) Unmarshal(i interface{}) error {
	return nil
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
