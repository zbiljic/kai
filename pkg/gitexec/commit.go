package gitexec

import (
	"errors"
	"fmt"
	"os/exec"
)

type CommitOptions struct {
	CmdDir string

	Message string
}

func CommitCmd(opts *CommitOptions) *exec.Cmd {
	args := []string{"commit"}

	if opts.Message != "" {
		args = append(args, fmt.Sprintf("-m %s", opts.Message))
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = opts.CmdDir

	return cmd
}

func Commit(opts *CommitOptions) ([]byte, error) {
	if opts.CmdDir == "" {
		return nil, errors.New("missing command working directory")
	}

	cmd := CommitCmd(opts)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}

	return out, nil
}
