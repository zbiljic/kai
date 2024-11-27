package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// isNotTerminal defines if the output is going into terminal or not.
// It's dynamically set to false or true based on the stdout's file
// descriptor referring to a terminal or not.
var isNotTerminal = os.Getenv("TERM") == "dumb" ||
	(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))

func init() {
	// if the output is not directed to a terminal return an error
	if isNotTerminal {
		cobra.CheckErr(errors.New("not a terminal"))
	}
}

// getWd is a convenience method to get the working directory.
func getWd() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %s", err.Error())
		cobra.CheckErr(err)
	}

	return dir
}
