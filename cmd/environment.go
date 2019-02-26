package cmd

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/launchdarkly/ldc/cmd/internal/path"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/launchdarkly/ldc/api"
)

func addEnvironmentCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name:    "environments",
		Aliases: []string{"environment", "env", "envs", "e"},
		Help:    "list and operate on environments",
		Func:    showEnvironments,
	}
	root.AddCmd(&ishell.Cmd{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Help:    "list environments",
		Func:    showEnvironments,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "show",
		Help:      "show environment",
		Completer: environmentCompleter,
		Func:      showEnvironment,
	})
	root.AddCmd(&ishell.Cmd{
		Name:    "create",
		Aliases: []string{"new", "c", "add"},
		Help:    "create a environment: environment create key [name]",
		Func:    createEnvironment,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "delete",
		Aliases:   []string{"remove", "d", "del", "rm"},
		Help:      "delete a environment: environment delete key",
		Completer: environmentCompleter,
		Func:      deleteEnvironment,
	})

	shell.AddCmd(root)
}

func listEnvironments(configKey *string, projKey string) ([]ldapi.Environment, error) {
	client, err := api.GetClient(getServer(configKey))
	if err != nil {
		return nil, err
	}
	auth := api.GetAuthCtx(getToken(configKey))
	project, _, err := client.ProjectsApi.GetProject(auth, projKey)
	if err != nil {
		return nil, err
	}
	return project.Environments, nil
}

func listEnvironmentKeys(configKey *string, project string) (keys []string, err error) {
	environments, err := listEnvironments(configKey, project)
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		keys = append(keys, environment.Key)
	}
	return keys, nil
}

func showEnvironments(c *ishell.Context) {
	showEnvironmentsForProject(c, projPath{path.NewAbsPath(currentConfig, currentProject)})
}

