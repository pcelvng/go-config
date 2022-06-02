# go-config

Effortless, stateful go configuration.

A straightforward go configuration library that supports flags, environment variables, toml, yaml and JSON 
configuration formats.

go-config also supports using multiple configuration format options at the same time. For example, you can provide 
flags and environment variables.

Because creating configuration files is tedious, go-config can generate configuration files for you. This helps remove
human error from configuration files and makes configuration files easier to maintain since you only need to make changes
in a single location.

# Getting Started

Get the library:
```sh
> go get -u github.com/pcelvng/go-config
```

General use:
```go
// myapp.go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pcelvng/go-config"
)

func main() {
	appCfg := options{
		RunDuration: time.Second * 1,
		EchoTime:    time.Now(),
		DuckNames:   []string{"Freddy", "Eugene", "Alladin", "Sarah"},
		DB: DB{
			Host:     "localhost:5432",
			Username: "default_username",
		},
	}
	err := config.Load(&appCfg)
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}
	// show values
	fmt.Println("configuration values:")
	if err := config.Show(); err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(1)
	}

	// Run for RunDuration.
	fmt.Println("waiting for " + appCfg.RunDuration.String() + "...")
	<-time.NewTicker(appCfg.RunDuration).C

	fmt.Printf("echo time: " + time.Now().Format(time.RFC3339) + "\n")
	fmt.Println("done")
}

type options struct {
	RunDuration time.Duration // Supports time.Duration
	EchoTime    time.Time     `fmt:"RFC3339"`      // Suports time.Time with go-style formatting.
	DuckNames   []string      `sep:";"`            // Supports slices. (Default separator is ",")
	IgnoreMe    int           `env:"-" flag:"-"`   // Ignore for specified types.
	DB          DB            `env:"DB" flag:"db"` // Supports struct types.
}

type DB struct {
	Name     string
	Host     string `help:"The db host:port."`
	Username string `env:"UN" flag:"un,u" help:"The db username."`
	Password string `env:"PW" flag:"pw,p" help:"The db password." show:"false"`
}
``` 

Build app:
```sh
> go build 
```

Built in help menu:
```sh
> ./myapp -h # or --help

  -c, --config string   Config file path. Extension must be toml|yaml|yml|json.
  -g, --gen string      Generate config template (json|env|toml|yaml).
      --show bool       Print loaded config values and exit. 

      --run-duration duration   (default: 1s)
      --echo-time time          fmt: RFC3339 (default: 2020-11-30T17:04:00-07:00)
      --duck-names strings      (default: [Freddy;Eugene;Alladin;Sarah])
      --db-name string          
      --db-host string          The db host:port. (default: "localhost:5432")
  -u, --db-un string            The db username. (default: "default_username")
  -p, --db-pw string            The db password.

```

Generate config templates to save typing:
```sh
> ./myapp -g=env
#!/usr/bin/env sh

export RUN_DURATION=1s
export ECHO_TIME=2020-11-30T17:04:41-07:00 # fmt: RFC3339
export DUCK_NAMES=[Freddy;Eugene;Alladin;Sarah]
export DB_NAME=
export DB_HOST=localhost:5432 # The db host:port.
export DB_UN=default_username # The db username.
export DB_PW= # The db password.

```

When assigning structs as field values you may ignore the value as a prefix by using the "omitprefix" env value.
This special value only works on struct and struct pointer types.

```sh
type options struct {
    Host    string                     // Defaults to 'HOST'.
    DB      DB      `env:"omitprefix"` // No prefix is expected or generated.
}

type DB struct {
    Username string `env:"UN"` 
    Password string `env:"PW"`
}

...

> ./myapp -gen=env

#!/usr/bin/env bash

export HOST=localhost:5432;
export UN=; # no prefix
export PW=; # no prefix
```

# Long Help Descriptions

For longer help descriptions you may call the "Help" method. Embedded struct methods are 
addressed using "." in between members. Long help descriptions will likely be rendered on
the line above a field when rendering to a template.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.FieldHelp("Host", "once upon a time there was a very long description....")
    config.FieldHelp("DB.Username", "a really long custom description for the username field...")
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(1)
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

By default all configuration modes are enabled. You may specify the exact modes you wish to use by calling
the "With" method. 

NOTE: Configs are loaded in the same order specified by "With".

```sh
func main() {
    appCfg := &options{}
    err := config.With("flag", "env", "toml", "json", "yaml").Load(appCfg)
    if err != nil {
        fmt.Printf("err: %v\n", err.Error())
        os.Exit(1)
    }
}

type options struct {
	RunDuration time.Duration // Supports time.Duration
	EchoTime    time.Time     `fmt:"RFC3339"`      // Suports time.Time with go-style formatting.
	DuckNames   []string      `sep:";"`            // Supports slices. (Default separator is ",")
	IgnoreMe    int           `env:"-" flag:"-"`   // Ignore for specified types.
	DB          DB            `env:"DB" flag:"db"` // Supports struct types.
}

type DB struct {
	Name     string
	Host     string `help:"The db host:port."`
	Username string `env:"UN" flag:"un,u" help:"The db username."`
	Password string `env:"PW" flag:"pw,p" help:"The db password." show:"false"`
}
```

Prefixed Keys (env and flags):

NOTE: prefix must be applied at time of construction and cannot be set after initialization

```sh
func main() {
    appCfg := &options{}
    cfg := config.NewWithPrefix("my_app").With("env")
    err := cfg.Load(appCfg)
    if err != nil {
        fmt.Printf("err: %v\n", err.Error())
        os.Exit(1)
    }
}

type options struct {
	Host string // Loaded from MY_APP_HOST env var
}
```