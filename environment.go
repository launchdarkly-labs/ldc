package main

import (
	"bytes"
	"ldc/api"
	"ldc/api/swagger"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

func AddEnvironmentCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name:    "environments",
		Aliases: []string{"environment", "env", "envs", "e"},
		Help:    "list and operate on environments",
		Func:    listEnvironmentsTable,
	}
	root.AddCmd(&ishell.Cmd{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Help:    "list environments",
		Func:    listEnvironmentsTable,
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

func listEnvironmentsP(projectKey string) []swagger.Environment {
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, projectKey)
	if err != nil {
		panic(err)
	}
	return project.Environments
}

func listEnvironments() []swagger.Environment {
	// TODO other project options
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, api.CurrentProject)
	if err != nil {
		panic(err)
	}
	return project.Environments
}

func listEnvironmentKeysP(project string) []string {
	//TODO errors
	var keys []string
	environments := listEnvironmentsP(project)
	for _, environment := range environments {
		keys = append(keys, environment.Key)
	}
	return keys
}
func listEnvironmentKeys() []string {
	//TODO errors
	var keys []string
	environments := listEnvironments()
	for _, environment := range environments {
		keys = append(keys, environment.Key)
	}
	return keys
}

func listEnvironmentsTable(c *ishell.Context) {
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, api.CurrentProject)
	if err != nil {
		panic(err)
	}
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

func environmentCompleterP(project string, args []string) []string {
	var completions []string
	// TODO caching?
	for _, key := range listEnvironmentKeysP(project) {
		// fuzzy?
		if len(args) == 0 || strings.HasPrefix(key, args[0]) {
			completions = append(completions, key)
		}
	}
	return completions
}

func environmentCompleter(args []string) []string {
	var completions []string
	// TODO caching?
	for _, key := range listEnvironmentKeys() {
		// fuzzy?
		if len(args) == 0 || strings.HasPrefix(key, args[0]) {
			completions = append(completions, key)
		}
	}
	return completions
}

func getEnvironmentArg(c *ishell.Context) *swagger.Environment {
	environments := listEnvironments()
	var foundEnvironment *swagger.Environment
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
		options := listEnvironmentKeys()
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
		c.Println("Please supply at least a key for the new environment")
	case 1:
		key = c.Args[0]
		name = key
	case 2:
		key = c.Args[0]
		name = c.Args[1]
	}
	_, err := api.Client.EnvironmentsApi.PostEnvironment(api.Auth, api.CurrentProject, swagger.EnvironmentBody{Key: key, Name: name, Color: "000000"})
	if err != nil {
		panic(err)
	}
	c.Printf("Created environment %s\n", key)
	c.Printf("Switching to environment %s\n", key)
	api.CurrentEnvironment = key
}

func deleteEnvironment(c *ishell.Context) {
	environment := getEnvironmentArg(c)
	if environment != nil {
		_, err := api.Client.EnvironmentsApi.DeleteEnvironment(api.Auth, api.CurrentProject, environment.Key)
		if err != nil {
			panic(err)
		}
		c.Printf("Deleted environment %s\n", environment.Key)
	}
}
