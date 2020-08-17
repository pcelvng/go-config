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
	flg "github.com/pcelvng/go-config/load/flag"
	"github.com/pcelvng/go-config/load/toml"
	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

var (
	withList = []string{
		// Listed in the default load order.
		"env", // env trumps defaults.
		"toml",
		"yaml",
		"json",
		// "..." <- custom names are loaded here.
		"flag", // flag trumps all.
	}

	// fileExts is a default map of loader names to associated file extensions.
	// An empty associated slice indicates the loader does not load from
	// file.
	fileExts = map[string][]string{
		"env":  {"env", "sh"},
		"toml": {"toml"},
		"yaml": {"yaml", "yml"},
		"json": {"json"},
		"flag": {},
	}

	// TODO: implement validation support for req struct tag.
	//reqTag = "req" // Field tag indicating a required field for basic validation.

	// TODO: built in support for validate struct tag.
	//validateTag = "validate" // See https://godoc.org/gopkg.in/go-playground/validator.v9
	configTag = "config" // Globally applied config arguments. Specific config types may override contents of this value.

	defaultCfg = New()
)

// Load is a package wrapper around goConfig.Load().
func Load(c interface{}) error {
	return defaultCfg.Load(c)
}

// LoadOrDie is a package wrapper around goConfig.LoadOrDie().
func LoadOrDie(c interface{}) {
	defaultCfg.LoadOrDie(c)
}

// With is a package wrapper around goConfig.With().
func With(w ...string) *goConfig {
	return defaultCfg.With(w...)
}

// RegisterLoader is a package wrapper around goConfig.RegisterLoader().
func RegisterLoader(name, fileExt string, loader load.LoadUnloader) *goConfig {
	return defaultCfg.RegisterLoader(name, fileExt, loader)
}

// AddHelpTxt is a package wrapper around goConfig.AddHelpTxt().
func AddHelpTxt(hlpPreTxt, hlpPostTxt string) *goConfig {
	return defaultCfg.AddHelpTxt(hlpPreTxt, hlpPostTxt)
}

// Version is a package wrapper around goConfig.Version().
func Version(s string) *goConfig {
	return defaultCfg.Version(s)
}

// New creates a new config.
func New() *goConfig {
	cfg := &goConfig{
		fullWith:  withList,
		with:      withList,
		fExts:     fileExts,
		loaders:   make(map[string]load.Loader),
		unloaders: make(map[string]load.Unloader),
	}

	return cfg
}

func registerDefaultLoaders(wList []string) map[string]load.Loader {
	loaders := make(map[string]load.Loader)

	for _, l := range wList {
		switch l {
		// flag is special and cannot be overwritten.
		//case "flag":
		//	loaders[l] = flag.Load
		case "toml":
			loaders[l] = toml.Load
		default:
			return loaders
		}
	}

	return loaders
}

// goConfig should probably be private so it can only be set through the new method.
// This does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
	// fullWith is used for validating the 'with' list. Every item in 'with' must be in 'fullWith'.
	// fullWith includes the default with list + custom loader names.
	fullWith []string

	// with is a list of loaders by name in the order they will be loaded.
	with []string // list of loaders in play.

	fExts       map[string][]string
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
	err = util.AreStructPointers(appCfgs...)

	stdFlgs := &standardFlags{}
	cfgs := make([]interface{}, 0)
	cfgs = append(cfgs, stdFlgs)
	cfgs = append(cfgs, appCfgs...)
	nGrps := node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, cfgs...)

	// Initialize renderer.
	//
	// Default values are recorded with the renderer on initialization.
	g.renderer, err = render.New(render.Options{
		Preamble:        g.showPreTxt,
		Conclusion:      g.showPostTxt,
		FieldNameFormat: g.showNameFmt,
	}, nGrps[1:])
	if err != nil {
		return err
	}

	// Choose how to render flag help screen and if to load
	// in app config values.
	flgLdr := g.flagLoader()
	if itemIn("flag", g.with) == "" {
		// flag excluded, don't render app configs. Standard
		// help screen is still provided.
		err = flgLdr.Load(nGrps[0:1]) // parse and load flags
	} else {
		err = flgLdr.Load(nGrps) // parse and load flags
	}
	if err != nil {
		return err
	}

	// Handle showing app version.
	if stdFlgs.ShowVersion {
		g.showVersion()
	}

	// Initialize standard loaders/unloaders.
	for _, name := range g.with {
		if _, ok := g.loaders[name]; !ok {
			switch name {
			case "env":
				g.loaders[name] = &env.Loader{}
				g.unloaders[name] = &env.Unloader{}
			case "toml":
				g.loaders[name] = toml.NewLoader(stdFlgs.ConfigPath)
				g.unloaders[name] = toml.NewUnloader()
			case "yaml":
				g.loaders[name] = toml.NewLoader(stdFlgs.ConfigPath)
				g.unloaders[name] = toml.NewUnloader()
			case "json":
				g.loaders[name] = toml.NewLoader(stdFlgs.ConfigPath)
				g.unloaders[name] = toml.NewUnloader()
			case "flag":
				// Another flag loader to trump potential previous loads.
				g.loaders[name] = flg.NewLoader(flg.Options{})
			}
		}
	}

	// Generate config template (if option provided)
	err = g.genTemplate(stdFlgs.GenConfig, nGrps[1:])
	if err != nil {
		return err
	}

	// Read in all values.
	err = g.loadAll(stdFlgs.ConfigPath, nGrps)
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

