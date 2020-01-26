package util

import (
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
