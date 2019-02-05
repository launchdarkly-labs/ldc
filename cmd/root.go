package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/launchdarkly/ldc/cmd/internal/path"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	ishell "gopkg.in/abiosoft/ishell.v2"

	"github.com/launchdarkly/ldc/api"
)

var currentConfig *string

var currentServer string
var currentToken string
var currentProject string
var currentEnvironment string

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "ldc",
	Short:            "ldc is a command-line api client for LaunchDarkly",
	PersistentPreRun: preRunCmd,
	Run:              runRootCmd,
}

// rootCmd represents the base command when called without any subcommands
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "start an interactive shell",
	Run:   runShellCmd,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := rootCmd
	rootCmdWithShell := *rootCmd
	rootCmdWithShell.AddCommand(shellCmd)
	foundCmd, _, err := rootCmdWithShell.Find(os.Args[1:])
	if err == nil && foundCmd.Use == "shell" {
		cmd = shellCmd
	}
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	api.Initialize("ldc/" + Version)
	cobra.OnInitialize(initConfig)

	pflag.String("token", "", "api key (e.g. api-...)")
	pflag.String("server", "", "alternate server base url")
	pflag.String("project", "", "Project key")
	pflag.String("environment", "", "Environment key")
	pflag.String("config", "", "Configuration to use")
	pflag.Bool("json", false, "Return json")
	pflag.Bool("debug", false, "Enable debugging")
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ldc")
	viper.SetConfigName("ldc")
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}
}

func preRunCmd(cmd *cobra.Command, args []string) {
	configs, err := listConfigs()
	if err != nil {
		configs = nil
	}

	config := viper.GetString("config")

	// Assume we can use the single config if there is only one
	if config == "" && len(configs) == 1 {
		for name := range configs {
			config = name
			break
		}
	}

	if config != "" {
		found := false
		for name, v := range configs {
			if name == config {
				setConfig(name, v)
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, `Unable to find config "%s"`, config)
			os.Exit(1)
		}
	}

	if viper.IsSet("token") {
		if token := viper.GetString("token"); token != "" {
			currentToken = token
		}
	}
	if viper.IsSet("server") {
		if server := viper.GetString("server"); server != "" {
			currentServer = server
		}
	}
	if viper.IsSet("project") {
		if project := viper.GetString("project"); project != "" {
			currentProject = project
		}
	}
	if viper.IsSet("environment") {
		if env := viper.GetString("environment"); env != "" {
			currentEnvironment = env
		}
	}

	api.Debug = viper.GetBool("debug")
}

func addTokenCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name: "token",
		Help: "set api key",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 1 {
				c.Err(errors.New("only one argument, the api key, is allowed"))
				return
			}

			var token string
			if len(c.Args) == 1 {
				token = c.Args[0]
			} else {
				c.Print("API Key: ")
				token = c.ReadPassword()
			}

			currentToken = token
			printCurrentToken(c)
		},
	}
	shell.AddCmd(root)
}

