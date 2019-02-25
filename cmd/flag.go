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
	"github.com/launchdarkly/ldc/cmd/internal/path"
)

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
		Completer: flagEnvCompleter,
		Func:      on,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "off",
		Help:      "turn a boolean flag off",
		Completer: flagEnvCompleter,
		Func:      off,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "rollout",
		Help:      "set the rollout for a flag.  rollout [N:][name:][variation 0 %] [N:][name:][variation 1 %] ...",
		Completer: rolloutCompleter,
		Func:      rollout,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "fallthrough",
		Help:      "set the fallthrough value for a flag.  fallthrough <index> ...",
		Completer: fallthruCompleter,
		Func:      fallthru,
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
		Completer: flagEnvCompleter,
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				flagPath, flag := getFlagConfigArg(c, 0)
				if flag == nil {
					return
				}
				auth := api.GetAuthCtx(getToken(flagPath.Config()))
				client, err := api.GetClient(getServer(flagPath.Config()))
				if err != nil {
					c.Err(err)
					return
				}
				status, _, err := client.FeatureFlagsApi.GetFeatureFlagStatus(auth, flagPath.Project(), flagPath.Environment(), flagPath.Key())
				if err != nil {
					c.Err(err)
					return
				}
				c.Println("Status: " + status.Name)
				c.Printf("Last Requested: %v\n", status.LastRequested)
			} else {
				auth := api.GetAuthCtx(getToken(currentConfig))
				client, err := api.GetClient(getServer(currentConfig))
				if err != nil {
					c.Err(err)
					return
				}
				statuses, _, err := client.FeatureFlagsApi.GetFeatureFlagStatuses(auth, currentProject, currentEnvironment)
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

type perProjectPath struct {
	path.ResourcePath
}

func (p perProjectPath) Project() string {
	return p.Keys()[0]
}

func (p perProjectPath) Key() string {
	return p.Keys()[1]
}

func realFlagPath(p path.ResourcePath) (perProjectPath, error) {
	if len(p.Keys()) != 2 {
		return perProjectPath{}, errors.New("invalid path")
	}
	np, err := path.ReplaceDefaults(p, getDefaultPath, 1)
	if err != nil {
		return perProjectPath{}, err
	}
	return perProjectPath{np}, nil
}

type perEnvironmentPath struct {
	path.ResourcePath
}

func (p perEnvironmentPath) Project() string {
	return p.Keys()[0]
}

func (p perEnvironmentPath) Environment() string {
	return p.Keys()[1]
}

func (p perEnvironmentPath) Key() string {
	return p.Keys()[2]
}

func (p perEnvironmentPath) PerProjectPath() perProjectPath {
	return perProjectPath{path.NewAbsPath(p.Config(), p.Project(), p.Key())}
}

func realFlagConfigPath(p path.ResourcePath) (perEnvironmentPath, error) {
	if len(p.Keys()) != 3 {
		return perEnvironmentPath{}, errors.New("invalid path")
	}
	np, err := path.ReplaceDefaults(p, getDefaultPath, 2)
	if err != nil {
		return perEnvironmentPath{}, err
	}
	return perEnvironmentPath{np}, nil
}

var flagLister = path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
	return listFlagKeys(parentPath.Config(), parentPath.Keys()[0])
})

var projLister = path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
	options, err := listProjectKeys(parentPath.Config())
	return options, err
})

var envLister = path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
	options, err := listEnvironmentKeys(parentPath.Config(), parentPath.Keys()[0])
	return options, err
})

var configLister = path.ListerFunc(func(path path.ResourcePath) (configs []string, err error) {
	configs = append(configs, path.Keys()...)
	return configs, nil
})

func flagCompleter(args []string) (completions []string) {
	if len(args) > 1 {
		return nil
	}

	completer := path.NewCompleter(getDefaultPath, configLister, projLister, flagLister)
	completions, _ = completer.GetCompletions(firstOrEmpty(args))
	return completions
}

