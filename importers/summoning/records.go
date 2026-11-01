package summoning

import "fmt"

// Object template layout (items.md).
const objectTemplateSize = 15

// ObjectTemplate is one OBJECTS template.
type ObjectTemplate [objectTemplateSize]byte

// Template fields (items.md, Templates).
func (o *ObjectTemplate) Size() int       { return int(o[0] & 0x0F) }
func (o *ObjectTemplate) Armor() int      { return int(o[0] >> 4) }
func (o *ObjectTemplate) NameIndex() int  { return int(o[1]) }
func (o *ObjectTemplate) Weight() int     { return int(o[2]) }
func (o *ObjectTemplate) Container() bool { return o[3] != 0 }
func (o *ObjectTemplate) Capacity() int   { return int(o[3]&0x0F) + 1 }
func (o *ObjectTemplate) Largest() int    { return int(o[3] >> 4) }
func (o *ObjectTemplate) Charges() int    { return int(o[9]) }

// ActionIcon and DamageCode are action i's icon and damage code, for actions 0 and 1;
// DelayCode holds the actions' delays, a nibble each.
func (o *ObjectTemplate) ActionIcon(i int) int { return int(o[4+i]) }
func (o *ObjectTemplate) DamageCode(i int) int { return int(o[6+i]) }
func (o *ObjectTemplate) DelayCode() int       { return int(o[8]) }

// Power returns the template's power, and whether it has one.
func (o *ObjectTemplate) Power() (int, bool) { return int(o[12] & 0x7F), o[12]&0x80 != 0 }
func (o *ObjectTemplate) Image() int         { return int(o[10]) }
func (o *ObjectTemplate) NeverRandom() bool  { return o[11]&0x80 != 0 }
func (o *ObjectTemplate) Category() int      { return int(o[13] & 0x3F) }
func (o *ObjectTemplate) Weapon() bool       { return o[13]&0x40 != 0 }
func (o *ObjectTemplate) TwoHanded() bool    { return o[13]&0x80 != 0 }
func (o *ObjectTemplate) Scroll() int        { return int(o[14]) }

// Objects is OBJECTS: the item templates and the TEXT id of the first name.
type Objects struct {
	NameBase  int
	Templates []ObjectTemplate
}

// ReadObjects reads OBJECTS.
func ReadObjects(data []byte) (*Objects, error) {
	r := &reader{data: data}
	n, base := int(r.u16()), int(r.u16())
	o := &Objects{NameBase: base, Templates: make([]ObjectTemplate, n)}
	for i := range o.Templates {
		r.copy(o.Templates[i][:])
	}
	if r.err != nil {
		return nil, fmt.Errorf("objects: %w", r.err)
	}
	return o, nil
}

// NPC record layout (creatures.md).
const npcRecordSize = 16

// NPCRecord is one NPC record.
type NPCRecord [npcRecordSize]byte

// Record fields (creatures.md, NPC record).
func (n *NPCRecord) Conversation() int { return int(n[0]) }
func (n *NPCRecord) Portrait() int     { return int(n[1]) }
func (n *NPCRecord) HitBonus() int     { return int(n[2]) }
func (n *NPCRecord) HitDice() int      { return int(n[3]) }
func (n *NPCRecord) Resistance() int   { return int(n[4]) }
func (n *NPCRecord) Bonus() int        { return int(n[5]) }
func (n *NPCRecord) Direction() int    { return int(n[6]) }
func (n *NPCRecord) Bank() int         { return int(n[7]) }
func (n *NPCRecord) Frames() int       { return int(n[8]) }
func (n *NPCRecord) Accuracy() int     { return int(n[10]) }
func (n *NPCRecord) Attack() int       { return int(n[9]) }
func (n *NPCRecord) Spell() int        { return int(n[12]) }
func (n *NPCRecord) Vulnerability() int {
	return int(n[13])
}
func (n *NPCRecord) Interval() int    { return int(n[11]) }
func (n *NPCRecord) Behavior() uint16 { return uint16(n[14]) | uint16(n[15])<<8 }

// ReadNPCs reads NPC.
func ReadNPCs(data []byte) ([]NPCRecord, error) {
	r := &reader{data: data}
	recs := make([]NPCRecord, r.u16())
	for i := range recs {
		r.copy(recs[i][:])
	}
	if r.err != nil {
		return nil, fmt.Errorf("npc: %w", r.err)
	}
	return recs, nil
}
