package cmd

import (
	"bufio"
	"fmt"
	"log"
	exec "os/exec"
	strings "strings"
	"unicode"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	includeSystemNotes bool
	contextWindow      int
	ignoreUsers        []string
	threaded           bool
	ignore             map[string]bool
	lineLen            = 100
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
		for _, username := range ignoreUsers {
			ignore[username] = true
		}
		if threaded {
			getDiscussions(mr)
		} else {
			getNotes(mr)
		}
	},
}

func getNotes(mr *gitlab.MergeRequest) {
	client := lab.Client()
	opts := gitlab.ListMergeRequestNotesOptions{ListOptions: gitlab.ListOptions{PerPage: 500}}
	for {
		list, resp, err := client.Notes.ListMergeRequestNotes(mr.ProjectID, mr.IID, &opts)
		if err != nil {
			log.Fatal(err)
		}
		for _, note := range list {
			if note.System && !includeSystemNotes {
				continue
			}
			if ignore[note.Author.Username] {
				continue
			}
			fmt.Println(printNote(note, false))
		}
		opts.ListOptions.Page = resp.NextPage
		if resp.CurrentPage >= resp.NextPage || resp.CurrentPage == resp.TotalPages {
			break
		}
	}
}

func getDiscussions(mr *gitlab.MergeRequest) {
	client := lab.Client()
	opts := gitlab.ListMergeRequestDiscussionsOptions{PerPage: 500}
	var w strings.Builder
	for {
		list, resp, err := client.Discussions.ListMergeRequestDiscussions(mr.ProjectID, mr.IID, &opts)
		if err != nil {
			log.Fatal(err)
		}
		for _, discussion := range list {
			w.Reset()
			written := 0
			for i, note := range discussion.Notes {
				if note.System && !includeSystemNotes {
					continue
				}
				if ignore[note.Author.Username] {
					continue
				}
				n, err := w.WriteString(printNote(note, i > 0))
				if err != nil {
					log.Fatal(err)
				}
				written += n
			}
			if written > 0 {
				fmt.Println(strings.Repeat("=", 80))
				fmt.Printf("Discussion: %s\n", discussion.ID)
				fmt.Println(w.String())
			}
		}
		opts.Page = resp.NextPage
		if resp.CurrentPage >= resp.NextPage || resp.CurrentPage == resp.TotalPages {
			break
		}
	}
}

// TODO: fix printing notes when the line-width is too small
func printNote(note *gitlab.Note, isReply bool) string {
	var b strings.Builder
	resolved := ""
	if note.Resolved {
		resolved = "RESOLVED"
	}
	position := ""
	if note.Position != nil && !isReply {
		context := getContext(note.Position.HeadSHA, note.Position.NewPath, note.Position.NewLine)
		position = fmt.Sprintf("\n%s:%d %s", note.Position.NewPath, note.Position.NewLine, context)
	}
	indent := " "
	if isReply {
		indent = strings.Repeat(" ", 4) + "| "
	}

	fmt.Fprintf(&b, "%s@%*s %s %s%s\n%s%s",
		indent, -40, note.Author.Username, note.CreatedAt, resolved, position,
		indent, strings.Repeat("-", 80))

	for _, line := range strings.Split(note.Body, "\n") {
		if len(line) == 0 {
			fmt.Fprintf(&b, "\n%s", indent)
			continue
		}

		pos := 0
		rs := []rune(line)
		for {
			for pos > 0 && pos < len(rs) && unicode.IsSpace(rs[pos]) {
				pos++
			}
			if pos >= len(rs) {
				break
			}

			endPos := pos + lineLen
			if endPos >= len(rs) {
				endPos = len(rs) - 1
			}
			for pos <= endPos && endPos+1 < len(rs) && !unicode.IsSpace(rs[endPos+1]) {
				endPos--
			}

			fmt.Fprintf(&b, "\n%s%s", indent, string(rs[pos:endPos+1]))
			pos = endPos + 1
		}
	}
	fmt.Fprintf(&b, "\n\n")

	return b.String()
}

func getContext(sha string, path string, line int) string {
	if contextWindow < 1 {
		return ""
	}
	halfWindow := contextWindow / 2

	proc := exec.Command("git", "show", fmt.Sprintf("%s:%s", sha, path))
	output, err := proc.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := proc.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(output)
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
	if err := proc.Wait(); err != nil {
		log.Fatal(err)
	}
	return context.String()
}

func init() {
	mrNotesCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrNotesCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrNotesCmd.Flags().BoolVarP(&includeSystemNotes, "system-notes", "s", false, "Include system notes, in addition to user comments")
	mrNotesCmd.Flags().BoolVarP(&threaded, "threaded", "t", true, "Show notes in their discussion threads, with replies below the head comment")
	mrNotesCmd.Flags().IntVarP(&contextWindow, "context-window", "c", 5, "How large the context window should be (0 == no context, 1 == just the line)")
	mrNotesCmd.Flags().IntVarP(&lineLen, "line-length", "l", 100, "How wide should comments be? Lines longer than this value will be wrapped")
	mrNotesCmd.Flags().StringSliceVar(&ignoreUsers, "ignore", ignoreUsers, "Set of users to ignore")
	mrCmd.AddCommand(mrNotesCmd)
}
