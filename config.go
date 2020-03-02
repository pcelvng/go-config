package config

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pcelvng/go-config/internal/render"
	"github.com/pcelvng/go-config/load"
	"github.com/pcelvng/go-config/load/env"
	"github.com/pcelvng/go-config/load/file"
	flg "github.com/pcelvng/go-config/load/flag"
	"github.com/pcelvng/go-config/util"
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
	// validExts contains a list of all supported/valid file extensions.
	validExts = []string{
		"env",
		"sh", // same output as "env" but with different file extension.
		"toml",
		"yaml",
		"yml",
		"json",
	}

	reqTag      = "req"      // Field tag indicating a required field for basic validation.
	validateTag = "validate" // See https://godoc.org/gopkg.in/go-playground/validator.v9
	configTag   = "config"   // Globally applied config arguments. Specific config types may override contents of this value.

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

// RegisterLoader is a package wrapper around goConfig.CustomLoader().
func RegisterLoader(name string, loader load.LoadUnloader) *goConfig {
	return defaultCfg.RegisterLoader(name, loader)
}

// AddHelpTxt is a package wrapper around goConfig.AddHelp().
func AddHelpTxt(hlpPreTxt, hlpPostTxt string) *goConfig {
	return defaultCfg.AddHelpTxt(hlpPreTxt, hlpPostTxt)
}

// Version is a package wrapper around goConfig.Version().
func Version(s string) *goConfig {
	return defaultCfg.Version(s)
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
	// with is a list of loaders by name in the order they will be loaded.
	with        []string
	loaders     map[string]load.Loader   // map of initialized loaders.
	unloaders   map[string]load.Unloader // map of initialized unloaders.
	version     string
	helpPreTxt  string // App custom help text, pre-pended to generated help menu.
	helpPostTxt string // App custom help text appended to the generated help menu.
	showPreTxt  string // Custom show text, pre-pended to generated show output.
	showPostTxt string // Custom show text appended to the generated show output.
	showNameFmt string // Field name format for the standard "show" renderer.
	renderer    *render.Renderer
}

