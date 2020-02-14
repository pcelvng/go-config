package node

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pcelvng/go-config/util"
)

// Node is an abstraction of a struct field.
//
// It contains a reference to the actual field value among other useful
// information.
type Node struct {
	Prefix string // The prefix name representing the parent(s).

	// Actual struct field value. Can be used to get/set the currently set value.
	// Note that "Value" will never get stored as a pointer but the actual value
	// the struct field pointer points to.
	FieldValue reflect.Value
	Field      reflect.StructField

	// Index is the field index in the struct. AKA the field's "order" relative to other
	// fields in the struct. Note that since private fields are skipped the Index value
	// can also skip. For example you may end up with fields in the same struct with
	// Index values {1,2,4,5} because '3' is a private member.
	Index int

	// tag holds tag overrides set at runtime.
	// Go does not allow struct tags to be modified at runtime.
	// Setting a tag value will either set a struct tag value that
	// didn't exist before or override an existing tag value.
	// The "raw" struct tag values can still be accessed by
	// accessing "Field.Tag".
	tag map[string]string

	// Meta provides allowance for pre or post processing meta data
	// for sharing information such as a resolved variable name.
	Meta map[string]string
}

// FieldName is offered for convenience in getting the
// string value of the struct field name. Same as calling "Field.Name".
func (n *Node) FieldName() string {
	return n.Field.Name
}

// FullName returns "Prefix.FieldName". If no prefix exists
// then "FieldName" is the full name.
func (n *Node) FullName() string {
	if n.Prefix == "" {
		return n.FieldName()
	}

	return n.Prefix + "." + n.FieldName()
}

// ParentName returns the full name of the field parent if one exists.
func (n *Node) ParentName() string {
	return n.Prefix
}

// ParentsNames returns a list of all parents by full name in
// order from most to least distant relative.
//
// Returns an empty slice if there is no lineage.
func (n *Node) ParentsNames() []string {
	names := make([]string, 0)
	if n.ParentName() == "" {
		return names
	}

	lineage := strings.Split(n.ParentName(), ".")

	// First item is the most distant ancestor. Last item is the direct parent.
	for i := 0; i < len(lineage); i++ {
		names = append(names, strings.Join(lineage[:i+1], "."))
	}

	return names
}

// ValueType is the string value of field.Type().String()
//
// For reference:
// time.Time == "time.Time"
// time.Duration == "time.Duration"
func (n *Node) ValueType() string {
	return n.FieldValue.Type().String()
}

// Kind is for convenience instead
// of calling n.FieldValue.Kind().
func (n *Node) Kind() reflect.Kind {
	return n.FieldValue.Kind()
}

// GetTag has the same behavior as "reflect.StructTag.Get"
// but checks first if the value exists as a runtime override first.
func (n *Node) GetTag(key string) string {
	if v, ok := n.tag[key]; ok {
		return v
	}

	return n.Field.Tag.Get(key)
}

func (n *Node) SetTag(key, value string) {
	if value == "" || key == "" {
		return
	}

	n.tag[key] = value
}

// GetBoolTag behaves like GetTag except the value is
// parsed as a bool value and returned.
// If the value doesn't exist then false is returned.
// If the value does not parse then false is returned.
func (n *Node) GetBoolTag(key string) bool {
	bv, _ := strconv.ParseBool(n.GetTag(key))
	return bv
}

// SetBoolTag behaves like SetTag only the tag value is correctly set as a
// string parsable bool.
func (n *Node) SetBoolTag(key string, value bool) {
	n.SetTag(key, strconv.FormatBool(value))
}

// IsStruct is called to determine if the node represents a struct.
// Struct node values cannot be "Get"ed or "Set"ed. They are mostly useful for
// their field tag information.
//
// If the value is a struct:
//
// Struct node values are typically not set
// unless it's a specially handled struct type such
// as time.Time.
//
// Usually struct values are not set directly
// Unless it's a special struct like time.Time
// In which case the time.Time struct fields are not
// recursed but is treated as a special case.
func (n *Node) IsStruct() bool {
	return n.Kind() == reflect.Struct
}

func (n *Node) IsDuration() bool {
	return n.ValueType() == "time.Duration"
}

// IsTime returns true when the node represents a time.Time
// struct.
func (n *Node) IsTime() bool {
	return n.ValueType() == "time.Time"
}

func (n *Node) IsSlice() bool {
	return n.Kind() == reflect.Slice
}

