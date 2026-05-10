---
name: slice-plan
description: Slice stories, epics, tasks, or implementation plans into thin vertical end-to-end slices using Elephant Carpaccio (Cockburn/Kniberg) plus Neil Killick's slicing heuristic. Use whenever the user asks to break down work, split a story, find the smallest shippable thing, slice an epic, decompose a plan, find vertical slices, find what to ship first, or asks "how do I make this thinner / how do I phase this / what's the MVP". Outputs a structured markdown plan to .claude/tasks/{NNN}/slices.md.
disable-model-invocation: true
---

# Slice Plan

Take a story, epic, task, or implementation plan and produce thin vertical end-to-end slices using **Elephant Carpaccio** (Cockburn, popularized by Kniberg) combined with **Neil Killick's Slicing Heuristic**. Output a structured markdown file under `.claude/tasks/{NNN}/slices.md`.

The point isn't to "phase" the work — it's to find the smallest thing that delivers real user-visible value end-to-end through every layer of the stack, then the next smallest, and so on. Horizontal layers ("Phase 1: schema, Phase 2: backend, Phase 3: UI") are forbidden — they're how this technique most often fails in practice.

## Arguments

`$ARGUMENTS` may be:
- A path to an existing markdown file (plan, spec, story, ADR draft, design doc)
- A short inline description of the work
- Empty — in which case use the most recent plan/spec discussed in this conversation

If multiple inputs are present (e.g. arg + conversation context), prefer the explicit argument.

## Workflow

### 1. Read the source

Read whatever was supplied. Identify:
- The **outcome** — what does success look like for the user?
- The **scope footprint** — which layers, files, components are involved?
- The **stated phases or sections** — these are usually too fat and need re-slicing.

If the source is missing or ambiguous, ask one clarifying question. Don't proceed on guesswork — slicing the wrong work product cleanly is worse than slicing the right one messily.

### 2. Apply Carpaccio + Killick

**Carpaccio rules** — every slice must satisfy all four:
1. **End-to-end.** Cuts through every relevant layer (UI → API → DB).
2. **User-visible value.** Someone could demo it. "Set up the build pipeline" is not a slice.
3. **Independently shippable.** Could ship and stop here without the system being broken.
4. **Smaller than your gut says.** The first instinct ("this is already as small as it gets") is almost always wrong. Carpaccio's whole premise is that you can carve thinner than feels reasonable.

**Killick's axes** — when a slice still feels too fat, peel along these levers, one or many at a time:

| # | Axis | Question to ask |
|---|------|-----------------|
| 1 | **Workflow steps** | Happy path only? Defer alternates, errors, edge branches? |
| 2 | **Business rules / variations** | One rule, role, currency, tier — others later? |
| 3 | **Data variations** | One record type / format / locale / size? |
| 4 | **Interface / input methods** | Hardcoded → config file → CLI → UI? Pick the earliest that delivers value. |
| 5 | **CRUD operations** | Read-only first. Then Create. Update / Delete much later. |
| 6 | **Cross-cutting concerns** | Defer auth, validation, error handling, i18n, perf, observability. |
| 7 | **Quality attributes** | Ugly-but-working before polished. Manual before automated. |

Apply axes recursively until each slice passes a 1-day-or-less smell test. Each slice should reference the axes used to peel it.

**Common anti-patterns to call out and reject:**
- *"Foundation slice" / "skeleton slice"* — usually horizontal infrastructure dressed as a slice. Force it to deliver visible value, or merge it into slice 1.
- *"All the schema up front."* Migrations are cheap (especially pre-prod). Add columns when the slice that uses them lands.
- *"Build all the bindings, then wire the UI."* Bind one thing, wire one thing, ship.
- *"Cover all the edge cases first."* Edge cases are later slices.
- *"Defer the user-facing part to the end."* Inverts the whole point — backend-then-frontend is the canonical horizontal failure mode.

### 3. Carve out the carpaccio

For each slice, capture:

- **Title** — verb-led, user-visible. *"User sees their first mistake in the demo viewer"* — not *"Add analysis_mistakes table"*.
- **Demo** — one sentence: what could the user actually do or see?
- **In scope** — the minimum code/data/UI to make the demo true.
- **Deferred** — what was peeled off, with the Killick axis that justified peeling (e.g. "Killick #5: read-only — no recompute yet").
- **Effort** — rough order of magnitude (½ day, 1 day, 2 days). If a slice exceeds 2 days, slice further.

Aim for **5–8 slices** for a typical epic. Fewer than 3 means you didn't try; more than ~12 means each slice is too trivial to merit its own row — group them.

