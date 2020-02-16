package flg

import (
	"strings"

	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

var (
	flagTag   = "flag"   // Expected flag struct tag name.
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
	helpTag   = "help" // Only used for encoding.
	ignoreTag = "ignore"
	sepTag    = "sep" // separator for slice values.

	defaultSep = "," // default separator for encoding/decoding slice values.
)

func registerFlagSet() {
}

// genFullName generates the full flag name including the prefix.
func genFullName(n *node.Node, heritage []*node.Node) (fullName string) {
	return genPrefix(append(heritage, n))
}

// genPrefix generates the flag name prefix.
//
// 'heritage' is expected to be ordered from most to least distant relative.
func genPrefix(heritage []*node.Node) (prefix string) {
	for _, hn := range heritage {
		flagName := nodeFlagName(hn)
		if flagName == "" {
			continue
		}

		if prefix == "" {
			prefix = flagName
		} else {
			prefix += "-" + flagName
		}
	}

	return prefix
}

// nodeFlagName generates the flag name of the node. Does
// not include the prefix.
func nodeFlagName(n *node.Node) string {
	ev := getFlagTag(n)
	switch ev {
	case "omitprefix":
		return ""
	case "":
		return util.ToKebab(n.FieldName())
	default:
		return ev
	}
}

// isAnyIgnored checks if any members of 'nodes' is ignored.
// if so, then returns true.
func isAnyIgnored(nodes []*node.Node) bool {
	for _, n := range nodes {
		if isIgnored(n) {
			return true
		}
	}

	return false
}

// isIgnored checks if the node is ignored.
//
// A node is ignored when one or more of the following struct
// field tag cases are met:
// - `ignore:"true"`
// - `config:"ignore"`
// - `flag:"-"`
func isIgnored(n *node.Node) bool {
	// "ignore" tag or "config" tag has ("ignore" value)
	if n.GetBoolTag(ignoreTag) ||
		n.GetTag(configTag) == "ignore" ||
		getFlagTag(n) == "-" {
		return true
	}

	return false
}

// getFlagTag returns the 'flag' tag value. It
// knows to exclude the supported ',string' suffix option if present.
func getFlagTag(n *node.Node) string {
	return strings.TrimRight(n.GetTag(flagTag), ",string")
}

// isFlagString returns true when the env tag value has the suffix ",string".
func isFlagString(n *node.Node) bool {
	return strings.HasSuffix(n.GetTag(flagTag), ",string")
}

// getSep returns the separator designed to be used for
// the struct field node if one is provided. If a separator is not
// provided then the default separator is returned.
func getSep(n *node.Node) string {
	sep := n.GetTag(sepTag)
	if sep == "" {
		sep = defaultSep
	}

	return sep
}
