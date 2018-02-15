package main

import (
	"flag"

	"ldc/api"
	"ldc/config"

	"github.com/abiosoft/ishell"
)

var apiTokenOverride = flag.String("token", "", "API token to use, overrides config file")
var serverOverride = flag.String("server", "https://app.launchdarkly.com/api/v2", "Server to use, overrides config file")

func main() {
	conf, err := config.ReadConfig()
	if err != nil {
		panic("error reading config " + err.Error())
	}

	flag.Parse()

	shell := ishell.New()

	token := config.GetFlagOrConfigValue(apiTokenOverride, conf.ApiToken, "No API token provided, set either via config or flag\n")
	api.SetToken(token)
	//TODO get from config
	server := config.GetFlagOrConfigValue(serverOverride, conf.Server, "No server provided, set either via config or flag\n")
	api.SetServer("http://localhost/api/v2")

	shell.Set("currentProject", conf.DefaultProject)
	shell.Set("currentEnvironment", conf.DefaultEnvironment)

	shell.AddCmd(&ishell.Cmd{
		Name: "pwd",
		Help: "Show current context (api key, project, environment)",
		Func: func(c *ishell.Context) {
			c.Println("Current Server: " + server)
			c.Println("Current API Key: " + token)
			currentProject, _ := c.Get("currentProject").(string)
			c.Println("Current Project: " + currentProject)
			currentEnvironment, _ := c.Get("currentEnvironment").(string)
			c.Println("Current Environment: " + currentEnvironment)
		},
	})

	AddFlagCommands(shell)

	if flag.NArg() > 0 {
		shell.Process(flag.Args()...)
	} else {
		shell.Println("LaunchDarkly CLI v0.0.1")
		shell.Process("pwd")
		shell.Run()
	}
}
