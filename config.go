package config

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"

	"github.com/hydronica/go-config/internal/encode/env"
	"github.com/hydronica/go-config/internal/encode/file"
	flg "github.com/hydronica/go-config/internal/encode/flag"
)

// goConfig should probably be private so it can only be set through the new method.
// This does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
	config interface{}

	options Options

	// special flags
	showVersion *bool
	appName     string // self proclaimed app name.
	showConfig  *bool
	version     string
	description string
	genConfig   *string
	configPath  *string

	defaultConfigPath string

	flags *flg.Flags
}

// Validator can be used as a way to validate the state of a config
// after it has been loaded.
type Validator interface {
	Validate() error
}

// New verifies that c is a valid - must be a struct pointer (maybe validation should happen in parse)
// it goes through the struct and sets up corresponding flags to be used during parsing
func New(c interface{}) *goConfig {
	return &goConfig{
		options: defaultOpts,

		config: c,
	}
}

type Options uint64

const (
	OptEnv Options = 1 << iota
	OptEnvFile  // load .env from working directory
	OptToml
	OptYaml
	OptJson
	OptFlag
	OptGenConf  // -g to generate config files
	OptShow     // -show to show the set config values
)
const OptFiles = OptToml | OptYaml | OptJson
const defaultOpts = OptEnv | OptFiles | OptFlag | OptShow | OptGenConf | OptEnvFile

// Disable Options. By Default all Options are enabled.
// OptEnv: ignore environment variables
// OptEnvFile: ignore .env file from working directory
// OptFiles: ignore supported config files
// OptYaml: ignore yaml config files
// OptJson: ignore json config files
// OptToml: ignore toml config files
// OptFlag: ignore flag config files
// OptGenConf: remove flag option to generate config files
// OptShow: remove flag option to print of config values
func (g *goConfig) Disable(opts Options) *goConfig {
	g.options &^= opts
	return g
}

// SetOptions overrides the default Options to just set the desired options
// Example SetOptions(OptFiles | OptGenConf | OptShow)
func (g *goConfig) SetOptions(opts Options) *goConfig {
	g.options = opts
	return g
}

// isEnabled is a helper method to check if the proper bits are set
func (o Options) isEnabled(v Options) bool {
	return o&v == v
}

// LoadOrDie is the same as Load except it exits if there is an error.
func (g *goConfig) LoadOrDie() {
	err := g.Load()
	if err != nil {
		log.Printf("err: %v", err.Error())
		os.Exit(0)
	}
}

