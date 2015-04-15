package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var usercmd = &cobra.Command{
	Use:   "user [cmd]",
	Short: "user commands",
	Long:  "user commands",
}

var keycmd = &cobra.Command{
	Use:   "key [cmd]",
	Short: "key commands",
	Long:  "key commands",
}
var addUser = &cobra.Command{
	Use:   "add [uid] [name] [roles...]",
	Short: "add a user",
	Long:  "add a user to the datastore",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		exitWhenError(c.createUser(args[0], args[1], args[2:]...))
	},
}

var listUsers = &cobra.Command{
	Use:   "list",
	Short: "list all users",
	Long:  "list all registered users",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		usrs, err := c.listUsers()
		exitWhenError(err)
		fmt.Printf("%-20s %-20s %-40s\n", "Uid", "Name", "Roles")
		for _, u := range usrs {
			fmt.Printf("%-20s %-20s %-40s\n", u.Id, u.Name, u.Roles)
			for _, k := range u.Keys {
				fmt.Printf("  %s:%s\n  %s\n", k.Id, k.Fingerprint, k.Value)
			}
		}
	},
}
var keyname string
var addKey = &cobra.Command{
	Use:   "add [uid] [key-file]",
	Short: "add a key",
	Long:  "Add a key to the authorized keys of the user. If no keyname is given, the key-file will be parsed to get one",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		exitWhenError(c.addKey(args[0], keyname, args[1]))
	},
}
var delKey = &cobra.Command{
	Use:   "del [uid] [key-name]",
	Short: "delete a key",
	Long:  "Delete a key from the authorized keys of the user.",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		exitWhenError(c.deleteKey(args[0], args[1]))
	},
}

func init() {
	usercmd.AddCommand(addUser, listUsers)
	keycmd.AddCommand(addKey, delKey)
	addKey.Flags().StringVarP(&keyname, "keyname", "k", "", "the keyname to use. if empty try to parse the given keyfile")
}

func exitWhenError(err error) {
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
