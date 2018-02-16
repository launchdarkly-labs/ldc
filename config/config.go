package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ConfigFile map[string]Config

type Config struct {
	ApiToken           string `json:"apiToken"`
	Server             string `json:"server"`
	DefaultProject     string `json:"defaultProject"`
	DefaultEnvironment string `json:"defaultEnvironment"`
}

func ReadConfig(creds string) (Config, error) {
	var config Config
	homedir := os.Getenv("HOME")
	// TODO path separator
	configFileName := homedir + "/.config/ldc.json"
	configFile, err := os.Open(configFileName)
	defer configFile.Close()
	if err == nil {
		configJson, _ := ioutil.ReadAll(configFile)
		var allConfigs ConfigFile
		err = json.Unmarshal(configJson, &allConfigs)
		if err == nil {
			return allConfigs[creds], err
		}
	} else if os.IsNotExist(err) {
		// create the file if it doesn't exist
		fmt.Printf("Could not find config file, creating template at %s\n", configFileName)
		configFile, err := os.Create(configFileName)
		if err != nil {
			return config, err
		}
		allConfigs := make(map[string]Config)
		allConfigs["staging"] = config
		bytes, err := json.MarshalIndent(allConfigs, "", "    ")
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

func GetFlagOrConfigValue(flagVal *string, configVal string, errorMsg string) string {
	if flagVal != nil && *flagVal != "" {
		return *flagVal
	}
	if configVal != "" {
		return configVal
	}
	panic(errorMsg)
}