// genTemplate writes a generated template
// either to a file (if file path provided) or stdout (otherwise).
//
// 'pth' can either be a stand-along file extension or a file path (with file extension).
//
// If 'pth' is empty then nil is returned. Otherwise either an error is returned or
// the application exits with os.Exit(0).
func (g *goConfig) genTemplate(pth string, nGrps []*node.Nodes) error {
	if pth == "" {
		return nil
	}
	pth, ext := g.parseGenPath(pth)

	// pick unloader
	u, err := g.unloaderFromFileExt(ext)
	if err != nil {
		return err
	}
	b, err := u.Unload(nGrps)
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

// unloaderFromFileExt returns an unloader from the provided file
// extension. If no unloader is found because of either a non-matching
// file extension or a non-initialized unloader then a UnloaderNotFoundErr
// is returned with a nil load.Unloader.
func (g *goConfig) unloaderFromFileExt(ext string) (load.Unloader, error) {
	for lName, lExts := range g.fExts {
		for _, lExt := range lExts {
			if ext == lExt {
				unl, ok := g.unloaders[lName]
				if ok {
					return unl, nil
				}
				return nil, &UnloaderNotFoundErr{lName: lName}
			}
		}
	}

	return nil, &UnloaderNotFoundErr{lExt: ext}
}

type UnloaderNotFoundErr struct {
	lName string // expected unloader name
	lExt  string // expected unloader file extension
}

func (ue UnloaderNotFoundErr) Error() string {
	if ue.lExt != "" {
		return fmt.Sprintf("unloader not initialized for file extension '.%v'", ue.lExt)
	}

	return fmt.Sprintf("unloader not initialized for name '%v'", ue.lName)
}

type LoaderNotFoundErr struct {
	lName string // expected loader name
	lExt  string // expected loader file extension
}

func (ue LoaderNotFoundErr) Error() string {
	if ue.lExt != "" {
		return fmt.Sprintf("unloader not initialized for file extension '.%v'", ue.lExt)
	}

	return fmt.Sprintf("unloader not initialized for name '%v'", ue.lName)
}

// showVersion will write the version to stderr and exit.
func (g *goConfig) showVersion() {
	fmt.Fprintln(os.Stderr, g.version)
	os.Exit(0)
}

// flagLoader returns a pre-configured flg.Loader.
func (g *goConfig) flagLoader() *flg.Loader {
	// Process special flags.
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

	return flgLdr
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
func (g *goConfig) parseGenPath(pth string) (fpth, ext string) {
	ext = path.Ext(pth)
	if ext == "" {
		// maybe pth is just an extension.
		if g.isValidExt(pth) {
			return "", pth
		}
	}

	return pth, ext

}

// isValidExt determines if the file extension output type is supported based
// on the current list of g.with values.
func (g *goConfig) isValidExt(ext string) bool {
	for _, v := range g.listFileExts() {
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
// app config node groups and nGrps[0] is the node group for the standard config (for standard flags).
//
// Expects "nGrps" to contain the standard config node group first followed by application
// config node groups.
func (g *goConfig) loadAll(pth string, nGrps []*node.Nodes) error {
	for _, w := range g.with {
		l, exists := g.loaders[w]
		if !exists {
			return &LoaderNotFoundErr{
				lName: w,
			}
		}
		l.Load(nGrps[1:])
		switch w {
		case "env":
			err := env.Load(nGrps[1:])
			if err != nil {
				return err
			}
		case "toml", "yaml", "json":
			if !doneFile && pth != "" {
				fl := file.NewLoader(pth)
				err := fl.Load(nGrps[1:])
				if err != nil {
					return err
				}

				doneFile = true
			}
		case "flag":
			err := flg.NewLoader(flg.Options{}).Load(nGrps)
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

// With sets which configuration loaders are enabled. Order matters.
// Configuration is loaded in the same order as the new specified with list.
//
// Values can be any of: "env", "toml", "yaml", "json", "flag". They
// can also be the names of custom loaders registered with RegisterLoader.
//
// If a custom loader is registered and With is called without the new custom name then
// the custom loader is ignored for loading.
//
// If a loader name doesn't exist then With panics.
func (g *goConfig) With(w ...string) *goConfig {
	newWith := make([]string, 0)

	for _, i := range w {
		newItem := itemIn(i, g.fullWith)
		if newItem == "" {
			panic(fmt.Sprintf("%v is not a registered loader", i))
		}

		newWith = append(newWith, newItem)
	}
	g.with = newWith

	return g
}

// RegisterLoader registers a custom LoadUnloader.
//
// The name is the name referenced when using With if specifying a custom subset of loaders.
//
// When using With you must also include the name of the custom registered loader or it will not be used.
//
// Using a standard loader name replaces that standard loader. If, for example "toml" is the provided
// custom name then the custom "toml" implementation will be used instead of the standard one. Note
// that custom implementations of standard file loaders will need to specify the desired fileExt file extensions.
//
// fileExt is optional and if specified indicates the configuration is found at a file with the provided
// file extension. Register more than one file extension by comma-separating the values. For example,
// to register both "yaml" and "yml" simple provide "yaml,yml". Optionally you can provide a "." in
// front of the extension. The behavior is the same. Therefore you could also provide ".yaml,.yml".
//
// Because of its special nature, overriding "flag" is not supported. Attempting to override "flag"
// will panic.
//
// Not providing name or loader will panic.
func (g *goConfig) RegisterLoader(name, fileExt string, loader load.LoadUnloader) *goConfig {
	if name == "flag" {
		panic("flag loader is special and cannot be replaced")
	}

	if name == "" {
		panic("loader name required")
	}

	if loader == nil {
		panic("loader required")
	}

	// Register with the full list of acceptable loaders.
	g.fullWith = append(g.fullWith, name)

	// Register file extension(s).
	switch len(fileExt) {
	case 0:
		g.fExts[name] = []string{""}
	default:
		exts := strings.Split(fileExt, ",")
		for i, ext := range exts {
			exts[i] = strings.TrimSpace(ext)
			exts[i] = strings.Trim(ext, ",")
		}

		g.fExts[name] = exts
	}

	// Set to load just before "flag" (if loading from flags last - otherwise make it the
	// last to load. The user can specify a completely custom order by calling With.
	switch len(g.with) {
	case 0:
		g.with = []string{name}
	case 1:
		g.with = []string{name, "flag"}
	default:
		g.with = append(g.with[:len(g.with)-1], name, "flag")
	}

	g.loaders[name] = loader
	g.unloaders[name] = loader

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

// listFileExts provides an ordered list of acceptable file
// extensions for the current goConfig instance. The extensions are ordered
// based on the current value of g.with.
func (g *goConfig) listFileExts() []string {
	allExts := make([]string, 0)

	for _, name := range g.with {
		exts, ok := g.fExts[name]
		if !ok {
			continue
		}

		// Skip names with no extensions.
		if len(exts) == 1 && exts[0] == "" {
			continue
		}

		allExts = append(allExts, exts...)
	}

	return allExts
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
