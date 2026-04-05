// Command gensound generates WAV sound files for the puzzle game.
// Usage: go run ./cmd/gensound
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

func main() {
	os.MkdirAll("sounds", 0o755)
	generateSuccessWAV("sounds/success.wav")
	fmt.Println("generated sounds/success.wav")
	generateThemeWAV("sounds/theme.wav")
	fmt.Println("generated sounds/theme.wav")
}

// ---------------------------------------------------------------------------
// Synthesis helpers
// ---------------------------------------------------------------------------

type note struct {
	freq  float64 // Hz, 0 = rest
	beats float64 // duration in quarter-note beats
}

func renderTrack(notes []note, sr, beatDur float64, waveFn func(float64) float64, amp, attackSec, releaseSec float64) []float64 {
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

func writeWAV(path string, sampleRate int, samples []int16) {
	dataSize := len(samples) * 2

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))           // chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))            // PCM
	binary.Write(f, binary.LittleEndian, uint16(1))            // mono
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))   // sample rate
	binary.Write(f, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(f, binary.LittleEndian, uint16(2))            // block align
	binary.Write(f, binary.LittleEndian, uint16(16))           // bits per sample
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))
	for _, s := range samples {
		binary.Write(f, binary.LittleEndian, s)
	}
}

// ---------------------------------------------------------------------------
// Success chime: C5 → E5 → G5 → C6
// ---------------------------------------------------------------------------

func generateSuccessWAV(path string) {
	sr := 44100
	sampleRate := float64(sr)
	notes := []note{
		{523.25, 0.12}, {659.25, 0.12}, {783.99, 0.12}, {1046.50, 0.30},
	}
	richWave := func(phase float64) float64 {
		return math.Sin(phase)*0.7 + math.Sin(2*phase)*0.15 + math.Sin(3*phase)*0.05
	}
	track := renderTrack(notes, sampleRate, 1.0, richWave, 0.35, 0.005, 0.04)
	writeWAV(path, sr, mixToInt16(track))
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

func generateThemeWAV(path string) {
	sr := 44100
	sampleRate := float64(sr)
	bpm := 150.0
	beatDur := 60.0 / bpm

	// --- Melody (chiptune square-ish wave) ---
	melody := []note{
		// Phrase A
		{E5, 1},
		{B4, 0.5},
		{C5, 0.5},
		{D5, 1},
		{C5, 0.5},
		{B4, 0.5},
		{A4, 1},
		{A4, 0.5},
		{C5, 0.5},
		{E5, 1},
		{D5, 0.5},
		{C5, 0.5},
		{B4, 1.5},
		{C5, 0.5},
		{D5, 1},
		{E5, 1},
		{C5, 1},
		{A4, 1},
		{A4, 2},
		// Phrase B
		{rest, 0.5},
		{D5, 1},
		{F5, 0.5},
		{A5, 1},
		{G5, 0.5},
		{F5, 0.5},
		{E5, 1.5},
		{C5, 0.5},
		{E5, 1},
		{D5, 0.5},
		{C5, 0.5},
		{B4, 1.5},
		{C5, 0.5},
		{D5, 1},
		{E5, 1},
		{C5, 1},
		{A4, 1},
		{A4, 2},
	}

	// --- Bass (pulsing root on beats 1 & 3) ---
	bass := []note{
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 1
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 2
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 3
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 4
		{D3, 1}, {rest, 1}, {D3, 1}, {rest, 1}, // bar 5
		{C3, 1}, {rest, 1}, {C3, 1}, {rest, 1}, // bar 6
		{E3, 1}, {rest, 1}, {E3, 1}, {rest, 1}, // bar 7
		{A3, 1}, {rest, 1}, {A3, 1}, {rest, 1}, // bar 8
	}

	sqWave := func(phase float64) float64 {
		return math.Sin(phase) + math.Sin(3*phase)/3 + math.Sin(5*phase)/5
	}

	melTrack := renderTrack(melody, sampleRate, beatDur, sqWave, 0.15, 0.005, 0.02)
	bassTrack := renderTrack(bass, sampleRate, beatDur, math.Sin, 0.10, 0.005, 0.02)

	writeWAV(path, sr, mixToInt16(melTrack, bassTrack))
}
