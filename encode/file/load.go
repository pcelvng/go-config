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
func (fl *Loader) Load(i interface{}) error {
	switch strings.Trim(filepath.Ext(fl.fPath), ".") {
	case "toml":
		_, err := toml.DecodeFile(fl.fPath, i)
		return err
	case "json":
		b, err := ioutil.ReadFile(fl.fPath)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, i)
	case "yaml", "yml":
		b, err := ioutil.ReadFile(fl.fPath)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, i)
	default:
		return fmt.Errorf("unknown file type %s", filepath.Ext(fl.fPath))
	}
}

// todo: issue how to properly handle custom formats for time.Time 'fmt' in json, yaml and toml
