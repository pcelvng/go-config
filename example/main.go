package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pcelvng/go-config"
)

var (
	helpBlock = `example

example is an example application to demonstrate the simplicity and usefulness 
of basic config.
`
)

func main() {
	i := 32
	j := int64(64)
	n := time.Now()
	f := 6.4
	f32 := float32(3.2)
	appCfg := options{
		Host: "localhost:5432", // default host value
		DB: &DB{
			Username: "username",
		},
		MyBool:     true,
		MyPInt:     &i,
		MyPInt64:   &j,
		MyTime:     time.Now(),
		MyPTime:    &n,
		MyDuration: time.Second,
		MyInt32:    32,
		MyPFloat:   &f,
		MyPFloat32: &f32,
		MyUInt:     640,
	}

	err := config.
		With("env", "flag").
		Version("0.1.0").
		AddShowMsg("example 0.1.0").
		AddHelp(helpBlock).
		Load(&appCfg)
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
		os.Exit(0)
	}
}

type options struct {
	Host       string
	DB         *DB
	MyInt      int
	MyInt64    int64
	MyFloat    float64
	MyFloat32  float32
	MyPFloat   *float64
	MyPFloat32 *float32
	MyBool     bool
	MyPInt     *int
	MyPInt64   *int64
	MyInt32    int32
	MyPInt32   *int32
	MyUInt     uint
	MyUInt64   uint64
	MyUInt32   uint32
	MyUInt16   uint16
	MyUint8    uint8
	MyPUInt    *uint
	MyTime     time.Time  `fmt:"2006-01-02"`
	MyPTime    *time.Time `fmt:"2006-01-02"`
	MyDuration time.Duration
}

type DB struct {
	Username string `req:"true"`
	Password string `env:"PW" show:"false" help:"the password"`
}
