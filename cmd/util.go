package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/mattbaird/jsonpatch"
	ishell "gopkg.in/abiosoft/ishell.v2"

	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	cINTERACTIVE = "interactive"
	cEDITOR      = "editor"
	cJSON        = "json"
)

func confirmDelete(c *ishell.Context, name string, expectedValue string) bool {
	if !isInteractive(c) {
		return true
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
	return func(args []string) (completions []string) {
		if len(args) > 1 {
			return nil
		}
		for _, key := range fetch() {
			if len(args) == 0 || strings.HasPrefix(key, args[0]) {
				if strings.Contains(key, " ") {
					key = `"` + key + `"`
				}
				completions = append(completions, key)
			}
		}
		return completions
	}
}

func editFile(c *ishell.Context, original []byte) (patch *ldapi.PatchComment, err error) {
	editor := c.Get(cEDITOR).(string)
	cmd := exec.Command("command", "-v", editor) // nolint:gosec // ok to launch subprocess with variable
	editorPathRaw, err := cmd.Output()
	if err != nil {
		c.Err(err)
		return nil, err
	}
	editorPath := strings.TrimSpace(string(editorPathRaw))

	var patchOps []jsonpatch.JsonPatchOperation
	current := original
	for {
		file, err := ioutil.TempFile("/tmp", "ldc")
		if err != nil {
			c.Err(err)
			return nil, err
		}
		name := file.Name()
		_, err = file.Write(current)
		if err != nil {
			c.Err(err)
			return nil, err
		}
		if err := file.Close(); err != nil {
			c.Err(err)
			return nil, err
		}

		proc, err := os.StartProcess(editorPath, []string{editor, name}, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
		if err != nil {
			return nil, err
		}
		if _, err := proc.Wait(); err != nil {
			c.Err(err)
			return nil, err
		}

		file, err = os.Open(name) // nolint:gosec // G304: Potential file inclusion via variable // ok because we created name
		if err != nil {
			return nil, err
		}

		newData, fileErr := ioutil.ReadAll(file)

		err = os.Remove(name)
		if err != nil {
			c.Println("Unable to delete temporary file: %s", err)
		}

		if fileErr != nil {
			c.Println("Unable to read file: %s", err)
			c.Print("Try again? [y]/n  ")
			if !yesOrNo(c) {
				c.Println("Edit aborted")
				break
			}
		}
		if err := file.Close(); err != nil {
			return nil, err
		}

		patchOps, err = jsonpatch.CreatePatch(original, newData)
		if err != nil {
			patchOps = nil
			if err.Error() == "Invalid JSON Document" {
				c.Print("Unable to parse json. Make changes? [y]/n ")
			} else {
				c.Printf("Unable to create patch: %s\n", err.Error())
				c.Print("Make changes? [y]/n ")
			}
			if !yesOrNo(c) {
				c.Println("Edit aborted")
				break
			}
			current = newData
			continue
		}

		break
	}

	if len(patchOps) == 0 {
		return nil, nil
	}

	var patchComment ldapi.PatchComment
	for _, op := range patchOps {
		patchComment.Patch = append(patchComment.Patch, ldapi.PatchOperation{
			Op:    op.Operation,
			Path:  op.Path,
			Value: &op.Value,
		})
	}

	c.Print("Enter comment: ")
	patchComment.Comment = c.ReadLine()
	return &patchComment, nil
}

func emptyOnError(f func() ([]string, error)) func() []string {
	return func() []string {
		values, err := f()
		if err != nil {
			return nil
		}
		return values
	}
}

func yesOrNo(c *ishell.Context) (yes bool) {
	val := c.ReadLine()
	if val == "" || strings.ToLower(val) == "y" {
		return true
	}
	return false
}

var jsonMode *bool

func setJSON(val bool) {
	jsonMode = &val
}

func renderJSON(c *ishell.Context) bool {
	if jsonMode != nil {
		return *jsonMode
	}
	return reflect.DeepEqual(c.Get(cJSON), true)
}

func isInteractive(c *ishell.Context) bool {
	return reflect.DeepEqual(c.Get(cINTERACTIVE), true)
}
