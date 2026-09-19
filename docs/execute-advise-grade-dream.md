# Execute / Advise / Grade / Dream

A reference architecture for splitting a skill's work across four distinct roles instead of one role doing everything. Not every skill needs all four — read "Not every skill needs all four roles" below before adopting this into anything.

## Why this exists

A role that just wrote a draft is bad at grading it, because it still holds the reasoning that produced the draft — a claim can read as clearly stated while writing it and only turn out vague once that reasoning is out of view. This is the exact problem `youtube-video-critic`'s Step 4.5 grading pass was built to fix: the skill used to run its reverse-attitude check and hype-language audit inline, in the same pass that wrote the verdict, and moved them into a separate pass over the finished draft alone. That fix generalizes. This file names the general pattern so it can be applied deliberately elsewhere, instead of re-discovered skill by skill.

## The four roles

**Execute** — does the actual task. The default "doer." Most of a skill's real work happens here: writing the code, drafting the prose, running the analysis, producing the output the user asked for.

**Advise** — a synchronous, mid-run consultation Execute calls out to when it hits a decision it shouldn't resolve alone: an ambiguous judgment call, a need for external verification, a borderline case worth a second opinion. Execute calls Advise and waits for the answer before continuing. This is not a parallel process running alongside Execute — it's a blocking call inside one run, and Execute doesn't proceed until it has the answer. Deciding *when* to make that call is harder than it looks — "call when unsure" is the wrong rule, for reasons given in "Advise's trigger and context" below.

**Grade** — runs only after Execute finishes one complete run. Scores the finished output against a rubric, given *only* the output and the rubric — not the reasoning, transcript excerpts, or working notes that produced it. This is the same "fresh eyes" principle behind `youtube-video-critic`'s Step 4.5: grading with only the output in view catches vague or inconsistent wording that a writer, still holding the reasoning behind it, tends to read as clear. A fail loops back to Execute, but not always the same way — see "Grade's two fail modes" below. Pass moves on to Dream.

**Dream** — runs only after a Grade pass. Reads back the full run history — not just the final output, but the reasoning, any Advise consultations along the way, and the Grade verdict — and distills durable learnings into persistent memory: a ledger, a profile file, a decision log, anything future runs will read. Low-frequency (only on a pass), reflective, and typically asynchronous relative to the user-facing interaction — nothing the user is waiting on.

## Role separation, not parallel execution

The default sequencing: these four roles run one at a time, never in parallel, and Grade/Dream fire at specific gates rather than being always-on:

```
Execute ──(zero or more blocking calls)──> Advise
   │                                          │
   │  <───────── waits for answer ────────────┘
   ▼
Execute finishes one full run
   ▼
Grade (output + rubric only, no reasoning carried over)
   │
   ├── fail ──> back to Execute (full rerun, or a targeted fix — see below)
   │
   └── pass ──> Dream (reads back full run history, writes learnings to memory)
```

This is the default flow, not an absolute law — a skill can deliberately run Dream independent of Grade's outcome, same as it can skip roles it doesn't need (see "Not every skill needs all four roles" below). The bar for deviating is that it's a stated, deliberate choice with a reason, not an accidental gap. `youtube-video-critic`'s Step 5 ledger is exactly this kind of deliberate deviation — see its entry below.

### Advise's trigger and context

"Call Advise when unsure" is the wrong trigger. It keys on Execute's own sense of uncertainty, which fails exactly where it matters: the costly judgment errors are the ones Execute makes *confidently*, and a felt-doubt trigger stays silent on those. Four rules, general to any kind of work:

**Trigger on observable conditions, not felt doubt.** State the trigger as something an outside reader could check: the *kind* of decision (a name that will be hard to change later; two designs that both fit; an unverified claim the output rests on), the *stage* (before the first hard-to-reverse step), or a *fixed budget* (one call per unit of work of a given type, always). The fixed budget is strongest — immune to overconfidence, cost capped by construction.

