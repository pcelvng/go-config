package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SpecialStruct struct{}
type SpecialInt int

func TestStructNodes(t *testing.T) {
	type DoubleEmbeddedStruct struct {
		String string
		Bool   bool
		Int    int
	}

	type EmbeddedStruct struct {
		String  string
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Float32 float32
		Float64 float64
		DE      DoubleEmbeddedStruct
	}

	type SimpleStruct struct {
		String  string `tagname:"tagvalue"`
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Float32 float32
		Float64 float64
		ES      EmbeddedStruct

		// pointers
		StringPtr  *string `tagname:"tagvalue"`
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
		Float32Ptr *float32
		Float64Ptr *float64
		ESPtr      *DoubleEmbeddedStruct

		// time stuff
		Time            time.Time
		TimePtr         *time.Time
		TimeDuration    time.Duration
		TimeDurationPtr *time.Duration

		// slice
		SliceInt    []int
		SlicePtrInt *[]int
		SliceIntPtr []*int
		SliceString []string
		SliceStruct []SpecialStruct // excluded because slice values are not basic.

		// ignore private
		myPrivateInt int

		// ignore types on ignore list
		IgnoreStruct SpecialStruct
		IgnoreInt    SpecialInt

		// ignored types - meaningless in a configuration context.
		Array      [3]int
		Func       func() error
		Chan       chan int
		Complex64  complex64
		Complex128 complex128
		Interface  interface{}
		Map        map[string]string
	}

	ss := &SimpleStruct{}
	nodes := StructNodes(ss, Options{
		NoFollow:    []string{"time.Time"},
		IgnoreTypes: []string{"node.SpecialStruct", "node.SpecialInt"},
	})

	// make sure correct number of nodes.
	assert.Equal(t, 56, len(nodes))

	// correct field names
	assert.Equal(t, "String", nodes["String"].FieldName())
	assert.Equal(t, "Bool", nodes["Bool"].FieldName())
	assert.Equal(t, "Int", nodes["Int"].FieldName())
	assert.Equal(t, "Int8", nodes["Int8"].FieldName())
	assert.Equal(t, "Int16", nodes["Int16"].FieldName())
	assert.Equal(t, "Int32", nodes["Int32"].FieldName())
	assert.Equal(t, "Int64", nodes["Int64"].FieldName())
	assert.Equal(t, "Uint", nodes["Uint"].FieldName())
	assert.Equal(t, "Uint8", nodes["Uint8"].FieldName())
	assert.Equal(t, "Uint16", nodes["Uint16"].FieldName())
	assert.Equal(t, "Uint32", nodes["Uint32"].FieldName())
	assert.Equal(t, "Float32", nodes["Float32"].FieldName())
	assert.Equal(t, "Float64", nodes["Float64"].FieldName())
	assert.Equal(t, "ES", nodes["ES"].FieldName())

	assert.Equal(t, "StringPtr", nodes["StringPtr"].FieldName())
	assert.Equal(t, "BoolPtr", nodes["BoolPtr"].FieldName())
	assert.Equal(t, "IntPtr", nodes["IntPtr"].FieldName())
	assert.Equal(t, "Int8Ptr", nodes["Int8Ptr"].FieldName())
	assert.Equal(t, "Int16Ptr", nodes["Int16Ptr"].FieldName())
	assert.Equal(t, "Int32Ptr", nodes["Int32Ptr"].FieldName())
	assert.Equal(t, "Int64Ptr", nodes["Int64Ptr"].FieldName())
	assert.Equal(t, "UintPtr", nodes["UintPtr"].FieldName())
	assert.Equal(t, "Uint8Ptr", nodes["Uint8Ptr"].FieldName())
	assert.Equal(t, "Uint16Ptr", nodes["Uint16Ptr"].FieldName())
	assert.Equal(t, "Uint32Ptr", nodes["Uint32Ptr"].FieldName())
	assert.Equal(t, "Float32Ptr", nodes["Float32Ptr"].FieldName())
	assert.Equal(t, "Float64Ptr", nodes["Float64Ptr"].FieldName())
	assert.Equal(t, "ESPtr", nodes["ESPtr"].FieldName())
	assert.Equal(t, "String", nodes["ESPtr.String"].FieldName())

	assert.Equal(t, "String", nodes["ES.String"].FieldName())
	assert.Equal(t, "Bool", nodes["ES.Bool"].FieldName())
	assert.Equal(t, "Int", nodes["ES.Int"].FieldName())
	assert.Equal(t, "Int8", nodes["ES.Int8"].FieldName())
	assert.Equal(t, "Int16", nodes["ES.Int16"].FieldName())
	assert.Equal(t, "Int32", nodes["ES.Int32"].FieldName())
	assert.Equal(t, "Int64", nodes["ES.Int64"].FieldName())
	assert.Equal(t, "Uint", nodes["ES.Uint"].FieldName())
	assert.Equal(t, "Uint8", nodes["ES.Uint8"].FieldName())
	assert.Equal(t, "Uint16", nodes["ES.Uint16"].FieldName())
	assert.Equal(t, "Uint32", nodes["ES.Uint32"].FieldName())
	assert.Equal(t, "Float32", nodes["ES.Float32"].FieldName())
	assert.Equal(t, "Float64", nodes["ES.Float64"].FieldName())

	assert.Equal(t, "DE", nodes["ES.DE"].FieldName())
	assert.Equal(t, "String", nodes["ES.DE.String"].FieldName())
	assert.Equal(t, "Bool", nodes["ES.DE.Bool"].FieldName())
	assert.Equal(t, "Int", nodes["ES.DE.Int"].FieldName())

	// correct full names
	assert.Equal(t, "String", nodes["String"].FullName())
	assert.Equal(t, "Bool", nodes["Bool"].FullName())
	assert.Equal(t, "Int", nodes["Int"].FullName())
	assert.Equal(t, "Int8", nodes["Int8"].FullName())
	assert.Equal(t, "Int16", nodes["Int16"].FullName())
	assert.Equal(t, "Int32", nodes["Int32"].FullName())
	assert.Equal(t, "Int64", nodes["Int64"].FullName())
	assert.Equal(t, "Uint", nodes["Uint"].FullName())
	assert.Equal(t, "Uint8", nodes["Uint8"].FullName())
	assert.Equal(t, "Uint16", nodes["Uint16"].FullName())
	assert.Equal(t, "Uint32", nodes["Uint32"].FullName())
	assert.Equal(t, "Float32", nodes["Float32"].FullName())
	assert.Equal(t, "Float64", nodes["Float64"].FullName())
	assert.Equal(t, "ES", nodes["ES"].FullName())

	assert.Equal(t, "StringPtr", nodes["StringPtr"].FullName())
	assert.Equal(t, "BoolPtr", nodes["BoolPtr"].FullName())
	assert.Equal(t, "IntPtr", nodes["IntPtr"].FullName())
	assert.Equal(t, "Int8Ptr", nodes["Int8Ptr"].FullName())
	assert.Equal(t, "Int16Ptr", nodes["Int16Ptr"].FullName())
	assert.Equal(t, "Int32Ptr", nodes["Int32Ptr"].FullName())
	assert.Equal(t, "Int64Ptr", nodes["Int64Ptr"].FullName())
	assert.Equal(t, "UintPtr", nodes["UintPtr"].FullName())
	assert.Equal(t, "Uint8Ptr", nodes["Uint8Ptr"].FullName())
	assert.Equal(t, "Uint16Ptr", nodes["Uint16Ptr"].FullName())
	assert.Equal(t, "Uint32Ptr", nodes["Uint32Ptr"].FullName())
	assert.Equal(t, "Float32Ptr", nodes["Float32Ptr"].FullName())
	assert.Equal(t, "Float64Ptr", nodes["Float64Ptr"].FullName())
	assert.Equal(t, "ESPtr", nodes["ESPtr"].FullName())
	assert.Equal(t, "ESPtr.String", nodes["ESPtr.String"].FullName())

	assert.Equal(t, "ES.String", nodes["ES.String"].FullName())
	assert.Equal(t, "ES.Bool", nodes["ES.Bool"].FullName())
	assert.Equal(t, "ES.Int", nodes["ES.Int"].FullName())
	assert.Equal(t, "ES.Int8", nodes["ES.Int8"].FullName())
	assert.Equal(t, "ES.Int16", nodes["ES.Int16"].FullName())
	assert.Equal(t, "ES.Int32", nodes["ES.Int32"].FullName())
	assert.Equal(t, "ES.Int64", nodes["ES.Int64"].FullName())
	assert.Equal(t, "ES.Uint", nodes["ES.Uint"].FullName())
	assert.Equal(t, "ES.Uint8", nodes["ES.Uint8"].FullName())
	assert.Equal(t, "ES.Uint16", nodes["ES.Uint16"].FullName())
	assert.Equal(t, "ES.Uint32", nodes["ES.Uint32"].FullName())
	assert.Equal(t, "ES.Float32", nodes["ES.Float32"].FullName())
	assert.Equal(t, "ES.Float64", nodes["ES.Float64"].FullName())

	assert.Equal(t, "ES.DE", nodes["ES.DE"].FullName())
	assert.Equal(t, "ES.DE.String", nodes["ES.DE.String"].FullName())
	assert.Equal(t, "ES.DE.Bool", nodes["ES.DE.Bool"].FullName())
	assert.Equal(t, "ES.DE.Int", nodes["ES.DE.Int"].FullName())

	// correct prefixes
	assert.Equal(t, "", nodes["String"].Prefix)
	assert.Equal(t, "", nodes["Bool"].Prefix)
	assert.Equal(t, "", nodes["Int"].Prefix)
	assert.Equal(t, "", nodes["Int8"].Prefix)
	assert.Equal(t, "", nodes["Int16"].Prefix)
	assert.Equal(t, "", nodes["Int32"].Prefix)
	assert.Equal(t, "", nodes["Int64"].Prefix)
	assert.Equal(t, "", nodes["Uint"].Prefix)
	assert.Equal(t, "", nodes["Uint8"].Prefix)
	assert.Equal(t, "", nodes["Uint16"].Prefix)
	assert.Equal(t, "", nodes["Uint32"].Prefix)
	assert.Equal(t, "", nodes["Float32"].Prefix)
	assert.Equal(t, "", nodes["Float64"].Prefix)
	assert.Equal(t, "", nodes["ES"].Prefix)

	assert.Equal(t, "", nodes["Float64Ptr"].Prefix)
	assert.Equal(t, "", nodes["ESPtr"].Prefix)
	assert.Equal(t, "ESPtr", nodes["ESPtr.String"].Prefix)

	assert.Equal(t, "ES", nodes["ES.String"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Bool"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Int"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Int8"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Int16"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Int32"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Int64"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Uint"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Uint8"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Uint16"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Uint32"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Float32"].Prefix)
	assert.Equal(t, "ES", nodes["ES.Float64"].Prefix)

	assert.Equal(t, "ES", nodes["ES.DE"].Prefix)
	assert.Equal(t, "ES.DE", nodes["ES.DE.String"].Prefix)
	assert.Equal(t, "ES.DE", nodes["ES.DE.Bool"].Prefix)
	assert.Equal(t, "ES.DE", nodes["ES.DE.Int"].Prefix)

	// make sure values can be set correctly.
	assert.Equal(t, nil, nodes["String"].SetFieldValue("hello"))
	assert.Equal(t, "hello", ss.String)
	assert.Equal(t, nil, nodes["Bool"].SetFieldValue("true"))
	assert.Equal(t, true, ss.Bool)
	assert.Equal(t, nil, nodes["Int"].SetFieldValue("-1"))
	assert.Equal(t, -1, ss.Int)
	assert.Equal(t, nil, nodes["Int8"].SetFieldValue("-8"))
	assert.Equal(t, int8(-8), ss.Int8)
	assert.Equal(t, nil, nodes["Int16"].SetFieldValue("-16"))
	assert.Equal(t, int16(-16), ss.Int16)
	assert.Equal(t, nil, nodes["Int32"].SetFieldValue("-32"))
	assert.Equal(t, int32(-32), ss.Int32)
	assert.Equal(t, nil, nodes["Int64"].SetFieldValue("-64"))
	assert.Equal(t, int64(-64), ss.Int64)
	assert.Equal(t, nil, nodes["Uint"].SetFieldValue("1"))
	assert.Equal(t, uint(1), ss.Uint)
	assert.Equal(t, nil, nodes["Uint8"].SetFieldValue("8"))
	assert.Equal(t, uint8(8), ss.Uint8)
	assert.Equal(t, nil, nodes["Uint16"].SetFieldValue("16"))
	assert.Equal(t, uint16(16), ss.Uint16)
	assert.Equal(t, nil, nodes["Uint32"].SetFieldValue("32"))
	assert.Equal(t, uint32(32), ss.Uint32)
	assert.Equal(t, nil, nodes["Float32"].SetFieldValue("32.32"))
	assert.Equal(t, float32(32.32), ss.Float32)
	assert.Equal(t, nil, nodes["Float64"].SetFieldValue("64.64"))
	assert.Equal(t, 64.64, ss.Float64)

	// Set slice.
	assert.Nil(t, nodes["SliceInt"].SetSlice([]string{"1", "2", "3"}))
	assert.Equal(t, []int{1, 2, 3}, ss.SliceInt)
	assert.Nil(t, nodes["SlicePtrInt"].SetSlice([]string{"1", "2", "3"}))
	assert.Equal(t, []int{1, 2, 3}, *ss.SlicePtrInt)
	assert.Nil(t, nodes["SliceIntPtr"].SetSlice([]string{"1", "2", "3"}))
	one := 1
	two := 2
	three := 3
	assert.Equal(t, []*int{&one, &two, &three}, ss.SliceIntPtr)
	assert.Nil(t, nodes["SliceString"].SetSlice([]string{"one", "2", "three"}))
	assert.Equal(t, []string{"one", "2", "three"}, ss.SliceString)
	assert.Panics(t, func() { nodes["String"].SetSlice([]string{"error"}) }) // panic - not a slice.

	// Set time.
	usedFmt, err := nodes["Time"].SetTime("2020-01-01T15:04:05Z", "") // default time: "2006-01-02T15:04:05Z07:00"
	assert.Equal(t, time.RFC3339, usedFmt)
	assert.Nil(t, err)
	expectedTime, _ := time.Parse(time.RFC3339, "2020-01-01T15:04:05Z")
	assert.Equal(t, expectedTime, ss.Time)

	usedFmt, err = nodes["TimePtr"].SetTime("2020-01-01T15:04:05Z", "") // default time: "2006-01-02T15:04:05Z07:00"
	assert.Equal(t, time.RFC3339, usedFmt)
	assert.Nil(t, err)
	expectedTime, _ = time.Parse(time.RFC3339, "2020-01-01T15:04:05Z")
	assert.Equal(t, expectedTime, *ss.TimePtr)

	// Set duration.
	assert.Nil(t, nodes["TimeDuration"].SetFieldValue("3s"))
	assert.Equal(t, time.Second*3, ss.TimeDuration)

	// Check tag is set
	assert.Equal(t, "tagvalue", nodes["String"].Tag().Get("tagname"))

	// Check time.Time special case is assessed correctly.
	assert.True(t, nodes["Time"].IsTime())
	assert.True(t, nodes["TimePtr"].IsTime())
	assert.False(t, nodes["String"].IsTime())

	// Check struct nodes are accurately described as such.
	assert.True(t, nodes["Time"].IsStruct())
	assert.True(t, nodes["TimePtr"].IsStruct())
	assert.True(t, nodes["ES"].IsStruct())
	assert.False(t, nodes["String"].IsStruct())

	// Check that time.Duration is accurately described as such.
	assert.True(t, nodes["TimeDuration"].IsDuration())
	assert.False(t, nodes["Int64"].IsDuration())

	// Check that slices are accurately described as such (only slices of string, bool, ints are included)
	assert.True(t, nodes["SliceInt"].IsSlice())
	assert.False(t, nodes["String"].IsSlice())

	// Check that string values come back correctly.
	assert.Equal(t, "hello", nodes["String"].String())
	assert.Equal(t, "true", nodes["Bool"].String())
	assert.Equal(t, "-1", nodes["Int"].String())
	assert.Equal(t, "-8", nodes["Int8"].String())
	assert.Equal(t, "-16", nodes["Int16"].String())
	assert.Equal(t, "-32", nodes["Int32"].String())
	assert.Equal(t, "-64", nodes["Int64"].String())
	assert.Equal(t, "1", nodes["Uint"].String())
	assert.Equal(t, "8", nodes["Uint8"].String())
	assert.Equal(t, "16", nodes["Uint16"].String())
	assert.Equal(t, "32", nodes["Uint32"].String())
	assert.Equal(t, "32.32", nodes["Float32"].String())
	assert.Equal(t, "64.64", nodes["Float64"].String())
	assert.Equal(t, "3s", nodes["TimeDuration"].String()) // time.Duration string special case.

	// Slice strings.
	assert.Equal(t, []string{"1", "2", "3"}, nodes["SliceInt"].SliceString())
	assert.Equal(t, []string{"1", "2", "3"}, nodes["SlicePtrInt"].SliceString())
	assert.Equal(t, []string{"1", "2", "3"}, nodes["SliceIntPtr"].SliceString())
	assert.Equal(t, []string{"one", "2", "three"}, nodes["SliceString"].SliceString())
	assert.Panics(t, func() { nodes["String"].SliceString() }) // panic - not a slice.

	// Time strings.
	assert.Equal(t, "2020-01-01T15:04:05Z", nodes["Time"].TimeString(""))    // default time: "2006-01-02T15:04:05Z07:00"
	assert.Equal(t, "2020-01-01T15:04:05Z", nodes["TimePtr"].TimeString("")) // default time: "2006-01-02T15:04:05Z07:00")
}
