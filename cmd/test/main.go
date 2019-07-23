package main

import (
	goconfig "github.com/pcelvng/go-config"
)

type config struct {
	Name  string
	Value int
}

func main() {
	c := &config{
		Name:  "hello",
		Value: 10,
	}
	goconfig.New(c).Version("1.1.0").LoadOrDie()
}