**Route by question type — Advise is one destination, not the last of three steps.** A *verifiable* question (the answer exists in the code, the repo, or a search) goes to research, never to Advise. A *preference* question (only the user can decide it) goes to the user — an advisor answering it confidently is worse than no answer — unless it has a conventional default, in which case take the default and say so rather than ask. Advise gets only *judgment* questions: no fact settles them and the user has delegated them. This is not a sequence of gates; it doesn't add a user checkpoint before an Advise call.

**Record the leaning before, log the outcome after.** One or two lines on which way Execute leans and why, written before the call — this often resolves the question by itself. After: one log line (date, task, question, prior leaning, answer, which was taken). The log yields a **decision-change rate**, the only way to tell a steering advisor from a rubber stamp, and drives re-calibration: fires on nearly every task → too broad; never fires → too narrow or unneeded; decision-change rate near zero → ceremony.

**Hand it primary sources, not summaries.** Minimal context risks framing bias — the advisor answers the framing Execute chose while already leaning one way. The full transcript makes the advisor inherit Execute's reasoning (the trap Grade is protected from) at the highest cost in the pattern, and a harness whose forks ignore model overrides can't run it on a different model anyway. Default to the middle: the question, the leaning with the case for and against, and the artifacts the decision turns on *verbatim* — files, diff, spec, error output. Ask the advisor to name any context it lacked.

### Grade's two fail modes

A Grade fail doesn't always call for the same response. There are two distinct modes, and a skill's Step 4.5-equivalent section should say explicitly which one it uses — guessing wrong either wastes tokens (a full rerun for a one-line fix) or under-corrects (patching one span when the whole output needed rethinking).

**Full rerun (the default).** Grade sends the run back to Execute to redo the task from scratch. This is the right call when the failure is about the output as a whole, not one identifiable spot: the wrong approach was taken, a whole section is missing, or the reasoning that produced the output was flawed in a way likely to have touched more than one place. Default to full rerun whenever Grade cannot point to a single bounded span that, fixed in isolation, would actually resolve the failure.

**Targeted fix.** Grade names the exact failing item and points to the specific offending span — a line, a sentence, a field — and Execute patches only that span, leaving the rest of the output untouched. This applies only when the failure is genuinely local: Grade can point to a specific, bounded piece of the output that caused it, fixing that piece resolves the failure without touching anything else, and there's no reason to think the same flaw recurs elsewhere in the output unchecked. `youtube-video-critic`'s Step 4.5 uses this mode exclusively, and says so directly: "If an item fails, name it and quote the offending line, then fix only that line — never a full rewrite. Re-check only the failed items after a fix, not the whole rubric." That works because each of its 9 rubric items maps to one identifiable span (a specific line, a specific sentence) rather than a property of the output as a whole.

Two things worth being explicit about, because they're easy to blur:

- **Advise and Grade are opposite information regimes.** Advise is consulted mid-task, so it needs the full relevant context — the primary sources the decision turns on, verbatim (see "Advise's trigger and context" above) — to answer the question in front of it. Grade is deliberately starved of that same context — fresh eyes is the entire point. Don't design a Grade pass that also gets the reasoning "for context"; that quietly turns it back into the same role re-checking its own work.
- **Grade and Dream are not the same gate.** Grade checks one run's output. Dream distills learnings *across* runs (or across one run's full history) into something durable. A skill can have Grade without Dream — most will, since persistent memory is a bigger commitment than a rubric check.

## Default models

Each role has a default model. All four are changeable per skill, and per role within a skill — nothing here is hardcoded.

| Role | Default model | Why |
|---|---|---|
| Execute | Sonnet | Main workhorse — needs strong general capability for the actual task. |
| Advise | Opus | Handles the hardest single-shot judgment calls Execute couldn't resolve alone — worth the strongest available reasoning, since it's a blocking consultation, not routine work. |
| Grade | Haiku | Most rubrics are checklist-shaped — mechanical, fast, cheap pass/fail against explicit criteria. |
| Dream | Opus | Runs rarely (only after a Grade pass), synthesizes across a full run history, and its output (memory) compounds into every future run — worth a stronger model since mistakes here propagate. |

**On Grade's default specifically: Haiku is the default, not a rule.** If a skill's rubric is complex enough — mixing mechanical checks with real judgment calls, not just counting or matching — consider switching that skill's Grade to Sonnet instead. Don't hardcode Haiku for every case just because it's the default.

