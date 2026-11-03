package cmd

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/audio/speaker"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/save"
	"github.com/sirgwain/the-summoning-go/engine/viewer"
	"github.com/sirgwain/the-summoning-go/engine/viewer/window"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newViewCmd(false), newViewCmd(true))
}

// newViewCmd returns the view command, or with play the play command, which opens the same
// window driving the player.
func newViewCmd(play bool) *cobra.Command {
	var level, screenshot, script, out, saves, load string
	var scale, shotTicks int
	var magnify, menu, mute bool
	var at, sel []int
	var seed uint64
	use, short, long := "view [pack-dir]", "Look around a content pack's levels and work their mechanisms",
		`Opens a window showing a pack's levels in the game's projection, running the simulation.
F12 switches to playing (see summoning play --help).
`
	if play {
		use, short, long = "play [pack-dir]", "Play a content pack's levels",
			`Opens a window playing a pack: the player starts where the pack says and the view follows it,
showing what the player sees.

A pack with character creation starts at the main menu (--menu=false skips it): create a
character (the descendant, the name, the attributes rolled or spent, the weapon and the school),
restore a saved game, or select a premade character. Arrows and Enter or the mouse choose;
Escape goes back. A new game shows the pack's opening scene (a key or click skips it). When the
player dies, or the game ends, play returns to the main menu.

The keys are laid out as the game's:

`+viewer.HelpText(viewer.PlayBindings)+`
F takes the top object of the player's cell to the first empty shelf. Enter pulls the lever
underfoot or puts the key a keyhole underfoot takes into it. E or C talks to the nearest creature
that talks, or hears the nearest mouth. Holding N naps: the world runs 32 times as fast, to get
magic points back, until N is released, the magic points are full, a creature comes near, the
player is hurt, a message or conversation starts, or another key or a click comes; it will not
start with full magic points or a creature near. Changing level saves the game too (the
autosave). The options menu saves under a name, restores, deletes or renames a save, leaves for
the main menu, or quits. The level map has a legend (a key or click closes it); the minimap in
the top-left corner shows it small. Some levels refuse the map.

F1-F4 cast the selected spell of Wizardry, Sorcery, Enchantment and Healing. In the spell
panel, type the letters A-L of a gesture (or click its icons), Backspace takes one off, Enter or
Learn tries it; clicking a spell of the spellbook memorizes it once more; Escape closes it.

In a conversation, a key or click goes on past each line, and clicking a marked word, or typing
its first letter, asks it next. Then click an offered keyword, or type a word and press Enter;
Escape, a right click or Bye leaves.

The hotbar along the bottom shows the portrait, hit points, magic points and load, the hands
with their action boxes (each hand's actions, then throwing), and the twenty shelves. Clicking
the player takes the top object underfoot onto the pointer, or puts the pointer's object down
(stand on a keyhole and click the player to put a key in); clicking a hand or shelf swaps it
with the pointer; clicking an action box acts with that hand. The right button puts the
pointer's object on the first empty shelf, or takes the object underfoot straight there; on a
box with nothing on the pointer, it uses a potion or other usable object where it lies, with
the right hand's delay. Shift and a click reads an object (a shelf, a hand, or the pointer's on
the player): a scroll shows its picture and teaches its icons, as selecting it (taking it from a
shelf onto the pointer) does. The four boxes after the shelves are the schools' selected
spells: click to cast, right click for the next; the box at the end shows the gesture playing,
and opens the spell panel. Over a creature that talks, or a mouth (a skull), the pointer becomes
a speech bubble: a click talks to the creature, or shows the mouth's message beside it. Taking a
scroll onto the pointer reads it.

The inventory panel shows the figure wearing what it wears, with a box for each worn slot
(head, body, arms, feet on the left; quiver, neck, right hand, left hand on the right), the pack
(an opened container, or a bag held in the left hand), the details of the object under the
pointer or on it, and the character's attributes, points, armor, level, experience and skills.
Its boxes work as the hotbar's: a click swaps a box with the pointer. A right click on a bag or
chest (on the hotbar or the panel, with nothing on the pointer) opens it as the pack, where its
contents can be taken and put; a second right click closes it, and the bag in the left hand is
the pack again. Resting the pointer on a box names its object, and for a container lists what
it holds.

The debug overlay is hidden at first (--debug shows it): the text, the selection's glow and
links, and the white glow of what is under the mouse. Clicking a cell inspects it.
`
	}
	viewCmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long: long + `
The tools:

` + viewer.HelpText(viewer.ToolBindings) + `
Click a cell or a thing to inspect it. It glows, and arrows show its links: what its feature's
hooks do (green switches on, red off, orange over; violet rewires a hook, blue sets a threshold,
an item or a message, cyan changes a cell, pink shoots, tan rolls a boulder, purple is where it
sends things), dashed for what a rewired hook would do. What acts on the selection shows the
same way and is listed in the panel, and everything a link reaches glows in its color.

Using a plate adds the weight the plate lacks to go down, and again takes that weight off,
leaving what is on it; using a slot puts in the item the slot takes. Reset sets a feature back
to how the level starts it (its state, hooks and settings; a plate keeps what lies on it); 1-9
run its hooks' procedures; X takes the cell's top object. N steps while paused.

In play and with the tools:

` + viewer.HelpText(viewer.CommonBindings) + `
Zooming with the wheel keeps the cell under the cursor in place, and + and - zoom around the
center. A flipbook plays in play too: a framed one as a view effect, placed by the selected
thing, else the selected cell, else the player; any other as a sparkle. The world waits while it
plays; Escape or a right click cuts it short where it allows. The world's own effects and
sparkles play the same way.

--script runs a script of input, screenshots and state dumps (docs/design.md, Testing) in place
of live input, and exits when it ends. Its files are written under --out.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "packs/summoning"
			if len(args) > 0 {
				dir = args[0]
			}
			pack, err := content.Load(os.DirFS(dir))
			if err != nil {
				return err
			}
			cell := func(name string, xy []int) (*content.Cell, error) {
				switch len(xy) {
				case 0:
					return nil, nil
				case 2:
					return &content.Cell{xy[0], xy[1]}, nil
				}
				return nil, fmt.Errorf("--%s wants a cell as x,y", name)
			}
			center, err := cell("at", at)
			if err != nil {
				return err
			}
			selected, err := cell("select", sel)
			if err != nil {
				return err
			}
			var steps viewer.Script
			if script != "" {
				if steps, err = viewer.LoadScript(script); err != nil {
					return err
				}
			}
			if screenshot != "" {
				steps = append(steps, viewer.Step{At: int64(shotTicks), Shot: screenshot})
			}
			v, err := viewer.New(pack, seed, level, scale, magnify, image.Pt(windowWidth, windowHeight))
			if err != nil {
				return err
			}
			v.SaveDir = saves
			v.HideDebug = play && !debugEnabled
			if !mute && script == "" && screenshot == "" {
				v.Audio = audio.New(speaker.New(), pack)
			}
			if v.SaveDir == "" && script == "" {
				v.SaveDir = save.DefaultDir(pack.Manifest.ID)
			}
			if load != "" {
				if err := v.LoadFile(savePath(load, v.SaveDir)); err != nil {
					return err
				}
				menu = false
			}
			if play {
				if v.World.Player() == nil {
					return fmt.Errorf("pack %s has no player", pack.Manifest.ID)
				}
				v.SetPlay(true)
				if menu && script == "" && pack.Creation != nil {
					v.OpenFrontend()
				}
			}
			if center != nil {
				v.CenterOn(*center)
			}
			if selected != nil {
				v.Select(*selected)
				if center == nil {
					v.CenterOn(*selected)
				}
			}
			w, err := window.New(v)
			if err != nil {
				return err
			}
			if steps != nil {
				w.SetScript(viewer.NewRunner(steps, out))
			}
			ebiten.SetWindowTitle("The Summoning: " + pack.Manifest.Name)
			ebiten.SetWindowSize(windowWidth, windowHeight)
			ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
			if err := ebiten.RunGame(w); err != nil && !errors.Is(err, ebiten.Termination) {
				return err
			}
			return nil
		},
	}
	viewCmd.Flags().StringVar(&level, "level", "", "level id to open (default: the start level)")
	viewCmd.Flags().IntVar(&scale, "scale", 2, "pixels per art pixel")
	viewCmd.Flags().IntSliceVar(&at, "at", nil, "center the view on this cell, as x,y")
	viewCmd.Flags().IntSliceVar(&sel, "select", nil, "select this cell, as x,y")
	viewCmd.Flags().BoolVar(&magnify, "magnify", false, "start with objects on the floor magnified")
	viewCmd.Flags().Uint64Var(&seed, "seed", 1, "seed for the simulation's random choices")
	viewCmd.Flags().StringVar(&script, "script", "", "run this script instead of live input, then exit")
	viewCmd.Flags().StringVar(&out, "out", ".", "directory for a script's screenshots and dumps")
	viewCmd.Flags().StringVar(&load, "load", "", "start from the saved game in this file, looked for in the saves directory when not found as given")
	viewCmd.Flags().StringVar(&saves, "saves", "", "directory for saved games (default: the user's config directory; none while a script runs)")
	if play {
		viewCmd.Flags().BoolVar(&menu, "menu", true, "start at the main menu when the pack has character creation (not while a script runs)")
	}
	viewCmd.Flags().BoolVar(&mute, "mute", false, "play no music or sound effects (none while a script runs)")
	viewCmd.Flags().StringVar(&screenshot, "screenshot", "", "save a frame to this PNG file and exit")
	viewCmd.Flags().IntVar(&shotTicks, "screenshot-ticks", 0, "updates to run before the screenshot")
	return viewCmd
}

const windowWidth, windowHeight = 1280, 800

// savePath returns path when a file is there, or else the file of that name in the saves
// directory dir when one is there, so --load takes a bare save name like quick.sav.
func savePath(path, dir string) string {
	if fileExists(path) || dir == "" || filepath.IsAbs(path) {
		return path
	}
	if alt := filepath.Join(dir, path); fileExists(alt) {
		return alt
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
