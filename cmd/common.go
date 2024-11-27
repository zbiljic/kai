package cmd

import (
	"context"

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
