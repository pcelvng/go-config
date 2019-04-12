package flg

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// setField converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
//
// All structs and that implement encoding.TextUnmarshaler are supported
//
// Does not support array literals.
func setField(value reflect.Value, s string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(s)
	case reflect.Bool:
		switch strings.ToLower(s) {
		case "true":
			value.SetBool(true)
		case "false", "":
			value.SetBool(false)
		default:
			// the bool value should be explicit to tell user
			// something is amiss.
			return fmt.Errorf("cannot assign '%v' to bool type", s)
		}
	case reflect.Int64:
		// check for time.Duration int64 special case.
		//
		// TODO: check if this still works when time package is vendored or there is a way to fake this.
		if value.Type().String() == "time.Duration" {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}

			value.SetInt(int64(d))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:

		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetUint(i)
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
		s = strings.Trim(s, "[]") // trim brackets for bracket support.
		vals := strings.Split(s, ",")

		slice := reflect.MakeSlice(value.Type(), 0, len(vals))
		for _, v := range vals {
			// trim whitespace from each value to support comma-separated with spaces.
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)

			// each item must be the correct type.
			baseValue := reflect.New(baseType).Elem()
			err := setField(baseValue, v)
			if err != nil {
				return err
			}
			slice = reflect.Append(slice, baseValue)
		}

		value.Set(slice)

	// structs as values are simply ignored. They don't map cleanly for environment variables.
	case reflect.Struct:
		return nil
	default:
		return fmt.Errorf("unsupported type '%v'", value.Kind())
	}

	return nil
}
