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
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))
	}
	vStruct := val.Elem()
	if vStruct.Kind() != reflect.Struct {
		return fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
	}
	return d.populate("", vStruct)
}

func (d *Decoder) populate(prefix string, vStruct reflect.Value) error {
	for i := 0; i < vStruct.NumField(); i++ {
		sf := vStruct.Type().Field(i)
		field := vStruct.Field(i)
		if !encode.ShouldProcessField(sf, field) {
			continue
		}

		switch field.Kind() {
		case reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		}

		if sf.Tag.Get(encode.ConfigTag) == "ignore" {
			continue
		}
		raw := sf.Tag.Get(encode.EnvTag)
		if raw == "-" {
			continue
		}
		name, omitPrefix, _ := encode.ParseTag(raw)

		if encode.IsRecursableStruct(field) {
			if omitPrefix {
				return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, "_")
			}
			if !subtreeHasEnvValues(d.GetVal, childPrefix, field) {
				continue
			}
			child, ok := encode.InitStructValue(field)
			if !ok {
				continue
			}
			if err := d.populate(childPrefix, child); err != nil {
				return err
			}
			continue
		}

		if name == "omitprefix" {
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}
		segment := strcase.ToScreamingSnake(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, "_")
		if omitPrefix {
			key = segment
		}
		envVal := d.GetVal(key)
		if envVal == "" {
			continue
		}
		if err := encode.SetField(field, envVal, sf); err != nil {
			return fmt.Errorf("'%s' from '%s' cannot be set to %s (%s) %v", envVal, key, sf.Name, field.Type(), err)
		}
	}

	return nil
}

func subtreeHasEnvValues(getVal func(string) string, prefix string, field reflect.Value) bool {
	child, ok := encode.StructValue(field)
	if !ok {
		return false
	}
	for i := 0; i < child.NumField(); i++ {
		sf := child.Type().Field(i)
		f := child.Field(i)
		if !encode.ShouldProcessField(sf, f) {
			continue
		}
		switch f.Kind() {
		case reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
			continue
		}
		if sf.Tag.Get(encode.ConfigTag) == "ignore" {
			continue
		}
		raw := sf.Tag.Get(encode.EnvTag)
		if raw == "-" {
			continue
		}
		name, omitPrefix, _ := encode.ParseTag(raw)

		if encode.IsRecursableStruct(f) {
			if omitPrefix {
				continue
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, "_")
			}
			if subtreeHasEnvValues(getVal, childPrefix, f) {
				return true
			}
			continue
		}

		if name == "omitprefix" {
			continue
		}
		segment := strcase.ToScreamingSnake(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, "_")
		if omitPrefix {
			key = segment
		}
		if getVal(key) != "" {
			return true
		}
	}
	return false
}
