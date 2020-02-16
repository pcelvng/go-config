package env

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pcelvng/go-config/util/node"
)

func NewUnloader() *Unloader {
	return &Unloader{
		buf: &bytes.Buffer{},
	}
}

type Unloader struct {
	buf *bytes.Buffer
}

func (e *Unloader) Unload(v interface{}) ([]byte, error) {
	return e.marshal(v)
}

func (e *Unloader) marshal(v interface{}) ([]byte, error) {
	// Write env preamble.
	fmt.Fprint(e.buf, "#!/usr/bin/env sh\n\n")

	nodes := node.MakeNodes(v, node.Options{})
	for _, n := range nodes.List() {
		heritage := node.Parents(n, nodes.Map())

		// Check if ignored or any parent(s) are ignored.
		//
		// Note that if this node or any ancestor node is ignored
		// then the res	ult is the same - this node is ignored.
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
			return nil, fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}

		// Write line bytes to buffer.
		e.doWrite(genFullName(n, heritage), n.GetTag(helpTag), fieldString(n))
	}

	return e.buf.Bytes(), nil
}

// fieldString handles the converting an existing/default field
// value to a string as it would be represented as an env value.
//
// The value includes double quotes for fields with the ",string"
// env tag suffix.
func fieldString(n *node.Node) string {
	if n.IsTime() {
		return n.TimeString(n.GetTag(fmtTag))
	} else if n.IsSlice() {
		vals := n.SliceString()
		if isEnvString(n) {
			for i := range vals {
				vals[i] = `"` + vals[i] + `"`
			}
		}

		return `[` + strings.Join(vals, getSep(n)) + `]`
	}

	val := n.String()
	if isEnvString(n) {
		val = `"` + val + `"`
	}

	return val
}

func (e *Unloader) doWrite(field, comment string, value interface{}) {
	if comment != "" {
		comment = " # " + comment
	}
	fmt.Fprintf(e.buf, "export %s=%v%v\n", field, value, comment)
}
