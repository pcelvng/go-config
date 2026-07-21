package file

import (
	"errors"
	"testing"
	"time"

	"github.com/hydronica/trial"
)

const filePath = "../../../test/"

type SimpleStruct struct {
	Name    string
	Value   int
	Enable  bool
	Float64 float64
	Dura    time.Duration `json:"-"`
	Time    time.Time     `format:"2006-01-02"`
}

func TestLoad(t *testing.T) {
	fn := func(in string) (*SimpleStruct, error) {
		c := &SimpleStruct{}
		err := Load(in, c)
		return c, err
	}
	cases := trial.Cases[string, *SimpleStruct]{
		"toml": {
			Input: filePath + "test.toml",
			Expected: &SimpleStruct{
				Name:   "toml",
				Value:  10,
				Enable: true,
				Dura:   10 * time.Second,
				Time:   trial.TimeDay("2010-08-10"),
			},
		},
		"json": {
			Input: filePath + "test.json",
			Expected: &SimpleStruct{
				Name:   "json",
				Value:  10,
				Enable: true,
				Time:   trial.TimeDay("2010-08-10"),
				//Dura: 10 * time.Second, //TODO add support
			},
		},
		"yaml": {
			Input: filePath + "test.yaml",
			Expected: &SimpleStruct{
				Name:   "yaml",
				Value:  10,
				Enable: true,
				Dura:   10 * time.Second,
				Time:   trial.TimeDay("2010-08-10"),
			},
		},
		"env": {
			Input: filePath + ".env",
			Expected: &SimpleStruct{
				Name:    "env",
				Value:   10,
				Enable:  true,
				Float64: 99.9,
				Dura:    10 * time.Second,
				Time:    trial.TimeDay("2010-08-10"),
			},
		},
		"unknown": {
			Input:       "test.unknown",
			ExpectedErr: errors.New("unknown file type"),
		},
		"missing file": {
			Input:       filePath + "missing.toml",
			ExpectedErr: errors.New("no such file or directory"),
		},
	}
	trial.New(fn, cases).Test(t)
}
