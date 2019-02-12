package cmd

import (
	"strings"

	shlex "github.com/flynn-archive/go-shlex"
	ishell "gopkg.in/abiosoft/ishell.v2"
)

type customCompleter struct {
	shell    *ishell.Shell
	disabled func() bool
}

// copied directly from ishell
func (cc customCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	if cc.disabled != nil && cc.disabled() {
		return nil, len(line)
	}
	var words []string
	if w, err := shlex.Split(string(line)); err == nil {
		words = w
	} else {
		// fall back
		words = strings.Fields(string(line))
	}

	var cWords []string
	prefix := ""
	if len(words) > 0 && pos > 0 && line[pos-1] != ' ' {
		prefix = words[len(words)-1]
	} else {
		words = append(words, "")
	}
	cWords = cc.getSuggestions(words)

	var suggestions [][]rune
	for _, w := range cWords {
		if strings.HasPrefix(w, prefix) {
			suggestions = append(suggestions, []rune(strings.TrimPrefix(w, prefix)))
		}
	}
	if len(suggestions) == 1 && prefix != "" && string(suggestions[0]) == "" {
		suggestions = [][]rune{[]rune(" ")}
	}
	return suggestions, len(prefix)
}

func (cc customCompleter) getSuggestions(w []string) (suggestions []string) {
	// Add all the top-level options
	if len(w) == 1 {
		for _, c := range cc.shell.Cmds() {
			suggestions = append(suggestions, c.Name)
		}
	}
	for _, c := range cc.shell.Cmds() {
		if !containsString(append(append([]string{}, c.Name), c.Aliases...), w[0]) {
			continue
		}

		// Search for children
		cmd, args := c.FindCmd(w[1:])
		if cmd != nil {
			if cmd.Completer != nil {
				return cmd.Completer(args)
			}
			if len(args) == 0 {
				return []string{""}
			}
			return nil
		}

		if len(c.Children()) > 0 {
			for _, k := range c.Children() {
				suggestions = append(suggestions, k.Name)
			}
			return suggestions
		}

		if c.Completer != nil {
			return c.Completer(args)
		}
	}
	return suggestions
}
