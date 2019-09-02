package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/pcelvng/go-config/encode/env"
	"github.com/pcelvng/go-config/encode/file"

	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/copier"
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

	defaultCfg = New()
)

func Load(c interface{}) error {
	return defaultCfg.Load(c)
}

func LoadOrDie(c interface{}) {
	defaultCfg.LoadOrDie(c)
}

// With is a package wrapper around goConfig.With().
func With(w ...string) *goConfig {
	return defaultCfg.With(w...)
}

// Version is a package wrapper around goConfig.Version().
func Version(s string) *goConfig {
	return defaultCfg.Version(s)
}

// AddHelp is a package wrapper around goConfig.AddHelp().
func AddHelp(hlp string) *goConfig {
	return defaultCfg.AddHelp(hlp)
}

// New creates a new config.
func New() *goConfig {
	return &goConfig{
		with: withList,
	}
}

// goConfig should probably be private so it can only be set through the new method.
// This does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
	with    []string
	version string // Self proclaimed app version.
	helpTxt string // App custom help text, pre-pended to generated help text.
}

type standardFlags struct {
	ConfigPath  string `flag:"config,c" env:"-" toml:"-"`
	GenConfig   string `flag:"gen,g" env:"-" toml:"-"`
	ShowValues  bool   `flag:"show" env:"-" toml:"-" help:"Show loaded config values and exit."`
	ShowVersion bool   `flag:"version,v" env:"-" toml:"-" help:"Show application version and exit."`
}

var (
	cfgPathHelp   = "Config file path. Extension must be %s."
	genConfigHelp = "Generate config template (%s)."
)

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
func (g *goConfig) Load(appCfg interface{}) error {
	// Verify that appCfg is struct pointer. Should not be nil.
	appCfgV := reflect.ValueOf(appCfg)
	if appCfgV.Kind() != reflect.Ptr || appCfgV.IsNil() {
		return fmt.Errorf("'%v' must be a non-nil pointer", reflect.TypeOf(appCfg))
	} else if pv := reflect.Indirect(appCfgV); pv.Kind() != reflect.Struct { // Must be pointing to a struct.
		return fmt.Errorf("'%v' must be a non-nil pointer struct", reflect.TypeOf(appCfg))
	}

	// new deep copy of appCfg (to preserve defaults).
	appCfgCopy := reflect.New(reflect.TypeOf(appCfg).Elem()).Interface()
	err := copier.Copy(appCfgCopy, appCfg)
	if err != nil {
		return err
	}

	// Process special flags.
	stdFlgs := &standardFlags{}
	flgDecoder := flg.NewDecoder(g.helpTxt)

	// Customize special flags help screen and options.
	if len(g.fileExts()) > 0 {
		msg := fmt.Sprintf(cfgPathHelp, strings.Join(g.fileExts(), "|"))
		flgDecoder.SetHelp("ConfigPath", msg)
	} else {
		// no file exts - ignore the help message, not applicable.
		flgDecoder.IgnoreField("ConfigPath")
	}

	if len(g.genTypes()) > 0 {
		msg := fmt.Sprintf(genConfigHelp, strings.Join(g.genTypes(), "|"))
		flgDecoder.SetHelp("GenConfig", msg)
	} else {
		// no generate-able types - ignore the help message, not applicable.
		flgDecoder.IgnoreField("GenConfig")
	}

	if g.version == "" {
		flgDecoder.IgnoreField("ShowVersion")
	}

	if itemIn("flag", g.with) == "" {
		// flag excluded, don't render app config.
		err = flgDecoder.Unmarshal(stdFlgs)
	} else {
		err = flgDecoder.Unmarshal(stdFlgs, appCfgCopy)
	}
	if err != nil {
		return err
	}

	// ShowVersion
	if stdFlgs.ShowVersion {
		fmt.Fprintln(os.Stderr, g.version)
		os.Exit(0)
	}

	// Generate config template.
	if stdFlgs.GenConfig != "" {
		err = file.Encode(os.Stdout, appCfg, stdFlgs.GenConfig)
		if err != nil {
			return err
		}
		os.Exit(0)
	}

	// Read in all values.
	err = g.loadAll(stdFlgs.ConfigPath, appCfg)
	if err != nil {
		return err
	}

	// ShowValues
	if stdFlgs.ShowValues {
		// TODO: show values in the README format.
		spew.Dump(appCfg)
		spew.Dump(appCfgCopy)
		os.Exit(0)
	}

	// Validate if struct implements validator interface.
	if val, ok := appCfg.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// loadAll iterates through the "with" list and loads
// the config values into appCfg.
func (g *goConfig) loadAll(pth string, appCfg interface{}) error {
	doneFile := false
	for _, w := range g.with {
		switch w {
		case "env":
			err := env.NewDecoder().Unmarshal(appCfg)
			if err != nil {
				return err
			}
		case "toml", "yaml", "json":
			if !doneFile && pth != "" {
				err := file.Load(pth, appCfg)
				if err != nil {
					return err
				}

				doneFile = true
			}
		case "flag":
			err := flg.NewDecoder("").Unmarshal(appCfg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// AddHelp allows the user to provide a block of text that is pre-pended to the
// generated application help screen.
func (g *goConfig) AddHelp(hlp string) *goConfig {
	g.helpTxt = hlp
	return g
}

// LoadOrDie calls Load and prints an error message and exits if there is an error.
func (g *goConfig) LoadOrDie(appCfg interface{}) {
	err := g.Load(appCfg)
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
	g.with = newWith

	return g
}

// Version sets the application version.
func (g *goConfig) Version(s string) *goConfig {
	g.version = s
	return g
}

// fileExts returns a slice of all the included file extensions.
func (g *goConfig) fileExts() []string {
	exts := make([]string, 0)

	i := itemIn("toml", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	i = itemIn("yaml", g.with)
	if i != "" {
		exts = append(exts, i, "yml")
	}

	i = itemIn("json", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	return exts
}

// genTypes returns a slice of all the included generate-able extensions.
func (g *goConfig) genTypes() []string {
	exts := make([]string, 0)

	i := itemIn("env", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	i = itemIn("toml", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	i = itemIn("yaml", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	i = itemIn("json", g.with)
	if i != "" {
		exts = append(exts, i)
	}

	return exts
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
