package config

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pcelvng/go-config/encoding/env"
	"github.com/pcelvng/go-config/encoding/file"

	flg "github.com/pcelvng/go-config/encoding/flag"
	"github.com/pkg/errors"
)

type Hide struct {
	HField string `hide`
}

// goConfig should probably be private so it can only be set through the new method.
// this does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
	config interface{}

	envEnabled  bool
	fileEnabled bool
	flagEnabled bool

	// flags
	showVersion *bool
	showConfig  *bool
	version     string
	description string
	genConfig   *string
	configPath  *string

	flags *flg.Flags
}

// New verifies that c is a valid - must be a struct pointer (maybe validation should happen in parse)
// it goes through the struct and sets up corresponding flags to be used during parsing
func New(c interface{}) *goConfig {
	return &goConfig{
		envEnabled:  true,
		fileEnabled: true,
		flagEnabled: true,
		config:      c,
	}
}

// Parse and set the configs in the following priority from lowest to highest
// 1. environment variables
// 2. flags (exception of config and version flag which are processed first)
// 3. files (toml, yaml, json)
func (g *goConfig) Parse() error {
	g.showConfig = flag.Bool("show", false, "print out the value of the config")
	f, err := flg.New(g.config)
	if err != nil {
		return errors.Wrap(err, "flag setup")
	}
	g.flags = f
	if g.fileEnabled {
		g.genConfig = flag.String("g", "", "generate config file (toml,json,yaml)")
		flag.StringVar(g.genConfig, "gen", "", "")
		g.configPath = flag.String("c", "", "path for config file")
		flag.StringVar(g.configPath, "config", "", "")
	}
	// prepend description to help usage
	if g.description != "" {
		f := g.flags
		f.Usage = func() {
			fmt.Fprint(os.Stderr, g.description, "\n")
			w := new(bytes.Buffer)
			f.SetOutput(w)
			f.PrintDefaults()

			//remove redundant outputs
			output := w.String()
			output = strings.Replace(output, "-g ", "-g,-gen ", 1)
			output = strings.Replace(output, "-c ", "-c,-config ", 1)
			output = strings.Replace(output, "-v\t", "-v,-version\n\t", 1)
			skipLine := false
			for _, s := range strings.Split(output, "\n") {
				if skipLine {
					skipLine = false
					continue
				}
				if len(strings.TrimSpace(s)) == 0 {
					continue
				}
				if strings.Contains(s, " -gen") || strings.Contains(s, " -config") {
					skipLine = true
					continue
				}
				if strings.Contains(s, " -version") {
					continue
				}
				fmt.Fprint(os.Stderr, s, "\n")
			}
		}
	}

	g.flags.Parse()

	if *g.showVersion {
		fmt.Println(g.version)
		os.Exit(0)
	}

	if *g.genConfig != "" {
		err := file.Encode(os.Stdout, g.config, *g.genConfig)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	// load in lowest priority order: env -> file -> flag
	if g.envEnabled {
		if err := env.New().Unmarshal(g.config); err != nil {
			return err
		}
	}
	if g.fileEnabled && *g.configPath != "" {
		if err := file.Load(*g.configPath, g.config); err != nil {
			return err
		}
	}
	if g.flagEnabled {
		if err := g.flags.Unmarshal(g.config); err != nil {
			return err
		}
	}

	if *g.showConfig {
		spew.Dump(g.config)
		os.Exit(0)
	}
	return nil
}

// ParseFile loads config date from a file (yaml, toml, json)
// into the struct i.
// this would be used if we only want to parse a file and don't
// want to use any other features. This is more or less what multi-config does
func ParseFile(f string, i interface{}) error {
	return file.Load(f, i)
}

// ParseEnv is similar to ParseFile, but only checks env vars
func ParseEnv(i interface{}) error {
	return env.New().Unmarshal(i)
}

// ParseFlag is similar to ParseFile, but only checks flags
func ParseFlag(i interface{}) error {
	f, err := flg.New(i)
	if err != nil {
		return err
	}
	return f.Parse()
}

// Version string that describes the app.
// this enables the -v (version) flag
func (g *goConfig) Version(s string) *goConfig {
	g.showVersion = flag.Bool("v", false, "show app version")
	flag.BoolVar(g.showVersion, "version", false, "")
	g.version = s
	return g
}

// Description for the app, this message is prepended to the help flag
func (g *goConfig) Description(s string) *goConfig {
	g.description = s
	return g
}

// DisableEnv tells goConfig not to use environment variables
func (g *goConfig) DisableEnv() *goConfig {
	g.envEnabled = false
	return g
}

// DisableFile removes the c (config) flag used for defining a config file
func (g *goConfig) DisableFile() *goConfig {
	g.fileEnabled = false
	return g
}

// DisableFlag prevents setting variables from flags.
// Non variable flags should still work [c (config), v (version), g (gen)]
func (g *goConfig) DisableFlag() *goConfig {
	g.flagEnabled = false
	return g
}
