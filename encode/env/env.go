package env

import (
	"strings"

	"github.com/pcelvng/go-config/util/node"
)

var (
	envTag    = "env"    // Expected env struct tag name.
	configTag = "config" // Expected general config values (only "ignore" supported ATM).
	fmtTag    = "fmt"
	helpTag   = "help" // Only used for encoding.
	ignoreTag = "ignore"
	sepTag    = "sep" // separator for slice values.
)

// getEnvTag returns the 'env' tag value. It
// knows to exclude the supported ',string' suffix option if present.
func getEnvTag(n *node.Node) string {
	return strings.TrimRight(n.GetTag(envTag), ",string")
}
