package load

import "github.com/pcelvng/go-config/util/node"

type Loader interface {
	// Load expects a struct pointer and will read in
	// the config values into the underlying struct.
	//
	// It's possible and expected that the struct will already be populated with
	// at least some values. Load should not clear out existing values
	// if a replacing value does not exist.
	//
	// Load should support embedded structs and struct types.
	//
	// Load should ignore reading in values where the public member has a struct tag
	// of this type of dash '-'. So, if the implementation were for loading flags
	// and a 'flag' struct tag was provided as `flag:"-"` then no value should be read
	// in to that struct member.
	//
	// Load should also ignore reading in values that have the 'config' struct tag
	// with an 'ignore' value.
	//
	// Load is not responsible for validation other than correct formatting
	// and type matching.
	//
	// Load should have specific support for reading in and parsing time.Time values
	// and time.Duration. time.Time values should also support the 'time' tag value
	// that specifies the time.Time expected format. The default format should be time.RFC3339.
	// The 'time' tag should accept a raw format or a standard format available in the time
	// standard library module. For example, instead of specifying "Mon, 02 Jan 2006 15:04:05 MST"
	// as the time tag value, the user could specify "RFC1123". If the provided time value
	// is incorrect, then Load should return an error indicating what the correct format is
	// and that the time format is incorrect. time.Duration type values should be read in as time duration
	// parsable strings.
	//
	// Load should be implemented with the fact in mind that other Loaders could
	// be called before or after it. Therefore, Load does not return an error if a 'req'
	// field value is not provided since that value could have already been provided or will
	// be provided by a call to a different Loader.
	Load([]byte, []*node.Nodes) error
}

type Unloader interface {
	// Unload will express the underlying provided pointer struct as a series of
	// pre-formatted bytes.
	//
	// If the underlying Unload implementation were for JSON then the returned bytes
	// would be indented json.
	//
	// All, non-hidden values should be represented in the returned bytes even if no default
	// value is specified. If a default value is specified then that value should be pre-populated
	// in the returned bytes template.
	//
	// Unload should support the following struct tags:
	// - config (config="ignore" ignore the field)
	// - req (communicates if the field is required)
	// - desc (field description; most likely expressed as a comment)
	// - format specific tag (ie "env" for environment variables)
	// - time (for time.Time field types)
	//
	// Additionally Unload will ignore the following field types:
	// - functions
	// - interfaces
	// - complex64, complex128
	// - arrays (slices are ok)
	// - channels
	// - private fields
	//
	// Generally speaking, Load and Unload should be idempotent. That is, the generated template
	// from Unload should produce identical struct values when read back in by Load.
	//
	// time.Time field types should provide some type of hint to the user what the requested
	// time format is. Most likely this will look like a comment with the specified time format.
	//
	// time.Duration type should be supported with default values time.Duration as string parsable
	// values.
	Unload([]*node.Nodes) ([]byte, error)
}

type LoadUnloader interface {
	Loader
	Unloader
}
