package cmd

import (
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zbiljic/kai/pkg/gitexec"
)

var (
	filesToExclude = []string{
		"*.lock*", // yarn.lock, Cargo.lock, Gemfile.lock, Pipfile.lock, etc.
		"go*.sum",
		"package-lock.json",
		"pnpm-lock.yaml",
	}

	excludeFromDiff = slice.FlatMap(filesToExclude, func(i int, s string) []string {
		return []string{":(exclude)" + s}
	})
)

func gitWorkingTreeDir(path string) (string, error) {
	out, err := gitexec.RevParse(&gitexec.RevParseOptions{
		CmdDir:       path,
		ShowTopLevel: true,
	})
	if err != nil {
		return string(out), err
	}

	return strings.TrimSpace(string(out)), nil
}

func gitDiffStaged(path string) ([]string, string, error) {
	out, err := gitexec.Diff(&gitexec.DiffOptions{
		CmdDir:   path,
		Cached:   true,
		Minimal:  true,
		NameOnly: true,
		Path:     excludeFromDiff,
	})
	if err != nil {
		return []string{}, "", err
	}

	outString := strings.TrimSpace(string(out))
	if outString == "" {
		return []string{}, "", nil
	}

	files := strings.Split(outString, "\n")

	out, err = gitexec.Diff(&gitexec.DiffOptions{
		CmdDir:  path,
		Cached:  true,
		Minimal: true,
		Path:    excludeFromDiff,
	})
	if err != nil {
		return []string{}, "", err
	}

	diff := strings.TrimSpace(string(out))

	return files, diff, nil
}

func gitCommit(path, message string) error {
	_, err := gitexec.Commit(&gitexec.CommitOptions{
		CmdDir:  path,
		Message: message,
	})
	if err != nil {
		return err
	}

	return nil
}
