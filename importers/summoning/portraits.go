package summoning

import (
	"fmt"

	"github.com/sirgwain/the-summoning-go/engine/content"
)

// F layout (formats.md, F).
const (
	portraitRowSize  = 8
	premadeSize      = 35
	figureColorCount = 4
)

// PortraitRow is one of F's portrait rows.
type PortraitRow [portraitRowSize]byte

// Portrait row fields.
func (r *PortraitRow) Mirrored() bool { return r[0] != 0 }
func (r *PortraitRow) FigureColors() []int {
	c := make([]int, figureColorCount)
	for i := range c {
		c[i] = int(r[1+i])
	}
	return c
}
func (r *PortraitRow) Other() int   { return int(r[5]) }
func (r *PortraitRow) Talk() int    { return int(r[6]) }
func (r *PortraitRow) Female() bool { return r[7] != 0 }

// Premade is one of F's premade characters.
type Premade struct {
	Name     string
	Portrait int
	// Attributes are STR, AGI, END, ACC, TAL, POW.
	Attributes          [6]int
	School, WeaponClass int
}

// ReadPremades reads F's premade characters.
func ReadPremades(data []byte) ([]Premade, error) {
	r := &reader{data: data}
	rows := int(r.u16())
	r.u16()
	r.u16()
	n := int(r.u16())
	r.bytes(rows * portraitRowSize)
	out := make([]Premade, n)
	for i := range out {
		b := r.bytes(premadeSize)
		p := Premade{Name: decode437([]byte(cString(b[:26]))), Portrait: int(b[26]), School: int(b[33]), WeaponClass: int(b[34])}
		for a := range p.Attributes {
			p.Attributes[a] = int(b[27+a])
		}
		out[i] = p
	}
	if r.err != nil {
		return nil, fmt.Errorf("f: %w", r.err)
	}
	return out, nil
}

// ReadPortraits reads F's portrait rows.
func ReadPortraits(data []byte) ([]PortraitRow, error) {
	r := &reader{data: data}
	rows := make([]PortraitRow, r.u16())
	r.u16()
	r.u16()
	premades := int(r.u16())
	for i := range rows {
		r.copy(rows[i][:])
	}
	if r.err == nil && r.remaining() < premades*premadeSize {
		return nil, fmt.Errorf("f: %d bytes for %d premade characters", r.remaining(), premades)
	}
	if r.err != nil {
		return nil, fmt.Errorf("f: %w", r.err)
	}
	return rows, nil
}

func portraitID(i int) string { return fmt.Sprintf("portrait.%d", i) }

// portraits adds a portrait for each of F's rows, its image the people bank of the same number
// (player.md, Portrait), and gives the player the first.
func (im *importer) portraits(rows []PortraitRow) {
	ref := func(i int) string {
		if i < 0 || i >= len(rows) {
			im.report.addf("portraits: no portrait %d", i)
			return ""
		}
		return portraitID(i)
	}
	for i, row := range rows {
		sid, err := im.stripSprite(int(im.init.Bases.People)+i, "sprite.portrait")
		if err != nil {
			im.report.addf("portrait %d: %v; left out", i, err)
			continue
		}
		pt := &content.Portrait{
			ID: portraitID(i), Image: content.Image{Sprite: sid}, Mirrored: row.Mirrored(),
			FigureColors: row.FigureColors(), Female: row.Female(),
		}
		if row.Other() != i {
			pt.Other = ref(row.Other())
		}
		if row.Talk() != i {
			pt.Talk = ref(row.Talk())
		}
		im.pack.Portraits[pt.ID] = pt
	}
	for _, pt := range im.pack.Portraits {
		if _, ok := im.pack.Portraits[pt.Other]; !ok {
			pt.Other = ""
		}
		if _, ok := im.pack.Portraits[pt.Talk]; !ok {
			pt.Talk = ""
		}
	}
	if t := im.pack.Things[playerID]; t != nil && im.pack.Portraits[portraitID(0)] != nil {
		t.Portrait = portraitID(0)
	}
}

// stripSprite adds bank id, a strip, as one image named prefix.id: the bank's frames are its
// rows, frame 0 at the top, each as high as a frame (formats.md, Strips).
func (im *importer) stripSprite(id int, prefix string) (string, error) {
	sid := fmt.Sprintf("%s.%d", prefix, id)
	if _, ok := im.pack.Sprites[sid]; ok {
		return sid, nil
	}
	b, err := im.bank(id)
	if err != nil {
		return "", err
	}
	var pix []byte
	for _, f := range b.Frames {
		pix = append(pix, f...)
	}
	im.addSprite(sid, [][]byte{pix}, b.Width, b.Height*len(b.Frames), b.Transparent)
	return sid, nil
}
