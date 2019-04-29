package env

import (
	"bytes"
	"fmt"
	"github.com/iancoleman/strcase"
	"os"
	"reflect"
)

func NewEncoder() *Encoder {
	return &Encoder{
		buf: &bytes.Buffer{},
	}
}

type Encoder struct {
	buf *bytes.Buffer
}

func (e *Encoder) Marshal(v interface{}) ([]byte, error) {
	return e.marshal("", v)
}

func (e *Encoder) marshal(prefix string, v interface{}) ([]byte, error) {
	// Verify that v is struct pointer. Should not be nil.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))

		// Must be pointing to a struct.
	} else if pv := reflect.Indirect(value); pv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
	}

	// iterate through the struct field.
	vStruct := reflect.ValueOf(v).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		field := vStruct.Field(i)

		if !field.CanSet() { // skip private variables
			continue
		}

		// Check general 'config' tag value. if it has a "ignore" value
		// then skip it entirely.
		if cfgV := vStruct.Type().Field(i).Tag.Get(configTag); cfgV == "ignore" {
			continue
		}

		// env tag name, if present, trumps the generated field name.
		//
		// If the field name is used it is converted to screaming snake case (uppercase with underscores).
		name := vStruct.Type().Field(i).Name
		tag := vStruct.Type().Field(i).Tag.Get(envTag) // env tag value
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

		// prepend prefix
		if prefix != "" {
			// An empty name takes on the prefix so that
			// it can passthrough if the type is a struct or pointer struct.
			if name == "" {
				name = prefix
			} else {
				// An existing underscore means there will be 2 underscores. The user is given almost full reign on
				// naming as long as it's valid.
				name = prefix + "_" + name
			}
		}

		// if the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// explicity ignored list of types.
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		case reflect.Struct:
			// time.Time special struct case
			if field.Type().String() == "time.Time" {
				// check for 'fmt' tag.
				timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)

				// get env value
				envVal := os.Getenv(name)

				// if no value found then don't set because it will
				// overwrite possible defaults.
				if envVal == "" {
					continue
				}

				timeFmt, err := setTime(field, envVal, timeFmt)
				if err != nil {
					return nil, fmt.Errorf("'%s' from '%s' cannot be set to %s (%s); using '%v' time format",
						envVal, name, vStruct.Type().Field(i).Name, field.Type(), timeFmt)
				}

				continue
			}

			// get a pointer and recurse
			err := populate(name, field.Addr().Interface())
			if err != nil {
				return nil, err
			}
		case reflect.Ptr:
			// if it's a ptr to a struct then recurse otherwise fallthrough
			if field.IsNil() {
				z := reflect.New(field.Type().Elem())
				field.Set(z)
			}
2
			// check if it's pointing to a struct
			if reflect.Indirect(field).Kind() == reflect.Struct {
				if reflect.Indirect(field).Type().String() == "time.Time" {
					// check for 'fmt' tag.
					//timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)

					// get env value
					//envVal := os.Getenv(name)

					// if no value found then don't set because it will
					// overwrite possible defaults.
					//if envVal == "" {
					//	continue
					//}
					//
					//timeFmt, err := setTime(reflect.Indirect(field), envVal, timeFmt)
					//if err != nil {
					//	return nil, fmt.Errorf("'%s' from '%s' cannot be set to %s (%s); using '%v' time format",
					//		envVal, name, vStruct.Type().Field(i).Name, field.Type(), timeFmt)
					//}

					continue
				}

				// recurse on ptr struct
				err := populate(name, field.Interface())
				if err != nil {
					return nil, err
				}

				continue
			}

			// fallthrough since the underlying type is not
			// a struct.
			fallthrough
		default:
			// Validate "omitprefix" usage.
			// Cannot be used on non-struct field types.
			if tag == "omitprefix" {
				return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			// get env value
			//envVal := os.Getenv(name)

			// if no value found then don't set because it will
			// overwrite possible defaults.
			//if envVal == "" {
			//	continue
			//}

			// set value to field.
			//if err := setField(field, envVal); err != nil {
			//	return nil, fmt.Errorf("'%s' from '%s' cannot be set to %s (%s)", envVal, name, vStruct.Type().Field(i).Name, field.Type())
			//}


		}
	}

	return nil, nil
}

// writeLine will write template info for a single env variable.
// Short desc are written inline with the variable; long descriptions
// are written above with an additional newline above.
//
// Supported field tags (fTags)
// - desc (field description)
// - env (contains expected field name)
// - fmt (time format - only applies to time.Time fields)
// - req (if the field is required; true=required, false=not required)
func (e *Encoder) writeEnv(fTags map[string]string, fv reflect.Value) {

}

