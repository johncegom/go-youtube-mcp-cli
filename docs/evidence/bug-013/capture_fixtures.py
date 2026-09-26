"""Task 21.0b: capture the real captionTracks + audioTracks of a watch page, baseUrl stripped, real order kept.
Usage: python capture_fixtures.py VIDEOID... -> prints one JSON object per video (compact)."""
import json, re, sys, urllib.request

UA = {"User-Agent": "Mozilla/5.0 (compatible; YoutubeMCP/1.0)"}  # same as fetchWatchPageHTML


def player_response(vid):
    h = urllib.request.urlopen(urllib.request.Request(f"https://www.youtube.com/watch?v={vid}", headers=UA), timeout=20).read().decode("utf-8", "replace")
    i = h.index("ytInitialPlayerResponse = ") + len("ytInitialPlayerResponse = ")
    obj, _ = json.JSONDecoder().raw_decode(h[i:])
    return obj


for vid in sys.argv[1:]:
    r = player_response(vid)["captions"]["playerCaptionsTracklistRenderer"]
    tracks = []
    for t in r["captionTracks"]:
        t = dict(t)
        t.pop("baseUrl", None)
        tracks.append(t)
    audio = [a["audioTrackId"] for a in r.get("audioTracks", []) if "audioTrackId" in a]
    print(json.dumps({"video": vid, "captionTracks": tracks, "audioIds": audio}, ensure_ascii=False, separators=(",", ":")))
