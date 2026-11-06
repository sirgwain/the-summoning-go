package viewer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sirgwain/the-summoning-go/devpack"
	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/engine/inspect"
	"github.com/sirgwain/the-summoning-go/engine/sim"
)

// labelPackages are the packages whose words are labels (docs/pack.md, Labels).
var labelPackages = []string{".", "window", "../inspect"}

// TestLabelsInSource checks the engine's panels keep no label's words of their own: no string
// in their code is a label's default, and each label key they name has a default.
func TestLabelsInSource(t *testing.T) {
	defaults := map[string]string{}
	for k, s := range inspect.Labels {
		// Words too plain to tell from code's own strings, short or a single lowercase word
		// ("pause", a tool's), are left out.
		if len(strings.TrimSpace(s)) > 2 && strings.ToLower(s) != s || strings.Contains(s, " ") {
			defaults[s] = k
		}
	}
	key := regexp.MustCompile(`^ui\.[a-z0-9_.]*[a-z0-9_]$`)
	for _, dir := range labelPackages {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") || filepath.Base(f) == "labels.go" {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if k, ok := defaults[s]; ok {
					t.Errorf("%s: %q is label %s's default", f, s, k)
				}
				if key.MatchString(s) {
					if _, ok := inspect.Labels[s]; !ok {
						t.Errorf("%s: label %s has no default", f, s)
					}
				}
				return true
			})
		}
	}
}

// TestLabelsDocumented checks pack.md lists every label.
func TestLabelsDocumented(t *testing.T) {
	doc, err := os.ReadFile("../../docs/pack.md")
	if err != nil {
		t.Fatal(err)
	}
	for k := range inspect.Labels {
		if !strings.Contains(string(doc), "`"+k+"`") {
			t.Errorf("pack.md does not list label %s", k)
		}
	}
}

// renamedPack returns the dev pack with every label renamed to its key in guillemets.
func renamedPack() *content.Pack {
	p := devpack.Build()
	for _, tx := range p.Texts {
		for k := range inspect.Labels {
			tx.Strings[k] = "«" + k + "»"
		}
	}
	return p
}

// TestLabelsRenamed walks the panels with a pack that renames every label and finds none of
// the engine's words left: the frontend's screens and the menus over play, the inventory, the
// hotbar, the help, the book's pages, the map's legend and the notices.
func TestLabelsRenamed(t *testing.T) {
	p := renamedPack()
	v, err := New(p, 1, "", 2, false, image.Pt(1280, 800))
	if err != nil {
		t.Fatal(err)
	}
	var shown []string
	add := func(s ...string) { shown = append(shown, s...) }
	front := func(f *Frontend) {
		add(f.Title)
		add(f.Lines...)
		for _, b := range f.Buttons {
			add(b.Label)
		}
	}

	f := NewFrontend(p, 1, []string{"one"}, v.Screen)
	front(f)
	for _, s := range []string{ScreenPortrait, ScreenName, ScreenMethod, ScreenWeapon, ScreenSchool, ScreenPremade, ScreenRestore} {
		f.show(s)
		front(f)
	}
	for m := range p.Creation.Methods {
		f.method, f.least = m, nil
		f.show(ScreenAttributes)
		front(f)
	}
	for _, kind := range []string{content.MenuSave, content.MenuDelete, content.MenuRename, content.MenuLeave, content.MenuQuit} {
		v.openMenu(kind)
		front(v.Front)
		if kind == content.MenuRename {
			v.Front.Rename = "one"
			v.Front.show(ScreenRenameTo)
			front(v.Front)
		}
	}
	v.Front = nil

	v.SetPlay(true)
	pl := v.World.Player()
	for _, l := range v.stats(pl, p.Inventory) {
		add(l.Text)
	}
	for _, l := range v.details(nil) {
		add(l.Text)
	}
	for _, id := range sortedItems(p) {
		add(v.objectDetails(&sim.Object{Item: id})...)
	}
	h := v.Hotbar()
	add(h.Log.Label, h.Book.Label)
	add(helpLines(p, PlayBindings, 80)...)
	add(helpLines(p, CommonBindings, 80)...)
	for k := range MapKinds {
		add(v.MapKindName(k))
	}
	add(v.levelName("nowhere"))

	for _, id := range sortedThings(p) {
		lines, hint := inspect.CreatureLines(p, id, inspect.FullEntry(&sim.Entry{Level: "l1"}, p.Things[id]))
		add(lines...)
		add(hint)
		_, hint = inspect.CreatureLines(p, id, &sim.Entry{})
		add(hint)
	}
	for _, id := range sortedItems(p) {
		lines, hint := inspect.ItemLines(p, id, &sim.Entry{Stage: sim.StageKilled})
		add(lines...)
		add(hint)
		_, hint = inspect.ItemLines(p, id, &sim.Entry{Stage: sim.StageHandled})
		add(hint)
	}

	v.SaveDir = ""
	v.SaveGame("x")
	add(v.Notice)

	// What is left once the pack's own words are taken out may hold no label's words.
	var texts []string
	for _, tx := range p.Texts {
		for _, s := range tx.Strings {
			texts = append(texts, s)
		}
	}
	for _, c := range p.Conversations {
		texts = append(texts, c.Name)
	}
	for _, m := range p.Creation.Methods {
		texts = append(texts, m.Name)
	}
	for _, pm := range p.Creation.Premade {
		texts = append(texts, pm.Name)
	}
	sort.Slice(texts, func(i, j int) bool { return len(texts[i]) > len(texts[j]) })
	var defaults []string
	for _, s := range inspect.Labels {
		if s = strings.TrimSpace(s); len(s) > 2 && (strings.ToLower(s) != s || strings.Contains(s, " ")) {
			defaults = append(defaults, s)
		}
	}
	for _, s := range shown {
		left := s
		for _, tx := range texts {
			if tx != "" {
				left = strings.ReplaceAll(left, tx, "")
			}
		}
		for _, d := range defaults {
			if strings.Contains(left, d) {
				t.Errorf("%q shows the default %q", s, d)
			}
		}
	}
	if !slices.ContainsFunc(shown, func(s string) bool { return strings.Contains(s, "«ui.hit_points»") }) {
		t.Error("the renamed hit points label is not shown")
	}
}

func sortedItems(p *content.Pack) []string {
	var ids []string
	for id := range p.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedThings(p *content.Pack) []string {
	var ids []string
	for id := range p.Things {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
