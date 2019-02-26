package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/launchdarkly/ldc/cmd/internal/path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	ishell "gopkg.in/abiosoft/ishell.v2"
)

type config struct {
	// APIToken is the authorization token
	APIToken string // `json:"apiToken"`
	// Server is the host url (i.e. https://app.launchdarkly.com)
	Server string // `json:"server,omitempty"`
	// DefaultProject is the initial project to use
	DefaultProject string // `json:"defaultProject"`
	// DefaultEnvironment is the initial environment to use
	DefaultEnvironment string // `json:"defaultEnvironment"`
}

var configFile map[string]config
var configViper *viper.Viper

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	configViper = viper.New()
	if configFileName != "" {
		// Use config file from the flag.
		configViper.SetConfigFile(configFileName)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		configViper.AddConfigPath(home)
		configViper.AddConfigPath(filepath.Join(home, ".config"))
		configViper.AddConfigPath(".")
		configViper.SetConfigName("ldc")
	}

	reloadConfigFile()
}

func reloadConfigFile() {
	configFile = nil
	if err := configViper.ReadInConfig(); err == nil {
		if err := configViper.Unmarshal(&configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse config file: %s", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error loading config file: %s\n", err)
	}
}

func addConfigCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name: "configs",
		Help: "Update configurations",
	}
	root.AddCmd(&ishell.Cmd{
		Name:      "set",
		Help:      "Change configuration",
		Completer: configCompleter,
		Func:      selectConfig,
	})
	root.AddCmd(&ishell.Cmd{
		Name: "add",
		Help: "add config <config name> <api token> <project> <environment> [server url]",
		Func: addConfig,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "rename",
		Aliases:   []string{"rn", "mv"},
		Help:      "rename config <config name> <new name>",
		Completer: configCompleter,
		Func:      renameConfig,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "edit",
		Aliases:   []string{"update"},
		Help:      "update config: <config name> <api token> <project> <environment> [server url]",
		Completer: configCompleter,
		Func:      updateConfig,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "rm",
		Aliases:   []string{"remove", "delete", "del"},
		Help:      "remove config: <config name>",
		Completer: configCompleter,
		Func:      removeConfig,
	})
	shell.AddCmd(root)
}

func listConfigs() (map[string]config, error) {
	return configFile, nil
}

func listConfigKeys() ([]string, error) {
	var keys []string
	configs, err := listConfigs()
	if err != nil {
		return nil, err
	}
	for c := range configs {
		keys = append(keys, c)
	}
	return keys, nil
}

func configCompleter(args []string) []string {
	keys, err := listConfigKeys()
	if err != nil {
		return nil
	}
	completions := withPrefix(keys, toPrefix(args))
	return completions
}

func getConfigArg(c *ishell.Context) (string, *config) {
	configs, err := listConfigs()
	if err != nil {
		c.Err(err)
		return "", nil
	}

	if len(c.Args) == 0 {
		options, err := listConfigKeys()
		if err != nil {
			c.Err(err)
			return "", nil
		}
		choice := c.MultiChoice(options, "Choose a config")
		if choice < 0 {
			return "", nil
		}
		config := configs[options[choice]]
		return options[choice], &config
	}

	configKey := c.Args[0]
	for c, v := range configs {
		if c == configKey {
			return c, &v // nolint:scopelint // ok because we break
		}
	}

	c.Err(errors.New("config does not exist"))
	return "", nil
}

func selectConfig(c *ishell.Context) {
	name, config := getConfigArg(c)
	if config == nil {
		c.Println("Config not found. Settings unchanged.")
		return
	}
	setConfig(name, *config)
	printCurrentSettings(c)
}

func setConfig(name string, config config) {
	currentConfig = &name
	currentServer = config.Server
	if config.Server == "" {
		currentServer = viper.GetString("server")
	}
	currentProject = config.DefaultProject
	currentEnvironment = config.DefaultEnvironment
}

