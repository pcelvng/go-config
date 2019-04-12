package flg

import (
	"errors"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/jbsmith7741/trial"
)

func TestNew(t *testing.T) {
	type tFlag struct {
		//Name  string
		Usage string
		Def   string
	}
	type Aint int
	type Astring string
	type AFloat64 float64
	type Auint uint
	fn := func(args ...interface{}) (interface{}, error) {
		f, err := New(args[0])
		if err != nil {
			return nil, err
		}
		result := make(map[string]*tFlag)
		f.FlagSet.VisitAll(func(flg *flag.Flag) {
			key := flg.Name
			result[key] = &tFlag{Usage: flg.Usage, Def: flg.DefValue}
		})
		return result, nil
	}
	cases := trial.Cases{
		"nil config": {
			ShouldErr: true,
		},
		"non pointer": {
			Input:     struct{}{},
			ShouldErr: true,
		},
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
				"int":    {Def: "1"},
				"int-8":  {Def: "2"},
				"int-16": {Def: "3"},
				"int-32": {Def: "4"},
				"int-64": {Def: "5"},
			},
		},
		"uints": {
			Input: &struct {
				Uint   uint
				Uint8  uint8
				Uint16 uint16
				Uint32 uint32
				Uint64 uint64
			}{
				Uint:   6,
				Uint8:  7,
				Uint16: 8,
				Uint32: 9,
				Uint64: 10,
			},
			Expected: map[string]*tFlag{
				"uint":    {Def: "6"},
				"uint-8":  {Def: "7"},
				"uint-16": {Def: "8"},
				"uint-32": {Def: "9"},
				"uint-64": {Def: "10"},
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
				"float-32": {Def: "1"},
				"float-64": {Def: "2.2"},
			},
		},
		"bool": {
			Input: &struct{ Bool bool }{Bool: true},
			Expected: map[string]*tFlag{
				"bool": {Def: "true"},
			},
		},
		"string": {
			Input: &struct{ String string }{String: "Hello"},
			Expected: map[string]*tFlag{
				"string": {Def: "Hello"},
			},
		},
		"with tags": {
			Input: &struct {
				Int  int    `flag:"Count" comment:"number of people in a room"`
				Name string `flag:"-" comment:"ignore me"`
			}{
				Int:  10,
				Name: "Bob",
			},
			Expected: map[string]*tFlag{
				"Count": {Def: "10", Usage: "number of people in a room"},
			},
		},
		"time": {
			Input: &struct {
				Time     time.Time
				CTime    time.Time `flag:"ctime" fmt:"2006-01-02"`
				WaitTime time.Duration
			}{
				Time:     trial.TimeDay("2019-01-02"),
				CTime:    trial.TimeDay("2019-01-02"),
				WaitTime: time.Hour,
			},
			Expected: map[string]*tFlag{
				"time":      {Def: "2019-01-02T00:00:00Z"},
				"ctime":     {Def: "2019-01-02"},
				"wait-time": {Def: "1h0m0s"},
			},
		},
		"text marshaler": {
			Input: &struct {
				MyStruct marshalStruct
				IStruct  marshalStruct `flag:"-"`
			}{
				MyStruct: marshalStruct{value: "a"},
				IStruct:  marshalStruct{value: "b"},
			},
			Expected: map[string]*tFlag{
				"my-struct": {Def: "a"},
			},
		},
		"alias no marshaler": {
			Input: &struct {
				Int    Aint
				Uint   Auint
				Float  AFloat64
				String Astring
			}{
				Int:    Aint(3),
				Uint:   Auint(4),
				Float:  AFloat64(12.32),
				String: Astring("hello"),
			},
			Expected: map[string]*tFlag{
				"int":    {Def: "3"},
				"uint":   {Def: "4"},
				"float":  {Def: "12.32"},
				"string": {Def: "hello"},
			},
		},
		"alias with marshaler": {
			Input: &struct {
				Number mAlias
			}{Number: mAlias(1)},
			Expected: map[string]*tFlag{
				"number": {Def: "one"},
			},
		},
		"pointers": {
			Input: &struct {
				Int *int
				//	Uint     *uint
				String   *string
				MyStruct *marshalStruct
			}{
				Int:      trial.IntP(1),
				String:   trial.StringP("a"),
				MyStruct: &marshalStruct{value: "c"},
			},
			Expected: map[string]*tFlag{
				"int":       {Def: "1"},
				"string":    {Def: "a"},
				"my-struct": {Def: "c"},
			},
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestUnmarshal(t *testing.T) {
	type tConfig struct {
		Int   int
		Int8  int8
		Int16 int16
		Int32 int32
		Int64 int64
	}

	type input struct {
		config interface{}
		args   []string
	}
	fn := func(args ...interface{}) (interface{}, error) {
		in := args[0].(input)
		os.Args = append([]string{"go-config"}, in.args...)
		f, err := New(in.config)
		if err != nil {
			return nil, err
		}
		err = f.Unmarshal(in.config)

		return in.config, err
	}
	cases := trial.Cases{
		"ints": {
			Input: input{
				config: &tConfig{},
			},
			Expected: &tConfig{},
		},
	}
	trial.New(fn, cases).Test(t)
}

type mAlias int

var nums = []string{"zero", "one", "two", "three", "four", "five"}

func (i mAlias) MarshalText() ([]byte, error) {
	if int(i) < len(nums) {
		return []byte(nums[i]), nil
	}
	return nil, errors.New("invalid number")
}

func (i *mAlias) UnmarshalText(b []byte) error {
	for k, v := range nums {
		if v == string(b) {
			*i = mAlias(k)
			return nil
		}
	}
	return errors.New("invalid number")
}

type marshalStruct struct {
	value string
}

func (m marshalStruct) MarshalText() ([]byte, error) {
	return []byte(m.value), nil
}

func (m *marshalStruct) UnmarshalText(b []byte) error {
	m.value = string(b)
	return nil
}
