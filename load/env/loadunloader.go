package env

import "github.com/pcelvng/go-config/util/node"

func New() *ENVLoadUnloader {
	return &ENVLoadUnloader{
		loader:   &EnvLoader{},
		unloader: &EnvUnloader{},
	}
}

// ENVLoadUnloader implements the LoadUnloader interface for environment configs.
type ENVLoadUnloader struct {
	loader   *EnvLoader
	unloader *EnvUnloader
}

// Load implements the EnvLoader interface for loading a TOML config.
func (lu *ENVLoadUnloader) Load(b []byte, nGrps []*node.Nodes) error {
	return lu.loader.Load(b, nGrps)
}

// Unload implements the EnvUnloader interface for unloading a TOML config.
func (lu *ENVLoadUnloader) Unload(nGrps []*node.Nodes) ([]byte, error) {
	return lu.unloader.Unload(nGrps)
}
