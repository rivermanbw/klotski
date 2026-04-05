package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

var (
	speakerInitOnce sync.Once
	speakerReady    bool
	muted           bool
	mutedMu         sync.Mutex
)

func toggleMute() {
	mutedMu.Lock()
	muted = !muted
	nowMuted := muted
	mutedMu.Unlock()

	themeMu.Lock()
	ctrl := themeCtrl
	themeMu.Unlock()

	if ctrl != nil {
		speaker.Lock()
		ctrl.Paused = nowMuted
		speaker.Unlock()
	}
}

func isMuted() bool {
	mutedMu.Lock()
	defer mutedMu.Unlock()
	return muted
}

func initSpeaker(sr beep.SampleRate) {
	speakerInitOnce.Do(func() {
		if err := speaker.Init(sr, sr.N(time.Second/10)); err == nil {
			speakerReady = true
		}
	})
}

// ---------------------------------------------------------------------------
// WAV encoding
// ---------------------------------------------------------------------------

func encodeWAV(sr int, samples []int16) []byte {
	buf := new(bytes.Buffer)
	dataSize := len(samples) * 2
	buf.Write([]byte("RIFF"))
	binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.Write([]byte("WAVE"))
	buf.Write([]byte("fmt "))
	binary.Write(buf, binary.LittleEndian, uint32(16))   // chunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))    // PCM
	binary.Write(buf, binary.LittleEndian, uint16(1))    // mono
	binary.Write(buf, binary.LittleEndian, uint32(sr))   // sample rate
	binary.Write(buf, binary.LittleEndian, uint32(sr*2)) // byte rate
	binary.Write(buf, binary.LittleEndian, uint16(2))    // block align
	binary.Write(buf, binary.LittleEndian, uint16(16))   // bits per sample
	buf.Write([]byte("data"))
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	for _, s := range samples {
		binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Synthesis helpers
// ---------------------------------------------------------------------------

type sndNote struct {
	freq  float64 // Hz, 0 = rest
	beats float64 // duration in quarter-note beats
}

// renderTrack synthesises a note sequence into float64 samples.
// waveFn receives the instantaneous phase and returns a value in [-1, 1].
func renderTrack(notes []sndNote, sr, beatDur float64, waveFn func(float64) float64, amp, attackSec, releaseSec float64) []float64 {
	var out []float64
	atk := int(attackSec * sr)
	rel := int(releaseSec * sr)
	for _, n := range notes {
		num := int(n.beats * beatDur * sr)
		for i := range num {
			if n.freq == 0 {
				out = append(out, 0)
				continue
			}
			t := float64(i) / sr
			env := 1.0
			if i < atk {
				env = float64(i) / float64(atk)
			} else if i > num-rel {
				rem := num - i
				if rem <= 0 {
					env = 0
				} else {
					env = float64(rem) / float64(rel)
				}
			}
			out = append(out, waveFn(2*math.Pi*n.freq*t)*env*amp)
		}
	}
	return out
}

// mixToInt16 mixes one or more float64 sample tracks with clipping.
func mixToInt16(tracks ...[]float64) []int16 {
	maxLen := 0
	for _, t := range tracks {
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}
	out := make([]int16, maxLen)
	for i := range out {
		var sum float64
		for _, t := range tracks {
			if i < len(t) {
				sum += t[i]
			}
		}
		if sum > 1 {
			sum = 1
		}
		if sum < -1 {
			sum = -1
		}
		out[i] = int16(sum * 32767)
	}
	return out
}

// ---------------------------------------------------------------------------
// Success chime
// ---------------------------------------------------------------------------

func generateSuccessWAV() []byte {
	sr := 44100
	sampleRate := float64(sr)
	notes := []sndNote{
		{523.25, 0.12}, {659.25, 0.12}, {783.99, 0.12}, {1046.50, 0.30},
	}
	richWave := func(phase float64) float64 {
		return math.Sin(phase)*0.7 + math.Sin(2*phase)*0.15 + math.Sin(3*phase)*0.05
	}
	// beats == seconds here (beatDur = 1.0)
	track := renderTrack(notes, sampleRate, 1.0, richWave, 0.35, 0.005, 0.04)
	return encodeWAV(sr, mixToInt16(track))
}

var successWAV = generateSuccessWAV()

func playSuccessSound() {
	if isMuted() {
		return
	}
	go func() {
		reader := bytes.NewReader(successWAV)
		streamer, format, err := wav.Decode(io.NopCloser(reader))
		if err != nil {
			return
		}
		defer streamer.Close()

		initSpeaker(format.SampleRate)
		if !speakerReady {
			return
		}

		done := make(chan struct{})
		speaker.Play(beep.Seq(streamer, beep.Callback(func() {
			close(done)
		})))
		<-done
	}()
}

// ---------------------------------------------------------------------------
// Theme music — Korobeiniki (Tetris Theme A)
// ---------------------------------------------------------------------------

// Note frequencies.
const (
	rest = 0
	C3   = 130.81
	D3   = 146.83
	E3   = 164.81
	A3   = 220.00
	B3   = 246.94
	C4   = 261.63
	D4   = 293.66
	E4   = 329.63
	A4   = 440.00
	B4   = 493.88
	C5   = 523.25
	D5   = 587.33
	E5   = 659.25
	F5   = 698.46
	G5   = 783.99
	A5   = 880.00
)

func generateThemeWAV() []byte {
	sr := 44100
	sampleRate := float64(sr)
	bpm := 150.0
	beatDur := 60.0 / bpm

	// --- Melody (chiptune square-ish wave) ---
	melody := []sndNote{
		// Phrase A
		// Bar 1
		{E5, 1},
		{B4, 0.5},
		{C5, 0.5},
		{D5, 1},
		{C5, 0.5},
		{B4, 0.5},
		// Bar 2
		{A4, 1},
		{A4, 0.5},
		{C5, 0.5},
		{E5, 1},
		{D5, 0.5},
		{C5, 0.5},
		// Bar 3
		{B4, 1.5},
		{C5, 0.5},
		{D5, 1},
		{E5, 1},
		// Bar 4
		{C5, 1},
		{A4, 1},
		{A4, 2},

		// Phrase B
		// Bar 5
		{rest, 0.5},
		{D5, 1},
		{F5, 0.5},
		{A5, 1},
		{G5, 0.5},
		{F5, 0.5},
		// Bar 6
		{E5, 1.5},
		{C5, 0.5},
		{E5, 1},
		{D5, 0.5},
		{C5, 0.5},
		// Bar 7
		{B4, 1.5},
		{C5, 0.5},
		{D5, 1},
		{E5, 1},
		// Bar 8
		{C5, 1},
		{A4, 1},
		{A4, 2},
	}

	// --- Bass (pulsing root on beats 1 & 3) ---
	bass := []sndNote{
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 1
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 2
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 3
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 4
		{D3, 1}, {rest, 1}, {D3, 1}, {rest, 1}, // bar 5
		{C3, 1}, {rest, 1}, {C3, 1}, {rest, 1}, // bar 6
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 7
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 8
	}

	// Band-limited square wave (odd harmonics).
	sqWave := func(phase float64) float64 {
		return math.Sin(phase) + math.Sin(3*phase)/3 + math.Sin(5*phase)/5
	}

	melTrack := renderTrack(melody, sampleRate, beatDur, sqWave, 0.15, 0.005, 0.02)
	bassTrack := renderTrack(bass, sampleRate, beatDur, math.Sin, 0.10, 0.005, 0.02)

	return encodeWAV(sr, mixToInt16(melTrack, bassTrack))
}

// --- Theme playback (looping) ---

var (
	themeWAV  = generateThemeWAV()
	themeCtrl *beep.Ctrl
	themeMu   sync.Mutex
)

func playThemeMusic() {
	go func() {
		reader := bytes.NewReader(themeWAV)
		streamer, format, err := wav.Decode(io.NopCloser(reader))
		if err != nil {
			return
		}

		initSpeaker(format.SampleRate)
		if !speakerReady {
			return
		}

		loop := beep.Loop(-1, streamer)
		ctrl := &beep.Ctrl{Streamer: loop}

		themeMu.Lock()
		themeCtrl = ctrl
		themeMu.Unlock()

		speaker.Play(ctrl)
	}()
}

func stopThemeMusic() {
	themeMu.Lock()
	defer themeMu.Unlock()
	if themeCtrl != nil {
		speaker.Lock()
		themeCtrl.Paused = true
		speaker.Unlock()
		themeCtrl = nil
	}
}
