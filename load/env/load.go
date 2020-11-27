package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/pcelvng/go-config/util/node"
)

var (
	loader = &EnvLoader{}
	Load   = loader.Load
)

type EnvLoader struct{}

// Load implements the go-config/load.EnvLoader interface.
//
// TODO: load env vars from a file (i.e. from bytes)
func (l *EnvLoader) Load(_ []byte, nGrps []*node.Nodes) error {
	for _, nGrp := range nGrps {
		err := load(nGrp)
		if err != nil {
			return err
		}
	}

	return nil
}

func load(nodes *node.Nodes) error {
	for _, n := range nodes.List() {
		heritage := node.Parents(n, nodes.Map())

		// Check if ignored or any parent(s) are ignored.
		//
		// Note that if this node or any ancestor node is ignored
		// then the result is the same - this node is ignored.
		if isAnyIgnored(append(heritage, n)) {
			continue
		}

		// Skip fields that are themselves structs (excluding special structs like time.Time).
		//
		// Note: for now time.Time is treated specifically. At some point we want to key
		// off something like non-stringer structs.
		if n.IsStruct() && !n.IsTime() {
			continue
		}

		// Validate that "omitprefix" is not used on value fields.
		if getEnvTag(n) == "omitprefix" {
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}

		// Set field from env value.
		err := setFieldValue(n, os.Getenv(genFullName(n, heritage)))
		if err != nil {
			return err
		}
	}

	return nil
}

// setFieldValue sets the field value. It takes into account
// special cases such as time.Time and slices.
//
// If 'envVal' is empty then nothing is set and nil is returned.
func setFieldValue(n *node.Node, envVal string) error {
	if envVal == "" {
		return nil
	}

	if n.IsTime() {
		_, err := n.SetTime(envVal, n.GetTag(fmtTag))
		return err
	} else if n.IsSlice() {
		return n.SetSlice(splitSlice(envVal, n.GetTag(sepTag), isEnvString(n)))
	}

	return n.SetFieldValue(envVal)
}

// splitSlice splits an env string.
// the 'isString' option reads in the values as possibly string quoted.
// the result is `"1"` is read in as `1` with the quotes stripped away
// before reading in the value.
//
// TODO: allow hook for a custom implementation of this function.
func splitSlice(envValue string, sep string, isString bool) []string {
	if sep == "" {
		sep = defaultSep
	}

	// Trim brackets for bracket support.
	vals := strings.Split(strings.Trim(envValue, "[]"), sep)

	// Trim out single and double quotes and spaces.
	for i := range vals {
		vals[i] = strings.TrimSpace(vals[i])
		if isString {
			// Strip away possible string quoted values.
			vals[i] = strings.Trim(vals[i], `"'`)
		}
	}

	return vals
}
