# Implementation

Where work on `mwanachama-backend-agency` is tracked. Two files carry the board:

- [todo.md](todo.md) — **open rows only**, ids in the `AG` prefix, status one of
  🚀 In Progress · 📋 Not Started · ⏸️ Blocked. No prose and no finished rows.
- [todo_done.md](todo_done.md) — every finished row, and the board context that
  used to sit above the open list. A finished id is never reused: scan this
  file before minting the next one.

- [update-keeps-unsent-fields.md](update-keeps-unsent-fields.md) — how a
  workflow, step or role update keeps the fields its request does not carry.
- [step-fields.md](step-fields.md) — the `compute` and `post` step kinds and
  the fields a calculating, posting or holding step stores.

## Rows that arrive with a failing test

Two scheduled routines file rows here, and each files a **deliberately red**
test beside its row, in this repo's ordinary suite:

- `Test<ID>_OpenHole_<slug>` — a security hole the security sweep reproduced
  (`🐞 **SECURITY — …**` rows, with Did · Observed · Threat · Control · Verdict).
- `Test<ID>_OpenBug_<slug>` — a defect the integration sweep reproduced
  (`🐞 **BUG — …**` rows).

The test asserts what the code *should* do, so the fix turns it green; it is
never edited to pass. Each red is listed under `Red by design` in
`mwanachama-developer/documentation/4. qa/security-sweeps/` or
`…/integration-sweeps/`, so a red named there is a known open row, and a red
named in neither is new. Clearing one — fix the code, move the row to
`todo_done.md`, rename the test to the guard it has become — is the
`fix-red-tests` skill's job.

## Verifying a change

`go test ./...`. In-memory SQLite, across the root package, `mcp/` and `routes/`.

## Opening a database written before AGD-009

`Provision` carries the old step-to-objective links forward. When `work_items` still has its `objective_id` column, it inserts one `workflow_objectives` row for each distinct draft, workflow and objective found on a live step with a non-empty `objective_id`, provided the workflow and the objective exist in that same draft and no link for the pair exists yet. It runs on every `Provision`, adds nothing the second time, and works on SQLite and Postgres, so a database that was already opened once after the table arrived is still caught up. The `objective_id` data and every old objective-level `role_assignments` row (empty `work_item_id`) are left alone; the owner has not decided whether to clean them, and until then the scan flags those rows as dangling. A link table that already holds exact duplicate pairs, from a build before the unique index `wf_obj_unique`, is collapsed to its oldest row (a live row before a retired one) before the index is created, so the index never stops a database from opening. A step must name a workflow of its own draft: an empty or foreign `workflow_id` is refused with `ErrWorkflowNotFound`.
