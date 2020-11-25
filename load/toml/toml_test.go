package toml

import (
	"testing"

	"github.com/jbsmith7741/trial"
	"github.com/pcelvng/go-config/util/node"
)

var bTOML = []byte(`name = "toml"
value = 10
enable = true
time = "2010-08-10T00:00:00Z"
float32 = 99.9`)

type SimpleStruct struct {
	Name   string
	Value  int
	Enable bool
}

func TestLoad(t *testing.T) {
	fn := func(args ...interface{}) (interface{}, error) {
		c := &SimpleStruct{}
		err := New().Load(bTOML, node.MakeAllNodes(node.Options{
			NoFollow: []string{"time.Time"},
		}, c))
		return c, err
	}
	cases := trial.Cases{
		"toml": {
			Input:    bTOML,
			Expected: &SimpleStruct{Name: "toml", Value: 10, Enable: true},
		},
	}
	trial.New(fn, cases).Test(t)
}
