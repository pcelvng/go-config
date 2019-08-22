package env

import (
	"strings"
	"testing"
	"time"

	"github.com/jbsmith7741/trial"
)

func TestEncoder_Marshal(t *testing.T) {
	type mStruct struct {
		value string
	}
	fn := func(args ...interface{}) (interface{}, error) {
		b, err := NewEncoder().Marshal(args[0])
		s := strings.Replace(string(b), "#!/bin/sh\n\n", "", 1)
		s = strings.Replace(s, "export ", "", -1)
		return s, err
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
			Expected: "INT=1\nINT_8=2\nINT_16=3\nINT_32=4\nINT_64=5\n",
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
			Expected: "UINT=6\nUINT_8=7\nUINT_16=8\nUINT_32=9\nUINT_64=10\n",
		},
		"floats": {
			Input: &struct {
				Float32 float32
				Float64 float64
			}{
				Float32: 1.0,
				Float64: 2.2,
			},
			Expected: "FLOAT_32=1\nFLOAT_64=2.2\n",
		},
		"bool": {
			Input: &struct {
				BoolF bool `env:"BOOL1"`
				BoolT bool `env:"BOOL2"`
			}{BoolT: true, BoolF: false},
			Expected: "BOOL1=false\nBOOL2=true\n",
		},
		"string": {
			Input:    &struct{ String string }{String: "Hello"},
			Expected: "STRING=Hello\n",
		},
		"with tags": {
			Input: &struct {
				Int  int    `env:"COUNT" comment:"number of people in a room"`
				Name string `env:"-" comment:"ignore me"`
			}{
				Int:  10,
				Name: "Bob",
			},
			Expected: "COUNT=10\n",
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
			Expected: "TIME=2019-01-02T00:00:00Z\nC_TIME=2019-01-02\nWAIT_TIME=1h0m0s\n",
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
				String:   trial.StringP("5"),
				MyStruct: &mStruct{value: "c"},
			},
			Expected: "INT=1\nUINT=2\nFLOAT=3.4\nSTRING=5\n",
		},
		"empty pointers": {
			Input: &struct {
				Int      *int
				Uint     *uint
				Float    *float64
				String   *string
				MyStruct *mStruct
			}{
				// Empty for nil values.
			},
			Expected: "INT=0\nUINT=0\nFLOAT=0\nSTRING=\n",
		},
	}
	trial.New(fn, cases).Test(t)
}
