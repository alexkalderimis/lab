package cmd

import (
	"errors"
	"log"
	"time"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrNewsCmd = &cobra.Command{
	Use:        "news [remote] [id]",
	Aliases:    []string{"breaking"},
	ArgAliases: []string{},
	Short:      "Show things that have happened since you last touched this",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		_, mr, err := parseProjectMR(args)
		if err != nil {
			log.Fatal(err)
		}
		var currentUserID *int
		err = getUserId("@", &currentUserID)
		if err != nil {
			log.Fatal(err)
		}
		for _, username := range ignoreUsers {
			ignore[username] = true
		}
		time, err := lastCommentAt(mr, *currentUserID)
		if err != nil {
			log.Fatal(err)
		}
		getNotesSince(mr, time)
	},
}

func lastCommentAt(mr *gitlab.MergeRequest, userID int) (*time.Time, error) {
	client := lab.Client()
	opts := gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 50},
		OrderBy:     gitlab.String("created_at"),
		Sort:        gitlab.String("desc"),
	}

	for {
		notes, resp, err := client.Notes.ListMergeRequestNotes(mr.ProjectID, mr.IID, &opts)
		if err != nil {
			return nil, err
		}
		for _, note := range notes {
			if note.Author.ID == userID {
				time := note.CreatedAt
				return time, nil
			}
		}
		opts.ListOptions.Page = resp.NextPage
		if resp.CurrentPage >= resp.NextPage || resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return nil, errors.New("No comment by that author")
}

func init() {
	cmd := mrNewsCmd
	cmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	cmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	cmd.Flags().BoolVarP(&includeSystemNotes, "system-notes", "s", false, "Include system notes, in addition to user comments")
	cmd.Flags().IntVarP(&contextWindow, "context-window", "c", 5, "How large the context window should be (0 == no context, 1 == just the line)")
	cmd.Flags().IntVarP(&lineLen, "line-length", "l", 100, "How wide should comments be? Lines longer than this value will be wrapped")
	cmd.Flags().StringSliceVar(&ignoreUsers, "ignore", ignoreUsers, "Set of users to ignore")
	mrCmd.AddCommand(cmd)
}
