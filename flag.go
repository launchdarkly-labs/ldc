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
	"time"

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
		Name:      "add-tag",
		Help:      "add a tag to a flag: flag add-tag flag tag",
		Completer: flagCompleter,
		Func:      addTag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "remove-tag",
		Help:      "remove a tag from a flag: flag remove-tag flag tag",
		Completer: flagCompleter,
		Func:      removeTag,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "on",
		Help:      "turn a boolean flag on",
		Completer: flagCompleter,
		Func:      on,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "off",
		Help:      "turn a boolean flag off",
		Completer: flagCompleter,
		Func:      off,
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
				buf := bytes.Buffer{}
				table := tablewriter.NewWriter(&buf)
				table.SetHeader([]string{"Status", "Last Requested"})
				for _, status := range statuses.Items {
					table.Append([]string{status.Name, status.LastRequested})
				}
				table.SetRowLine(true)
				table.Render()
				c.Println(buf.String())
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

func getFlagArg(c *ishell.Context) *swagger.FeatureFlag {
	flags := listFlags()
	var foundFlag *swagger.FeatureFlag
	var flagKey string
	if len(c.Args) > 0 {
		flagKey = c.Args[0]
		for _, flag := range flags {
			if flag.Key == flagKey {
				copy := flag
				foundFlag = &copy
			}
		}
	} else {
		// TODO LOL
		options := listFlagKeys()
		choice := c.MultiChoice(options, "Choose an environment")
		foundFlag = &flags[choice]
		flagKey = foundFlag.Key
	}
	return foundFlag
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
		flag := getFlagArg(c)
		c.Printf("Key: %s\n", flag.Key)
		c.Printf("Name: %s\n", flag.Name)
		c.Printf("Tags: %v\n", flag.Tags)
		c.Printf("Kind: %s\n", flag.Kind)

		if flag.Kind == "multivariate" {
			c.Println("Variations:")
			buf := bytes.Buffer{}
			table := tablewriter.NewWriter(&buf)
			table.SetHeader([]string{"Name", "Description", "Value"})
			for _, variation := range flag.Variations {
				table.Append([]string{variation.Name, variation.Description, fmt.Sprintf("%v", *variation.Value)})
			}
			table.SetRowLine(true)
			table.Render()
			c.Println(buf.String())
		}

		buf := bytes.Buffer{}
		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"Environment", "On", "Last Modified"})
		for envKey, envStatus := range flag.Environments {
			table.Append([]string{envKey, fmt.Sprintf("%v", envStatus.On), time.Unix(envStatus.LastModified/1000, 0).Format("2006/01/02 15:04")})
		}
		table.SetRowLine(true)
		table.Render()
		c.Println(buf.String())
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
	flag := getFlagArg(c)
	jsonbytes, _ := json.MarshalIndent(flag, "", "    ")
	file, _ := ioutil.TempFile("/tmp", "ldc")
	name := file.Name()
	file.Write(jsonbytes)
	file.Close()
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	// TODO why doesn't $EDITOR work? $PATH?
	proc, err := os.StartProcess("/usr/local/bin/nvim", []string{"nvim", name}, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
	proc.Wait()
	if err != nil {
		panic(err)
	}
	file, _ = os.Open(name)
	newBytes, _ := ioutil.ReadAll(file)
	file.Close()
	err = os.Remove(name)
	if err != nil {
		panic(err)
	}
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
			Value: op.Value,
		})
	}
	patchComment.Comment = "Hey, this is a comment!"
	_, _, err = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
	if err != nil {
		// well duh
		c.Err(err)
	} else {
		c.Println("Updated flag")
	}
}

func addTag(c *ishell.Context) {
	flag := getFlagArg(c)
	tag := c.Args[1]
	var patchComment swagger.PatchComment
	patchComment.Patch = []swagger.FlagsprojectKeyfeatureFlagKeyPatch{
		swagger.FlagsprojectKeyfeatureFlagKeyPatch{
			Op:    "add",
			Path:  "/tags/-",
			Value: tag,
		},
	}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func removeTag(c *ishell.Context) {
	flag := getFlagArg(c)
	tag := c.Args[1]
	index := -1
	for i, taga := range flag.Tags {
		if tag == taga {
			index = i
		}
	}
	if index < 0 {
		c.Printf("Flag does not have tag %s", tag)
	}
	var patchComment swagger.PatchComment
	patchComment.Patch = []swagger.FlagsprojectKeyfeatureFlagKeyPatch{
		swagger.FlagsprojectKeyfeatureFlagKeyPatch{
			Op:   "remove",
			Path: fmt.Sprintf("/tags/%d", index),
		},
	}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func on(c *ishell.Context) {
	flag := getFlagArg(c)
	var patchComment swagger.PatchComment
	patchComment.Patch = []swagger.FlagsprojectKeyfeatureFlagKeyPatch{
		swagger.FlagsprojectKeyfeatureFlagKeyPatch{
			Op:    "replace",
			Path:  fmt.Sprintf("/environments/%s/on", api.CurrentEnvironment),
			Value: true,
		},
	}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func off(c *ishell.Context) {
	flag := getFlagArg(c)
	var patchComment swagger.PatchComment
	patchComment.Patch = []swagger.FlagsprojectKeyfeatureFlagKeyPatch{
		swagger.FlagsprojectKeyfeatureFlagKeyPatch{
			Op:    "replace",
			Path:  fmt.Sprintf("/environments/%s/on", api.CurrentEnvironment),
			Value: false,
		},
	}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

// add/remove tag, toggle on/off

func createFlag(c *ishell.Context) {
	// uhhh
	createToggleFlag(c)
}

func deleteFlag(c *ishell.Context) {
}
