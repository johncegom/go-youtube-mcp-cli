"""Offline analysis of research.json (written by research.py): how often today's
resolveDefaultLanguage rule disagrees with yt-dlp's `language`, and how the page's
captions.audioTracks ids relate to it.  Usage: python analyze_research.py
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
rows = json.load(open(os.path.join(HERE, "research.json"), encoding="utf-8"))
base = lambda l: (l or "").split("-")[0]

print(f"videos: {len(rows)}   page errors: {[r['id'] for r in rows if 'page_error' in r]}   yt-dlp errors: {[r['id'] for r in rows if 'ytdlp_error' in r]}")
print(f"\n{'id':12} {'intended':8} {'.4 audio lang':13} {'yt-dlp':7} {'today rule':10} {'#trk':4} verdict")
rule_ok = rule_bad = struct = agree = 0
wrong = []
for r in rows:
    ids = r.get("audio_ids") or []
    dot4 = [i.rsplit(".", 1)[0] for i in ids if i.endswith(".4")]
    a, y, cr = (dot4[0] if len(dot4) == 1 else None), r.get("ytdlp_language"), r["current_rule"]
    v = []
    if ids:
        struct += 1
        ok = len(dot4) == 1 and base(a) == base(y)
        agree += ok
        v.append(".4==yt-dlp" if ok else ".4 MISMATCH")
    if y:
        if base(cr) == base(y):
            rule_ok += 1
        else:
            rule_bad += 1
            wrong.append(r["id"])
            v.append(f"RULE WRONG ({cr} vs {y})")
    print(f"{r['id']:12} {r['intended']:8} {str(a):13} {str(y):7} {cr:10} {len(r['tracks']):4} {'; '.join(v)}")
print(f"\nwith an audioTracks structure: {struct}; exactly one '.4' id and equal to yt-dlp's language: {agree}/{struct}")
print(f"with a yt-dlp language: {rule_ok + rule_bad}; today's rule right on {rule_ok}, WRONG on {rule_bad}: {wrong}")

print("\naudio language (.4) vs the auto-caption track codes that would have to be requested:")
for r in rows:
    ids = r.get("audio_ids") or []
    dot4 = [i.rsplit(".", 1)[0] for i in ids if i.endswith(".4")]
    if not dot4:
        continue
    a = dot4[0]
    asr = [l for l, k in r["tracks"] if k == "asr"]
    print(f"  {r['id']:12} audio={a:6} exact-code-in-asr={a in asr!s:5} base-language-match={[l for l in asr if base(l) == base(a)][:3]}")