// fieldString converts basic types to a string representation
// of the value.
//
// As a special case "time.Duration" support is baked in because it's
// a common case in application configuration.
func fieldString(value reflect.Value) string {
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Support case of int64 as a time.Duration.
		if value.Type().String() == "time.Duration" {
			return value.Interface().(time.Duration).String()
		}

		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		// Doing it this way is a little janky
		// but it seems like the simplest way given the default
		// behavior of converting float to string in go.
		f := fmt.Sprintf("%f", value.Float()) // likes to add trailing "0"s
		f = strings.TrimRight(f, "0")
		return strings.TrimRight(f, ".") // In case everything to the right was "0"s.
	default:
		panic(fmt.Sprintf("unexpected king to convert to string '%v'", value.Kind().String()))
	}

	return ""
}

// String fulfills the "Stringer" interface
// and returns the string representation of the value.
//
// The string representation is such that the value could then be set
// again with one of the appropriate "Set" methods.
func (n *Node) String() string {
	return fieldString(n.FieldValue)
}

// SliceString performs the same action as String but for slices, returning
// a slice of strings in such a format that the result can be fed back
// into the "SetSlice" method to obtain the same initial result.
//
// SliceString panics if the underlying field value type is not a slice
// or the underlying slice type is not on the basic type list.
func (n *Node) SliceString() []string {
	// Must be slice.
	if !n.IsSlice() {
		panic(fmt.Sprintf("field value '%v' must be a slice and instead was '%v'",
			n.FullName(),
			n.FieldValue.Kind().String()),
		)
	}

	items := make([]string, 0)
	for i := 0; i < n.FieldValue.Len(); i++ {
		itemValue := n.FieldValue.Index(i)

		// The slice value may be a pointer to a basic type.
		if itemValue.Kind() == reflect.Ptr {
			itemValue = itemValue.Elem()
		}

		items = append(items, fieldString(itemValue))
	}

	return items
}

// TimeString is a special case method for handling "time.Time" field types.
//
// It's a separate method because it needs the timeFmt format information
// in order to know the string format.
//
// TimeString panics if the value type is not "time.Time".
//
// timeFmt can be an exact go time format definition or the format name
// as it appears in the common time package formats. For example if the value
// of timeFmt == "RFC3339" then time.RFC3339 format will be used.
func (n *Node) TimeString(timeFmt string) string {
	// Must be "time.Time".
	if !n.IsTime() {
		panic(fmt.Sprintf("field '%v' must be a 'time.Time' value type but is '%v' instead",
			n.FullName(),
			n.ValueType(),
		))
	}

	timeFmt = NormTimeFormat(timeFmt)
	return n.FieldValue.Interface().(time.Time).Format(timeFmt)
}

// SetValue attempts to convert the field value "fv" represented
// as a string and assign it to the node value. An error is returned
// if the string cannot be converted to the underlying go type.
//
// Panics if the node is a pointer, slice or struct.
func (n *Node) SetFieldValue(s string) error {
	// Panics if called on pointer, slice or struct.
	switch n.Kind() {
	case reflect.Ptr:
		panic(fmt.Sprintf("node '%s' type is a pointer", n.FullName()))
	case reflect.Slice:
		// Should call "SetSlice" to handle slices.
		panic(fmt.Sprintf("node '%s' type is a slice - call SetSlice method instead", n.FullName()))
	case reflect.Struct:
		// Should call "SetStruct" to handle structs.
		panic(fmt.Sprintf("node '%s' type is a struct - call SetStruct method instead", n.FullName()))
	}

	return setField(n.FieldValue, s)
}

// SetSlice attempts to convert slice values "vals" to the underlying field
// slice primitive type and set the resulting slice as the field value.
//
// SetSlice is a different method than SetFieldValue because it requires
// a slice of strings instead of just a string. Even if it accepted a single stream
// representing a list of items it would also need to know the item separator or
// other information about separation such as if the starting and terminating "[]"
// should be removed.
func (n *Node) SetSlice(vals []string) error {
	// Must be slice.
	if !n.IsSlice() {
		panic(fmt.Sprintf("field value '%v' must be a slice and instead was '%v'",
			n.FullName(),
			n.FieldValue.Kind().String()),
		)
	}

	fValue := n.FieldValue

	// create a slice and recursively assign the elements
	baseType := reflect.TypeOf(fValue.Interface()).Elem()

	slice := reflect.MakeSlice(fValue.Type(), 0, len(vals))
	for _, v := range vals {
		// Each item must be the correct type.
		baseValue := reflect.New(baseType).Elem()
		err := setField(baseValue, v)
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, baseValue)
	}

	fValue.Set(slice)

	return nil
}

