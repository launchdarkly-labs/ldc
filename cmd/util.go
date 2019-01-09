package cmd

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/mattbaird/jsonpatch"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/abiosoft/ishell"
)

func confirmDelete(c *ishell.Context, name string, expectedValue string) bool {
	if !interactive {
		return false
	}
	c.Printf("Re-enter the %s '%s' to delete: ", name, expectedValue)
	value := c.ReadLine()
	return value == expectedValue
}

func withPrefix(keys []string, prefix string) []string {
	var completions []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			completions = append(completions, key)
		}
	}
	return completions
}

func toPrefix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func makeCompleter(fetch func() []string) func(args []string) []string {
	return func(args []string) []string {
		var completions []string
		for _, key := range fetch() {
			if len(args) == 0 || strings.HasPrefix(key, args[0]) {
				completions = append(completions, key)
			}
		}
		return completions
	}
}

func editFile(c *ishell.Context, original []byte) (patch *ldapi.PatchComment, err error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	current := original
	for {
		file, _ := ioutil.TempFile("/tmp", "ldc")
		name := file.Name()
		file.Write(current)
		file.Close()

		// TODO why doesn't $EDITOR work? $PATH?
		proc, err := os.StartProcess("/usr/local/bin/nvim", []string{"nvim", name}, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
		if err != nil {
			return nil, err
		}
		proc.Wait()

		file, err = os.Open(name)
		if err != nil {
			return nil, err
		}

		newData, fileErr := ioutil.ReadAll(file)

		err = os.Remove(name)
		if err != nil {
			c.Println("Unable to delete temporary file: %s", err)
		}

		if fileErr != nil {
			c.Err(fileErr)
			c.Println("Unable to read file: %s", err)
			c.Print("Try again (y/n)?")
			c.ReadLineWithDefault("y")
			continue
		}
		file.Close()

		patch, err := jsonpatch.CreatePatch(original, newData)
		if err != nil {
			c.Err(err)
			c.Println("Unable to parse json. Try again? (y/n) ")
			c.Print("Try again (y/n)")
			c.ReadLineWithDefault("y")
			current = newData
			continue
		}

		if len(patch) == 0 {
			c.Println("No changes")
			return nil, nil
		}

		var patchComment ldapi.PatchComment
		for _, op := range patch {
			patchComment.Patch = append(patchComment.Patch, ldapi.PatchOperation{
				Op:    op.Operation,
				Path:  op.Path,
				Value: &op.Value,
			})
		}

		c.Print("Unable to parse json.  Make changes? (y/n) ")
		if strings.ToLower(c.ReadLineWithDefault("y")) == "y" {
			continue
		}

		c.Print("Enter comment: ")
		patchComment.Comment = c.ReadLine()
		return &patchComment, nil
	}
}
