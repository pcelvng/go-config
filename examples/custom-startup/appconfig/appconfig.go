// Package appconfig is an example of wrapping go-config with a customized
// configuration library. It customizes the help screen and the loaded-values
// display used at application startup.
package appconfig

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pcelvng/go-config"
	flg "github.com/pcelvng/go-config/load/flag"
	"github.com/pcelvng/go-config/render"
)

// asciiArt is generated banner text featuring the pcelvng/go-config package
// (figlet -f slant go-config, with a pcelvng/ attribution line).
const asciiArt = `
pcelvng/
                                       _____
   ____ _____        _________  ____  / __(_)___ _
  / __ ` + "`" + `/ __ \______/ ___/ __ \/ __ \/ /_/ / __ ` + "`" + `/
 / /_/ / /_/ /_____/ /__/ /_/ / / / / __/ / /_/ /
 \__, /\____/      \___/\____/_/ /_/_/ /_/\__, /
/____/                                   /____/
`

const sensitiveMsg = "[redacted — sensitive]"

// Config wraps *config.GoConfig with a branded help/startup screen.
type Config struct {
	*config.GoConfig
}

// New returns a go-config instance customized for this application.
func New() *Config {
	g := config.New().
		WithShowOptions(render.Options{
			RenderFunc: renderScreen,
		}).
		WithFlagOptions(flg.Options{
			HelpFunc: renderHelp,
		})
	return &Config{GoConfig: g}
}

// Version sets the application version for the --version flag.
// It returns *Config so callers can keep chaining on the wrapper.
func (c *Config) Version(v string) *Config {
	c.GoConfig.Version(v)
	return c
}

// With keeps loader selection on the wrapper type.
func (c *Config) With(names ...string) *Config {
	c.GoConfig.With(names...)
	return c
}

// Load loads configuration and prints the customized startup screen.
// Pass -h / --help to see the same branded screen as the help menu.
func (c *Config) Load(appCfgs ...interface{}) error {
	if err := c.GoConfig.Load(appCfgs...); err != nil {
		return err
	}
	return c.ShowValues()
}

// LoadOrDie is like Load but prints the error and exits on failure.
func (c *Config) LoadOrDie(appCfgs ...interface{}) {
	if err := c.Load(appCfgs...); err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err.Error())
		os.Exit(1)
	}
}

// renderHelp builds the custom --help screen: ASCII art followed by each
// flag with its current (pre-load) value. Sensitive fields are redacted.
func renderHelp(_, _ string, fGroups [][]*flg.Flag) string {
	var b strings.Builder
	b.WriteString(strings.TrimLeft(asciiArt, "\n"))
	b.WriteString("\nConfiguration options:\n\n")

	for _, grp := range fGroups {
		for _, f := range grp {
			b.WriteString(flagLine(f))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	b.WriteString("Run without -h/--help to start the app (startup prints this same branded screen with resolved values).\n")
	return b.String()
}

func flagLine(f *flg.Flag) string {
	name := "--" + f.Name
	if f.Alias != "" {
		name = fmt.Sprintf("-%s, --%s", f.Alias, f.Name)
	}

	help := f.Help()
	typ := f.ValueType()
	val := f.String()

	display := val
	if !f.Show() {
		display = sensitiveMsg
	} else if typ == "string" {
		display = fmt.Sprintf("%q", val)
	}

	line := fmt.Sprintf("  %-28s (%s)", name, typ)
	line += fmt.Sprintf("  default/current: %s", display)
	if help != "" {
		line += "  — " + help
	}
	if !f.Show() {
		line += "  (value redacted due to being sensitive)"
	}
	return line
}

// renderScreen builds the customized startup / --show screen: ASCII art
// followed by each field with default and resolved values. Sensitive fields
// never reveal their values.
func renderScreen(_, _ string, fieldGroups [][]*render.Field) []byte {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, strings.TrimLeft(asciiArt, "\n"))
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "Loaded configuration:")
	fmt.Fprintln(buf)

	for _, grp := range fieldGroups {
		for _, f := range grp {
			fmt.Fprintln(buf, fieldLine(f))
		}
		fmt.Fprintln(buf)
	}

	return buf.Bytes()
}

func fieldLine(f *render.Field) string {
	resolved := f.ValueAfter
	def := f.ValueBefore

	if !f.Show {
		resolved = sensitiveMsg
		if !f.IsZero(f.ValueBefore) {
			def = sensitiveMsg
		}
	} else if f.Type == "string" {
		resolved = fmt.Sprintf("%q", f.ValueAfter)
		if !f.IsZero(f.ValueBefore) {
			def = fmt.Sprintf("%q", f.ValueBefore)
		}
	}

	line := fmt.Sprintf("  %-24s (%s)  resolved: %s", f.Name, f.Type, resolved)
	if !f.IsZero(f.ValueBefore) {
		line += fmt.Sprintf("  (default: %s)", def)
	}
	if f.Req {
		line += "  (required)"
	}
	if !f.Show {
		line += "  [redacted due to being sensitive]"
	}
	return line
}
