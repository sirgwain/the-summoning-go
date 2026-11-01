package summoning

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirgwain/the-summoning-go/engine/content"
	"github.com/sirgwain/the-summoning-go/internal/opl2"
)

// SampleRate is the rate songs are rendered at.
const SampleRate = 44100

// firstEffect is the track of sound effect 0 (audio.md, Songs).
const firstEffect = 0x11

// loopBars are the tracks the game loops, and for how many bars (audio.md, Songs).
var loopBars = map[int]int{1: 48, 5: 24, 6: 16, 7: 8, 8: 12, 15: 8}

// writeTime is how long the game's driver takes over each register write (audio.md,
// Register writes).
const writeTime = 80 * time.Microsecond

const (
	maxSongSeconds = 600
	maxTailSeconds = 5
	tailBlock      = opl2.Rate / 100 // 10 ms
	// A tail block whose samples span no more than this has died away: silent operators
	// output 0 or -1, so silence is not quite flat.
	tailQuiet = 16
)

// Rendered is a song rendered to mono samples at SampleRate. A looping song plays Intro
// samples, then repeats the Loop samples after them; a song that plays once has Loop 0.
type Rendered struct {
	Samples     []int16
	Intro, Loop int
}

// RenderSong plays a song through an OPL2 as the game does. With bars above 0 the song loops:
// the result holds its first pass as the intro and its second as the loop, which starts with
// the first pass's notes still ringing. Otherwise it holds the song and the release after it.
func RenderSong(s *Song, bars int) (*Rendered, error) {
	chip := opl2.New()
	chip.SetWriteTime(writeTime)
	p := newSongPlayer()
	for _, e := range p.timeline {
		chip.Write(int(e.Reg), e.Val)
	}
	for chip.Pending() > 0 {
		chip.Sample()
	}
	p.timeline = p.timeline[:0]
	p.start(s, bars)

	var out []int16
	var cycles, tick int64
	divisor := int64(1 << 16)
	fed := 0
	feed := func() error {
		for ; fed < len(p.timeline); fed++ {
			e := p.timeline[fed]
			if e.Tick > tick {
				cycles += (e.Tick - tick) * divisor
				tick = e.Tick
			}
			if cycles > maxSongSeconds*PITRate {
				return fmt.Errorf("song runs past %d seconds", maxSongSeconds)
			}
			for target := cycles * opl2.Rate / PITRate; int64(len(out)) < target; {
				out = append(out, chip.Sample())
			}
			if e.Divisor != 0 {
				divisor = int64(e.Divisor)
			} else {
				chip.Write(int(e.Reg), e.Val)
			}
		}
		return nil
	}
	// at is the chip sample of the player's current tick.
	at := func() int64 { return (cycles + (p.tick-tick)*divisor) * opl2.Rate / PITRate }

	var restarts [2]int64
	for {
		if err := feed(); err != nil {
			return nil, err
		}
		if !p.playing {
			break
		}
		before := p.restarts
		p.advance()
		if bars > 0 && p.restarts > before {
			restarts[p.restarts-1] = at()
			if p.restarts == 2 {
				break
			}
		}
		if p.tick > 0 && at() > maxSongSeconds*opl2.Rate {
			return nil, fmt.Errorf("song runs past %d seconds", maxSongSeconds)
		}
	}
	if p.err != nil {
		return nil, p.err
	}

	r := &Rendered{}
	if bars > 0 {
		for int64(len(out)) < restarts[1] {
			out = append(out, chip.Sample())
		}
		out = out[:restarts[1]]
		r.Intro = int(restarts[0] * SampleRate / opl2.Rate)
		r.Loop = int(restarts[1]*SampleRate/opl2.Rate) - r.Intro
	} else {
		for i := 0; i < maxTailSeconds*100 && !chip.Silent(); i++ {
			lo, hi := int16(32767), int16(-32768)
			for k := 0; k < tailBlock; k++ {
				v := chip.Sample()
				out = append(out, v)
				lo, hi = min(lo, v), max(hi, v)
			}
			if int(hi)-int(lo) <= tailQuiet {
				break
			}
		}
	}
	r.Samples = opl2.Resample(out, SampleRate)
	if bars > 0 {
		r.Samples = r.Samples[:min(len(r.Samples), r.Intro+r.Loop)]
		r.Loop = len(r.Samples) - r.Intro
	}
	return r, nil
}

// soundID is the id of track n: a music track, or a sound effect from track 11h on.
func soundID(n int) string {
	if n >= firstEffect {
		return fmt.Sprintf("%s%d", content.SFXPrefix, n-firstEffect)
	}
	return fmt.Sprintf("%s%d", content.MusicPrefix, n)
}

// audio renders every track, from the music base to the first resource that is not a song.
func (im *importer) audio() {
	type job struct {
		track int
		song  *Song
		out   *Rendered
		err   error
	}
	var jobs []*job
	base := int(im.init.Bases.Music)
	for n := 0; base+n < len(im.resources); n++ {
		r, err := ParseResource(im.resources[base+n])
		if err != nil || r.Type != ResourceMusic {
			break
		}
		s, err := DecodeSong(r)
		if err != nil {
			im.report.addf("%s: %v", soundID(n), err)
			continue
		}
		jobs = append(jobs, &job{track: n, song: s})
	}
	if len(jobs) == 0 {
		im.report.addf("audio: no songs at resource %d", base)
		return
	}

	work := make(chan *job)
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(jobs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range work {
				j.out, j.err = RenderSong(j.song, loopBars[j.track])
			}
		}()
	}
	for _, j := range jobs {
		work <- j
	}
	close(work)
	wg.Wait()

	for _, j := range jobs {
		id := soundID(j.track)
		if j.err != nil {
			im.report.addf("%s: %v", id, j.err)
			continue
		}
		_, name, _ := cutPrefix(id)
		im.pack.Sounds[id] = &content.Sound{
			ID: id, File: name + ".wav", Intro: j.out.Intro, Loop: j.out.Loop,
			Data: content.EncodeWAV(j.out.Samples, SampleRate),
		}
	}
}

// cutPrefix splits a sound id into its kind and name.
func cutPrefix(id string) (kind, name string, ok bool) {
	for _, p := range []string{content.MusicPrefix, content.SFXPrefix} {
		if len(id) > len(p) && id[:len(p)] == p {
			return p, id[len(p):], true
		}
	}
	return "", "", false
}
