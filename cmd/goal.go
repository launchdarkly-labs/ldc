package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/launchdarkly/ldc/goal_api"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

var goalCompleter = makeCompleter(listGoalKeys)

func AddGoalsCommands(shell *ishell.Shell) {

	root := &ishell.Cmd{
		Name:    "goals",
		Aliases: []string{"goals"},
		Help:    "list and operate on goals",
		Func:    showGoals,
	}
	root.AddCmd(&ishell.Cmd{
		Name:      "list",
		Help:      "list goals",
		Completer: goalCompleter,
		Func:      showGoals,
	})
	root.AddCmd(&ishell.Cmd{
		Name:    "create",
		Aliases: []string{"new"},
		Help:    "Create new goal",
		Func:    createGoal,
	})
	//root.AddCmd(&ishell.Cmd{
	//	Name:      "attach",
	//	Help:      "attach to flag",
	//	Completer: attachGoalCompleter,
	//	Func:      attachGoal,
	//})
	//root.AddCmd(&ishell.Cmd{
	//	Name:      "detach",
	//	Help:      "detach from flag",
	//	Completer: detachGoalCompleter,
	//	Func:      detachGoal,
	//})
	root.AddCmd(&ishell.Cmd{
		Name:      "edit",
		Help:      "edit a goal's json in a text editor",
		Completer: goalCompleter,
		Func:      editGoal,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "delete",
		Aliases:   []string{"remove"},
		Help:      "Delete a goal",
		Completer: goalCompleter,
		Func:      deleteGoal,
	})

	shell.AddCmd(root)
}

func getGoalArg(c *ishell.Context) *goal_api.Goal {
	goals, _ := goal_api.GetGoals()
	var foundGoal *goal_api.Goal
	var goalKey string
	if len(c.Args) > 0 {
		goalKey = c.Args[0]
		for _, g := range goals {
			if g.Name == goalKey {
				return &g
			}
		}
	} else {
		options := listGoalKeys()
		choice := c.MultiChoice(options, "Choose a goal")
		foundGoal, _ = goal_api.GetGoal(options[choice])
	}
	return foundGoal
}

func listGoalKeys() []string {
	var keys []string
	g, _ := goal_api.GetGoals()
	for _, g := range g {
		keys = append(keys, g.Name)
	}
	return keys
}

func showGoals(c *ishell.Context) {
	if len(c.Args) > 0 {
		goal := getGoalArg(c)
		c.Printf("Name: %s\n", goal.Name)
		c.Printf("Description: %s\n", goal.Description)
		c.Printf("Kind: %s\n", goal.Kind)
		if goal.AttachedFeatureCount > 0 {
			c.Println("Attached Flags:")
			for _, f := range goal.AttachedFeatures {
				c.Printf("    %s[%s] (%s)\n", f.Key, f.Name, boolToCheck(f.On))
			}
		}
	} else {
		goals, _ := goal_api.GetGoals()
		buf := bytes.Buffer{}
		table := tablewriter.NewWriter(&buf)
		table.SetHeader([]string{"Name", "Description", "Kind", "Attached Flags"})
		for _, goal := range goals {
			var attached []string
			for _, f := range goal.AttachedFeatures {
				attached = append(attached, fmt.Sprintf("%s(%s)", f.Key, boolToCheck(f.On)))
			}
			attachedStr := strings.Join(attached, " ")
			table.Append([]string{goal.Name, goal.Description, goal.Kind, attachedStr})
		}
		table.SetRowLine(true)
		table.Render()
		if buf.Len() > 1000 {
			c.ShowPaged(buf.String())
		} else {
			c.Print(buf.String())
		}
	}
}

func editGoal(c *ishell.Context) {
	goal := getGoalArg(c)
	data, _ := json.MarshalIndent(goal, "", "    ")

	patchComment, err := editFile(c, data)
	if err != nil {
		c.Err(err)
		return
	}

	_, err = goal_api.PatchGoal(goal.Id, *patchComment)
	if err != nil {
		c.Err(err)
	} else {
		c.Println("Updated goal")
	}
}

func createGoal(c *ishell.Context) {
	var name string
	var kind string
	if len(c.Args) > 1 {
		name = c.Args[0]
		kind = c.Args[1]
	} else {
		c.Print("Name: ")
		name = c.ReadLine()
		choice := c.MultiChoice(goal_api.AvailableKinds, "Kind: ")
		kind = goal_api.AvailableKinds[choice]
	}
	goal := goal_api.Goal{
		Name: name,
		Kind: kind,
	}
	_, err := goal_api.CreateGoal(goal)
	if err != nil {
		c.Err(err)
	} else {
		c.Println("Created goal")
	}
}

func deleteGoal(c *ishell.Context) {
	goal := getGoalArg(c)

	err := goal_api.DeleteGoal(goal.Id)
	if err != nil {
		c.Err(err)
	} else {
		c.Println("Deleted goal")
	}
}

func boolToCheck(b bool) string {
	if b {
		return "X"
	} else {
		return " "
	}
}
