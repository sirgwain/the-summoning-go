package content

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWAV(t *testing.T) {
	samples := []int16{0, 1000, -1000, 32767, -32768}
	b := EncodeWAV(samples, 44100)
	w, err := ReadWAV(b)
	if err != nil {
		t.Fatal(err)
	}
	if w != (WAV{Rate: 44100, Channels: 1, Bits: 16, Frames: 5}) {
		t.Fatalf("format %+v", w)
	}
	if b[44+4] != 0x18 || b[44+5] != 0xFC { // -1000, little-endian
		t.Fatalf("sample 2 bytes % X", b[48:50])
	}
	if _, err := ReadWAV(b[:30]); err == nil {
		t.Error("a truncated WAV: no error")
	}
	if _, err := ReadWAV([]byte("OggS....")); err == nil {
		t.Error("not a WAV: no error")
	}
}

func TestSoundWriteLoad(t *testing.T) {
	p := testPack()
	p.Sounds["music.3"] = &Sound{ID: "music.3", File: "3.wav", Intro: 2, Loop: 3, Data: EncodeWAV(make([]int16, 5), 44100)}
	p.Sounds["sfx.0"] = &Sound{ID: "sfx.0", File: "0.wav", Data: EncodeWAV([]int16{1, 2}, 44100)}
	if err := p.Verify(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(p, dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"audio/music/3.json", "audio/music/3.wav", "audio/sfx/0.json", "audio/sfx/0.wav"} {
		if _, err := os.Stat(dir + "/" + f); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	got, err := Load(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Sounds, p.Sounds) {
		t.Fatalf("sounds %+v, want %+v", got.Sounds, p.Sounds)
	}
}

func TestSoundVerify(t *testing.T) {
	wav := EncodeWAV(make([]int16, 10), 44100)
	for _, tc := range []struct {
		s    Sound
		want string
	}{
		{Sound{ID: "song.1", File: "1.wav", Data: wav}, "must start with"},
		{Sound{ID: "music.1", File: "../1.wav", Data: wav}, "beside the record"},
		{Sound{ID: "music.1", File: "1.mp3", Data: wav}, "beside the record"},
		{Sound{ID: "music.1", File: "1.wav", Intro: 6, Loop: 5, Data: wav}, "run past"},
		{Sound{ID: "music.1", File: "1.wav", Data: []byte("nope")}, "not a WAV"},
		{Sound{ID: "music.1", File: "1.wav", Loop: -1, Data: wav}, "negative"},
	} {
		p := testPack()
		s := tc.s
		p.Sounds[s.ID] = &s
		err := p.Verify()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: error %v, want %q", tc.s.File, err, tc.want)
		}
	}
	p := testPack()
	p.Sounds["music.2"] = &Sound{ID: "music.2", File: "2.ogg", Data: []byte("OggS")}
	if err := p.Verify(); err != nil {
		t.Errorf("an Ogg sound: %v", err)
	}
}
