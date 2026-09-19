# EAGD log

Append-only. One row per event, one line per row, no line breaks inside a cell, a literal `|` written as `\|`, `—` for a field that doesn't apply. Rationale: [execute-advise-grade-dream.md](execute-advise-grade-dream.md). Mechanism: `CLAUDE.md`, "Execute / Advise / Grade".

## Advise calls

| Date | Branch | Question | Prior leaning | Answer | Taken | Tool | Requested | Reported | Status |
|------|--------|----------|---------------|--------|-------|------|-----------|----------|--------|
| 2026-09-20 | feat/task-18-native-language-fallback | Fall back to the native caption track after an en 429, or resolve the default language from captionTracks before fetching? | Fallback on 429 (smallest diff, no change on working paths) | Resolve first + per-video memo: the 429 category cannot tell a translated-track rejection from a real throttle, so a fallback could silently serve the wrong language; also drop parsedTranscript.Language, keep the language note out of the transcript body, add a DECISIONS entry for the Save/brief scope cut | Answer taken, except naming the spoken language in the BUG-011 message (deferred as a follow-up) | Agent | opus | claude-opus-5 | ok |

## Binding changes

| Date | Role | Tool | Old | New | Reason |
|------|------|------|-----|-----|--------|
| 2026-09-19 | advise | Agent | — | opus | initial install, probe reported claude-opus-5 |
| 2026-09-19 | grade | Agent | — | haiku | initial install, probe reported claude-haiku-4-5-20251001 |

## Grade fallbacks

| Date | Branch | Tool | Reason |
|------|--------|------|--------|
