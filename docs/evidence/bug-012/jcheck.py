import json, os, subprocess, sys, time
from urllib.parse import urlparse, parse_qs

YTDLP = os.path.join(os.environ["LOCALAPPDATA"], "go-ytdlp", "yt-dlp-2026.07.04.exe")
HERE = os.path.dirname(os.path.abspath(__file__))
MANIFEST = os.path.join(HERE, "manifest.json")  # written by measure.py
PACE = 6


def params(entries):
    """lang/tlang/kind/variant of the first entry's URL (all formats share them)."""
    if not entries:
        return None
    q = parse_qs(urlparse(entries[0].get("url", "")).query)
    return {k: q[k][0] for k in ("lang", "tlang", "kind", "variant") if k in q}


manifest = json.load(open(MANIFEST, encoding="utf-8"))
rows = []
for m in manifest:
    vid = m["id"]
    if "unavailable" in m["en"]["status"]:
        continue
    p = subprocess.run([YTDLP, "--no-update", "--no-warnings", "--force-ipv4", "--no-playlist",
                        "-J", "--skip-download", f"https://www.youtube.com/watch?v={vid}"],
                       capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=180)
    try:
        info = json.loads(p.stdout)
    except Exception:
        rows.append({"id": vid, "error": (p.stderr or "no json")[-200:]})
        print(vid, "ERROR", flush=True)
        time.sleep(PACE)
        continue
    subs = info.get("subtitles") or {}
    auto = info.get("automatic_captions") or {}
    row = {
        "id": vid, "origin": m["origin"],
        "fetch_en": m["en"]["status"], "fetch_orig": m["en-orig"]["status"],
        "manual_en_keys": [k for k in subs if k == "en" or k.startswith("en-")],
        "auto_has_en": "en" in auto,
        "auto_orig_keys": [k for k in auto if k.endswith("-orig")],
        "en_url": params(auto.get("en")),
        "en_orig_url": params(auto.get("en-orig")),
        "manual_en_url": params(subs.get("en")),
        "language": info.get("language"),
    }
    rows.append(row)
    print(vid, row["manual_en_keys"], row["auto_orig_keys"], row["en_url"], flush=True)
    json.dump(rows, open(os.path.join(HERE, "j.json"), "w", encoding="utf-8"), indent=1)
    time.sleep(PACE)
print("DONE", flush=True)
