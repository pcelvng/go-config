package env

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pcelvng/go-config/util/node"
)

func NewEnvUnloader() *EnvUnloader {
	return &EnvUnloader{}
}

func (u *EnvUnloader) Unload(nss []*node.Nodes) ([]byte, error) {
	u.buf = &bytes.Buffer{}

	// Write env preamble.
	fmt.Fprint(u.buf, "#!/usr/bin/env sh\n\n")

	for _, ns := range nss {
		err := u.unload(ns)
		if err != nil {
			return nil, err
		}
	}

	return u.buf.Bytes(), nil
}

type EnvUnloader struct {
	buf *bytes.Buffer
}

func (u *EnvUnloader) unload(nodes *node.Nodes) error {
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
			return fmt.Errorf("'omitprefix' cannot be used on non-struct field types")
		}

		// Write line bytes to buffer.
		u.doWrite(genFullName(n, heritage), genHelp(n), toStr(n))
	}

	return nil
}

func genHelp(n *node.Node) string {
	helpMsg := n.GetTag(helpTag)

	if n.IsTime() {
		fmtV := n.GetTag(fmtTag)
		if fmtV == "" {
			fmtV = node.NormTimeFormat("")
		}
		fmtV = "fmt: " + fmtV

		if helpMsg == "" {
			helpMsg = fmtV
		} else {
			helpMsg = helpMsg + " " + fmtV
		}
	}

	return helpMsg
}

// toStr handles the converting an existing/default field
// value to a string as it would be represented as an env value.
//
// The value includes double quotes for fields with the ",string"
// env tag suffix.
func toStr(n *node.Node) string {
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

func (u *EnvUnloader) doWrite(field, comment string, value interface{}) {
	if comment != "" {
		comment = " # " + comment
	}
	fmt.Fprintf(u.buf, "export %s=%v%v\n", field, value, comment)
}
