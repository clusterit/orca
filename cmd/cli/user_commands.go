package main

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
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
	Use:   "add [# network] [# uid] [# name] [# roles...]",
	Short: "add a user from the given network",
	Long:  "add a user to the datastore. the network could be one of your oauth-providers",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		if len(args) < 4 {
			cmd.Usage()
			os.Exit(1)
		}
		exitWhenError(c.createUser(args[0], args[1], args[2], args[3:]...))
	},
}

var removeAlias bool
var userAlias = &cobra.Command{
	Use:   "alias [# uid] [# network] [# alias]",
	Short: "add a alias for user",
	Long:  "add a alias for user. you can suffx existing users with their network name to link the alias to them, aka. add '@google' to them",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		if len(args) < 3 {
			cmd.Usage()
			os.Exit(1)
		}
		if removeAlias {
			exitWhenError(c.removeAlias(args[0], args[1], args[2]))
		} else {
			exitWhenError(c.addAlias(args[0], args[1], args[2]))
		}
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
			spew.Dump(u)
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
		if len(args) < 2 {
			cmd.Usage()
			os.Exit(1)
		}
		exitWhenError(c.addKey(args[0], keyname, args[1]))
	},
}
var delKey = &cobra.Command{
	Use:   "del [uid] [key-name]",
	Short: "delete a key",
	Long:  "Delete a key from the authorized keys of the user.",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		if len(args) < 2 {
			cmd.Usage()
			os.Exit(1)
		}
		exitWhenError(c.deleteKey(args[0], args[1]))
	},
}

func init() {
	usercmd.AddCommand(addUser, listUsers, userAlias)
	keycmd.AddCommand(addKey, delKey)
	addKey.Flags().StringVarP(&keyname, "keyname", "k", "", "the keyname to use. if empty try to parse the given keyfile")
	userAlias.Flags().BoolVar(&removeAlias, "remove", false, "remove the alias")
}

func exitWhenError(err error) {
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