func flagEnvCompleter(args []string) (completions []string) {
	if len(args) > 1 {
		return nil
	}

	completer := path.NewCompleter(getDefaultPath, configLister, projLister, envLister, flagLister)
	completions, _ = completer.GetCompletions(firstOrEmpty(args))
	return completions
}

func getFlagArg(c *ishell.Context, pos int) (perProjectPath, *ldapi.FeatureFlag) { // nolint:dupl
	var pathArg path.ResourcePath
	if len(c.Args) > pos {
		pathArg = path.ResourcePath(c.Args[pos])
		if !pathArg.IsAbs() && pathArg.Depth() == 1 {
			pathArg = path.NewAbsPath(currentConfig, currentProject, pathArg.Keys()[0])
		}
	} else {
		flagKey, err := chooseFlagFromCurrentProject(c)
		if err != nil {
			c.Err(err)
			return perProjectPath{}, nil
		}
		pathArg = path.NewAbsPath(currentConfig, currentProject, flagKey)
	}

	realPath, err := realFlagPath(pathArg)
	if err != nil {
		c.Err(err)
		return perProjectPath{}, nil
	}

	flag, err := getFlag(realPath)
	if err != nil {
		c.Err(err)
		return perProjectPath{}, nil
	}
	return realPath, flag
}

func chooseFlagFromCurrentProject(c *ishell.Context) (string, error) {
	options, err := listFlagKeys(currentConfig, currentProject)
	if err != nil {
		return "", err
	}
	choice := c.MultiChoice(options, "Choose a flag: ")
	if choice < 0 {
		return "", errAborted
	}
	return options[choice], nil
}

func getFlagConfigArg(c *ishell.Context, pos int) (perEnvironmentPath, *ldapi.FeatureFlag) { // nolint:dupl
	var pathArg path.ResourcePath
	if len(c.Args) > pos {
		pathArg = path.ResourcePath(c.Args[pos])
		if !pathArg.IsAbs() && pathArg.Depth() == 1 {
			pathArg = path.NewAbsPath(currentConfig, currentProject, currentEnvironment, pathArg.Keys()[0])
		}
	} else {
		flagKey, err := chooseFlagFromCurrentProject(c)
		if err != nil {
			c.Err(err)
			return perEnvironmentPath{}, nil
		}
		pathArg = path.NewAbsPath(currentConfig, currentProject, flagKey)
	}

	realPath, err := realFlagConfigPath(pathArg)
	if err != nil {
		c.Err(err)
		return perEnvironmentPath{}, nil
	}

	flag, err := getFlag(realPath.PerProjectPath())
	if err != nil {
		c.Err(err)
		return perEnvironmentPath{}, nil
	}
	return realPath, flag
}

func getFlag(p perProjectPath) (*ldapi.FeatureFlag, error) {
	client, err := api.GetClient(getServer(p.Config()))
	if err != nil {
		return nil, err
	}

	auth := api.GetAuthCtx(getToken(p.Config()))
	flag, _, err := client.FeatureFlagsApi.GetFeatureFlag(auth, p.Project(), p.Key(), nil)
	if err != nil {
		return nil, err
	}
	return &flag, err
}

func listFlags(configKey *string, projKey string) ([]ldapi.FeatureFlag, error) {
	auth := api.GetAuthCtx(getToken(configKey))

	client, err := api.GetClient(getServer(configKey))
	if err != nil {
		return nil, err
	}

	flags, _, err := client.FeatureFlagsApi.GetFeatureFlags(auth, projKey, nil)
	if err != nil {
		return nil, err
	}
	return flags.Items, nil
}

func getToken(configKey *string) string {
	var token string
	if configKey != nil {
		token = configFile[*configKey].APIToken
	} else if currentConfig != nil {
		token = configFile[*currentConfig].APIToken
	}
	if token == "" {
		token = currentToken
	}
	return token
}

func getServer(configKey *string) string {
	if configKey != nil {
		return configFile[*configKey].Server
	}
	return currentServer
}

