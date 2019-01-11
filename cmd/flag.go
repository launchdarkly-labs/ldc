package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/launchdarkly/ldc/api"
)

var flagCompleter = makeCompleter(emptyOnError(listFlagKeys))

func addFlagCommands(shell *ishell.Shell) {

	root := &ishell.Cmd{
		Name:    "flags",
		Aliases: []string{"flag"},
		Help:    "list and operate on flags",
		Func:    showFlags,
	}
	root.AddCmd(&ishell.Cmd{
		Name:      "list",
		Help:      "list flags",
		Aliases:   []string{"ls", "l", "show"},
		Completer: flagCompleter,
		Func:      showFlags,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "show",
		Help:      "show",
		Completer: flagCompleter,
		Func:      showFlag,
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
					c.Err(err)
					return
				}
				c.Println("Status: " + status.Name)
				c.Printf("Last Requested: %v\n", status.LastRequested)
			} else {
				statuses, _, err := api.Client.FeatureFlagsApi.GetFeatureFlagStatuses(api.Auth, api.CurrentProject, api.CurrentEnvironment)
				if err != nil {
					c.Err(err)
					return
				}
				buf := bytes.Buffer{}
				table := tablewriter.NewWriter(&buf)
				table.SetHeader([]string{"Status", "Last Requested"})
				for _, status := range statuses.Items {
					table.Append([]string{status.Name, status.LastRequested})
				}
				table.Render()
				c.Println(buf.String())
			}
		},
	})

	shell.AddCmd(root)
}

func getFlagArg(c *ishell.Context, pos int) *ldapi.FeatureFlag {
	flags, err := listFlags()
	if err != nil {
		c.Err(err)
		return nil
	}

	if len(c.Args) > pos {
		flagKey := c.Args[pos]
		for _, flag := range flags {
			if flag.Key == flagKey {
				return &flag // nolint:scopelint // ok because we return here
			}
		}
		return nil
	}

	options, err := listFlagKeys()
	if err != nil {
		c.Err(err)
		return nil
	}

	choice := c.MultiChoice(options, "Choose a flag: ")
	if choice < 0 {
		return nil
	}

	return &flags[choice]
}

func listFlags() ([]ldapi.FeatureFlag, error) {
	// TODO other projects
	flags, _, err := api.Client.FeatureFlagsApi.GetFeatureFlags(api.Auth, api.CurrentProject, nil)
	if err != nil {
		return nil, err
	}
	return flags.Items, nil
}

func listFlagKeys() ([]string, error) {
	var keys []string
	flags, err := listFlags()
	if err != nil {
		return nil, err
	}
	for _, flag := range flags {
		keys = append(keys, flag.Key)
	}
	return keys, nil
}

func showFlag(c *ishell.Context) {
	flag := getFlagArg(c, 0)
	if flag == nil {
		c.Err(errors.New("flag not found"))
		return
	}
	renderFlag(c, *flag)
}

func showFlags(c *ishell.Context) {
	if len(c.Args) > 0 {
		flag := getFlagArg(c, 0)
		if flag == nil {
			c.Println("Flag not found")
			return
		}
		renderFlag(c, *flag)
		return
	}

	flags, err := listFlags()
	if err != nil {
		c.Err(err)
		return
	}
	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Key", "Name", "Description"})
	for _, flag := range flags {
		table.Append([]string{flag.Key, flag.Name, flag.Description})
	}
	table.Render()
	renderPagedTable(c, buf)
}

func renderFlag(c *ishell.Context, flag ldapi.FeatureFlag) {
	if renderJSON(c) {
		data, err := json.MarshalIndent(flag, "", "  ")
		if err != nil {
			c.Err(err)
			return
		}
		c.Println(string(data))
		return
	}

	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Field", "Value"})
	table.Append([]string{"Key", flag.Key})
	table.Append([]string{"Name", flag.Key})
	table.Append([]string{"Tags", strings.Join(flag.Tags, " ")})
	table.Append([]string{"Kind", flag.Kind})
	table.Append([]string{"Goal IDs", strings.Join(flag.GoalIds, " ")})
	table.Render()
	c.Print(buf.String())

	if flag.Kind == "multivariate" {
		c.Println("Variations:")
		buf := bytes.Buffer{}
		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"Index", "Name", "Description", "Value"})
		for i, variation := range flag.Variations {
			valueBuf, _ := json.Marshal(variation.Value)
			table.Append([]string{strconv.Itoa(i), variation.Name, variation.Description, string(valueBuf)})
		}
		table.Render()
		c.Println(buf.String())
	}

	buf = bytes.Buffer{}
	table = tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Environment", "On", "Last Modified"})
	for envKey, envStatus := range flag.Environments {
		table.Append([]string{envKey, fmt.Sprintf("%v", envStatus.On), time.Unix(envStatus.LastModified/1000, 0).Format("2006/01/02 15:04")})
	}
	table.Render()
	c.Println(buf.String())
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
	flag, _, err := api.Client.FeatureFlagsApi.PostFeatureFlag(api.Auth, api.CurrentProject, ldapi.FeatureFlagBody{
		Name: name,
		Key:  key,
		Variations: []ldapi.Variation{
			ldapi.Variation{Value: &t},
			ldapi.Variation{Value: &f},
		},
	}, nil)
	if err != nil {
		c.Err(err)
		return
	}
	if renderJSON(c) {
		renderFlag(c, flag)
	}
}

func editFlag(c *ishell.Context) {
	flag := getFlagArg(c, 0)
	if flag == nil {
		return
	}
	data, _ := json.MarshalIndent(flag, "", "    ")
	patchComment, err := editFile(c, data)
	if err != nil {
		c.Err(err)
		return
	}

	if patchComment == nil {
		c.Println("No changes")
		return
	}

	_, _, err = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, *patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println("Updated flag")
}

func addTag(c *ishell.Context) {
	flag := getFlagArg(c, 0)
	tag := c.Args[1]
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "add",
		Path:  "/tags/-",
		Value: interfacePtr(tag),
	}}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func removeTag(c *ishell.Context) {
	flag := getFlagArg(c, 0)
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
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:   "remove",
		Path: fmt.Sprintf("/tags/%d", index),
	}}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func on(c *ishell.Context) {
	flag := getFlagArg(c, 0)
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/on", api.CurrentEnvironment),
		Value: interfacePtr(true),
	}}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

func off(c *ishell.Context) {
	flag := getFlagArg(c, 0)
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/on", api.CurrentEnvironment),
		Value: interfacePtr(false),
	}}
	_, _, _ = api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
}

// add/remove tag, toggle on/off

func createFlag(c *ishell.Context) {
	// uhhh
	createToggleFlag(c)
}

func deleteFlag(c *ishell.Context) {
	c.Println("Not implemented")
}

func interfacePtr(i interface{}) *interface{} {
	return &i
}
