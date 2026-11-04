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
// time.Time and types that implement encoding.TextMarshaler or encoding.TextUnmarshaler
// are treated as leaves so single-key SetField / MarshalText paths still run.
func IsRecursableStruct(field reflect.Value) bool {
	field = derefStructField(field)
	if field.Kind() != reflect.Struct {
		return false
	}
	if field.Type().String() == "time.Time" {
		return false
	}
	return !implementsIface(field, reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()) &&
		!implementsIface(field, reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem())
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

func implementsIface(v reflect.Value, iface reflect.Type) bool {
	t := v.Type()
	if t.Implements(iface) {
		return true
	}
	if v.CanAddr() && v.Addr().Type().Implements(iface) {
		return true
	}
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(iface) {
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
