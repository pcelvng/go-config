package config

type Hide struct {
	HField string `hide`
}

// goConfig should probably be private so it can only be set through the new method.
// this does mean that the variable can probably only be set with a ":=" which would prevent
// usage outside of a single function.
type goConfig struct {
}

// New verifies that c is a valid - must be a struct pointer (maybe validation should happen in parse)
// it goes through the struct and sets up corresponding flags to be used during parsing
func New(c interface{}) *goConfig {
	return &goConfig{}
}

// Parse and set the configs in the following priority from lowest to highest
// 1. environment variables
// 2. flags (exception of config and version flag which are processed first)
// 3. files (toml, yaml, json)
func (g *goConfig) Parse() error {
	return nil
}

// ParseFile loads config date from a file (yaml, toml, json)
// into the struct i.
// this would be used if we only want to parse a file and don't
// want to use any other features. This is more or less what multi-config does
func ParseFile(i interface{}) error {
	return nil
}

// ParseEnv is similar to ParseFile, but only checks env vars
func ParseEnv(i interface{}) error {
	return nil
}

// ParseFlag is similar to ParseFile, but only checks flags
func ParseFlag(i interface{}) error {
	return nil
}

// Version string that describes the app.
// this enables the -v (version) flag
func (g *goConfig) Version(s string) *goConfig {
	return g
}

// Description for the app, this message is prepended to the help flag
func (g *goConfig) Description(s string) *goConfig {
	return g
}

// DisableEnv tells goConfig not to use environment variables
func (g *goConfig) DisableEnv() *goConfig {
	return g
}

// DisableFile removes the c (config) flag used for defining a config file
func (g *goConfig) DisableFile() *goConfig {
	return g
}

// DisableFlag prevents setting variables from flags.
// Non variable flags should still work [c (config), v (version), g (gen)]
func (g *goConfig) DisableFlag() *goConfig {
	return g
}
