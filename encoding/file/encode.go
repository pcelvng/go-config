package file

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hydronica/toml"
	"gopkg.in/yaml.v2"
)

// Encode a config to a file based on the ext passed in
// Note: only toml supports comments
func Encode(w io.Writer, i interface{}, ext string) error {
	switch ext {
	case "toml":
		enc := toml.NewEncoder(w)
		return enc.Encode(i)
	case "yaml", "yml":
		b, err := yaml.Marshal(i)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	case "json":
		b, err := json.MarshalIndent(i, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	default:
		return fmt.Errorf("unsupported config extension %s", ext)
	}
}
