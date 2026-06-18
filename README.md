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

Per-field behavior is controlled with struct tags:

| Tag      | Description                                                                            |
| -------- | -------------------------------------------------------------------------------------- |
| `env`    | Override the env var name. Use `-` to ignore, or `omitprefix` on a struct (see below). |
| `flag`   | Override the flag name. Supports a short alias, e.g. `flag:"username,u"`. Use `-` to ignore. |
| `toml`   | Override the TOML key name.                                                            |
| `yaml`   | Override the YAML key name.                                                            |
| `json`   | Override the JSON key name.                                                            |
| `help`   | Help text shown in the `--help` screen and as comments in generated templates.         |
| `fmt`    | Time format for `time.Time` fields (Go reference layout or names like `RFC3339`).      |
| `sep`    | Separator used for slice values (default is `,`).                                      |
| `show`   | Set to `false` to redact a value when printing loaded config (e.g. secrets).           |
| `req`    | Set to `true` to mark a field as required.                                             |
| `ignore` | Set to `true` to ignore a field entirely (equivalent to `config:"ignore"`).            |

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
| `Load(cfgs...)`          | Load configuration into one or more struct pointers.              |
| `LoadOrDie(cfgs...)`     | Like `Load` but prints the error and exits on failure.            |
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
