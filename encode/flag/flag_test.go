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
				MyStruct mStruct
				IStruct  mStruct `flag:"-"`
			}{
				MyStruct: mStruct{value: "a"},
				IStruct:  mStruct{value: "b"},
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
				Int      *int
				Uint     *uint
				Float    *float64
				String   *string
				MyStruct *mStruct
			}{
				Int:      trial.IntP(1),
				Uint:     trial.UintP(2),
				Float:    trial.Float64P(3.4),
				MyStruct: &mStruct{value: "c"},
			},
			Expected: map[string]*tFlag{
				"int":       {Def: "1"},
				"uint":      {Def: "2"},
				"float":     {Def: "3.4"},
				"string":    {Def: ""},
				"my-struct": {Def: "c"},
			},
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestUnmarshal(t *testing.T) {
	type Aint int
	type Astring string
	type tConfig struct {
		Dura   time.Duration
		Time   time.Time `fmt:"2006-01-02"`
		Bool   bool
		String string

		Int   int
		Int8  int8  `flag:"int8"`
		Int16 int16 `flag:"int16"`
		Int32 int32 `flag:"int32"`
		Int64 int64 `flag:"int64"`

		Uint   uint
		Uint8  uint8  `flag:"uint8"`
		Uint16 uint16 `flag:"uint16"`
		Uint32 uint32 `flag:"uint32"`
		Uint64 uint64 `flag:"uint64"`

		Float32 float32 `flag:"float32"`
		Float64 float64 `flag:"float64"`

		IntP    *int     `flag:"intp"`
		UintP   *uint    `flag:"uintp"`
		StringP *string  `flag:"stringp"`
		FloatP  *float64 `flag:"floatp"`

		//alias'
		Aint    Aint
		Astring Astring
		Number  mAlias
	}

	type tStruct struct {
		MStruct mStruct  `flag:"mstruct"`
		PStruct *mStruct `flag:"pstruct"`
	}

	type input struct {
		config interface{}
		args   []string
	}
	fn := func(args ...interface{}) (interface{}, error) {
		in := args[0].(input)
		if in.config == nil {
			in.config = &tConfig{}
		}
		os.Args = append([]string{"go-config"}, in.args...)
		f, err := New(in.config)
		if err != nil {
			return nil, err
		}
		f.Parse()
		err = f.Unmarshal(in.config)

		return in.config, err
	}
	cases := trial.Cases{
		"time.duration": {
			Input:    input{args: []string{"-dura=10s"}},
			Expected: &tConfig{Dura: 10 * time.Second},
		},
		"time.duration (int)": {
			Input:    input{args: []string{"-dura=1000"}},
			Expected: &tConfig{Dura: 1000},
		},
		"time.Time": {
			Input:    input{args: []string{"-time=2010-01-02"}},
			Expected: &tConfig{Time: trial.TimeDay("2010-01-02")},
		},
		"int": {
			Input:    input{args: []string{"-int=10", "-int8=8", "-int16=16", "-int32=32", "-int64=64"}},
			Expected: &tConfig{Int: 10, Int8: 8, Int16: 16, Int32: 32, Int64: 64},
		},
		"uint": {
			Input:    input{args: []string{"-uint=10", "-uint8=8", "-uint16=16", "-uint32=32", "-uint64=64"}},
			Expected: &tConfig{Uint: 10, Uint8: 8, Uint16: 16, Uint32: 32, Uint64: 64},
		},
		"float": {
			Input:    input{args: []string{"-float32=3.2", "-float64=6.4"}},
			Expected: &tConfig{Float32: 3.2, Float64: 6.4},
		},
		"bool=true": {
			Input:    input{args: []string{"-bool=true"}},
			Expected: &tConfig{Bool: true},
		},
		"bool=false": {
			Input:    input{args: []string{"-bool=false"}},
			Expected: &tConfig{Bool: false},
		},
		"bool (default)": {
			Input:    input{args: []string{}},
			Expected: &tConfig{Bool: false},
		},
		"string": {
			Input:    input{args: []string{"-string=hello"}},
			Expected: &tConfig{String: "hello"},
		},
		"alias": {
			Input: input{args: []string{"-aint=10", "-astring=abc", "-number=two"}},
			Expected: &tConfig{
				Aint:    10,
				Astring: "abc",
				Number:  2,
			},
		},
		"pointers": {
			Input:    input{args: []string{"-intp=3", "-uintp=4", "-floatp=5.6"}},
			Expected: &tConfig{IntP: trial.IntP(3), UintP: trial.UintP(4), FloatP: trial.Float64P(5.6)},
		},
		"struct": {
			Input: input{
				config: &tStruct{},
				args:   []string{"-mstruct=abc", "-pstruct=def"},
			},
			Expected: &tStruct{MStruct: mStruct{"abc"}, PStruct: &mStruct{"def"}},
		},
		"keep value for default": {
			Input: input{
				config: &tConfig{
					Int:     1,
					Uint:    2,
					Float64: 3.4,
					String:  "abc",
				},
			},
			Expected: &tConfig{
				Int:     1,
				Uint:    2,
				Float64: 3.4,
				String:  "abc",
			},
		},
		"ignore maps": {
			Input: input{
				config: &struct {
					Value int
					Map   map[string]string
				}{},
				args: []string{"-value=10"},
			},
			Expected: &struct {
				Value int
				Map   map[string]string
			}{Value: 10},
		},
	}
	trial.New(fn, cases).SubTest(t)
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

type mStruct struct {
	value string
}

func (m mStruct) MarshalText() ([]byte, error) {
	return []byte(m.value), nil
}

func (m *mStruct) UnmarshalText(b []byte) error {
	m.value = string(b)
	return nil
}
