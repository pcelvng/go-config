package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hydronica/toml"
	"gopkg.in/yaml.v2"
)

// Load config from file, type is determined by the file extension
func Load(f string, i interface{}) error {
	switch filepath.Ext(f) {
	case "toml":
		_, err := toml.DecodeFile(f, i)
		return err
	case "json":
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, i)
	case "yaml":
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		return yaml.Unmarshal(b, i)
	default:
		return fmt.Errorf("unknown file type %s", filepath.Ext(f))
	}
}
