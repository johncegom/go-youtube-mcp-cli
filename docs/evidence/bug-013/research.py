"""BUG-013 research: where does the watch page carry the video's declared/original language,
does it agree with yt-dlp's `language`, and how often does today's resolveDefaultLanguage rule
disagree with it?  Read-only: plain page GETs + yt-dlp -J / ytsearch.  Output: research.json.
"""
import json, os, re, subprocess, sys, time, urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
YTDLP = os.path.join(os.environ["LOCALAPPDATA"], "go-ytdlp", "yt-dlp-2026.07.04.exe")
UA = {"User-Agent": "Mozilla/5.0 (compatible; YoutubeMCP/1.0)"}  # same as fetchWatchPageHTML
PER_LANG = 4
QUERIES = {  # intended language -> search query (proxy truth only)
    "vi": "tin tức thời sự hôm nay",
    "es": "noticias de hoy en vivo",
    "ja": "ニュース 今日 解説",
    "uk": "новини україна сьогодні",
    "hi": "हिंदी समाचार आज",
    "de": "nachrichten heute deutschland",
}
# already-measured videos (17 usable in docs/evidence/bug-012): intended = "en" unless noted
KNOWN = {
    "5oer61Xyi4c": "en", "X0UI0O8YzJM": "en", "kjoQPn--F7A": "en", "r8CppXSqVDU": "vi",
    "BqRhBq-_kgE": "en", "I8XaYkRW1tA": "?", "vyIgAO8aCbA": "en", "qN6OM1IzjIE": "en",
    "vsGwx28z4jk": "en", "LXb3EKWsInQ": "?", "QcdpeFbuy8o": "en", "KcVkq5L-0f0": "en",
    "dQw4w9WgXcQ": "en", "9EUTRL_4Cj8": "vi", "qp0HIF3SfI4": "en", "iG9CE55wbtY": "en",
    "UF8uR6Z6KLc": "en",
}


def run(args, timeout=180):
    return subprocess.run(args, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout)


def search(query, n):
    p = run([YTDLP, "--no-update", "--no-warnings", "--force-ipv4", "--flat-playlist",
             "--print", "%(id)s", f"ytsearch{n}:{query}"])
    return [l.strip() for l in p.stdout.splitlines() if re.fullmatch(r"[A-Za-z0-9_-]{11}", l.strip())]


def page_info(vid):
    h = urllib.request.urlopen(urllib.request.Request(f"https://www.youtube.com/watch?v={vid}", headers=UA), timeout=20).read().decode("utf-8", "replace")
    tracks = []
    m = re.search(r'"captionTracks":\[(.*?)\],"audioTracks"|"captionTracks":\[(.*?)\],"translationLanguages"', h, re.S)
    if m:
        blob = m.group(1) or m.group(2)
        for t in re.finditer(r'\{"baseUrl".*?"languageCode":"([^"]+)"(.*?)\}(?=,\{"baseUrl"|$)', blob, re.S):
            tracks.append((t.group(1), "asr" if 'kind":"asr' in t.group(2) else "uploaded"))
    audio_ids = re.findall(r'"audioTrackId":"([^"]+)"', h)
    dflt = re.findall(r'"audioTrack":\{"displayName":"[^"]*","id":"([^"]+)","audioIsDefault":true\}', h)
    return {
        "tracks": tracks,
        "audio_ids": audio_ids,
        "audio_default_id": dflt[0] if dflt else None,
        "playable": '"playabilityStatus":{"status":"OK"' in h,
    }


def current_rule(tracks):
    """Today's resolveDefaultLanguage (internal/core/language.go): any en/en-* track -> en; else first asr's language; else en."""
    for l, _ in tracks:
        if l == "en" or l.startswith("en-"):
            return "en"
    for l, k in tracks:
        if k == "asr" and l:
            return l
    return "en"


def audio_lang(track_id):
    return track_id.rsplit(".", 1)[0] if track_id else None


def ytdlp_language(vid):
    p = run([YTDLP, "--no-update", "--no-warnings", "--force-ipv4", "--no-playlist", "-J", "--skip-download",
             f"https://www.youtube.com/watch?v={vid}"])
    try:
        info = json.loads(p.stdout)
    except Exception:
        return {"error": (p.stderr or "no json")[-120:]}
    return {"language": info.get("language"), "title_lang_hint": None}


# 1) candidates
cands = dict((v, i) for v, i in KNOWN.items())
for lang, q in QUERIES.items():
    ids = search(q, PER_LANG)
    print("search", lang, ids, flush=True)
    for v in ids:
        cands.setdefault(v, lang)
    time.sleep(3)

# 2) per video
rows = []
for vid, intended in cands.items():
    row = {"id": vid, "intended": intended, "known": vid in KNOWN}
    try:
        row.update(page_info(vid))
    except Exception as e:  # noqa: BLE001
        row["page_error"] = str(e)[:80]
        rows.append(row)
        continue
    row["page_audio_lang"] = audio_lang(row["audio_default_id"])
    row["current_rule"] = current_rule(row["tracks"])
    time.sleep(1.5)
    y = ytdlp_language(vid)
    row["ytdlp_language"] = y.get("language")
    if "error" in y:
        row["ytdlp_error"] = y["error"]
    rows.append(row)
    print(vid, intended, "audio_default=", row["audio_default_id"], "ytdlp=", row["ytdlp_language"], "rule=", row["current_rule"], flush=True)
    json.dump(rows, open(os.path.join(HERE, "research.json"), "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    time.sleep(6)
json.dump(rows, open(os.path.join(HERE, "research.json"), "w", encoding="utf-8"), indent=1, ensure_ascii=False)
print("DONE", flush=True)