func updateConfig(c *ishell.Context) {
	if len(c.Args) > 1 && len(c.Args) < 4 {
		c.Err(errTooFewArgs)
		return
	}

	if len(c.Args) > 5 {
		c.Err(errTooManyArgs)
		return
	}

	name, config := getConfigArg(c)

	if config == nil {
		c.Println("Config not found. Settings unchanged.")
		return
	}

	newConfig := config

	if len(c.Args) <= 1 {
		c.Printf(`API Token (default "%s"): `, config.APIToken)
		val, err := c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.APIToken = ifNotBlank(val, config.APIToken)

		c.Printf(`Default Project (default "%s"): `, config.DefaultProject)
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.DefaultProject = ifNotBlank(val, config.DefaultProject)

		c.Printf(`Default Environment (default "%s"): `, config.DefaultEnvironment)
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.DefaultEnvironment = ifNotBlank(val, config.DefaultEnvironment)

		c.Printf(`Server (leave blank for "%s" or "-" for default): `, config.Server)
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		if val == "-" {
			val = ""
		}
		newConfig.Server = ifNotBlank(val, config.Server)
	} else {
		newConfig.APIToken = c.Args[1]
		newConfig.DefaultProject = c.Args[2]
		newConfig.DefaultEnvironment = c.Args[3]
		if len(c.Args) > 4 {
			newConfig.Server = c.Args[4]
		} else {
			newConfig.Server = ""
		}
	}

	configViper.Set(name, newConfig)
	if err := configViper.WriteConfig(); err != nil {
		c.Err(err)
		return
	}
	reloadConfigFile()
	c.Println("configuration updated")
}

func removeConfig(c *ishell.Context) {
	name, config := getConfigArg(c)
	if config == nil {
		c.Err(errors.New("config not found"))
		return
	}
	if !confirmDelete(c, "config", name) {
		return
	}
	configViper.Set(name, nil)
	if err := configViper.WriteConfig(); err != nil {
		c.Err(err)
		return
	}
	reloadConfigFile()
	c.Println("configuration removed")
}

func addConfig(c *ishell.Context) {
	if len(c.Args) > 1 && len(c.Args) < 4 {
		c.Err(errTooFewArgs)
		return
	}

	if len(c.Args) > 5 {
		c.Err(errTooManyArgs)
		return
	}

	var name string
	if len(c.Args) > 0 {
		name = c.Args[0]
	} else {
		name = pickNewConfigName(c)
	}
	if strings.TrimSpace(name) == "" {
		c.Err(errors.New("invalid name"))
		return
	}

	newConfig := config{}
	if len(c.Args) <= 1 {
		c.Printf("API Token: ")
		val, err := c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.APIToken = val

		c.Printf(`Default Project (default "default"): `)
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.DefaultProject = ifNotBlank(val, "default")

		c.Printf(`Default Environment (default "production"): `)
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.DefaultEnvironment = ifNotBlank(val, "production")

		c.Printf("Server (leave blank for default): ")
		val, err = c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return
		}
		newConfig.Server = ifNotBlank(val, "")
	} else {
		newConfig.APIToken = c.Args[1]
		newConfig.DefaultProject = c.Args[2]
		newConfig.DefaultEnvironment = c.Args[3]
		if len(c.Args) > 4 {
			newConfig.Server = c.Args[4]
		} else {
			newConfig.Server = ""
		}
	}

	configViper.Set(name, newConfig)
	if err := configViper.WriteConfig(); err != nil {
		c.Err(err)
		return
	}
	reloadConfigFile()
	c.Println("configuration added")
}

func pickNewConfigName(c *ishell.Context) string {
	var name string
	for {
		c.Printf(`New config name: `)
		val, err := c.ReadLineErr()
		if err != nil {
			c.Err(err)
			return ""
		}
		name = strings.TrimSpace(val)
		if name == "" {
			c.Err(errors.New("must not be blank"))
			continue
		}
		if _, exists := configFile[name]; exists {
			c.Err(errors.New("config already exists"))
			continue
		}
		break
	}
	return name
}

func renameConfig(c *ishell.Context) {
	if len(c.Args) > 2 {
		c.Err(errTooFewArgs)
		return
	}

	name, cfg := getConfigArg(c)

	if cfg == nil {
		c.Err(errNotFound)
		return
	}

	var newName string
	if len(c.Args) > 1 {
		newName = c.Args[1]
	} else {
		newName = pickNewConfigName(c)
	}
	if strings.TrimSpace(name) == "" {
		c.Err(errors.New("invalid name"))
		return
	}

	if name == newName {
		return
	}

	if _, exists := configFile[newName]; exists {
		c.Err(errors.New("target already exists"))
		return
	}
	configViper.Set(newName, cfg)
	configViper.Set(name, nil)
	if err := configViper.WriteConfig(); err != nil {
		c.Err(err)
		return
	}
	reloadConfigFile()
	c.Println("configuration renamed")
}

var getDefaultPath = path.DefaultPathSourceFunc(func(configKey *string) (path.ResourcePath, error) {
	project := currentProject
	environment := currentEnvironment
	if configKey == nil {
		configKey = currentConfig
	}
	if configKey != nil {
		config, exists := configFile[*configKey]
		if !exists {
			return "", errors.New("config not found")
		}
		project = config.DefaultProject
		environment = config.DefaultEnvironment
	}
	return path.NewAbsPath(configKey, project, environment), nil
})
