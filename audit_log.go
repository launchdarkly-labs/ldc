package main

import (
	"bytes"
	"fmt"
	"ldc/api"
	"strings"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

func AddAuditLogCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name: "log",
		Help: "Search audit log entries",
		Func: func(c *ishell.Context) {
			options := make(map[string]interface{})
			if len(c.Args) > 0 {
				options["q"] = strings.Join(c.Args, " ")
				options["limit"] = 5
				//options["spec"] = ""
				//options["after"] = 1518163200000
				//options["before"] = time.Now().UnixNano() / int64(time.Millisecond)
			}
			entries, _, err := api.Client.AuditLogApi.GetAuditLogEntries(api.Auth, options)
			if err != nil {
				panic(err)
			}
			buf := bytes.Buffer{}
			table := tablewriter.NewWriter(&buf)
			//table.SetHeader([]string{"Key", "Name"})
			for _, entry := range entries.Items {

				table.Append([]string{time.Unix(entry.Date/1000, 0).Format("2006/1/2 15:04:05"), entry.Title})
			}
			table.SetRowLine(true)
			table.Render()
			fmt.Println(buf.String())
		},
	}

	shell.AddCmd(root)

}
