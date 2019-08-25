package config

import (
	"fmt"
	"os"

	flg "github.com/pcelvng/go-config/encode/flag"
)

var (
	withList = []string{
		// Listed in order they are loaded.
		// Warning: modifying this order will change load order.
		"env", // env trumps defaults.
		"toml",
		"yaml",
		"json",
		"flag", // flag trumps all.
	}

	fileExts = []string{
		"toml",
		"yaml",
		"yml",
		"json",
	}

	defaultCfg = New(nil)
)

func Load(c interface{}) error {
	defaultCfg.cfgRef = c
	return defaultCfg.Load()
}

func LoadOrDie(c interface{}) {
	defaultCfg.cfgRef = c
	defaultCfg.LoadOrDie()
}

// With is a package wrapper around goConfig.With().
func With(w ...string) *goConfig {
	return defaultCfg.With(w...)
}

// Version is a package wrapper around goConfig.Version().
func Version(s string) *goConfig {
	return defaultCfg.Version(s)
}

// New creates a new config.
func New(c interface{}) *goConfig {
	return &goConfig{
		cfgRef: c,
		with:   withList,
	}
}

// goConfig should probably be private so it can only be set through the new method.
// This does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
	cfgRef  interface{}
	with    []string
	version string // Self proclaimed app version.
	helpTxt string // App custom help text, pre-pended to generated help text.

	// standard flags
	sFlgs standardFlags

	//flags *flg.Flags
}

type standardFlags struct {
	ConfigPath  string `flag:"config,c" env:"-" toml:"-" help:"The config file path (if using one). File extension must be one of "toml", "yaml", "yml", "json".`
	GenConfig   string `flag:"gen,g" env:"-" toml:"-" help:"Generate config template. One of "toml", "yaml", "json", "env"."`
	ShowValues  bool   `flag:"show" env:"-" toml:"-" help:"Show loaded config values and exit."`
	ShowVersion bool   `flag:"version,v" env:"-" toml:"-" help:"Show application version and exit."`
}

// Load the configs in the following priority from most passive to most active:
//
// 1. Defaults
// 2. Environment variables
// 3. File (toml, yaml, json)
// 4. Flags (exception of config and version flag which are processed first)
//
// After the configs are loaded validate the result if config is a Validator.
//
// Special flags are processed before loading config values.
func (g *goConfig) Load() error {
	// Process general/special flags.
	stdFlgs := &standardFlags{}
	flg.New(stdFlgs)

	// Validate if struct implements validator interface.
	if val, ok := g.cfgRef.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// LoadOrDie calls Load and prints an error message and exits if there is an error.
func (g *goConfig) LoadOrDie() {
	err := g.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err.Error())
		os.Exit(1)
	}
}

// With sets which configuration modes are enabled.
// "With" item order will not change load order precedence.
//
// Values can be any of: "env", "toml", "yaml", "json", "flag".
func (g *goConfig) With(w ...string) *goConfig {
	newWith := make([]string, 0)

	for _, j := range withList {
		if newItem := itemIn(j, w); newItem != "" {
			newWith = append(newWith, newItem)
		}
	}

	return g
}

// Version sets the application version.
func (g *goConfig) Version(s string) *goConfig {
	g.version = s
	return g
}

// itemIn will return 'i' when 'i' exists in 'all'.
func itemIn(i string, all []string) string {
	for _, j := range all {
		if i == j {
			return i
		}
	}

	return ""
}

// Validator can be implemented by the user provided config struct.
// Validate() is called after loading and running tag level validation.
type Validator interface {
	Validate() error
}
