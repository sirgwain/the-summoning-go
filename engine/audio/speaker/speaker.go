// Package speaker plays sounds through Ebitengine: the backend of engine/audio.
package speaker

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

	"github.com/sirgwain/the-summoning-go/engine/audio"
	"github.com/sirgwain/the-summoning-go/engine/content"
)

// SampleRate is the rate the speaker plays at.
const SampleRate = 44100

// frameBytes is the size of a decoded frame: 16-bit stereo.
const frameBytes = 4

// cacheLimit is the largest decoded sound kept for the next time it plays: effects, not tracks.
const cacheLimit = 1 << 20

// Backend plays sounds through Ebitengine's audio context.
type Backend struct {
	ctx   *eaudio.Context
	cache map[string]decoded
}

// New returns a backend on Ebitengine's audio context, making it if there is none.
func New() *Backend {
	ctx := eaudio.CurrentContext()
	if ctx == nil {
		ctx = eaudio.NewContext(SampleRate)
	}
	return &Backend{ctx: ctx, cache: map[string]decoded{}}
}

// Play starts sound s.
func (b *Backend) Play(s *content.Sound) (audio.Voice, error) {
	pcm, ratio, err := b.decode(s)
	if err != nil {
		return nil, fmt.Errorf("sound %s: %w", s.ID, err)
	}
	var p *eaudio.Player
	if s.Loop > 0 {
		at := func(frames int) int64 { return int64(float64(frames)*ratio) * frameBytes }
		intro, loop := at(s.Intro), at(s.Intro+s.Loop)-at(s.Intro)
		if intro+loop > int64(len(pcm)) {
			loop = int64(len(pcm)) - intro
		}
		if p, err = b.ctx.NewPlayer(eaudio.NewInfiniteLoopWithIntro(bytes.NewReader(pcm), intro, loop)); err != nil {
			return nil, err
		}
	} else {
		p = b.ctx.NewPlayerFromBytes(pcm)
	}
	p.Play()
	return voice{p}, nil
}

// decode returns s as 16-bit stereo at the context's rate, and the context's rate over the
// file's.
func (b *Backend) decode(s *content.Sound) ([]byte, float64, error) {
	if d, ok := b.cache[s.ID]; ok {
		return d.pcm, d.ratio, nil
	}
	rate := b.ctx.SampleRate()
	var src int
	var stream io.Reader
	if strings.HasSuffix(s.File, ".ogg") {
		raw, err := vorbis.DecodeWithoutResampling(bytes.NewReader(s.Data))
		if err != nil {
			return nil, 0, err
		}
		src = raw.SampleRate()
		if stream, err = vorbis.DecodeWithSampleRate(rate, bytes.NewReader(s.Data)); err != nil {
			return nil, 0, err
		}
	} else {
		w, err := content.ReadWAV(s.Data)
		if err != nil {
			return nil, 0, err
		}
		src = w.Rate
		if stream, err = wav.DecodeWithSampleRate(rate, bytes.NewReader(s.Data)); err != nil {
			return nil, 0, err
		}
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, 0, err
	}
	d := decoded{pcm[:len(pcm)-len(pcm)%frameBytes], float64(rate) / float64(src)}
	if len(d.pcm) <= cacheLimit {
		b.cache[s.ID] = d
	}
	return d.pcm, d.ratio, nil
}

type decoded struct {
	pcm   []byte
	ratio float64
}

type voice struct{ p *eaudio.Player }

func (v voice) Playing() bool         { return v.p.IsPlaying() }
func (v voice) SetVolume(vol float64) { v.p.SetVolume(vol) }
func (v voice) Stop()                 { v.p.Close() }
