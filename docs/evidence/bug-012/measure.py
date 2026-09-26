import json, os, re, subprocess, sys, time

YTDLP = os.path.join(os.environ["LOCALAPPDATA"], "go-ytdlp", "yt-dlp-2026.07.04.exe")
OUT = os.path.dirname(os.path.abspath(__file__))
WORK = os.path.join(OUT, "vtt")
os.makedirs(WORK, exist_ok=True)

# 15 IDs from errors.log (failure-biased) + 3 controls (talks, uploaded subs).
LOG_IDS = ["5oer61Xyi4c", "X0UI0O8YzJM", "kjoQPn--F7A", "r8CppXSqVDU", "BqRhBq-_kgE",
           "I8XaYkRW1tA", "vyIgAO8aCbA", "qN6OM1IzjIE", "vsGwx28z4jk", "LXb3EKWsInQ",
           "QcdpeFbuy8o", "osLrm8nve_A", "KcVkq5L-0f0", "dQw4w9WgXcQ", "9EUTRL_4Cj8"]
CONTROL_IDS = ["qp0HIF3SfI4", "iG9CE55wbtY", "UF8uR6Z6KLc"]
VIDEOS = [(v, "log") for v in LOG_IDS] + [(v, "control") for v in CONTROL_IDS]
PACE_SECONDS = 15


def fetch(video, lang):
    d = os.path.join(WORK, f"{video}__{lang}")
    os.makedirs(d, exist_ok=True)
    args = [YTDLP, "--no-update", "--no-check-formats", "--skip-download", "--force-ipv4",
            "--sub-format", "vtt", "--write-auto-subs", "--write-subs", "--sub-langs", lang,
            "--output", os.path.join(d, "sub"), f"https://www.youtube.com/watch?v={video}"]
    t = time.time()
    try:
        p = subprocess.run(args, capture_output=True, text=True, encoding="utf-8",
                           errors="replace", timeout=120)
        text = (p.stdout or "") + (p.stderr or "")
    except subprocess.TimeoutExpired:
        return {"status": "timeout", "path": None, "seconds": round(time.time() - t, 1)}
    vtts = [f for f in os.listdir(d) if f.endswith(".vtt")]
    if vtts:
        status, path = "ok", os.path.join(d, vtts[0])
    elif "HTTP Error 429" in text:
        status, path = "429", None
    elif "There are no subtitles for the requested languages" in text:
        status, path = "none", None
    else:
        err = next((l for l in text.splitlines() if l.startswith("ERROR")), "unknown")
        status, path = "other: " + err[:120], None
    return {"status": status, "path": path, "seconds": round(time.time() - t, 1)}


results = []
for i, (video, origin) in enumerate(VIDEOS):
    order = ["en", "en-orig"] if i % 2 == 0 else ["en-orig", "en"]
    row = {"id": video, "origin": origin, "order": order}
    for lang in order:
        row[lang] = fetch(video, lang)
    results.append(row)
    print(f"{video} ({origin}) en={row['en']['status']} en-orig={row['en-orig']['status']}", flush=True)
    with open(os.path.join(OUT, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(results, f, indent=1)
    if i < len(VIDEOS) - 1:
        time.sleep(PACE_SECONDS)
print("DONE", flush=True)
