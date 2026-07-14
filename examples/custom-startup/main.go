// Example: wrapping go-config with a customized configuration library.
//
// This app uses examples/custom-startup/appconfig, which sets WithFlagOptions
// and WithShowOptions so both --help and normal startup show a branded screen
// (ASCII art, field defaults/resolved values, and redacted sensitive fields).
//
//	go run . -h
//	go run .
//	DB_PW=s3cret go run . --db-host=db.example.com:5432
package main

import (
	"fmt"
	"time"

	"github.com/pcelvng/go-config/examples/custom-startup/appconfig"
)

func main() {
	appCfg := options{
		RunDuration: time.Second * 1,
		Host:        "localhost:8080",
		DB: DB{
			Host:     "localhost:5432",
			Username: "default_username",
			Password: "change-me", // default; overridden by env/flag; always redacted on screen
		},
	}

	cfg := appconfig.New().Version("custom-startup-example 1.0.0")
	cfg.LoadOrDie(&appCfg)

	fmt.Println("waiting for " + appCfg.RunDuration.String() + "...")
	<-time.NewTicker(appCfg.RunDuration).C
	fmt.Println("done")
}

type options struct {
	RunDuration time.Duration `help:"How long the example app runs."`
	Host        string        `help:"HTTP listen host:port."`
	DB          DB            `env:"DB" flag:"db"`
}

type DB struct {
	Host     string `help:"The db host:port." req:"true"`
	Username string `env:"UN" flag:"un,u" help:"The db username."`
	Password string `env:"PW" flag:"pw,p" help:"The db password." show:"false"`
}
