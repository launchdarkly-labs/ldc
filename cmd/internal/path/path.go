package path

import "strings"

// ResourcePath is a path to a resource
type ResourcePath string

// String returns a string representation of a path
func (p ResourcePath) String() string {
	return string(p)
}

// IsAbs returns true if the path is absolute
func (p ResourcePath) IsAbs() bool {
	return strings.HasPrefix(string(p), "/")
}

// Config returns the config key if there is one
func (p ResourcePath) Config() *string {
	if !strings.HasPrefix(string(p), "//") {
		return nil
	}
	config := strings.Split(string(p), "/")[2]
	return &config
}

// Keys returns a list of keys
func (p ResourcePath) Keys() []string {
	parts := strings.Split(string(p), "/")
	if p.Config() != nil {
		return parts[3:]
	}
	if p.IsAbs() {
		if len(p) > 1 {
			return parts[1:]
		}
		return []string{}
	}
	return parts
}

// Depth returns the depth of the path
func (p ResourcePath) Depth() int {
	return len(p.Keys())
}

// NewAbsPath creates a new absolute path with optional config key
func NewAbsPath(configKey *string, p ...string) ResourcePath {
	root := "/"
	if configKey != nil {
		root = "//" + *configKey + "/"
	}
	return ResourcePath(root + strings.Join(p, "/"))
}
