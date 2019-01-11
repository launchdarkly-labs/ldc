package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/launchdarkly/ldc/api"
)

func AddProjectCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name:    "projects",
		Aliases: []string{"project"},
		Help:    "list and operate on projects",
		Func:    listProjectsTable,
	}
	root.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "list projects",
		Func: listProjectsTable,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "show",
		Help:      "show project",
		Completer: projectCompleter,
		Func:      listProjectsTable,
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

func listProjects() ([]ldapi.Project, error) {
	projects, _, err := api.Client.ProjectsApi.GetProjects(api.Auth)
	if err != nil {
		return nil, err
	}
	return projects.Items, nil
}

func listProjectKeys() ([]string, error) {
	var keys []string
	projects, _, err := api.Client.ProjectsApi.GetProjects(api.Auth)
	if err != nil {
		return nil, err
	}
	for _, project := range projects.Items {
		keys = append(keys, project.Key)
	}
	return keys, nil
}

func listProjectsTable(c *ishell.Context) {
	projects, err := listProjects()
	if err != nil {
		c.Err(err)
		return
	}
	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Key", "Name"})
	for _, project := range projects {
		table.Append([]string{project.Key, project.Name})
	}
	table.Render()
	if buf.Len() > 1000 {
		c.Err(c.ShowPaged(buf.String()))
	} else {
		c.Print(buf.String())
	}
}

func switchToProject(c *ishell.Context, project *ldapi.Project) {
	if isInteractive(c) {
		c.Printf("Switching to project %s\n", project.Key)
	}
	api.CurrentProject = project.Key

	if len(project.Environments) == 0 {
		if isInteractive(c) {
			c.Println("This project has no environments")
		}
		api.CurrentEnvironment = ""
	} else {
		environmentKey := project.Environments[0].Key
		if isInteractive(c) {
			c.Printf("Switching to environment %s\n", environmentKey)
		}
		api.CurrentEnvironment = environmentKey
	}
	c.SetPrompt(api.CurrentProject + "/" + api.CurrentEnvironment + "> ")
}

var projectCompleter = makeCompleter(emptyOnError(listProjectKeys))

func getProjectArg(c *ishell.Context) *ldapi.Project {
	projects, err := listProjects()
	if err != nil {
		c.Err(err)
		return nil
	}
	if len(c.Args) > 0 {
		projectKey := c.Args[0]
		for _, project := range projects {
			if project.Key == projectKey {
				return &project // nolint:scopelint // ok because we return immediately
			}
		}
		c.Err(fmt.Errorf("Project %s does not exist\n", projectKey))
		return nil
	}

	options, err := listProjectKeys()
	if err != nil {
		c.Err(err)
		return nil
	}

	choice := c.MultiChoice(options, "Choose a project")
	if choice < 0 {
		return nil
	}

	return &projects[choice]
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
	// TODO: openapi should be updated to return the new project
	if _, err := api.Client.ProjectsApi.PostProject(api.Auth, ldapi.ProjectBody{Key: key, Name: name}); err != nil {
		c.Err(err)
		return
	}
	if !renderJson(c) {
		c.Printf("Created project %s\n", key)
	}
	project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, key)
	if err != nil {
		c.Err(err)
		return
	}
	switchToProject(c, &project)
	if renderJson(c) {
		data, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			c.Err(err)
			return
		}
		c.Println(string(data))
		return
	}
}

func deleteProject(c *ishell.Context) {
	project := getProjectArg(c)
	if project == nil {
		c.Err(fmt.Errorf("project does not exist"))
		return
	}
	if !confirmDelete(c, "project key", project.Key) {
		return
	}
	_, err := api.Client.ProjectsApi.DeleteProject(api.Auth, project.Key)
	if err != nil {
		c.Err(err)
		return
	}
	if isInteractive(c) {
		c.Printf("Deleted project %s\n", project.Key)
	}
}
