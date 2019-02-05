package path_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/launchdarkly/ldc/cmd/internal/path"
)

func TestResourcePath(t *testing.T) {
	specs := []struct {
		path           string
		expectedConfig *string
		expectedKeys   []string
		expectedAbs    bool
	}{
		{"flagA", nil, []string{"flagA"}, false},
		{"", nil, []string{""}, false},

		{"/flagA", nil, []string{"flagA"}, true},
		{"//configA/flagA", strPtr("configA"), []string{"flagA"}, true},
		{"/", nil, []string{}, true},
		{"///", strPtr(""), []string{""}, true},
	}
	for _, tt := range specs {
		t.Run(tt.path, func(t *testing.T) {
			p := path.ResourcePath(tt.path)
			assert.Equal(t, tt.expectedConfig, p.Config())
			assert.Equal(t, tt.expectedKeys, p.Keys())
			assert.Equal(t, tt.expectedAbs, p.IsAbs())
		})
	}
}

func strPtr(s string) *string {
	return &s
}
