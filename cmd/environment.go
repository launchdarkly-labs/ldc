package cmd

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	"github.com/launchdarkly/api-client-go"
	"github.com/launchdarkly/ldc/api"
)

func AddEnvironmentCommands(shell *ishell.Shell) {
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
		Help:      "list environments",
		Completer: environmentCompleter,
		Func:      showEnvironments,
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
	root.AddCmd(&ishell.Cmd{
		Name:      "switch",
		Aliases:   []string{"select", "s", "sel"},
		Help:      "switch the current environment",
		Completer: environmentCompleter,
		Func: func(c *ishell.Context) {
			foundEnvironment := getEnvironmentArg(c)
			if foundEnvironment == nil {
				c.Printf("Environment %s does not exist in the current project\n", foundEnvironment.Key)
			} else {
				c.Printf("Switching to environment %s\n", foundEnvironment.Key)
				api.CurrentEnvironment = foundEnvironment.Key
				c.SetPrompt(api.CurrentProject + "/" + api.CurrentEnvironment + "> ")
			}
		},
	})

	shell.AddCmd(root)
}

func listEnvironmentsForProject(projectKey string) ([]ldapi.Environment, error) {
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, projectKey)
	if err != nil {
		return nil, err
	}
	return project.Environments, nil
}

func listEnvironments() ([]ldapi.Environment, error) {
	// TODO other project options
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, api.CurrentProject)
	if err != nil {
		return nil, err
	}
	return project.Environments, nil
}

func listEnvironmentKeysForProject(project string) ([]string, error) {
	var keys []string
	environments, err := listEnvironmentsForProject(project)
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		keys = append(keys, environment.Key)
	}
	return keys, nil
}

func listEnvironmentKeys() ([]string, error) {
	var keys []string
	environments, err := listEnvironments()
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		keys = append(keys, environment.Key)
	}
	return keys, nil
}

func showEnvironments(c *ishell.Context) {
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, api.CurrentProject)
	if err != nil {
		c.Err(err)
		return
	}
	if len(c.Args) > 0 {
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
				table.SetRowLine(true)
				table.SetAlignment(tablewriter.ALIGN_LEFT)
				table.Render()
				c.Print(buf.String())
				return
			}
		}
		c.Println("Environment not found")
	} else {
		c.Println("Environments for " + project.Name)
		buf := bytes.Buffer{}
		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"Key", "Name"})
		for _, environment := range project.Environments {
			table.Append([]string{environment.Key, environment.Name})
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

var environmentCompleter = makeCompleter(emptyOnError(listEnvironmentKeys))

func getEnvironmentArg(c *ishell.Context) *ldapi.Environment {
	environments, err := listEnvironments()
	if err != nil {
		c.Err(err)
		return nil
	}
	var foundEnvironment *ldapi.Environment
	var environmentKey string
	if len(c.Args) > 0 {
		environmentKey = c.Args[0]
		for _, environment := range environments {
			if environment.Key == environmentKey {
				copy := environment
				foundEnvironment = &copy
			}
		}
	} else {
		// TODO LOL
		options, err := listEnvironmentKeys()
		if err != nil {
			c.Err(err)
			return nil
		}
		choice := c.MultiChoice(options, "Choose an environment")
		foundEnvironment = &environments[choice]
		environmentKey = foundEnvironment.Key
	}
	return foundEnvironment
}

func createEnvironment(c *ishell.Context) {
	var key, name string
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("please supply at least a key for the new environment"))
		return
	case 1:
		key = c.Args[0]
		name = key
	case 2:
		key = c.Args[0]
		name = c.Args[1]
	default:
		c.Err(errors.New("too many arguments.  Expected arguments are: key [name]."))
		return
	}
	_, err := api.Client.EnvironmentsApi.PostEnvironment(api.Auth, api.CurrentProject, ldapi.EnvironmentPost{Key: key, Name: name, Color: "000000"})
	if err != nil {
		c.Err(err)
		return
	}
	c.Printf("Created environment %s\n", key)
	c.Printf("Switching to environment %s\n", key)
	api.CurrentEnvironment = key
}

func deleteEnvironment(c *ishell.Context) {
	environment := getEnvironmentArg(c)
	if environment == nil {
		return
	}
	if !confirmDelete(c, "environment key", environment.Key) {
		return
	}
	if environment != nil {
		_, err := api.Client.EnvironmentsApi.DeleteEnvironment(api.Auth, api.CurrentProject, environment.Key)
		if err != nil {
			c.Err(err)
			return
		}
		c.Printf("Deleted environment %s\n", environment.Key)
	}
}
