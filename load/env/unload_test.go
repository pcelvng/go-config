package env

import (
	"testing"
	"time"

	"github.com/pcelvng/go-config/util/node"

	"github.com/jbsmith7741/trial"
)

func TestEncoder_Marshal(t *testing.T) {
	type mStruct struct {
		value string
	}
	fn := func(args ...interface{}) (interface{}, error) {
		b, err := NewEnvUnloader().Unload(node.MakeAllNodes(node.Options{
			NoFollow: []string{"time.Time"},
		}, args[0]))
		return string(b), err
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
			Expected: `#!/usr/bin/env sh

export INT=1
export INT_8=2
export INT_16=3
export INT_32=4
export INT_64=5
`,
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
			Expected: `#!/usr/bin/env sh

export UINT=6
export UINT_8=7
export UINT_16=8
export UINT_32=9
export UINT_64=10
`,
		},
		"floats": {
			Input: &struct {
				Float32 float32
				Float64 float64
			}{
				Float32: 1.0,
				Float64: 2.2,
			},
			Expected: `#!/usr/bin/env sh

export FLOAT_32=1
export FLOAT_64=2.2
`,
		},
		"bool": {
			Input: &struct {
				BoolF bool `env:"BOOL1"`
				BoolT bool `env:"BOOL2"`
			}{BoolF: true, BoolT: false},
			Expected: `#!/usr/bin/env sh

export BOOL1=true
export BOOL2=false
`,
		},
		"string": {
			Input: &struct {
				String         string
				StringAsString string `env:",string"`
			}{
				String:         "Hello",
				StringAsString: "World",
			},
			Expected: `#!/usr/bin/env sh

export STRING=Hello
export STRING_AS_STRING="World"
`,
		},
		"with tags": {
			Input: &struct {
				Int  int    `env:"COUNT" help:"number of people in a room"`
				Name string `env:"-" help:"ignore me"`
			}{
				Int:  10,
				Name: "Bob",
			},
			Expected: `#!/usr/bin/env sh

export COUNT=10 # number of people in a room
`,
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
			Expected: `#!/usr/bin/env sh

export TIME=2019-01-02T00:00:00Z # fmt: 2006-01-02T15:04:05Z07:00
export C_TIME=2019-01-02 # fmt: 2006-01-02
export WAIT_TIME=1h0m0s
`,
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
			Expected: `#!/usr/bin/env sh

export INT=1
export UINT=2
export FLOAT=3.4
export STRING=5
`,
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
			Expected: `#!/usr/bin/env sh

export INT=0
export UINT=0
export FLOAT=0
export STRING=
`,
		},
	}
	trial.New(fn, cases).Test(t)
}

func TestEncoder_Marshal_Prefix(t *testing.T) {
	fn := func(args ...interface{}) (interface{}, error) {
		b, err := NewEnvUnloader().WithPrefix("prefix").Unload(node.MakeAllNodes(node.Options{
			NoFollow: []string{"time.Time"},
		}, args[0]))
		return string(b), err
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
			Expected: `#!/usr/bin/env sh

export PREFIX_INT=1
export PREFIX_INT_8=2
export PREFIX_INT_16=3
export PREFIX_INT_32=4
export PREFIX_INT_64=5
`,
		},
	}
	trial.New(fn, cases).Test(t)
}
