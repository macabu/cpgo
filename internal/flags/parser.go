package flags

import (
	"bytes"
	"flag"
	"fmt"
)

type Flags struct {
	GithubToken string
	ConfigPath  string
	LogVerbose  bool
}

// Parse command line flags into a flags.Flags struct.
// API choice taken from: https://eli.thegreenplace.net/2020/testing-flag-parsing-in-go-programs/
func Parse(programName string, args []string) (Flags, string, error) {
	var (
		buf   bytes.Buffer
		flags Flags
	)

	flagSet := flag.NewFlagSet(programName, flag.ContinueOnError)
	flagSet.SetOutput(&buf)

	flagSet.StringVar(
		&flags.GithubToken,
		"githubToken",
		"",
		"The Github token to be able to read the repositories and create the pull requests",
	)

	flagSet.StringVar(
		&flags.ConfigPath,
		"configPath",
		"./config.yaml",
		"The path (to) including the name of the config file with extension. Defaults to: ./config.yaml",
	)

	flagSet.BoolVar(&flags.LogVerbose, "verbose", false, "Whether to log debug messages")

	if err := flagSet.Parse(args); err != nil {
		return Flags{}, buf.String(), fmt.Errorf("flagSet.Parse: %w", err)
	}

	if flags.GithubToken == "" {
		return Flags{}, buf.String(), ErrGitHubTokenNotFound
	}

	return flags, buf.String(), nil
}