func listFlagKeys(configKey *string, projKey string) ([]string, error) {
	var keys []string
	flags, err := listFlags(configKey, projKey)
	if err != nil {
		return nil, err
	}
	for _, flag := range flags {
		keys = append(keys, flag.Key)
	}
	return keys, nil
}

func showFlag(c *ishell.Context) {
	_, flag := getFlagArg(c, 0)
	if flag == nil {
		c.Err(errors.New("flag not found"))
		return
	}
	renderFlag(c, *flag)
}

func showFlags(c *ishell.Context) {
	configKey := currentConfig
	projectKey := currentProject

	if len(c.Args) > 0 {
		p := path.ResourcePath(c.Args[0])
		switch {
		case p.Depth() == 2:
			_, flag := getFlagArg(c, 0)
			if flag == nil {
				c.Println("Flag not found")
				return
			}
			renderFlag(c, *flag)
			return
		case p.Depth() == 1:
			realPath, err := realProjPath(p)
			if err != nil {
				c.Err(err)
				return
			}
			configKey = realPath.Config()
			projectKey = realPath.Key()
		default:
			c.Err(errors.New("invalid path to project or flag"))
			return
		}
	}

	flags, err := listFlags(configKey, projectKey)
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
		printJSON(c, flag)
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
	table.SetHeader([]string{"Environment", "On", "Last Modified", "Rollout"})
	for envKey, envStatus := range flag.Environments {
		row := []string{envKey, fmt.Sprintf("%v", envStatus.On), time.Unix(envStatus.LastModified/1000, 0).Format("2006/01/02 15:04")}
		if envStatus.Fallthrough_ != nil && envStatus.Fallthrough_.Rollout != nil {
			var rollout []string
		NextVariation:
			for i := range flag.Variations {
				for variationNum, v := range envStatus.Fallthrough_.Rollout.Variations {
					if variationNum == i {
						rollout = append(rollout, fmt.Sprintf("%2.2f%%", float64(v.Weight)/1000.0))
						continue NextVariation
					}
				}
				rollout = append(rollout, "-")
			}
			row = append(row, strings.Join(rollout, "/"))
		} else {
			row = append(row, "")
		}
		table.Append(row)
	}
	table.Render()
	c.Println(buf.String())
}

func createToggleFlag(c *ishell.Context) {
	var name string
	var p perProjectPath
	switch len(c.Args) {
	case 0:
		c.Print("Key: ")
		key := c.ReadLine()
		c.Print("Name: ")
		name = c.ReadLine()
		p = perProjectPath{path.NewAbsPath(currentConfig, currentProject, key)}
	case 1, 2:
		bp := path.ResourcePath(c.Args[0])
		if !bp.IsAbs() && bp.Depth() == 1 {
			bp = path.NewAbsPath(currentConfig, currentProject, bp.Keys()[0])
			fmt.Printf("Creating flag: %s", bp)
		}
		if bp.Depth() != 2 {
			c.Err(errors.New("invalid path"))
			return
		}
		if len(c.Args) > 1 {
			name = c.Args[1]
		}
		p = perProjectPath{bp}
	}
	if name == "" {
		name = p.Key()
	}
	var t, f interface{}
	t = true
	f = false
	client, err := api.GetClient(getServer(p.Config()))
	if err != nil {
		c.Err(err)
		return
	}

	auth := api.GetAuthCtx(getToken(p.Config()))
	flag, _, err := client.FeatureFlagsApi.PostFeatureFlag(auth, p.Project(), ldapi.FeatureFlagBody{
		Name:       name,
		Key:        p.Key(),
		Variations: []ldapi.Variation{{Value: &t}, {Value: &f}},
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
	flagPath, flag := getFlagArg(c, 0)
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

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flag.Key, *patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println("Updated flag")
}

func addTag(c *ishell.Context) {
	flagPath, flag := getFlagArg(c, 0)
	tag := c.Args[1]
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "add",
		Path:  "/tags/-",
		Value: interfacePtr(tag),
	}}

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flag.Key, patchComment)
	if err != nil {
		c.Err(err)
	}

}

