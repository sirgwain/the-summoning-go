package content

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path"
	"strings"
)

// Sound is a music track or a sound effect: an audio file and how it loops.
type Sound struct {
	ID string `json:"id"`
	// File is the audio file beside the record, WAV or Ogg Vorbis.
	File string `json:"file"`
	// Intro and Loop count samples of the file. The sound plays its first Intro samples, then
	// repeats the Loop samples after them until stopped. With Loop 0 it plays the whole file
	// once.
	Intro int `json:"intro,omitempty"`
	Loop  int `json:"loop,omitempty"`
	// Data is the audio file's contents.
	Data []byte `json:"-"`
}

// Sound id prefixes and where each kind lives.
const (
	MusicPrefix = "music."
	SFXPrefix   = "sfx."
	musicDir    = "audio/music"
	sfxDir      = "audio/sfx"
	cuesFile    = "audio/cues.json"
)

// Cues maps the engine's sound cues to sounds.
type Cues struct {
	ID   string            `json:"id"`
	Cues map[string]string `json:"cues"`
}

// Engine sound cues: what the simulation names when it makes a sound.
const (
	CueLever         = "lever"
	CueFootstep      = "footstep"
	CueDoor          = "door"
	CueUnseal        = "unseal"
	CueTeleport      = "teleport"
	CueLevelUp       = "level_up"
	CueMiss          = "miss"
	CueHit           = "hit"
	CueHurt          = "hurt"
	CueBow           = "bow"
	CueShatter       = "shatter"
	CueTake          = "take"
	CueDrop          = "drop"
	CueCast          = "cast"
	CueGesture       = "gesture"
	CueWeaponSkillUp = "weapon_skill_up"
	CueMagicSkillUp  = "magic_skill_up"
)

var cueKeys = map[string]bool{
	CueLever: true, CueFootstep: true, CueDoor: true, CueUnseal: true, CueTeleport: true, CueLevelUp: true,
	CueWeaponSkillUp: true, CueMagicSkillUp: true, CueMiss: true, CueHit: true, CueHurt: true, CueBow: true,
	CueShatter: true, CueTake: true, CueDrop: true, CueCast: true, CueGesture: true,
}

// soundDir is the directory of a sound record by its id, or "" for an id of no sound kind.
func soundDir(id string) (dir, prefix string) {
	switch {
	case strings.HasPrefix(id, MusicPrefix):
		return musicDir, MusicPrefix
	case strings.HasPrefix(id, SFXPrefix):
		return sfxDir, SFXPrefix
	}
	return "", ""
}

func validSoundFile(name string) bool {
	return name != "" && path.Base(name) == name && (strings.HasSuffix(name, ".wav") || strings.HasSuffix(name, ".ogg"))
}

// WAV holds what the engine needs of a WAV file's format.
type WAV struct {
	Rate, Channels, Bits int
	// Frames is the number of samples per channel.
	Frames int
}

// EncodeWAV encodes mono 16-bit samples as a WAV file.
func EncodeWAV(samples []int16, rate int) []byte {
	data := len(samples) * 2
	b := make([]byte, 44+data)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+data))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], uint32(rate))
	binary.LittleEndian.PutUint32(b[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(data))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[44+2*i:], uint16(s))
	}
	return b
}

// ReadWAV reads the format of a PCM WAV file.
func ReadWAV(b []byte) (WAV, error) {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return WAV{}, errors.New("not a WAV file")
	}
	var w WAV
	var haveFormat bool
	for pos := 12; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4:]))
		body := pos + 8
		if size < 0 || body+size > len(b) {
			return WAV{}, fmt.Errorf("WAV chunk %q runs past the file", id)
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return WAV{}, errors.New("WAV format chunk too short")
			}
			if binary.LittleEndian.Uint16(b[body:]) != 1 {
				return WAV{}, errors.New("WAV is not PCM")
			}
			w.Channels = int(binary.LittleEndian.Uint16(b[body+2:]))
			w.Rate = int(binary.LittleEndian.Uint32(b[body+4:]))
			w.Bits = int(binary.LittleEndian.Uint16(b[body+14:]))
			haveFormat = true
		case "data":
			if !haveFormat || w.Channels == 0 || w.Bits == 0 {
				return WAV{}, errors.New("WAV data before its format")
			}
			w.Frames = size / (w.Channels * w.Bits / 8)
			return w, nil
		}
		pos = body + size + size&1
	}
	return WAV{}, errors.New("WAV has no data")
}

func (v *verifier) sound(s *Sound) {
	if dir, _ := soundDir(s.ID); dir == "" {
		v.fail("sound %s: id must start with %q or %q", s.ID, MusicPrefix, SFXPrefix)
	}
	if !validSoundFile(s.File) {
		v.fail("sound %s: file %q is not a .wav or .ogg beside the record", s.ID, s.File)
		return
	}
	if s.Intro < 0 || s.Loop < 0 {
		v.fail("sound %s: negative intro or loop", s.ID)
	}
	if !strings.HasSuffix(s.File, ".wav") {
		return
	}
	w, err := ReadWAV(s.Data)
	if err != nil {
		v.fail("sound %s: %v", s.ID, err)
		return
	}
	if w.Bits != 16 || w.Channels < 1 || w.Channels > 2 {
		v.fail("sound %s: %d channels of %d bits, want 16-bit mono or stereo", s.ID, w.Channels, w.Bits)
	}
	if s.Intro+s.Loop > w.Frames {
		v.fail("sound %s: intro %d and loop %d run past the file's %d samples", s.ID, s.Intro, s.Loop, w.Frames)
	}
}

func (v *verifier) cues() {
	c := v.p.Cues
	if c == nil {
		return
	}
	for _, key := range sortedKeys(c.Cues) {
		if !cueKeys[key] {
			v.fail("%s: unknown cue %q", cuesFile, key)
		}
		if v.p.Sounds[c.Cues[key]] == nil {
			v.fail("%s: cue %s plays %q, which is not a sound", cuesFile, key, c.Cues[key])
		}
	}
}

// music fails unless id is empty or a music track of the pack.
func (v *verifier) music(owner, id string) {
	if id != "" && (!strings.HasPrefix(id, MusicPrefix) || v.p.Sounds[id] == nil) {
		v.fail("%s: %q is not a music track", owner, id)
	}
}

// effect fails unless id is empty or a sound of the pack.
func (v *verifier) effect(owner, id string) {
	if id != "" && v.p.Sounds[id] == nil {
		v.fail("%s: %q is not a sound", owner, id)
	}
}

// cue fails unless name is empty or an engine cue.
func (v *verifier) cue(owner, name string) {
	if name != "" && !cueKeys[name] {
		v.fail("%s: unknown cue %q", owner, name)
	}
}
