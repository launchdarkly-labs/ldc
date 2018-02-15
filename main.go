package main

import (
	"flag"
	"fmt"

	"./api"
	"./config"
	"./flags"
)

var apiTokenOverride = flag.String("token", "", "API token to use, overrides config file")

func main() {

	conf, err := config.ReadConfig()
	if err != nil {
		panic("error reading config " + err.Error())
	}

	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	token := config.GetFlagOrConfigValue(apiTokenOverride, conf.ApiToken, "No API token provided, set either via config or flag\n")
	api.SetToken(token)

	//TODO get from config
	api.SetServer("http://localhost/api/v2")

	subcommand := flag.Arg(0)
	switch subcommand {
	case "flag":
		fallthrough
	case "flags":
		flags.Main(flag.Args()[1:], conf)

	default:
		fmt.Printf("Unrecognized subcommand: %s\n", subcommand)
	}
}

func usage() {
	fmt.Println("usage: ldc subcommand")
	flag.PrintDefaults()
}
