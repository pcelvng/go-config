package flag

import (
	"os"

	"github.com/pcelvng/go-config/util/node"
)

type Options struct {
	HlpPreText  string      // Help text prepended to generated help menu.
	HlpPostText string      // Help text appended to generated help menu.
	HlpFunc     GenHelpFunc // Optional func to override the default help menu generator.
}

func NewLoader(o Options) *Loader {
	return &Loader{
		o:       o,
		hlpMsgs: make(map[string]string),
		ignore:  make([]string, 0),
	}
}

type Loader struct {
	o       Options
	hlpMsgs map[string]string
	ignore  []string
}

func (l *Loader) Load(nGrps []*node.Nodes) error {
	fs, err := newFlagSet(l.o, l.hlpMsgs, l.ignore, nGrps)
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
	l.hlpMsgs[fName] = helpMsg
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
