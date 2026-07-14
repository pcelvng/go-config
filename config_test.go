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

// EmbeddedDB is used to exercise embedded (anonymous) struct promotion. It must
// be exported so its fields are settable when embedded.
type EmbeddedDB struct {
	Host string `yaml:"host" toml:"host" json:"host"`
	Port int    `yaml:"port" toml:"port" json:"port"`
}

// embeddedConfig embeds EmbeddedDB. The yaml:",inline" tag is required for the
// yaml loader to promote the embedded fields (gopkg.in/yaml.v2 does not inline
// anonymous structs by default).
type embeddedConfig struct {
	EmbeddedDB `yaml:",inline"`
	Name       string `yaml:"name" toml:"name" json:"name"`
}

// TestEmbeddedStructPromotionEnv verifies that fields of an embedded
// (anonymous) struct are promoted in the env loader: the embedded type name is
// NOT used as a prefix.
func TestEmbeddedStructPromotionEnv(t *testing.T) {
	os.Clearenv()
	t.Setenv("HOST", "promoted-host")
	t.Setenv("PORT", "5432")
	t.Setenv("NAME", "myapp")

	c := &embeddedConfig{}
	if err := New().DisableStdFlags().With("env").Load(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Host != "promoted-host" {
		t.Errorf("expected Host=%q from HOST env var, got %q", "promoted-host", c.Host)
	}
	if c.Port != 5432 {
		t.Errorf("expected Port=5432 from PORT env var, got %d", c.Port)
	}
	if c.Name != "myapp" {
		t.Errorf("expected Name=%q, got %q", "myapp", c.Name)
	}
}

// TestEmbeddedStructPromotionFiles verifies that embedded struct fields are
// promoted (flattened) consistently across the yaml, toml, and json loaders.
func TestEmbeddedStructPromotionFiles(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		contents string
		loader   string
	}{
		{
			name:     "yaml",
			file:     "config.yaml",
			contents: "host: yaml-host\nport: 1\nname: yaml-name\n",
			loader:   "yaml",
		},
		{
			name:     "toml",
			file:     "config.toml",
			contents: "host = \"toml-host\"\nport = 2\nname = \"toml-name\"\n",
			loader:   "toml",
		},
		{
			name:     "json",
			file:     "config.json",
			contents: `{"host":"json-host","port":3,"name":"json-name"}`,
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

			c := &embeddedConfig{}
			if err := New().DisableStdFlags().With(tc.loader).SetConfigPath(fpath).Load(c); err != nil {
				t.Fatalf("unexpected error loading %q: %v", tc.file, err)
			}

			if c.Host == "" {
				t.Errorf("expected promoted Host to be populated from %q, got empty", tc.file)
			}
			if c.Port == 0 {
				t.Errorf("expected promoted Port to be populated from %q, got 0", tc.file)
			}
			if c.Name == "" {
				t.Errorf("expected Name to be populated from %q, got empty", tc.file)
			}
		})
	}
}
