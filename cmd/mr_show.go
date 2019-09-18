package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
)

var mrShowCmd = &cobra.Command{
	Use:        "show [remote] [id]",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Describe a merge request",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mr, err := parseProjectMR(args)
		if err != nil {
			log.Fatal(err)
		}
		printMR(mr, rn)
	},
}

func printMR(mr *gitlab.MergeRequest, project string) {
	assignees := "None"
	milestone := "None"
	labels := "None"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
		"merged": "Merged",
	}[mr.State]
	var sb strings.Builder
	for _, assignee := range mr.Assignees {
		if assignee.Username == "" {
			continue
		}

		if sb.Len() > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(assignee.Username)
	}
	if sb.Len() > 0 {
		assignees = sb.String()
	}
	if mr.Milestone != nil {
		milestone = mr.Milestone.Title
	}
	if len(mr.Labels) > 0 {
		labels = strings.Join(mr.Labels, ", ")
	}

	fmt.Printf(`
#%d %s
===================================
%s
-----------------------------------
Project: %s
Branches: %s->%s
Status: %s
Assignees: %s
Author: %s
Milestone: %s
Labels: %s
WebURL: %s
`,
		mr.IID, mr.Title, mr.Description, project, mr.SourceBranch,
		mr.TargetBranch, state, assignees,
		mr.Author.Username, milestone, labels, mr.WebURL)
}

func init() {
	mrShowCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrShowCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrShowCmd)
}
