package cmd

import (
	"bytes"
	"errors"

	"github.com/launchdarkly/ldc/cmd/internal/path"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/launchdarkly/ldc/api"
)

func addProjectCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name:    "projects",
		Aliases: []string{"project"},
		Help:    "list and operate on projects",
		Func:    showProjects,
	}
	root.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "list projects",
		Func: showProjects,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "show",
		Help:      "show project",
		Completer: projectCompleter,
		Func:      showProject,
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

	shell.AddCmd(root)
}

type projPath struct {
	path.ResourcePath
}

func (f projPath) Key() string {
	return f.Keys()[0]
}

func listProjects(configKey *string) ([]ldapi.Project, error) {
	client, err := api.GetClient(getServer(configKey))
	if err != nil {
		return nil, err
	}
	auth := api.GetAuthCtx(getToken(configKey))
	projects, _, err := client.ProjectsApi.GetProjects(auth)
	if err != nil {
		return nil, err
	}
	return projects.Items, nil
}

func listProjectKeys(configKey *string) (keys []string, err error) {
	projects, err := listProjects(configKey)
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		keys = append(keys, project.Key)
	}
	return keys, nil
}

func showProject(c *ishell.Context) {
	projPath, proj := getProjectArg(c)

	if proj == nil {
		c.Println("Project not found")
		return
	}

	if renderJSON(c) {
		printJSON(c, proj)
		return
	}

	showEnvironmentsForProject(c, projPath)
}

func showProjects(c *ishell.Context) {
	projects, err := listProjects(currentConfig)
	if err != nil {
		c.Err(err)
		return
	}

	if renderJSON(c) {
		printJSON(c, projects)
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

func switchToProject(c *ishell.Context, path projPath, project *ldapi.Project) {
	if isInteractive(c) {
		c.Printf("Switching to project %s\n", path.Key())
	}
	currentProject = path.Key()
	currentConfig = path.Config()

	if len(project.Environments) == 0 {
		if isInteractive(c) {
			c.Println("This project has no environments")
		}
		currentEnvironment = ""
	} else {
		environmentKey := project.Environments[0].Key
		if isInteractive(c) {
			c.Printf("Switching to environment %s\n", environmentKey)
		}
		currentEnvironment = environmentKey
	}
	c.SetPrompt(currentProject + "/" + currentEnvironment + "> ")
}

func projectCompleter(args []string) (completions []string) {
	if len(args) > 1 {
		return nil
	}

	completer := path.NewCompleter(getDefaultPath, configLister, projLister)
	completions, _ = completer.GetCompletions(firstOrEmpty(args))
	return completions
}

func getProjectArg(c *ishell.Context) (projPath, *ldapi.Project) {
	var err error
	var realPath projPath
	if len(c.Args) > 0 {
		realPath, err = realProjPath(c.Args[0])
		if err != nil {
			c.Err(err)
			return projPath{}, nil
		}
		auth := api.GetAuthCtx(getToken(realPath.Config()))
		client, err := api.GetClient(getServer(realPath.Config()))
		if err != nil {
			c.Err(err)
			return projPath{}, nil
		}
		proj, _, err := client.ProjectsApi.GetProject(auth, realPath.Key())
		if err != nil {
			c.Err(err)
			return projPath{}, nil
		}
		return realPath, &proj
	}

	proj, err := chooseProject(c, currentConfig)
	if err != nil {
		c.Err(err)
		return projPath{}, nil
	}
	realPath, err = realProjPath(proj.Key)
	if err != nil {
		c.Err(err)
		return projPath{}, nil
	}
	return realPath, proj
}

func chooseProject(c *ishell.Context, config *string) (*ldapi.Project, error) {
	projects, err := listProjects(config)
	if err != nil {
		return nil, err
	}
	if len(c.Args) > 0 {
		projectKey := c.Args[0]
		for _, project := range projects {
			if project.Key == projectKey {
				return &project, nil // nolint:scopelint // ok because we return immediately
			}
		}
		return nil, errNotFound
	}

	options := keysForProjects(projects)
	choice := c.MultiChoice(options, "Choose a project")
	if choice < 0 {
		return nil, nil
	}

	return &projects[choice], nil
}

func createProject(c *ishell.Context) {
	var name string
	var p projPath
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("please supply at least a key for the new environment"))
		return
	case 1, 2:
		var err error
		p, err = realProjPath(c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
		if p.Depth() != 1 {
			c.Err(errors.New("invalid path"))
		}
		if len(c.Args) > 1 {
			name = c.Args[1]
		} else {
			name = p.Key()
		}
	default:
		c.Err(errors.New(`expected arguments are "key [name]"`))
		return
	}
	// TODO: openapi should be updated to return the new project
	client, err := api.GetClient(getServer(p.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(p.Config()))

	if _, err := client.ProjectsApi.PostProject(auth, ldapi.ProjectBody{Key: p.Key(), Name: name}); err != nil {
		c.Err(err)
		return
	}
	if !renderJSON(c) {
		c.Printf("Created project %s\n", p.Key())
	}
	project, _, err := client.ProjectsApi.GetProject(auth, p.Key())
	if err != nil {
		c.Err(err)
		return
	}
	switchToProject(c, p, &project)
	if renderJSON(c) {
		printJSON(c, project)
		return
	}
}

func deleteProject(c *ishell.Context) {
	projPath, project := getProjectArg(c)
	if project == nil {
		c.Err(errNotFound)
		return
	}
	if !confirmDelete(c, "project key", project.Key) {
		return
	}
	client, err := api.GetClient(getServer(projPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(projPath.Config()))
	_, err = client.ProjectsApi.DeleteProject(auth, projPath.Key())
	if err != nil {
		c.Err(err)
		return
	}
	if isInteractive(c) {
		c.Printf(`Project "%s" was deleted\n`, project.Key)
	}
}

func keysForProjects(projects []ldapi.Project) (keys []string) {
	for _, e := range projects {
		keys = append(keys, e.Key)
	}
	return keys
}

func realProjPath(rawPath string) (projPath, error) {
	p := toAbsPath(rawPath, currentConfig)
	if p.Depth() != 1 {
		return projPath{}, errors.New("invalid path")
	}
	np, err := path.ReplaceDefaults(p, getDefaultPath, 1)
	if err != nil {
		return projPath{}, err
	}
	return projPath{np}, nil
}
