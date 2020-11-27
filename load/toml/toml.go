package toml

import (
	"bytes"

	"github.com/hydronica/toml"
	"github.com/pcelvng/go-config/util/node"
)

func NewTOMLLoadUnloader() *TOMLLoadUnloader {
	return &TOMLLoadUnloader{}
}

// TOMLLoadUnloader implements the LoadUnloader interface for TOML configs.
type TOMLLoadUnloader struct{}

// Load implements the Loader interface for loading a TOML config.
func (_ TOMLLoadUnloader) Load(b []byte, nGrps []*node.Nodes) error {
	for _, nGrp := range nGrps {
		// Provide the underlying struct directly since this is
		// not a custom implementation relying on a third party.
		_, err := toml.Decode(string(b), nGrp.StructPtr())
		if err != nil {
			return err
		}
	}

	return nil
}

// Unload implements the Unloader interface for unloading a TOML config.
func (_ TOMLLoadUnloader) Unload(nGrps []*node.Nodes) ([]byte, error) {
	buf := &bytes.Buffer{}
	tEnc := toml.NewEncoder(buf)
	for _, nGrp := range nGrps {
		err := tEnc.Encode(nGrp.StructPtr())
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
