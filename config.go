package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pcelvng/go-config/load"
	"github.com/pcelvng/go-config/load/env"
	flg "github.com/pcelvng/go-config/load/flag"
	"github.com/pcelvng/go-config/load/json"
	"github.com/pcelvng/go-config/load/toml"
	"github.com/pcelvng/go-config/load/yaml"
	"github.com/pcelvng/go-config/render"
	"github.com/pcelvng/go-config/util"
	"github.com/pcelvng/go-config/util/node"
)

// Load is a package wrapper around *GoConfig.Load().
func Load(appCfgs ...interface{}) error {
	return defaultCfg.Load(appCfgs...)
}

// LoadOrDie is a package wrapper around *GoConfig.LoadOrDie().
func LoadOrDie(appCfgs ...interface{}) {
	defaultCfg.LoadOrDie(appCfgs...)
}

// With is a package wrapper around *GoConfig.With().
func With(with ...string) *GoConfig {
	return defaultCfg.With(with...)
}

// RegisterLoadUnloader is a package wrapper around *GoConfig.RegisterLoadUnloader().
func RegisterLoadUnloader(loadUnloader *LoadUnloader) *GoConfig {
	return defaultCfg.RegisterLoadUnloader(loadUnloader)
}

// DisableStdFlags is a package wrapper around *GoConfig.DisableFlagHelp().
func DisableStdFlags() *GoConfig {
	return defaultCfg.DisableStdFlags()
}

// SetConfigPath is a package wrapper around *GoConfig.SetConfigPath().
func SetConfigPath(pth string) *GoConfig {
	return defaultCfg.SetConfigPath(pth)
}

// Version is a package wrapper around GoConfig.Version().
func Version(s string) *GoConfig {
	return defaultCfg.Version(s)
}

func WithShowOptions(o render.Options) *GoConfig {
	return defaultCfg.WithShowOptions(o)
}

func WithFlagOptions(o flg.Options) *GoConfig {
	return defaultCfg.WithFlagOptions(o)
}

func FieldHelp(fieldName, helpTxt string) *GoConfig {
	return defaultCfg.FieldHelp(fieldName, helpTxt)
}

func FieldTag(fieldName, tagName, helpTxt string) *GoConfig {
	return defaultCfg.FieldTag(fieldName, tagName, helpTxt)
}

// New creates a new config.
func New() *GoConfig {
	return NewWithPrefix("")
}

// NewWithPrefix creates a new config with a global prefix for loaders.
func NewWithPrefix(prefix string) *GoConfig {
	cfg := &GoConfig{
		initialized: true,
		prefix:      prefix,
		lus: map[string]*LoadUnloader{
			// Note: "flag" is not listed here - it's a special case.
			"env": {
				Name:     "env",
				FileExts: []string{},
				Loader:   env.NewEnvLoader().WithPrefix(prefix),
				Unloader: env.NewEnvUnloader().WithPrefix(prefix),
			},
			"toml": {
				Name:     "toml",
				FileExts: []string{"toml"},
				Loader:   toml.NewTOMLLoadUnloader(),
				Unloader: toml.NewTOMLLoadUnloader(),
			},
			"yaml": {
				Name:     "yaml",
				FileExts: []string{"yaml", "yml"},
				Loader:   yaml.NewYAMLLoadUnloader(),
				Unloader: yaml.NewYAMLLoadUnloader(),
			},
			"json": {
				Name:     "json",
				FileExts: []string{"json"},
				Loader:   json.NewJSONLoadUnloader(),
				Unloader: json.NewJSONLoadUnloader(),
			},
			"flag": {
				Name:     "flag",
				FileExts: []string{},
				Loader:   flg.NewLoader(flg.Options{}).WithPrefix(prefix),
			},
		},
		with: []string{
			// Listed in the default load order.
			"env", // env trumps defaults.
			"toml",
			"yaml",
			"json",
			// "..." <- custom names are loaded here by default.
			"flag", // flag trumps all (by default - unless custom order specified).
		},
		stdFlgs:      &stdFlgs{},
		showOptions:  render.Options{},
		tagOverrides: make([]tagOverride, 0),
	}

	return cfg
}

