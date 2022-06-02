package flag

import (
	"github.com/pcelvng/go-config/util"
	"os"

	"github.com/pcelvng/go-config/util/node"
)

type Options struct {
	// HelpPreamble is optional text prepended to the help screen menu.
	HelpPreamble string

	// HelpPostamble is optional text appended to the generated help menu.
	HelpPostamble string

	// HelpFunc defines an optional custom help screen help menu render function to override the
	// default.
	HelpFunc GenHelpFunc
}

func NewLoader(o Options) *Loader {
	return &Loader{o: o}
}

func (l *Loader) WithPrefix(prefix string) *Loader {
	l.prefix = util.ToKebab(prefix)
	return l
}

type Loader struct {
	o      Options
	prefix string
}

func (l *Loader) Load(_ []byte, nGrps []*node.Nodes) error {
	fs, err := newFlagSet(l.o, nGrps)
	if err != nil {
		return err
	}

	fs.prefix = l.prefix
	// -help and -h are already reserved. The following
	// provides more support for "help" and "h"
	// without the dash "-" prefix.
	argList := os.Args[1:]
	if len(argList) > 0 && (argList[0] == "help" || argList[0] == "h") {
		fs.fs.Usage()
		os.Exit(0)
	}

	return fs.fs.Parse(os.Args[1:])
}
