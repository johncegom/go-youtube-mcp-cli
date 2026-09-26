"""Does the viewer's locale change what the watch page says about the ORIGINAL audio track?
Fetches the same pages with different Accept-Language headers and prints (a) the streamingData
`audioIsDefault:true` track and (b) the ids ending ".4" in captions.audioTracks.
Plain GETs, no yt-dlp.  Usage: PYTHONIOENCODING=utf-8 python locale_check.py
"""
import re
import time
import urllib.request

UA = "Mozilla/5.0 (compatible; YoutubeMCP/1.0)"
VIDEOS = [("r8CppXSqVDU", "Vietnamese original"), ("cZSgL76ddDs", "German original"), ("vyIgAO8aCbA", "English original")]
LOCALES = (None, "en-US,en;q=0.9", "de-DE,de;q=0.9", "ja-JP,ja;q=0.9")


def get(vid, accept):
    headers = {"User-Agent": UA}
    if accept:
        headers["Accept-Language"] = accept
    req = urllib.request.Request(f"https://www.youtube.com/watch?v={vid}", headers=headers)
    return urllib.request.urlopen(req, timeout=20).read().decode("utf-8", "replace")


print("== A. streamingData default audio track (audioIsDefault:true) per viewer locale")
print(f"{'video':12} {'Accept-Language':16} {'default audio id':17} {'#audioTracks':12} hostLanguage")
for vid, note in VIDEOS:
    for acc in LOCALES:
        h = get(vid, acc)
        d = re.findall(r'"audioTrack":\{"displayName":"([^"]*)","id":"([^"]+)","audioIsDefault":true\}', h)
        ids = re.findall(r'"audioTrackId":"([^"]+)"', h)
        host = re.findall(r'"hostLanguage":"([^"]+)"', h)[:1]
        print(f"{vid:12} {str(acc):16} {str(d[0][1] if d else None):17} {len(ids):<12} {host}  ({note})")
        time.sleep(1.5)

print("\n== B. the '.4' id in captions.audioTracks per viewer locale")
for vid, _ in VIDEOS:
    row = []
    for acc in LOCALES:
        ids = re.findall(r'"audioTrackId":"([^"]+)"', get(vid, acc))
        row.append(([i for i in ids if i.endswith(".4")], len(ids)))
        time.sleep(1.5)
    print(f"{vid:12} .4 ids per locale = {[r[0] for r in row]}  ids-count = {[r[1] for r in row]}  identical across locales: {all(r == row[0] for r in row)}")
