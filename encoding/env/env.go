package env

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	envTag = "env" // Expected env struct tag name.
	hideTag = "hide" // Expected hide struct tag name.

	// list of characters that are not allowed in an env name.
	envInvalidChars = []byte{
		'=',
		'\x00', // NUL character
	}

	// Default separator for environment vars representing slices.
	defSep = ","
)

func New() *Decoder {
	return &Decoder{

	}
}

type Decoder struct {}

// Unmarshal implements the go-config/encoding.Unmarshaler interface.
func (d *Decoder) Unmarshal(v interface{}) error {
	return populate("", v)
}

// populate is a recursive function for generating names of expected env variables.
// The case-sensitive value of prefix is pre-pended to each returned expected env variable
// separated by an underscore '_'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func populate(prefix string, v interface{}) error {
	// verify that v is struct pointer. Should always be a pointer.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("%v must be a non nil pointer", reflect.TypeOf(v))
	}

	// iterate through struct fields.
	vStruct := reflect.ValueOf(v).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)

		if !field.CanSet() { // skip private variables
			continue
		}

		// tag name, if present, trumps the generated field name.
		//
		// If the field name is used it is converted to screaming snake case (uppercase with underscores).
		name := vStruct.Type().Field(i).Name
		tag := vStruct.Type().Field(i).Tag.Get(envTag) // env tag value
		if tag != "" {
			name = tag
		} else {
			// convert default name to screaming snake case.
			name = strcase.ToScreamingSnake(name)
		}

		// prepend prefix
		if prefix != "" {
			// yes - an existing underscore means there will be 2 underscores. The user is given almost full reign on
			// naming as long as it's valid.
			name = prefix + "_" + name
		}

		// get env value
		envVal := os.Getenv(name)

		// set value to field.
		if err := setField(field, envVal); err != nil {
			return fmt.Errorf("%s can not be set to %s (%s)", envVal, name, field.Type())
		}
	}

	return nil
}

// setField converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
//
// All structs and that implement encoding.TextUnmarshaler are supported
func setField(value reflect.Value, s string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(s)
	case reflect.Bool:
		b := strings.ToLower(s) == "true" || s == ""
		value.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		value.SetInt(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 0)
		if err != nil {
			return err
		}
		value.SetFloat(f)
	case reflect.Ptr:
		// create non pointer type and recursively assign
		z := reflect.New(value.Type().Elem())
		err := setField(z.Elem(), s)
		if err != nil {
			return err
		}

		value.Set(z)
	case reflect.Slice:
		// create a slice and recursively assign the elements
		baseType := reflect.TypeOf(value.Interface()).Elem()
		vals := strings.Split(s, defSep)

		slice := reflect.MakeSlice(value.Type(), len(vals), len(vals))
		for _, v := range vals {
			// each item must be the correct type.
			baseValue := reflect.New(baseType).Elem()
			err := setField(baseValue, v)
			if err != nil {
				return err
			}
			slice = reflect.Append(slice, baseValue)
		}

		value.Set(slice)
	// struct is a special case that is handled elsewhere.
	// maybe only handle structs that can be expressed as a single
	// value such as time.Time or other structs with a custom UnmarshalText
	// method.
	//case reflect.Struct:
	//	v := reflect.New(value.Type())
	//	value.Set(v.Elem())

	default:
		return fmt.Errorf("Unsupported type %v", value.Kind())
	}

	return nil
}