// setField converts the string s to the type of value and sets the value if possible.
// Pointers and slices are recursively dealt with by following the pointer
// or creating a generic slice of type value.
func setField(value reflect.Value, s string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(s)
	case reflect.Bool:
		switch strings.ToLower(s) {
		case "true":
			value.SetBool(true)
		case "false", "":
			value.SetBool(false)
		default:
			// the bool value should be explicit to tell user
			// something is amiss.
			return fmt.Errorf("cannot assign '%v' to bool type", s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// check for time.Duration int64 special case.
		//
		// TODO: check if this still works when time package is vendored or there is a way to fake this.
		if value.Type().String() == "time.Duration" {
			d, err := time.ParseDuration(s)
			if err != nil {
				return err
			}

			value.SetInt(int64(d))
			return nil
		}

		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return err
		}

		value.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 0)
		if err != nil {
			return err
		}
		value.SetFloat(f)
	case reflect.Ptr:
		// Create non-pointer type and recursively assign.
		z := reflect.New(value.Type().Elem())
		err := setField(z.Elem(), s)
		if err != nil {
			return err
		}

		value.Set(z)
	default:
		return fmt.Errorf("unsupported type '%v'", value.Kind())
	}

	return nil
}

// SetStruct will attempt to assign "v" as the underlying node field value.
// panics if "v" is not a struct.
func (n *Node) SetStruct(v interface{}) {
	if n.FieldValue.Kind() != reflect.Struct {
		panic(
			fmt.Sprintf("attempting to assign struct to '%s' on non-struct type '%s'",
				n.FullName(), n.ValueType()),
		)
	}
	n.FieldValue.Set(reflect.ValueOf(v))
}

var timeFormats = map[string]string{
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
}

// NormTimeFormat accepts an actual time format
// or a shorthand version of all the common go time package formats
// using the same name as the variable time in accordance to the "timeFormats"
// map defined in this module.
//
// If timeFmt is empty then time.RFC3339 is returned.
func NormTimeFormat(timeFmt string) string {
	if timeFmt == "" {
		return time.RFC3339
	}

	mappedFmt, ok := timeFormats[timeFmt]
	if ok {
		return mappedFmt
	}

	return timeFmt
}

// SetTime attempts to set a time.Time value from a string. Optionally
// a go format can be provided. "time.Time" types are a special explicitly supported
// struct case.
//
// If node is not a time.Time type then an error is returned.
// Returns the timeFmt applied in converting the tv string to a time.Time
// value.
//
// Default timeFmt format is time.RFC3339.
func (n *Node) SetTime(tv, timeFmt string) (usedFmt string, err error) {
	timeFmt = NormTimeFormat(timeFmt)

	if !n.IsTime() {
		return timeFmt, errors.New("cannot set value because it is not of type time.Time")
	}

	t, err := time.Parse(timeFmt, tv)
	if err != nil {
		return timeFmt, err
	}

	n.SetStruct(t)
	return timeFmt, nil
}

// GetStructNodes will generate a map of nodes
// representing the fields in the struct including
// all child struct fields recursively.
//
// The map key is a dot separated string Field.ChildField.ChildField
// where Field is a struct field name and ChildField is an embedded struct
// field.
//
// GetStructNodes only returns public fields. Thus there is no risk
// of editing an un-editable field and causing a panic.
//
// s interface{} must be a struct pointer or GetStructNodes will panic.
//
// Certain types do not map for the purposes of reading in configuration values. Therefore
// the following types are ignored:
// - arrays (slices are used)
// - functions
// - channels
// - complex
// - interface
// - maps
//
// Note that any nil pointers will get initialized. Therefore, using "GetStructNodes"
// has the side effect of fully initializing the provided struct and all its
// sub-parts (except private members which are skipped).
//
// "time.Time" is NOT included by default in the Options.NoFollow list.
//
// The nodes returned from "StructNodes" should not be deleted or modified except through sanctioned
// node methods as there may be unintended side effects.
func StructNodes(v interface{}, options Options) (nodes map[string]*Node) {
	_, err := util.IsStructPointer(v)
	if err != nil {
		panic(err.Error())
	}

	return getNodes("", v, options)
}

type Options struct {
	// NoFollow is a slice of strings of struct types
	// where the underlying struct fields are ignored but
	// the struct itself is still added as a node.
	NoFollow []string

	// IgnoreTypes are types that are ignored when generating the
	// struct node tree. Certain underlying go types such as functions and channels
	// are always ignored. This list allows the user to define a list of types
	// such as a custom "int" type.
	IgnoreTypes []string
}