Concrete contrast from the two skills that already have a Grade-shaped step:

- `youtube-video-critic`'s Step 4.5 rubric has 9 items, and several are judgment calls, not mechanical checks: whether the title-gap line states a real gap or an invented one, whether the verdict paragraph uses hype language, whether the verdict would flip under a reversed user attitude, whether a takeaway actually states a mechanism or just a conclusion. This is a Sonnet-Grade candidate.
- `sound-human`'s Quality Checklist (17 items) leans more mechanical — banned-word counts, structural pattern counts (parallel negation, tricolons, em-dash frequency), paragraph-ending ratios. Most of it is closer to a genuine Haiku fit, though a couple of items ("reads like a real person talking, not a polished essay") are still judgment calls worth watching if Grade were ever formalized here.

## Not every skill needs all four roles

This is a menu, not a mandate. Use only the roles a skill actually needs — forcing all four onto a skill that doesn't need them adds ceremony without benefit. The four skills read while writing this file span the range:

**`youtube-video-critic`** already has Grade and Dream built, just not labeled this way:
- **Execute** — Steps 1-4 (gather, analyze, verdict, takeaways).
- **Grade** — Step 4.5, added this session: a 9-item pass/fail rubric run on the finished draft alone.
- **Dream** — Step 5, the persistent ledger: reads back past evaluations, tracks watched/applied status, informs future personal-relevance judgments.
- **Advise** — not currently present, but a plausible fit: the persona's "verify checkable claims" rule (searching to confirm a specific factual claim before treating it as established) or a borderline verdict call are both mid-run decisions Execute could hand off and wait on.
- **A stated deviation, not a gap:** the skill's Step 5 runs independent of Step 4.5's pass/fail — deliberately, per the skill's own text on why. That's a legitimate deviation from the default Dream-only-after-Grade-pass flow, of exactly the kind this file's "Role separation" section allows: a considered choice with a reason, not an accidental gap. Noted here as the concrete example of what a justified deviation looks like; not something to reconcile by editing the skill in this pass.

**`sound-human`** has Execute (the rewrite itself) and a Grade-shaped step (the Quality Checklist, run silently before returning) — see the Haiku-vs-Sonnet discussion above for what Grade would look like here. Advise and Dream aren't present today. If added: Advise could resolve voice-choice ambiguity mid-rewrite (which of several plausible fixes best matches the selected voice); Dream could persist a learned voice profile across sessions instead of rebuilding "mirror" voice from scratch each time.

**`goal-to-code-unblock`** deliberately has *no* Execute role for the artifact itself — the user is the executor by design; the skill's entire premise is that the agent must not author the implementation. The agent's job is almost entirely **Advise**: checkpoint review, and naming a candidate trap as a question rather than a diagnosis. The closest thing to a Grade moment is the final guardrail step (empty/nil handling, fail-loud vs. fail-silent, cleanup, readability), but it stays advisory feedback rather than a formal rubric gate. No Dream — no persistent cross-session memory in this skill. This is the cleanest example of "only use what's needed": one role, deployed well, is enough here.

**`learn-technology-by-building`** follows the same no-Execute pattern as `goal-to-code-unblock` (the learner writes the code), but has a genuine Dream role where the other skill doesn't:
- **Advise** — concept explanations at the point of use, checkpoint guidance, resource recommendations.
- **Grade** — the trap-check before confirming a checkpoint or stage complete: a rubric checking for achievement, location, progression, and misleading patterns.
- **Dream** — the continuity ledger: completed checkpoints, concepts introduced, and reusable design/architecture decisions are logged, then read back and re-verified ("ask the learner to recall the decision... before reusing it") when they become relevant again in a later session. This is Dream doing exactly what it's meant to: turning one run's history into something a future run actually draws on.

## Scope

This file is a reference for future adoption. It does not change how `sound-human`, `youtube-video-critic`, `goal-to-code-unblock`, or `learn-technology-by-building` currently behave — none of their `SKILL.md` files were edited to write this. Adopting any part of this pattern into a specific skill is a separate, deliberate pass.
