package cmd

import (
	"fmt"
	"os"

	"github.com/sirgwain/the-summoning-go/version"
	"github.com/spf13/cobra"
)

// semver is the semantic-release semver (added at compile time)
var (
	semver    = version.Semver
	commit    = version.Commit
	buildTime = version.BuildTime
)

var debugEnabled bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version:           semver,
	Use:               "the-summoning",
	Short:             "A Summoning clone",
	Long:              `A Summoning clone`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show usage
		cmd.Help()
		os.Exit(1)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("version: {{ .Version }}\nbuild: %s (%s)", commit, buildTime))
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// all commands have debug mode
	rootCmd.PersistentFlags().BoolVar(&debugEnabled, "debug", false, "enable debug logging")
}
