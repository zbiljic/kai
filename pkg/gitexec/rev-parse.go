package gitexec

import (
	"errors"
	"os/exec"
)

type RevParseOptions struct {
	CmdDir string

	ShowTopLevel bool
}

func RevParseCmd(opts *RevParseOptions) *exec.Cmd {
	args := []string{"rev-parse"}

	if opts.ShowTopLevel {
		args = append(args, "--show-toplevel")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = opts.CmdDir

	return cmd
}

func RevParse(opts *RevParseOptions) ([]byte, error) {
	if opts.CmdDir == "" {
		return nil, errors.New("missing command working directory")
	}

	cmd := RevParseCmd(opts)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}

	return out, nil
}
