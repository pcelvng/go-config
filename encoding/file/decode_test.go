package file

import (
	"errors"
	"testing"

	"github.com/jbsmith7741/trial"
)

type SimpleStruct struct {
	Name   string
	Value  int
	Enable bool
}

func TestLoad(t *testing.T) {
	fn := func(args ...interface{}) (interface{}, error) {
		c := &SimpleStruct{}
		f := args[0].(string)
		err := Load(f, c)
		return c, err
	}
	cases := trial.Cases{
		"toml": {
			Input:    "../../test/test.toml",
			Expected: &SimpleStruct{Name: "toml", Value: 10, Enable: true},
		},
		"json": {
			Input:    "../../test/test.json",
			Expected: &SimpleStruct{Name: "json", Value: 10, Enable: true},
		},
		"yaml": {
			Input:    "../../test/test.yaml",
			Expected: &SimpleStruct{Name: "yaml", Value: 10, Enable: true},
		},
		"unknown": {
			Input:       "test.unknown",
			ExpectedErr: errors.New("unknown file type"),
		},
		"missing file": {
			Input:       "../../test/missing.toml",
			ExpectedErr: errors.New("no such file or directory"),
		},
	}
	trial.New(fn, cases).Test(t)
}
