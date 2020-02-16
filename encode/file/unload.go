package file

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hydronica/toml"
	"github.com/pcelvng/go-config/encode/env"
	"gopkg.in/yaml.v2"
)

// Unload a config to a file based on file extension.
func Unload(v interface{}, ext string) ([]byte, error) {
	switch ext {
	// TODO: cleanup
	case "env": // TODO: env is not a file type.
		return env.NewUnloader().Unload(v)
	case "json":
		return json.MarshalIndent(v, "", "  ")
	case "toml":
		buf := &bytes.Buffer{}
		err := toml.NewEncoder(buf).Encode(v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "yaml", "yml":
		return yaml.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported config extension %s", ext)
	}
}
