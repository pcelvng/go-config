package env

import (
	"os"
	"testing"
	"time"

	"github.com/jbsmith7741/go-tools/appenderr"
	"github.com/hydronica/trial"
)

func TestDecoder_Unmarshal(t *testing.T) {
	type Aint int
	type Astring string
	type tConfig struct {
		Dura   time.Duration
		Time   time.Time `format:"2006-01-02"`
		Bool   bool
		String string

		Int   int
		Int8  int8  `env:"INT8"`
		Int16 int16 `env:"INT16"`
		Int32 int32 `env:"INT32"`
		Int64 int64 `env:"INT64"`

		Uint   uint
		Uint8  uint8  `env:"UINT8"`
		Uint16 uint16 `env:"UINT16"`
		Uint32 uint32 `env:"UINT32"`
		Uint64 uint64 `env:"UINT64"`

		Float32 float32 `env:"FLOAT32"`
		Float64 float64 `env:"FLOAT64"`

		IntP    *int     `env:"INTP"`
		UintP   *uint    `env:"UINTP"`
		StringP *string  `env:"STRINGP"`
		FloatP  *float64 `env:"FLOATP"`

		// slices
		ArrayField       [3]int
		SliceStringField []string
		SliceIntField    []int
		SliceFloatField  []float64

		// ignored fields
		IgnoreMe         string `env:"-"`
		privateField     string // should not be populated
		privateFieldWTag string `env:"private_field_w_tag"` // Will not populate private field even with tag.

		//alias'
		Aint    Aint
		Astring Astring
	}
	type tStruct struct {
		MStruct mStruct  `env:"MSTRUCT"`
		PStruct *mStruct `env:"PSTRUCT"`
	}
	type input struct {
		config interface{}
		args   map[string]string
	}
	type dbCreds struct {
		Username string `env:"UN"`
		Password string `env:"PW"`
	}
	fn := func(in input) (interface{}, error) {
		os.Clearenv()
		errs := appenderr.New()
		if in.config == nil {
			in.config = &tConfig{}
		}
		for key, value := range in.args {
			errs.Add(os.Setenv(key, value))
		}

		d := New()
		errs.Add(d.Unmarshal(in.config))
		return in.config, errs.ErrOrNil()
	}
	cases := trial.Cases[input, interface{}]{
		"default": {
			Input:    input{args: map[string]string{}},
			Expected: &tConfig{},
		},
		"time.duration": {
			Input:    input{args: map[string]string{"DURA": "10s"}},
			Expected: &tConfig{Dura: 10 * time.Second},
		},
		"time.duration (int)": {
			Input:    input{args: map[string]string{"DURA": "1000"}},
			Expected: &tConfig{Dura: 1000},
		},
		"time.Time": {
			Input:    input{args: map[string]string{"TIME": "2010-01-02"}},
			Expected: &tConfig{Time: trial.TimeDay("2010-01-02")},
		},
		"int": {
			Input:    input{args: map[string]string{"INT": "10", "INT8": "8", "INT16": "16", "INT32": "32", "INT64": "64"}},
			Expected: &tConfig{Int: 10, Int8: 8, Int16: 16, Int32: 32, Int64: 64},
		},
		"uint": {
			Input:    input{args: map[string]string{"UINT": "10", "UINT8": "8", "UINT16": "16", "UINT32": "32", "UINT64": "64"}},
			Expected: &tConfig{Uint: 10, Uint8: 8, Uint16: 16, Uint32: 32, Uint64: 64},
		},
		"float": {
			Input:    input{args: map[string]string{"FLOAT32": "3.2", "FLOAT64": "6.4"}},
			Expected: &tConfig{Float32: 3.2, Float64: 6.4},
		},
		"bool=true": {
			Input:    input{args: map[string]string{"BOOL": "true"}},
			Expected: &tConfig{Bool: true},
		},
		"bool=false": {
			Input:    input{args: map[string]string{"BOOL": "false"}},
			Expected: &tConfig{Bool: false},
		},
		"bool (default)": {
			Input:    input{args: map[string]string{}},
			Expected: &tConfig{Bool: false},
		},
		"string": {
			Input:    input{args: map[string]string{"STRING": "hello"}},
			Expected: &tConfig{String: "hello"},
		},
		"array/slice": {
			Input: input{
				args: map[string]string{
					"SLICE_STRING_FIELD": "part1,part2",
					"ARRAY_FIELD":        "1,2,3",
					"SLICE_INT_FIELD":    "1,2,3",
					"SLICE_FLOAT_FIELD":  "1.1,2.2,3.3",
				},
			},
			Expected: &tConfig{
				ArrayField:       [3]int{1, 2, 3},
				SliceStringField: []string{"part1", "part2"},
				SliceIntField:    []int{1, 2, 3},
				SliceFloatField:  []float64{1.1, 2.2, 3.3},
			},
		},
		"slice with quotes\"": {
			Input: input{
				args: map[string]string{"SLICE_INT_FIELD": `"1","2","3"`},
			},
			Expected: &tConfig{
				SliceIntField: []int{1, 2, 3},
			},
		},
		"slice with quotes'": {
			Input: input{
				args: map[string]string{"SLICE_INT_FIELD": `'1','2','3'`},
			},
			Expected: &tConfig{
				SliceIntField: []int{1, 2, 3},
			},
		},
		"slice with brackets[]": {
			Input: input{
				args: map[string]string{"SLICE_INT_FIELD": `[1,2,3]`},
			},
			Expected: &tConfig{
				SliceIntField: []int{1, 2, 3},
			},
		},
		"ignored": {
			Input: input{
				args: map[string]string{
					"IGNORE_ME":           "WHAT?",
					"PRIVATE_FIELD":       "hello",
					"private_field_w_tag": "vprivate_field_w_tag",
					"PRIVATE_FIELD_W_TAG": "vPRIVATE_FIELD_W_TAG", // just checking this variation in case a logic slip.
				},
			},
			Expected: &tConfig{},
		},
		"alias": {
			Input: input{args: map[string]string{"AINT": "10", "ASTRING": "abc", "NUMBER": "two"}},
			Expected: &tConfig{
				Aint:    10,
				Astring: "abc",
			},
		},
		"pointers": {
			Input:    input{args: map[string]string{"INTP": "3", "UINTP": "4", "FLOATP": "5.6"}},
			Expected: &tConfig{IntP: trial.IntP(3), UintP: trial.UintP(4), FloatP: trial.Float64P(5.6)},
		},
		"struct": {
			Input: input{
				config: &tStruct{},
				args:   map[string]string{"MSTRUCT": "abc", "PSTRUCT": "def"},
			},
			Expected: &tStruct{MStruct: mStruct{"abc"}, PStruct: &mStruct{"def"}},
		},

		// Nested struct regression: blank struct tag inherits parent prefix;
		// non-blank tag appends segment; "NAME,omitprefix" opts leaf out of prefix;
		// struct "omitprefix" resets subtree; marshaler+explicit tag is a leaf.
		"nested: flat blank struct tag": {
			Input: input{
				config: &struct {
					Host string
					DB   dbCreds
				}{},
				args: map[string]string{
					"UN": "u", "PW": "p", "HOST": "h",
				},
			},
			Expected: &struct {
				Host string
				DB   dbCreds
			}{
				Host: "h",
				DB:   dbCreds{Username: "u", Password: "p"},
			},
		},
		"nested: prefixed struct tag": {
			Input: input{
				config: &struct {
					Host string
					DB   dbCreds `env:"DB"`
				}{},
				args: map[string]string{
					"DB_UN": "u", "DB_PW": "p",
				},
			},
			Expected: &struct {
				Host string
				DB   dbCreds `env:"DB"`
			}{
				DB: dbCreds{Username: "u", Password: "p"},
			},
		},
		"nested: anonymous embed": {
			Input: input{
				config: &struct {
					dbCreds
					Host string
				}{},
				args: map[string]string{
					"UN": "u", "PW": "p", "HOST": "h",
				},
			},
			Expected: &struct {
				dbCreds
				Host string
			}{
				dbCreds: dbCreds{Username: "u", Password: "p"},
				Host:    "h",
			},
		},
		"nested: blank inner under parent prefix": {
			Input: input{
				config: &struct {
					Config struct {
						Creds dbCreds
					} `env:"APP"`
				}{},
				args: map[string]string{"APP_UN": "u"},
			},
			Expected: &struct {
				Config struct {
					Creds dbCreds
				} `env:"APP"`
			}{
				Config: struct {
					Creds dbCreds
				}{Creds: dbCreds{Username: "u"}},
			},
		},
		"nested: leaf omitprefix": {
			Input: input{
				config: &struct {
					Config struct {
						Creds dbCreds
						Host  string `env:"HOST,omitprefix"`
					} `env:"APP"`
				}{},
				args: map[string]string{"HOST": "h"},
			},
			Expected: &struct {
				Config struct {
					Creds dbCreds
					Host  string `env:"HOST,omitprefix"`
				} `env:"APP"`
			}{
				Config: struct {
					Creds dbCreds
					Host  string `env:"HOST,omitprefix"`
				}{Host: "h"},
			},
		},
		"nested: struct omitprefix resets subtree": {
			Input: input{
				config: &struct {
					DB struct {
						Creds dbCreds
					} `env:"APP"`
					Legacy struct {
						Host string
					} `env:"omitprefix"`
				}{},
				args: map[string]string{"HOST": "h"},
			},
			Expected: &struct {
				DB struct {
					Creds dbCreds
				} `env:"APP"`
				Legacy struct {
					Host string
				} `env:"omitprefix"`
			}{
				Legacy: struct {
					Host string
				}{Host: "h"},
			},
		},
		"nested: marshaler leaf prefixed": {
			Input: input{
				config: &struct {
					Block struct {
						Meta mStruct `env:"META"`
					} `env:"APP"`
				}{},
				args: map[string]string{"APP_META": "abc"},
			},
			Expected: &struct {
				Block struct {
					Meta mStruct `env:"META"`
				} `env:"APP"`
			}{
				Block: struct {
					Meta mStruct `env:"META"`
				}{Meta: mStruct{"abc"}},
			},
		},
		"nested: marshaler leaf omitprefix": {
			Input: input{
				config: &struct {
					Block struct {
						Meta mStruct `env:"META,omitprefix"`
					} `env:"APP"`
				}{},
				args: map[string]string{"META": "abc"},
			},
			Expected: &struct {
				Block struct {
					Meta mStruct `env:"META,omitprefix"`
				} `env:"APP"`
			}{
				Block: struct {
					Meta mStruct `env:"META,omitprefix"`
				}{Meta: mStruct{"abc"}},
			},
		},
		"nested: pointer struct": {
			Input: input{
				config: &struct {
					DB *dbCreds `env:"DB"`
				}{},
				args: map[string]string{"DB_UN": "u"},
			},
			Expected: &struct {
				DB *dbCreds `env:"DB"`
			}{
				DB: &dbCreds{Username: "u"},
			},
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
	}
	trial.New(fn, cases).SubTest(t)
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
