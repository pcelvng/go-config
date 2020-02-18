package flag

import (
	"os"
)

func NewLoader(helpPreamble string) *Loader {
	return &Loader{
		helpPreamble: helpPreamble,
		helpMsgs:     make(map[string]string),
		ignore:       make([]string, 0),
	}
}

type Loader struct {
	helpPreamble string
	helpMsgs     map[string]string
	ignore       []string
}

func (l *Loader) Load(vs ...interface{}) error {
	fs, err := newFlagSet(l.helpPreamble, l.helpMsgs, l.ignore, vs...)
	if err != nil {
		return err
	}

	// -help and -h are already reserved. The following
	// provides more support for "help" and "h"
	// without the dash "-" prefix.
	argList := os.Args[1:]
	if len(argList) > 0 && (argList[0] == "help" || argList[0] == "h") {
		fs.fs.Usage()
		os.Exit(2)
	}

	return fs.fs.Parse(os.Args[1:])
}

// SetHelp will override an existing field "help" value or create
// one if not provided as a tag value.
//
// Useful for dynamically created help messages or adding help messages
// to struct fields a user does not control.
//
// SetHelp must be called before "Load" to be effective.
func (l *Loader) SetHelp(fName, helpMsg string) {
	l.helpMsgs[fName] = helpMsg
}

// IgnoreField works by addressing the struct field name as a string.
// Fields in embedded structs can be addressed by separating field names with a ".".
//
// Ignore must be called before "Load".
//
// Example:
// - "MyField.EmbeddedField"
func (l *Loader) IgnoreField(fName string) {
	l.ignore = append(l.ignore, fName)
}
