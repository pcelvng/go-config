package main

import (
	//"flag"
	"fmt"
	"os"

	flg "github.com/pcelvng/go-config/encode/flag"
	//"flag"
)

//func main() {
//	appCfg := options{
//		Host: "localhost:5432", // default host value
//		DB: DB{
//			Username: "username",
//		},
//	}
//
//	err := config.Load(&appCfg)
//	if err != nil {
//		println("err: %v", err.Error())
//		os.Exit(0)
//	}
//}
//
//type options struct {
//	Host string
//	DB   DB
//}
//
//type DB struct {
//	Username string
//	Password string `env:"PW" help:"the password"`
//}

//func main() {
//	fs := flag.NewFlagSet("example", flag.ExitOnError)
//	//fs.String("blah", "default", "this is how it's used.")
//	fs.String("dablah", "default", "this is how it's used.")
//	err := fs.Parse(os.Args[1:])
//	if err != nil {
//		fmt.Println("err parsing flagset", err.Error())
//		os.Exit(1)
//	}
//	fs.PrintDefaults()
//	f := fs.Lookup("blah")
//	if f == nil {
//		fmt.Println("doesn't exist")
//		os.Exit(1)
//	}
//	fmt.Println(f.Value.String())
//	fmt.Println("done")
//}

var (
	helpBlock = `example

example is an example application to demonstrate the simplicity and usefulness 
of basic config.
`
)

func main() {
	stdFlgs := &standardFlags{
		ConfigPath:  "",
		GenConfig:   "",
		ShowValues:  false,
		ShowVersion: false,
	}
	appCfg := &options{
		Name: "ryan",
		Database: db{
			Host:     "localhost:5432",
			Username: "root",
			Password: "admin123",
		},
	}
	d := flg.NewDecoder(helpBlock)
	err := d.Unmarshal(stdFlgs, appCfg)
	if err != nil {
		fmt.Println("err unmarshaling flagset: ", err.Error())
		os.Exit(1)
	}

	if err != nil {
		fmt.Println("err initing flagset: ", err.Error())
		os.Exit(1)
	}

	//f.Unmarshal(appCfg)
	//f.PrintDefaults()
	fmt.Printf("%+v\n", stdFlgs)
	fmt.Printf("%+v\n", appCfg)
}

type standardFlags struct {
	ConfigPath  string `flag:"config,c" env:"-" toml:"-" help:"The config file path (if using one). Extension must be toml|yaml|yml|json."`
	GenConfig   string `flag:"gen,g" env:"-" toml:"-" help:"Generate config template (toml|yaml|json|env)."`
	ShowValues  bool   `flag:"show" env:"-" toml:"-" help:"Show loaded config values and exit."`
	ShowVersion bool   `flag:"version,v" env:"-" toml:"-" help:"Show application version and exit."`
}

type options struct {
	Name     string
	Database db `flag:"db"`
}

type db struct {
	Host     string `help:"db host"`
	Username string `flag:"un"`
	Password string `flag:"pw"`
}
