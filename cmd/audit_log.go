package cmd

import (
	"bytes"
	"strings"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"

	"github.com/launchdarkly/ldc/api"
)

func AddAuditLogCommands(shell *ishell.Shell) {
	root := &ishell.Cmd{
		Name: "log",
		Help: "search audit log entries",
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

				table.Append([]string{time.Unix(entry.Date/1000, 0).Format("2006/01/02 15:04:05"), entry.Title})
			}
			table.SetRowLine(true)
			table.Render()
			if buf.Len() > 1000 {
				c.ShowPaged(buf.String())
			} else {
				c.Println(buf.String())
			}
		},
	}

	shell.AddCmd(root)

}
