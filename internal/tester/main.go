package main

import (
	"time"

	"github.com/hydronica/go-config"
)

type opts struct {
	Name   string
	Value  int
	Enable bool
	Dura   time.Duration `json:"-"`
	Time   time.Time     `format:"2006-01-02" env:"TIME"`
}

func main() {
	c := &opts{}
	config.LoadOrDie(c)

}
