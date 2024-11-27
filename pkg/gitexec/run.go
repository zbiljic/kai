package gitexec

import "os/exec"

//nolint:unused
func run(cmd *exec.Cmd) ([]byte, error) {
	withSysProcAttr(cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}

	return out, nil
}
