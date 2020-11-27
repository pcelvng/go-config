package env

import "github.com/pcelvng/go-config/util/node"

func NewEnvLoadUnloader() *EnvLoadUnloader {
	return &EnvLoadUnloader{
		loader:   &EnvLoader{},
		unloader: &EnvUnloader{},
	}
}

// EnvLoadUnloader implements the LoadUnloader interface for environment configs.
type EnvLoadUnloader struct {
	loader   *EnvLoader
	unloader *EnvUnloader
}

// Load implements the EnvLoader interface for loading a TOML config.
func (lu *EnvLoadUnloader) Load(b []byte, nGrps []*node.Nodes) error {
	return lu.loader.Load(b, nGrps)
}

// Unload implements the EnvUnloader interface for unloading a TOML config.
func (lu *EnvLoadUnloader) Unload(nGrps []*node.Nodes) ([]byte, error) {
	return lu.unloader.Unload(nGrps)
}
