package cmd

import (
	"bytes"
	"errors"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"

	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/launchdarkly/ldc/api"
)

func AddProjectCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name:    "projects",
		Aliases: []string{"project"},
		Help:    "showFlags and operate on projects",
		Func:    listProjectsTable,
	}
	root.AddCmd(&ishell.Cmd{
		Name: "showFlags",
		Help: "showFlags projects",
		Func: listProjectsTable,
	})
	root.AddCmd(&ishell.Cmd{
		Name:    "create",
		Aliases: []string{"new"},
		Help:    "create a project: project create key [name]",
		Func:    createProject,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "delete",
		Aliases:   []string{"remove"},
		Help:      "delete a project: project delete key",
		Completer: projectCompleter,
		Func:      deleteProject,
	})

	root.AddCmd(&ishell.Cmd{
		Name:      "switch",
		Aliases:   []string{"select"},
		Help:      "switch the current project",
		Completer: projectCompleter,
		Func: func(c *ishell.Context) {
			foundProject := getProjectArg(c)
			if foundProject != nil {
				switchToProject(c, foundProject)
			}
		},
	})

	shell.AddCmd(root)
}

func listProjects() []ldapi.Project {
	projects, _, err := api.Client.ProjectsApi.GetProjects(api.Auth)
	if err != nil {
		panic(err)
	}
	return projects.Items
}

func listProjectKeys() []string {
	//TODO errors
	var keys []string
	projects, _, _ := api.Client.ProjectsApi.GetProjects(api.Auth)
	for _, project := range projects.Items {
		keys = append(keys, project.Key)
	}
	return keys
}

func listProjectsTable(c *ishell.Context) {
	projects := listProjects()
	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Key", "Name"})
	for _, project := range projects {
		table.Append([]string{project.Key, project.Name})
	}
	table.SetRowLine(true)
	table.Render()
	if buf.Len() > 1000 {
		c.ShowPaged(buf.String())
	} else {
		c.Print(buf.String())
	}
}

func switchToProject(c *ishell.Context, project *ldapi.Project) {
	c.Printf("Switching to project %s\n", project.Key)
	api.CurrentProject = project.Key

	if len(project.Environments) == 0 {
		c.Println("This project has no environments")
		api.CurrentEnvironment = ""
	} else {
		environmentKey := project.Environments[0].Key
		c.Printf("Switching to environment %s\n", environmentKey)
		api.CurrentEnvironment = environmentKey
	}
	c.SetPrompt(api.CurrentProject + "/" + api.CurrentEnvironment + "> ")
}

func projectCompleter(args []string) []string {
	var completions []string
	// TODO caching?
	for _, key := range listProjectKeys() {
		// fuzzy?
		if len(args) == 0 || strings.HasPrefix(key, args[0]) {
			completions = append(completions, key)
		}
	}
	return completions
}

func getProjectArg(c *ishell.Context) *ldapi.Project {
	projects := listProjects()
	var foundProject *ldapi.Project
	if len(c.Args) > 0 {
		projectKey := c.Args[0]
		for _, project := range projects {
			if project.Key == projectKey {
				copy := project
				foundProject = &copy
			}
		}
		if foundProject == nil {
			c.Printf("Project %s does not exist\n", projectKey)
		}
	} else {
		// TODO LOL
		options := listProjectKeys()
		choice := c.MultiChoice(options, "Choose a project")
		foundProject = &projects[choice]
	}
	return foundProject
}

func createProject(c *ishell.Context) {
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
	_, err := api.Client.ProjectsApi.PostProject(api.Auth, ldapi.ProjectBody{Key: key, Name: name})
	if err != nil {
		panic(err)
	}
	c.Printf("Created project %s\n", key)
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, key)
	if err != nil {
		panic(err)
	}
	switchToProject(c, &project)
}

func deleteProject(c *ishell.Context) {
	project := getProjectArg(c)
	confirmDelete(c, "project key", project.Key)
	if project != nil {
		_, err := api.Client.ProjectsApi.DeleteProject(api.Auth, project.Key)
		if err != nil {
			panic(err)
		}
		c.Printf("Deleted project %s\n", project.Key)
	}
}

func updateProject(c *ishell.Context) {
	//???
	// this sucks, json patch
	//api.Client.ProjectsApi.PatchProject(api.Auth, "abc"

}
