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
	"github.com/jbsmith7741/go-tools/appenderr"
)

const (
	flagTag   = "flag"
	descTag   = "comment" // do we want a different tag for the flag vs toml?
	fmtTag    = "fmt"
	configTag = "config"
)

type Flags struct {
	*flag.FlagSet
}

// New creates a custom flagset based on the struct i.
//
func New(i interface{}) (*Flags, error) {
	if !isValidConfig(i) {
		return nil, errors.New("invalid config")
	}
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	vStruct := reflect.ValueOf(i).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)
		dField := vStruct.Type().Field(i)
		tag := dField.Tag.Get(flagTag)
		name := strcase.ToKebab(dField.Name)
		desc := dField.Tag.Get(descTag)
		confTag := dField.Tag.Get(configTag)
		if tag == "" {
			tag = name
		}

		// skip private variables and disabled flags
		if tag == "-" || confTag == "ignore" || !field.CanSet() {
			continue
		}

		if isAlias(field) {
			if field.Type().String() == "time.Duration" {
				d := field.Interface().(time.Duration)
				flagSet.String(tag, d.String(), desc)
				continue
			}
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				flagSet.String(tag, string(b), desc)
				continue
			}
		}
	switchStart:
		switch field.Kind() {
		// explicit list of unsupported types
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map, reflect.Slice:
			continue
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			flagSet.Int(tag, int(field.Int()), desc)
		case reflect.Int64:
			flagSet.Int64(tag, field.Int(), desc)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			flagSet.Uint(tag, uint(field.Uint()), desc)
		case reflect.Uint64:
			flagSet.Uint64(tag, field.Uint(), desc)
		case reflect.String:
			flagSet.String(tag, field.String(), desc)
		case reflect.Bool:
			flagSet.Bool(tag, field.Bool(), desc)
		case reflect.Float32, reflect.Float64:
			flagSet.Float64(tag, field.Float(), desc)
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
				flagSet.String(tag, t.Format(timeFmt), desc)
				continue
			}
			if implementsMarshaler(field) {
				b, _ := field.Interface().(encoding.TextMarshaler).MarshalText()
				flagSet.String(tag, string(b), desc)
			}
		}
	}
	return &Flags{flagSet}, nil
}

// Parse the internal flags and the user defined flags
func (f *Flags) Parse() error {

	// add other defined flags
	flag.VisitAll(func(flg *flag.Flag) {
		f.Var(flg.Value, flg.Name, flg.Usage)
	})

	return f.FlagSet.Parse(os.Args[1:])
}

// Unmarshal the given struct from the flagSet
func (f Flags) Unmarshal(i interface{}) error {
	if !isValidConfig(i) {
		return errors.New("invalid config")
	}

	vStruct := reflect.ValueOf(i).Elem()
	errs := appenderr.New()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)
		dField := vStruct.Type().Field(i)
		tag := dField.Tag.Get(flagTag)
		name := strcase.ToKebab(dField.Name)
		confTag := dField.Tag.Get(configTag)
		if tag != "" {
			name = tag
		}

		// skip private variables and disabled flags
		if tag == "-" || confTag == "ignore" || !field.CanSet() {
			continue
		}
		flg := f.FlagSet.Lookup(name)
		if flg == nil {
			return fmt.Errorf("matching flag not found %s", name)
		}
		errs.Add(setField(field, flg.Value.String()))
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
