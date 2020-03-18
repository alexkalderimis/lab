package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
	strings "strings"
)

var (
	issueLabels           []string
	issueShowScopedLabels []string
	issueState            string
	issueSearch           string
	issueNumRet           int
	issueAll              bool
	issueMine             bool
	issueAssignee         string
	issueAuthor           string
	issueAssigneeID       *int
	issueAuthorID         *int
)

type PrintableIssue struct {
	iid    string
	labels string
	title  string
}

var issueListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List issues",
	Long:    ``,
	Example: `lab issue list                        # list all open issues
lab issue list "search terms"         # search issues for "search terms"
lab issue search "search terms"       # same as above
lab issue list remote "search terms"  # search "remote" for issues with "search terms"`,
	Run: func(cmd *cobra.Command, args []string) {
		rn, issueSearch, err := parseArgsRemoteString(args)
		if err != nil {
			log.Fatal(err)
		}
		if issueMine {
			issueAssignee = "@"
		}
		err = getUserId(issueAssignee, &issueAssigneeID)
		if err != nil {
			log.Fatal(err)
		}
		err = getUserId(mrAuthor, &issueAuthorID)
		if err != nil {
			log.Fatal(err)
		}

		opts := gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: issueNumRet,
			},
			Labels:     issueLabels,
			State:      &issueState,
			OrderBy:    gitlab.String("updated_at"),
			AssigneeID: issueAssigneeID,
			AuthorID:   issueAuthorID,
		}

		if issueSearch != "" {
			opts.Search = &issueSearch
		}

		num := issueNumRet
		if issueAll {
			num = -1
		}
		issues, err := lab.IssueList(rn, opts, num)
		if err != nil {
			log.Fatal(err)
		}
		var w strings.Builder
		var printables []PrintableIssue
		printables = make([]PrintableIssue, len(issues), len(issues))
		for i, issue := range issues {
			w.Reset()
			written := 0
			for _, scope := range issueShowScopedLabels {
				for _, label := range issue.Labels {
					if strings.HasPrefix(label, scope) {
						if written > 0 {
							w.WriteString(",")
						}
						w.WriteString("[" + label + "]")
						written += 1
					}
				}
			}
			printables[i] = PrintableIssue{
				iid:    fmt.Sprintf("#%d", issue.IID),
				labels: w.String(),
				title:  issue.Title,
			}
		}
		maxIIDLen := 0
		maxLabelLen := 0
		for _, p := range printables {
			if len(p.iid) > maxIIDLen {
				maxIIDLen = len(p.iid)
			}
			if len(p.labels) > maxLabelLen {
				maxLabelLen = len(p.labels)
			}
		}
		for _, p := range printables {
			fmt.Printf("%*s %*s %s\n", -maxIIDLen, p.iid, -maxLabelLen, p.labels, p.title)
		}
	},
}

func init() {
	cmd := issueListCmd
	cmd.Flags().StringSliceVarP(
		&issueLabels, "label", "l", []string{},
		"Filter issues by label")
	cmd.Flags().StringSliceVarP(
		&issueShowScopedLabels, "show-label", "", []string{},
		"Show labels with a given scope (e.g. 'workflow' for 'workflow::x' and 'workflow::y')")
	cmd.Flags().StringVarP(
		&issueState, "state", "s", "opened",
		"Filter issues by state (opened/closed)")
	cmd.Flags().IntVarP(
		&issueNumRet, "number", "n", 10,
		"Number of issues to return")
	cmd.Flags().BoolVarP(
		&issueAll, "all", "a", false,
		"List all issues on the project")
	cmd.Flags().BoolVarP(
		&issueMine, "mine", "m", false,
		"List only issues assigned to me")
	cmd.Flags().StringVar(
		&issueAssignee, "assignee", "",
		"List only issues assigned to $username (or @ for assigned to me)")
	cmd.Flags().StringVar(
		&issueAuthor, "author", "",
		"List only MRs authored by $username (or @ for by me)")

	cmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	cmd.MarkFlagCustom("state", "(opened closed)")
	issueCmd.AddCommand(cmd)
}
