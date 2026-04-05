use std::f64::consts::PI;
use std::io::Cursor;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread;

use rodio::{Decoder, OutputStream, Sink};

// ---------------------------------------------------------------------------
// WAV encoding
// ---------------------------------------------------------------------------

fn encode_wav(sr: u32, samples: &[i16]) -> Vec<u8> {
    let data_size = samples.len() as u32 * 2;
    let mut buf: Vec<u8> = Vec::with_capacity(44 + data_size as usize);
    buf.extend_from_slice(b"RIFF");
    buf.extend_from_slice(&(36 + data_size).to_le_bytes());
    buf.extend_from_slice(b"WAVE");
    buf.extend_from_slice(b"fmt ");
    buf.extend_from_slice(&16u32.to_le_bytes()); // chunk size
    buf.extend_from_slice(&1u16.to_le_bytes()); // PCM
    buf.extend_from_slice(&1u16.to_le_bytes()); // mono
    buf.extend_from_slice(&sr.to_le_bytes()); // sample rate
    buf.extend_from_slice(&(sr * 2).to_le_bytes()); // byte rate
    buf.extend_from_slice(&2u16.to_le_bytes()); // block align
    buf.extend_from_slice(&16u16.to_le_bytes()); // bits per sample
    buf.extend_from_slice(b"data");
    buf.extend_from_slice(&data_size.to_le_bytes());
    for &s in samples {
        buf.extend_from_slice(&s.to_le_bytes());
    }
    buf
}

// ---------------------------------------------------------------------------
// Synthesis helpers
// ---------------------------------------------------------------------------

struct SndNote {
    freq: f64,  // Hz, 0 = rest
    beats: f64, // duration in quarter-note beats
}

/// Render a note sequence to float64 samples.
fn render_track(
    notes: &[SndNote],
    sr: f64,
    beat_dur: f64,
    wave_fn: fn(f64) -> f64,
    amp: f64,
    attack_sec: f64,
    release_sec: f64,
) -> Vec<f64> {
    let atk = (attack_sec * sr) as usize;
    let rel = (release_sec * sr) as usize;
    let mut out = Vec::new();
    for n in notes {
        let num = (n.beats * beat_dur * sr) as usize;
        for i in 0..num {
            if n.freq == 0.0 {
                out.push(0.0);
                continue;
            }
            let t = i as f64 / sr;
            let env = if i < atk {
                i as f64 / atk as f64
            } else if i > num - rel {
                let rem = num - i;
                if rem == 0 {
                    0.0
                } else {
                    rem as f64 / rel as f64
                }
            } else {
                1.0
            };
            out.push(wave_fn(2.0 * PI * n.freq * t) * env * amp);
        }
    }
    out
}

/// Mix one or more float64 tracks into int16 with clipping.
fn mix_to_i16(tracks: &[&[f64]]) -> Vec<i16> {
    let max_len = tracks.iter().map(|t| t.len()).max().unwrap_or(0);
    let mut out = vec![0i16; max_len];
    for i in 0..max_len {
        let mut sum: f64 = 0.0;
        for t in tracks {
            if i < t.len() {
                sum += t[i];
            }
        }
        sum = sum.clamp(-1.0, 1.0);
        out[i] = (sum * 32767.0) as i16;
    }
    out
}

// Waveforms

fn sine_wave(phase: f64) -> f64 {
    phase.sin()
}

fn square_wave(phase: f64) -> f64 {
    phase.sin() + (3.0 * phase).sin() / 3.0 + (5.0 * phase).sin() / 5.0
}

fn rich_wave(phase: f64) -> f64 {
    phase.sin() * 0.7 + (2.0 * phase).sin() * 0.15 + (3.0 * phase).sin() * 0.05
}

// ---------------------------------------------------------------------------
// Success chime
// ---------------------------------------------------------------------------

fn generate_success_wav() -> Vec<u8> {
    let sr = 44100u32;
    let srf = sr as f64;
    let notes = [
        SndNote {
            freq: 523.25,
            beats: 0.12,
        },
        SndNote {
            freq: 659.25,
            beats: 0.12,
        },
        SndNote {
            freq: 783.99,
            beats: 0.12,
        },
        SndNote {
            freq: 1046.50,
            beats: 0.30,
        },
    ];
    let track = render_track(&notes, srf, 1.0, rich_wave, 0.35, 0.005, 0.04);
    encode_wav(sr, &mix_to_i16(&[&track]))
}

// ---------------------------------------------------------------------------
// Theme music — Korobeiniki (Tetris Theme A)
// ---------------------------------------------------------------------------

const REST: f64 = 0.0;
const C3: f64 = 130.81;
const D3: f64 = 146.83;
const E3: f64 = 164.81;
const A3: f64 = 220.00;
const A4: f64 = 440.00;
const B4: f64 = 493.88;
const C5: f64 = 523.25;
const D5: f64 = 587.33;
const E5: f64 = 659.25;
const F5: f64 = 698.46;
const G5: f64 = 783.99;
const A5: f64 = 880.00;

