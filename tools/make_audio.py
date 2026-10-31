"""Composes Keyhole's original sound effects and background music.

Run: python3 tools/make_audio.py   (needs numpy, scipy and ffmpeg)
Writes Ogg Vorbis files into game/assets/sfx and game/assets/music.
"""
import os, random, subprocess, tempfile
import numpy as np
from scipy.signal import butter, lfilter, fftconvolve
from scipy.io import wavfile

SR = 44100
HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "game", "assets")
rng = np.random.default_rng(7)
random.seed(7)


def t_(d):
    return np.arange(int(d * SR)) / SR


def bp(x, lo, hi, order=2):
    b, a = butter(order, [lo / (SR / 2), min(hi / (SR / 2), 0.99)], btype="band")
    return lfilter(b, a, x)


def lp(x, f, order=2):
    b, a = butter(order, f / (SR / 2), btype="low")
    return lfilter(b, a, x)


def hp(x, f, order=2):
    b, a = butter(order, f / (SR / 2), btype="high")
    return lfilter(b, a, x)


def env(n, a, d):
    t = np.arange(n) / SR
    e = np.exp(-t / d)
    k = max(1, int(a * SR))
    e[:k] *= np.linspace(0, 1, k)
    return e


def noise(d):
    return rng.standard_normal(int(d * SR))


def place(buf, x, at):
    i = int(at * SR)
    end = min(len(buf), i + len(x))
    buf[i:end] += x[: end - i]


def ping(f, d, decay, amp=1.0, partials=((1, 1.0), (2.76, 0.4), (5.4, 0.18))):
    t = t_(d)
    s = sum(a * np.sin(2 * np.pi * f * k * t) * np.exp(-t / (decay / (1 + 0.6 * i))) for i, (k, a) in enumerate(partials))
    return s * amp


def tick(f=3200, d=0.05, amp=1.0, body=0.6):
    """A crisp metallic tick: a noise transient plus a short resonant ping."""
    n = noise(d) * env(int(d * SR), 0.0005, 0.004)
    n = bp(n, f * 0.5, f * 2.2)
    return (n * 0.9 + ping(f, d, 0.012, body)) * amp


def reverb(x, size=0.6, mix=0.25):
    ir_len = int(SR * (0.25 + size))
    ir = rng.standard_normal(ir_len) * np.exp(-np.arange(ir_len) / (SR * size * 0.35))
    ir = lp(ir, 5000)
    ir /= np.abs(ir).sum() ** 0.5 * 3
    wet = fftconvolve(x, ir)
    dry = np.concatenate([x, np.zeros(len(wet) - len(x))])
    return dry * (1 - mix) + wet * mix


def norm(x, peak=0.9):
    m = np.abs(x).max()
    return x / m * peak if m > 0 else x


def trim(x, tail=0.02):
    idx = np.where(np.abs(x) > 1e-3 * np.abs(x).max())[0]
    end = min(len(x), idx[-1] + int(tail * SR)) if len(idx) else len(x)
    x = x[:end]
    f = min(len(x), int(0.01 * SR))
    x[-f:] *= np.linspace(1, 0, f)
    return x


def write_ogg(name, x, stereo=False, q=4):
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as f:
        path = f.name
    data = np.clip(x, -1, 1)
    if stereo and data.ndim == 1:
        data = np.stack([data, data], axis=1)
    wavfile.write(path, SR, (data * 32767).astype(np.int16))
    os.makedirs(os.path.dirname(name), exist_ok=True)
    subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-i", path, "-c:a", "libvorbis", "-q:a", str(q), name], check=True)
    os.unlink(path)


# ------------------------------------------------------------------ effects

