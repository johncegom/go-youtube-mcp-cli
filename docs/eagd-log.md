# EAGD log

Append-only. One row per event, one line per row, no line breaks inside a cell, a literal `|` written as `\|`, `—` for a field that doesn't apply. Rationale: [execute-advise-grade-dream.md](execute-advise-grade-dream.md). Mechanism: `CLAUDE.md`, "Execute / Advise / Grade".

## Advise calls

| Date | Branch | Question | Prior leaning | Answer | Taken | Tool | Requested | Reported | Status |
|------|--------|----------|---------------|--------|-------|------|-----------|----------|--------|

## Binding changes

| Date | Role | Tool | Old | New | Reason |
|------|------|------|-----|-----|--------|
| 2026-09-19 | advise | Agent | — | opus | initial install, probe reported claude-opus-5 |
| 2026-09-19 | grade | Agent | — | haiku | initial install, probe reported claude-haiku-4-5-20251001 |

## Grade fallbacks

| Date | Branch | Tool | Reason |
|------|--------|------|--------|
