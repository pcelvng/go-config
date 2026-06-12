package env

import (
	"fmt"
	"os"
	"reflect"

	"github.com/iancoleman/strcase"

	"github.com/hydronica/go-config/internal/encode"
)

func New() *Decoder {
	return &Decoder{
		GetVal: os.Getenv,
	}
}

type Decoder struct {
	GetVal func(key string) string
}

// Unmarshal implements the go-config/encoding.Unmarshaler interface.
func (d *Decoder) Unmarshal(v interface{}) error {
	if d.GetVal == nil {
		d.GetVal = os.Getenv
	}
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

		// Check general 'config' tag value. if it has a "ignore" value
		// then skip it entirely.
		if cfgV := vStruct.Type().Field(i).Tag.Get(encode.ConfigTag); cfgV == "ignore" {
			continue
		}

		// env tag name, if present, trumps the generated field name.
		//
		// If the field name is used it is converted to screaming snake case (uppercase with underscores).
		name := vStruct.Type().Field(i).Name
		tag := vStruct.Type().Field(i).Tag.Get(encode.EnvTag) // env tag value
		switch tag {
		case "-":
			continue // ignore field
		case "omitprefix":
			// Should only be used on struct field types, in
			// which case an existing prefix is passed through
			// to the struct fields. The immediate struct field
			// has no prefix.
			name = ""
		case "":
			name = strcase.ToScreamingSnake(name)
		default:
			name = tag
		}

		// if the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// explicity ignored list of types.
		case reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		default:
			// Validate "omitprefix" usage.
			// Cannot be used on non-struct field types.
			if tag == "omitprefix" {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			// get env value
			envVal := d.GetVal(name)

			// if no value found then don't set because it will
			// overwrite possible defaults.
			if envVal == "" {
				continue
			}
			// set value to field.
			if err := encode.SetField(field, envVal, vStruct.Type().Field(i)); err != nil {
				return fmt.Errorf("'%s' from '%s' cannot be set to %s (%s) %v", envVal, name, vStruct.Type().Field(i).Name, field.Type(), err)
			}
		}
	}

	return nil
}
