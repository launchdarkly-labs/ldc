package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	"github.com/launchdarkly/ldc/api"
	"github.com/launchdarkly/ldc/goalapi"
)

var goalCompleter = makeCompleter(emptyOnError(listGoalNames))

func addGoalCommands(shell *ishell.Shell) {

	root := &ishell.Cmd{
		Name:    "goals",
		Aliases: []string{"goals"},
		Help:    "list and operate on goals",
		Func:    showGoals,
	}
	root.AddCmd(&ishell.Cmd{
		Name:      "list",
		Help:      "list goals",
		Aliases:   []string{"ls", "l"},
		Completer: goalCompleter,
		Func:      showGoals,
	})
	create := &ishell.Cmd{
		Name:    "create",
		Aliases: []string{"new"},
		Help:    "Create new goal",
	}
	create.AddCmd(&ishell.Cmd{
		Name: "custom",
		Help: "Create new custom goal",
		Func: createCustomGoal,
	})
	root.AddCmd(create)

	root.AddCmd(&ishell.Cmd{
		Name:      "show",
		Help:      "show a goal's details",
		Completer: goalCompleter,
		Func:      showGoals,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "attach",
		Help:      "attach to flag",
		Completer: goalCompleter,
		//Completer: attachGoalCompleter, // TODO: figure out how to do context-dependent completion
		Func: attachGoal,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "detach",
		Help:      "detach from flag",
		Completer: goalCompleter,
		//Completer: detachGoalCompleter, // TODO: figure out how to do context-dependent completion
		Func: detachGoal,
	})
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

func getGoalArg(c *ishell.Context) *goalapi.Goal {
	goals, _ := goalapi.GetGoals()
	if len(c.Args) > 0 {
		goalKey := c.Args[0]
		for _, g := range goals {
			if g.ID == goalKey || g.Name == goalKey {
				foundGoal, err := goalapi.GetGoal(g.ID)
				if err != nil {
					c.Err(err)
					return nil
				}
				return foundGoal
			}
		}
	}

	options, err := listGoalNames()
	if err != nil {
		c.Err(err)
		return nil
	}
	choice := c.MultiChoice(options, "Choose a goal: ")
	if choice < 0 {
		return nil
	}
	foundGoal, _ := goalapi.GetGoal(options[choice])
	return foundGoal
}

func listGoalNames() ([]string, error) {
	var keys []string
	g, err := goalapi.GetGoals()
	if err != nil {
		return nil, err
	}
	for _, g := range g {
		keys = append(keys, g.Name)
	}
	return keys, nil
}

func showGoals(c *ishell.Context) {
	if len(c.Args) > 0 {
		goal := getGoalArg(c)
		if goal == nil {
			c.Println("Unknown goal")
			return
		}
		renderGoal(c, goal)
		return
	}

	goals, err := goalapi.GetGoals()
	if err != nil {
		c.Err(err)
		return
	}
	buf := bytes.Buffer{}
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Name", "ID", "Description", "Kind", "Attached Flags"})
	for _, goal := range goals {
		table.Append([]string{goal.Name, goal.ID, goal.Description, goal.Kind, strconv.Itoa(goal.AttachedFeatureCount)})
	}
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	table.Render()
	if buf.Len() > 1000 {
		c.Err(c.ShowPaged(buf.String()))
	} else {
		c.Print(buf.String())
	}
}

func renderGoal(c *ishell.Context, goal *goalapi.Goal) {
	if renderJSON(c) {
		data, err := json.MarshalIndent(goal, "", " ")
		if err != nil {
			c.Err(err)
			return
		}
		c.Println(string(data))
		return
	}

	buf := bytes.NewBufferString("")
	table := tablewriter.NewWriter(buf)
	table.SetHeader([]string{"Field", "Value"})
	table.Append([]string{"Name", goal.Name})
	table.Append([]string{"Description", goal.Description})
	table.Append([]string{"Kind", goal.Kind})
	table.Append([]string{"Attached Flags", strconv.Itoa(goal.AttachedFeatureCount)})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.Render()
	c.Print(buf.String())
	if goal.AttachedFeatureCount > 0 {
		c.Println()
		c.Println("Attached Flags:")
		buf := bytes.NewBufferString("")
		table := tablewriter.NewWriter(buf)
		for _, f := range goal.AttachedFeatures {
			table.SetHeader([]string{"Key", "Name", "On"})
			table.Append([]string{f.Key, f.Name, boolToCheck(f.On)})
		}
		table.Render()
		c.Print(buf.String())
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

	if patchComment == nil {
		c.Println("No changes")
		return
	}

	_, err = goalapi.PatchGoal(goal.ID, *patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println("Updated goal")
}

func createCustomGoal(c *ishell.Context) {
	var name string
	var key string
	if len(c.Args) > 1 {
		name = c.Args[0]
		key = c.Args[1]
	} else {
		c.Print("Name: ")
		name = c.ReadLine()
		c.Print("Key: ")
		key = c.ReadLine()
	}
	goal := goalapi.Goal{
		Name: name,
		Kind: "custom",
		Key:  &key,
	}
	newGoal, err := goalapi.CreateGoal(goal)
	if err != nil {
		c.Err(err)
		return
	}
	if isInteractive(c) {
		c.Println("Created goal")
	}
	if renderJSON(c) {
		renderGoal(c, newGoal)
	}
}

func deleteGoal(c *ishell.Context) {
	goal := getGoalArg(c)

	err := goalapi.DeleteGoal(goal.ID)
	if err != nil {
		c.Err(err)
	} else {
		c.Println("Deleted goal")
	}
}

func boolToCheck(b bool) string {
	if b {
		return "X"
	}
	return " "
}

func attachGoal(c *ishell.Context) {
	var goal *goalapi.Goal
	var flag *ldapi.FeatureFlag
	goal = getGoalArg(c)
	flag = getFlagArg(c, 1)

	for _, g := range flag.GoalIds {
		if g == goal.ID {
			c.Println("Goal already attached")
			return
		}
	}

	patchComment := ldapi.PatchComment{
		Patch: []ldapi.PatchOperation{{Op: "add", Path: "/goalIds/-", Value: interfacePtr(goal.ID)}},
	}
	_, _, err := api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
	if err != nil {
		c.Err(err)
		return
	}
}

func detachGoal(c *ishell.Context) {
	var goal *goalapi.Goal
	var flag *ldapi.FeatureFlag
	goal = getGoalArg(c)
	flag = getFlagArg(c, 1)

	var pos *int
	for p, g := range flag.GoalIds {
		if g == goal.ID {
			pos = &p // nolint:scopelint // ok because we break
			break
		}
	}

	if pos == nil {
		c.Println("Goal is not currently attached")
		return
	}

	patchComment := ldapi.PatchComment{
		Patch: []ldapi.PatchOperation{{Op: "remove", Path: fmt.Sprintf("/goalIds/%d", *pos)}},
	}
	_, _, err := api.Client.FeatureFlagsApi.PatchFeatureFlag(api.Auth, api.CurrentProject, flag.Key, patchComment)
	if err != nil {
		c.Err(err)
		return
	}
}

// TODO: enable these when we can do context specific completion
//
//func attachGoalCompleter(args []string) []string {
//	fmt.Printf("attach completer: %+v\n", args)
//	if len(args) <= 1 {
//		return goalCompleter(args)
//	}
//
//	if len(args) > 2 {
//		return nil
//	}
//
//	return flagCompleter(args[1:])
//}
//
//func detachGoalCompleter(args []string) (completions []string) {
//	if len(args) <= 1 {
//		return goalCompleter(args)
//	}
//
//	if len(args) > 2 {
//		return nil
//	}
//
//	goals, _ := goalapi.GetGoals()
//	goalKey := args[0]
//	for _, g := range goals {
//		if g.ID == goalKey || g.Name == goalKey {
//			goal, err := goalapi.GetGoal(g.ID)
//			if err != nil {
//				return nil
//			}
//			for _, f := range goal.AttachedFeatures {
//				completions = append(completions, f.Key)
//			}
//		}
//	}
//
//	return nil
//}
