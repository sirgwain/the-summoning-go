package cmd

import (
	"fmt"
	"os"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/spf13/cobra"
)

func init() {
	var force bool
	var art string
	devpackCmd := &cobra.Command{
		Use:   "devpack [out-dir]",
		Short: "Write the dev pack, a pack made by code for tests and development",
		Long: `Writes the dev pack: plain drawn art and levels that lay out each of the engine's
mechanisms, made by code so that nothing needs the original game's data. Open it with
"summoning view packs/dev".

With --art, the dev pack is dressed in the art of a pack imported from the game: the same
levels and records, drawn with the game's tiles, figures, icons and effects. It is written to
packs/dev.game, which is not checked in.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "packs/dev"
			if art != "" {
				dir = "packs/dev.game"
			}
			if len(args) > 0 {
				dir = args[0]
			}
			if err := prepareOutput(dir, force); err != nil {
				return err
			}
			p := devpack.Build()
			if art != "" {
				from, err := content.Load(os.DirFS(art))
				if err != nil {
					return err
				}
				for _, m := range devpack.Dress(p, from, devpack.GameArt) {
					fmt.Println("not dressed:", m)
				}
			}
			if err := p.Verify(); err != nil {
				return err
			}
			if err := content.Write(p, dir); err != nil {
				return err
			}
			fmt.Println("wrote", p, "to", dir)
			return nil
		},
	}
	devpackCmd.Flags().StringVar(&art, "art", "", "dress the pack in the art of this imported game pack")
	devpackCmd.Flags().BoolVar(&force, "force", false, "replace the output directory if it exists")
	rootCmd.AddCommand(devpackCmd)
}