### 4. Identify the carved-off backlog

Things that were explicitly *not* in the first slices but will surface eventually. Don't lose them — list them under "Deferred / future slices" so the user can audit what got dropped on the cutting-room floor and re-prioritize anything they disagree with.

### 5. Write the output file

**Determine task number:**
1. `ls .claude/tasks/` (create the directory if missing).
2. Find the highest existing 3-digit prefix. Increment by 1, zero-pad to 3 digits (`001`, `002`, ...).

**Determine folder name:**
- `{NNN}-{kebab-slug-of-source-title}` if a source title is identifiable.
- `{NNN}` alone if no clear title (e.g. inline description without a clear name).

**Write to** `.claude/tasks/{folder}/slices.md` using the template below.

If `.claude/tasks/{folder}/slices.md` already exists (re-running the skill against the same source), don't silently overwrite — show the user the existing file and ask whether to overwrite, append a new dated section, or write to a new task number.

### 6. Report back

Show the user:
- The path written to.
- A 3-line summary: source, slice count, total estimated effort.
- The first slice's title — the thing they could start on today.

Don't re-explain the methodology in the chat reply. The methodology lives in the file.

## Output template

Use this exact structure. Slicing is most useful when scannable.

```markdown
# Slices: {Source title}

**Source:** {path or "inline description"}
**Sliced:** {YYYY-MM-DD}
**Method:** Carpaccio (Cockburn/Kniberg) + Killick's heuristic

## North star
{1–2 sentences: what does success look like for the user when all slices are shipped?}

## Slices

### Slice 1 — {verb-led title}
**Demo:** {one sentence — what does the user see/do?}
**In scope:**
- {bullet}
- {bullet}
**Deferred:** {what was peeled off; reference Killick axes by # where applicable}
**Effort:** {½ day | 1 day | 2 days}

### Slice 2 — {…}
…

## Deferred / future slices
- {feature/concern} — {why it was peeled off; when it would naturally arrive}
- …

## Slicing rationale
- **Axes used:** {e.g. "Killick #5 (read-only first), #6 (no auth in slices 1–3)"}
- **Anti-patterns rejected:** {what fat slices you considered and rejected, and why}
- **Risks of this slicing:** {e.g. "schema churn — acceptable because migrations are cheap pre-prod"}
```

## Examples

### Example 1 — fat plan with built-in horizontal phases

Source plan describes adding analytics to a desktop app with 4 phases (*"Phase A: skeleton + 2 categories"*, *"B: trades+positioning"*, *"C: utility"*, *"D: aim+spray+movement"*).

Diagnosis: Phase A bundles ~10 distinct deliverables (migrations, bindings, side panel, recompute, backfill, two analyzer categories). Apply Killick #2, #5, #6 to slice it.

Output excerpt:

```
### Slice 1 — User sees one mistake in the demo viewer
Demo: Importing a fresh demo surfaces "you died with a flash unused" entries in a list inside the existing viewer.
In scope:
- migration with `analysis_mistakes` table only (no aggregate tables)
- one analyzer rule: `died_with_util_unused`
- one Wails binding: GetMistakeTimeline
- one component: <MistakeList /> mounted in the demo viewer
Deferred: scoring, click-to-seek, second mistake type, recompute, legacy backfill, all other categories (Killick #2, #5, #6)
Effort: 1 day
```

### Example 2 — small story, already nearly atomic

Source: *"Let users export their match stats as CSV."*

This may already be a single slice. Don't manufacture phases. Output one slice plus a "Deferred" section listing the obvious extensions (XLSX, JSON, custom date ranges, scheduled exports) that were peeled off.

## Principles

- **Slicing is about discovering value, not assigning work.** A slice that doesn't change what a user can do isn't a slice — it's infrastructure pretending.
- **Schema churn is fine pre-prod.** Migrations are cheap; don't justify fat slices with *"we need the whole schema first."* For projects with live users / multi-tenant DBs / public APIs, weight more conservatively and call this out under "Risks."
- **Trust the source's architecture.** Your job is to thin its slices, not redesign it. If you spot architectural concerns, surface them under "Risks" rather than rewriting the plan.
- **Be honest about uncertainty.** If a slice's effort estimate is a wild guess (no precedent, novel territory), say so. A slice marked *"½ day or maybe 3 days — first time hitting this API"* is more useful than false precision.
- **One slices.md per task folder.** If the source spans multiple loosely-related epics, split them into separate task numbers and produce one file per epic.
- **Don't lecture in chat.** The recipient of this skill wants a usable plan, not a TED talk on agile. Save the philosophy for the file.
