package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	strings "strings"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	includeSystemNotes bool
	contextWindow      int
)

var mrNotesCmd = &cobra.Command{
	Use:        "notes [remote] [id]",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Show notes of a Merge Request",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		_, mr, err := parseProjectMR(args)
		if err != nil {
			log.Fatal(err)
		}
		getNotes(mr)
	},
}

func getNotes(mr *gitlab.MergeRequest) {
	client := lab.Client()
	opts := gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 500,
		},
	}
	for {
		list, resp, err := client.Notes.ListMergeRequestNotes(mr.ProjectID, mr.IID, &opts)
		if err != nil {
			log.Fatal(err)
		}
		for _, note := range list {
			if note.System && !includeSystemNotes {
				continue
			}
			printNote(note)
		}
		opts.Page = resp.NextPage
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
}

func printNote(note *gitlab.Note) {
	resolved := ""
	if note.Resolved {
		resolved = "RESOLVED"
	}
	position := ""
	if note.Position != nil {
		context := getContext(note.Position.NewPath, note.Position.NewLine)
		position = fmt.Sprintf("\n%s:%d %s", note.Position.NewPath, note.Position.NewLine, context)
	}
	fmt.Printf(`
@%*s %s %s%s
%s
%s

`,
		-40, note.Author.Username, note.CreatedAt, resolved, position, strings.Repeat("=", 80), note.Body)
}

func getContext(path string, line int) string {
	if contextWindow < 1 {
		return ""
	}
	halfWindow := contextWindow / 2

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "(NOT FOUND)"
		}
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var context strings.Builder
	currentLine := 1
	for scanner.Scan() {
		offset := currentLine - line
		if -halfWindow <= offset && offset <= halfWindow {
			context.WriteString("\n")
			if currentLine == line {
				context.WriteString(" > ")
			} else {
				context.WriteString(" | ")
			}
			context.WriteString(scanner.Text())
		}
		if offset >= halfWindow {
			break
		}
		currentLine += 1
	}
	if err := scanner.Err(); err != nil {
		context.WriteString("\n")
		context.WriteString(fmt.Sprintf("Error reading %s: %s", path, err))
	}
	file.Close()
	return context.String()
}

func init() {
	mrNotesCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrNotesCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrNotesCmd.Flags().BoolVarP(&includeSystemNotes, "system-notes", "s", false, "Include system notes, in addition to user comments")
	mrNotesCmd.Flags().IntVarP(&contextWindow, "context-window", "c", 5, "How large the context window should be (0 == no context, 1 == just the line)")
	mrCmd.AddCommand(mrNotesCmd)
}
