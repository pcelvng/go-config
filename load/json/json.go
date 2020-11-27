package json

import (
	"encoding/json"

	"github.com/pcelvng/go-config/util/node"
)

func NewJSONLoadUnloader() *JSONLoadUnloader {
	return &JSONLoadUnloader{}
}

// JSONLoadUnloader implements the LoadUnloader interface for JSON configs.
type JSONLoadUnloader struct{}

// Load implements the Loader interface for loading a JSON config.
func (_ JSONLoadUnloader) Load(b []byte, nGrps []*node.Nodes) error {
	for _, nGrp := range nGrps {
		// Provide the underlying struct directly since this is
		// not a custom implementation relying on a third party.
		err := json.Unmarshal(b, nGrp.StructPtr())
		if err != nil {
			return err
		}
	}

	return nil
}

// Unload implements the Unloader interface for unloading a JSON config.
func (_ JSONLoadUnloader) Unload(nGrps []*node.Nodes) ([]byte, error) {
	allB := make([]byte, 0)
	for _, nGrp := range nGrps {
		b, err := json.Marshal(nGrp.StructPtr())
		if err != nil {
			return nil, err
		}

		allB = append(allB, b...)
	}

	return allB, nil
}
