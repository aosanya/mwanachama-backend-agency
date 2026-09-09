# mwanachama-backend-agency (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

| ID | Pri | Status | Task | Depends on |
|---|---|---|---|---|
| AG6 | P1 | 📋 | Build the RACI matrix: `Role` model, Role-to-WorkItem assignment with a Responsible/Accountable/Consulted/Informed label, and validation (e.g. at most one Accountable per WorkItem). Net new — nothing to port. **Has a real downstream dependent, 2026-09-09**: `mwanachama-backend-api-kazi`'s K6 (human-in-the-loop write approval) reads this RACI document to decide which agent-initiated writes need human approval — its own board is blocked on this landing. | AG3, AG5 |
| AG7 | P2 | 📋 | Seed the NeoLabHQ test Agency: all 21 personas from [NeoLabHQ/context-engineering-kit](https://github.com/NeoLabHQ/context-engineering-kit)'s `agents/` directory (`bug-hunter`, `business-analyst`, `change-expectation-agent`, `change-failure-agent`, `change-impact-agent`, `change-story-agent`, `code-explorer`, `code-quality-reviewer`, `code-reviewer`, `contracts-reviewer`, `developer`, `fpf-agent`, `historical-context-reviewer`, `judge`, `meta-judge`, `researcher`, `security-auditor`, `software-architect`, `tech-lead`, `tech-writer`, `test-coverage-reviewer`) become Roles under one software-type department, then exercise AG6's RACI matrix end-to-end against them. | AG6 |
| AG9 | P2 | 📋 | Add a `routes/` package covering the domain, matching `mwanachama-backend-actor`'s HTTP route conventions. | AG3, AG5, AG6 |
| AG10 | P3 | 📋 | Wire into `mwanachama-backend-api-gateway` (deps.go, migration, capability-gated routes) — same pattern as actor/taskmanager/git's own wiring tasks. | AG9 |
