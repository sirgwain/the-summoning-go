package cmd

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/spf13/cobra"
)

func init() {
	var force bool
	devpackCmd := &cobra.Command{
		Use:   "devpack [out-dir]",
		Short: "Write the dev pack, a pack made by code for tests and development",
		Long: `Writes the dev pack: plain drawn art and levels that lay out each of the engine's
mechanisms, made by code so that nothing needs the original game's data. Open it with
"summoning view packs/dev".`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "packs/dev"
			if len(args) > 0 {
				dir = args[0]
			}
			if err := prepareOutput(dir, force); err != nil {
				return err
			}
			p := devpack.Build()
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
	devpackCmd.Flags().BoolVar(&force, "force", false, "replace the output directory if it exists")
	rootCmd.AddCommand(devpackCmd)
}
