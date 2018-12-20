package cmd

import (
	"strings"

	"github.com/abiosoft/ishell"
)

func confirmDelete(c *ishell.Context, name string, expectedValue string) bool {
	if !interactive {
		return false
	}
	c.Printf("Re-enter the %s '%s' to delete: ", name, expectedValue)
	value := c.ReadLine()
	return value == expectedValue
}

func withPrefix(keys []string, prefix string) []string {
	var completions []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			completions = append(completions, key)
		}
	}
	return completions
}

func toPrefix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
