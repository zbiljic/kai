package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/orochaa/go-clack/prompts"
	"github.com/spf13/cobra"

	"github.com/zbiljic/kai/internal/buildinfo"
	"github.com/zbiljic/kai/pkg/versioninfo"
)

// AppName - the name of the application.
const AppName = "kai"

var rootCmd = &cobra.Command{
	Use:   AppName,
	Short: "Generate Git commit message using AI",
	Long:  `Generate Git commit message using AI`,
	Version: versioninfo.Info{
		Version: buildinfo.Version,
		Commit:  buildinfo.GitCommit,
		BuiltBy: buildinfo.BuiltBy,
	}.String(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
		cmd.SetContext(ctx)
	},
	RunE:          runRootE,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called my main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if cmd, err := rootCmd.ExecuteC(); err != nil {
		if strings.Contains(err.Error(), "arg(s)") || strings.Contains(err.Error(), "usage") {
			cmd.Usage() //nolint:errcheck
		}

		val, ok := cmd.Context().Value(ctxKeyClackPromptStarted{}).(bool)
		if ok && val {
			prompts.ExitOnError(err)
		} else {
			cobra.CheckErr(err)
		}
	}
}

func runRootE(cmd *cobra.Command, args []string) error {
	switch {
	case isGenCmd():
		return runGenE(cmd, args)
	default:
		cmd.Usage() //nolint:errcheck
		return nil
	}
}
