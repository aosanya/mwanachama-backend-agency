# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG11 | P1 | 📋 | `AgencyManager` has `CreateAgency`/`GetAgency` but no `UpdateAgency` — Mission/Vision can be set at creation but never edited after. Add `UpdateAgency(ctx, a models.Agency) (models.Agency, error)` (explicit column-map update, matching `UpdateGoal`/`UpdateWorkflow`/`UpdateWorkItem`'s convention, not `Updates(&row)`) plus a `PATCH /agency` route in `routes/agency.go`. Note: this repo's routes are consumed by two callers that both auto-register whatever `agencyroutes.Routes` returns — `mwanachama-wakala-api` (WK5) picks the new route up for free, but `mwanachama-backend-api-gateway`'s AG10 wiring has a hard `TestPostmanCollectionCoversEveryRoute` gate that will start failing until that repo's Postman collection gets a matching request added. | — |
