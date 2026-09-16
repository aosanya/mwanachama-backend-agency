# Design & architecture board

New board — this repo isn't part of the mirrored 7-repo platform set
(`developer`, `mwanachama-frontend-*`, `mwanachama-network-admin*`,
`mwanachama-orchestrator`, `mwanachama-website*`) that shares a `DSN-XXX`
board verbatim, so it gets its own local numbering: **`AGD-XXX`**
(Agency Design gap), starting at AGD-001. Completed rows move to
`todo_done.md` (create it on first completion) — this file is
pending-only.

Row format: `| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |`
— 📋 not started; Responsible is 🤖 agent (a routine works it), 👤 human (a
decision the user makes), or 🦸 super human (owner review).

| ID | Pri | Status | Responsible | Phase | Task | Raised by | Depends On |
|---|---|---|---|---|---|---|---|
| AGD-005 | P1 | 📋 | 🤖 agent | implementation | A publication carries the whole Agency as JSON and production runs off tags, not rows — decided 2026-09-16, full record in [todo_details/AGD-005.md](todo_details/AGD-005.md). Today a publication stores only a content hash, so nothing can answer "what was in v1" (no rollback, no diff) and production reads mutable working rows that change mid-flight, making publishing operationally meaningless. Decisions: the tag carries the import/export JSON document (decision #19's shape, so publish↔import round-trips); production reads that JSON **directly**, not a materialized row cache; `Status` becomes `Channel` — several tags live at once on uniquely-named channels ("production"/"staging"/`""`=archived), one tag per channel; rollback is re-living an archived tag; a Draft may fork from the working set **or** a named tag; `WorkPlan`/`ContextSource` are decommissioned outright (code removed, tables dropped); `EventFlowStep` explicitly kept. The `draft_id IS NULL` row set stops being called "live" — it is the working set. Implementation: AG28/AG29/AG30. | 👤 human (2026-09-16 session) | AGD-004 (done) |

AGD-001, AGD-002 and AGD-004 are done for this repo — moved to
[todo_done.md](todo_done.md). Their cross-repo follow-ups
(`mwanachama-wakala-api`, `mwanachama-wakala-studio`,
`mwanachama-backend-api-kazi`, `mwanachama-backend-api-gateway`'s Postman
coverage / digitaltwin-accounting-assetmanager module wiring) are tracked
on [../3. implementation/todo.md](../3.%20implementation/todo.md) (this
repo) and each named repo's own board, not here — they're
implementation-time work now, not open design questions.
