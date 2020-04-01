package cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	color "github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	onlyFailures bool
	noSkipped    bool
	wait         bool
	noCreated    bool
	summaryOnly  bool
)

const (
	jobFormat = "%*s: %*s - %10s id: %d\n"
)

// ciStatusCmd represents the run command
var ciStatusCmd = &cobra.Command{
	Use:     "status [branch]",
	Aliases: []string{"run"},
	Short:   "Textual representation of a CI pipeline",
	Long:    ``,
	Example: `lab ci status
lab ci status --wait`,
	RunE: nil,
	Run:  runCommand,
}

func runCommand(cmd *cobra.Command, args []string) {
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 1, byte(' '), 0)
	pid, mr, err := parseProjectMR(args)

	if err != nil {
		log.Fatal(err)
	}
	// TODO: select pipeline
	jobs, err := lab.PipelineJobs(pid, mr.HeadPipeline.ID)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to find ci jobs"))
	}
	jobs = latestJobs(jobs)

	if len(jobs) == 0 {
		return
	}

	if !summaryOnly {
		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
	}
	color.NoColor = !lab.UseColor
	var (
		printer *color.Color
	)
	pipelineId := jobs[0].Pipeline.ID
	pipeline, _, err := lab.Client().Pipelines.GetPipeline(pid, pipelineId)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to get pipeline information"))
	}
	maxNameLength := 0
	maxStageLength := 0
	for _, job := range jobs {
		if len(job.Name) > maxNameLength {
			maxNameLength = len(job.Name)
		}
		if len(job.Stage) > maxStageLength {
			maxStageLength = len(job.Stage)
		}
	}

	for {
		if !summaryOnly {
			for _, job := range jobs {
				if noSkipped && job.Status == "skipped" {
					continue
				} else if onlyFailures && job.Status != "failed" {
					continue
				} else if noCreated && job.Status == "created" {
					continue
				} else {
					printer = lab.StatusColor(job.Status)
					printer.Fprintf(w, jobFormat, maxStageLength, job.Stage, -maxNameLength, job.Name, job.Status, job.ID)
				}
			}
		}
		if !wait {
			break
		}
		pl, _, err := lab.Client().Pipelines.GetPipeline(pid, pipelineId)
		if err != nil {
			log.Fatal(errors.Wrap(err, "failed to get pipeline information"))
		}
		pipeline = pl
		if pipeline.Status != "pending" && pipeline.Status != "running" {
			break
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, pipelineStatus(pipeline, jobs))
	if wait && pipeline.Status != "success" {
		os.Exit(1)
	}
	w.Flush()
}

func pipelineStatus(pipeline *gitlab.Pipeline, jobs []*gitlab.Job) string {
	return fmt.Sprintf("\nPipeline Status:\t%s\n%s\n\n%s\n",
		lab.StatusColor(pipeline.Status).Sprintf(pipeline.Status), timeMessage(pipeline), jobSummary(jobs))
}

func jobSummary(jobs []*gitlab.Job) string {
	numPassed := 0
	totalJobs := 0
	numQueued := 0
	numFailed := 0

	for _, job := range jobs {
		totalJobs++
		if job.Status == "success" {
			numPassed++
		}
		if job.Status == "created" {
			numQueued++
		}
		if job.Status == "failed" {
			numFailed++
		}
	}

	return fmt.Sprintf("total\tpassed\tfailed\tqueued\n%d\t%d\t%d\t%d",
		totalJobs, numPassed, numFailed, numQueued)
}

func pluralize(noun string, quantity int64) string {
	if quantity == 0 {
		return ""
	} else if quantity == 1 {
		return "1 " + noun
	} else {
		return fmt.Sprintf("%d %ss", quantity, noun)
	}
}

func sinceMessage(moment time.Time) string {
	ago := time.Since(moment)
	var parts [2]string
	var buff strings.Builder
	if ago.Minutes() < 60 {
		parts[0] = pluralize("minute", int64(math.Floor(ago.Minutes())))
		parts[1] = pluralize("second", int64(math.Floor(ago.Seconds()))%60)
	} else if ago.Hours() < 24 {
		parts[0] = pluralize("second", int64(math.Floor(ago.Hours())))
		parts[1] = pluralize("minute", int64(math.Floor(ago.Minutes()))%60)
	} else {
		parts[0] = pluralize("day", int64(math.Floor(ago.Hours()))/24)
		parts[1] = pluralize("hour", int64(math.Floor(ago.Minutes()))%24)
	}
	for _, part := range parts {
		if buff.Len() > 0 {
			buff.WriteString(", ")
		}
		if len(part) > 0 {
			buff.WriteString(part)
		}
	}
	if buff.Len() > 0 {
		return "(" + buff.String() + " ago)"
	}
	return ""
}

func layoutTime(when time.Time) string {
	var layout string
	if time.Since(when).Hours() < 12 && time.Now().YearDay() == when.YearDay() {
		layout = time.Kitchen
	} else {
		layout = time.Stamp
	}
	return fmt.Sprintf("%s %s", when.Format(layout), sinceMessage(when))
}

func timeMessage(pipeline *gitlab.Pipeline) string {
	if pipeline.Status == "pending" {
		return fmt.Sprintf("created at %s", layoutTime(*pipeline.CreatedAt))
	} else if pipeline.Status == "running" {
		return fmt.Sprintf("started at %s", layoutTime(*pipeline.StartedAt))
	} else {
		hours := pipeline.Duration / (60 * 60)
		minutes := (pipeline.Duration / 60) - (hours * 60)
		seconds := pipeline.Duration % 60
		finished := fmt.Sprintf("finished at %s\n", layoutTime(*pipeline.FinishedAt))
		if hours > 0 {
			return finished + fmt.Sprintf("duration:\t%d.%d:%d", hours, minutes, seconds)
		} else if minutes > 0 {
			return finished + fmt.Sprintf("duration:\t%d:%d", minutes, seconds)
		} else {
			return finished + fmt.Sprintf("duration:\t%d secs", pipeline.Duration)
		}
	}
}

func aliasFailures(f *flag.FlagSet, name string) flag.NormalizedName {
	switch name {
	case "failed":
		name = "failures"
		break
	}
	return flag.NormalizedName(name)
}

func init() {
	ciStatusCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote_branches")
	ciStatusCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	ciStatusCmd.Flags().BoolVarP(&wait, "wait", "w", false, "Continuously print the status and wait to exit until the pipeline finishes. Exit code indicates pipeline status")
	ciStatusCmd.Flags().BoolVarP(&noSkipped, "no-skipped", "", false, "Ignore skipped tests - do not print them")
	ciStatusCmd.Flags().BoolVarP(&lab.UseColor, "color", "c", false, "Use color for success and failure")
	ciStatusCmd.Flags().BoolVarP(&onlyFailures, "failures", "f", false, "Only print failures")
	ciStatusCmd.Flags().BoolVarP(&noCreated, "results-only", "r", false, "Only show completed and running tests. Does not report queued jobs")
	ciStatusCmd.Flags().BoolVarP(&summaryOnly, "summary", "s", false, "Do not show individual jobs, just the pipeline summary")
	ciStatusCmd.Flags().SetNormalizeFunc(aliasFailures)
	ciCmd.AddCommand(ciStatusCmd)
}
