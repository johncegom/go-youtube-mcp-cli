r"""Task 20.0b: is `<L>-orig` genuine for non-English speech? For the 7 Vietnamese/German videos of task 21's smoke,
L = what task 21 resolves (vi / de). Fetch plain L and L-orig (alternating order), compare the caption text.
Output: orig_nonenglish.json + a printed table. Needs %LOCALAPPDATA%\go-ytdlp\yt-dlp-2026.07.04.exe. ~4 min."""
import difflib, json, os, re, subprocess, sys, time

YTDLP = os.path.join(os.environ["LOCALAPPDATA"], "go-ytdlp", "yt-dlp-2026.07.04.exe")
OUT = os.path.dirname(os.path.abspath(__file__))
WORK = os.path.join(os.environ.get("CLAUDE_JOB_DIR", OUT), "tmp", "orig_nonenglish_vtt")
os.makedirs(WORK, exist_ok=True)
VIDEOS = [("r8CppXSqVDU", "vi"), ("B9MBdB1Ih6Q", "vi"), ("Za_PoC0D3CQ", "vi"), ("fdkYE4uxL0A", "vi"),
          ("cZSgL76ddDs", "de"), ("0n5AYXkXP3Y", "de"), ("iLnTZhrkUpA", "de")]


def fetch(video, lang):
    d = os.path.join(WORK, f"{video}__{lang}")
    os.makedirs(d, exist_ok=True)
    args = [YTDLP, "--no-update", "--no-check-formats", "--skip-download", "--force-ipv4", "--sub-format", "vtt",
            "--write-auto-subs", "--write-subs", "--sub-langs", lang, "--output", os.path.join(d, "sub"),
            f"https://www.youtube.com/watch?v={video}"]
    t = time.time()
    try:
        p = subprocess.run(args, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=120)
        text = (p.stdout or "") + (p.stderr or "")
    except subprocess.TimeoutExpired:
        return {"status": "timeout", "text": ""}
    vtts = [f for f in os.listdir(d) if f.endswith(".vtt")]
    if vtts:
        raw = open(os.path.join(d, vtts[0]), encoding="utf-8", errors="replace").read()
        body = " ".join(l for l in raw.splitlines() if l and "-->" not in l and not l.startswith(("WEBVTT", "Kind:", "Language:")))
        body = re.sub(r"<[^>]+>", "", body)
        status, txt = "ok", re.sub(r"\s+", " ", body).strip()
    elif "HTTP Error 429" in text:
        status, txt = "429", ""
    elif "There are no subtitles for the requested languages" in text:
        status, txt = "none", ""
    else:
        status, txt = "other: " + next((l for l in text.splitlines() if l.startswith("ERROR")), "unknown")[:100], ""
    return {"status": status, "text": txt, "seconds": round(time.time() - t, 1)}


rows = []
for i, (vid, L) in enumerate(VIDEOS):
    order = [L, L + "-orig"] if i % 2 == 0 else [L + "-orig", L]
    r = {"id": vid, "L": L, "order": order}
    for lang in order:
        r[lang] = fetch(vid, lang)
    a, b = r[L], r[L + "-orig"]
    r["similarity"] = round(difflib.SequenceMatcher(None, a["text"], b["text"], autojunk=False).ratio(), 3) if a["text"] and b["text"] else None
    r["identical"] = bool(a["text"]) and a["text"] == b["text"]
    for k in (L, L + "-orig"):
        r[k]["chars"] = len(r[k].pop("text"))
    rows.append(r)
    print(f"{vid} L={L}: plain={a['status']}/{r[L]['chars']}ch orig={b['status']}/{r[L+'-orig']['chars']}ch identical={r['identical']} sim={r['similarity']}", flush=True)
    json.dump(rows, open(os.path.join(OUT, "orig_nonenglish.json"), "w"), indent=1)
    if i < len(VIDEOS) - 1:
        time.sleep(15)
print("DONE")
