package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/importers/summoning"
	"github.com/spf13/cobra"
)

func init() {
	var out string
	var force, noAudio bool
	importCmd := &cobra.Command{
		Use:   "import <data-dir>",
		Short: "Convert The Summoning's data files into a content pack",
		Long: `Reads a copy of The Summoning's data files from data-dir and writes a content pack.
The pack holds the game's art and data: keep it local and do not distribute it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := prepareOutput(out, force); err != nil {
				return err
			}
			src, err := summoning.ReadSource(args[0])
			if err != nil {
				return err
			}
			pack, report, err := summoning.Import(src, summoning.Options{NoAudio: noAudio})
			if err != nil {
				return err
			}
			if err := content.Write(pack, out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s to %s\n", pack, out)
			if lines := report.Lines(); len(lines) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "not mapped:")
				for _, l := range lines {
					fmt.Fprintln(cmd.OutOrStdout(), "  "+l)
				}
			}
			return nil
		},
	}
	importCmd.Flags().StringVarP(&out, "out", "o", filepath.Join("packs", "summoning"), "directory to write the pack to")
	importCmd.Flags().BoolVar(&force, "force", false, "replace an existing pack in the output directory")
	importCmd.Flags().BoolVar(&noAudio, "no-audio", false, "skip rendering music and sound effects (faster)")
	rootCmd.AddCommand(importCmd)
}

// prepareOutput makes sure dir can receive a pack: it must not exist, be empty, or (with force)
// hold a pack, which is removed.
func prepareOutput(dir string, force bool) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) || (err == nil && len(entries) == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "pack.json")); err != nil {
		return fmt.Errorf("%s is not empty and holds no pack; not writing there", dir)
	}
	if !force {
		return fmt.Errorf("%s already holds a pack; use --force to replace it", dir)
	}
	return os.RemoveAll(dir)
}
