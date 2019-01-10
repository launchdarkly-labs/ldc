package cmd

import (
	ishell "gopkg.in/abiosoft/ishell.v2"

	"github.com/launchdarkly/ldc/api"
)

func listConfigs() (map[string]Config, error) {
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

func getConfigArg(c *ishell.Context) (string, *Config) {
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

	var foundConfig *Config
	configKey := c.Args[0]
	for c, v := range configs {
		if c == configKey {
			foundConfig = &v
			break
		}
	}
	if foundConfig == nil {
		c.Printf("Config %s does not exist\n", configKey)
	}
	return configKey, foundConfig
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

func setConfig(name string, config Config) {
	currentConfig = name
	api.CurrentProject = config.DefaultProject
	api.CurrentEnvironment = config.DefaultEnvironment
	api.SetToken(config.ApiToken)
	api.SetServer(config.Server)
}