def sfx():
    out = {}
    out["type"] = norm(tick(2600, 0.045, 1, 0.25), 0.35)
    out["erase"] = norm(tick(1500, 0.05, 1, 0.3), 0.3)
    out["button"] = norm(tick(2100, 0.06, 1, 0.5), 0.4)
    # tumbler clicks into place: two quick clicks and a bright ping
    b = np.zeros(int(0.35 * SR))
    place(b, tick(3600, 0.05, 0.8), 0)
    place(b, tick(4200, 0.08, 1.0, 0.9), 0.028)
    place(b, ping(2900, 0.3, 0.07, 0.25), 0.03)
    out["click"] = norm(reverb(b, 0.25, 0.15), 0.6)
    # wiggle: a small rattle of loose pins
    b = np.zeros(int(0.32 * SR))
    for i, at in enumerate([0, 0.035, 0.062, 0.095, 0.12]):
        place(b, tick(2000 + 400 * (i % 2), 0.04, 0.8 - 0.1 * i, 0.35), at)
    out["wiggle"] = norm(reverb(b, 0.2, 0.12), 0.5)
    # silent: a dull thud
    t = t_(0.18)
    th = np.sin(2 * np.pi * (140 - 60 * t / 0.18) * t) * env(len(t), 0.002, 0.04)
    th += lp(noise(0.18), 900) * env(len(t), 0.001, 0.02) * 0.6
    out["silent"] = norm(th, 0.45)
    # invalid: two low clunks
    b = np.zeros(int(0.3 * SR))
    for at in (0, 0.11):
        t = t_(0.12)
        place(b, np.sin(2 * np.pi * 95 * t) * env(len(t), 0.002, 0.035) + lp(noise(0.12), 600) * env(len(t), 0.001, 0.015), at)
    out["invalid"] = norm(b, 0.5)
    # unlock: ratchet, shackle spring and a bell
    b = np.zeros(int(1.4 * SR))
    for i in range(6):
        place(b, tick(2400 + 150 * i, 0.04, 0.5 + 0.08 * i, 0.3), i * 0.04)
    t = t_(0.25)
    spring = np.sin(2 * np.pi * (300 + 900 * t) * t) * env(len(t), 0.001, 0.05) * 0.35
    place(b, spring, 0.25)
    place(b, tick(1800, 0.1, 1.2, 0.8), 0.26)
    for f, at in ((1318.5, 0.3), (1975.5, 0.36)):
        place(b, ping(f, 1.0, 0.35, 0.4, ((1, 1), (2.0, 0.25), (3.0, 0.1))), at)
    out["unlock"] = norm(reverb(b, 0.6, 0.25), 0.7)
    # vault door: heavy bolts, a deep groan and a shimmering chord
    b = np.zeros(int(3.2 * SR))
    for i, at in enumerate((0, 0.12, 0.24)):
        t = t_(0.25)
        place(b, (np.sin(2 * np.pi * 70 * t) * env(len(t), 0.002, 0.08) + lp(noise(0.25), 1200) * env(len(t), 0.001, 0.03)) * 0.9, at)
    t = t_(1.6)
    groan = lp(noise(1.6), 300) * np.sin(np.pi * t / 1.6) * 0.5 + np.sin(2 * np.pi * (55 + 8 * np.sin(2 * np.pi * 1.3 * t)) * t) * np.sin(np.pi * t / 1.6) * 0.35
    place(b, groan, 0.35)
    for i, f in enumerate((523.25, 659.25, 783.99, 1046.5, 1318.5)):
        place(b, ping(f, 2.2, 0.9, 0.22, ((1, 1), (2, 0.2))) * env(int(2.2 * SR), 0.05, 99), 1.0 + i * 0.06)
    out["vault"] = norm(reverb(b, 1.0, 0.35), 0.8)
    # gold key: a sparkling arpeggio
    b = np.zeros(int(1.6 * SR))
    for i, f in enumerate((1046.5, 1318.5, 1568.0, 2093.0, 2637.0)):
        place(b, ping(f, 0.9, 0.3, 0.35, ((1, 1), (3.01, 0.15))), i * 0.07)
    out["key"] = norm(reverb(b, 0.7, 0.3), 0.6)
    # pin test: a soft bell
    out["pin"] = norm(reverb(ping(1760, 0.6, 0.18, 1, ((1, 1), (2.4, 0.2))), 0.3, 0.2), 0.4)
    # dial ticks during the finale
    out["tick"] = norm(tick(3000, 0.05, 1, 0.6), 0.4)
    # low on guesses: a gentle double knock
    b = np.zeros(int(0.35 * SR))
    for at in (0, 0.14):
        t = t_(0.1)
        place(b, np.sin(2 * np.pi * 220 * t) * env(len(t), 0.002, 0.03), at)
    out["warn"] = norm(b, 0.35)
    # vault sealed: descending clunks and a low hum
    b = np.zeros(int(1.6 * SR))
    for i, f in enumerate((180, 140, 100)):
        t = t_(0.3)
        place(b, np.sin(2 * np.pi * f * t) * env(len(t), 0.003, 0.09) + lp(noise(0.3), 700) * env(len(t), 0.001, 0.02) * 0.5, i * 0.22)
    t = t_(1.0)
    place(b, np.sin(2 * np.pi * 73.4 * t) * env(len(t), 0.05, 0.4) * 0.4, 0.66)
    out["fail"] = norm(reverb(b, 0.6, 0.25), 0.6)
    for k, v in out.items():
        write_ogg(os.path.join(OUT, "sfx", k + ".ogg"), trim(v), q=3)
    print("sfx:", ", ".join(sorted(out)))


