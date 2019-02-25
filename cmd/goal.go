package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/launchdarkly/ldc/cmd/internal/path"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/olekukonko/tablewriter"
	ishell "gopkg.in/abiosoft/ishell.v2"

	"github.com/launchdarkly/ldc/api"
	"github.com/launchdarkly/ldc/goalapi"
)

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
		Help:      "show a goal's details [goal]",
		Completer: goalCompleter,
		Func:      showGoal,
	})

	root.AddCmd(&ishell.Cmd{
		Name:      "results",
		Help:      "show a goal's experiment results for a flag [show <goal name> <flag key>]",
		Completer: detachGoalCompleter,
		Func:      showExperimentResults,
	})

	root.AddCmd(&ishell.Cmd{
		Name:      "attach",
		Help:      "attach to flag",
		Completer: attachGoalCompleter,
		Func:      attachGoal,
	})
	root.AddCmd(&ishell.Cmd{
		Name:      "detach",
		Help:      "detach from flag",
		Completer: detachGoalCompleter,
		Func:      detachGoal,
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

type goalPath struct {
	perEnvironmentPath
}

func (p goalPath) ID() string {
	return p.Keys()[2]
}

func (p goalPath) EnvPath() perProjectPath {
	return perProjectPath{path.NewAbsPath(p.Config(), p.Project(), p.Environment())}
}

func getGoalArg(c *ishell.Context) (goalPath, *goalapi.Goal) {
	var goal *goalapi.Goal
	var realPath path.ResourcePath
	if len(c.Args) > 0 {
		pathArg := path.ResourcePath(c.Args[0])
		if !pathArg.IsAbs() && pathArg.Depth() == 1 {
			pathArg = path.NewAbsPath(currentConfig, currentProject, currentEnvironment, pathArg.Keys()[0])
		}
		realPath, err := realGoalPath(pathArg)
		if err != nil {
			c.Err(err)
			return goalPath{}, nil
		}

		ctx, err := newGoalAPIContext(realPath.EnvPath())
		if err != nil {
			c.Err(err)
			return goalPath{}, nil
		}

		goals, err := goalapi.GetGoals(ctx)
		if err != nil {
			c.Err(err)
			return goalPath{}, nil
		}

		// match either id or name
		for _, g := range goals {
			if g.ID == realPath.ID() || g.Name == realPath.Key() {
				goal, err = goalapi.GetGoal(ctx, g.ID)
				if err != nil {
					c.Err(err)
					return goalPath{}, nil
				}
				break
			}
		}
	} else {
		goal, err := chooseGoalFromCurrentEnvironment(c)
		if err != nil {
			c.Err(err)
			return goalPath{}, nil
		}
		realPath = path.NewAbsPath(currentConfig, currentProject, currentEnvironment, *goal.Key)
	}

	return goalPath{perEnvironmentPath{realPath}}, goal
}

func chooseGoalFromCurrentEnvironment(c *ishell.Context) (goal *goalapi.Goal, err error) {
	envPath := perProjectPath{ResourcePath: path.NewAbsPath(currentConfig, currentProject, currentEnvironment)}
	ctx, err := newGoalAPIContext(envPath)
	if err != nil {
		return nil, err
	}

	goals, _ := goalapi.GetGoals(ctx)
	var options []string
	for _, g := range goals {
		options = append(options, g.Name)
	}
	if err != nil {
		return nil, err
	}
	choice := c.MultiChoice(options, "Choose a goal: ")
	if choice < 0 {
		return nil, err
	}
	foundGoal, _ := goalapi.GetGoal(ctx, goals[choice].ID)
	return foundGoal, nil
}

func goalCompleter(args []string) (completions []string) {
	if len(args) > 1 {
		return nil
	}

	completer := path.NewCompleter(getDefaultPath, configLister, projLister, envLister, goalLister)
	completions, _ = completer.GetCompletions(firstOrEmpty(args))
	return completions
}

var goalLister = path.ListerFunc(func(parentPath path.ResourcePath) ([]string, error) {
	return listGoalNames(parentPath.Config(), parentPath.Keys()[0], parentPath.Keys()[1])
})

func listGoalNames(configKey *string, projKey string, envKey string) ([]string, error) {
	envPath := perProjectPath{ResourcePath: path.NewAbsPath(configKey, projKey, envKey)}
	ctx, err := newGoalAPIContext(envPath)
	if err != nil {
		return nil, err
	}

	var keys []string
	g, err := goalapi.GetGoals(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range g {
		keys = append(keys, g.Name)
	}
	return keys, nil
}

func showGoal(c *ishell.Context) {
	_, goal := getGoalArg(c)
	if goal == nil {
		c.Println("Unknown goal")
		return
	}
	renderGoal(c, goal)
}

func showGoals(c *ishell.Context) {
	if len(c.Args) > 0 {
		_, goal := getGoalArg(c)
		if goal == nil {
			c.Println("Unknown goal")
			return
		}
		renderGoal(c, goal)
		return
	}

	envPath := perProjectPath{ResourcePath: path.NewAbsPath(currentConfig, currentProject, currentEnvironment)}

	ctx, err := newGoalAPIContext(envPath)
	if err != nil {
		c.Err(err)
		return
	}

	goals, err := goalapi.GetGoals(ctx)
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

func showExperimentResults(c *ishell.Context) {
	p, goal := getGoalArg(c)
	if goal == nil {
		c.Println("Unknown goal")
		return
	}

	_, flag := getFlagArg(c, 1)
	if flag == nil {
		c.Println("Unknown flag")
		return
	}

	ctx, err := newGoalAPIContext(p.EnvPath())
	if err != nil {
		c.Err(err)
		return
	}

	results, err := goalapi.GetExperimentResults(ctx, goal.ID, flag.Key)
	if err != nil {
		c.Err(err)
		return
	}

	renderExperimentResults(c, results)
}

func renderGoal(c *ishell.Context, goal *goalapi.Goal) {
	if renderJSON(c) {
		printJSON(c, goal)
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
	p, goal := getGoalArg(c)
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

	ctx, err := newGoalAPIContext(p.EnvPath())
	if err != nil {
		c.Err(err)
		return
	}

	_, err = goalapi.PatchGoal(ctx, goal.ID, *patchComment)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println("Updated goal")
}

func createCustomGoal(c *ishell.Context) {
	var p goalPath
	var key string
	if len(c.Args) > 1 {
		p = goalPath{perEnvironmentPath{path.ResourcePath(c.Args[0])}}
		key = c.Args[1]
	} else {
		c.Print("Name: ")
		name := c.ReadLine()
		c.Print("Key: ")
		key = c.ReadLine()
		p = goalPath{perEnvironmentPath{path.NewAbsPath(currentConfig, currentProject, currentEnvironment, name)}}
	}
	goal := goalapi.Goal{
		Name: p.Key(),
		Kind: "custom",
		Key:  &key,
	}

	ctx, err := newGoalAPIContext(p.EnvPath())
	if err != nil {
		c.Err(err)
		return
	}

	newGoal, err := goalapi.CreateGoal(ctx, goal)
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
	p, goal := getGoalArg(c)

	ctx, err := newGoalAPIContext(p.EnvPath())
	if err != nil {
		c.Err(err)
		return
	}

	err = goalapi.DeleteGoal(ctx, goal.ID)
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
	var flag *ldapi.FeatureFlag
	goalPath, goal := getGoalArg(c)
	_, flag = getFlagArg(c, 1)

	for _, g := range flag.GoalIds {
		if g == goal.ID {
			c.Println("Goal already attached")
			return
		}
	}

	patchComment := ldapi.PatchComment{
		Patch: []ldapi.PatchOperation{{Op: "add", Path: "/goalIds/-", Value: interfacePtr(goal.ID)}},
	}

	client, err := api.GetClient(getServer(goalPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(goalPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, currentProject, flag.Key, patchComment)
	if err != nil {
		c.Err(err)
		return
	}
	c.Println("Goal was attached")
}

func detachGoal(c *ishell.Context) {
	var flag *ldapi.FeatureFlag
	goalPath, goal := getGoalArg(c)
	_, flag = getFlagArg(c, 1)

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

	client, err := api.GetClient(getServer(goalPath.Config()))
	if err != nil {
		c.Err(err)
		return
	}
	auth := api.GetAuthCtx(getToken(goalPath.Config()))
	_, _, err = client.FeatureFlagsApi.PatchFeatureFlag(auth, currentProject, flag.Key, patchComment)
	if err != nil {
		c.Err(err)
		return
	}
	c.Println("Goal was detached")
}

func attachGoalCompleter(args []string) []string {
	if len(args) == 0 {
		return nonFinalCompleter(goalCompleter)(args)
	}

	if len(args) > 2 {
		return nil
	}

	return flagCompleter(args[1:])
}

func detachGoalCompleter(args []string) (completions []string) {
	if len(args) == 0 {
		return nonFinalCompleter(goalCompleter)(args)
	}

	if len(args) > 2 {
		return nil
	}

	ctx, err := newGoalAPIContext(perProjectPath{path.NewAbsPath(currentConfig, currentProject, currentEnvironment)})
	if err != nil {
		return
	}

	goals, _ := goalapi.GetGoals(ctx)
	goalKey := args[0]
	for _, g := range goals {
		if g.ID == goalKey || g.Name == goalKey {
			goal, err := goalapi.GetGoal(ctx, g.ID)
			if err != nil {
				return nil
			}
			for _, f := range goal.AttachedFeatures {
				completions = append(completions, f.Key)
			}
		}
	}

	return completions
}

func renderExperimentResults(c *ishell.Context, results *goalapi.ExperimentResults) {
	if renderJSON(c) {
		data, err := json.MarshalIndent(results, "", " ")
		if err != nil {
			c.Err(err)
			return
		}
		c.Println(string(data))
		return
	}

	buf := bytes.NewBufferString("")
	table := tablewriter.NewWriter(buf)
	table.SetHeader([]string{"Change", "Confidence", "Z Score"})
	table.Append([]string{floatToStr(results.Change), floatToStr(results.ConfidenceScore), floatToStr(results.ZScore)})
	table.Render()
	c.Print(buf.String())

	buf = bytes.NewBufferString("")
	table = tablewriter.NewWriter(buf)
	table.SetHeader([]string{"Field", "Control", "Experiment"})
	table.Append([]string{"Conversions", strconv.Itoa(results.Control.Conversions), strconv.Itoa(results.Experiment.Conversions)})
	table.Append([]string{"Impressions", strconv.Itoa(results.Control.Impressions), strconv.Itoa(results.Experiment.Impressions)})
	table.Append([]string{"Conversion Rate", floatToStr(results.Control.ConversionRate), floatToStr(results.Experiment.ConversionRate)})
	table.Append([]string{"Confidence Interval", floatToStr(results.Control.ConfidenceInterval), floatToStr(results.Experiment.ConfidenceInterval)})
	table.Append([]string{"Standard Error", floatToStr(results.Control.StandardError), floatToStr(results.Experiment.StandardError)})
	table.Render()
	c.Print(buf.String())
}

func floatToStr(f float64) string {
	return fmt.Sprintf("%f", f)
}

func realGoalPath(p path.ResourcePath) (goalPath, error) {
	if p.Depth() != 2 {
		return goalPath{}, errors.New("invalid path")
	}
	np, err := path.ReplaceDefaults(p, getDefaultPath, 2)
	if err != nil {
		return goalPath{}, err
	}
	return goalPath{perEnvironmentPath{np}}, nil
}

func newGoalAPIContext(envPath perProjectPath) (goalapi.Context, error) {
	host := getServer(envPath.Config())
	auth := api.GetAuthCtx(getToken(envPath.Config()))
	client, err := api.GetClient(host)
	if err != nil {
		return goalapi.Context{}, err
	}
	env, _, err := client.EnvironmentsApi.GetEnvironment(auth, envPath.Project(), envPath.Key())
	if err != nil {
		return goalapi.Context{}, err
	}
	return goalapi.NewContext(host, env.ApiKey), nil
}