func createShell(interactive bool) *ishell.Shell {
	shell := ishell.New()
	shell.SetHomeHistoryPath(".ldc_history")

	prompt := fmt.Sprintf("%s/%s> ", currentProject, currentEnvironment)
	if currentConfig != nil {
		prompt = fmt.Sprintf(`[%s] %s`, *currentConfig, prompt)
	}
	shell.SetPrompt(prompt)

	shell.AddCmd(&ishell.Cmd{
		Name:    "pwd",
		Aliases: []string{"status", "current"},
		Help:    "show current context (api key, project, environment)",
		Func:    printCurrentSettings,
	})

	shell.AddCmd(&ishell.Cmd{
		Name:      "json",
		Help:      "set json mode",
		Completer: boolCompleter,
		Func:      setJSONMode,
	})

	shell.CustomCompleter(customCompleter{shell, nil})

	shell.AddCmd(&ishell.Cmd{
		Name:      "switch",
		Help:      "Switch to a given project and environment: switch [//config][/project]<environment>",
		Aliases:   []string{"s", "select"},
		Completer: environmentCompleter,
		Func: func(c *ishell.Context) {
			if len(c.Args) > 1 {
				return
			}
			p := path.ResourcePath(firstOrEmpty(c.Args))
			var dest string
			switch {
			case !p.IsAbs():
				auth := api.GetAuthCtx(getToken(currentConfig))
				client, err := api.GetClient(getServer(currentConfig))
				if err != nil {
					c.Err(err)
					return
				}
				_, _, err = client.EnvironmentsApi.GetEnvironment(auth, currentProject, p.Keys()[0])
				if err != nil {
					c.Printf("No environment \"%s\"\n", p.Keys()[0])
					return
				}
				currentEnvironment = c.Args[0]
				dest += fmt.Sprintf("environment %s", currentEnvironment)
			case p.Depth() == 2:
				configKey := p.Config()
				if configKey == nil {
					configKey = currentConfig
				}
				auth := api.GetAuthCtx(getToken(configKey))
				client, err := api.GetClient(getServer(configKey))
				if err != nil {
					c.Err(err)
					return
				}
				if p.Config() != nil {
					dest = fmt.Sprintf("config %s", *p.Config())
				}

				p := perProjectPath{p}
				_, _, err = client.ProjectsApi.GetProject(auth, p.Project())
				if err != nil {
					c.Printf("No project '%s'\n", p.Project())
					return
				}
				_, _, err = client.EnvironmentsApi.GetEnvironment(auth, p.Project(), p.Key())
				if err != nil {
					c.Printf("No environment '%s'\n", p.Key())
					return
				}
				currentProject = p.Project()
				currentEnvironment = p.Key()
				dest += fmt.Sprintf("project %s, environment %s", currentProject, currentEnvironment)
			default:
				c.Printf("'%s' is not a valid environment\n", p)
				return
			}
			c.Printf("Switched to '%s'\n", dest)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "shell",
		Help: "Run shell",
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "version",
		Help: "Show version",
		Func: func(c *ishell.Context) {
			c.Println(Version)
		},
	})

	addConfigCommands(shell)
	addFlagCommands(shell)
	addProjectCommands(shell)
	addEnvironmentCommands(shell)
	addAuditLogCommands(shell)
	addTokenCommands(shell)
	addGoalCommands(shell)

	isJSON := viper.GetBool("json")
	shell.Set(cJSON, isJSON)
	if !isJSON {
		if configViper.ConfigFileUsed() != "" {
			fmt.Printf("Using config file: %s\n", configViper.ConfigFileUsed())
		}
	}

	shell.Set(cEDITOR, "vi")
	if editor := os.Getenv("cEDITOR"); editor != "" {
		shell.Set(cEDITOR, editor)
	}

	shell.Set(cINTERACTIVE, interactive)
	return shell
}

func runRootCmd(cmd *cobra.Command, args []string) {
	shell := createShell(false)
	if len(args) == 0 {
		_ = cmd.Usage()
		fmt.Print(shell.HelpText())
		os.Exit(0)
	}
	err := shell.Process(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func runShellCmd(cmd *cobra.Command, args []string) {
	shell := createShell(true)
	shell.Printf("LaunchDarkly CLI %s\n", Version)
	_ = shell.Process("pwd")
	shell.Run()
}

func printCurrentToken(c *ishell.Context) {
	c.Printf("Current API Key: ends in '%s'\n", last4(getToken(currentConfig)))
}

func printCurrentSettings(c *ishell.Context) {
	c.Println("Current Config: " + noneIfNil(currentConfig))
	c.Println("Current Server: " + currentServer)
	printCurrentToken(c)
	c.Println("Current Project: " + currentProject)
	c.Println("Current Environment: " + currentEnvironment)
}

func last4(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-5:]
}

var boolOptions = []string{"false", "true"}
var boolCompleter = makeCompleter(func() []string { return boolOptions })

func setJSONMode(c *ishell.Context) {
	var value string
	if len(c.Args) == 1 {
		value = c.Args[0]
		if !containsString(boolOptions, strings.ToLower(value)) {
			c.Println(`Value must be "true" or "false"`)
			return
		}
	} else {
		choice := c.MultiChoice(boolOptions, "Show JSON? ")
		if choice < 0 {
			c.Println("Value unchanged")
			return
		}
		value = boolOptions[choice]
	}
	isJSON := strings.ToLower(value) == "true" || strings.ToLower(value) == "t"
	setJSON(isJSON)
	if isJSON {
		c.Println("JSON enabled")
	} else {
		c.Println("JSON disabled")
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func noneIfNil(s *string) string {
	if s == nil {
		return "<none>"
	}
	return *s
}
