"""Compares "does an uploaded (non-asr) English caption track exist?" as read from
the watch page's captionTracks (what the app already scrapes) with yt-dlp's own
`subtitles` list (j.json, from jcheck.py), for every video in j.json.

Plain HTTP GETs of the watch page only; no yt-dlp. Usage: python pagecheck.py
"""
import json
import os
import re
import time
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
rows = [r for r in json.load(open(os.path.join(HERE, "j.json"), encoding="utf-8")) if "error" not in r]
UA = {"User-Agent": "Mozilla/5.0 (compatible; YoutubeMCP/1.0)"}  # same UA as fetchWatchPageHTML

mism, out = [], []
for r in rows:
    vid = r["id"]
    try:
        req = urllib.request.Request(f"https://www.youtube.com/watch?v={vid}", headers=UA)
        html = urllib.request.urlopen(req, timeout=20).read().decode("utf-8", "replace")
    except Exception as e:  # noqa: BLE001 - a measurement script, report and continue
        out.append((vid, "FETCH-ERROR", str(e)[:60]))
        continue
    m = re.search(r'"captionTracks":\[(.*?)\],"audioTracks"|"captionTracks":\[(.*?)\],"translationLanguages"', html, re.S)
    tracks = []
    if m:
        blob = m.group(1) or m.group(2)
        for t in re.finditer(r'\{"baseUrl".*?"languageCode":"([^"]+)"(.*?)\}(?=,\{"baseUrl"|$)', blob, re.S):
            tracks.append((t.group(1), 'kind":"asr' in t.group(2)))
    page_manual_en = any((l == "en" or l.startswith("en-")) and not asr for l, asr in tracks)
    page_asr_en = any((l == "en" or l.startswith("en-")) and asr for l, asr in tracks)
    ytdlp_manual_en = bool(r["manual_en_keys"])
    ok = page_manual_en == ytdlp_manual_en
    out.append((vid, len(tracks), page_manual_en, ytdlp_manual_en, page_asr_en, "match" if ok else "MISMATCH"))
    if not ok:
        mism.append(vid)
    time.sleep(1.5)

print(f"{'id':12} {'#tracks':7} {'page_manual_en':14} {'ytdlp_manual_en':15} {'page_asr_en':11}")
for o in out:
    print(*o)
print("mismatches:", mism)
