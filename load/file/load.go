package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hydronica/toml"
	"gopkg.in/yaml.v2"
)

func NewLoader(fPath string) *Loader {
	return &Loader{
		fPath: fPath,
	}
}

// Loader represents a loader that loads from a config file such
// as a toml, yaml or json.
type Loader struct {
	fPath string // File path.
}

// Load config from file, type is determined by the file extension.
func (l *Loader) Load(vs ...interface{}) error {
	for _, v := range vs {
		err := l.load(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) load(v interface{}) error {
	switch strings.Trim(filepath.Ext(l.fPath), ".") {
	case "toml":
		_, err := toml.DecodeFile(l.fPath, v)
		return err
	case "json":
		b, err := ioutil.ReadFile(l.fPath)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, v)
	case "yaml", "yml":
		b, err := ioutil.ReadFile(l.fPath)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, v)
	default:
		return fmt.Errorf("unknown file type %s", filepath.Ext(l.fPath))
	}
}

// todo: issue how to properly handle custom formats for time.Time 'fmt' in json, yaml and toml
