package env

import (
	"os"
	"testing"
	"time"

	"github.com/pcelvng/go-config/util/node"

	"github.com/stretchr/testify/assert"
)

func TestDecoder_Unmarshal2(t *testing.T) {
	// Setup
	envMap := map[string]string{
		"STRING":    "vSTRING",
		"BOOL":      "true",
		"INT":       "-1",
		"INT_8":     "-8",
		"INT_16":    "-16",
		"INT_32":    "-32",
		"INT_64":    "-64",
		"UINT":      "1",
		"UINT_8":    "8",
		"UINT_16":   "16",
		"UINT_32":   "32",
		"UINT_64":   "64",
		"FLOAT_32":  "32",
		"FLOAT_64":  "64",
		"EO_STRING": "vEO_STRING",

		// pointers
		"STRING_PTR":           "vSTRING",
		"BOOL_PTR":             "true",
		"INT_PTR":              "-1",
		"INT_8_PTR":            "-8",
		"INT_16_PTR":           "-16",
		"INT_32_PTR":           "-32",
		"INT_64_PTR":           "-64",
		"UINT_PTR":             "1",
		"UINT_8_PTR":           "8",
		"UINT_16_PTR":          "16",
		"UINT_32_PTR":          "32",
		"UINT_64_PTR":          "64",
		"FLOAT_32_PTR":         "32",
		"FLOAT_64_PTR":         "64",
		"EO_PTR_STRING":        "vEO_PTR_STRING",
		"NEW_EO_PREFIX_STRING": "vNEW_EO_PREFIX_STRING",
		"DEO_EO_STRING":        "vDEO_EO_STRING",
		"DEO_NO_PREFIX_STRING": "vDEO_NO_PREFIX_STRING",

		// time stuff
		"TIME":     "2020-01-01T15:04:05Z",
		"TIME_PTR": "2020-01-01T15:04:05Z",
		"TIME_FMT": "03 Feb 20 13:12 MST", // RFC822 "02 Jan 06 15:04 MST"
		"DURATION": "12s",

		// slices
		"STRING_SLICE":           "one,two,three",         // default sep is ","
		"STRING_SLICE_SEP":       "one;two;three",         // custom sep
		"STRING_SLICE_SPACES":    "one, two,three\n",      // with spaces
		"STRING_SLICE_BRACKETS":  "[one,two,three]",       // with square brackets
		"STRING_SLICE_QUOTES":    `"one",'two','"three"'`, // quotes
		"STRING_SLICE_AS_QUOTED": `"one",'two','"three'"`, // quotes remain without ",string" option
		"INT_SLICE":              "1,2,3",
		"UINT_SLICE":             "1,2,3",
		"FLOAT_SLICE":            "1,2,3",

		"STRING_OVERRIDE":  "vSTRING_OVERRIDE",
		"CONFIG_IGNORE":    "should-not-be-read-in",
		"NO_PREFIX_STRING": "vNO_PREFIX_STRING",
	}

	for k, v := range envMap {
		os.Setenv(k, v)
	}

	type EmbeddedOptions struct {
		String string
	}

	type DoubleEmbeddedOptions struct {
		EO         EmbeddedOptions
		EONoPrefix EmbeddedOptions `env:"omitprefix"`
	}

	type NoPrefixOptions struct {
		NoPrefixString string
	}

	type MainOptions struct {
		String      string
		Bool        bool
		Int         int
		Int8        int8
		Int16       int16
		Int32       int32
		Int64       int64
		Uint        uint
		Uint8       uint8
		Uint16      uint16
		Uint32      uint32
		Uint64      uint64
		Float32     float32
		Float64     float64
		EO          EmbeddedOptions
		EOPrefix    EmbeddedOptions `env:"NEW_EO_PREFIX"`
		DEO         DoubleEmbeddedOptions
		DeoNoPrefix DoubleEmbeddedOptions

		// pointers
		StringPtr  *string
		BoolPtr    *bool
		IntPtr     *int
		Int8Ptr    *int8
		Int16Ptr   *int16
		Int32Ptr   *int32
		Int64Ptr   *int64
		UintPtr    *uint
		Uint8Ptr   *uint8
		Uint16Ptr  *uint16
		Uint32Ptr  *uint32
		Uint64Ptr  *uint64
		Float32Ptr *float32
		Float64Ptr *float64
		EOPtr      *EmbeddedOptions

		// time stuff
		Time     time.Time  // default fmt is time.RFC3339
		TimePtr  *time.Time // default fmt is time.RFC3339
		TimeFmt  time.Time  `fmt:"RFC822"`
		Duration time.Duration

		// slices
		StringSlice         []string // default separator is ","
		StringSliceSep      []string `sep:";"`
		StringSliceSpaces   []string
		StringSliceBrackets []string
		StringSliceQuotes   []string `env:",string"`
		StringSliceAsQuoted []string
		IntSlice            []int
		UintSlice           []uint
		FloatSlice          []float32

		// env name override.
		StringOverride string `env:"STRING_OVERRIDE"`

		// Other
		ConfigIgnore string          `config:"ignore"`
		OmitPrefix   NoPrefixOptions `env:"omitprefix"`
		IgnoreEnv    string          `env:"-"`
	}

	options := &MainOptions{}
	nss := node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, options)
	err := Load([]byte{}, nss)
	assert.Nil(t, err)

	// basic types
	assert.Equal(t, envMap["STRING"], options.String)
	assert.Equal(t, true, options.Bool)
	assert.Equal(t, -1, options.Int)
	assert.Equal(t, int8(-8), options.Int8)
	assert.Equal(t, int16(-16), options.Int16)
	assert.Equal(t, int32(-32), options.Int32)
	assert.Equal(t, int64(-64), options.Int64)
	assert.Equal(t, uint(1), options.Uint)
	assert.Equal(t, uint8(8), options.Uint8)
	assert.Equal(t, uint16(16), options.Uint16)
	assert.Equal(t, uint32(32), options.Uint32)
	assert.Equal(t, uint64(64), options.Uint64)
	assert.Equal(t, float32(32), options.Float32)
	assert.Equal(t, float64(64), options.Float64)
	assert.Equal(t, envMap["EO_STRING"], options.EO.String)
	assert.Equal(t, envMap["NEW_EO_PREFIX_STRING"], options.EOPrefix.String)
	assert.Equal(t, envMap["DEO_EO_STRING"], options.DEO.EO.String)
	assert.Equal(t, envMap["DEO_NO_PREFIX_STRING"], options.DeoNoPrefix.EONoPrefix.String)

	// pointers
	assert.Equal(t, envMap["STRING_PTR"], *options.StringPtr)
	assert.Equal(t, true, *options.BoolPtr)
	assert.Equal(t, -1, *options.IntPtr)
	assert.Equal(t, int8(-8), *options.Int8Ptr)
	assert.Equal(t, int16(-16), *options.Int16Ptr)
	assert.Equal(t, int32(-32), *options.Int32Ptr)
	assert.Equal(t, int64(-64), *options.Int64Ptr)
	assert.Equal(t, uint(1), *options.UintPtr)
	assert.Equal(t, uint8(8), *options.Uint8Ptr)
	assert.Equal(t, uint16(16), *options.Uint16Ptr)
	assert.Equal(t, uint32(32), *options.Uint32Ptr)
	assert.Equal(t, uint64(64), *options.Uint64Ptr)
	assert.Equal(t, float32(32), *options.Float32Ptr)
	assert.Equal(t, float64(64), *options.Float64Ptr)
	assert.Equal(t, envMap["EO_PTR_STRING"], options.EOPtr.String)

	// time stuff
	expectedTime, _ := time.Parse(time.RFC3339, envMap["TIME"])
	assert.Equal(t, expectedTime, options.Time)
	expectedTime, _ = time.Parse(time.RFC3339, envMap["TIME_PTR"])
	assert.Equal(t, expectedTime, *options.TimePtr)
	expectedTime, _ = time.Parse(time.RFC822, envMap["TIME_FMT"])
	assert.Equal(t, expectedTime, options.TimeFmt)
	assert.Equal(t, time.Second*12, options.Duration)

	// slices
	assert.Equal(t, []string{"one", "two", "three"}, options.StringSlice)
	assert.Equal(t, []string{"one", "two", "three"}, options.StringSliceSep)
	assert.Equal(t, []string{"one", "two", "three"}, options.StringSliceSpaces)
	assert.Equal(t, []string{"one", "two", "three"}, options.StringSliceBrackets)
	assert.Equal(t, []string{"one", "two", "three"}, options.StringSliceQuotes)
	assert.Equal(t, []string{`"one"`, `'two'`, `'"three'"`}, options.StringSliceAsQuoted)
	assert.Equal(t, []int{1, 2, 3}, options.IntSlice)
	assert.Equal(t, []uint{1, 2, 3}, options.UintSlice)
	assert.Equal(t, []float32{1, 2, 3}, options.FloatSlice)

	assert.Equal(t, "vSTRING_OVERRIDE", options.StringOverride)
	assert.Equal(t, "", options.ConfigIgnore)
	assert.Equal(t, envMap["NO_PREFIX_STRING"], options.OmitPrefix.NoPrefixString)

	for k := range envMap {
		os.Unsetenv(k)
	}

	// 'omitprefix' should return error when placed on
	// a non-struct field (time.Time) is ok.
	type OmitprefixStruct struct {
		Omitprefix string `env:"omitprefix"`
	}
	err = Load([]byte{}, node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, &OmitprefixStruct{}))
	assert.EqualError(t, err, "'omitprefix' cannot be used on non-struct field types")

	type OmitprefixTimeStruct struct {
		OmitprefixTime time.Time `env:"omitprefix"`
	}
	err = Load([]byte{}, node.MakeAllNodes(node.Options{
		NoFollow: []string{"time.Time"},
	}, &OmitprefixTimeStruct{}))
	assert.EqualError(t, err, "'omitprefix' cannot be used on non-struct field types")
}