// Load the configs in the following priority from most passive to most active:
//
// 1. Defaults
// 2. Environment variables and the working directory ".env" file (read line by line;
//    for each struct field, a non-empty value on that key in ".env" overrides os.Getenv)
//    mapped into the struct
// 3. File (toml, yaml, json)
// 4. Flags (exception of config and version flag which are processed first)
//
// After the configs are loaded validate the result if config is a Validator.
//
// Defaults are loaded first (on struct initialization by the user) then env variables supplant
// defaults and then file config values are loaded which supplant env or default values and finally
// flag values trump everything else.
//
// Before loading values, special flags (ie -help, -show, -config, -gen) are processed.
func (g *goConfig) Load() error {
	if g.options.isEnabled(OptShow) {
		g.showConfig = flag.Bool("show", false, "print out the value of the config")
	}

	var f *flg.Flags
	var err error
	if g.options.isEnabled(OptFlag) {
		f, err = flg.New(g.config)
	} else { // don't add flags when disabled
		f, err = flg.New(nil)
	}

	if err != nil {
		return fmt.Errorf("flag setup %w", err)
	}
	g.flags = f

	if g.options.isEnabled(OptFiles) {
		if g.options.isEnabled(OptGenConf) {
			g.genConfig = flag.String("g", "", "generate config file (toml,json,yaml,env)")
			flag.StringVar(g.genConfig, "gen", "", "")
		}
		g.configPath = flag.String("c", g.defaultConfigPath, "path for config file")
		flag.StringVar(g.configPath, "config", g.defaultConfigPath, "")
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
		return fmt.Errorf("flag parse %w", err)
	}

	if g.showVersion != nil && *g.showVersion {
		fmt.Println(g.version)
		os.Exit(0)
	}

	// load in lowest priority order: env -> .env file -> config file -> flag
	if g.options.isEnabled(OptEnv) {
		if err := env.New().Unmarshal(g.config); err != nil {
			return err
		}
	}

	if g.options.isEnabled(OptEnvFile) {
		if _, err := os.Stat(".env"); err == nil {
			log.Println("loading .env file from working directory")
			if err := env.LoadEnvFile(".env", g.config); err != nil {
				return err
			}
		}
	}

	if g.options.isEnabled(OptFiles) && *g.configPath != "" {
		if err := file.Load(*g.configPath, g.config); err != nil {
			return err
		}
	}
	if g.options.isEnabled(OptFlag) {
		if err := g.flags.Unmarshal(g.config); err != nil {
			return err
		}
	}

	if g.options.isEnabled(OptGenConf) && *g.genConfig != "" {
		err := file.Encode(os.Stdout, g.config, *g.genConfig)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if g.options.isEnabled(OptShow) && *g.showConfig {
		spew.Dump(g.config)
		os.Exit(0)
	}

	// validate if struct implements validator interface
	if val, ok := g.config.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// LoadFile loads configuration values from a file (yaml, toml, json)
// into the struct configuration c.
//
// This would be used if we only want to parse a file and don't
// want to use any other features. This is more or less what multi-config does.
func LoadFile(f string, c interface{}) error {
	return file.Load(f, c)
}

// LoadEnv maps only the process environment variables into c.
// Use LoadFile with a .env path, or use Load() which auto-loads both env and .env by default.
func LoadEnv(c interface{}) error {
	return env.New().Unmarshal(c)
}

// LoadFlag is similar to LoadFile, but only checks flags.
func LoadFlag(c interface{}) error {
	f, err := flg.New(c)
	if err != nil {
		return err
	}
	if err := f.Parse(); err != nil {
		return err
	}
	defaultCfg.flags = f
	return f.Unmarshal(c)
}

// Args returns the non-flag command-line arguments remaining after Load,
// LoadOrDie, or LoadFlag. Positional args may appear before, after, or between
// flags. Returns nil if flags have not been parsed yet.
func Args() []string {
	return defaultCfg.Args()
}

// Args returns the non-flag command-line arguments remaining after Load or
// LoadOrDie on this config instance. Returns nil if flags have not been parsed.
func (g *goConfig) Args() []string {
	if g.flags == nil {
		return nil
	}
	return g.flags.Args()
}

// Version string that describes the app which enables the -v (version) flag.
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

// ConfigPath sets the default config file path used when -c or -config is not
// provided. The path appears as the default in help output and is loaded
// automatically. An explicit -c or -config flag overrides this value.
func (g *goConfig) ConfigPath(path string) *goConfig {
	g.defaultConfigPath = path
	return g
}

// Deprecated: Use Disable(OptEnv) instead
// DisableEnv tells goConfig not to use environment variables
func (g *goConfig) DisableEnv() *goConfig {
	g.Disable(OptEnv)
	return g
}

// Deprecated: Use Disable(OptFiles) instead
// DisableFiles removes the c (config) flag used for defining a config file
// from the help menu and skips file parsing when reading in
// values.
func (g *goConfig) DisableFiles() *goConfig {
	g.Disable(OptFiles)
	return g
}

// DisableTOML will prevent files with a '.toml' extension
// from being parsed and will remove 'toml' type options
// from the help menu.
func (g *goConfig) DisableTOML() *goConfig {
	g.Disable(OptToml)
	return g
}

// Deprecated: Use Disable(OptYaml) instead
// DisableYAML will prevent files with a '.yaml', '.yml' extension
// from being parsed and will remove 'yaml', 'yml' type options
// from the help menu.
func (g *goConfig) DisableYAML() *goConfig {
	g.Disable(OptYaml)
	return g
}

// Deprecated: Use Disable(OptJson) instead
// DisableJSON will prevent files with a '.json' extension
// from being parsed and will remove 'json' type options
// from the help menu.
func (g *goConfig) DisableJSON() *goConfig {
	g.Disable(OptJson)
	return g
}

// Deprecated: Use Disable(OptFlag) instead
func (g *goConfig) DisableFlags() *goConfig {
	g.Disable(OptFlag)
	return g
}

var defaultCfg = New(nil)

func Load(c interface{}) error {
	defaultCfg.config = c
	return defaultCfg.Load()
}

func LoadOrDie(c interface{}) {
	defaultCfg.config = c
	defaultCfg.LoadOrDie()
}
