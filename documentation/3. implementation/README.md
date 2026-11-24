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
- [input-sources.md](input-sources.md) — the list of sources a builder may pick for
  a calculating step's input, and the step fields on the work item tools.
- [step-kind-registry.md](step-kind-registry.md) — the seam where a step kind
  registers its checks, scan findings, fork remap, document fields and hash
  record.
- [compute-step-checks.md](compute-step-checks.md) — what a calculating step is
  refused for when it is saved, what the Issue scan flags in a stored one, the
  function catalog and what stays open (ISS-60).
- [compute-failure-checks.md](compute-failure-checks.md) — the save refusals and
  Issue scan findings of a calculating step's failure choice, and the reading of
  a loop of steps with no way out (ISS-61).
- [post-hold-checks.md](post-hold-checks.md) — what is refused when a posting
  step or a hold is saved, what the Issue scan flags and what stays open
  (ISS-63).
- [step-copy-rules.md](step-copy-rules.md) — what each field of the calculating,
  posting and holding steps does on a fork, a publication, the hash and an
  import (ISS-66).
- [scenario-coverage.md](scenario-coverage.md) — the warning for a calculating or
  posting step that no current scenario run passes through (ISS-70).
- [compute-step-host.md](compute-step-host.md) — what wakala-api does at a
  calculating step (ISS-19).
- [compute-step-failure-host.md](compute-step-failure-host.md) — what wakala-api
  does when a calculating step fails.
- [posting-step-host.md](posting-step-host.md) — what the host and the accounting
  repo must do for the posting step.
- [held-figures-host.md](held-figures-host.md) — what the host and the accounting
  repo must do for held figures (ISS-22).
- [scenario-compute-host.md](scenario-compute-host.md) — what wakala-api and the
  test runner do for a scenario that computes or posts (ISS-15).
- [accounting-ledger-audit.md](accounting-ledger-audit.md) — what
  `mwanachama-backend-accounting` actually builds of its ledger, judged from its
  code and tests.
- [hub-copy-diff.md](hub-copy-diff.md) — how this repo and the hub's copy of the
  agency module are compared.
- [hub-import-drift.md](hub-import-drift.md) — every difference between the hub's
  copy and this repo on 2026-11-19, what the merge took from the hub, and the
  hub wiring that stays in the hub.
- [library-title-concurrency.md](library-title-concurrency.md) — library parts
  created or renamed at the same moment (MUN698).
- [provision-index-check.md](provision-index-check.md) — `Provision` gives back
  a declared index that an existing table lacks (MUN317).

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

`Provision` carries the old step-to-objective links forward. When `work_items` still has its `objective_id` column, it inserts one `workflow_objectives` row for each distinct draft, workflow and objective found on a live step with a non-empty `objective_id`, provided the workflow and the objective exist in that same draft and no link for the pair exists yet. It runs on every `Provision`, adds nothing the second time, and works on SQLite and Postgres, so a database that was already opened once after the table arrived is still caught up. The `objective_id` data and the old `role_assignments` table, which the blueprint no longer declares, are left alone; nothing reads that table, and the scan no longer looks at it (ISS-179). A link table that already holds exact duplicate pairs, from a build before the unique index `wf_obj_unique`, is collapsed to its oldest row (a live row before a retired one) before the index is created, so the index never stops a database from opening. A step must name a workflow of its own draft: an empty or foreign `workflow_id` is refused with `ErrWorkflowNotFound`.
