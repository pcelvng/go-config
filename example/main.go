package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pcelvng/go-config"
	"github.com/pcelvng/go-config/render"
)

func main() {
	appCfg := options{
		RunDuration: time.Second * 1,
		EchoTime:    time.Now(),
		DuckNames:   []string{"Freddy", "Eugene", "Alladin", "Sarah"},
		IsRich:      true,
		DB: &DB{
			Host:     "localhost:5432",
			Username: "default_username",
		},
	}
	//err := config.With("env").Load(&appCfg)
	//err := config.DisableStdFlags().With("env").Load(&appCfg)
	err := config.WithShowOptions(render.Options{
		Preamble:        "",
		Postamble:       "",
		FieldNameFormat: "env",
	}).Load(&appCfg)
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
	EchoTime    time.Time     `fmt:"RFC3339"`              // Suports time.Time with go-style formatting.
	DuckNames   []string      `sep:";"`                    // Supports slices. (Default separator is ",")
	IgnoreMe    int           `env:"-" flag:"-"`           // Ignore for specified types.
	DB          *DB           `env:"omitprefix" flag:"db"` // Supports struct types.
	IsRich      bool
}

type DB struct {
	Name     string
	Host     string `help:"The db host:port."`
	Username string `env:"UN" flag:"un,u" help:"The db username."`
	Password string `env:"PW" flag:"pw,p" help:"The db password." show:"false"`
	Omitted  string `env:"-" flag:"-" help:"The db password." show:"false"`
	//ES       EmbeddedString `env:"omitprefix"`
}

type EmbeddedString struct {
	MyString string
}
