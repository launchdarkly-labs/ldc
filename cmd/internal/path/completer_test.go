package path_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/launchdarkly/ldc/cmd/internal/path"
)

func TestCompletions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Completions Suite")
}

var _ = Describe("GetCompletions", func() {
	var defaultPathSource path.DefaultPathSource

	Describe("Without a config or default config (just a default path)", func() {
		BeforeEach(func() {
			defaultPathSource = path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
				return path.ResourcePath("/defaultProj/defaultEnv"), nil
			})
		})

		It("allows completions relative to the default path", func() {
			completer := path.NewCompleter(defaultPathSource, nil, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("/defaultProj/defaultEnv"))
					return []string{"flagA"}, nil
				}))
			completions, _ := completer.GetCompletions("")
			Expect(completions).To(BeEquivalentTo([]string{"/", "flagA"}))
		})

		It("completes absolute paths without a config", func() {
			completer := path.NewCompleter(defaultPathSource, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("/"))
					return []string{"projA"}, nil
				}))
			completions, _ := completer.GetCompletions("/")
			Expect(completions).To(BeEquivalentTo([]string{"/...", "//", "/projA"}))
		})

		It("sorts the results", func() {
			completer := path.NewCompleter(defaultPathSource, nil, nil, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					return []string{"flagB", "flagC", "flagA"}, nil
				}))
			completions, _ := completer.GetCompletions("")
			Expect(completions).To(Equal([]string{"/", "flagA", "flagB", "flagC"}))
		})

		It("appends a slash for partial path completions", func() {
			completer := path.NewCompleter(defaultPathSource, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("/"))
					return []string{"projA"}, nil
				}),
				nil)
			completions, _ := completer.GetCompletions("/")
			Expect(completions).To(BeEquivalentTo([]string{"/.../", "//", "/projA/"}))
		})
	})

	Describe("With a config", func() {
		It("allows completions relative to the default path for that config", func() {
			defaultPathSource = path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
				Expect(configKey).To(BeEquivalentTo(strPtr("configA")))
				return path.ResourcePath("/defaultProj/defaultEnv"), nil
			})

			completer := path.NewCompleter(defaultPathSource, nil, nil, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("//configA/defaultProj/defaultEnv"))
					return []string{"flagA"}, nil
				}))
			completions, _ := completer.GetCompletions("//configA/.../.../")
			Expect(completions).To(BeEquivalentTo([]string{"//configA/.../.../flagA"}))
		})

		It("completes absolute paths without a config", func() {
			defaultPathSource = path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
				Expect(configKey).To(BeNil())
				return path.ResourcePath("/defaultProj/defaultEnv"), nil
			})

			completer := path.NewCompleter(defaultPathSource, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("/"))
					return []string{"projA"}, nil
				}))
			completions, _ := completer.GetCompletions("/")
			Expect(completions).To(BeEquivalentTo([]string{"/...", "//", "/projA"}))
		})

		It("completes configs", func() {
			completer := path.NewCompleter(defaultPathSource,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("//"))
					return []string{"my-config"}, nil
				}))
			completions, _ := completer.GetCompletions("//")
			Expect(completions).To(BeEquivalentTo([]string{"//my-config/"}))
		})

		It("does not include '...' as a suggestion unless defaults starts from root", func() {
			defaultPathSource = path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
				Expect(configKey).To(BeNil())
				return path.ResourcePath("/defaultProj/defaultEnv"), nil
			})

			completer := path.NewCompleter(defaultPathSource, nil, nil,
				path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
					Expect(parentPath).To(BeEquivalentTo("/projA"))
					return []string{"envB"}, nil
				}))
			completions, _ := completer.GetCompletions("/projA/")
			Expect(completions).To(BeEquivalentTo([]string{"/projA/envB"}))
		})
	})
})

func TestReplaceDefaults(t *testing.T) {
	specs := []struct {
		expectedPath string
		path         string
		defaultPath  string
		depth        int
	}{
		{"/default-proj/default-env", "/.../default-env", "/default-proj/default-env", 2},
		{"/default-proj/...", "/default-proj/...", "/default-proj/default-env", 2},
		{"/default-proj", "/...", "/default-proj/default-env", 1},
	}
	for _, tt := range specs {
		t.Run(tt.path, func(t *testing.T) {
			defaultPathSource := path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
				return path.ResourcePath(tt.defaultPath), nil
			})

			actualPath, err := path.ReplaceDefaults(path.ResourcePath(tt.path), defaultPathSource, tt.depth)
			require.NoError(t, err)
			assert.Equal(t, path.ResourcePath(tt.expectedPath), actualPath)
		})
	}
}