type standardFlags struct {
	ConfigPath  string `flag:"config,c" env:"-" toml:"-"`
	GenConfig   string `flag:"gen,g" env:"-" toml:"-"` // TODO: value can be path or extension. 'env' can also be 'sh'. 'env' or 'sh' is also attempts to make executable.
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
func (g *goConfig) Load(appCfgs ...interface{}) error {
	var err error
	if g == nil {
		panic("goConfig not initialized")
	}

	if len(appCfgs) == 0 {
		return fmt.Errorf("nothing to load into")
	}

	// Verify all appCfgs are struct pointers.
	for _, appCfg := range appCfgs {
		if _, err := util.IsStructPointer(appCfg); err != nil {
			return err
		}
	}

	// Initialize renderer.
	//
	// Default values are recorded with the renderer on initialization.
	g.renderer, err = render.New(render.Options{
		Preamble:        g.showPreTxt,
		Conclusion:      g.showPostTxt,
		FieldNameFormat: g.showNameFmt,
	}, appCfgs...)
	if err != nil {
		return err
	}

	// Process special flags.
	stdFlgs := &standardFlags{}
	flgLdr := flg.NewLoader(flg.Options{
		HlpPreText:  g.helpPreTxt,
		HlpPostText: g.helpPostTxt,
	})

	// Customize special flags help screen and options.
	if len(g.fileExts()) > 0 {
		msg := fmt.Sprintf(cfgPathHelp, strings.Join(g.fileExts(), "|"))
		flgLdr.SetHelp("config", msg)
	} else {
		// no file exts - ignore the help message, not applicable.
		flgLdr.IgnoreField("config")
	}

	if len(g.genTypes()) > 0 {
		msg := fmt.Sprintf(genConfigHelp, strings.Join(g.genTypes(), "|"))
		flgLdr.SetHelp("gen", msg)
	} else {
		// no generate-able types - ignore the help message, not applicable.
		flgLdr.IgnoreField("gen")
	}

	if g.version == "" {
		flgLdr.IgnoreField("version")
	}

	cfgs := make([]interface{}, 0)
	cfgs = append(cfgs, stdFlgs)
	cfgs = append(cfgs, appCfgs...)
	if itemIn("flag", g.with) == "" {
		// flag excluded, don't render app config.
		err = flgLdr.Load(stdFlgs)
	} else {
		err = flgLdr.Load(cfgs...)
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
		pth, ext := parseGenPath(stdFlgs.GenConfig)
		u := file.NewUnloader(ext)
		b, err := u.Unload(appCfgs...)
		if err != nil {
			return err
		}

		if pth != "" { // Write to file.
			f, err := os.Create(pth)
			if err != nil {
				return err
			}
			defer f.Close()

			// TODO: consider making 'env' and 'sh' files executable.
			_, err = f.Write(b)
			if err != nil {
				return err
			}
		} else {
			os.Stdout.Write(b)
		}

		os.Exit(0)
	}

	// Read in all values.
	err = g.loadAll(stdFlgs.ConfigPath, cfgs...)
	if err != nil {
		return err
	}

	// ShowValues
	if stdFlgs.ShowValues {
		err = g.ShowValues()
		if err != nil {
			return err
		}
		os.Exit(0)
	}

	// Validate if struct implements validator interface.
	// TODO: implement full validate tag support.
	// TODO: validate on the 'req:"true"' struct tag.
	for _, appCfg := range appCfgs {
		if val, ok := appCfg.(Validator); ok {
			return val.Validate()
		}
	}

	return nil
}

func Show() error {
	return defaultCfg.ShowValues()
}

func ShowOrDie() {
	err := Show()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// parseGenPath parses the provide "GenConfig" path returning the
// path value if it's a file path and the file extension. If no
// extension is discerned then ext is empty.
//
// The value can either be a standalone file extension such as "toml"
// or it can be a full file path (with extension) to write the
// generated config template.
func parseGenPath(pth string) (fpth, ext string) {
	ext = path.Ext(pth)
	if ext == "" {
		// maybe pth is just an extension.
		if isValidExt(pth) {
			return "", pth
		}
	}

	return pth, ext

}

// isValidExt determines if the file extension output type is supported.
func isValidExt(ext string) bool {
	for _, v := range validExts {
		if v == ext {
			return true
		}
	}

	return false
}

// ShowValues writes the values to os.Stderr.
func (g *goConfig) ShowValues() error {
	return g.FShowValues(os.Stderr)
}

func (g *goConfig) FShowValues(w io.Writer) error {
	b := g.renderer.Render()
	_, err := fmt.Fprintln(w, string(b))

	return err
}

// AddShowTxt registers text blocks to pre-pend and append to the rendered "show" body.
func (g *goConfig) AddShowTxt(pre, post string) *goConfig {
	g.showPreTxt = pre
	g.showPostTxt = post
	return g
}

// loadAll iterates through the "with" list and loads
// the config values into cfgs[1:] where cfgs[1:] are all the
// appCfgs and cfgs[0] is the standard config for standard flags.
//
// Expects "cfgs" to contain the standard config first followed by application
// configs.
func (g *goConfig) loadAll(pth string, cfgs ...interface{}) error {
	doneFile := false
	for _, w := range g.with {
		switch w {
		case "env":
			err := env.Load(cfgs[1:]...)
			if err != nil {
				return err
			}
		case "toml", "yaml", "json":
			if !doneFile && pth != "" {
				fl := file.NewLoader(pth)
				err := fl.Load(cfgs[1:]...)
				if err != nil {
					return err
				}

				doneFile = true
			}
		case "flag":
			err := flg.NewLoader(flg.Options{}).Load(cfgs...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// AddHelp allows the user to provide supplemental pre and post help blocks
// that are prepended and appended to the generated help menu.
func (g *goConfig) AddHelpTxt(pre, post string) *goConfig {
	g.helpPreTxt = pre
	g.helpPostTxt = post
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

// RegisterLoader registers a custom LoadUnloader.
func (g *goConfig) RegisterLoader(name string, loader load.LoadUnloader) *goConfig {
	if name == "" {
		panic("loader name required")
	}

	if loader == nil {
		panic("loader required")
	}

	return nil
}

// Version sets the application version. If a version is provided then
// the user can specify the --version flag to show the version. Otherwise the version flag
// is not seen on the help screen.
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
