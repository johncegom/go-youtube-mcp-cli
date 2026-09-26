"""Joins j.json (yt-dlp -J metadata, from jcheck.py) with the fetch outcomes it
carries (from manifest.json via measure.py) and tests two hypotheses:

  H1: a `tlang=` parameter in the auto-caption "en" URL <=> plain `--sub-langs en` returns 429.
  H2: the rule "uploaded English track exists -> plain en; else en-orig if listed;
      else plain en" would have chosen a request that returned a transcript.

Usage: python analyze.py   (reads j.json next to this file)
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
rows = [r for r in json.load(open(os.path.join(HERE, "j.json"), encoding="utf-8")) if "error" not in r]

print(f"{'id':12} {'origin':7} {'manual_en':9} {'auto_en':7} {'en-orig':7} {'tlang':5} {'fetch_en':9} {'fetch_orig':10}")
for r in rows:
    u = r["en_url"]
    tl = "-" if u is None else ("yes" if u.get("tlang") else "no")
    print(f"{r['id']:12} {r['origin']:7} {str(bool(r['manual_en_keys'])):9} {str(r['auto_has_en']):7} "
          f"{str('en-orig' in r['auto_orig_keys']):7} {tl:5} {r['fetch_en'][:9]:9} {r['fetch_orig'][:10]:10}")

h1 = [(r["id"], bool((r["en_url"] or {}).get("tlang")), r["fetch_en"]) for r in rows]
print("\nH1 tlang vs 429 mismatches:", [(i, t, f) for i, t, f in h1 if t != (f == "429")])


def choose(r):
    if r["manual_en_keys"]:
        return "en"
    if "en-orig" in r["auto_orig_keys"]:
        return "en-orig"
    return "en"


print("\nH2 rule outcomes (choice -> fetch status of that choice):")
bad = []
for r in rows:
    c = choose(r)
    st = r["fetch_en"] if c == "en" else r["fetch_orig"]
    print(f"  {r['id']:12} choose={c:8} -> {st}")
    if st != "ok":
        bad.append((r["id"], c, st, r["fetch_en"], r["fetch_orig"]))
print("\nchoices that did NOT yield a transcript:", bad)
