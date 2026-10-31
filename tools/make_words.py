"""Builds assets/words/answers-N.txt and assets/words/valid.txt for Keyhole.

Answers: common (ESDB/SCOWL size 35) American base-form words, no slang,
no vulgar or offensive words, nothing on the blocklist.
Valid guesses: every lowercase 4-6 letter word in ESDB up to size 70 plus the
Wordnik word list, so players aren't told a real word isn't a word.

Usage: python3 tools/make_words.py <path to en-wl/wordlist with scowl.db> <path to wordnik wordlist.txt>
"""
import os, re, sqlite3, sys, random

esdb, wordnik = sys.argv[1], sys.argv[2]
here = os.path.dirname(os.path.abspath(__file__))
out = os.path.join(here, "..", "rules", "words")
os.makedirs(out, exist_ok=True)
block = set(open(os.path.join(here, "blocklist.txt")).read().split())

c = sqlite3.connect(os.path.join(esdb, "scowl.db"))
rows = c.execute("""select word, size, pos, usage_note, category from scowl_v0
                    where spelling in ('A','_') and region in ('','US') and variant_level <= 1
                    and length(word) between 4 and 6""").fetchall()
BASE = {"n0", "v0", "aj0", "av0", "m0", "a0", "pp", "i", "c", "d", "pn0", "vn", "pnp"}
bad, base, valid = set(), set(), set()
for w, size, pos, note, cat in rows:
    if not re.fullmatch(r"[a-z]+", w):
        continue
    if note.startswith(("vulgar", "offensive")):
        bad.add(w)
    if size <= 70:
        valid.add(w)
    if size <= 35 and note == "" and cat == "" and pos in BASE:
        base.add(w)
for line in open(wordnik):
    w = line.strip().strip('"')
    if re.fullmatch(r"[a-z]{4,6}", w):
        valid.add(w)
answers = sorted(base - bad - block)
valid |= set(answers)
for n in (4, 5, 6):
    ws = [w for w in answers if len(w) == n]
    open(os.path.join(out, f"answers-{n}.txt"), "w").write("\n".join(ws) + "\n")
    print(n, len(ws))
open(os.path.join(out, "valid.txt"), "w").write("\n".join(sorted(valid)) + "\n")
print("valid", len(valid))
