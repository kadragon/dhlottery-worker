# Workflows

Five workflows for this repo. Pick the primary one per cycle.

## `code` — Implementation

The primary cycle for behavioral changes.

**Step 0: Branch** — Never commit to `main`. Create `git checkout -b <type>/<slug>` first.

**Step 1: Read docs** — Before editing, read `docs/architecture.md` (source structure) and `docs/conventions.md` (patterns). Read `docs/endpoints.md` before changing any HTTP call.

**Step 2: Sprint Contract** — Define "done" before writing code. Write concrete acceptance criteria (testable conditions, not impressions). Add to `tasks.md` or the backlog item.

**Step 3: Implement** — Red → Green → Refactor (TDD). New behavior must have tests. Coverage must stay above thresholds (lines/functions/statements 85%, branches 75%).

**Step 4: Verify** — Run `bun run test` (never `bun test`). Run `bun run precommit`. Fix everything before committing.

**Step 5: Evaluate** — Grade against `docs/eval-criteria.md`. All criteria must be pass or acceptable before the feature is done.

## `plan` — Spec Generation

Expand a backlog item into a full design before coding.

1. Write `docs/design/{feature}.md`: goal, constraints, approach, done-when criteria.
2. Review with user. Do not proceed until approved.
3. Generate `backlog.md` items from the approved spec.

Skip for trivial changes (single-file, <20 LOC, no new behavior).

## `draft` — Documentation

Write or update `docs/`. Ground every claim in current code. Never modify production code in this workflow. If the doc reveals a missing constraint, add it to `backlog.md`.

## `constrain` — Architectural Enforcement

1. Write a structural test or lint rule first.
2. Run it. If existing code violates → add remediation to `backlog.md`, don't fix inline.
3. Update `docs/architecture.md`.

## `sweep` — Garbage Collection

Run between features or when entropy is visible.

1. Run `bun run check` and `bun run test:coverage`.
2. Review open items in `tasks.md` — close completed ones, escalate stale ones.
3. Check `docs/` for staleness against current code.
4. Tag new findings as `[doc]`, `[constraint]`, `[debt]`, or `[harness]` in `tasks.md`.
5. Fix trivials inline; leave complex items for a dedicated cycle.

## Context Anxiety

Prefer context resets over compaction. When starting multi-session work, write a `handoff-{feature}.md` at the start — not after context degrades. Include:

- Current branch and last commit hash
- What's done, what's next
- Any unresolved questions or blockers
- Which files were last touched

Discard the handoff file after the sprint closes.
