package config

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"

	"github.com/pcelvng/go-config/encode/env"
	"github.com/pcelvng/go-config/encode/file"

	"github.com/pkg/errors"

	flg "github.com/pcelvng/go-config/encode/flag"
)

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

// Validator can be used as a way to validate the state of a config
// after it has been loaded
type Validator interface {
	Validate() error
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

// LoadOrDie is the same as Load except it exit if there is an error
func (g *goConfig) LoadOrDie() {
	err := g.Load()
	if err != nil {
		log.Fatal(err)
	}
}

// Load the configs in the following priority from lowest to highest
// 1. environment variables
// 2. flags (exception of config and version flag which are processed first)
// 3. files (toml, yaml, json)
// After the configs are loaded validate the result if config implements validate interface
func (g *goConfig) Load() error {
	g.showConfig = flag.Bool("show", false, "print out the value of the config")
	f, err := flg.New(g.config)
	if err != nil {
		return errors.Wrap(err, "flag setup")
	}
	g.flags = f
	if g.fileEnabled {
		g.genConfig = flag.String("g", "", "generate config file (toml,json,yaml,env)")
		flag.StringVar(g.genConfig, "gen", "", "")
		g.configPath = flag.String("c", "", "path for config file")
		flag.StringVar(g.configPath, "config", "", "")
	}

	f.Usage = func() {
		// prepend description to help usage
		if g.description != "" {
			fmt.Fprint(os.Stderr, g.description, "\n")
		}
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

	if err := g.flags.Parse(); err != nil {
		return errors.Wrap(err, "flag parse")
	}

	if g.showVersion != nil && *g.showVersion {
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

	// validate if struct implements validator interface
	if val, ok := g.config.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// VarComment will add a variable comment/description to generated help messages
func (g *goConfig) VarComment(field, comment string) *goConfig {
	// todo: add map[field]comment to go through to add description
	// how do we handle embedded structs that have the same Variable name as the parent
	/*type dchild struct {
		Name string
	}
	type dummy struct {
		Name string
		Child dchild
	}*/

	return g
}

// LoadFile loads config date from a file (yaml, toml, json)
// into the struct i.
// this would be used if we only want to parse a file and don't
// want to use any other features. This is more or less what multi-config does
func LoadFile(f string, i interface{}) error {
	return file.Load(f, i)
}

// LoadEnv is similar to LoadFile, but only checks env vars
func LoadEnv(i interface{}) error {
	return env.New().Unmarshal(i)
}

// LoadFlag is similar to LoadFile, but only checks flags
func LoadFlag(i interface{}) error {
	f, err := flg.New(i)
	if err != nil {
		return err
	}
	if err := f.Parse(); err != nil {
		return err
	}
	return f.Unmarshal(i)
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
