package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	ApiToken           string `json:"apiToken"`
	Server             string `json:"server"`
	DefaultProject     string `json:"defaultProject"`
	DefaultEnvironment string `json:"DefaultEnvironment"`
}

func ReadConfig() (Config, error) {
	var config Config
	homedir := os.Getenv("HOME")
	// TODO path separator
	configFileName := homedir + "/.config/ldc.json"
	configFile, err := os.Open(configFileName)
	defer configFile.Close()
	if err == nil {
		configJson, _ := ioutil.ReadAll(configFile)
		err = json.Unmarshal(configJson, &config)
		if err != nil {
			return config, err
		}
	} else if os.IsNotExist(err) {
		// create the file if it doesn't exist
		fmt.Printf("Could not find config file, creating template at %s\n", configFileName)
		configFile, err := os.Create(configFileName)
		if err != nil {
			return config, err
		}
		bytes, err := json.MarshalIndent(config, "", "    ")
		if err != nil {
			return config, err
		}
		_, err = configFile.Write(bytes)
		if err != nil {
			return config, err
		}
	}
	return config, err
}
