# go-config

[![Build Status](https://github.com/pcelvng/go-config/actions/workflows/build.yml/badge.svg)](https://github.com/pcelvng/go-config/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pcelvng/go-config.svg)](https://pkg.go.dev/github.com/pcelvng/go-config)
[![Go Report Card](https://goreportcard.com/badge/github.com/pcelvng/go-config)](https://goreportcard.com/report/github.com/pcelvng/go-config)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Effortless, stateful Go configuration.

`go-config` is a straightforward configuration library that loads values from flags, environment
variables, TOML, YAML, and JSON, all into a plain Go struct. You describe your configuration once as a
struct and `go-config` handles parsing, precedence, help screens, and template generation for you.

You can mix and match formats freely. For example, you can read defaults from a TOML file, override them
with environment variables, and override those with command-line flags, all at the same time.

Because hand-writing configuration files is tedious and error prone, `go-config` can also generate
config templates (env, TOML, YAML, JSON) directly from your struct, so the struct stays the single
source of truth.

## Features

- One struct definition drives flags, env vars, and TOML/YAML/JSON files.
- Sensible load precedence with the ability to fully customize the order.
- Auto-generated `--help` screen and config-file templates (`--gen`).
- Rich type support including `time.Duration`, `time.Time`, slices, nested structs, and pointers.
- Per-field control via struct tags (custom names, help text, required, redaction, separators, ignore).
- Optional global key prefixes for env and flag names.
- Register your own custom loaders/unloaders (e.g. for etcd, consul, or vault).

## Installation

```sh
go get github.com/pcelvng/go-config
```

`go-config` requires Go 1.26 or newer.

## Getting Started

```go
// myapp.go
package main

import (
	"fmt"
	"os"
	"time"

	config "github.com/pcelvng/go-config"
)

func main() {
	appCfg := options{
		RunDuration: time.Second * 1,
		EchoTime:    time.Now(),
		DuckNames:   []string{"Freddy", "Eugene", "Aladdin", "Sarah"},
		DB: DB{
			Host:     "localhost:5432",
			Username: "default_username",
		},
	}
	if err := config.Load(&appCfg); err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}

	// Print the loaded values.
	fmt.Println("configuration values:")
	if err := config.Show(); err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}

	// Run for RunDuration.
	fmt.Println("waiting for " + appCfg.RunDuration.String() + "...")
	<-time.NewTicker(appCfg.RunDuration).C

	fmt.Println("echo time: " + time.Now().Format(time.RFC3339))
	fmt.Println("done")
}

type options struct {
	RunDuration time.Duration // Supports time.Duration.
	EchoTime    time.Time     `fmt:"RFC3339"`      // Supports time.Time with Go-style formatting.
	DuckNames   []string      `sep:";"`            // Supports slices. (Default separator is ",".)
	IgnoreMe    int           `env:"-" flag:"-"`   // Ignore for specified loaders.
	DB          DB            `env:"DB" flag:"db"` // Supports nested struct types.
}

type DB struct {
	Name     string
	Host     string `help:"The db host:port."`
	Username string `env:"UN" flag:"un,u" help:"The db username."`
	Password string `env:"PW" flag:"pw,p" help:"The db password." show:"false"`
}
```

Build the app:

```sh
go build
```

### Built-in help menu

```sh
./myapp -h # or --help

  -c, --config string   Config file path. Extension must be toml|yaml|yml|json.
  -g, --gen string      Generate config template (json|env|toml|yaml).
      --show            Print loaded config values and exit.

      --run-duration duration   (default: 1s)
      --echo-time time          fmt: RFC3339 (default: 2020-11-30T17:04:00-07:00)
      --duck-names strings      (default: [Freddy;Eugene;Aladdin;Sarah])
      --db-name string
      --db-host string          The db host:port. (default: "localhost:5432")
  -u, --db-un string            The db username. (default: "default_username")
  -p, --db-pw string            The db password.
```

### Generating config templates

Save yourself the typing and generate a template from your struct:

```sh
./myapp -g=env
#!/usr/bin/env sh

export RUN_DURATION=1s
export ECHO_TIME=2020-11-30T17:04:41-07:00 # fmt: RFC3339
export DUCK_NAMES=[Freddy;Eugene;Aladdin;Sarah]
export DB_NAME=
export DB_HOST=localhost:5432 # The db host:port.
export DB_UN=default_username # The db username.
export DB_PW= # The db password.
```

`-g` (or `--gen`) accepts `env`, `toml`, `yaml`, or `json`.

## Supported Types

- `string`, `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `time.Duration` and `time.Time`
- Slices of the above (using a configurable separator)
- Nested structs and pointers to structs

## Struct Tags

Per-field behavior is controlled with struct tags. The table below lists every supported tag, its
accepted values, and an example.

| Tag      | Supported values                                                                                  | Example                          |
| -------- | ------------------------------------------------------------------------------------------------- | -------------------------------- |
| `env`    | A custom env var name; `-` to ignore; `omitprefix` (struct fields only); optional `,string` suffix | `env:"DB_HOST"`, `env:"-"`        |
| `flag`   | A custom flag name, optionally `name,alias` (alias must be one char); `-` to ignore; `omitprefix`; optional `,string` suffix | `flag:"username,u"`, `flag:"-"`  |
| `toml`   | A custom TOML key name                                                                            | `toml:"db_host"`                 |
| `yaml`   | A custom YAML key name; `,inline` to promote an embedded struct                                   | `yaml:"db_host"`, `yaml:",inline"` |
| `json`   | A custom JSON key name                                                                            | `json:"db_host"`                 |
| `help`   | Any text (shown in `--help` and as comments in generated templates)                               | `help:"The db host:port."`       |
| `fmt`    | A Go time layout, or a named layout (`RFC3339`, `RFC1123`, `Kitchen`, `UnixDate`, etc.) for `time.Time` fields | `fmt:"RFC3339"`, `fmt:"2006-01-02"` |
| `sep`    | Any separator string used for slice values (default `,`)                                          | `sep:";"`                        |
| `show`   | `true` or `false` — set `false` to redact the value when printing config (default `true`)         | `show:"false"`                   |
| `req`    | `true` or `false` — annotate the field as required in the help/printed output, e.g. `(required)` (default `false`; not auto-enforced — use the `Validator` interface to enforce) | `req:"true"`                     |
| `ignore` | `true` or `false` — ignore the field entirely (default `false`)                                   | `ignore:"true"`                  |
| `config` | `ignore` to ignore the field entirely (alternative to `ignore:"true"`)                            | `config:"ignore"`                |

Notes:

- **`,string` suffix** (env and flag only): treats slice elements as quoted strings, so values keep
  surrounding quotes when generated and have them stripped when parsed — e.g. `env:"NAMES,string"`.
- **`-` vs `omitprefix`**: `-` ignores a field for that loader; `omitprefix` keeps the field but drops
  the struct's name from the generated prefix (see [Omitting a struct prefix](#omitting-a-struct-prefix)).
- A field with no tag for a given loader uses its struct field name, re-cased to the loader's
  convention (e.g. `SCREAMING_SNAKE` for env, `kebab-case` for flags).

### All tags at a glance

```go
type Config struct {
	// env/flag/file name overrides + a short flag alias.
	Host string `env:"DB_HOST" flag:"host,H" toml:"db_host" yaml:"db_host" json:"db_host" help:"The db host:port." req:"true"`

	// Redacted when printing loaded config.
	Password string `flag:"pw,p" help:"The db password." show:"false"`

	// time.Time with a named layout.
	StartAt time.Time `fmt:"RFC3339"`

	// Slice with a custom separator.
	Tags []string `sep:";"`

	// Ignored entirely (both forms are equivalent).
	Internal  string `ignore:"true"`
	Internal2 string `config:"ignore"`

	// Ignored only for specific loaders.
	OnlyFromFile string `env:"-" flag:"-"`
}
```

### Nested structs and env prefixes

When a field is itself a struct, the parent field's name becomes a **prefix** for every key inside it.
For env vars the prefix is the field name in `SCREAMING_SNAKE_CASE`, joined to the nested keys with an
underscore (`_`). The same idea applies to the other loaders using their respective casing.

This means you should **not** repeat the parent name in the nested field's tag. Each nested field's `env`
tag should contain only its own (leaf) name:

```go
type Config struct {
	S3 S3Configuration
}

type S3Configuration struct {
	Endpoint string `env:"ENDPOINT"` // -> S3_ENDPOINT
	Region   string `env:"REGION"`   // -> S3_REGION
	Bucket   string `env:"BUCKET"`   // -> S3_BUCKET
}
```

A common gotcha is to include the parent name in the nested tag:

```go
type Config struct {
	S3 S3Configuration
}

type S3Configuration struct {
	Endpoint string `env:"S3_ENDPOINT"` // -> S3_S3_ENDPOINT (probably not what you want!)
}
```

Because the `S3` parent field is automatically prefixed, the tag above resolves to `S3_S3_ENDPOINT`.

To control the prefix itself, put an `env` tag on the **parent** struct field. This overrides the
default (field-name) prefix:

```go
type Config struct {
	S3 S3Configuration `env:"S3"` // Prefix is explicitly "S3".
}

type S3Configuration struct {
	Endpoint string `env:"ENDPOINT"` // -> S3_ENDPOINT
}
```

If you don't want any prefix at all, use the special `omitprefix` value described next.

### Omitting a struct prefix

When a struct field is itself a struct, its name is used as a prefix for the nested fields. You can drop
that prefix with the special `omitprefix` value on the `env` tag. This only works on struct and struct
pointer types.

```go
type options struct {
	Host string                     // Defaults to 'HOST'.
	DB   DB     `env:"omitprefix"`   // No prefix is expected or generated.
}

type DB struct {
	Username string `env:"UN"`
	Password string `env:"PW"`
}
```

```sh
./myapp -gen=env

#!/usr/bin/env sh

export HOST=localhost:5432
export UN= # no prefix
export PW= # no prefix
```

### Embedded (anonymous) structs

Embedding a struct **promotes** its fields, so they are treated as if they were declared directly on the
parent — there is no type-name prefix. This mirrors how Go's own `encoding/json` handles embedded structs.

```go
type DB struct {
	Host string
	Port int
}

type Config struct {
	DB           // embedded (anonymous)
	Name string
}
```

With the struct above, the fields are promoted across every loader:

```sh
# env (no "DB_" prefix)
export HOST=localhost
export PORT=5432
export NAME=myapp
```

```sh
# flags (no "db-" prefix)
./myapp --host=localhost --port=5432 --name=myapp
```

```yaml
# yaml / json / toml (top-level, no "db" nesting)
host: localhost
port: 5432
name: myapp
```

Two things to keep in mind:

- **YAML requires an inline tag.** `gopkg.in/yaml.v2` does not inline anonymous structs automatically, so
  add `yaml:",inline"` to the embedded field if you load YAML:

  ```go
  type Config struct {
      DB   `yaml:",inline"`
      Name string
  }
  ```

- **The embedded type must be exported.** Fields of an unexported embedded type (e.g. `db`) are not
  settable via reflection and are skipped by the env and flag loaders.

If you want the type name to act as a prefix instead of promoting the fields, use a **named** field rather
than embedding:

```go
type Config struct {
	DB   DB     // named field -> DB_HOST, --db-host, nested "db" in files
	Name string
}
```

## Long Help Descriptions

For longer help descriptions, call `FieldHelp`. Nested struct fields are addressed with a `.` between
members. Long help descriptions are typically rendered on the line above the field in generated
templates.

```go
func main() {
	appCfg := options{
		Host: "localhost:5432", // default host value
	}
	config.FieldHelp("Host", "once upon a time there was a very long description....")
	config.FieldHelp("DB.Username", "a really long custom description for the username field...")
	if err := config.Load(&appCfg); err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}
}

type options struct {
	Host string
	DB   DB
}

type DB struct {
	Username string
}
```

## Choosing Loaders

By default all loaders are enabled and applied in this order (later loaders win):

1. `env`
2. `toml`
3. `yaml`
4. `json`
5. `flag`

Use `With` to select the exact loaders you want. The values are loaded in the order you specify, so the
last loader listed takes precedence.

```go
func main() {
	appCfg := &options{}
	err := config.With("flag", "env", "toml", "json", "yaml").Load(appCfg)
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}
}
```

## Prefixed Keys (env and flags)

Apply a global prefix to all env and flag keys at construction time.

> Note: the prefix must be applied at construction and cannot be set after initialization.

```go
func main() {
	appCfg := &options{}
	cfg := config.NewWithPrefix("my_app").With("env")
	if err := cfg.Load(appCfg); err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}
}

type options struct {
	Host string // Loaded from MY_APP_HOST env var.
}
```

## Validation

If your config struct implements the `Validator` interface, `Validate` is called automatically after
loading. Return an error to fail the load.

```go
type options struct {
	Port int
}

func (o *options) Validate() error {
	if o.Port == 0 {
		return errors.New("port is required")
	}
	return nil
}
```

## Application Version

Provide a version string to enable the `--version` (`-v`) flag, which prints the version and exits.

```go
config.Version("myapp v1.2.3")
```

## Custom Loaders

Register your own loader/unloader to source configuration from anywhere (etcd, consul, vault, etc.) via
`RegisterLoadUnloader`. The registered name can then be used with `With`. See the
[package reference](https://pkg.go.dev/github.com/pcelvng/go-config) for the `LoadUnloader`, `load.Loader`,
and `load.Unloader` types.

## API Reference

Most package-level functions are thin wrappers around a default `*GoConfig` instance. Use `config.New()`
or `config.NewWithPrefix(prefix)` when you need an isolated instance.

| Function / Method        | Description                                                       |
| ------------------------ | ----------------------------------------------------------------- |
| `Load(cfgs ...any)`      | Load configuration into one or more struct pointers.              |
| `LoadOrDie(cfgs ...any)` | Like `Load` but prints the error and exits on failure.            |
| `With(names...)`         | Select and order the loaders to use.                              |
| `Show()` / `ShowValues()`| Print the loaded configuration values.                            |
| `Version(s)`             | Set the app version and enable the `--version` flag.              |
| `FieldHelp(name, txt)`   | Set help text for a field at runtime.                             |
| `FieldTag(name, k, v)`   | Override a struct field tag at runtime.                           |
| `SetConfigPath(path)`    | Set the config file path without using the `--config` flag.       |
| `DisableStdFlags()`      | Disable the standard flags (`--gen`, `--show`, etc.).             |
| `RegisterLoadUnloader()` | Register a custom loader/unloader.                                |
| `WithShowOptions(o)`     | Customize how loaded values are rendered.                         |
| `WithFlagOptions(o)`     | Customize flag parsing behavior.                                  |

Full documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/pcelvng/go-config).

## License

Released under the [MIT License](LICENSE).