func removeTag(c *ishell.Context) {
	flagPath, flag := getFlagArg(c, 0)
	tag := c.Args[1]
	index := -1
	for i, tagA := range flag.Tags {
		if tag == tagA {
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

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))

	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flagPath.Key(), patchComment)
	if err != nil {
		c.Err(err)
	}

}

func on(c *ishell.Context) {
	flagPath, flag := getFlagConfigArg(c, 0)
	if flag == nil {
		return
	}
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/on", flagPath.Environment()),
		Value: interfacePtr(true),
	}}

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flagPath.Key(), patchComment)
	if err != nil {
		c.Err(err)
	}
}

func rollout(c *ishell.Context) {
	flagPath, flag := getFlagConfigArg(c, 0)
	var patchComment ldapi.PatchComment

	if flag == nil {
		c.Err(errors.New("flag not found"))
		return
	}

	var variations []ldapi.WeightedVariation
	for i, v := range flag.Variations {
		index := i
		var percent float64
		var err error
		if len(c.Args) > i+1 {
			parts := strings.Split(c.Args[i+1], ":")
			if len(parts) > 1 {
				index, err = strconv.Atoi(parts[0])
				if err != nil {
					c.Err(err)
					return
				}
			}
			percent, err = strconv.ParseFloat(parts[len(parts)-1], 64)
			if err != nil {
				c.Err(err)
				return
			}
		} else {
			for {
				c.Printf("Enter rollout %% for variation %d (%v): ", i, *v.Value)
				value, err := c.ReadLineErr()
				if err != nil {
					c.Err(err)
					return
				}
				percent, err = strconv.ParseFloat(value, 64)
				if err != nil {
					c.Println("Value must be number.  Try again.")
					continue
				}
				break
			}
		}
		weight := int32(1000.0 * percent)
		variations = append(variations, ldapi.WeightedVariation{Variation: int32(index), Weight: weight})
	}

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	originalFlag, _, err := client.FeatureFlagsApi.GetFeatureFlag(auth, flagPath.Project(), flagPath.Key(), nil)
	if err != nil {
		c.Err(err)
		return
	}

	var patches []ldapi.PatchOperation
	originalFallthrough := originalFlag.Environments[currentEnvironment].Fallthrough_
	if originalFallthrough.Rollout == nil {
		patches = append(patches, ldapi.PatchOperation{
			Op:   "remove",
			Path: fmt.Sprintf("/environments/%s/fallthrough/variation", flagPath.Environment()),
		})
	}

	patches = append(patches, ldapi.PatchOperation{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/fallthrough/rollout", flagPath.Environment()),
		Value: interfacePtr(ldapi.Rollout{Variations: variations}),
	})

	patchComment.Patch = patches

	patchedFlag, _, err := client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flagPath.Key(), patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	final := patchedFlag.Environments[flagPath.Environment()].Fallthrough_.Rollout

	if renderJSON(c) {
		printJSON(c, final)
		return
	}

	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Index", "Weight"})
	for _, v := range final.Variations {
		table.Append([]string{strconv.Itoa(int(v.Variation)), fmt.Sprintf("%2.2f%%", float64(v.Weight)/1000.0)})
	}
	table.Render()
	c.Print(buf.String())
}

func rolloutCompleter(args []string) (completions []string) {
	if len(args) == 0 {
		return nonFinalCompleter(flagCompleter)(args)
	}

	client, err := api.GetClient(getServer(currentConfig))
	if err != nil {
		return nil
	}
	auth := api.GetAuthCtx(getToken(currentConfig))

	currentFlag, _, err := client.FeatureFlagsApi.GetFeatureFlag(auth, currentProject, args[0], nil)
	if err != nil {
		return nil
	}

	for i, v := range currentFlag.Variations {
		name := v.Name
		if name == "" {

			data, err := json.Marshal(v.Value)
			if err != nil {
				continue
			}
			name = string(data)
		}
		completions = append(completions, fmt.Sprintf(`%d:"%s":`, i, name))
	}

	return completions
}

