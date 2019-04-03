package flag

import (
	"encoding"
	"flag"
	"fmt"
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
	f := &Flags{
		values:  make(map[string]interface{}),
		flagSet: flag.NewFlagSet("go-config", flag.ExitOnError),
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
		case reflect.Int:
			f.values[name] = f.flagSet.Int(name, field.Interface().(int), desc)
		case reflect.Int8:
			f.values[name] = f.flagSet.Int(name, int(field.Interface().(int8)), desc)
		case reflect.Int16:
			f.values[name] = f.flagSet.Int(name, int(field.Interface().(int16)), desc)
		case reflect.Int32:
			f.values[name] = f.flagSet.Int(name, int(field.Interface().(int32)), desc)
		case reflect.Int64:
			f.values[name] = f.flagSet.Int64(name, field.Interface().(int64), desc)
		case reflect.Uint:
			f.values[name] = f.flagSet.Uint(name, field.Interface().(uint), desc)
		case reflect.Uint8:
			f.values[name] = f.flagSet.Uint(name, uint(field.Interface().(uint8)), desc)
		case reflect.Uint16:
			f.values[name] = f.flagSet.Uint(name, uint(field.Interface().(uint16)), desc)
		case reflect.Uint32:
			f.values[name] = f.flagSet.Uint(name, uint(field.Interface().(uint32)), desc)
		case reflect.Uint64:
			f.values[name] = f.flagSet.Uint64(name, field.Interface().(uint64), desc)
		case reflect.String:
			f.values[name] = f.flagSet.String(name, field.Interface().(string), desc)
		case reflect.Bool:
			f.values[name] = f.flagSet.Bool(name, field.Interface().(bool), desc)
		case reflect.Float32:
			// note float32 has precision issues
			f.values[name] = f.flagSet.Float64(name, float64(field.Interface().(float32)), desc)
		case reflect.Float64:
			f.values[name] = f.flagSet.Float64(name, field.Interface().(float64), desc)
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
