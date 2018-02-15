package main

import (
	"bytes"
	"ldc/api"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

func AddFlagCommands(shell *ishell.Shell) {

	root := &ishell.Cmd{
		Name:    "flags",
		Aliases: []string{"flag"},
		Help:    "list and operate on flags",
		Func:    list,
	}
	root.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "list flags",
		Func: list,
	})

	shell.AddCmd(root)
}

func list(c *ishell.Context) {
	var project string
	if len(c.Args) > 0 {
		project = c.Args[0]
	} else {
		project, _ = c.Get("currentProject").(string)
		// TODO errors
	}

	flags, _, err := api.Client.FeatureFlagsApi.GetFeatureFlags(api.Auth, project, nil)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%v\n", flag)

	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Key", "Name", "Description"})
	for _, flag := range flags.Items {
		table.Append([]string{flag.Key, flag.Name, flag.Description})
	}
	table.SetRowLine(true)
	table.Render()
	if buf.Len() > 1000 {
		c.ShowPaged(buf.String())
	} else {
		c.Print(buf.String())
	}

}
