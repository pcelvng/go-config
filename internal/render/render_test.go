package render

import (
	"fmt"
	"testing"

	"github.com/pcelvng/go-config/util/node"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	type Embedded struct {
		String string
		Int    int `show:"false" req:"true"`
	}

	type RenderMe struct {
		MyString string `help:"my help msg"`
		Int      int    `req:"true"`
		IntSlice []int
		ES       Embedded `env:"ENVPRE" json:"JSON" flag:"FLAG"`
	}
	rm := &RenderMe{
		MyString: "mystring",
		Int:      4,
		IntSlice: []int{1, 2, 3},
	}

	nGrps := node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, rm, rm)

	r, err := New(Options{
		Preamble:        "my preamble",
		Conclusion:      "my conclusion",
		FieldNameFormat: " as field",
	}, nGrps)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)

	rm.MyString = "newmystringval"
	rm.IntSlice = []int{4, 5, 6}
	rm.ES.String = "nowpopulated"
	r.Render()
	//fmt.Println(string(b))
}
