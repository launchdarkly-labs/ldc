package flags

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"../api"
	"../config"
	"github.com/olekukonko/tablewriter"
)

type Flag struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
}

func Main(args []string, conf config.Config) {
	f := flag.NewFlagSet("flags", flag.ExitOnError)
	projectOverride := f.String("project", "", "Which project to use")
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
			project := config.GetFlagOrConfigValue(projectOverride, conf.DefaultProject, "Need to set a project by flag or config value")
			list := api.ItemList{}
			err := api.Call("GET", "/flags/"+project, &list)
			if err != nil {
				panic(err)
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Key", "Name", "Description"})
			for _, flagJson := range list.Items {
				flag := Flag{}
				err := json.Unmarshal(flagJson, &flag)
				if err != nil {
					panic(err)
				}
				table.Append([]string{flag.Key, flag.Name, flag.Description})
			}
			table.SetRowLine(true)
			table.Render()
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
	default:
		fmt.Println("")
	}
}

func show(envId string, flagId string) {

}
