package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/launchdarkly/ldc/api"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ldc",
	Short: "ldc is a command-line api client for LaunchDarkly",
	Run:   RootCmd,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetEnvPrefix("ldc")
		viper.SetConfigName("ldc")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ldc")
	viper.SetConfigName("ldc")

	pflag.String("token", "", "api key (e.g. api-...)")
	pflag.String("server", "https://app.launchdarkly.com/api/v2", "alternate server base url")
	pflag.String("project", "default", "Default project key")
	pflag.String("environment", "default", "Default environment key")
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
}

func RootCmd(cmd *cobra.Command, args []string) {
	shell := ishell.New()

	api.SetToken(viper.GetString("token"))
	api.SetServer(viper.GetString("server"))

	api.CurrentProject = viper.GetString("project")
	api.CurrentEnvironment = viper.GetString("environment")

	shell.SetPrompt(api.CurrentProject + "/" + api.CurrentEnvironment + "> ")

	shell.AddCmd(&ishell.Cmd{
		Name:    "pwd",
		Aliases: []string{"status", "current"},
		Help:    "show current context (api key, project, environment)",
		Func: func(c *ishell.Context) {
			c.Println("Current Server: " + api.CurrentServer)
			printCurrentToken(c)
			c.Println("Current Project: " + api.CurrentProject)
			c.Println("Current Environment: " + api.CurrentEnvironment)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		// TODO wanted / syntax but oh well
		Name:    "switch",
		Help:    "Switch to a given project and environment: switch project [environment]",
		Aliases: []string{"select"},
		Completer: func(args []string) []string {
			switch len(args) {
			case 0:
				// projects
				return projectCompleter(args)
			case 1:
				// env
				return environmentCompleterP(args[0], args[1:])
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
	AddTokenCommand(shell)

	if flag.NArg() > 0 {
		shell.Process(flag.Args()...)
	} else {
		shell.Printf("LaunchDarkly CLI %s\n", Version)
		shell.Process("pwd")
		shell.Run()
	}
}

func AddTokenCommand(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name: "token",
		Help: "set api key",
		Func: func(c *ishell.Context) {
			var token string
			if len(c.Args) == 1 {
				token = c.Args[0]
			}
			if len(c.Args) > 1 {
				c.Err(errors.New("Only one argument, the api key, is allowed"))
				return
			}
			c.Print("API Key: ")
			token = c.ReadPassword()
			api.SetToken(token)
			printCurrentToken(c)
		},
	}
	shell.AddCmd(root)
}

func printCurrentToken(c *ishell.Context) {
	c.Printf("Current API Key: ends in '%s'\n", last4(api.CurrentToken))
}

func last4(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-5:]
}
