package json

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

var bJSON = []byte(`{"name": "json", "value": 10, "enable":true}`)

func TestLoad(t *testing.T) {
	fn := func(args ...interface{}) (interface{}, error) {
		c := &SimpleStruct{}
		err := NewJSONLoadUnloader().Load(bJSON, node.MakeAllNodes(node.Options{
			NoFollow: []string{"time.Time"},
		}, c))
		return c, err
	}
	cases := trial.Cases{
		"json": {
			Input:    bJSON,
			Expected: &SimpleStruct{Name: "json", Value: 10, Enable: true},
		},
	}
	trial.New(fn, cases).Test(t)
}
