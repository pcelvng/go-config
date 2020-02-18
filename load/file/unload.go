package file

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hydronica/toml"
	"github.com/pcelvng/go-config/load/env"
	"gopkg.in/yaml.v2"
)

func NewUnloader(ext string) *Unloader {
	return &Unloader{
		ext: ext,
	}
}

type Unloader struct {
	ext string // file extension.
}

// Unload a config to a file based on file extension.
func (u *Unloader) Unload(vs ...interface{}) ([]byte, error) {
	allB := make([]byte, 0)
	for _, v := range vs {
		b, err := unload(v, u.ext)
		if err != nil {
			return nil, err
		}

		allB = append(allB, b...)
	}

	return allB, nil
}

func unload(v interface{}, ext string) ([]byte, error) {
	switch ext {
	case "env":
		return env.Unload(v)
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
