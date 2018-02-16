package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ldc/api"
	"ldc/api/swagger"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/mattbaird/jsonpatch"
	"github.com/olekukonko/tablewriter"
)

func AddFlagCommands(shell *ishell.Shell) {

	root := &ishell.Cmd{
		Name:    "flags",
		Aliases: []string{"flag"},
		Help:    "list and operate on flags",
		Func:    list,
	}
	root.AddCmd(&ishell.Cmd{
		Name:      "list",
		Help:      "list flags",
		Completer: flagCompleter,
		Func:      list,
	})
	root.AddCmd(&ishell.Cmd{
		Name:    "create",
		Aliases: []string{"new"},
		Help:    "Create new flag",
		Func:    createFlag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:    "create-toggle",
		Aliases: []string{"new-toggle", "create-boolean"},
		Help:    "Create new boolean flag",
		Func:    createToggleFlag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "edit",
		Help:      "edit a flag's json in a text editor",
		Completer: flagCompleter,
		Func:      editFlag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "delete",
		Aliases:   []string{"remove"},
		Help:      "Delete a flag",
		Completer: flagCompleter,
		Func:      deleteFlag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "status",
		Help:      "show flag's statuses",
		Completer: flagCompleter,
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				status, _, err := api.Client.FeatureFlagsApi.GetFeatureFlagStatus(api.Auth, api.CurrentProject, api.CurrentEnvironment, c.Args[0])
				if err != nil {
					panic(err)
				}
				c.Println("Status: " + status.Name)
				c.Printf("Last Requested: %v\n", status.LastRequested)
			} else {
				statuses, _, err := api.Client.FeatureFlagsApi.GetFeatureFlagStatuses(api.Auth, api.CurrentProject, api.CurrentEnvironment)
				if err != nil {
					panic(err)
				}
				for _, status := range statuses.Items {
					c.Printf("%v\n", status)
				}
			}
		},
	})

	shell.AddCmd(root)
}

func flagCompleter(args []string) []string {
	var completions []string
	// TODO caching?
	for _, key := range listFlagKeys() {
		// fuzzy?
		if len(args) == 0 || strings.HasPrefix(key, args[0]) {
			completions = append(completions, key)
		}
	}
	return completions
}

func getFlag(key string) swagger.FeatureFlag {
	// TODO other projects
	flag, _, err := api.Client.FeatureFlagsApi.GetFeatureFlag(api.Auth, api.CurrentProject, key, nil)
	if err != nil {
		panic(err)
	}
	return flag
}

func listFlags() []swagger.FeatureFlag {
	// TODO other projects
	flags, _, err := api.Client.FeatureFlagsApi.GetFeatureFlags(api.Auth, api.CurrentProject, nil)
	if err != nil {
		panic(err)
	}
	return flags.Items
}

func listFlagKeys() []string {
	var keys []string
	for _, flag := range listFlags() {
		keys = append(keys, flag.Key)
	}
	return keys
}

func list(c *ishell.Context) {
	if len(c.Args) > 0 {
		flag := getFlag(c.Args[0])
		c.Printf("%v\n", flag)
	} else {
		flags := listFlags()
		buf := bytes.Buffer{}
		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"Key", "Name", "Description"})
		for _, flag := range flags {
			table.Append([]string{flag.Key, flag.Name, flag.Description})
		}
		table.SetRowLine(true)
		table.Render()
		if buf.Len() > 1000 {
			c.ShowPaged(buf.String())
		} else {
			c.Print(buf.String())
		}
	}
}

func show(c *ishell.Context) {
}

func createToggleFlag(c *ishell.Context) {
	var key, name string
	switch len(c.Args) {
	case 0:
		c.Println("Need at least a key for the new boolean flag")
	case 1:
		key = c.Args[0]
		name = key
	case 2:
		key = c.Args[0]
		name = c.Args[1]
	}
	var t, f interface{}
	t = true
	f = false
	_, err := api.Client.FeatureFlagsApi.PostFeatureFlag(api.Auth, api.CurrentProject, swagger.FeatureFlagBody{
		Name: name,
		Key:  key,
		Variations: []swagger.Variation{
			swagger.Variation{Value: &t},
			swagger.Variation{Value: &f},
		},
	})
	if err != nil {
		panic(err)
	}
}

func editFlag(c *ishell.Context) {
	// wow lol
	flag := getFlag(c.Args[0])
	jsonbytes, _ := json.MarshalIndent(flag, "", "    ")
	file, _ := ioutil.TempFile("/tmp", "ldc")
	name := file.Name()
	file.Write(jsonbytes)
	fmt.Println(string(jsonbytes))
	file.Close()
	proc, err := os.StartProcess("/usr/local/bin/nvim", []string{"nvim", name}, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
	proc.Wait()
	if err != nil {
		panic(err)
	}
	file, _ = os.Open(name)
	newBytes, _ := ioutil.ReadAll(file)
	file.Close()
	os.Remove(name)
	patch, err := jsonpatch.CreatePatch(jsonbytes, newBytes)
	if err != nil {
		c.Println("you broke it (could not create json patch)")
		return
	}
	var patchComment swagger.PatchComment
	if len(patch) == 0 {
		c.Println("Flag unchanged")
		return
	}
	for _, op := range patch {
		patchComment.Patch = append(patchComment.Patch, swagger.FlagsprojectKeyfeatureFlagKeyPatch{
			Op:    op.Operation,
			Path:  op.Path,
			Value: op.Value.(string),
		})
	}
	patchComment.Comment = "Hey, this is a comment!"
	_, _, err = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, c.Args[0], patchComment)
	if err != nil {
		// well duh
		panic(err)
	}
	// var newFlag swagger.FeatureFlag
	// json.Unmarshal(bytes, &newFlag)
	// diff and patch flag
}

// add/remove tag

func createFlag(c *ishell.Context) {
}

func deleteFlag(c *ishell.Context) {
}
