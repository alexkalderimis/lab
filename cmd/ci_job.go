package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

type Action int

const (
	Run Action = iota
	Delete
)

var (
	runJob    bool   = true
	deleteJob bool   = false
	jobAction Action = Run
)

// ciLintCmd represents the lint command
var ciJobCmd = &cobra.Command{
	Use:     "job jobId",
	Aliases: []string{"logs"},
	Short:   "manipulate CI jobs (default: re-run)",
	Run: func(cmd *cobra.Command, args []string) {
		var (
			remote   string
			jobIdStr string
			jobId    int
		)

		branch, err := git.CurrentBranch()
		if err != nil {
			log.Fatal(err)
		}
		if len(args) == 2 {
			jobIdStr = args[1]
			if strings.Contains(args[1], ":") {
				ps := strings.Split(args[1], ":")
				branch, jobIdStr = ps[0], ps[1]
			}
		} else if len(args) == 1 {
			jobIdStr = args[0]
			if strings.Contains(args[0], ":") {
				ps := strings.Split(args[1], ":")
				branch, jobIdStr = ps[0], ps[1]
			}
		} else {
			log.Fatal("Expected at most 2 arguments")
		}

		remote = determineSourceRemote(branch)
		jobId, err = strconv.Atoi(jobIdStr)
		if err != nil {
			log.Fatal(err)
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		if deleteJob {
			jobAction = Delete
		} else if runJob {
			jobAction = Run
		}
		client := lab.Client()
		job, _, err := client.Jobs.GetJob(project.ID, jobId)
		if err != nil {
			log.Fatal(err)
		}

		switch jobAction {
		case Run:
			_, err := lab.CIPlayOrRetry(project.ID, job.ID, job.Status)
			if err != nil {
				log.Fatal(err)
			}
		default:
			log.Fatal(fmt.Sprintf("Cannot handle %v", jobAction))
		}
	},
}

func init() {
	ciJobCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	ciJobCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_remote_branches $words[2]")
	ciJobCmd.Flags().BoolVarP(&runJob, "run", "r", true, "Run the job (default)")
	ciJobCmd.Flags().BoolVarP(&deleteJob, "delete", "d", false, "Delete the job")
	ciCmd.AddCommand(ciJobCmd)
}
