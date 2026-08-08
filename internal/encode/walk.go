package encode

import (
	"encoding"
	"reflect"
)

// JoinPrefix joins parent and segment with sep when both are non-empty.
func JoinPrefix(parent, segment, sep string) string {
	if segment == "" {
		return parent
	}
	if parent == "" {
		return segment
	}
	return parent + sep + segment
}

// IsRecursableStruct reports whether field is a nested config struct to walk into.
// time.Time and encoding.TextMarshaler types are treated as leaves.
func IsRecursableStruct(field reflect.Value) bool {
	field = derefStructField(field)
	if field.Kind() != reflect.Struct {
		return false
	}
	if field.Type().String() == "time.Time" {
		return false
	}
	return !implementsTextMarshaler(field)
}

func derefStructField(field reflect.Value) reflect.Value {
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return reflect.New(field.Type().Elem()).Elem()
		}
		field = field.Elem()
	}
	return field
}

func implementsTextMarshaler(v reflect.Value) bool {
	t := v.Type()
	marshaler := reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	if t.Implements(marshaler) {
		return true
	}
	if v.CanAddr() && v.Addr().Type().Implements(marshaler) {
		return true
	}
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(marshaler) {
		return true
	}
	return false
}

// ShouldProcessField reports whether a struct field should be visited.
// Anonymous embedded structs are processed even when CanSet is false.
func ShouldProcessField(sf reflect.StructField, field reflect.Value) bool {
	if field.CanSet() {
		return true
	}
	return sf.Anonymous && IsRecursableStruct(field)
}

// StructValue returns the struct value to recurse into without modifying nil pointers.
func StructValue(field reflect.Value) (reflect.Value, bool) {
	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			return reflect.New(field.Type().Elem()).Elem(), true
		}
		return field.Elem(), true
	case reflect.Struct:
		return field, true
	default:
		return reflect.Value{}, false
	}
}

// InitStructValue returns the struct value to recurse into, allocating nil pointers when settable.
func InitStructValue(field reflect.Value) (reflect.Value, bool) {
	switch field.Kind() {
	case reflect.Ptr:
		if field.IsNil() {
			if !field.CanSet() {
				return reflect.Value{}, false
			}
			field.Set(reflect.New(field.Type().Elem()))
		}
		return field.Elem(), true
	case reflect.Struct:
		return field, true
	default:
		return reflect.Value{}, false
	}
}
