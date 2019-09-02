package env

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
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
	fmt.Fprint(e.buf, "#!/usr/bin/env sh\n\n")
	return e.marshal("", v)
}

func (e *Encoder) marshal(prefix string, v interface{}) ([]byte, error) {
	err := e.writeAll("", v)
	if err != nil {
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func (e *Encoder) writeAll(prefix string, v interface{}) error {
	// Verify that v is struct pointer. Should not be nil.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))

		// Must be pointing to a struct.
	} else if pv := reflect.Indirect(value); pv.Kind() != reflect.Struct {
		return fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
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
		tag := vStruct.Type().Field(i).Tag.Get(envTag) // "env" tag value
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

		comment := vStruct.Type().Field(i).Tag.Get(helpTag) // "comment" tag value

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

		// If the value type is a struct or struct pointer then recurse.
		switch field.Kind() {
		// Explicitly ignored list of types.
		case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		case reflect.Struct:
			// time.Time special struct case.
			if field.Type().String() == "time.Time" {
				// Check for 'fmt' tag.
				timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)
				if timeFmt == "" {
					timeFmt = time.RFC3339
				}

				e.doWrite(name, comment, field.Interface().(time.Time).Format(timeFmt))
				continue
			}

			// Get a pointer and recurse.
			err := e.writeAll(name, field.Addr().Interface())
			if err != nil {
				return err
			}
		case reflect.Ptr:
			// If it's a ptr to a struct then recurse, otherwise fallthrough.
			if field.IsNil() {
				// initialize underlying struct.
				// TODO: consider a deep copy at the beginning of e.Marshal so the original struct is untouched.
				field.Set(reflect.New(field.Type().Elem()))
			}

			// Check if it's pointing to a struct.
			if reflect.Indirect(field).Kind() == reflect.Struct {
				if reflect.Indirect(field).Type().String() == "time.Time" {
					// TODO: add time format comment.
					// Check for 'fmt' tag.
					timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)
					if timeFmt == "" {
						timeFmt = time.RFC3339
					}

					e.doWrite(name, comment, field.Interface().(time.Time).Format(timeFmt))
					continue
				}

				// Recurse on ptr struct.
				err := e.writeAll(name, field.Interface())
				if err != nil {
					return err
				}

				continue // Important, so the fallthrough is not hit.
			}

			// Fallthrough, since the underlying type is not
			// a struct.
			fallthrough
		default:
			// Validate "omitprefix" usage.
			// Cannot be used on non-struct field types.
			if tag == "omitprefix" {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			e.writeFieldLine(name, comment, field)
		}
	}

	return nil
}

// writeFieldLine converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
//
// All structs that implement encoding.TextUnmarshaler are supported
//
// Does not support array literals.
func (e *Encoder) writeFieldLine(name, comment string, field reflect.Value) error {
	// TODO: handle formatting of zero values.
	switch field.Kind() {
	case reflect.String:
		e.doWrite(name, comment, field.String())
	case reflect.Bool:
		e.doWrite(name, comment, field.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Support case of int64 as a time.Duration.
		if field.Type().String() == "time.Duration" {
			e.doWrite(name, comment, field.Interface().(time.Duration).String())
			return nil
		}

		e.doWrite(name, comment, field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.doWrite(name, comment, field.Uint())
	case reflect.Float32, reflect.Float64:
		e.doWrite(name, comment, field.Float())
	case reflect.Ptr:
		if field.IsNil() {
			// Create non-pointer type and recursively assign.
			z := reflect.New(field.Type().Elem())
			err := e.writeFieldLine(name, comment, z.Elem())
			if err != nil {
				return err
			}
		}

		err := e.writeFieldLine(name, comment, reflect.Indirect(field))
		if err != nil {
			return err
		}
	case reflect.Slice:
		// Create a slice and recursively assign the elements.
		baseType := reflect.TypeOf(field.Interface()).Elem()

		// Handle empty slice - no defaults.
		if field.Len() == 0 {
			// TODO: make a note of underlying type?
			e.doWrite(name, comment, "[]")
		}

		// TODO: consider using native bash arrays. https://www.tldp.org/LDP/Bash-Beginners-Guide/html/sect_10_02.html
		outValue := "["
		sep := ","
		for i := 0; i < field.Len(); i++ {
			item := field.Index(i)

		typeCheck:
			switch baseType.Kind() {
			case reflect.String:
				outValue += item.String()
			case reflect.Bool:
				outValue += strconv.FormatBool(item.Bool())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				outValue += strconv.FormatInt(item.Int(), 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				outValue += strconv.FormatUint(item.Uint(), 10)
			case reflect.Float32, reflect.Float64:
				outValue += fmt.Sprintf("%f", item.Float())
			case reflect.Ptr:
				item = item.Elem()
				goto typeCheck
			default:
				// Skip all other types. Structs, for example, do not map.
				// Only simple types supported.
				return nil
			}

			if i < field.Len() {
				outValue += sep
			}
		}

		e.doWrite(name, comment, outValue+"]")

	// structs as values are simply ignored. They don't map cleanly for environment variables.
	case reflect.Struct:
		return nil
	default:
		return fmt.Errorf("unsupported type '%v'", field.Kind())
	}

	return nil
}

func (e *Encoder) doWrite(field, comment string, value interface{}) {
	if comment != "" {
		comment = " # " + comment
	}
	fmt.Fprintf(e.buf, "export %s=%v%v\n", field, value, comment)
}