# ------------------------------------------------------------------ music

NOTES = {"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11}


def hz(m):
    return 440 * 2 ** ((m - 69) / 12)


def rhodes(m, dur, vel):
    f = hz(m)
    n = int((dur + 1.4) * SR)
    t = np.arange(n) / SR
    mod = np.sin(2 * np.pi * f * t) * 1.4 * np.exp(-t / 0.35)
    s = np.sin(2 * np.pi * f * t + mod) * np.exp(-t / 1.8) + 0.18 * np.sin(2 * np.pi * f * 2 * t) * np.exp(-t / 0.5)
    s *= 1 + 0.07 * np.sin(2 * np.pi * 4.2 * t)
    s *= env(n, 0.004, 99)
    off = int(dur * SR)
    s[off:] *= np.exp(-np.arange(n - off) / SR / 0.3)
    return s * vel * 0.22


def bass(m, dur, vel):
    f = hz(m)
    n = int((dur + 0.4) * SR)
    t = np.arange(n) / SR
    s = np.sin(2 * np.pi * f * t) + 0.35 * np.sin(2 * np.pi * 2 * f * t) * np.exp(-t / 0.15) + 0.12 * np.sin(2 * np.pi * 3 * f * t) * np.exp(-t / 0.08)
    s *= env(n, 0.006, 0.9)
    off = int(dur * SR)
    s[off:] *= np.exp(-np.arange(n - off) / SR / 0.06)
    return lp(s, 900) * vel * 0.5


def brush(d, vel):
    n = noise(d)
    return lp(hp(bp(n, 3000, 9000), 2500), 8000) * env(len(n), 0.01, d * 0.3) * vel * 0.06


def vibes(m, dur, vel):
    return ping(hz(m), dur + 1.5, 1.1, vel * 0.16, ((1, 1), (4.0, 0.12), (10.0, 0.03))) * (1 + 0.15 * np.sin(2 * np.pi * 5.5 * t_(dur + 1.5)))


def chord_tones(name):
    root, kind = name.split(":")
    r = NOTES[root]
    shapes = {
        "m9": [0, 3, 7, 10, 14], "m7": [0, 3, 7, 10], "13": [0, 4, 10, 14, 21], "maj9": [0, 4, 7, 11, 14],
        "7b9": [0, 4, 7, 10, 13], "maj7": [0, 4, 7, 11], "m11": [0, 3, 10, 14, 17], "9": [0, 4, 10, 14],
        "m6": [0, 3, 7, 9], "7#9": [0, 4, 10, 15],
        "sus": [0, 5, 7, 10, 14], "add9": [0, 4, 7, 14], "maj7#11": [0, 4, 7, 11, 18], "6": [0, 4, 7, 9, 14],
    }
    return r, shapes[kind]


def track(name, progression, bpm, bars_per_chord, loops, melody_scale, seed, vibe_mel=True):
    r2 = random.Random(seed)
    beat = 60 / bpm
    bar = beat * 4
    total = len(progression) * bars_per_chord * bar * loops
    L = np.zeros(int((total + 4) * SR))
    R = np.zeros_like(L)
    pos = 0.0
    swing = 0.62
    for lp_i in range(loops):
        for ci, ch in enumerate(progression):
            root, iv = chord_tones(ch)
            for b in range(bars_per_chord):
                t0 = pos + b * bar
                # comp: rhodes voicing on 1 and the "and" of 2
                voicing = [48 + root + i if 48 + root + i < 72 else 36 + root + i for i in iv[1:]]
                for hit, dur, vel in ((0, beat * 1.4, 0.8), (beat * 1 + beat * swing, beat * 1.8, 0.6)):
                    if lp_i == 0 and ci == 0 and b == 0 and hit > 0:
                        continue
                    for k, m in enumerate(sorted(voicing)):
                        x = rhodes(m, dur, vel * (0.9 + 0.2 * r2.random()))
                        place(L, x * (0.6 + 0.1 * k), t0 + hit + k * 0.008)
                        place(R, x * (0.9 - 0.1 * k), t0 + hit + k * 0.008)
                # walking bass
                broot = 36 + root if root <= 7 else 24 + root
                walk = [broot, broot + iv[2] - 12 if iv[2] > 12 else broot + iv[2], broot + 12 if r2.random() < 0.4 else broot + iv[1], broot + 10 if r2.random() < 0.5 else broot + 11]
                for k, m in enumerate(walk):
                    x = bass(m, beat * 0.9, 0.9 if k == 0 else 0.7)
                    place(L, x, t0 + k * beat)
                    place(R, x, t0 + k * beat)
                # brushes with swing
                for k in range(4):
                    place(L, brush(beat * 0.9, 0.7 if k % 2 else 0.45), t0 + k * beat)
                    place(R, brush(beat * 0.4, 0.4), t0 + k * beat + beat * swing)
                # sparse vibraphone melody
                if vibe_mel and lp_i > 0 and r2.random() < 0.75:
                    n_notes = r2.choice([2, 3, 3, 4])
                    starts = sorted(r2.sample([0, 1, 1 + swing, 2, 3, 3 + swing], n_notes))
                    for s0 in starts:
                        deg = r2.choice(melody_scale)
                        m = 72 + (root + deg) % 12 - (12 if (root + deg) % 12 > 7 else 0)
                        x = vibes(m, beat * 0.8, 0.7 + 0.3 * r2.random())
                        place(L, x * 0.8, t0 + s0 * beat)
                        place(R, x * 0.6, t0 + s0 * beat + 0.012)
            pos += bars_per_chord * bar
    mix = np.stack([L, R], axis=1)
    mix[:, 0] = reverb(mix[:, 0], 0.9, 0.22)[: len(L)]
    mix[:, 1] = reverb(mix[:, 1], 0.9, 0.22)[: len(R)]
    end = int((total + 2.5) * SR)
    mix = mix[:end]
    fade = int(2.5 * SR)
    mix[-fade:] *= np.linspace(1, 0, fade)[:, None]
    fin = int(0.05 * SR)
    mix[:fin] *= np.linspace(0, 1, fin)[:, None]
    mix = norm(mix, 0.8)
    write_ogg(os.path.join(OUT, "music", name + ".ogg"), mix, stereo=True, q=3)
    print("music:", name, round(len(mix) / SR), "s")



def finish(name, L, R, total):
    mix = np.stack([L, R], axis=1)
    mix[:, 0] = reverb(mix[:, 0], 1.0, 0.25)[: len(L)]
    mix[:, 1] = reverb(mix[:, 1], 1.0, 0.25)[: len(R)]
    end = int((total + 2.5) * SR)
    mix = mix[:end]
    fade = int(2.5 * SR)
    mix[-fade:] *= np.linspace(1, 0, fade)[:, None]
    fin = int(0.05 * SR)
    mix[:fin] *= np.linspace(0, 1, fin)[:, None]
    mix = norm(mix, 0.8)
    write_ogg(os.path.join(OUT, "music", name + ".ogg"), mix, stereo=True, q=3)
    print("music:", name, round(len(mix) / SR), "s")


def warm_pad(ms, dur):
    """Soft detuned pad with a slow swell, for Zen."""
    n = int((dur + 2.0) * SR)
    t = np.arange(n) / SR
    s = np.zeros(n)
    for m in ms:
        f = hz(m)
        for det in (-0.12, 0.0, 0.11):
            ph = 2 * np.pi * f * (1 + det / 100) * t
            s += np.sin(ph) * 0.6 + np.sin(2 * ph) * 0.15 + np.sin(3 * ph) * 0.05
    a = np.minimum(1, t / 1.6)
    rel = np.ones(n)
    off = int(dur * SR)
    rel[off:] = np.exp(-np.arange(n - off) / SR / 1.2)
    s *= a * rel * (1 + 0.04 * np.sin(2 * np.pi * 0.2 * t))
    return lp(s, 2200) * 0.03


def zen_track(name, progression, bpm, loops, scale, seed):
    """No drums: slow pads, a low bass note per chord and a few soft bells."""
    r2 = random.Random(seed)
    beat = 60 / bpm
    bar = beat * 4
    per = 2 * bar
    total = len(progression) * per * loops
    L = np.zeros(int((total + 5) * SR))
    R = np.zeros_like(L)
    pos = 0.0
    for lp_i in range(loops):
        for ch in progression:
            root, iv = chord_tones(ch)
            voicing = sorted(48 + root + i if 48 + root + i < 70 else 36 + root + i for i in iv)
            pad = warm_pad(voicing, per)
            place(L, pad * 1.0, pos)
            place(R, pad * 0.9, pos + 0.015)
            b = bass(36 + root if root <= 5 else 24 + root, per * 0.9, 0.55)
            place(L, b, pos)
            place(R, b, pos)
            if lp_i > 0 or r2.random() < 0.5:
                for k in range(r2.choice([2, 3, 4])):
                    deg = r2.choice(scale)
                    m = 72 + (root + deg) % 12
                    x = ping(hz(m), 3.0, 1.2, 0.09 + 0.05 * r2.random(), ((1, 1), (2.0, 0.08), (3.0, 0.04)))
                    at = pos + r2.choice([0, 1, 2, 3, 4, 5, 6]) * beat + r2.random() * 0.1
                    place(L, x * (0.5 + 0.5 * r2.random()), at)
                    place(R, x * (0.5 + 0.5 * r2.random()), at + 0.02)
            pos += per
    finish(name, L, R, total)


def bossa_track(name, progression, bpm, loops, scale, seed):
    """Bright and playful: bossa rhythm with rim clicks, shaker and vibes."""
    r2 = random.Random(seed)
    beat = 60 / bpm
    bar = beat * 4
    per = 2 * bar
    total = len(progression) * per * loops
    L = np.zeros(int((total + 4) * SR))
    R = np.zeros_like(L)
    pos = 0.0
    rim_pattern = [0, 0.75, 1.5, 2.5, 3.25, 4.5, 5.25, 6, 7]  # in beats over two bars
    for lp_i in range(loops):
        for ch in progression:
            root, iv = chord_tones(ch)
            voicing = sorted(52 + root + i if 52 + root + i < 76 else 40 + root + i for i in iv[1:])
            for hit in (0, 1.5, 3, 4.5, 6):
                for k, m in enumerate(voicing):
                    x = rhodes(m, beat * 0.9, 0.55 + 0.1 * r2.random())
                    place(L, x * (0.7 - 0.08 * k), pos + hit * beat + k * 0.006)
                    place(R, x * (0.6 + 0.08 * k), pos + hit * beat + k * 0.006)
            broot = 36 + root if root <= 5 else 24 + root
            for hit, m in ((0, broot), (1.5, broot + 7), (2, broot + 7), (4, broot), (5.5, broot + 7), (6, broot + 7)):
                x = bass(m, beat * 1.2, 0.8)
                place(L, x, pos + hit * beat)
                place(R, x, pos + hit * beat)
            for hit in rim_pattern:
                rim = tick(1800, 0.05, 0.25, 0.4)
                place(L, rim * 0.6, pos + hit * beat)
                place(R, rim * 0.4, pos + hit * beat)
            for k in range(16):
                sh = hp(noise(0.06), 6000) * env(int(0.06 * SR), 0.005, 0.02) * (0.05 if k % 2 else 0.03)
                place(L, sh * 0.5, pos + k * beat / 2)
                place(R, sh, pos + k * beat / 2)
            if lp_i > 0 or r2.random() < 0.6:
                t0 = pos
                for s0 in sorted(r2.sample([0, 0.5, 1, 1.5, 2.5, 3, 4, 4.5, 5.5, 6, 6.5], r2.choice([3, 4, 5]))):
                    m = 74 + (root + r2.choice(scale)) % 12
                    x = vibes(m, beat * 0.6, 0.8)
                    place(L, x * 0.7, t0 + s0 * beat)
                    place(R, x * 0.8, t0 + s0 * beat + 0.01)
            pos += per
    finish(name, L, R, total)


if __name__ == "__main__":
    sfx()
    track("after-hours", ["D:m9", "G:13", "C:maj9", "A:7b9"], 84, 2, 3, [0, 2, 3, 7, 10], 1)
    track("velvet-tumblers", ["F:maj7", "E:m7", "A:7#9", "D:m9", "Bb:maj7", "A:7b9"], 76, 2, 2, [0, 4, 7, 9, 11], 2)
    # menu: a slow, welcoming lounge tune
    track("lobby-lights", ["Bb:maj9", "G:m9", "C:m9", "F:13", "Eb:maj7#11", "D:7#9", "G:m9", "C:9"], 68, 1, 3, [0, 2, 4, 7, 9], 3)
    # zen: no drums, slow pads and bells
    zen_track("still-water", ["Eb:maj9", "Ab:maj7#11", "C:m9", "Bb:sus", "Ab:maj9", "G:m7", "F:m9", "Bb:sus"], 60, 2, [0, 2, 4, 7, 9], 4)
    # themes: bright, playful bossa
    bossa_track("open-house", ["C:maj9", "A:m9", "D:m9", "G:13", "E:m7", "A:7#9", "D:m9", "G:13"], 112, 2, [0, 2, 4, 7, 9], 5)