func showEnvironmentsForProject(c *ishell.Context, projPath projPath) {
	client, err := api.GetClient(getServer(projPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(projPath.Config()))
	project, _, err := client.ProjectsApi.GetProject(auth, projPath.Key())
	if err != nil {
		c.Err(err)
		return
	}

	if renderJSON(c) {
		printJSON(c, project.Environments)
		return
	}

	c.Println("Environments for " + projPath.String())
	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Key", "Name"})
	for _, environment := range project.Environments {
		table.Append([]string{environment.Key, environment.Name})
	}
	table.SetRowLine(true)
	table.Render()
	if buf.Len() > 1000 {
		c.Err(c.ShowPaged(buf.String()))
	} else {
		c.Print(buf.String())
	}
}

func showEnvironment(c *ishell.Context) {
	envPath, env := getEnvironmentArg(c)

	if env == nil {
		c.Println("Environment not found")
		return
	}

	if renderJSON(c) {
		printJSON(c, env)
		return
	}

	client, err := api.GetClient(getServer(envPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(envPath.Config()))

	project, _, err := client.ProjectsApi.GetProject(auth, currentProject)
	if err != nil {
		c.Err(err)
		return
	}

	environmentKey := c.Args[0]
	buf := bytes.NewBufferString("")
	table := tablewriter.NewWriter(buf)
	for _, environment := range project.Environments {
		if environmentKey == environment.Key {
			table.SetHeader([]string{"Field", "Value"})
			table.Append([]string{"Key", environment.Key})
			table.Append([]string{"Name", environment.Name})
			table.Append([]string{"SDK Key", environment.ApiKey})
			table.Append([]string{"Mobile Key", environment.MobileKey})
			table.Append([]string{"Default TTL", fmt.Sprintf("%.0f", environment.DefaultTtl)})
			table.Append([]string{"Color", environment.Color})
			table.Append([]string{"Secure Mode", fmt.Sprintf("%t", environment.SecureMode)})
			table.Append([]string{"Default Track Events", fmt.Sprintf("%t", environment.DefaultTrackEvents)})
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.Render()
			c.Print(buf.String())
			return
		}
	}
}

func environmentCompleter(args []string) (completions []string) {
	if len(args) > 1 {
		return nil
	}

	completer := path.NewCompleter(getDefaultPath, configLister, projLister, envLister)
	completions, _ = completer.GetCompletions(firstOrEmpty(args))
	return completions
}

func getEnvironmentArg(c *ishell.Context) (perProjectPath, *ldapi.Environment) {
	if len(c.Args) > 0 {
		realPath, err := realEnvPath(c.Args[0])
		if err != nil {
			c.Err(err)
			return perProjectPath{}, nil
		}
		auth := api.GetAuthCtx(getToken(realPath.Config()))
		client, err := api.GetClient(getServer(realPath.Config()))
		if err != nil {
			c.Err(err)
			return perProjectPath{}, nil
		}
		env, _, err := client.EnvironmentsApi.GetEnvironment(auth, realPath.Project(), realPath.Key())
		if err != nil {
			c.Err(err)
			return perProjectPath{}, nil
		}
		return realPath, &env
	}

	env, err := chooseEnvironment(c, currentConfig, currentProject)
	if err != nil {
		c.Err(err)
		return perProjectPath{}, nil
	}
	realPath, err := realEnvPath(env.Key)
	if err != nil {
		c.Err(err)
		return perProjectPath{}, nil
	}
	return realPath, env
}

func chooseEnvironment(c *ishell.Context, config *string, project string) (*ldapi.Environment, error) {
	environments, err := listEnvironments(config, project)
	if err != nil {
		return nil, err
	}
	var environmentKey string
	if len(c.Args) > 0 {
		environmentKey = c.Args[0]
		for _, environment := range environments {
			if environment.Key == environmentKey {
				return &environment, nil // nolint:scopelint // ok because we return
			}
		}
		return nil, errors.New("unknown environment")
	}

	options := keysForEnvironments(environments)
	choice := c.MultiChoice(options, "Choose an environment")
	if choice < 0 {
		return nil, nil
	}
	return &environments[choice], nil
}

func createEnvironment(c *ishell.Context) {
	var name string
	var p perProjectPath
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("please supply at least a key for the new environment"))
		return
	case 1, 2:
		var err error
		p, err = realEnvPath(c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
		if len(c.Args) > 1 {
			name = c.Args[1]
		} else {
			name = p.Key()
		}
	default:
		c.Err(errors.New(`expected arguments are "key [name]""`))
		return
	}
	client, err := api.GetClient(getServer(p.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	key := p.Key()
	auth := api.GetAuthCtx(getToken(p.Config()))
	_, err = client.EnvironmentsApi.PostEnvironment(auth, p.Project(), ldapi.EnvironmentPost{Key: key, Name: name, Color: "000000"})
	if err != nil {
		c.Err(err)
		return
	}
	if isInteractive(c) {
		c.Printf("Created environment %s\n", key)
		c.Printf("Switching to environment %s\n", key)
	}
	currentEnvironment = key
}

func deleteEnvironment(c *ishell.Context) {
	envPath, environment := getEnvironmentArg(c)
	if environment == nil {
		return
	}
	if !confirmDelete(c, "environment key", environment.Key) {
		return
	}
	client, err := api.GetClient(getServer(envPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(envPath.Config()))
	_, err = client.EnvironmentsApi.DeleteEnvironment(auth, envPath.Project(), environment.Key)
	if err != nil {
		c.Err(err)
		return
	}
	c.Printf("Deleted environment %s\n", environment.Key)
}

func keysForEnvironments(environments []ldapi.Environment) (keys []string) {
	for _, e := range environments {
		keys = append(keys, e.Key)
	}
	return keys
}

func realEnvPath(rawPath string) (perProjectPath, error) {
	p := toAbsPath(rawPath, currentConfig, currentProject)
	if p.Depth() != 2 {
		return perProjectPath{}, errors.New("invalid path")
	}
	np, err := path.ReplaceDefaults(p, getDefaultPath, 2)
	if err != nil {
		return perProjectPath{}, err
	}
	return perProjectPath{np}, nil
}
