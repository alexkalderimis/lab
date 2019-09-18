package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	mrLabels       []string
	mrState        string
	mrTargetBranch string
	mrNumRet       int
	mrAll          bool
	mrMine         bool
	assigneeID     *int
	authorID       *int
	mrAssignee     string
	mrAuthor       string
	ciStatus       bool
	mrSourceBranch bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [remote]",
	Aliases: []string{"ls"},
	Short:   "List merge requests",
	Long:    ``,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, _, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		num := mrNumRet
		if mrAll {
			num = -1
		}

		if mrMine {
			mrAssignee = "@"
		}
		err = getUserId(mrAssignee, &assigneeID)
		if err != nil {
			log.Fatal(err)
		}
		err = getUserId(mrAuthor, &authorID)
		if err != nil {
			log.Fatal(err)
		}
		mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: mrNumRet,
			},
			Labels:       mrLabels,
			State:        &mrState,
			TargetBranch: &mrTargetBranch,
			OrderBy:      gitlab.String("updated_at"),
			AssigneeID:   assigneeID,
			AuthorID:     authorID,
		}, num)
		if err != nil {
			log.Fatal(err)
		}
		maxTitleLength := 0
		maxBranchNameLength := 0
		for _, mr := range mrs {
			if len(mr.Title) > maxTitleLength {
				maxTitleLength = len(mr.Title)
			}
			if len(mr.SourceBranch) > maxBranchNameLength {
				maxBranchNameLength = len(mr.SourceBranch)
			}
		}

		for _, mr := range mrs {
			fmt.Printf("#%d %*s", mr.IID, -maxTitleLength, mr.Title)
			if mrSourceBranch {
				fmt.Printf(" | %*s |", -maxBranchNameLength, mr.SourceBranch)
			}
			if ciStatus {
				pipelines, err := lab.MRPipelines(rn, mr)
				if err != nil {
					log.Fatal(err)
				}
				if len(pipelines) > 0 {
					status := pipelines[0].Status
					printer := lab.StatusColor(status)
					printer.Printf(" %s", status)
				}
			}
			fmt.Println("")
		}
	},
}

func getUserId(username string, userId **int) error {
	if username == "@" {
		_userId, err := lab.UserID()
		if err != nil {
			return err
		}
		*userId = &_userId
	} else if username != "" {
		_userId, err := lab.UserIDByUserName(username)
		if err != nil {
			return err
		}
		*userId = &_userId
	}
	return nil
}

func init() {
	listCmd.Flags().StringSliceVarP(
		&mrLabels, "label", "l", []string{}, "filter merge requests by label")
	listCmd.Flags().StringVarP(
		&mrState, "state", "s", "opened",
		"filter merge requests by state (opened/closed/merged)")
	listCmd.Flags().IntVarP(
		&mrNumRet, "number", "n", 10,
		"number of merge requests to return")
	listCmd.Flags().StringVarP(
		&mrTargetBranch, "target-branch", "t", "",
		"filter merge requests by target branch")
	listCmd.Flags().BoolVarP(&mrAll, "all", "a", false, "List all MRs on the project")
	listCmd.Flags().BoolVarP(&mrMine, "mine", "m", false, "List only MRs assigned to me")
	listCmd.Flags().StringVar(
		&mrAssignee, "assignee", "", "List only MRs assigned to $username (or @ for assigned to me)")
	listCmd.Flags().StringVar(
		&mrAuthor, "author", "", "List only MRs authored by $username (or @ for by me)")
	listCmd.Flags().BoolVarP(&ciStatus, "ci-status", "c", false, "Include CI Status in the results")
	listCmd.Flags().BoolVarP(&mrSourceBranch, "show-branch", "b", false, "Include source branch in the results")
	listCmd.Flags().BoolVarP(&lab.UseColor, "color", "", false, "Use color for CI job status")

	listCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	listCmd.MarkFlagCustom("state", "(opened closed merged)")
	mrCmd.AddCommand(listCmd)
}