type LoadUnloader struct {
	Name string

	// FileExts tells go-config what extensions to look for when matching a config file
	// name to a LoadUnloader. No file extensions means the config can be loaded by some
	// other means such as environment variables loading from the environment or from a server
	// like etcd, consul or vault.
	FileExts []string
	//LoadUnloader load.LoadUnloader

	// Loader is required.
	Loader load.Loader

	// Unloader is not required; if not present then config templates will not be generatable for it.
	Unloader load.Unloader
}

func (lu *LoadUnloader) canUnload() bool {
	return lu.Unloader != nil
}

type GoConfig struct {
	initialized bool

	// with is a list of loaders by name in the order they will be loaded.
	with []string

	// prefix is a global prefix for this loader for config keys in env or flags
	prefix string

	// lus contains a map by Name of registered LoadUnloaders.
	lus map[string]*LoadUnloader

	// flgLoader holds the flag loader for pre-loading to handle the help screen and
	// standard options.
	flgLoader *flg.Loader

	// stdFlgs is an instance of the standard flags for supporting pre-load
	// functionality such showing a help screen, application version, generating
	// config templates and determining which config file to load from.
	stdFlgs    *stdFlgs
	flgOptions flg.Options

	showOptions render.Options

	// tagOverrides stores struct field tag overrides allowing for long tag values and setting values at runtime.
	tagOverrides []tagOverride

	// showRenderer contains an instance of the showRenderer for customizing the display of
	// loaded values.
	showRenderer *render.Renderer

	// version contains the application name and version as provided by calling "Version".
	version string

	// stdFlgsDisabled will disable std flag support such as usage of the --gen flag.
	stdFlgsDisabled bool
}

type tagOverride struct {
	FieldName string
	Tag       string
	TagValue  string

	err   error
	found bool
}

var (
	cfgPathHelp   = "Config file path. Extension must be %s."
	genConfigHelp = "Generate config template (%s)."

	// TODO: built in support for validate struct tag.
	//validateTag = "validate" // See https://godoc.org/gopkg.in/go-playground/validator.v9

	defaultCfg = New()
)

// stdFlgs contains the set of standard flags used by the config library.
type stdFlgs struct {
	ConfigPath string `flag:"config,c" env:"-" toml:"-"` // Dynamically generated "help" text.

	// TODO: value can be path or extension. 'env' can also be 'sh'. 'env' or 'sh' is also attempts to make executable.
	Gen         string `flag:"gen,g" env:"-" toml:"-"` // Dynamically generated "help" text.
	ShowValues  bool   `flag:"show" env:"-" toml:"-" help:"Print loaded config values and exit."`
	ShowVersion bool   `flag:"version,v" env:"-" toml:"-" help:"Show application version and exit."`
}

