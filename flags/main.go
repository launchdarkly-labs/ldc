package flags

import (
	"encoding/json"
	"flag"
	"fmt"

	"../api"
)

type Flag struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

func Main(args []string, config Config) {
	f := flag.NewFlagSet("flags", flag.ExitOnError)
	f.String("project", "", "Which project to use")
	f.Parse(args)

	subcommand := f.Arg(0)
	switch subcommand {
	case "list":
		switch f.NArg() {
		case 0: // TODO can't happen, move to project list
			list := api.LinkList{}
			err := api.Call("GET", "/flags", &list)
			if err != nil {
				panic(err)
			}
			fmt.Println("ldc flags list [project]\n")
			fmt.Println("Available Projects:")
			for _, project := range list.Links.Projects {
				fmt.Printf("%v\n", api.Name(project.Href))
			}
		case 1:
			project := 
			list := api.ItemList{}
			err := api.Call("GET", "/flags/"+project, &list)
			if err != nil {
				panic(err)
			}
			fmt.Printf("key\t\tname\t\tdescription\n")
			for _, flagJson := range list.Items {
				flag := Flag{}
				err := json.Unmarshal(flagJson, &flag)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s\t\t%s\t\t%s\n", flag.Key, flag.Name, flag.Description)
			}
		case 3:
			project := f.Arg(1)
			flagKey := f.Arg(2)
			flag := Flag{}
			err := api.Call("GET", "/flags/"+project+"/"+flagKey, &flag)
			if err != nil {
				panic(err)
			}
			fmt.Println("Name: " + flag.Name)
		}
	}
}

func show(envId string, flagId string) {

}
