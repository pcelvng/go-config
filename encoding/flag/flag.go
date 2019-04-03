package flag

import (
	"flag"
	"reflect"
)

const (
	flagTag = "flag"
	descTag = "comment" // do we want a different tag for the flag vs toml?
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
		name := dField.Name
		desc := dField.Tag.Get(descTag)
		if tag != "" {
			name = tag
		}

		// skip private variables and disabled flags
		if tag == "-" || !field.CanSet() {
			continue
		}

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
			// todo deference and handle. use recursition
		case reflect.Struct:
			// todo check if struct implements TextMarshaler
			// check if struct is time.Duration or time.Time
		}
	}
	return f, nil
}

// ToFlagName converts variable names to the flag syntax.
// names are split based on Capitals and numbers and separated with a dash -
func ToFlagName(s string) string {
	return s
}