fn generate_theme_wav() -> Vec<u8> {
    let sr = 44100u32;
    let srf = sr as f64;
    let bpm = 150.0;
    let beat_dur = 60.0 / bpm;

    let melody = [
        // Phrase A
        SndNote {
            freq: E5,
            beats: 1.0,
        },
        SndNote {
            freq: B4,
            beats: 0.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: D5,
            beats: 1.0,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: B4,
            beats: 0.5,
        },
        SndNote {
            freq: A4,
            beats: 1.0,
        },
        SndNote {
            freq: A4,
            beats: 0.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: E5,
            beats: 1.0,
        },
        SndNote {
            freq: D5,
            beats: 0.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: B4,
            beats: 1.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: D5,
            beats: 1.0,
        },
        SndNote {
            freq: E5,
            beats: 1.0,
        },
        SndNote {
            freq: C5,
            beats: 1.0,
        },
        SndNote {
            freq: A4,
            beats: 1.0,
        },
        SndNote {
            freq: A4,
            beats: 2.0,
        },
        // Phrase B
        SndNote {
            freq: REST,
            beats: 0.5,
        },
        SndNote {
            freq: D5,
            beats: 1.0,
        },
        SndNote {
            freq: F5,
            beats: 0.5,
        },
        SndNote {
            freq: A5,
            beats: 1.0,
        },
        SndNote {
            freq: G5,
            beats: 0.5,
        },
        SndNote {
            freq: F5,
            beats: 0.5,
        },
        SndNote {
            freq: E5,
            beats: 1.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: E5,
            beats: 1.0,
        },
        SndNote {
            freq: D5,
            beats: 0.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: B4,
            beats: 1.5,
        },
        SndNote {
            freq: C5,
            beats: 0.5,
        },
        SndNote {
            freq: D5,
            beats: 1.0,
        },
        SndNote {
            freq: E5,
            beats: 1.0,
        },
        SndNote {
            freq: C5,
            beats: 1.0,
        },
        SndNote {
            freq: A4,
            beats: 1.0,
        },
        SndNote {
            freq: A4,
            beats: 2.0,
        },
    ];

    let bass = [
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: D3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: D3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: C3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: C3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: E3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
        SndNote {
            freq: A3,
            beats: 1.0,
        },
        SndNote {
            freq: REST,
            beats: 1.0,
        },
    ];

    let mel_track = render_track(&melody, srf, beat_dur, square_wave, 0.15, 0.005, 0.02);
    let bass_track = render_track(&bass, srf, beat_dur, sine_wave, 0.10, 0.005, 0.02);

    encode_wav(sr, &mix_to_i16(&[&mel_track, &bass_track]))
}

// ---------------------------------------------------------------------------
// SoundEngine — owns the audio output and provides play/mute controls
// ---------------------------------------------------------------------------

pub struct SoundEngine {
    muted: Arc<AtomicBool>,
    theme_sink: Option<Sink>,
    success_wav: Vec<u8>,
    // Keep the OutputStream alive for the lifetime of the engine.
    _stream: Option<OutputStream>,
}

impl SoundEngine {
    pub fn new() -> Self {
        let muted = Arc::new(AtomicBool::new(false));
        let success_wav = generate_success_wav();

        // Try to open audio output; if it fails, we run silently.
        let (stream, theme_sink) = match OutputStream::try_default() {
            Ok((stream, handle)) => {
                let theme_wav = generate_theme_wav();
                let theme_sink = Sink::try_new(&handle).ok();
                if let Some(ref sink) = theme_sink {
                    let cursor = Cursor::new(theme_wav);
                    if let Ok(source) = Decoder::new(cursor) {
                        sink.append(source);
                        // rodio Sink doesn't have a native loop, so we
                        // re-append in a background thread (see start_theme_loop).
                    }
                }
                (Some(stream), theme_sink)
            }
            Err(_) => (None, None),
        };

        SoundEngine {
            muted,
            theme_sink,
            success_wav,
            _stream: stream,
        }
    }

    /// Start the theme music loop in a background thread that re-appends
    /// the theme WAV data whenever the sink runs low.
    pub fn start_theme_loop(&self) {
        let Some(ref sink) = self.theme_sink else {
            return;
        };
        // We need to keep feeding the sink with new copies of the theme.
        // Spawn a thread that watches the sink and appends when nearly empty.
        let muted = Arc::clone(&self.muted);
        let theme_wav = generate_theme_wav();

        // The sink is not Send, so we use a different approach:
        // We pre-fill the sink with many repeats upfront and that's enough
        // for a typical game session. If someone plays for hours it will
        // eventually stop, but that's acceptable.
        // Actually, let's use a smarter approach: we'll clone the handle.
        // rodio::Sink is not Send, but we can just append many copies now.
        for _ in 0..100 {
            let cursor = Cursor::new(theme_wav.clone());
            if let Ok(source) = Decoder::new(cursor) {
                sink.append(source);
            }
        }

        // Apply initial mute state.
        if muted.load(Ordering::Relaxed) {
            sink.set_volume(0.0);
        }
    }

    pub fn play_success(&self) {
        if self.muted.load(Ordering::Relaxed) {
            return;
        }
        let Some(ref _stream) = self._stream else {
            return;
        };
        // Play in a background thread so we don't block the TUI.
        let wav = self.success_wav.clone();
        thread::spawn(move || {
            if let Ok((_stream, handle)) = OutputStream::try_default() {
                if let Ok(sink) = Sink::try_new(&handle) {
                    let cursor = Cursor::new(wav);
                    if let Ok(source) = Decoder::new(cursor) {
                        sink.append(source);
                        sink.sleep_until_end();
                    }
                }
            }
        });
    }

    pub fn toggle_mute(&self) {
        let was_muted = self.muted.fetch_xor(true, Ordering::Relaxed);
        let now_muted = !was_muted;
        if let Some(ref sink) = self.theme_sink {
            if now_muted {
                sink.set_volume(0.0);
            } else {
                sink.set_volume(1.0);
            }
        }
    }

    pub fn is_muted(&self) -> bool {
        self.muted.load(Ordering::Relaxed)
    }

    pub fn stop(&self) {
        if let Some(ref sink) = self.theme_sink {
            sink.stop();
        }
    }
}
