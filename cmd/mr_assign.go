package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrAssignCmd = &cobra.Command{
	Use:     "assign [remote] <id> username",
	Aliases: []string{},
	Short:   "Assign merge request",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username, args := args[len(args)-1], args[:len(args)-1]
		rn, id, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}
		userID, err := lab.UserIDFromUsername(username)
		if err != nil {
			log.Fatal(err)
		}
		if userID <= 0 {
			log.Fatalf("Cannot find %s", username)
		}
		mr, err := lab.MRGet(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}
		err = lab.MRAssign(mr, userID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Current assignees:")
		for _, user := range mr.Assignees {
			fmt.Printf("\t@%s\n", user.Username)
		}
	},
}

func init() {
	mrAssignCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrAssignCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrAssignCmd)
}
