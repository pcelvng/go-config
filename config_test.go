package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadFromFileExtensions verifies that config files are loaded based on
// their file extension regardless of the leading dot returned by path.Ext.
//
// Regression test for https://github.com/pcelvng/go-config/issues/37 where
// loading a ".yml"/".yaml"/etc. file failed with:
//
//	loader not found for file extension '..yml'
func TestLoadFromFileExtensions(t *testing.T) {
	type sub struct {
		Endpoint string `yaml:"endpoint" toml:"endpoint" json:"endpoint"`
	}
	type cfg struct {
		Name string `yaml:"name" toml:"name" json:"name"`
		Sub  sub    `yaml:"sub" toml:"sub" json:"sub"`
	}

	cases := []struct {
		name     string
		file     string
		contents string
		loader   string
	}{
		{
			name:     "yml",
			file:     "config.yml",
			contents: "name: from-yaml\nsub:\n  endpoint: yaml-endpoint\n",
			loader:   "yaml",
		},
		{
			name:     "yaml",
			file:     "config.yaml",
			contents: "name: from-yaml\nsub:\n  endpoint: yaml-endpoint\n",
			loader:   "yaml",
		},
		{
			name:     "toml",
			file:     "config.toml",
			contents: "name = \"from-toml\"\n[sub]\nendpoint = \"toml-endpoint\"\n",
			loader:   "toml",
		},
		{
			name:     "json",
			file:     "config.json",
			contents: `{"name":"from-json","sub":{"endpoint":"json-endpoint"}}`,
			loader:   "json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fpath := filepath.Join(dir, tc.file)
			if err := os.WriteFile(fpath, []byte(tc.contents), 0o600); err != nil {
				t.Fatalf("writing temp config: %v", err)
			}

			c := &cfg{}
			err := New().DisableStdFlags().With(tc.loader).SetConfigPath(fpath).Load(c)
			if err != nil {
				t.Fatalf("unexpected error loading %q: %v", tc.file, err)
			}

			if c.Name == "" {
				t.Errorf("expected Name to be populated from %q, got empty", tc.file)
			}
			if c.Sub.Endpoint == "" {
				t.Errorf("expected Sub.Endpoint to be populated from %q, got empty", tc.file)
			}
		})
	}
}
