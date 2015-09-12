package main

import "github.com/spf13/cobra"

var root = &cobra.Command{}

func main() {
	root.AddCommand(serve)
	root.Execute()
}
