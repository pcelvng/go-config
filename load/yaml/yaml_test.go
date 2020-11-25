package yaml

import (
	"testing"

	"github.com/jbsmith7741/trial"
	"github.com/pcelvng/go-config/util/node"
)

type SimpleStruct struct {
	Name   string
	Value  int
	Enable bool
}

var bYAML = []byte(`name: "yaml"
value: 10
enable: true`)

func TestLoad(t *testing.T) {
	fn := func(args ...interface{}) (interface{}, error) {
		c := &SimpleStruct{}
		err := New().Load(bYAML, node.MakeAllNodes(node.Options{
			NoFollow: []string{"time.Time"},
		}, c))
		return c, err
	}
	cases := trial.Cases{
		"yaml": {
			Input:    bYAML,
			Expected: &SimpleStruct{Name: "yaml", Value: 10, Enable: true},
		},
	}
	trial.New(fn, cases).Test(t)
}