func fallthru(c *ishell.Context) {
	flagPath, flag := getFlagConfigArg(c, 0)
	var patchComment ldapi.PatchComment

	if flag == nil {
		c.Err(errors.New("flag not found"))
		return
	}

	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))

	originalFlag, _, err := client.FeatureFlagsApi.GetFeatureFlag(auth, flagPath.Project(), flag.Key, nil)
	if err != nil {
		c.Err(err)
		return
	}

	var value int
	if len(c.Args) > 1 {
		parts := strings.SplitN(c.Args[1], ":", 2)
		value, err = strconv.Atoi(parts[0])
		if err != nil {
			c.Err(err)
		}
	} else {
		var options []string
		for _, v := range originalFlag.Variations {
			name := v.Name
			if name == "" {
				data, err := json.Marshal(v.Value)
				if err != nil {
					c.Err(err)
					return
				}
				name = string(data)
			}
			options = append(options, name)
		}
		value = c.MultiChoice(options, "Choose a fallthrough variation: ")
		if value < 0 {
			c.Err(errors.New("unknown choice"))
			return
		}
	}

	var patches []ldapi.PatchOperation
	originalFallthrough := originalFlag.Environments[currentEnvironment].Fallthrough_
	if originalFallthrough.Rollout != nil {
		patches = append(patches, ldapi.PatchOperation{
			Op:   "remove",
			Path: fmt.Sprintf("/environments/%s/fallthrough/rollout", flagPath.Environment()),
		})
	}

	patches = append(patches, ldapi.PatchOperation{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/fallthrough/variation", flagPath.Environment()),
		Value: interfacePtr(value),
	})

	patchComment.Patch = patches

	patchedFlag, _, err := client.FeatureFlagsApi.PatchFeatureFlag(auth, currentProject, flag.Key, patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	final := patchedFlag.Environments[currentEnvironment].Fallthrough_.Variation
	printJSON(c, final)
}

func fallthruCompleter(args []string) (completions []string) {
	if len(args) == 0 {
		return nonFinalCompleter(flagCompleter)(args)
	}

	if len(args) > 2 {
		return nil
	}

	client, err := api.GetClient(getServer(currentConfig))
	if err != nil {
		return nil
	}
	auth := api.GetAuthCtx(getToken(currentConfig))

	currentFlag, _, err := client.FeatureFlagsApi.GetFeatureFlag(auth, currentProject, args[0], nil)
	if err != nil {
		return nil
	}

	for i, v := range currentFlag.Variations {
		name := v.Name
		if name == "" {
			data, err := json.Marshal(v.Value)
			if err != nil {
				continue
			}
			name = string(data)
		}
		completions = append(completions, fmt.Sprintf(`%d:"%s"`, i, name))
	}

	return completions
}

func off(c *ishell.Context) {
	flagPath, flag := getFlagConfigArg(c, 0)
	var patchComment ldapi.PatchComment
	patchComment.Patch = []ldapi.PatchOperation{{
		Op:    "replace",
		Path:  fmt.Sprintf("/environments/%s/on", flagPath.Environment()),
		Value: interfacePtr(false),
	}}
	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, flagPath.Project(), flag.Key, patchComment)
	if err != nil {
		c.Err(err)
	}
}

// add/remove tag, toggle on/off

func createFlag(c *ishell.Context) {
	// uhhh
	createToggleFlag(c)
}

func deleteFlag(c *ishell.Context) {
	flagPath, flag := getFlagArg(c, 0)
	if flag == nil {
		c.Err(errNotFound)
		return
	}

	if !confirmDelete(c, "flag key", flag.Key) {
		return
	}
	client, err := api.GetClient(getServer(flagPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(flagPath.Config()))
	_, err = client.FeatureFlagsApi.DeleteFeatureFlag(auth, flagPath.Project(), flag.Key)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println("flag was deleted")
}

func interfacePtr(i interface{}) *interface{} {
	return &i
}
