package env

import (
	"bytes"
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/hydronica/go-config/internal/encode"
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
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))
	}
	vStruct := val.Elem()
	if vStruct.Kind() != reflect.Struct {
		return nil, fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
	}
	e.buf.Reset()
	return e.marshal("", vStruct)
}

func (e *Encoder) marshal(prefix string, vStruct reflect.Value) ([]byte, error) {
	for i := 0; i < vStruct.NumField(); i++ {
		sf := vStruct.Type().Field(i)
		field := vStruct.Field(i)
		if !encode.ShouldProcessField(sf, field) {
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
				return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
			}
			childPrefix := prefix
			switch {
			case name == "omitprefix":
				childPrefix = ""
			case name != "":
				childPrefix = encode.JoinPrefix(prefix, name, "_")
			}
			if field.Kind() == reflect.Ptr && field.IsNil() {
				continue
			}
			child, ok := encode.StructValue(field)
			if !ok {
				continue
			}
			if _, err := e.marshal(childPrefix, child); err != nil {
				return nil, err
			}
			continue
		}

		if name == "omitprefix" {
			return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}
		segment := strcase.ToScreamingSnake(sf.Name)
		if name != "" {
			segment = name
		}
		key := encode.JoinPrefix(prefix, segment, "_")
		if omitPrefix {
			key = segment
		}
		if err := e.writeField(key, field, sf); err != nil {
			return nil, err
		}
	}

	return e.buf.Bytes(), nil
}

func (e *Encoder) writeField(name string, field reflect.Value, sf reflect.StructField) error {
typeCheck:
	switch field.Kind() {
	case reflect.Array, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Interface, reflect.Map:
		return nil
	case reflect.String:
		e.write(name, field.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type().String() == "time.Duration" {
			e.write(name, field.Interface().(time.Duration).String())
			return nil
		}
		e.write(name, field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.write(name, field.Uint())
	case reflect.Bool:
		e.write(name, field.Bool())
	case reflect.Float32, reflect.Float64:
		e.write(name, field.Float())
	case reflect.Struct:
		if field.Type().String() == "time.Time" {
			timeFmt := sf.Tag.Get(encode.FormatTag)
			if timeFmt == "" {
				timeFmt = time.RFC3339
			}
			e.write(name, field.Interface().(time.Time).Format(timeFmt))
			return nil
		}
		if b, ok := marshalText(field); ok {
			e.write(name, string(b))
			return nil
		}
	case reflect.Ptr:
		if field.IsNil() {
			return nil
		}
		field = field.Elem()
		goto typeCheck
	case reflect.Slice:
		e.write(name, field.Interface())
	}
	return nil
}

func marshalText(field reflect.Value) ([]byte, bool) {
	if field.CanInterface() {
		if m, ok := field.Interface().(encoding.TextMarshaler); ok {
			b, err := m.MarshalText()
			return b, err == nil
		}
	}
	if field.CanAddr() {
		if m, ok := field.Addr().Interface().(encoding.TextMarshaler); ok {
			b, err := m.MarshalText()
			return b, err == nil
		}
	}
	return nil, false
}

func (e *Encoder) write(field string, value interface{}) {
	fmt.Fprintf(e.buf, "%s=%s\n", field, stringifyForEnv(value))
}

func stringifyForEnv(v interface{}) string {
	switch x := v.(type) {
	case string:
		return formatEnvScalar(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	default:
		return formatEnvScalar(fmt.Sprint(v))
	}
}
