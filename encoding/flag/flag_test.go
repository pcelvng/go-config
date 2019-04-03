package flag

import (
	"testing"

	"github.com/jbsmith7741/trial"
)

func TestNew(t *testing.T) {
	type tFlag struct {
		//Name  string
		Usage string
		Def   string
	}
	fn := func(args ...interface{}) (interface{}, error) {
		f, err := New(args[0])
		if err != nil {
			return nil, err
		}
		result := make(map[string]*tFlag)
		for key := range f.values {
			v := f.flagSet.Lookup(key)
			if v == nil {
				result[key] = nil
				continue
			}
			result[key] = &tFlag{Usage: v.Usage, Def: v.DefValue}
		}
		return result, nil
	}
	cases := trial.Cases{
		"ints": {
			Input: &struct {
				Int   int
				Int8  int8
				Int16 int16
				Int32 int32
				Int64 int64
			}{
				Int:   1,
				Int8:  2,
				Int16: 3,
				Int32: 4,
				Int64: 5,
			},
			Expected: map[string]*tFlag{
				"Int":   {Def: "1"},
				"Int8":  {Def: "2"},
				"Int16": {Def: "3"},
				"Int32": {Def: "4"},
				"Int64": {Def: "5"},
			},
		},
		"uints": {
			Input: &struct {
				Uint   int
				Uint8  int8
				Uint16 int16
				Uint32 int32
				Uint64 int64
			}{
				Uint:   6,
				Uint8:  7,
				Uint16: 8,
				Uint32: 9,
				Uint64: 10,
			},
			Expected: map[string]*tFlag{
				"Uint":   {Def: "6"},
				"Uint8":  {Def: "7"},
				"Uint16": {Def: "8"},
				"Uint32": {Def: "9"},
				"Uint64": {Def: "10"},
			},
		},
		"floats": {
			Input: &struct {
				Float32 float32
				Float64 float64
			}{
				Float32: 1.0,
				Float64: 2.2,
			},
			Expected: map[string]*tFlag{
				"Float32": {Def: "1"},
				"Float64": {Def: "2.2"},
			},
		},
		"bool": {
			Input: &struct{ Bool bool }{Bool: true},
			Expected: map[string]*tFlag{
				"Bool": {Def: "true"},
			},
		},
		"string": {
			Input: &struct{ String string }{String: "Hello"},
			Expected: map[string]*tFlag{
				"String": {Def: "Hello"},
			},
		},
	}
	trial.New(fn, cases).Test(t)
}
