# go-config

A straightforward go configuration library that supports flags, environment variables, toml, yaml and JSON 
configuration formats.

go-config also supports using multiple configuration format options at the same time. For example, you can provide 
flags and environment variables.

Because creating configuration files is tedious, go-config can generate configration files for you. This helps remove
human error from configuration files and makes configuration files easier to maintain since you only need to make changes
in a single location.

All configuration options are controlled via struct tags. A 'struct' is the principal commanding player.

# Getting Started

Get the library:
```sh
> go get github.com/pcelvng/go-config
```

Use in your application:
```sh
package main

import (
    "os"
    
    "github.com/pcelvng/go-config"
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

You may embed structures. 

```sh
type options struct {
    DBName  string `flag:"db-name"`
    DBCreds string `flag:"db"`
}

type DBCreds struct {
    Username string `flag:"un"`
    Password string `flag:"pw"`
}
```

You may provide flag aliases (for shorter referencing)

```sh
type options struct {
    DBName `flag:"db-name,n"` // "db-name" or "n" can be supplied.
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
    DBName `flag:"db-name,n" desc:"It's the db name.` // "db-name" or "n" can be supplied.
    // Note: flag name conflicts (two or more flags with the same name) will cause config to return an error message 
    // and terminate the program.
}
```

You may ignore flag struct fields.

```sh
type options struct {
    DBName `flag:"-"` // DBName disabled for flags.
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
    Host     string `flag:"db-host,h" desc:"The db host:port."`
    Username string `flag:"db-un" desc:"The db username."`
    Password string `flag:"db-pw" desc:"The db password."`
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

## Other General Options

You may customize the help screen.

```sh
package main

import (
    "log"
    "os"
    
    "github.com/pcelvng/go-config"
)

var hlp = `
{{app}}

Welcome to my application. The purpose of this application is to connect to the database
and demonstrate the power of go-config.

Flag Options:
{{flags}}
`

func main() {
    appCfg := options{}
    config.HelpTemplate(hlp)
    config.AppName("myapp") // Could also include the app version here.
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
    }
}

type options struct {
    Host     string `flag:"db-host,h" desc:"The db host:port."`
    Username string `flag:"db-un" desc:"The db username."`
    Password string `flag:"db-pw" desc:"The db password."`
}
``` 

You can disable a struct field entirely by providing the 'hide' tag.

```sh
type options struct {
    DBName `hide:"true"` // DBName disabled entirely.
}
```

For longer descriptions you may call the "Describe" package function.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.Describe("Host", "once upon a time there was a very long description....")
    config.Describe("DB.Username", "a really long custom description for the username field...")
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

If the specified describe field is not found then config returns an error.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.Describe("DoesNotExist", "once upon a time there was a very long description....")
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

...

> ./myapp

"DoesNotExist" is a field that does not exist.

```

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

You may disable any configuration type. This is great for simplifying the application user experience so the user 
is not bombarded with too many options.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.DisableFlags() // disables custom flags.
    
    // Disabling any of the file config formats will mean that file type is not accepted 
    // for the application and options to generate a template of that type are also disabled.
    //
    // Users may wish to disable one or more formats to simplify the user experience. In this way
    // users are not given too many configuration choices.
    config.DisableTOML()
    config.DisableYAML()
    config.DisableJSON()
    
    // You may disable all file configuration types at once to make the application only accept flags and env variables.
    // If all file config types are disabled then the default help screen and flags will no longer support the 'config'
    // option.
    config.DisableFiles()
    
    // You may disable env.
    config.DisableEnv()
    
    // You may also specify that 'only' one configuration type is active.
    config.OnlyFlags()
    config.OnlyEnv()
    config.OnlyTOML()
    config.OnlyJSON()
   
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
```

You may choose to provide a Validate() hook for more complex validations and config related initialization. This is
also convenient from the perspective of unifying where initialization/config related errors come from.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    
    err := config.LoadWithValidation(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(0)
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
        return errors.New("Invalid username password combination."
    }
}
```

Use "LoadOrDie" to automatically print the error and exit the program. This can further simplify initialization but
at the expense of losing a little control.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    
    // If there is an error then the config library will display the error and terminate the 
    // application.
    config.LoadOrDie(&appCfg)
    
    // alternatively...
    config.LoadWithValidationOrDie(&appCfg)
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
        return errors.New("Invalid username password combination."
    }
}
```

After reading in config values you can dump the values to stderr. By default, everything is shown but sensitive 
information may be omitted by setting the "show" tag to false.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    
    config.Load(&appCfg)
    config.ShowValues() // Values dumped to stderr.
}

type options struct {
    Host     string
    Username string `req:"true"`
    Password string `show:"false"` // default is "true"
}

...

> ./myapp -host=myhost:5432 -username=myusername -password=mypassword
Host:     "myhost:5432" (default: "localhost:5432")
Username: "myusername"
Password: [redacted]
```

You may show the values by providing the 'show' flag. If provided, the application will show all the 
config values and exit.

```sh
> ./myapp -show -host=myhost:5432 -username=myusername -password=mypassword

Host:     "myhost:5432" (default: "localhost:5432")
Username: "myusername"
Password: [redacted]
```


### Precedence

When a field value is provided through more than one avenue at once then the following takes precedence.

1. Flags
3. Config file (value from one of the config files)
2. Environment
4. Default value
