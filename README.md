# go-config

Effortless, stateful go configuration.

A straightforward go configuration library that supports flags, environment variables, toml, yaml and JSON 
configuration formats.

go-config also supports using multiple configuration format options at the same time. For example, you can provide 
flags and environment variables.

Because creating configuration files is tedious, go-config can generate configuration files for you. This helps remove
human error from configuration files and makes configuration files easier to maintain since you only need to make changes
in a single location.

All configuration options are controlled via struct tags. A 'struct' is the principal commanding player.

# Table of Contents

[Getting Started](#getting-started)
[Flags](#flags)
[Environment Variables](#environment-variables)
[Customize Help Screen](#customize-help-screen)
[Ignoring Fields](#ignoring-fields)
[Long Help Descriptions](#long-help-descriptions)
[Struct Help Description](#struct-help-description)
[Help Field Not Found](#help-field-not-found)
[Correct Struct Tag Formation](#correct-struct-tag-formation)
[Validation](#validation)
[LoadOrDie](#loadordie)
[Precedence](#precedence)
[Advanced](#advanced)
[Future Features Under Consideration](#future-features-under-consideration)
[Hot Loading](#hot-loading)

# Getting Started

Get the library:
```sh
> go get -u github.com/pcelvng/go-config
```

General use:
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
        os.Exit(1)
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

```sh
> ./myapp -db.un=myusername -db-pw=mypassword
```

Supports slices with basic types. By default
the value is assumed to be comma-separated. To
override the default provide the 'sep' tag value.

```sh
type options struct {
	Hosts []string `sep:";"`
}
```


You may provide flag name aliases.

```sh
type options struct {
    // Note: flag name conflicts will cause config to return an error message 
    // and terminate the program.
    DBName `flag:"db-name,n"` // "db-name" or "n" can be supplied.
}
```

```sh
> ./myapp -db-name=mydbname 

# using alias
> ./myapp -n=mydbname 
```


You may provide a description with the 'help' tag.

```sh
type options struct {
    DBName `flag:"db-name,n" help:"Database name.`
}
```

You may ignore flag struct fields using a '-' value.

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
        Host: "localhost:5432", // Default host value.
    }

    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(1)
    }
}

type options struct {
    Host     string `flag:"db-host,host" help:"The db host:port."`
    Username string `flag:"db-un" help:"The db username."`
    Password string `flag:"db-pw" help:"The db password."`
}
```

You may ask for help.

```sh
> ./myapp -h # or --help

-config,-c      The config file path (if using one). File extension must be one of "toml", "yaml", "yml", "json".
-gen,-g         Generate config template. One of "toml", "yaml", "json", "env".
-show           Show loaded config values and exit.
-version,v      Show application version and exit.

-db-host,host   The db host:port. (default: localhost:5432)
-db-un          The db username.
-db-pw          The db password.

```

You may disable flags entirely by taking advantage of the "With" method described below

# Environment Variables

The default environment naming converts the field name to screaming snake case. Embedded struct fields
are namespaced using the struct field name.

```sh
type options struct {
    MyConfigField   string        
    Database        DB
}

type DB struct {
    Host     string
    Username string
    Password string
}

...
# Expected environment variables.
MY_CONFIG_FIELD=fieldvalue;
DATABASE_HOST=localhost:5432;
DATABASE_USERNAME=root;
DATABASE_PASSWORD=pw;
```

You can override the default naming with the "env" tag.
```sh
type options struct {
    MyConfigField   string  `env:"NEW_NAME"`        
    Database        DB      `env:"DB"`
}

type DB struct {
    Host     string
    Username string `env:"UN"`
    Password string `env:"PW"`
}

...
# Expected environment variables.
NEW_NAME=fieldvalue;
DB_HOST=localhost:5432;
DB_UN=root;
DB_PW=pw;
```

You can generate an env template with pre-populated default values.

```sh

func main() {
    appCfg := options{
        DBHost: "localhost:5432", // Default value.
    }
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(1)
    }
}

type options struct {
    DBHost      string `env:"DB_HOST"`   
    DBUsername  string `env:"DB_UN"`
    DBPassword  string `env:"DB_PW"`
}

...

> ./myapp -gen=env

#!/usr/bin/env bash

export DB_HOST=localhost:5432;
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

# Customize Help Screen 

Pre-pend text to the default help screen.

```sh
package main

import ...

var hlp = `
Welcome to my application. The purpose of this application is to connect to the database
and demonstrate go-config.
`

func main() {
    appCfg := options{}

    config.AppHelp(hlp)
    config.LoadOrDie(&appCfg)
}

type options struct {
    Host     string `flag:"db-host,h" help:"The db host:port."`
    Username string `flag:"db-un" help:"The db username."`
    Password string `flag:"db-pw" help:"The db password."`
}

...

> ./myapp -help

Welcome to my application. The purpose of this application is to connect to the database
and demonstrate go-config.

-config,-c      The config file path (if using one). File extension must be one of "toml", "yaml", "yml", "json".
-gen,-g         Generate config template. One of "toml", "yaml", "json", "env".
-show           Show loaded config values and exit.
-version,v      Show application version and exit.

-db-host        The db host:port. (default: localhost:5432)
-db-un          The db username.
-db-pw          The db password.

``` 

# Ignoring Fields

You can disable a struct field entirely by providing the 'ignore' value in
the 'config' tag.

```sh
type options struct {
    // DBName disabled entirely.
    DBName `config:"ignore"` 
}
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
    config.Help("Host", "once upon a time there was a very long description....")
    config.Help("DB.Username", "a really long custom description for the username field...")
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

# Struct Help Description

Help descriptions can be provided for embedded structs.

# Help Field Not Found

If the specified variable field is not found the config will return an error.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.Help("DoesNotExist", "once upon a time there was a very long description....")
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

...

> ./myapp

"DoesNotExist" is a field that does not exist.

```

You may specify if a field is required. By default, fields are not required.

```sh
type options struct {
    Host    string `req:"true"`
    DB      DB     `req:"false"` // Allowed but unnessary.
}
```

# Correct Struct Tag Formation

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

By default all configuration modes are enabled. You may specify the exact modes you wish to use by calling
the "With" method.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    config.With("flag", "env", "toml", "json", "yaml")

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
```

# Validation 

You may choose to provide a Validate() hook for more complex validations and config related initialization. This is
also convenient from the perspective of unifying where initialization/config related errors come from.

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(1)
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
        return errors.New("Invalid username/password combination."
    }
}
```

# LoadOrDie

For simplification, consider using "LoadOrDie".

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // Default host value.
    }
    
    config.LoadOrDie(&appCfg)
}

type options struct {
    Username string `req:"true"`
    Password string
}
```

After reading in config values you can dump the values to stderr. By default, everything is shown but sensitive 
information may be omitted by setting the "show" tag to false.

"AddShowMsg" takes a string that, if not empty will be prepended to the shown
output. It must be called before "Load".

```sh
func main() {
    appCfg := options{
        Host: "localhost:5432", // default host value
    }
    
    config.AddShowMsg("myapp version 0.1.0")
    err := config.Load(&appCfg)
    if err != nil {
        println("err: %v", err.Error())
        os.Exit(1)
    }
    config.ShowValues() // Values dumped to stderr.
}

type options struct {
    Host     string
    Username string   `req:"true"`
    Password string   `show:"false"` // Default is "true".
}

...

> ./myapp -host=myhost:5432 -username=myusername -password=mypassword
myapp version 0.1.0

host:      "myhost:5432" [default: "localhost:5432"]
username:  "myusername" (required)
password:  [redacted]
```

You may show the values by providing the 'show' flag. If provided, the application will show all the 
config values and exit.

```sh
> ./myapp -show -host=myhost:5432 -username=myusername -password=mypassword
myapp version 0.1.0

host:      "myhost:5432" [default: "localhost:5432"]
username:  "myusername" (required)
password:  [redacted]
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
        CustomTimeField: time.
    }
}

# env example but same idea for other formats.
> ./myapp -gen=env

#!/usr/bin/env bash
export DEFAULT_FORMAT= ; # "2006-01-02T15:04:05Z07:00" (RFC3339)
export OTHER_STANDARD_FORMAT= ; # "2006-01-02T15:04:05.999999999Z07:00" (RFC3339Nano)
export CUSTOM_TIME_FORMAT= ; # "2006/01/02"
```

# Precedence

When a field value is provided through more than one channel at once then the following takes precedence.

1. Flags
2. Config file (value from one of the config files)
3. Environment
4. Default value

Defaults overwritten by environment variables overwritten by config file values overwritten by flags. Flag values always
trump.

# Advanced

Supported advanced features allow you to customize default behavior such as implementing as custom loader
or implementing a custom help screen or implementing a custom output of the calling "Show()".

## Custom Renderers

Out of the box go-config supports the ability to dump the contents of runtime application configuration. Showing the 
contents of provided configuration is helpful for debugging either locally during development or in production. For
this reason go-config supports the "--show" flag which, if provided will render the final configuration values. The same
rendering can be seen by calling the package "Show()" function which will render the final configuration and continue 
normal application execution. 

Some organizations or people may wish to customize this screen and go-config has an api to implement 
such as feature.





# Future Features Under Consideration

1. Load from consul.
2. Load from etcd.
3. Load from vault.
4. Full template support for help menu customization.
5. Flag "commands".
6. Hot loading (see below for discussion).
7. Hot loading from an HTTP endpoint.
8. HTTP call for configuration state and meta configuration info (like source type).
9. Support for env loaded from a file.
10. Support loading files from multiple locations (useful for loading from common default paths first).
11. First class CLI tools.
12. Support for "options" flag to list and automatically validate a short list of options.
13. Support for field level validation from "validate" field tag (https://github.com/go-validator/validator).

# Hot Loading

Hot loading is the practice of re-loading configuration values after the application has started. In general, we feel 
applications should load configuration once upon initialization and keep configuration state for the duration of 
instance life.

Hot loading can lead to difficult-to-manage application state which in turn leads to:
- Increased difficulty debugging.
- Inconsistent state across many instances of the same application.
- Increased application logic to handle state. 

However, we also feel there are some good use cases such as:
- Rotating passwords.
- Updating configuration for front-line "always on" applications.
- Updating shortlists such as a blacklist or whitelist without needing to reload the application.

