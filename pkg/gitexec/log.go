package gitexec

import (
	"errors"
	"fmt"
	"os/exec"
)

// LogOptions represents options for git log command.
type LogOptions struct {
	CmdDir string

	MaxCount int    // Number of commits to retrieve
	Format   string // Format of the log output

	Path []string // Show only commits that affected the specified paths
}

// LogCmd creates an exec.Cmd to execute git log with the given options.
func LogCmd(opts *LogOptions) *exec.Cmd {
	args := []string{"log"}

	if opts.MaxCount > 0 {
		args = append(args, "--max-count", fmt.Sprintf("%d", opts.MaxCount))
	}

	if opts.Format != "" {
		args = append(args, "--pretty=format:"+opts.Format)
	}

	if len(opts.Path) > 0 {
		args = append(args, "--")
		args = append(args, opts.Path...)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = opts.CmdDir

	return cmd
}

// Log executes git log with the given options.
func Log(opts *LogOptions) ([]byte, error) {
	if opts.CmdDir == "" {
		return nil, errors.New("missing command working directory")
	}

	cmd := LogCmd(opts)

	return run(cmd)
}
