package env

import (
	"strings"

	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

var (
	envTag    = "env"    // Expected env struct tag name.
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
	helpTag   = "help" // Only used for encoding.
	ignoreTag = "ignore"
	sepTag    = "sep" // separator for slice values.

	defaultSep = "," // default separator for encoding/decoding slice values.
)

// genFullName generates the full env name including the prefix.
func genFullName(n *node.Node, heritage []*node.Node) (fullName string) {
	return genPrefix(append(heritage, n))
}

// genPrefix generates the env name prefix.
//
// 'heritage' is expected to be ordered from most to least distant relative.
func genPrefix(heritage []*node.Node) (prefix string) {
	for _, hn := range heritage {
		envName := nodeEnvName(hn)
		if envName == "" {
			continue
		}

		if prefix == "" {
			prefix = envName
		} else {
			prefix += "_" + envName
		}
	}

	return prefix
}

// nodeEnvName generates the env name of the node. Does
// not include the prefix.
func nodeEnvName(n *node.Node) string {
	ev := getEnvTag(n)
	switch ev {
	case "omitprefix":
		return ""
	case "":
		return util.ToScreamingSnake(n.FieldName())
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
// - `env:"-"`
func isIgnored(n *node.Node) bool {
	// "ignore" tag or "config" tag has ("ignore" value)
	if n.GetBoolTag(ignoreTag) ||
		n.GetTag(configTag) == "ignore" ||
		getEnvTag(n) == "-" {
		return true
	}

	return false
}

// getEnvTag returns the 'env' tag value. It
// knows to exclude the supported ',string' suffix option if present.
func getEnvTag(n *node.Node) string {
	return strings.TrimRight(n.GetTag(envTag), ",string")
}

// isEnvString returns true when the env tag value has the suffix ",string".
func isEnvString(n *node.Node) bool {
	return strings.HasSuffix(n.GetTag(envTag), ",string")
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
