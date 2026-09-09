# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG4 | P1 | 📋 | Port `drafts.go`'s edit lifecycle (`CreateDraft`→edit→`PromoteDraft`→`PublishAgency`, SHA-256 `AgencyPublication`) and `event_flow_steps.go`'s per-workflow ordered lookup, adapted onto AG3's GORM types. | AG3 |
| AG5 | P1 | 📋 | Model the org chart as a local, self-contained type shaped like `mwanachama-backend-actor`'s Group/Hierarchy pattern (parent/child departments, named levels) — no `go.mod` dependency on actor, no live sync or provisioning calls to it (decision #8: Agency is a declarative document, not an operational store). | AG3 |
| AG6 | P1 | 📋 | Build the RACI matrix: `Role` model, Role-to-WorkItem assignment with a Responsible/Accountable/Consulted/Informed label, and validation (e.g. at most one Accountable per WorkItem). Net new — nothing to port. **Has a real downstream dependent, 2026-09-09**: `mwanachama-backend-api-kazi`'s K6 (human-in-the-loop write approval) reads this RACI document to decide which agent-initiated writes need human approval — its own board is blocked on this landing. | AG3, AG5 |
| AG7 | P2 | 📋 | Seed the NeoLabHQ test Agency: all 21 personas from [NeoLabHQ/context-engineering-kit](https://github.com/NeoLabHQ/context-engineering-kit)'s `agents/` directory (`bug-hunter`, `business-analyst`, `change-expectation-agent`, `change-failure-agent`, `change-impact-agent`, `change-story-agent`, `code-explorer`, `code-quality-reviewer`, `code-reviewer`, `contracts-reviewer`, `developer`, `fpf-agent`, `historical-context-reviewer`, `judge`, `meta-judge`, `researcher`, `security-auditor`, `software-architect`, `tech-lead`, `tech-writer`, `test-coverage-reviewer`) become Roles under one software-type department, then exercise AG6's RACI matrix end-to-end against them. | AG6 |
| AG8 | P2 | 📋 | Port the WorkPlan/`ContextSource` event-routing logic currently misnamed `raci.go` in `CodeValdAgency` (`CreateWorkPlan`/`MatchWorkPlans` — matches inbound event topics, dispatches to an AI agent) under a name that matches what it does. Independent of AG6's RACI work. | AG3 |
| AG9 | P2 | 📋 | Add a `routes/` package covering the domain, matching `mwanachama-backend-actor`'s HTTP route conventions. | AG3, AG5, AG6 |
| AG10 | P3 | 📋 | Wire into `mwanachama-backend-api-gateway` (deps.go, migration, capability-gated routes) — same pattern as actor/taskmanager/git's own wiring tasks. | AG9 |
