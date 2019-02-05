package path

import (
	"fmt"
	"sort"
)

// Lister is an interface that returns a list of children for a parent path
type Lister interface {
	List(parentPath ResourcePath) ([]string, error)
}

// ListerFunc is a convenience method for creating a Lister from a function
type ListerFunc func(parentPath ResourcePath) ([]string, error)

// List returns a list of children for a parentPath, using defaultPath for any "..." path components
func (l ListerFunc) List(parentPath ResourcePath) ([]string, error) {
	return l(parentPath)
}

// DefaultPathSource allows fetching of a default path for a config key
type DefaultPathSource interface {
	GetDefaultPath(configKey *string) (ResourcePath, error)
}

// DefaultPathSourceFunc is a convenience method for creating a DefaultPathSource from a function
type DefaultPathSourceFunc func(configKey *string) (ResourcePath, error)

// GetDefaultPath returns the default path for a config key
func (d DefaultPathSourceFunc) GetDefaultPath(configKey *string) (ResourcePath, error) {
	return d(configKey)
}

// Completer implements a path completer
type Completer struct {
	defaultPathSource DefaultPathSource
	configLister      Lister
	listers           []Lister
}

// NewCompleter Creates a
func NewCompleter(defaultPathSource DefaultPathSource, configLister Lister, listers ...Lister) *Completer {
	return &Completer{
		defaultPathSource: defaultPathSource,
		configLister:      configLister,
		listers:           listers,
	}
}

// GetCompletions return completions or nil if there is an error retrieving completions
func (c *Completer) GetCompletions(arg string) ([]string, error) {
	var parentPath ResourcePath
	var lister Lister
	var prefix string
	var suffix string
	var err error
	var extras []string

	argPath := ResourcePath(arg)
	switch {
	case argPath.IsAbs(): // absolute
		var originalParentPath ResourcePath
		if argPath.Depth() == 0 {
			originalParentPath = argPath
			parentPath = argPath
			if argPath.Config() != nil {
				lister = c.configLister
				prefix = "//"
				suffix = "/"
				break
			}
		} else {
			originalParentPath = NewAbsPath(argPath.Config(), argPath.Keys()[0:argPath.Depth()-1]...)
			parentPath, err = ReplaceDefaults(originalParentPath, c.defaultPathSource, len(c.listers))
			if err != nil {
				return nil, err
			}
		}

		trueParentDepth := parentPath.Depth()
		if parentPath.Depth() == 1 && parentPath.Keys()[0] == "" {
			trueParentDepth = 0
		}
		lister = c.listers[trueParentDepth]
		if len(c.listers) > trueParentDepth+1 {
			suffix = "/"
		}
		defaultPath, err := c.defaultPathSource.GetDefaultPath(argPath.Config())
		if err != nil {
			return nil, err
		}

		if trueParentDepth < len(c.listers) && len(c.listers) <= defaultPath.Depth() {
			parentIsDefault := true
			for _, p := range parentPath.Keys() {
				if p != "..." {
					parentIsDefault = false
				}
			}
			if parentIsDefault {
				extras = append(extras, "..."+suffix)
			}
		}
		if originalParentPath.Depth() >= 1 {
			prefix = originalParentPath.String() + "/"
		} else {
			prefix = "/"
		}
		if arg == "/" {
			extras = append(extras, "/")
		}
	default: // relative
		parentPath, err = c.defaultPathSource.GetDefaultPath(nil)
		if err != nil {
			return nil, err
		}
		lister = c.listers[len(c.listers)-1]
		extras = append(extras, "/")
		prefix = ""
	}

	options, err := lister.List(parentPath)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Got option: %+v", options)

	var results []string
	for _, r := range options {
		results = append(results, prefix+r+suffix)
	}
	for _, r := range extras {
		results = append(results, prefix+r)
	}
	sort.Strings(results)

	return results, nil
}

// ReplaceDefaults is a helper function for interpolating default path components into a path
func ReplaceDefaults(p ResourcePath, defaultPathSource DefaultPathSource, depth int) (ResourcePath, error) {
	for pos := 0; pos < depth && pos < p.Depth(); pos++ {
		keys := p.Keys()
		if keys[pos] != "..." {
			return p, nil
		}
		defaultPath, err := defaultPathSource.GetDefaultPath(p.Config())
		if err != nil {
			return "", err
		}
		if pos >= defaultPath.Depth() {
			break
		}
		newKeys := append(keys[0:pos], append([]string{defaultPath.Keys()[pos]}, keys[pos+1:]...)...)
		p = NewAbsPath(p.Config(), newKeys...)
	}
	return p, nil
}