// getNodes iterates and recurses through the provided struct pointer.
//
// "v" is already known and assumed to be a struct pointer.
// "prefix" is simply the "parent" full path.
func getNodes(prefix string, v interface{}, options Options) (nodes map[string]*Node) {
	nodes = make(map[string]*Node)
	// Iterate through struct fields.
	vStruct := reflect.ValueOf(v).Elem()
	for i := 0; i < vStruct.NumField(); i++ {
		rawField := vStruct.Field(i)

		if !rawField.CanSet() { // skip private variables
			continue
		}

		// Check if field is pointer and follow to get the actual
		// value. If the pointer is nil then initialize.
		field := rawField
		if rawField.Kind() == reflect.Ptr {
			if rawField.IsNil() {
				z := reflect.New(rawField.Type().Elem())
				rawField.Set(z)
			}

			// Follow pointer.
			field = reflect.Indirect(rawField)
		}

		// Skip ignored kinds like functions.
		if isIgnoredKind(field.Kind()) {
			continue
		}

		// Skip ignored types
		if isIgnoredType(field.Type().String(), options.IgnoreTypes) {
			continue
		}

		// Skip slice if underlying type is not a basic type.
		if field.Kind() == reflect.Slice {
			baseType := reflect.TypeOf(field.Interface()).Elem()
			if baseType.Kind() == reflect.Ptr {
				baseType = baseType.Elem()
			}
			if !isBasicType(baseType.Kind()) {
				continue
			}
		}

		node := &Node{
			Prefix:     prefix,
			FieldValue: field,
			Field:      vStruct.Type().Field(i),
			Index:      i,
		}

		nodes[node.FullName()] = node

		// If node is a struct then recurse (skip if it's on the noFollow type list).
		if field.Kind() == reflect.Struct && followStruct(node.ValueType(), options.NoFollow) {
			mergeNodes(nodes, getNodes(node.FullName(), field.Addr().Interface(), options))
		}
	}
	return nodes
}

// ignoreList contains basic go types that are always ignored
// because they don't make sense in the context of app configuration.
var ignoreList = []reflect.Kind{
	reflect.Array,
	reflect.Func,
	reflect.Chan,
	reflect.Complex64,
	reflect.Complex128,
	reflect.Interface,
	reflect.Map,
}

func isIgnoredKind(fieldKind reflect.Kind) bool {
	for _, v := range ignoreList {
		if fieldKind == v {
			return true
		}
	}

	return false
}

func isIgnoredType(fieldType string, ignoredTypes []string) bool {
	for _, v := range ignoredTypes {
		if fieldType == v {
			return true
		}
	}

	return false
}

// basicTypes is the list of basic types that are directly set/get
// by default.
var basicTypes = []reflect.Kind{
	reflect.String,
	reflect.Bool,
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
	reflect.Float32,
	reflect.Float64,
}

func isBasicType(t reflect.Kind) bool {
	for _, bt := range basicTypes {
		if t == bt {
			return true
		}
	}

	return false
}

// mergeNodes adds "nodesB" to "nodesA" and returns "nodesA"
func mergeNodes(nodesA, nodesB map[string]*Node) map[string]*Node {
	for k, v := range nodesB {
		nodesA[k] = v
	}

	return nodesA
}

// followStruct decides if the struct should be followed by checking the value type against
// the noFollow list (provided by the user). "noFollow" slice items are the type name not the
// kind. So, for "time.Time" structs the noFollow item is "time.Time".
func followStruct(typeName string, noFollow []string) bool {
	for _, fv := range noFollow {
		if typeName == fv {
			return false
		}
	}
	return true
}

// Parent returns the parent struct node if one exists.
func Parent(child *Node, nodes map[string]*Node) (parent *Node) {
	parent, _ = nodes[child.ParentName()]
	return parent
}

// Parents returns all parents of the child. The returned slice
// is ordered from oldest parent to the immediate parent of the child.
func Parents(child *Node, nodes map[string]*Node) (parents []*Node) {
	// First parent name is most distant ancestor.
	for _, name := range child.ParentsNames() {
		ancestor, ok := nodes[name]
		if !ok {
			// All ancestors should exist unless the node map was modified inappropriately.
			panic(fmt.Sprintf("ancestor '%v' of '%v' should exist", name, child.FullName()))
		}

		parents = append(parents, ancestor)
	}

	return parents
}
