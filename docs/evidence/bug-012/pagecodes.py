"""Per-track view of the watch page's captionTracks joined with yt-dlp's -J data (j.json).

For every video in j.json prints:
  - the page's auto-generated (asr) and uploaded track language codes,
  - whether an asr track with the EXACT code "en" exists on the page,
  - whether yt-dlp lists "en-orig" (automatic_captions),
  - yt-dlp's own `language` field (the video's declared language),
  - the uploaded-English keys yt-dlp sees (`subtitles`).
Tests: exact-asr-"en" on the page <=> yt-dlp lists "en-orig".

Plain GETs of the watch page (same UA as fetchWatchPageHTML); no yt-dlp. Usage: python pagecodes.py
"""
import json
import os
import re
import time
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
rows = [r for r in json.load(open(os.path.join(HERE, "j.json"), encoding="utf-8")) if "error" not in r]
UA = {"User-Agent": "Mozilla/5.0 (compatible; YoutubeMCP/1.0)"}

print(f"{'id':12} {'yt-dlp language':15} {'page asr codes (en-like / all count)':38} {'exact asr en':12} {'yt-dlp en-orig':14} {'uploaded en keys'}")
mism = []
for r in rows:
    req = urllib.request.Request(f"https://www.youtube.com/watch?v={r['id']}", headers=UA)
    html = urllib.request.urlopen(req, timeout=20).read().decode("utf-8", "replace")
    m = re.search(r'"captionTracks":\[(.*?)\],"audioTracks"|"captionTracks":\[(.*?)\],"translationLanguages"', html, re.S)
    asr, uploaded = [], []
    if m:
        blob = m.group(1) or m.group(2)
        for t in re.finditer(r'\{"baseUrl".*?"languageCode":"([^"]+)"(.*?)\}(?=,\{"baseUrl"|$)', blob, re.S):
            (asr if 'kind":"asr' in t.group(2) else uploaded).append(t.group(1))
    en_like = [c for c in asr if c == "en" or c.startswith("en-")]
    exact = "en" in asr
    listed = "en-orig" in r["auto_orig_keys"]
    if exact != listed:
        mism.append(r["id"])
    print(f"{r['id']:12} {str(r.get('language')):15} {str(en_like) + ' / ' + str(len(asr)):38} {str(exact):12} {str(listed):14} {r['manual_en_keys']}")
    time.sleep(1.5)
print("exact-asr-en vs yt-dlp en-orig mismatches:", mism)
