package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

type (
	ctxKeyClackPromptStarted struct{}
)

func injectIntoCommandContextWithKey[K, V comparable](cmd *cobra.Command, key K, value V) {
	ctx := cmd.Context()
	ctx = context.WithValue(ctx, key, value)
	cmd.SetContext(ctx)
}

// setupGitWorkDir validates and returns the git working directory
func setupGitWorkDir() (string, error) {
	workDir, err := gitWorkingTreeDir(getWd())
	if err != nil {
		return "", errors.New("The current directory must be a Git repository") //nolint:staticcheck
	}
	return workDir, nil
}
