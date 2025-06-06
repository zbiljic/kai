package gitexec

import (
	"errors"
	"os/exec"
)

// StatusOptions represents options for git status command.
type StatusOptions struct {
	CmdDir string

	Short     bool // Short format
	Porcelain bool // Porcelain format
}

// StatusCmd creates an exec.Cmd to execute git status with the given options.
func StatusCmd(opts *StatusOptions) *exec.Cmd {
	args := []string{"status"}

	if opts.Short {
		args = append(args, "--short")
	}

	if opts.Porcelain {
		args = append(args, "--porcelain")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = opts.CmdDir

	return cmd
}

// Status executes git status with the given options.
func Status(opts *StatusOptions) ([]byte, error) {
	if opts.CmdDir == "" {
		return nil, errors.New("missing command working directory")
	}

	cmd := StatusCmd(opts)

	return run(cmd)
}
