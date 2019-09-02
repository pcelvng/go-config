package main

import (
	"fmt"
	"os"

	"github.com/pcelvng/go-config"
)

var (
	helpBlock = `example

example is an example application to demonstrate the simplicity and usefulness 
of basic config.
`
)

func main() {
	appCfg := options{
		Host: "localhost:5432", // default host value
		DB: &DB{
			Username: "username",
		},
	}

	err := config.
		//With("env", "flag").
		Version("0.1.0").
		AddHelp(helpBlock).
		Load(&appCfg)
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(0)
	}
}

type options struct {
	Host string
	DB   *DB
}

type DB struct {
	Username string
	Password string `env:"PW" help:"the password"`
}
