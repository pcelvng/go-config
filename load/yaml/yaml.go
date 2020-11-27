package yaml

import (
	"github.com/pcelvng/go-config/util/node"
	"gopkg.in/yaml.v2"
)

func NewYAMLLoadUnloader() *YAMLLoadUnloader {
	return &YAMLLoadUnloader{}
}

// YAMLLoadUnloader implements the LoadUnloader interface for YAML configs.
type YAMLLoadUnloader struct{}

// Load implements the Loader interface for loading a YAML config.
func (_ YAMLLoadUnloader) Load(b []byte, nGrps []*node.Nodes) error {
	for _, nGrp := range nGrps {
		err := yaml.Unmarshal(b, nGrp.StructPtr())
		if err != nil {
			return err
		}
	}

	return nil
}

// Unload implements the Unloader interface for unloading a YAML config.
func (_ YAMLLoadUnloader) Unload(nGrps []*node.Nodes) ([]byte, error) {
	allB := make([]byte, 0)
	for _, nGrp := range nGrps {
		b, err := yaml.Marshal(nGrp.StructPtr())
		if err != nil {
			return nil, err
		}

		allB = append(allB, b...)
	}

	return allB, nil
}
