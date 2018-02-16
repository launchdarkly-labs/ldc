package main

import (
	"flag"
	"os"
	"strings"

	"ldc/api"
	"ldc/config"

	"github.com/abiosoft/ishell"
)

var creds = flag.String("creds", "staging", "Which configured server/apitoken to use")
var apiTokenOverride = flag.String("token", "", "API token to use, overrides config file")
var serverOverride = flag.String("server", "", "Server to use, overrides config file")

func main() {
	flag.Parse()

	conf, err := config.ReadConfig(*creds)
	if err != nil {
		panic("error reading config " + err.Error())
	}

	shell := ishell.New()

	token := config.GetFlagOrConfigValue(apiTokenOverride, conf.ApiToken, "No API token provided, set either via config or flag\n")
	api.SetToken(token)
	//TODO get from config
	server := config.GetFlagOrConfigValue(serverOverride, conf.Server, "No server provided, set either via config or flag\n")
	api.SetServer(server)

	api.CurrentProject = conf.DefaultProject
	api.CurrentEnvironment = conf.DefaultEnvironment

	shell.AddCmd(&ishell.Cmd{
		Name:    "pwd",
		Aliases: []string{"status"},
		Help:    "show current context (api key, project, environment)",
		Func: func(c *ishell.Context) {
			c.Println("Current Server: " + server)
			c.Println("Current API Key: " + token)
			currentProject := api.CurrentProject
			c.Println("Current Project: " + currentProject)
			currentEnvironment := api.CurrentEnvironment
			c.Println("Current Environment: " + currentEnvironment)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "creds",
		Help: "switch to a different api key",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				return
			}
			conf, err := config.ReadConfig(c.Args[0])
			if err != nil {
				c.Err(err)
			} else {
				api.SetToken(conf.ApiToken)
				api.SetServer(conf.Server)
				api.CurrentProject = conf.DefaultProject
				api.CurrentEnvironment = conf.DefaultEnvironment
			}
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "nyan",
		Help: "don't do this",
		Func: func(c *ishell.Context) {
			proc, err := os.StartProcess("/Users/kevinbrackbill/code/go-nyancat/go-nyancat", []string{""}, &os.ProcAttr{
				Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
				Dir:   "/Users/kevinbrackbill/code/go-nyancat/",
			})
			proc.Wait()
			if err != nil {
				panic(err)
			}
		},
	})

	shell.AddCmd(&ishell.Cmd{
		// TODO wanted / syntax but oh well
		Name:    "switch",
		Aliases: []string{"select"},
		Completer: func(args []string) []string {
			switch len(args) {
			case 0:
				// projects
				return projectCompleter(args)
			case 1:
				// env
				return environmentCompleter(args[1:])
			}
			return []string{}
		},
		Func: func(c *ishell.Context) {
			// TODO switch to proj or environment (or saved API key?)
			switch len(c.Args) {
			case 0:
				return
			case 1:
				if strings.Contains(c.Args[0], "/") {
					c.Args = strings.Split(c.Args[0], "/")
					// TODO copy paste
					_, _, err := api.Client.ProjectsApi.GetProject(api.Auth, c.Args[0])
					if err != nil {
						c.Printf("No project %s\n", c.Args[0])
						return
					}
					_, _, err = api.Client.EnvironmentsApi.GetEnvironment(api.Auth, c.Args[0], c.Args[1])
					if err != nil {
						c.Printf("No environment %s\n", c.Args[1])
						return
					}
					api.CurrentProject = c.Args[0]
					api.CurrentEnvironment = c.Args[1]
					c.Printf("Switched to project %s environment %s\n", api.CurrentProject, api.CurrentEnvironment)
					return
				}
				project, _, err := api.Client.ProjectsApi.GetProject(api.Auth, c.Args[0])
				if err != nil {
					c.Printf("No project %s\n", c.Args[0])
					return
				}
				switchToProject(c, &project)
			case 2:
				_, _, err := api.Client.ProjectsApi.GetProject(api.Auth, c.Args[0])
				if err != nil {
					c.Printf("No project %s\n", c.Args[0])
					return
				}
				_, _, err = api.Client.EnvironmentsApi.GetEnvironment(api.Auth, c.Args[0], c.Args[1])
				if err != nil {
					c.Printf("No environment %s\n", c.Args[1])
					return
				}
				api.CurrentProject = c.Args[0]
				api.CurrentEnvironment = c.Args[1]
				c.Printf("Switched to project %s environment %s\n", api.CurrentProject, api.CurrentEnvironment)
			}
		},
	})

	AddFlagCommands(shell)
	AddProjectCommands(shell)
	AddEnvironmentCommands(shell)
	AddAuditLogCommands(shell)

	if flag.NArg() > 0 {
		shell.Process(flag.Args()...)
	} else {
		shell.Println("LaunchDarkly CLI v0.0.1")
		shell.Process("pwd")
		shell.Run()
	}
}
