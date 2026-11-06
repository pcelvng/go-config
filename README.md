# go-config
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/hydronica/go-config)
![Tests](https://github.com/github/docs/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/hydronica/go-config)](https://goreportcard.com/report/github.com/hydronica/go-config)
[![codecov](https://codecov.io/gh/hydronica/go-config/branch/master/graph/badge.svg)](https://codecov.io/gh/hydronica/go-config)

Effortless, stateful go configuration.

A straightforward go configuration library that supports flags, environment variables, toml, yaml and JSON 
configuration formats.

go-config also supports using multiple configuration format options at the same time. For example, you can provide 
flags and environment variables.

Because creating configuration files is tedious, go-config can generate configuration files for you. This helps remove
human error from configuration files and makes configuration files easier to maintain since you only need to make changes
in a single location.

All configuration options are controlled via struct tags. A 'struct' is the principal commanding player.

# Getting Started

Get the library:
```sh
> go get github.com/hydronica/go-config
```

Use in your application:
```sh
package main

import (
    "os"
    
    "github.com/hydronica/go-config"
)

func main() {
    appCfg := options{}
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}

type options struct {
    Username string
    Password string
}
``` 

# Flags

Flag key defaults are lowercase separated by dashes. 

```sh
// options example
type options struct {
    MyUsername string // Defaults to "my-username" as the flag key.
}
```

Flags follow the same default go-lang flag behavior. For example, there is no difference between the following two 
usages:

```sh 
> ./myapp -my-username=username
```

```sh 
> ./myapp --my-username=username
```

You may customize the flag name by providing the 'flag' struct tag.

```sh
type options struct {
    DBName string `flag:"db-name"`
}
```

`Load` registers `-show`, `-g`/`-gen`, `-c`/`-config`, and `-v`/`-version` on that call's flag set. `-v` and `-version` are registered only when `Version` was called. Those names are reserved while the matching option is on. 

Free the name with `Disable(OptShow)`, `Disable(OptGenConf)`, or `Disable(OptFiles)`, or by not calling `Version`. These flags are not on the standard `flag.CommandLine`, so use `Load` instead of `flag.Parse`. Global flags with any other name are still parsed, so a flag registered by another library keeps working. `Load` can be called again in the same process.

You may embed structures. A blank struct tag inherits the parent prefix; a non-blank tag
adds a namespace segment (`.` for flags, `_` for env). Use `,omitprefix` on a leaf field
to opt it out of the parent prefix.

```sh
type options struct {
    Config struct {
        Creds DBCreds
        Host  string `flag:"host,omitprefix"` // -host (not -app.host)
    } `flag:"app"` // -app.un, -app.pw for nested creds
}

type DBCreds struct {
    Username string `flag:"un"`
    Password string `flag:"pw"`
}
```

You may provide flag aliases (for shorter referencing)

```sh
type options struct {
    DBName string `flag:"db-name,n"` // "db-name" or "n" can be supplied.
    // Note: flag name conflicts (two or more flags with the same name) will cause config to return an error message 
    // and terminate the program.
}
```

```sh
> myapp -db.un=myusername -db-pw=mypassword
```


You may provide a description.

```sh
type options struct {
    DBName string `flag:"db-name,n" comment:"It's the db name."` // "db-name" or "n" can be supplied.
    // Note: flag name conflicts (two or more flags with the same name) will cause config to return an error message 
    // and terminate the program.
}
```

You may ignore flag struct fields.

```sh
type options struct {
    DBName string `flag:"-"` // DBName disabled for flags.
}
```

You may provide a default value. Regardless of the config input type a default is set by providing a struct value on
initialization.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}

type options struct {
    Host     string `flag:"db-host,h" comment:"The db host:port."`
    Username string `flag:"db-un" comment:"The db username."`
    Password string `flag:"db-pw" comment:"The db password."`
}
```

You may ask for help.

```sh
> myapp -h # or --help
myapp

Available Flags:
-config,-c      The config file path (if using one). File extension must be one of "toml,yaml,yml,json,ini"
-gen,-g         Generate a config template file. Accepts one of "toml,yaml,yml,json,env", sends the template 
                to stdout and exits. Default values are pre-populated in a template. The 'env' template generates
                the environment values with a shebang for execution in a shell script file.
-show           Will show all config values and exit the application.

-db-host,-h     The db host:port. (default: localhost:5432)
-db-name,-n     It's the db name.
-db-pw
-db-un

```

You may set one or more default config file paths. Paths are tried in order; the first that
exists is loaded. If none exist, a warning is logged and loading continues with env and flag
values. An explicit `-c` or `-config` flag overrides the defaults. Missing defaults are listed
in the `-c` help text.

```sh
func main() {
    appCfg := options{}
    err := config.New(&appCfg).ConfigPath("config.toml", "config.yaml").Load()
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}
```

By default, relative default paths resolve against the working directory. Call
`RelativeToBinary()` to resolve them against the directory containing the executable instead.
In that mode, absolute paths and `..` segments are rejected so defaults cannot escape the
binary directory. Call `RelativeToWorkDir()` to switch back.

```sh
func main() {
    appCfg := options{}
    err := config.New(&appCfg).
        ConfigPath("config.toml").
        RelativeToBinary().
        Load()
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}
```

You may disable flags entirely. Note, general config flags such as the '-gen' flag are not turned off and will still
be shown on the help screen.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.DisableFlags()
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}

type options struct {
    Host    string
    DB      DB
}

type DB struct {
    Username string 
}
```

## Environment Variables

The default naming of environment variables is to make the field name uppercase separated by underscores.

```sh
type options struct {
    Host    string             // Defaults to 'HOST'.
    DB      DB      `env:"DB"` // Acts as a namespace for structs.
}

type DB struct {
    Username string `env:"UN"` 
    Password string `env:"PW"`
}

...
# Expected environment variables.
HOST=myhost;
DB_UN=myusername;
DB_PW=mypassword;
```

You can generate an env template.

```sh
type options struct {
    Host    string             // Defaults to 'HOST'.
    DB      DB      `env:"DB"` // Acts as a namespace for structs.
}

type DB struct {
    Username string `env:"UN"` 
    Password string `env:"PW"`
}

...

> ./myapp -gen=env

#!/usr/bin/env bash

export HOST=localhost:5432;
export DB_UN=;
export DB_PW=;

# alternatively write directly to bash file.
> ./myapp -gen=env > myconfig.sh
```

When assigning structs as field values you may ignore the value as a prefix by using the
`omitprefix` tag value on struct fields, or `,omitprefix` on leaf fields under a prefixed parent.

```sh
type options struct {
    Host  string `env:"HOST,omitprefix"` // HOST (not APP_HOST)
    DB    DB     `env:"omitprefix"`      // subtree uses flat prefix (UN, PW)
    Block Block  `env:"APP"`             // APP_UN, APP_PW for nested creds
}

type DB struct {
    Username string `env:"UN"`
    Password string `env:"PW"`
}

type Block struct {
    Creds DBCreds // blank tag inherits APP -> APP_UN
}

...

> ./myapp -gen=env

#!/usr/bin/env bash

export HOST=localhost:5432;
export UN=; # no prefix
export PW=; # no prefix
```

## Other General Options

You may prepend text to the help output with Description. Version enables `-v` and `-version`.

```go
func main() {
    appCfg := options{}
    err := config.New(&appCfg).
        Description("Welcome to my application.\n").
        Version("1.2.3").
        Load()
    if err != nil {
        log.Fatal(err)
    }
}

type options struct {
    Host     string `flag:"db-host" comment:"The db host:port."`
    Username string `flag:"db-un" comment:"The db username."`
    Password string `flag:"db-pw" comment:"The db password."`
}
``` 

You can disable a struct field entirely by providing the 'ignore' value in
the 'config' tag.

```sh
type options struct {
    DBName string `config:"ignore"` // DBName disabled entirely.
}
```

Put longer help text in the `comment` tag. go-config reads that tag when it builds flag usage.

You may specify if a field is required. By default, fields are not required.

```sh
type options struct {
    Host    string `req:"true"`
    DB      DB     `req:"false"` // Allowed but not necessary since this is the default.
}
```

Struct tags must be formed according to golang best practices. If not, then the option will not be honored.

```sh
type options struct {
    // Bad
    DB      string     `req` // Must provide the tag name followed by a colon and the value in quotes (no spaces).
}

type options struct {
    // Good
    DB      string     `req:"true"` // Must provide the tag name followed by a colon and the value in quotes (no spaces).
}
```

You may turn sources off with Disable, or replace the defaults with SetOptions.
Disable(OptFiles) removes `-c`/`-config`. SetOptions(OptFlag) keeps flags only.

```go
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    err := config.New(&appCfg).
        Disable(config.OptEnv | config.OptEnvFile | config.OptFiles).
        Load()
    if err != nil {
        log.Fatal(err)
    }

    // Flags only.
    err = config.New(&appCfg).SetOptions(config.OptFlag).Load()
    if err != nil {
        log.Fatal(err)
    }
}

type options struct {
    Host    string
    DB      DB
}
```

You may choose to provide a Validate() hook for more complex validations and config related initialization. Load calls Validate after the values are read, when the config implements it.

```go
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }

    err := config.Load(&appCfg)
    if err != nil {
        log.Fatal(err)
    }
}

type options struct {
    Username string `req:"true"`
    Password string
}

// Validate implements the config validator interface.
//
// Validate is called after the config values are read in.
func (o *options) Validate() error {
    if o.Username == "" && o.Password == "" {
        return errors.New("Invalid username password combination.")
    }
    return nil
}
```

Use LoadOrDie to print the error and exit with status 1. LoadOrDie runs Validate the same way Load does.

```go
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }

    // On error, go-config prints the error and exits with status 1.
    config.LoadOrDie(&appCfg)
}
```

After reading in config values you can write them with Show. Fields tagged `mask:"true"` print `****` when they have a value and print blank when they are zero. A masked struct masks each of its fields. Nil pointers print `<nil>`. This lets you confirm a secret was picked up without printing it.

```go
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }

    cfg := config.New(&appCfg)
    if err := cfg.Load(); err != nil {
        log.Fatal(err)
    }
    cfg.Show(os.Stdout)
}

type options struct {
    Host     string
    Username string `req:"true"`
    Password string `mask:"true"` // prints **** when set
}
```

```text
> ./myapp -host=myhost:5432 -username=myusername
Host: localhost:5432
Username: myusername
Password:
```

`-show` prints the same text and then exits. Pass a password to see the masked form.

```text
> ./myapp -show -host=myhost:5432 -username=myusername -password=mypassword
Host: myhost:5432
Username: myusername
Password: ****
```

All types support time.Time and time.Duration marshaling and unmarshaling. 

time.Time default expected format is time.RFC339. You can specify a custom format in
the value of the 'fmt' struct tag. Formatting is the same as that supported in the
time package. For readability and simplicity you can also supply time package variable
name of the format. For example, if you wanted to use the time.RFC3339Nano format the 'fmt'
tag/value would be `fmt:"RFC3339Nano"`. Unmarshaling will expect that format and marshaling
will place the default value in that format. Marshaling will also automatically place
an inline comment specifying the expected time format.

```sh
type options struct {
    DefaultFormat time.Time // Defaults to expect time.RFC339 format.
    OtherStandardFormat time.Time `fmt:"RFC339Nano"` // Expects the time.RFC339Nano format.
    CustomTimeField time.Time `fmt:"2006/01/02"`
}

func main() {
    cfg := &options{
        CustomTimeField: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
    }
}

# env example but same idea for other formats.
> ./myapp -gen=env

#!/usr/bin/env bash
export DEFAULT_FORMAT= ; # "2006-01-02T15:04:05Z07:00" (RFC3339)
export OTHER_STANDARD_FORMAT= ; # "2006-01-02T15:04:05.999999999Z07:00" (RFC3339Nano)
export CUSTOM_TIME_FORMAT= ; # "2006/01/02"
```

### Precedence

When a field value is provided through more than one avenue at once then the following takes precedence.

1. Flags
3. Config file (value from one of the config files)
2. Environment
4. Default value