// Load handles:
// - basic validation
// - flag pre-loading for handling standard flags and customizing the help screen
// - final config load
// - post load validation by:
//   - enforcing "validate" struct field tag directives TODO
//   - calling the custom Validate method (if implemented) TODO
func (g *GoConfig) Load(appCfgs ...interface{}) error {
	if !g.initialized {
		panic("uninitialized go config")
	}
	var err error

	if len(appCfgs) == 0 {
		return fmt.Errorf("nothing to load into")
	}

	// Verify all appCfgs are struct pointers.
	if err := util.AreStructPointers(appCfgs...); err != nil {
		return err
	}

	cfgs := make([]interface{}, 0)
	if !g.stdFlgsDisabled {
		cfgs = append(cfgs, g.stdFlgs)
	}
	cfgs = append(cfgs, appCfgs...)
	allNGrps := node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, cfgs...)

	stdNGrp := make([]*node.Nodes, 0)
	nGrps := allNGrps
	if !g.stdFlgsDisabled {
		stdNGrp = allNGrps[0:1]
		nGrps = allNGrps[1:]
	}

	// Apply field tag overrides.
	err = g.applyTagOverrides(nGrps)
	if err != nil {
		return err
	}

	// Initialize showRenderer.
	//
	// Default values are recorded with the showRenderer on initialization.
	// Standard flags are excluded.
	g.showRenderer, err = render.New(g.showOptions, nGrps, g.prefix)
	if err != nil {
		return err
	}

	// Handle flag pre-loading.
	//
	// Note: flags are loaded twice - once to handle
	// the help screen and handle standard options and again later on for the final
	// load resolution. This is the initial load.
	g.prepStdFlags(nGrps[0])
	preLdr := flg.NewLoader(g.flgOptions)
	// Handle flags, std flags enabled combinations. If both flags and std flags
	// are disabled then do not create a flag set at all.
	switch true {
	case itemIn("flag", g.with) == "" && !g.stdFlgsDisabled:
		// flags disabled
		// std flags enabled
		if err := preLdr.Load([]byte{}, stdNGrp); err != nil {
			return err
		}
	case itemIn("flag", g.with) == "flag" && g.stdFlgsDisabled:
		// flags enabled
		// std flags disabled
		if err := preLdr.Load([]byte{}, nGrps); err != nil {
			return err
		}
	case itemIn("flag", g.with) == "flag" && !g.stdFlgsDisabled:
		// flags enabled
		// std flags enabled
		if err := preLdr.Load([]byte{}, allNGrps); err != nil {
			return err
		}
	}

	if !g.stdFlgsDisabled {
		// Handle showing app version.
		if g.stdFlgs.ShowVersion {
			g.showVersion()
		}

		// Generate config template (if option provided).
		err = g.writeTemplate(g.stdFlgs.Gen, nGrps[1:])
		if err != nil {
			return err
		}
	}

	// Read in all values.
	// Note: If stdFlgs are disabled then g.stdFlags.ConfigPath will be empty
	// unless the user has set a default value via *GoConfig.SetConfigPath().
	err = g.loadAll(g.stdFlgs.ConfigPath, nGrps)
	if err != nil {
		return err
	}

	// ShowValues
	if g.stdFlgs.ShowValues {
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

func (g *GoConfig) applyTagOverrides(nGrps []*node.Nodes) error {
	for _, nGrp := range nGrps {
		for i, override := range g.tagOverrides {
			err := nGrp.SetTag(override.FieldName, override.Tag, override.TagValue)
			if err != nil && !override.found {
				g.tagOverrides[i].err = err
			}
			if err == nil {
				g.tagOverrides[i].err = nil
				g.tagOverrides[i].found = true
			}
		}
	}

	// validate field tag overrides.
	for _, override := range g.tagOverrides {
		if !override.found {
			return override.err
		}
	}

	return nil
}

// writeTemplate writes a generated template
// either to a file (if file path provided) or stdout (otherwise).
//
// 'path' can either be a stand-along file extension or a file path (with file extension).
//
// If 'path' is empty then nil is returned. Otherwise either an error is returned or
// the application exits with os.Exit(0).
func (g *GoConfig) writeTemplate(name string, nGrps []*node.Nodes) error {
	var err error
	if name == "" {
		return nil
	}

	// choose unloader
	lu, ok := g.lus[name]
	if !ok {
		return errors.New("unable to generate config template from unregistered name")
	}
	u := lu.Unloader

	if u == nil {
		return errors.New("template generation not supported for " + name)
	}

	// unload
	b, err := u.Unload(nGrps)
	if err != nil {
		return err
	}

	// write
	if _, err := os.Stdout.Write(b); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}

// hasRegisteredExt checks if at least one LoadUnloader is registered with the provided
// file extension.
func (g *GoConfig) hasRegisteredExt(ext string) bool {
	for _, lu := range g.lus {
		for _, lExt := range lu.FileExts {
			if lExt == ext {
				return true
			}
		}
	}

	return false
}

// loaderFromNameOrExt locates a load.Loader from a name and file extension (if provided).
// If one or more exts are defined on the LoadUnloader then an ext is required for a match.
// Otherwise match on the name.
//
// It's not expected that a match is always found since one particular file
// extension can be used over another at runtime.
func (g *GoConfig) loaderFromNameOrExt(name, ext string) load.Loader {
	for _, lu := range g.lus {

		// Match on name if no file exts defined.
		if len(lu.FileExts) == 0 {
			if lu.Name == name {
				return lu.Loader
			}

			continue
		}

		// Match on ext if file exts are defined.
		if len(lu.FileExts) > 0 {
			for _, lExt := range lu.FileExts {
				if lExt == ext {
					return lu.Loader
				}
			}
		}
	}

	return nil
}

func (g *GoConfig) loaderFromExt(ext string) (load.Loader, error) {
	for _, lu := range g.lus {
		for _, lExt := range lu.FileExts {
			if ext == lExt {
				return lu.Loader, nil
			}
		}
	}

	return nil, &LoaderNotFoundErr{lExt: ext}
}

func (g *GoConfig) loaderFromName(name string) (load.Loader, error) {
	lu, ok := g.lus[name]
	if !ok {
		return nil, &LoaderNotFoundErr{lExt: name}
	}

	return lu.Loader, nil
}

type UnloaderNotFoundErr struct {
	lName string // expected unloader name
	lExt  string // expected unloader file extension
}

func (ue UnloaderNotFoundErr) Error() string {
	if ue.lExt != "" {
		return fmt.Sprintf("unloader not available for file extension '.%v'", ue.lExt)
	}

	return fmt.Sprintf("unloader not available for name '%v'", ue.lName)
}

type LoaderNotFoundErr struct {
	lName string // expected loader name
	lExt  string // expected loader file extension
}

func (ue LoaderNotFoundErr) Error() string {
	if ue.lExt != "" {
		return fmt.Sprintf("loader not found for file extension '.%v'", ue.lExt)
	}

	return fmt.Sprintf("loader not found for '%v'", ue.lName)
}

type ConfigExtNotFoundErr struct {
	path string
}

func (ce ConfigExtNotFoundErr) Error() string {
	return fmt.Sprintf("filename extension not found for path '%v'", ce.path)
}

// showVersion will write the version to stderr and exit.
func (g *GoConfig) showVersion() {
	fmt.Fprintln(os.Stderr, g.version)

	os.Exit(0)
}

func (g *GoConfig) prepStdFlags(nGrp *node.Nodes) {
	// "config" standard flag.
	exts := g.allExts()
	if len(exts) > 0 {
		nGrp.SetTag("ConfigPath", "help", fmt.Sprintf(cfgPathHelp, strings.Join(exts, "|")))
	} else {
		nGrp.SetTag("ConfigPath", "flag", "-")
	}

	// "gen" standard flag.
	allNames := g.allNames()
	if len(allNames) > 0 {
		nGrp.SetTag("Gen", "help", fmt.Sprintf(genConfigHelp, strings.Join(allNames, "|")))
	} else {
		nGrp.SetTag("Gen", "flag", "-") // no exts - ignore
	}

	// "version" standard flag.
	if g.version == "" {
		nGrp.SetTag("ShowVersion", "flag", "-")
	}
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

// parsePath parses the provide file path returning the
// path value if it's a file path and the file extension. If no
// extension is discerned then ext is empty.
//
// The value can either be a standalone file extension such as "toml"
// or it can be a full file path ("with" extension) to write the
// generated config template.
//
// Returns an error if the extension is not registered.
//
// If "pth" is empty then no path or extension is returned and err == nil.
func (g *GoConfig) parsePath(pth string) (fpth, ext string, err error) {
	if pth == "" {
		return "", "", nil
	}

	ext = path.Ext(pth)
	if ext == "" {
		// maybe pth is just an extension.
		if g.isValidExt(pth) {
			return "", pth, nil
		}

		return "", "", errors.New("invalid file path")
	}

	return pth, ext, nil

}

// isValidExt determines if the file extension output type is supported based
// on the current list of g.with values.
func (g *GoConfig) isValidExt(ext string) bool {
	for _, v := range g.allExts() {
		if v == ext {
			return true
		}
	}

	return false
}

// ShowValues writes the values to os.Stderr.
func (g *GoConfig) ShowValues() error {
	return g.FShowValues(os.Stderr)
}

func (g *GoConfig) FShowValues(w io.Writer) error {
	b := g.showRenderer.Render()
	_, err := fmt.Fprintln(w, string(b))

	return err
}

// loadAll iterates through the "with" list and loads
// the config values into cfgs[1:] where cfgs[1:] are all the
// app config node groups and nGrps[0] is the node group for the standard config (for standard flags).
//
// Expects "nGrps" to contain the standard config node group first followed by application
// config node groups.
func (g *GoConfig) loadAll(fPath string, nGrps []*node.Nodes) error {
	// read in config file
	var cfgB []byte
	var err error
	if fPath != "" {
		cfgB, err = ioutil.ReadFile(fPath)
		if err != nil {
			return err
		}
	}

	_, pthExt, err := g.parsePath(fPath)
	if err != nil {
		return err
	}

	// Extension required if fPath is provided.
	if len(fPath) > 0 && len(pthExt) == 0 {
		return &LoaderNotFoundErr{lExt: pthExt}
	}

	// Extension must match at least one loader (when present).
	if len(pthExt) > 0 {
		if !g.hasRegisteredExt(pthExt) {
			return &LoaderNotFoundErr{lExt: pthExt}
		}
	}

	// Load all.
	for _, w := range g.with {
		if l := g.loaderFromNameOrExt(w, pthExt); l != nil {
			if err := l.Load(cfgB, nGrps); err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadOrDie calls Load and prints an error message and exits if there is an error.
func (g *GoConfig) LoadOrDie(appCfg ...interface{}) {
	err := g.Load(appCfg...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err.Error())
		os.Exit(1)
	}
}

// With sets which configuration loaders are enabled. Order matters.
// Configuration is loaded in the same order as the new specified "with" list.
//
// Values can be any of: "env", "toml", "yaml", "json", "flag" or names of
// custom loaders registered with RegisterLoadUnloader.
//
// If a loader name does not exist then With panics. Custom LoadUnloaders
// must be registered before calling With.
func (g *GoConfig) With(newWith ...string) *GoConfig {
	validNames := loadUnloaderNames(g.lus)

	for _, w := range newWith {
		if itemIn(w, validNames) == "" {
			panic(fmt.Sprintf("%v is not a registered loader options are %v",
				w, strings.Join(validNames, ", ")))
		}

		newWith = append(newWith, w)
	}
	g.with = newWith

	return g
}

func loadUnloaderNames(lus map[string]*LoadUnloader) []string {
	names := []string{"flag"}
	for name, _ := range lus {
		names = append(names, name)
	}

	return names
}

// RegisterLoadUnloader registers a LoadUnloader.
//
// The name is the name referenced when using With if specifying a custom subset of loaders.
//
// TODO: Change this behavior to include the custom loader unless "With" is exercised and the loader is not included.
// When using "With" you must also include the name of the custom registered loader or it will not be used.
//
// Using a standard loader name replaces that standard loader. If, for example "toml" is the provided
// custom name then the custom "toml" implementation will be used instead of the standard one. Note
// that custom implementations of standard file loaders will need to specify the desired fileExt file extensions.
//
// FileExts is optional and if specified indicates the configuration is found at a file with the provided
// file extension. At least one FileExts is required for loading from a configuration file. Multiple file
// extensions can be registered. Space characters and leading periods (".") are trimmed. Therefore you could
// also provide ".yaml" or "yaml". It doesn't matter.
//
// Because of its special nature, overriding "flag" is not allowed. Attempting to override "flag"
// will panic. Flags can be disabled by taking advantage of the "With" method and omitting "flag".
//
// Not providing name or LoadUnloader will panic.
func (g *GoConfig) RegisterLoadUnloader(loadUnloader *LoadUnloader) *GoConfig {
	// validate
	if err := validateLoadUnloader(loadUnloader); err != nil {
		panic(err.Error())
	}

	// sanitize extensions
	loadUnloader = sanitizeExts(loadUnloader)

	// update list
	g.lus[loadUnloader.Name] = loadUnloader

	// update with
	g.with = appendUnique(g.with, loadUnloader.Name)

	return g
}

func validateLoadUnloader(lu *LoadUnloader) error {
	if lu == nil {
		return errors.New("loadunloader is nil")
	}

	if lu.Name == "flag" {
		return errors.New("cannot use reserved name flag")
	}

	if lu.Name == "" {
		return errors.New("loadunloader name required")
	}

	if lu.Loader == nil {
		return errors.New("loader required")
	}

	return nil
}

func appendUnique(withList []string, name string) []string {
	for _, listName := range withList {
		if listName == name {
			return withList
		}
	}

	return append(withList, name)
}

func sanitizeExts(lu *LoadUnloader) *LoadUnloader {
	if lu == nil {
		return lu
	}

	for i, ext := range lu.FileExts {
		lu.FileExts[i] = strings.TrimSpace(ext)
		lu.FileExts[i] = strings.Trim(ext, ".")
	}

	return lu
}

// Version sets the application version. If a version is provided then
// the user can specify the --version flag to show the version. Otherwise the version flag
// is not seen on the help screen.
func (g *GoConfig) Version(v string) *GoConfig {
	g.version = v
	return g
}

func (g *GoConfig) WithShowOptions(o render.Options) *GoConfig {
	g.showOptions = o
	return g
}

func (g *GoConfig) WithFlagOptions(o flg.Options) *GoConfig {
	g.flgOptions = o
	return g
}

// DisableStdFlags will disable standard CLI options such as --gen.
//
// Note: This does not disable flag usage. To disable flags entirely
// call "With" providing the config types you wish to include.
func (g *GoConfig) DisableStdFlags() *GoConfig {
	g.stdFlgsDisabled = true
	return g
}

// FieldHelp allows adding a struct field help tag at runtime. Field names are dot "." separated
// values when referring to struct fields in struct fields.
//
// Field names are validated when "Load" is called.
func (g *GoConfig) FieldHelp(fieldName, helpTxt string) *GoConfig {
	g.tagOverrides = append(g.tagOverrides, tagOverride{
		FieldName: fieldName,
		Tag:       "help",
		TagValue:  helpTxt,
	})
	return g
}

// FieldTag allows for runtime modification of struct field tags. Field names are dot "." separated
// values when referring to struct fields in struct fields.
//
// Field names are validated when "Load" is called.
func (g *GoConfig) FieldTag(fieldName, tagName, helpTxt string) *GoConfig {
	g.tagOverrides = append(g.tagOverrides, tagOverride{
		FieldName: fieldName,
		Tag:       tagName,
		TagValue:  helpTxt,
	})
	return g
}

// SetConfigPath can be used to set the config path in a manner other than through the
// standard "--config,-c" standard flag.
//
// This can be useful when the user:
// - Wishes to set a default config flag value.
// - Wishes to set a config path while having standard flags disabled.
//
// Note: The --config,-c flag value will override this value unless standard flags
// are disabled.
func (g *GoConfig) SetConfigPath(pth string) *GoConfig {
	g.stdFlgs.ConfigPath = pth
	return g
}

// allExts returns a unique list of all included file extensions excluding "flag".
func (g *GoConfig) allExts() []string {
	exts := make([]string, 0)

	seen := map[string]bool{}
	for _, lu := range g.lus {
		if itemIn(lu.Name, g.with) == "" {
			continue
		}

		for _, ext := range lu.FileExts {
			if !seen[ext] {
				exts = append(exts, ext)
				seen[lu.Name] = true
			}
		}
	}

	return exts
}

// allNames returns a unique list all LoaderUnloader names that can unload.
func (g *GoConfig) allNames() []string {
	names := make([]string, 0)

	seen := map[string]bool{}
	for _, lu := range g.lus {
		if !lu.canUnload() {
			continue
		}

		if itemIn(lu.Name, g.with) == "" {
			continue
		}
		if !seen[lu.Name] {
			names = append(names, lu.Name)
			seen[lu.Name] = true
		}
	}

	return names
}

// itemIn will return 'i' when 'i' exists in 'all'.
func itemIn(name string, withList []string) string {
	for _, listName := range withList {
		if listName == name {
			return name
		}
	}

	return ""
}

// Validator can be implemented by the user provided config struct.
// Validate() is called after loading and running tag level validation.
type Validator interface {
	Validate() error
}
