package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
)

// ScreamingSnake converts "name" to SCREAMING_SNAKE_CASE.
// Any periods "." are removed before conversion.
//
// Special thanks to iancoleman's strcase library.
// see: "github.com/iancoleman/strcase"
func ToScreamingSnake(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	return strcase.ToScreamingSnake(name)
}

// ToSnake converts "name" to snake_case.
// Any periods "." are removed before conversion.
//
// Special thanks to iancoleman's strcase library.
// see: "github.com/iancoleman/strcase"
func ToSnake(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	return strcase.ToSnake(name)
}

// ToKebab converts "name" to snake_case.
// Any periods "." are removed before conversion.
//
// Special thanks to iancoleman's strcase library.
// see: "github.com/iancoleman/strcase"
func ToKebab(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	return strcase.ToKebab(name)
}

// IsStructPointer is a utility that checks if a given
// interface is a struct pointer. If it is a struct pointer
// then true is returned with no error message. Otherwise false
// is returned with a specific error message indicating what type was passed.
func IsStructPointer(v interface{}) (bool, error) {
	// Verify that v is struct pointer. Should not be nil.
	if value := reflect.ValueOf(v); value.Kind() != reflect.Ptr || value.IsNil() {
		return false, fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(v))

		// Must be pointing to a struct.
	} else if pv := reflect.Indirect(value); pv.Kind() != reflect.Struct {
		return false, fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(v))
	}

	return true, nil
}
