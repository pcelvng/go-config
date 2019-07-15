package env

import (
	"bytes"
	"fmt"
	"reflect"
	"time"

	"github.com/iancoleman/strcase"
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
	fmt.Fprint(e.buf, "#!/bin/sh\n\n")
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
	typeCheck:
		// if the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// explicitly ignored list of types.
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		case reflect.String:
			e.write(name, field.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Type().String() == "time.Duration" {
				e.write(name, field.Interface().(time.Duration).String())
				continue
			}
			e.write(name, field.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			e.write(name, field.Uint())
		case reflect.Bool:
			e.write(name, field.Bool())
		case reflect.Float32, reflect.Float64:
			e.write(name, field.Float())

		case reflect.Struct:
			// time.Time special struct case
			if field.Type().String() == "time.Time" {
				// check for 'fmt' tag.
				timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)
				if timeFmt == "" {
					timeFmt = time.RFC3339
				}
				e.write(name, field.Interface().(time.Time).Format(timeFmt))
				continue
			}

		case reflect.Ptr:
			// if it's a ptr to a struct then recurse otherwise fallthrough
			if field.IsNil() {
				continue
				// should we set nil variables
			}

			field = field.Elem()
			// dereference and reprocess
			goto typeCheck
		}
	}

	return e.buf.Bytes(), nil
}

func (e *Encoder) write(field string, value interface{}) {
	fmt.Fprintf(e.buf, "%s=%v\n", field, value)
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
