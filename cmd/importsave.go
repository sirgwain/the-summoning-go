package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/importers/summoning"
	"github.com/spf13/cobra"
)

func init() {
	var packDir, out string
	var seed uint64
	cmd := &cobra.Command{
		Use:   "import-save <data-dir> <save-file>...",
		Short: "Convert saved games of The Summoning into engine saves",
		Long: `Reads saved games of The Summoning (SAVES/dNNN.sav) with the game's data files from
data-dir and the pack imported from them, and writes each as an engine save named after the
save's own name and its file, so the game can be carried on in the engine.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pack, err := content.Load(os.DirFS(packDir))
			if err != nil {
				return err
			}
			src, err := summoning.ReadSource(args[0])
			if err != nil {
				return err
			}
			si, err := summoning.NewSaveImporter(src)
			if err != nil {
				return err
			}
			if out == "" {
				if out = save.DefaultDir(pack.Manifest.ID); out == "" {
					return fmt.Errorf("no configuration directory for saves; give --out")
				}
			}
			for _, path := range args[1:] {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				sf, err := summoning.ReadSave(data)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				st, report, err := si.Import(pack, sf, seed)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				name := saveName(sf.Name, path)
				to := filepath.Join(out, name+".sav")
				if err := save.WriteState(to, name, st); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: wrote %q to %s\n", path, name, to)
				for _, l := range report.Lines() {
					fmt.Fprintln(cmd.OutOrStdout(), "  "+l)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&packDir, "pack", filepath.Join("packs", "summoning"), "the pack imported from the data files")
	cmd.Flags().StringVarP(&out, "out", "o", "", "directory to write the saves to (default: the game's save directory for the pack)")
	cmd.Flags().Uint64Var(&seed, "seed", 1, "seed for the imported worlds' random choices")
	rootCmd.AddCommand(cmd)
}

// saveName names an imported save after the original's name and file: "gwain (d212)". Characters
// a file name cannot hold become "_".
func saveName(name, path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = strings.TrimSpace(name)
	if name == "" {
		name = base
	} else {
		name = fmt.Sprintf("%s (%s)", name, base)
	}
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < ' ' {
			return '_'
		}
		return r
	}, name)
}
