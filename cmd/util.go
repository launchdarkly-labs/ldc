package cmd

import "github.com/abiosoft/ishell"

func confirmDelete(c *ishell.Context, name string, expectedValue string) bool {
	if !interactive {
		return false
	}
	c.Printf("Re-enter the %s '%s' to delete: ", name, expectedValue)
	value := c.ReadLine()
	return value == expectedValue
}
