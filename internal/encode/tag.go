package encode

import "strings"

// ParseTag parses a struct tag value. The first comma-separated segment is the name;
// "omitprefix" is a reserved modifier.
func ParseTag(tag string) (name string, omitPrefix bool, aliases []string) {
	if tag == "" {
		return "", false, nil
	}
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		switch part {
		case "":
			continue
		case "omitprefix":
			omitPrefix = true
		default:
			aliases = append(aliases, part)
		}
	}
	return name, omitPrefix, aliases
}
