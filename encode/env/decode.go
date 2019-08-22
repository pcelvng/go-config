package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
)

var (
	envTag     = "env"    // Expected env struct tag name.
	configTag  = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag     = "fmt"
	commentTag = "comment" // Only used for encoding.
)

func New() *Decoder {
	return &Decoder{}
}

type Decoder struct{}

// Unmarshal implements the go-config/encoding.Unmarshaler interface.
func (d *Decoder) Unmarshal(v interface{}) error {
	return populate("", v)
}

// populate is a recursive function for populating struct values from env variables.
//
// The case-sensitive value of prefix is pre-pended to each returned expected env variable
// separated by an underscore '_'.
//
// If a struct pointer value is nil then the struct will be initialized and the struct pointer value
// populated.
func populate(prefix string, v interface{}) error {
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
					return fmt.Errorf("'%s' from '%s' cannot be set to %s (%s); using '%v' time format",
						envVal, name, vStruct.Type().Field(i).Name, field.Type(), timeFmt)
				}

				continue
			}

			// get a pointer and recurse
			err := populate(name, field.Addr().Interface())
			if err != nil {
				return err
			}
		case reflect.Ptr:
			// if it's a ptr to a struct then recurse otherwise fallthrough
			if field.IsNil() {
				z := reflect.New(field.Type().Elem())
				field.Set(z)
			}

			// check if it's pointing to a struct
			if reflect.Indirect(field).Kind() == reflect.Struct {
				if reflect.Indirect(field).Type().String() == "time.Time" {
					// check for 'fmt' tag.
					timeFmt := vStruct.Type().Field(i).Tag.Get(fmtTag)

					// get env value
					envVal := os.Getenv(name)

					// if no value found then don't set because it will
					// overwrite possible defaults.
					if envVal == "" {
						continue
					}

					timeFmt, err := setTime(reflect.Indirect(field), envVal, timeFmt)
					if err != nil {
						return fmt.Errorf("'%s' from '%s' cannot be set to %s (%s); using '%v' time format",
							envVal, name, vStruct.Type().Field(i).Name, field.Type(), timeFmt)
					}

					continue
				}

				// recurse on ptr struct
				err := populate(name, field.Interface())
				if err != nil {
					return err
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
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}

			// get env value
			envVal := os.Getenv(name)

			// if no value found then don't set because it will
			// overwrite possible defaults.
			if envVal == "" {
				continue
			}

			// set value to field.
			if err := setField(field, envVal); err != nil {
				return fmt.Errorf("'%s' from '%s' cannot be set to %s (%s)", envVal, name, vStruct.Type().Field(i).Name, field.Type())
			}
		}
	}

	return nil
}

// setTime expects value to be time.Time.
//
// tFmt can be any time package handy time format like "RFC3339Nano".
// Default format is time.RFC3339.
func setTime(value reflect.Value, tv, timeFmt string) (string, error) {
	if timeFmt == "" {
		timeFmt = time.RFC3339 // default format
	}

	// check for standard time formats
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

	t, err := time.Parse(timeFmt, tv)
	if err != nil {
		return timeFmt, err
	}

	tStruct := reflect.ValueOf(t)
	value.Set(tStruct)

	return timeFmt, nil
}

// setField converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
//
// All structs that implement encoding.TextUnmarshaler are supported
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
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// check for time.Duration int64 special case.
		//
		// TODO: check if this still works when time package is vendored or there is a way to fake this.
		if value.Type().String() == "time.Duration" {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}

			value.SetInt(int64(d))
			return nil
		}

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
		// Create non-pointer type and recursively assign.
		z := reflect.New(value.Type().Elem())
		err := setField(z.Elem(), s)
		if err != nil {
			return err
		}

		value.Set(z)
	case reflect.Slice:
		// TODO: underlying slice type must not be complex.
		// TODO: consider supporting native bash arrays.
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
