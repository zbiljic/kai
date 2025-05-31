package gitexec

import (
	"errors"
	"os/exec"
)

// AddOptions represents the options for the git add command.
type AddOptions struct {
	CmdDir string

	All bool

	Path []string
}

// AddCmd creates an *exec.Cmd for the git add command.
func AddCmd(opts *AddOptions) *exec.Cmd {
	args := []string{"add"}

	if opts.All {
		args = append(args, "--all")
	} else {
		args = append(args, opts.Path...)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = opts.CmdDir

	return cmd
}

func Add(opts *AddOptions) ([]byte, error) {
	if opts.CmdDir == "" {
		return nil, errors.New("missing command working directory")
	}

	if !opts.All && len(opts.Path) == 0 {
		return nil, errors.New("no files specified to add")
	}

	cmd := AddCmd(opts)

	return run(cmd)
}
